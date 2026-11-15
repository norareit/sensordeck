package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// serviceHarness gives a harness whose config directory is a temporary one that holds sensordeck/config.yaml with a
// built-in theme, and whose executable is a file in another temporary directory.
func serviceHarness(test *testing.T) *harness {
	test.Helper()
	harness := newHarness()
	harness.configDir = test.TempDir()
	harness.executablePath = filepath.Join(test.TempDir(), "sensordeck")
	writeTestFile(test, harness.executablePath, "#!/bin/sh\n")
	writeTestFile(test, filepath.Join(harness.configDir, "sensordeck", "config.yaml"), "theme: minimal-dark\n")
	return harness
}

func writeTestFile(test *testing.T, path, content string) {
	test.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
}

func readTestFile(test *testing.T, path string) string {
	test.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	return string(content)
}

func unitFileOf(harness *harness) string { return unitFilePath(harness.configDir) }

func expectUnitFile(test *testing.T, harness *harness, executable, config string) {
	test.Helper()
	want, err := unitFileContent(executable, config)
	if err != nil {
		test.Fatal(err)
	}
	if got := readTestFile(test, unitFileOf(harness)); got != want {
		test.Errorf("unit file %q, want %q", got, want)
	}
}

func expectNoUnitFileAndNoCall(test *testing.T, harness *harness) {
	test.Helper()
	if _, err := os.Stat(unitFileOf(harness)); !errors.Is(err, os.ErrNotExist) {
		test.Errorf("unit file: %v", err)
	}
	if len(harness.calls) != 0 {
		test.Errorf("calls %v", harness.calls)
	}
}

var installCalls = [][]string{{"daemon-reload"}, {"enable", "sensordeck.service"}, {"restart", "sensordeck.service"}}

