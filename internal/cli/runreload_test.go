package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// rewrite replaces the config file, and gives it a modification time that no earlier rewrite had, so that the change is
// seen whatever the clock of the file system does.
func rewrite(test *testing.T, path, content string, generation int) {
	test.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
	moment := time.Date(2030, 1, 1, generation, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, moment, moment); err != nil {
		test.Fatal(err)
	}
}

// changeAtWait runs the function in the hook of the wait of the given count, and stops after the waits of stopAfter.
func changeAtWait(harness *harness, at, stopAfter int, change func()) {
	harness.onWait = func(count int) {
		if count == at {
			change()
		}
		if count >= stopAfter {
			harness.cancelStop()
		}
	}
}

func TestRunSendsTheNewMappingsFromTheRoundAfterTheChange(test *testing.T) {
	harness := runWith(twoMetrics(), 3)
	path := writeConfig(test, "tags: {cpu.temp: 1}\n")
	changeAtWait(harness, 1, 3, func() { rewrite(test, path, "tags: {cpu.clock: 2}\n", 1) })
	harness.run("run", "--config", path)
	reports := valueReports(harness)
	first, second := valueReport(smartmonitor.Value{Tag: 1, Value: 42}), valueReport(smartmonitor.Value{Tag: 2, Value: 1234})
	if len(reports) != 3 || reports[0] != first || reports[1] != second || reports[2] != second {
		test.Errorf("%d reports", len(reports))
	}
	if harness.stderr.Len() != 0 {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestRunKeepsTheRunningMappingsForAConfigThatIsInvalidOrDeletedWithOneLineEach(test *testing.T) {
	harness := runWith(twoMetrics(), 5)
	path := writeConfig(test, "tags: {cpu.temp: 1}\n")
	harness.onWait = func(count int) {
		switch count {
		case 1:
			rewrite(test, path, "tags: {cpu.tmp: 1}\n", 1)
		case 3:
			_ = os.Remove(path)
		case 5:
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", path)
	for position, report := range valueReports(harness) {
		if report != valueReport(smartmonitor.Value{Tag: 1, Value: 42}) {
			test.Errorf("round %d: % X", position, report[:8])
		}
	}
	want := "config file " + path + ": tags: \"cpu.tmp\" is not a metric name, keeping the running config\n" +
		"config file " + path + ": cannot read it: no such file or directory, keeping the running config\n"
	if harness.stderr.String() != want {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestRunDoesNotApplyTheKeysThatNeedARestartAndSaysSo(test *testing.T) {
	harness := runWith(twoMetrics(), 4)
	path := writeConfig(test, "tags: {cpu.temp: 1}\n")
	harness.onWait = func(count int) {
		switch count {
		case 1:
			rewrite(test, path, "interval: 2s\ncommander_interval: 3m\ntags: {cpu.clock: 2}\n", 1)
		case 2:
			rewrite(test, path, "commander_color: \"#ff0000\"\ntags: {cpu.clock: 2}\n", 2)
		case 3:
			rewrite(test, path, "librehardwaremonitor_url: http://localhost:9000/data.json\ntags: {cpu.clock: 2}\n", 3)
		case 4:
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", path)
	prefix := "config file " + path + ": "
	want := prefix + "interval changed, restart run to apply it\n" + prefix + "commander_interval changed, restart run to apply it\n" +
		prefix + "commander_color changed, restart run to apply it\n" + prefix + "librehardwaremonitor_url changed, restart run to apply it\n"
	if harness.stderr.String() != want {
		test.Errorf("stderr %q", harness.stderr.String())
	}
	// the interval of the start stays, so each of the four rounds follows after one second
	if len(harness.waits) != 4 || harness.waits[1] != time.Second || harness.waits[3] != time.Second {
		test.Errorf("waits %v", harness.waits)
	}
	if reports := valueReports(harness); len(reports) != 4 || reports[3] != valueReport(smartmonitor.Value{Tag: 2, Value: 1234}) {
		test.Errorf("%d reports", len(reports))
	}
}

func TestRunUploadsTheThemeAfterAChangeFromTagsToThemeAndSendsItsTags(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 3)
	path := writeConfig(test, "tags: {cpu.clock: 2}\n")
	changeAtWait(harness, 1, 3, func() { rewrite(test, path, "theme: '"+folder+"'\n", 1) })
	harness.run("run", "--config", path)
	if got := themeKinds(harness.transport.writes); got != "CUCVCV" {
		test.Errorf("writes %s", got)
	}
	if !strings.Contains(harness.stdout.String(), "uploading theme "+folder+"\n") {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestRunUploadsNothingAfterAChangeFromThemeToTags(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 3)
	path := themeConfig(test, folder)
	changeAtWait(harness, 1, 3, func() { rewrite(test, path, "tags: {cpu.clock: 2}\n", 1) })
	harness.run("run", "--config", path)
	if got := strings.Count(harness.stdout.String(), "uploading"); got != 1 {
		test.Errorf("%d uploads, stdout %q", got, harness.stdout.String())
	}
	if reports := valueReports(harness); reports[len(reports)-1] != valueReport(smartmonitor.Value{Tag: 2, Value: 1234}) {
		test.Errorf("last values % X", reports[len(reports)-1][:8])
	}
}

func TestRunUploadsTheSameThemeAgainAfterAChangeToTagsAndBack(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 5)
	path := themeConfig(test, folder)
	harness.onWait = func(count int) {
		switch count {
		case 1:
			rewrite(test, path, "tags: {cpu.clock: 2}\n", 1)
		case 3:
			rewrite(test, path, "theme: '"+folder+"'\n", 2)
		case 5:
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", path)
	if got := strings.Count(harness.stdout.String(), "uploading theme"); got != 2 {
		test.Errorf("%d uploads, stdout %q", got, harness.stdout.String())
	}
}

func TestAReloadBackToTheThemeOfTheLastUploadAfterAFailedUploadOfAnotherUploadsIt(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	edited := strings.Replace(compiledTheme, "Hello", "Howdy", 1)
	harness := runningTheme(test, 14)
	path := themeConfig(test, folder)
	harness.onWait = func(count int) {
		switch count {
		case 1:
			// the upload of the edited theme fails, which leaves the screen with neither theme
			harness.transport.failWrites, harness.transport.failErr = []bool{true}, errFake
			writeTestFile(test, folder+"/theme.yaml", edited)
			rewrite(test, path, "theme: '"+folder+"'\n", 1)
		case 2:
			writeTestFile(test, folder+"/theme.yaml", compiledTheme)
			rewrite(test, path, "theme: '"+folder+"'\n", 2)
		case 14:
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", path)
	if got := strings.Count(harness.stdout.String(), "theme uploaded"); got != 2 {
		test.Errorf("%d completed uploads, stdout %q", got, harness.stdout.String())
	}
}

func TestARewriteOfTheSameThemeUploadsNothingAndAnEditedFolderIsUploaded(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 5)
	path := themeConfig(test, folder)
	harness.onWait = func(count int) {
		switch count {
		case 1:
			rewrite(test, path, "theme: '"+folder+"'\n", 1)
		case 3:
			edited := strings.Replace(compiledTheme, "Hello", "Howdy", 1)
			if err := os.WriteFile(folder+"/theme.yaml", []byte(edited), 0o644); err != nil {
				test.Fatal(err)
			}
			rewrite(test, path, "theme: '"+folder+"'\n", 2)
		case 5:
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", path)
	if got := strings.Count(harness.stdout.String(), "uploading theme"); got != 2 {
		test.Errorf("%d uploads, stdout %q", got, harness.stdout.String())
	}
	if got := themeKinds(harness.transport.writes); got != "UCVCVCVUCVCV" {
		test.Errorf("writes %s", got)
	}
}

func TestAReloadToAThemeUploadsItWithNoCacheDirectory(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	harness := runningTheme(test, 3)
	path := writeConfig(test, "tags: {cpu.temp: 1}\n")
	changeAtWait(harness, 1, 3, func() { rewrite(test, path, "theme: '"+folder+"'\n", 1) })
	harness.run("run", "--config", path)
	if harness.stderr.Len() != 0 || strings.Count(harness.stdout.String(), "uploading theme") != 1 || harness.cacheDirCalls != 0 {
		test.Errorf("stderr %q, stdout %q, %d look-ups", harness.stderr.String(), harness.stdout.String(), harness.cacheDirCalls)
	}
}

func TestAReloadThatMapsAProbeGetsItsValueFromTheKeeper(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 4)
	path := writeConfig(test, "interval: 1s\ncommander_color: \"#ff0000\"\ntags: {cpu.temp: 1}\n")
	changeAtWait(harness, 1, 4, func() { rewrite(test, path, colourTags("1s"), 1) })
	harness.run("run", "--config", path)
	reports := valueReports(harness)
	withProbes := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 4, Value: 37}, smartmonitor.Value{Tag: 5, Value: 23})
	if len(reports) < 3 || reports[0] != valueReport(smartmonitor.Value{Tag: 1, Value: 42}) || reports[len(reports)-1] != withProbes {
		test.Errorf("%d reports, last % X", len(reports), reports[len(reports)-1][:14])
	}
	if harness.stderr.Len() != 0 {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}
