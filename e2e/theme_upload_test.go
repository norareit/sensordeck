//go:build linux

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/uploadstate"
)

// uploadLimit is long enough for an upload with its two pauses of a second, and for a second round that follows it.
const uploadLimit = 20 * time.Second

const screenSerial = "2022100600"

// clockOnlyTheme is a theme that shows cpu.clock alone, which the tree of the tests reads as 1234 MHz.
const clockOnlyTheme = "name: clock\nbackground: \"#102030\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\n" +
	"widgets:\n  - {type: number, font: body, metric: cpu.clock, x: 4, y: 4}\n"

// themeSetup gives a fake screen with a serial, the environment of the tests, the config file and the state file paths.
type themeSetup struct {
	test        *testing.T
	fake        *fakescreen.Fake
	screen      *fakescreen.Screen
	environment []string
	configPath  string
	themes      string // the directory of the installed themes
	statePath   string
}

func newThemeSetup(test *testing.T) *themeSetup {
	test.Helper()
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", screenSerial)
	tree, _ := runningTree(test, fake)
	configHomeVariables, themes := configHome(test)
	cache := test.TempDir()
	environment := append(environmentFor(test, tree, ""), configHomeVariables...)
	environment = append(environment, "XDG_CACHE_HOME="+cache)
	configPath := filepath.Join(filepath.Dir(filepath.Dir(themes)), "sensordeck", "config.yaml")
	return &themeSetup{test: test, fake: fake, screen: screen, environment: environment, configPath: configPath,
		themes: themes, statePath: uploadstate.Path(cache)}
}

// compiled gives the theme file of the theme `theme compile` makes of the name or folder.
func (setup *themeSetup) compiled(nameOrFolder string) []byte {
	setup.test.Helper()
	out := filepath.Join(setup.test.TempDir(), "compiled.dat")
	if outcome := runWith(setup.test, setup.fake, setup.environment, "theme", "compile", nameOrFolder, out); outcome.status != 0 {
		setup.test.Fatalf("theme compile %s: status %d, stderr %q", nameOrFolder, outcome.status, outcome.stderr)
	}
	content, err := os.ReadFile(out)
	if err != nil {
		setup.test.Fatal(err)
	}
	return content
}

func (setup *themeSetup) stateHolds(serial string, theme []byte) bool {
	setup.test.Helper()
	entries, err := uploadstate.Read(setup.statePath)
	if err != nil {
		setup.test.Fatal(err)
	}
	return entries[serial] == uploadstate.Hash(theme)
}

func (setup *themeSetup) installClockOnlyTheme() {
	setup.test.Helper()
	folder := filepath.Join(setup.themes, "clock")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		setup.test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "theme.yaml"), []byte(clockOnlyTheme), 0o644); err != nil {
		setup.test.Fatal(err)
	}
}

// expectUploadAt checks that the reports from the position on start with the reports of an upload of the theme, and gives
// the position after them.
func (setup *themeSetup) expectUploadAt(position int, theme []byte) int {
	setup.test.Helper()
	path := writeThemeFile(setup.test, theme)
	expected := expectedUploadReports(setup.test, path)
	reports, reached := setup.screen.WaitReports(position+len(expected), uploadLimit)
	if !reached {
		setup.test.Fatalf("%d reports arrived, want the %d of an upload from %d", len(reports), len(expected), position)
	}
	for offset, report := range expected {
		if reports[position+offset] != report {
			setup.test.Fatalf("report %d is not report %d of the upload", position+offset, offset)
		}
	}
	return position + len(expected)
}

