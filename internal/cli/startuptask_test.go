package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// taskHarness gives the harness of the service on Windows, with a cache directory of its own and a user.
func taskHarness(test *testing.T) *harness {
	test.Helper()
	harness := serviceHarness(test)
	harness.system = "windows"
	harness.cacheDir = test.TempDir()
	harness.cacheDirErr = nil
	harness.user = sampleUser
	return harness
}

func taskFileOf(harness *harness) string { return taskFilePath(harness.cacheDir) }

func defaultConfigOf(harness *harness) string {
	return filepath.Join(harness.configDir, "sensordeck", "config.yaml")
}

func resolved(test *testing.T, path string) string {
	test.Helper()
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		test.Fatal(err)
	}
	return path
}

func expectTaskFile(test *testing.T, harness *harness, executable, config string) {
	test.Helper()
	want, err := taskFileContent(sampleUser, executable, config)
	if err != nil {
		test.Fatal(err)
	}
	if got := readTestFile(test, taskFileOf(harness)); got != string(want) {
		test.Errorf("task file %q, want %q", got, want)
	}
}

func expectNoTaskFileAndNoCall(test *testing.T, harness *harness) {
	test.Helper()
	if _, err := os.Stat(taskFileOf(harness)); !errors.Is(err, os.ErrNotExist) {
		test.Errorf("task file: %v", err)
	}
	if len(harness.calls) != 0 || len(harness.steps) != 0 {
		test.Errorf("calls %v, steps %v", harness.calls, harness.steps)
	}
}

func installCallsOf(harness *harness) [][]string {
	return [][]string{{"/Create", "/TN", "sensordeck", "/XML", taskFileOf(harness), "/F"}, {"/Run", "/TN", "sensordeck"}}
}

