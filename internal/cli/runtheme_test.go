package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/uploadstate"
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

// runningTheme gives a harness with a cache directory, a screen with the serial S1 that acknowledges every report, and a
// stop after the given number of waits.
func runningTheme(test *testing.T, stopAfterWaits int) (*harness, string) {
	test.Helper()
	harness := runWith(twoMetrics(), stopAfterWaits)
	statePath := harness.useCacheDirectory(test)
	harness.transport.ackWhenEmpty, harness.transport.serial = true, "S1"
	return harness, statePath
}

func TestRunWithAThemeUploadsItBeforeTheClockAndTheValuesAndRecordsTheHash(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	compiled := compileExpected(test, folder)
	harness, statePath := runningTheme(test, 1)
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
	if entries, err := uploadstate.Read(statePath); err != nil || entries["S1"] != uploadstate.Hash(compiled.File) {
		test.Errorf("the state file holds %v, %v", entries, err)
	}
}

func TestTwoStartsWithTheSameThemeAndScreenMakeOneUpload(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	first, _ := runningTheme(test, 1)
	config := themeConfig(test, folder)
	first.run("run", "--config", config)
	second := runWith(twoMetrics(), 1)
	second.cacheDir, second.cacheDirErr = first.cacheDir, nil
	second.transport.ackWhenEmpty, second.transport.serial = true, "S1"
	if status := second.run("run", "--config", config); status != 0 {
		test.Fatalf("status %d, stderr %q", status, second.stderr.String())
	}
	if got := themeKinds(second.transport.writes); got != "CV" {
		test.Errorf("writes %s", got)
	}
	if want := "running: 1 metrics every 1s\ntheme " + folder + " is on the screen\nstopped\n"; second.stdout.String() != want {
		test.Errorf("stdout %q", second.stdout.String())
	}
}

func TestRunSendsNothingWhileTheThemeIsNotOnTheScreenAfterAFailedUpload(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness, statePath := runningTheme(test, 5)
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
	if entries, _ := uploadstate.Read(statePath); len(entries) != 0 {
		test.Errorf("the state file holds %v", entries)
	}
	if strings.Count(harness.stdout.String(), "uploading theme") != 1 {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestRunTriesAFailedUploadAgainAfterTenSeconds(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	compiled := compileExpected(test, folder)
	harness, statePath := runningTheme(test, 13)
	harness.transport.queue = []readResult{{report: reportOf(0x43)}}
	harness.run("run", "--config", themeConfig(test, folder))
	if got := themeKinds(harness.transport.writes); got != "UCVCVCV" {
		test.Errorf("writes %s", got)
	}
	if strings.Count(harness.stdout.String(), "uploading theme") != 2 || strings.Count(harness.stdout.String(), "theme uploaded") != 1 {
		test.Errorf("stdout %q", harness.stdout.String())
	}
	if entries, _ := uploadstate.Read(statePath); entries["S1"] != uploadstate.Hash(compiled.File) {
		test.Errorf("the state file holds %v", entries)
	}
}

func TestRunChecksTheThemeAgainAfterTheScreenCameBack(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	cases := []struct {
		serial, want string
	}{{"S1", "UCVCV"}, {"OTHER", "UCVUCV"}}
	for _, each := range cases {
		harness, _ := runningTheme(test, 3)
		harness.onWait = func(count int) {
			if count == 1 {
				harness.transport.failWrites = []bool{true}
				harness.transport.serial = each.serial
			}
			if count >= 3 {
				harness.cancelStop()
			}
		}
		harness.run("run", "--config", themeConfig(test, folder))
		if got := themeKinds(harness.transport.writes); got != each.want {
			test.Errorf("serial %s: writes %s, want %s", each.serial, got, each.want)
		}
	}
}

func TestAStopDuringAnUploadEndsRunAfterTheUploadWithTheHashRecorded(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	compiled := compileExpected(test, folder)
	harness, statePath := runningTheme(test, 100)
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
	if entries, _ := uploadstate.Read(statePath); entries["S1"] != uploadstate.Hash(compiled.File) {
		test.Errorf("the state file holds %v", entries)
	}
}

func TestRunRefusesWhatItCannotStartWithAThemeBeforeOpeningTheScreen(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	missing := filepath.Join(test.TempDir(), "nothing")
	cases := []struct {
		name, content, want string
		noCache             bool
	}{
		{"theme and tags", "theme: x\n" + twoTags, "config file %s: theme and tags cannot both be given, the theme gives the tags", false},
		{"no compile", "theme: '" + missing + "'\n", "config file %s: theme: " + filepath.Join(missing, "theme.yaml") + ": cannot read the file: no such file or directory", false},
		{"no cache directory", "theme: '" + folder + "'\n", "cannot find the cache directory: no cache directory in this test", true},
	}
	for _, each := range cases {
		harness := newHarness()
		harness.cacheDir, harness.cacheDirErr = test.TempDir(), nil
		if each.noCache {
			harness.cacheDirErr = os.ErrNotExist
		}
		path := writeConfig(test, each.content)
		want := each.want
		if strings.Contains(want, "%s") {
			want = strings.Replace(want, "%s", path, 1)
		}
		if each.noCache {
			want = "cannot find the cache directory: file does not exist"
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
	harness, _ := runningTheme(test, 16)
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