// waitForReset gives the position of the first reset report from the position on, which starts an upload.
func (setup *themeSetup) waitForReset(from int) int {
	setup.test.Helper()
	deadline := time.Now().Add(uploadLimit)
	for time.Now().Before(deadline) {
		for index, report := range setup.screen.Reports() {
			if index >= from && report == smartmonitor.ResetCommand() {
				return index
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	setup.test.Fatal("no upload arrived")
	return 0
}

func (setup *themeSetup) expectClockThenValue(position int, tag uint8, value uint16) {
	setup.test.Helper()
	reports, reached := setup.screen.WaitReports(position+2, uploadLimit)
	if !reached {
		setup.test.Fatalf("%d reports arrived, want two from %d", len(reports), position)
	}
	if reports[position][0] != 0x03 {
		setup.test.Errorf("report %d is % X, want the clock", position, reports[position][:4])
	}
	if _, found := waitForReport(setup.screen, position+1, uploadLimit, func(pairs map[uint8]uint16) bool {
		return pairs != nil && pairs[tag] == value
	}); !found {
		setup.test.Errorf("no value report with the tag %d at %d", tag, value)
	}
}

func TestThemeUseCreatesTheConfigFile(test *testing.T) {
	setup := newThemeSetup(test)
	outcome := runWith(test, setup.fake, setup.environment, "theme", "use", "minimal-dark")
	outcome.expect(test, 0, "theme minimal-dark set in "+setup.configPath+"\n", "")
	if got, _ := os.ReadFile(setup.configPath); string(got) != "theme: minimal-dark\n" {
		test.Errorf("the config file holds %q", got)
	}
	if len(setup.screen.Reports()) != 0 {
		test.Error("theme use wrote to the screen")
	}
}

func TestRunUploadsTheThemeOnceAndAStartAfterItUploadsNothing(test *testing.T) {
	setup := newThemeSetup(test)
	minimalDark := setup.compiled("minimal-dark")
	runWith(test, setup.fake, setup.environment, "theme", "use", "minimal-dark")
	setup.screen.AcceptUpload(minimalDark)
	running := start(test, setup.fake, setup.environment, "run", "--config", setup.configPath)
	position := setup.expectUploadAt(0, minimalDark)
	setup.expectClockThenValue(position, 1, 42)
	outcome, _ := running.stop(syscall.SIGTERM)
	if outcome.status != 0 || !strings.Contains(outcome.stdout, "uploading theme minimal-dark\ntheme uploaded: ") {
		test.Errorf("status %d, stdout %q, stderr %q", outcome.status, outcome.stdout, outcome.stderr)
	}
	if !setup.stateHolds(screenSerial, minimalDark) {
		test.Error("the state file does not hold the hash under the serial")
	}
	if names := listDirectory(test, filepath.Dir(setup.statePath)); len(names) != 1 {
		test.Errorf("the cache directory holds %v", names)
	}

	// a second start with the same theme and screen makes no upload
	setup.screen.Remove()
	secondScreen := setup.fake.Add(3, "name", screenSerial)
	secondRunning := start(test, setup.fake, setup.environment, "run", "--config", setup.configPath)
	reports, reached := secondScreen.WaitReports(2, uploadLimit)
	if !reached || reports[0][0] != 0x03 {
		test.Fatalf("%d reports, the first % X", len(reports), reports[0][:4])
	}
	secondOutcome, _ := secondRunning.stop(syscall.SIGTERM)
	if !strings.Contains(secondOutcome.stdout, "theme minimal-dark is on the screen\n") || strings.Contains(secondOutcome.stdout, "uploading") {
		test.Errorf("stdout %q", secondOutcome.stdout)
	}
}

func TestThemeUseWhileRunRunsPutsTheNewThemeOnTheScreenWithItsTags(test *testing.T) {
	setup := newThemeSetup(test)
	setup.installClockOnlyTheme()
	minimalDark, clock := setup.compiled("minimal-dark"), setup.compiled("clock")
	runWith(test, setup.fake, setup.environment, "theme", "use", "minimal-dark")
	setup.screen.AcceptUpload(minimalDark)
	running := start(test, setup.fake, setup.environment, "run", "--config", setup.configPath)
	position := setup.expectUploadAt(0, minimalDark)
	setup.expectClockThenValue(position, 1, 42)

	position = len(setup.screen.Reports())
	setup.screen.AcceptUpload(clock)
	if outcome := runWith(test, setup.fake, setup.environment, "theme", "use", "clock"); outcome.status != 0 {
		test.Fatalf("theme use: status %d, stderr %q", outcome.status, outcome.stderr)
	}
	// the reports of the round that was under way when the file changed may come before the upload
	after := setup.expectUploadAt(setup.waitForReset(position), clock)
	if _, reached := waitForReport(setup.screen, after, uploadLimit, func(pairs map[uint8]uint16) bool {
		return pairs != nil && pairs[1] == 1234
	}); !reached {
		test.Error("no value report with the tag 1 and the value of cpu.clock")
	}
	outcome, _ := running.stop(syscall.SIGTERM)
	if outcome.status != 0 || !setup.stateHolds(screenSerial, clock) {
		test.Errorf("status %d, stderr %q", outcome.status, outcome.stderr)
	}
}

func TestRunRefusesAThemeBesideTagsBeforeAnyReport(test *testing.T) {
	setup := newThemeSetup(test)
	config := writeConfigFile(test, "theme: minimal-dark\ntags:\n  cpu.temp: 1\n")
	want := "sensordeck: config file " + config + ": theme and tags cannot both be given, the theme gives the tags\n"
	runWith(test, setup.fake, setup.environment, "run", "--config", config).expect(test, 1, "", want)
	if got := setup.screen.Received(); len(got) != 0 {
		test.Errorf("%d bytes reached the screen", len(got))
	}
}

func TestAThemeThatDoesNotCompileIsRefusedByThemeUseAndByRun(test *testing.T) {
	setup := newThemeSetup(test)
	folder := writeThemeFolder(test, "name: broken\nwidgets:\n  - {type: nothing}\n")
	use := runWith(test, setup.fake, setup.environment, "theme", "use", folder)
	if use.status != 1 || use.stdout != "" || !strings.Contains(use.stderr, filepath.Join(folder, "theme.yaml")) {
		test.Errorf("theme use: status %d, stdout %q, stderr %q", use.status, use.stdout, use.stderr)
	}
	if _, err := os.Stat(setup.configPath); !os.IsNotExist(err) {
		test.Errorf("theme use left a config file: %v", err)
	}
	config := writeConfigFile(test, "theme: '"+folder+"'\n")
	outcome := runWith(test, setup.fake, setup.environment, "run", "--config", config)
	if outcome.status != 1 || !strings.HasPrefix(outcome.stderr, "sensordeck: config file "+config+": theme: "+filepath.Join(folder, "theme.yaml")) {
		test.Errorf("run: status %d, stderr %q", outcome.status, outcome.stderr)
	}
	if got := setup.screen.Received(); len(got) != 0 {
		test.Errorf("%d bytes reached the screen", len(got))
	}
}

func TestUploadKeepsTheStateFileTrueForRun(test *testing.T) {
	setup := newThemeSetup(test)
	minimalDark := setup.compiled("minimal-dark")
	runWith(test, setup.fake, setup.environment, "theme", "use", "minimal-dark")

	other := make([]byte, 1500)
	otherPath := writeThemeFile(test, other)
	setup.screen.AcceptUpload(other)
	if outcome := runWith(test, setup.fake, setup.environment, "upload", otherPath); outcome.status != 0 {
		test.Fatalf("upload: status %d, stderr %q", outcome.status, outcome.stderr)
	}
	if !setup.stateHolds(screenSerial, other) {
		test.Fatal("upload did not record the hash of the file")
	}
	position := len(setup.screen.Reports())

	// the screen holds another file now, so run puts its own theme back
	setup.screen.AcceptUpload(minimalDark)
	running := start(test, setup.fake, setup.environment, "run", "--config", setup.configPath)
	setup.expectUploadAt(position, minimalDark)
	if _, reached := waitForReport(setup.screen, position, uploadLimit, func(pairs map[uint8]uint16) bool { return pairs != nil }); !reached {
		test.Error("no values after the upload")
	}
	running.stop(syscall.SIGTERM)

	// an upload of the compiled file of the theme of the config, after which run uploads nothing
	minimalPath := writeThemeFile(test, minimalDark)
	setup.screen.AcceptUpload(other)
	runWith(test, setup.fake, setup.environment, "upload", otherPath)
	setup.screen.AcceptUpload(minimalDark)
	if outcome := runWith(test, setup.fake, setup.environment, "upload", minimalPath); outcome.status != 0 {
		test.Fatalf("upload: status %d, stderr %q", outcome.status, outcome.stderr)
	}
	position = len(setup.screen.Reports())
	second := start(test, setup.fake, setup.environment, "run", "--config", setup.configPath)
	reports, reached := setup.screen.WaitReports(position+1, uploadLimit)
	if !reached || reports[position][0] != 0x03 {
		test.Errorf("the first report after the upload is % X", reports[position][:4])
	}
	outcome, _ := second.stop(syscall.SIGTERM)
	if strings.Contains(outcome.stdout, "uploading") || !strings.Contains(outcome.stdout, "theme minimal-dark is on the screen") {
		test.Errorf("stdout %q", outcome.stdout)
	}
}