func TestTaskInstallWritesTheTaskFileForTheDefaultConfigFileAndMakesTwoCalls(test *testing.T) {
	harness := taskHarness(test)
	harness.sourcesErr = errors.New("install read a sensor")
	if status := harness.run("service", "install"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	expectTaskFile(test, harness, resolved(test, harness.executablePath), defaultConfigOf(harness))
	want := "wrote " + taskFileOf(harness) + "\nthe startup task sensordeck is created and started\n"
	if harness.stdout.String() != want || harness.stderr.Len() != 0 || len(harness.opened) != 0 {
		test.Errorf("stdout %q, stderr %q, opened %v", harness.stdout.String(), harness.stderr.String(), harness.opened)
	}
	if !reflect.DeepEqual(harness.calls, installCallsOf(harness)) {
		test.Errorf("calls %v", harness.calls)
	}
}

func TestTaskInstallNamesTheConfigFileGivenAbsoluteOrRelative(test *testing.T) {
	directory := test.TempDir()
	writeTestFile(test, filepath.Join(directory, "other.yaml"), "theme: minimal-dark\n")
	harness := taskHarness(test)
	if status := harness.run("service", "install", "--config", directory+"/sub/../other.yaml"); status != 0 {
		test.Fatalf("absolute: status %d, %q", status, harness.stderr.String())
	}
	expectTaskFile(test, harness, resolved(test, harness.executablePath), filepath.Join(directory, "other.yaml"))

	harness = taskHarness(test)
	working, err := os.Getwd()
	if err != nil {
		test.Fatal(err)
	}
	relative, err := filepath.Rel(working, filepath.Join(directory, "other.yaml"))
	if err != nil {
		test.Fatal(err)
	}
	if status := harness.run("service", "install", "--config", relative); status != 0 {
		test.Fatalf("relative: status %d, %q", status, harness.stderr.String())
	}
	expectTaskFile(test, harness, resolved(test, harness.executablePath), filepath.Join(directory, "other.yaml"))
}

func TestTaskInstallEscapesPathsWithASpaceAndXMLCharacters(test *testing.T) {
	harness := taskHarness(test)
	executable := filepath.Join(test.TempDir(), "a b & <c>", "sensordeck")
	writeTestFile(test, executable, "#!/bin/sh\n")
	config := filepath.Join(test.TempDir(), "d e > f", "config.yaml")
	writeTestFile(test, config, "theme: minimal-dark\n")
	harness.executablePath = executable
	if status := harness.run("service", "install", "--config", config); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	expectTaskFile(test, harness, resolved(test, executable), config)
	content := decodeUTF16(readTestFile(test, taskFileOf(harness))[2:])
	for _, part := range []string{"a b &amp; &lt;c&gt;", "d e &gt; f"} {
		if !strings.Contains(content, part) {
			test.Errorf("the task file does not hold %s", part)
		}
	}
}

func TestTaskInstallRefusesAPathWithAQuoteAPercentOrALineFeed(test *testing.T) {
	for _, name := range []string{`a"b`, "a%b", "a\nb"} {
		for _, inExecutable := range []bool{true, false} {
			harness := taskHarness(test)
			path := filepath.Join(test.TempDir(), name, "file")
			writeTestFile(test, path, "theme: minimal-dark\n")
			args := []string{"service", "install", "--config", path}
			if inExecutable {
				harness.executablePath = path
				args = args[:2]
			}
			want := "cannot write the path " + strconv.Quote(resolvedPath(test, path, inExecutable)) + ` into the task file: it holds ", % or a control character`
			harness.expectFailure(test, want, args...)
			expectNoTaskFileAndNoCall(test, harness)
		}
	}
}

// resolvedPath gives the path as the file names it: the executable has its links resolved.
func resolvedPath(test *testing.T, path string, executable bool) string {
	if executable {
		return resolved(test, path)
	}
	return path
}

func TestTaskInstallFailsWithoutACacheDirectoryAUserOrAnExecutable(test *testing.T) {
	harness := taskHarness(test)
	harness.cacheDirErr = errors.New("no cache")
	harness.expectFailure(test, "cannot find the cache directory: no cache", "service", "install")
	if len(harness.calls) != 0 {
		test.Errorf("calls %v", harness.calls)
	}

	harness = taskHarness(test)
	harness.userErr = errors.New("no such user")
	harness.expectFailure(test, "cannot find the user: no such user", "service", "install")
	expectNoTaskFileAndNoCall(test, harness)

	harness = taskHarness(test)
	harness.executableErr = errors.New("no executable")
	harness.expectFailure(test, "cannot find the path of the executable: no executable", "service", "install")
	expectNoTaskFileAndNoCall(test, harness)
}

func TestTaskInstallChecksTheConfigFileAsRunDoesAndWritesNothingWhenItFails(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "missing.yaml")
	hint := "; sensordeck theme use <theme> creates it"
	for name, scenario := range map[string]struct{ content, hint string }{
		"a missing file":                {"", hint},
		"a file that does not parse":    {"tags: [", ""},
		"a theme that does not compile": {"theme: no-such-theme\n", ""},
	} {
		harness := taskHarness(test)
		configPath := missing
		if name != "a missing file" {
			configPath = filepath.Join(test.TempDir(), "config.yaml")
			writeTestFile(test, configPath, scenario.content)
		}
		want := strings.TrimSuffix(runErrorOf(test, configPath), "\n") + scenario.hint + "\n"
		if status := harness.run("service", "install", "--config", configPath); status != 1 || harness.stderr.String() != want {
			test.Errorf("%s: status %d, stderr %q, want %q", name, status, harness.stderr.String(), want)
		}
		expectNoTaskFileAndNoCall(test, harness)
	}
}