func TestInstallWritesTheUnitFileNamingTheDefaultConfigFileAndMakesThreeCalls(test *testing.T) {
	harness := serviceHarness(test)
	harness.sourcesErr = errors.New("install read a sensor")
	if status := harness.run("service", "install"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	expectUnitFile(test, harness, harness.executablePath, filepath.Join(harness.configDir, "sensordeck", "config.yaml"))
	want := "wrote " + unitFileOf(harness) + "\nsensordeck.service is enabled and started\n"
	if harness.stdout.String() != want || harness.stderr.Len() != 0 {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
	if !reflect.DeepEqual(harness.calls, installCalls) {
		test.Errorf("calls %v", harness.calls)
	}
	if len(harness.opened) != 0 {
		test.Errorf("opened %v", harness.opened)
	}
}

func TestInstallNamesTheConfigFileGivenAbsoluteOrRelativeAndCleaned(test *testing.T) {
	harness := serviceHarness(test)
	directory := test.TempDir()
	writeTestFile(test, filepath.Join(directory, "other.yaml"), "theme: minimal-dark\n")
	if status := harness.run("service", "install", "--config", directory+"/sub/../other.yaml"); status != 0 {
		test.Fatalf("absolute: status %d, %q", status, harness.stderr.String())
	}
	expectUnitFile(test, harness, harness.executablePath, filepath.Join(directory, "other.yaml"))

	harness = serviceHarness(test)
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
	expectUnitFile(test, harness, harness.executablePath, filepath.Join(directory, "other.yaml"))
}

func TestInstallKeepsAConfigFileThatIsALinkAndResolvesALinkToTheExecutable(test *testing.T) {
	harness := serviceHarness(test)
	directory := test.TempDir()
	writeTestFile(test, filepath.Join(directory, "real.yaml"), "theme: minimal-dark\n")
	link := filepath.Join(directory, "config.yaml")
	if err := os.Symlink(filepath.Join(directory, "real.yaml"), link); err != nil {
		test.Fatal(err)
	}
	executableLink := filepath.Join(directory, "sensordeck-link")
	if err := os.Symlink(harness.executablePath, executableLink); err != nil {
		test.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(harness.executablePath)
	if err != nil {
		test.Fatal(err)
	}
	harness.executablePath = executableLink
	if status := harness.run("service", "install", "--config", link); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	expectUnitFile(test, harness, resolved, link)
}

func TestInstallWritesPathsWithSpecialCharactersAsTheSpecSays(test *testing.T) {
	harness := serviceHarness(test)
	executableDirectory := filepath.Join(test.TempDir(), `a b"c\d%e$f`)
	executable := filepath.Join(executableDirectory, "sensordeck")
	writeTestFile(test, executable, "#!/bin/sh\n")
	configPath := filepath.Join(test.TempDir(), `g h"i\j%k$l`, "config.yaml")
	writeTestFile(test, configPath, "theme: minimal-dark\n")
	harness.executablePath = executable
	if status := harness.run("service", "install", "--config", configPath); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		test.Fatal(err)
	}
	expectUnitFile(test, harness, resolved, configPath)
	content := readTestFile(test, unitFileOf(harness))
	for _, part := range []string{`a b\"c\\d%%e$$f`, `g h\"i\\j%%k$$l`} {
		if !strings.Contains(content, part) {
			test.Errorf("the unit file does not hold %s: %q", part, content)
		}
	}
}

func TestInstallRefusesAConfigPathWithALineFeed(test *testing.T) {
	harness := serviceHarness(test)
	configPath := filepath.Join(test.TempDir(), "a\nb.yaml")
	writeTestFile(test, configPath, "theme: minimal-dark\n")
	harness.expectFailure(test, `cannot write the path `+`"`+strings.ReplaceAll(configPath, "\n", `\n`)+`"`+` into the unit file: it holds a control character`,
		"service", "install", "--config", configPath)
	expectNoUnitFileAndNoCall(test, harness)
}

func TestInstallFailsWithoutAConfigDirectoryAndWritesNothing(test *testing.T) {
	harness := serviceHarness(test)
	harness.configDirErr = errors.New("no home")
	harness.expectFailure(test, "cannot find the config directory: no home", "service", "install")
	harness.stderr.Reset()
	harness.expectFailure(test, "cannot find the config directory: no home", "service", "install", "--config", "/x.yaml")
	if len(harness.calls) != 0 {
		test.Errorf("calls %v", harness.calls)
	}
}

func TestInstallFailsWithoutTheExecutableAndWritesNothing(test *testing.T) {
	harness := serviceHarness(test)
	harness.executableErr = errors.New("no executable")
	harness.expectFailure(test, "cannot find the path of the executable: no executable", "service", "install")
	expectNoUnitFileAndNoCall(test, harness)

	harness = serviceHarness(test)
	harness.executablePath = filepath.Join(test.TempDir(), "dangling")
	if err := os.Symlink(filepath.Join(test.TempDir(), "nowhere"), harness.executablePath); err != nil {
		test.Fatal(err)
	}
	if status := harness.run("service", "install"); status != 1 || !strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot find the path of the executable: ") {
		test.Errorf("dangling: status %d, %q", status, harness.stderr.String())
	}
	expectNoUnitFileAndNoCall(test, harness)
}

// runErrorOf gives the standard error of run with the config file in another harness, so that a test compares the text with
// the one of run.
func runErrorOf(test *testing.T, configPath string) string {
	test.Helper()
	other := newHarness()
	if status := other.run("run", "--config", configPath); status != 1 {
		test.Fatalf("run: status %d", status)
	}
	return other.stderr.String()
}

func TestInstallChecksTheConfigFileAsRunDoesAndWritesNothingWhenItFails(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "missing.yaml")
	hint := "; sensordeck theme use <theme> creates it"
	for name, tc := range map[string]struct{ content, hint string }{
		"a missing file":                {"", hint},
		"a file that does not parse":    {"tags: [", ""},
		"a theme that does not compile": {"theme: no-such-theme\n", ""},
	} {
		harness := serviceHarness(test)
		configPath := missing
		if name != "a missing file" {
			configPath = filepath.Join(test.TempDir(), "config.yaml")
			writeTestFile(test, configPath, tc.content)
		}
		want := strings.TrimSuffix(runErrorOf(test, configPath), "\n") + tc.hint + "\n"
		if status := harness.run("service", "install", "--config", configPath); status != 1 {
			test.Errorf("%s: status %d", name, status)
		}
		if harness.stderr.String() != want {
			test.Errorf("%s: stderr %q, want %q", name, harness.stderr.String(), want)
		}
		expectNoUnitFileAndNoCall(test, harness)
	}
}

func TestInstallWithNoConfigFileGivesTheHint(test *testing.T) {
	harness := serviceHarness(test)
	configPath := filepath.Join(harness.configDir, "sensordeck", "config.yaml")
	if err := os.Remove(configPath); err != nil {
		test.Fatal(err)
	}
	if status := harness.run("service", "install"); status != 1 || !strings.HasSuffix(harness.stderr.String(), "; sensordeck theme use <theme> creates it\n") {
		test.Errorf("status %d, %q", status, harness.stderr.String())
	}
	expectNoUnitFileAndNoCall(test, harness)
}

