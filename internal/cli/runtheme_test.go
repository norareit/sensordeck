package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// themeKinds writes U for each run of reports of an upload, C for a clock report and V for each run of the value report of the
// theme of compiledTheme with twoMetrics. A data block of an upload starts with the byte of a value report too, so a
// value report is known by all of its bytes.
func themeKinds(writes []smartmonitor.Report) string {
	var result strings.Builder
	for _, report := range writes {
		switch {
		case report[0] == 0x03:
			result.WriteString("C")
		case report == valueReport(smartmonitor.Value{Tag: 1, Value: 42}):
			// An upload takes the clock past the place of the next round, which follows at once and sends values again.
			if !strings.HasSuffix(result.String(), "V") {
				result.WriteString("V")
			}
		default:
			if !strings.HasSuffix(result.String(), "U") {
				result.WriteString("U")
			}
		}
	}
	return result.String()
}

// themeConfig writes a config file that names the folder as its theme, and gives its path.
func themeConfig(test *testing.T, folder string) string {
	test.Helper()
	return writeConfig(test, "theme: '"+folder+"'\n")
}

// runningTheme gives a harness with a screen that acknowledges every report, and a stop after the given number of waits.
func runningTheme(test *testing.T, stopAfterWaits int) *harness {
	test.Helper()
	harness := runWith(twoMetrics(), stopAfterWaits)
	harness.transport.ackWhenEmpty = true
	return harness
}