func TestTaskInstallStopsARunningBackgroundRunBeforeItWritesTheFileAndReplacesTheFile(test *testing.T) {
	harness := taskHarness(test)
	writeTestFile(test, taskFileOf(harness), "something else\n")
	harness.backgroundRun = func(check int) (bool, error) {
		if got := readTestFile(test, taskFileOf(harness)); got != "something else\n" {
			test.Errorf("check %d: the task file was written before the run ended", check)
		}
		if check < 3 {
			return true, nil
		}
		harness.steps = append(harness.steps, "end of the event")
		return false, nil
	}
	if status := harness.run("service", "install"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	want := []string{"signal", "end of the event", "schtasks /Create /TN sensordeck /XML " + taskFileOf(harness) + " /F", "schtasks /Run /TN sensordeck"}
	if !reflect.DeepEqual(harness.steps, want) {
		test.Errorf("steps %q, want %q", harness.steps, want)
	}
	expectTaskFile(test, harness, resolved(test, harness.executablePath), defaultConfigOf(harness))
}

func TestTaskInstallFailsWhenTheTaskFileCannotBeWrittenAndMakesNoCall(test *testing.T) {
	harness := taskHarness(test)
	writeTestFile(test, filepath.Join(harness.cacheDir, "sensordeck"), "a file\n")
	if status := harness.run("service", "install"); status != 1 ||
		!strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot write the task file "+taskFileOf(harness)+": ") ||
		strings.Count(harness.stderr.String(), taskFileOf(harness)) != 1 {
		test.Errorf("status %d, %q", status, harness.stderr.String())
	}
	if len(harness.calls) != 0 || harness.stdout.Len() != 0 {
		test.Errorf("calls %v, stdout %q", harness.calls, harness.stdout.String())
	}
}

func TestTaskInstallStopsAtTheFirstCallThatFailsAndKeepsTheTaskFile(test *testing.T) {
	for failing := 0; failing < 2; failing++ {
		for _, stderr := range []string{"  the reason\n", ""} {
			harness := taskHarness(test)
			harness.answer = func(arguments []string) (string, string, error) {
				if len(harness.calls) == failing+1 {
					return "", stderr, errors.New("exit status 1")
				}
				return "", "", nil
			}
			cause := "the reason"
			if stderr == "" {
				cause = "exit status 1"
			}
			want := "schtasks " + strings.Join(installCallsOf(harness)[failing], " ") + " failed: " + cause
			if status := harness.run("service", "install"); status != 1 || harness.stderr.String() != "sensordeck: "+want+"\n" {
				test.Errorf("call %d: status %d, stderr %q, want %q", failing, status, harness.stderr.String(), want)
			}
			if !reflect.DeepEqual(harness.calls, installCallsOf(harness)[:failing+1]) {
				test.Errorf("call %d: calls %v", failing, harness.calls)
			}
			if _, err := os.Stat(taskFileOf(harness)); err != nil {
				test.Errorf("call %d: the task file was not kept: %v", failing, err)
			}
			if strings.Contains(harness.stdout.String(), "created and started") {
				test.Errorf("call %d: stdout %q", failing, harness.stdout.String())
			}
		}
	}
}

func TestTaskCallsThatCannotStartGiveTheErrorOfTheStartAndExitStatusesAreNamed(test *testing.T) {
	_, startErr := exec.LookPath("no-such-schtasks-in-this-test")
	harness := taskHarness(test)
	harness.answer = func([]string) (string, string, error) { return "", "", startErr }
	harness.run("service", "install")
	if want := "sensordeck: schtasks /Create /TN sensordeck /XML " + taskFileOf(harness) + " /F failed: " + startErr.Error() + "\n"; harness.stderr.String() != want {
		test.Errorf("stderr %q, want %q", harness.stderr.String(), want)
	}
}

func TestTaskUninstallWithNoTaskFileSaysSoAndTouchesNothing(test *testing.T) {
	harness := taskHarness(test)
	if status := harness.run("service", "uninstall"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	if harness.stdout.String() != "the startup task sensordeck is not installed\n" || harness.stderr.Len() != 0 ||
		len(harness.calls) != 0 || harness.signals != 0 || harness.checks != 0 {
		test.Errorf("stdout %q, stderr %q, calls %v, signals %d, checks %d", harness.stdout.String(), harness.stderr.String(), harness.calls, harness.signals, harness.checks)
	}
}

func TestTaskUninstallStopsDeletesAndRemovesTheFileInThatOrder(test *testing.T) {
	harness := taskHarness(test)
	writeTestFile(test, taskFileOf(harness), "task\n")
	harness.backgroundRun = func(check int) (bool, error) {
		if check == 1 {
			return true, nil
		}
		harness.steps = append(harness.steps, "end of the event")
		return false, nil
	}
	harness.answer = func(arguments []string) (string, string, error) {
		if _, err := os.Stat(taskFileOf(harness)); err != nil {
			test.Errorf("the task file was removed before the call: %v", err)
		}
		return "", "", nil
	}
	if status := harness.run("service", "uninstall"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	want := []string{"signal", "end of the event", "schtasks /Delete /TN sensordeck /F"}
	if !reflect.DeepEqual(harness.steps, want) || harness.stdout.String() != "the startup task sensordeck is stopped and removed\n" {
		test.Errorf("steps %q, stdout %q", harness.steps, harness.stdout.String())
	}
	if _, err := os.Stat(taskFileOf(harness)); !errors.Is(err, os.ErrNotExist) {
		test.Errorf("task file: %v", err)
	}
}

func TestTaskUninstallFailuresKeepTheFileAndMakeNoFurtherCall(test *testing.T) {
	cases := map[string]func(harness *harness) (want string){
		"a failed signal": func(harness *harness) string {
			harness.backgroundRun = func(int) (bool, error) { return true, nil }
			harness.signalErr = errors.New("cannot signal the stop event x: denied")
			return "cannot signal the stop event x: denied"
		},
		"a failed check": func(harness *harness) string {
			harness.backgroundRun = func(int) (bool, error) { return false, errors.New("cannot check the stop event x: denied") }
			return "cannot check the stop event x: denied"
		},
		"a failed delete": func(harness *harness) string {
			harness.answer = func([]string) (string, string, error) { return "", "no such task\n", errors.New("exit status 1") }
			return "schtasks /Delete /TN sensordeck /F failed: no such task"
		},
	}
	for name, setup := range cases {
		harness := taskHarness(test)
		writeTestFile(test, taskFileOf(harness), "task\n")
		harness.expectFailure(test, setup(harness), "service", "uninstall")
		if _, err := os.Stat(taskFileOf(harness)); err != nil {
			test.Errorf("%s: the task file was not kept: %v", name, err)
		}
		if len(harness.calls) > 1 {
			test.Errorf("%s: calls %v", name, harness.calls)
		}
	}
}

func TestTaskUninstallFailureOfTheRemovalLeavesTheTaskDeleted(test *testing.T) {
	harness := taskHarness(test)
	writeTestFile(test, filepath.Join(taskFileOf(harness), "inside"), "a file in a directory\n")
	if status := harness.run("service", "uninstall"); status != 1 ||
		!strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot remove the task file "+taskFileOf(harness)+": ") ||
		strings.Count(harness.stderr.String(), taskFileOf(harness)) != 1 {
		test.Errorf("status %d, %q", status, harness.stderr.String())
	}
	if !reflect.DeepEqual(harness.calls, [][]string{{"/Delete", "/TN", "sensordeck", "/F"}}) {
		test.Errorf("calls %v", harness.calls)
	}
}

func TestTaskUninstallAndStatusFailWithoutACacheDirectoryOrWhenThePresenceCannotBeChecked(test *testing.T) {
	for _, name := range []string{"uninstall", "status"} {
		harness := taskHarness(test)
		harness.cacheDirErr = errors.New("no cache")
		harness.expectFailure(test, "cannot find the cache directory: no cache", "service", name)

		harness = taskHarness(test)
		writeTestFile(test, filepath.Join(harness.cacheDir, "sensordeck"), "a file\n")
		if status := harness.run("service", name); status != 1 ||
			!strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot read the task file "+taskFileOf(harness)+": ") {
			test.Errorf("%s: status %d, %q", name, status, harness.stderr.String())
		}
		if len(harness.calls) != 0 {
			test.Errorf("%s: calls %v", name, harness.calls)
		}
	}
}

func TestTaskStatusWithNoTaskFileSaysSoAndMakesNoCall(test *testing.T) {
	harness := taskHarness(test)
	if status := harness.run("service", "status"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if harness.stdout.String() != "installed: no\n" || len(harness.calls) != 0 || harness.checks != 0 {
		test.Errorf("stdout %q, calls %v, checks %d", harness.stdout.String(), harness.calls, harness.checks)
	}
}

func TestTaskStatusWritesTheFourLines(test *testing.T) {
	disabled := taskOutput("<Enabled>false</Enabled>")
	cases := []struct {
		name, output string
		found        bool
		checkErr     error
		want         string
	}{
		{"enabled and running in UTF-8", taskOutput("<Enabled>true</Enabled>"), true, nil, "enabled: enabled\nactive: running\n"},
		{"disabled and stopped in UTF-16", inUTF16(disabled), false, nil, "enabled: disabled\nactive: stopped\n"},
		{"no Enabled element", taskOutput(""), true, nil, "enabled: enabled\nactive: running\n"},
		{"not XML", "garbage", false, nil, "enabled: unknown\nactive: stopped\n"},
		{"a failed check", disabled, false, errors.New("denied"), "enabled: disabled\nactive: unknown\n"},
	}
	for _, scenario := range cases {
		harness := taskHarness(test)
		writeTestFile(test, taskFileOf(harness), "task\n")
		harness.answer = func([]string) (string, string, error) { return scenario.output, "", nil }
		harness.backgroundRun = func(int) (bool, error) { return scenario.found, scenario.checkErr }
		if status := harness.run("service", "status"); status != 0 {
			test.Fatalf("%s: status %d", scenario.name, status)
		}
		want := "installed: yes, " + taskFileOf(harness) + "\n" + scenario.want + "log: " + logPath(harness.cacheDir) + "\n"
		if harness.stdout.String() != want || !reflect.DeepEqual(harness.calls, [][]string{{"/Query", "/TN", "sensordeck", "/XML"}}) {
			test.Errorf("%s: stdout %q, calls %v", scenario.name, harness.stdout.String(), harness.calls)
		}
	}
}

func TestTaskStatusFailureOfTheCallFailsAsInstallDoes(test *testing.T) {
	harness := taskHarness(test)
	writeTestFile(test, taskFileOf(harness), "task\n")
	harness.answer = func([]string) (string, string, error) { return "", "", errors.New("exit status 1") }
	harness.expectFailure(test, "schtasks /Query /TN sensordeck /XML failed: exit status 1", "service", "status")
}

func TestTheStopEndsAfterThreeChecksWithNoEndOfTheTask(test *testing.T) {
	harness := taskHarness(test)
	harness.backgroundRun = func(check int) (bool, error) { return check < 3, nil }
	if status := harness.run("service", "install"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	if harness.checks != 3 || harness.signals != 1 || !reflect.DeepEqual(harness.waits, []time.Duration{200 * time.Millisecond, 200 * time.Millisecond}) {
		test.Errorf("%d checks, %d signals, waits %v", harness.checks, harness.signals, harness.waits)
	}
	if !reflect.DeepEqual(harness.calls, installCallsOf(harness)) || strings.Contains(harness.stdout.String(), "did not stop") {
		test.Errorf("calls %v, stdout %q", harness.calls, harness.stdout.String())
	}
}

func TestARunThatNeverEndsIsEndedWithTheTaskAfterThirtySeconds(test *testing.T) {
	harness := taskHarness(test)
	harness.backgroundRun = func(int) (bool, error) { return true, nil }
	if status := harness.run("service", "install"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	var total time.Duration
	for _, wait := range harness.waits {
		total += wait
	}
	if total != 30*time.Second {
		test.Errorf("waited %v", total)
	}
	wantSteps := []string{"signal", "schtasks /End /TN sensordeck", "schtasks /Create /TN sensordeck /XML " + taskFileOf(harness) + " /F", "schtasks /Run /TN sensordeck"}
	if !reflect.DeepEqual(harness.steps, wantSteps) {
		test.Errorf("steps %q", harness.steps)
	}
	line := "sensordeck run did not stop within 30 seconds and was ended with schtasks /End\n"
	if !strings.HasPrefix(harness.stdout.String(), line) {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestAFailedEndOfTheTaskFailsTheStopBeforeAnyFileIsWritten(test *testing.T) {
	harness := taskHarness(test)
	harness.backgroundRun = func(int) (bool, error) { return true, nil }
	harness.answer = func([]string) (string, string, error) { return "", "access denied\n", errors.New("exit status 1") }
	harness.expectFailure(test, "schtasks /End /TN sensordeck failed: access denied", "service", "install")
	if _, err := os.Stat(taskFileOf(harness)); !errors.Is(err, os.ErrNotExist) {
		test.Errorf("task file: %v", err)
	}
}

func TestAStopEventThatCannotBeSignalledOrCheckedFailsInstall(test *testing.T) {
	harness := taskHarness(test)
	harness.backgroundRun = func(int) (bool, error) { return true, nil }
	harness.signalErr = errors.New("cannot signal the stop event x: denied")
	harness.expectFailure(test, "cannot signal the stop event x: denied", "service", "install")
	expectNoTaskFileAndNoCallExceptSignal(test, harness)

	harness = taskHarness(test)
	harness.backgroundRun = func(int) (bool, error) { return false, errors.New("cannot check the stop event x: denied") }
	harness.expectFailure(test, "cannot check the stop event x: denied", "service", "install")
	expectNoTaskFileAndNoCall(test, harness)
}

func expectNoTaskFileAndNoCallExceptSignal(test *testing.T, harness *harness) {
	test.Helper()
	if _, err := os.Stat(taskFileOf(harness)); !errors.Is(err, os.ErrNotExist) || len(harness.calls) != 0 {
		test.Errorf("task file: %v, calls %v", err, harness.calls)
	}
}

func TestTheHelpOfServiceNamesTheTaskOnWindowsAndTheSystemdServiceOnLinux(test *testing.T) {
	for system, want := range map[string]string{"windows": "startup task", "linux": "systemd user service"} {
		for _, args := range [][]string{{"service", "--help"}, {"service", "install", "--help"}, {"service", "status", "--help"}, {"service", "uninstall", "--help"}} {
			harness := taskHarness(test)
			harness.system = system
			wanted := want
			if system == "linux" && args[1] == "status" {
				wanted = "service" // the help of status on Linux is as it was, and does not name systemd
			}
			if status := harness.run(args...); status != 0 || !strings.Contains(harness.stdout.String(), wanted) {
				test.Errorf("%s %v: status %d, help %q", system, args, status, harness.stdout.String())
			}
		}
	}
	harness := taskHarness(test)
	if status := harness.run("service"); status != 0 || !strings.Contains(harness.stdout.String(), "uninstall") || harness.stderr.Len() != 0 {
		test.Errorf("service on windows: status %d, %q, %q", status, harness.stdout.String(), harness.stderr.String())
	}
}