func TestInstallMakesTheCallsAfterTheFileIsWrittenAndReplacesAnExistingFile(test *testing.T) {
	harness := serviceHarness(test)
	writeTestFile(test, unitFileOf(harness), "something else\n")
	harness.answer = func(arguments []string) (string, string, error) {
		if len(harness.calls) == 1 {
			expectUnitFile(test, harness, harness.executablePath, filepath.Join(harness.configDir, "sensordeck", "config.yaml"))
		}
		return "", "", nil
	}
	if status := harness.run("service", "install"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	if !reflect.DeepEqual(harness.calls, installCalls) {
		test.Errorf("calls %v", harness.calls)
	}
}

func TestInstallCreatesTheDirectoryOfTheUnitFile(test *testing.T) {
	harness := serviceHarness(test)
	if _, err := os.Stat(filepath.Join(harness.configDir, "systemd")); !errors.Is(err, os.ErrNotExist) {
		test.Fatalf("the directory exists: %v", err)
	}
	if status := harness.run("service", "install"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if _, err := os.Stat(unitFileOf(harness)); err != nil {
		test.Error(err)
	}
}

func TestInstallFailsWhenTheUnitFileCannotBeWrittenAndMakesNoCall(test *testing.T) {
	harness := serviceHarness(test)
	writeTestFile(test, filepath.Join(harness.configDir, "systemd", "user"), "a file\n")
	if status := harness.run("service", "install"); status != 1 ||
		!strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot write the unit file "+unitFileOf(harness)+": ") ||
		strings.Count(harness.stderr.String(), unitFileOf(harness)) != 1 {
		test.Errorf("status %d, %q", status, harness.stderr.String())
	}
	if len(harness.calls) != 0 || harness.stdout.Len() != 0 {
		test.Errorf("calls %v, stdout %q", harness.calls, harness.stdout.String())
	}
}

func TestInstallStopsAtTheFirstCallThatFailsAndKeepsTheUnitFile(test *testing.T) {
	for failing := range installCalls {
		harness := serviceHarness(test)
		harness.answer = func(arguments []string) (string, string, error) {
			if len(harness.calls) == failing+1 {
				return "", "  the reason\n", errors.New("exit status 1")
			}
			return "", "", nil
		}
		want := "systemctl --user " + strings.Join(installCalls[failing], " ") + " failed: the reason"
		harness.stdout.Reset()
		if status := harness.run("service", "install"); status != 1 || harness.stderr.String() != "sensordeck: "+want+"\n" {
			test.Errorf("call %d: status %d, stderr %q", failing, status, harness.stderr.String())
		}
		if !reflect.DeepEqual(harness.calls, installCalls[:failing+1]) {
			test.Errorf("call %d: calls %v", failing, harness.calls)
		}
		if _, err := os.Stat(unitFileOf(harness)); err != nil {
			test.Errorf("call %d: the unit file was not kept: %v", failing, err)
		}
		if strings.Contains(harness.stdout.String(), "enabled and started") {
			test.Errorf("call %d: stdout %q", failing, harness.stdout.String())
		}
	}
}

func TestACallThatWritesNothingGivesItsExitStatusAndACallThatCannotStartGivesTheErrorOfTheStart(test *testing.T) {
	exitErr := exec.Command("sh", "-c", "exit 3").Run()
	if exitErr == nil {
		test.Fatal("no error from exit 3")
	}
	harness := serviceHarness(test)
	harness.answer = func([]string) (string, string, error) { return "", "", exitErr }
	expectCallFailure(test, harness, "systemctl --user daemon-reload failed: exit status 3")

	_, startErr := exec.LookPath("no-such-systemctl-in-this-test")
	harness = serviceHarness(test)
	harness.answer = func([]string) (string, string, error) { return "", "", startErr }
	expectCallFailure(test, harness, "systemctl --user daemon-reload failed: "+startErr.Error())
}

// expectCallFailure runs install, whose first call fails, and checks the error. Standard output holds the line of the
// file written, which a failure after the write leaves.
func expectCallFailure(test *testing.T, harness *harness, want string) {
	test.Helper()
	if status := harness.run("service", "install"); status != 1 || harness.stderr.String() != "sensordeck: "+want+"\n" {
		test.Errorf("status %d, stderr %q, want %q", status, harness.stderr.String(), want)
	}
}

func TestUninstallWithNoUnitFileSaysSoAndMakesNoCall(test *testing.T) {
	harness := serviceHarness(test)
	if status := harness.run("service", "uninstall"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	if harness.stdout.String() != "sensordeck.service is not installed\n" || harness.stderr.Len() != 0 || len(harness.calls) != 0 {
		test.Errorf("stdout %q, stderr %q, calls %v", harness.stdout.String(), harness.stderr.String(), harness.calls)
	}
}

func TestUninstallDisablesRemovesTheFileAndReloads(test *testing.T) {
	harness := serviceHarness(test)
	writeTestFile(test, unitFileOf(harness), "unit\n")
	harness.answer = func(arguments []string) (string, string, error) {
		_, err := os.Stat(unitFileOf(harness))
		removed := errors.Is(err, os.ErrNotExist)
		if removed != (len(harness.calls) == 2) {
			test.Errorf("call %v: the unit file removed is %v", arguments, removed)
		}
		return "", "", nil
	}
	if status := harness.run("service", "uninstall"); status != 0 {
		test.Fatalf("status %d, %q", status, harness.stderr.String())
	}
	want := [][]string{{"disable", "--now", "sensordeck.service"}, {"daemon-reload"}}
	if !reflect.DeepEqual(harness.calls, want) || harness.stdout.String() != "sensordeck.service is stopped, disabled and removed\n" {
		test.Errorf("calls %v, stdout %q", harness.calls, harness.stdout.String())
	}
}

func TestUninstallFailureOfTheDisableKeepsTheFileAndMakesNoFurtherCall(test *testing.T) {
	harness := serviceHarness(test)
	writeTestFile(test, unitFileOf(harness), "unit\n")
	harness.answer = func([]string) (string, string, error) { return "", "no such unit\n", errors.New("exit status 1") }
	harness.expectFailure(test, "systemctl --user disable --now sensordeck.service failed: no such unit", "service", "uninstall")
	if _, err := os.Stat(unitFileOf(harness)); err != nil || len(harness.calls) != 1 {
		test.Errorf("unit file %v, calls %v", err, harness.calls)
	}
}

func TestUninstallFailureOfTheRemovalMakesNoFurtherCall(test *testing.T) {
	harness := serviceHarness(test)
	writeTestFile(test, filepath.Join(unitFileOf(harness), "inside"), "a file in a directory\n")
	if status := harness.run("service", "uninstall"); status != 1 ||
		!strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot remove the unit file "+unitFileOf(harness)+": ") ||
		strings.Count(harness.stderr.String(), unitFileOf(harness)) != 1 {
		test.Errorf("status %d, %q", status, harness.stderr.String())
	}
	if len(harness.calls) != 1 {
		test.Errorf("calls %v", harness.calls)
	}
}

func TestUninstallFailureOfTheLastCallLeavesTheFileRemoved(test *testing.T) {
	harness := serviceHarness(test)
	writeTestFile(test, unitFileOf(harness), "unit\n")
	harness.answer = func(arguments []string) (string, string, error) {
		if arguments[0] == "daemon-reload" {
			return "", "bus down\n", errors.New("exit status 1")
		}
		return "", "", nil
	}
	harness.expectFailure(test, "systemctl --user daemon-reload failed: bus down", "service", "uninstall")
	if _, err := os.Stat(unitFileOf(harness)); !errors.Is(err, os.ErrNotExist) {
		test.Errorf("unit file: %v", err)
	}
}

func TestUninstallAndStatusFailWithoutAConfigDirectoryAndMakeNoCall(test *testing.T) {
	for _, name := range []string{"uninstall", "status"} {
		harness := serviceHarness(test)
		harness.configDirErr = errors.New("no home")
		harness.expectFailure(test, "cannot find the config directory: no home", "service", name)
		if len(harness.calls) != 0 {
			test.Errorf("%s: calls %v", name, harness.calls)
		}
	}
}

func TestUninstallAndStatusFailWhenThePresenceOfTheUnitFileCannotBeChecked(test *testing.T) {
	for _, name := range []string{"uninstall", "status"} {
		harness := serviceHarness(test)
		// A file where the directory systemd is expected makes the check fail with ENOTDIR and not with an absence.
		writeTestFile(test, filepath.Join(harness.configDir, "systemd"), "a file\n")
		if status := harness.run("service", name); status != 1 ||
			!strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot read the unit file "+unitFileOf(harness)+": ") {
			test.Errorf("%s: status %d, %q", name, status, harness.stderr.String())
		}
		if len(harness.calls) != 0 {
			test.Errorf("%s: calls %v", name, harness.calls)
		}
	}
}

func TestStatusWithNoUnitFileSaysSoAndMakesNoCall(test *testing.T) {
	harness := serviceHarness(test)
	if status := harness.run("service", "status"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if harness.stdout.String() != "installed: no\n" || len(harness.calls) != 0 {
		test.Errorf("stdout %q, calls %v", harness.stdout.String(), harness.calls)
	}
}

func TestStatusWritesTheThreeLinesFromOneCall(test *testing.T) {
	for name, tc := range map[string]struct{ output, want string }{
		"three properties":   {"UnitFileState=enabled\nActiveState=active\nSubState=running\n", "enabled: enabled\nactive: active (running)\n"},
		"a missing property": {"ActiveState=inactive\nSubState=dead\n", "enabled: unknown\nactive: inactive (dead)\n"},
		"an empty value":     {"UnitFileState=\nActiveState=failed\nSubState=failed\n", "enabled: unknown\nactive: failed (failed)\n"},
	} {
		harness := serviceHarness(test)
		writeTestFile(test, unitFileOf(harness), "unit\n")
		harness.answer = func([]string) (string, string, error) { return tc.output, "", nil }
		if status := harness.run("service", "status"); status != 0 {
			test.Fatalf("%s: status %d", name, status)
		}
		want := "installed: yes, " + unitFileOf(harness) + "\n" + tc.want
		wantCalls := [][]string{{"show", "--property=UnitFileState", "--property=ActiveState", "--property=SubState", "sensordeck.service"}}
		if harness.stdout.String() != want || !reflect.DeepEqual(harness.calls, wantCalls) {
			test.Errorf("%s: stdout %q, calls %v", name, harness.stdout.String(), harness.calls)
		}
	}
}

func TestStatusFailureOfTheCallFailsAsInstallDoes(test *testing.T) {
	harness := serviceHarness(test)
	writeTestFile(test, unitFileOf(harness), "unit\n")
	harness.answer = func([]string) (string, string, error) { return "", "", errors.New("exit status 4") }
	harness.expectFailure(test, "systemctl --user show --property=UnitFileState --property=ActiveState --property=SubState sensordeck.service failed: exit status 4", "service", "status")
}

func TestEveryServiceCommandFailsOnAnotherSystemAndTouchesNothing(test *testing.T) {
	const want = "the service is not supported on darwin"
	for _, args := range [][]string{
		{"service"}, {"service", "install"}, {"service", "install", "--config", "/missing.yaml"}, {"service", "status"}, {"service", "uninstall"},
	} {
		harness := serviceHarness(test)
		harness.system = "darwin"
		harness.expectFailure(test, want, args...)
		if harness.configDirCalls != 0 || harness.executableCalls != 0 || len(harness.calls) != 0 {
			test.Errorf("%v: userConfigDir %d, executable %d, calls %v", args, harness.configDirCalls, harness.executableCalls, harness.calls)
		}
		if harness.cacheDirCalls != 0 || harness.userCalls != 0 || harness.holds != 0 || harness.checks != 0 || harness.signals != 0 || len(harness.steps) != 0 {
			test.Errorf("%v: userCacheDir %d, currentUser %d, holds %d, checks %d, signals %d, steps %v",
				args, harness.cacheDirCalls, harness.userCalls, harness.holds, harness.checks, harness.signals, harness.steps)
		}
		entries, err := os.ReadDir(harness.configDir)
		if err != nil || len(entries) != 1 || entries[0].Name() != "sensordeck" {
			test.Errorf("%v: the config directory holds %v, %v", args, entries, err)
		}
	}
}

func TestServiceHelpNamesTheSubcommandsOnEverySystemAndServiceAloneShowsItOnLinux(test *testing.T) {
	for _, system := range []string{"windows", "linux"} {
		harness := serviceHarness(test)
		harness.system = system
		if status := harness.run("service", "--help"); status != 0 {
			test.Errorf("%s: status %d", system, status)
		}
		for _, name := range []string{"install", "status", "uninstall"} {
			if !strings.Contains(harness.stdout.String(), name) {
				test.Errorf("%s: the help does not name %s", system, name)
			}
		}
	}
	harness := serviceHarness(test)
	if status := harness.run("service"); status != 0 || !strings.Contains(harness.stdout.String(), "uninstall") || harness.stderr.Len() != 0 {
		test.Errorf("service: status %d, %q, %q", status, harness.stdout.String(), harness.stderr.String())
	}
}