// writeEarlierStateFile writes the state file that a release before unit 0022 kept, with the hash of the compiled theme
// for the serial, and gives its path. run and upload no longer use it.
func writeEarlierStateFile(test *testing.T, cache, serial string, compiled []byte) string {
	test.Helper()
	sum := sha256.Sum256(compiled)
	path := filepath.Join(cache, "sensordeck", "uploaded-themes.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		test.Fatal(err)
	}
	content := `{"` + serial + `":"` + hex.EncodeToString(sum[:]) + `"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

func TestRunWithAThemeUploadsItBeforeTheClockAndTheValues(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	compiled := compileExpected(test, folder)
	harness := runningTheme(test, 1)
	if status := harness.run("run", "--config", themeConfig(test, folder)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := themeKinds(harness.transport.writes); got != "UCV" {
		test.Errorf("writes %s", got)
	}
	want := "running: 1 metrics every 1s\nuploading theme " + folder + "\ntheme uploaded: " + strconv.Itoa(len(compiled.File)) + " bytes\nstopped\n"
	if harness.stdout.String() != want {
		test.Errorf("stdout %q", harness.stdout.String())
	}
	if last := harness.transport.writes[len(harness.transport.writes)-1]; last != valueReport(smartmonitor.Value{Tag: 1, Value: 42}) {
		test.Errorf("values % X", last[:8])
	}
}

func TestRunUploadsAlthoughAnEarlierStateFileNamesTheThemeAndLeavesItAsItWas(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	compiled := compileExpected(test, folder)
	harness := runningTheme(test, 1)
	harness.cacheDir, harness.cacheDirErr = test.TempDir(), nil
	statePath := writeEarlierStateFile(test, harness.cacheDir, "S1", compiled.File)
	before := readTestFile(test, statePath)
	if status := harness.run("run", "--config", themeConfig(test, folder)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := themeKinds(harness.transport.writes); got != "UCV" {
		test.Errorf("writes %s", got)
	}
	if after := readTestFile(test, statePath); after != before {
		test.Errorf("the state file became %q", after)
	}
}

func TestTwoStartsWithTheSameThemeAndScreenMakeTwoUploads(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	config := themeConfig(test, folder)
	for range 2 {
		harness := runningTheme(test, 1)
		if status := harness.run("run", "--config", config); status != 0 {
			test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
		}
		if got := themeKinds(harness.transport.writes); got != "UCV" {
			test.Errorf("writes %s", got)
		}
		if strings.Contains(harness.stdout.String(), "is on the screen") || strings.Count(harness.stdout.String(), "uploading theme") != 1 {
			test.Errorf("stdout %q", harness.stdout.String())
		}
	}
}

func TestRunSendsNothingWhileTheThemeIsNotOnTheScreenAfterAFailedUpload(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 5)
	harness.transport.queue = []readResult{{report: reportOf(0x43)}}
	if status := harness.run("run", "--config", themeConfig(test, folder)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := themeKinds(harness.transport.writes); got != "U" {
		test.Errorf("writes %s", got)
	}
	wantStderr := "theme upload failed: upload failed at the ymodem command: unexpected answer 0x43\nscreen back\n"
	if !strings.HasPrefix(harness.stderr.String(), wantStderr) || strings.Contains(harness.stderr.String(), "screen lost") {
		test.Errorf("stderr %q", harness.stderr.String())
	}
	if strings.Count(harness.stderr.String(), "screen back") != 1 {
		test.Errorf("stderr %q", harness.stderr.String())
	}
	if strings.Count(harness.stdout.String(), "uploading theme") != 1 {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestRunTriesAFailedUploadAgainAfterTenSeconds(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 13)
	harness.transport.queue = []readResult{{report: reportOf(0x43)}}
	harness.run("run", "--config", themeConfig(test, folder))
	if got := themeKinds(harness.transport.writes); got != "UCVCVCV" {
		test.Errorf("writes %s", got)
	}
	if strings.Count(harness.stdout.String(), "uploading theme") != 2 || strings.Count(harness.stdout.String(), "theme uploaded") != 1 {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestRunUploadsNothingWhenTheScreenCameBackAfterAFailedWrite(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 3)
	harness.onWait = func(count int) {
		if count == 1 {
			harness.transport.failWrites = []bool{true}
		}
		if count >= 3 {
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", themeConfig(test, folder))
	if got := themeKinds(harness.transport.writes); got != "UCVCV" {
		test.Errorf("writes %s, want UCVCV", got)
	}
}

func TestRunMakesAnUploadThatFailedBeforeTheScreenCameBackOnceTheWaitHasPassed(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 13)
	harness.transport.failWrites, harness.transport.failErr = []bool{true}, errFake
	written := stampWrites(harness, startOfTheRun())
	harness.run("run", "--config", themeConfig(test, folder))
	starts := uploadStarts(*written)
	// the report of the failed first attempt is not recorded, so the one start is that of the second attempt
	if len(starts) != 1 || starts[0] < 10*time.Second {
		test.Fatalf("uploads started at %v", starts)
	}
	if got := themeKinds(harness.transport.writes); !strings.HasPrefix(got, "UCV") {
		test.Errorf("writes %s", got)
	}
	for _, write := range *written {
		if write.isClock() && write.at < starts[0] {
			test.Errorf("a clock report at %v, before the upload at %v", write.at, starts[0])
		}
	}
}

func TestAStopDuringAnUploadEndsRunAfterTheUpload(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 100)
	harness.transport.afterWrite = func(count int) {
		if count == 1 {
			harness.cancelStop()
		}
	}
	if status := harness.run("run", "--config", themeConfig(test, folder)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := themeKinds(harness.transport.writes); got != "U" {
		test.Errorf("writes %s", got)
	}
	if !strings.Contains(harness.stdout.String(), "theme uploaded: ") || !strings.HasSuffix(harness.stdout.String(), "stopped\n") {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestRunWithAThemeNeedsNoCacheDirectory(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 1)
	harness.cacheDirErr = os.ErrNotExist
	if status := harness.run("run", "--config", themeConfig(test, folder)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := themeKinds(harness.transport.writes); got != "UCV" || harness.cacheDirCalls != 0 {
		test.Errorf("writes %s, %d look-ups of the cache directory", got, harness.cacheDirCalls)
	}
}

func TestRunRefusesWhatItCannotStartWithAThemeBeforeOpeningTheScreen(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "nothing")
	cases := []struct {
		name, content, want string
	}{
		{"theme and tags", "theme: x\n" + twoTags, "config file %s: theme and tags cannot both be given, the theme gives the tags"},
		{"no compile", "theme: '" + missing + "'\n", "config file %s: theme: " + filepath.Join(missing, "theme.yaml") + ": cannot read the file: no such file or directory"},
	}
	for _, each := range cases {
		harness := newHarness()
		path := writeConfig(test, each.content)
		want := each.want
		if strings.Contains(want, "%s") {
			want = strings.Replace(want, "%s", path, 1)
		}
		harness.expectFailure(test, want, "run", "--config", path)
		if len(harness.opened) != 0 || len(harness.transport.writes) != 0 {
			test.Errorf("%s: opened %v, wrote %d", each.name, harness.opened, len(harness.transport.writes))
		}
	}
}

// uploadStarts gives the moments, since the start of the run, of the reports that equal the first report of the run,
// which is the first report of an upload and the same for each one.
func uploadStarts(written []stamped) []time.Duration {
	var starts []time.Duration
	for _, write := range written {
		if write.report == written[0].report {
			starts = append(starts, write.at)
		}
	}
	return starts
}

func TestATickWritesNothingAndUploadsNothingWhileTheThemeIsNotOnTheScreen(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 16)
	harness.transport.queue = []readResult{{report: reportOf(0x43)}}
	written := stampWrites(harness, startOfTheRun())
	harness.run("run", "--config", writeConfig(test, "interval: 5s\ntheme: '"+folder+"'\n"))
	if got := themeKinds(harness.transport.writes); got != "UCVC" { // the last clock is the tick after the round of the second upload
		test.Errorf("writes %s", got)
	}
	if starts := uploadStarts(*written); len(starts) != 2 || starts[0] != 0 || starts[1] < 10*time.Second || starts[1]%(5*time.Second) != 0 {
		test.Errorf("uploads started at %v", starts)
	}
	for _, write := range *written {
		if write.isClock() && write.at < 10*time.Second {
			test.Errorf("a clock report at %v, before the theme was on the screen", write.at)
		}
	}
	if strings.Count(harness.stdout.String(), "uploading theme") != 2 {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}
