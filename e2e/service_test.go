//go:build linux

package e2e

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/fakescreen"
)

// fakeSystemctl is a directory that holds an executable systemctl, a script that uses only built-ins of the shell. It
// appends its arguments after --user as one line to a log, fails when its arguments are the text it was told to fail at, and writes
// the three properties for show. The tests run with this directory as their whole PATH, so that no real systemctl is
// ever run.
type fakeSystemctl struct {
	directory string
	log       string
}

func newFakeSystemctl(test *testing.T, failAt string) fakeSystemctl {
	test.Helper()
	fake := fakeSystemctl{directory: test.TempDir(), log: filepath.Join(test.TempDir(), "calls.log")}
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" != --user ]; then
  echo "fake systemctl: the first argument is not --user" >&2
  exit 2
fi
shift
echo "$*" >> '%s'
if [ "$*" = '%s' ]; then
  echo "fake failure of $*" >&2
  exit 1
fi
if [ "$1" = show ]; then
  echo UnitFileState=enabled
  echo ActiveState=active
  echo SubState=running
fi
`, fake.log, failAt)
	if err := os.WriteFile(filepath.Join(fake.directory, "systemctl"), []byte(script), 0o755); err != nil {
		test.Fatal(err)
	}
	return fake
}

// calls gives the lines of the log, none when no call was made.
func (fake fakeSystemctl) calls(test *testing.T) []string {
	test.Helper()
	content, err := os.ReadFile(fake.log)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		test.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
}

// serviceSetup is one test of the service: a config directory of its own and a fake systemctl.
type serviceSetup struct {
	test        *testing.T
	configHome  string
	systemctl   fakeSystemctl
	environment []string
}

func newServiceSetup(test *testing.T, failAt string) serviceSetup {
	test.Helper()
	setup := serviceSetup{test: test, configHome: test.TempDir(), systemctl: newFakeSystemctl(test, failAt)}
	setup.environment = []string{"XDG_CONFIG_HOME=" + setup.configHome, "PATH=" + setup.systemctl.directory}
	return setup
}

func (setup serviceSetup) run(args ...string) result {
	setup.test.Helper()
	return runWith(setup.test, fakescreen.New(setup.test), setup.environment, args...)
}

func (setup serviceSetup) unitFile() string {
	return filepath.Join(setup.configHome, "systemd", "user", "sensordeck.service")
}

func (setup serviceSetup) configFile() string {
	return filepath.Join(setup.configHome, "sensordeck", "config.yaml")
}

func (setup serviceSetup) expectNoUnitFile() {
	setup.test.Helper()
	if _, err := os.Stat(setup.unitFile()); !errors.Is(err, os.ErrNotExist) {
		setup.test.Errorf("unit file: %v", err)
	}
}

func equalLines(got, want []string) bool { return strings.Join(got, "\n") == strings.Join(want, "\n") }

var installCalls = []string{"daemon-reload", "enable sensordeck.service", "restart sensordeck.service"}

// install makes the setup the state after a successful install.
func (setup serviceSetup) install() {
	setup.test.Helper()
	if used := setup.run("theme", "use", "minimal-dark"); used.status != 0 {
		setup.test.Fatalf("theme use: %+v", used)
	}
	if installed := setup.run("service", "install"); installed.status != 0 {
		setup.test.Fatalf("service install: %+v", installed)
	}
}

func TestServiceInstallWritesTheUnitFileAndMakesTheThreeCalls(test *testing.T) {
	setup := newServiceSetup(test, "")
	if used := setup.run("theme", "use", "minimal-dark"); used.status != 0 {
		test.Fatalf("theme use: %+v", used)
	}
	installed := setup.run("service", "install")
	want := "wrote " + setup.unitFile() + "\nsensordeck.service is enabled and started\n"
	if installed.status != 0 || installed.stdout != want || installed.stderr != "" {
		test.Errorf("install: %+v", installed)
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		test.Fatal(err)
	}
	content, err := os.ReadFile(setup.unitFile())
	if err != nil {
		test.Fatal(err)
	}
	wantLine := fmt.Sprintf("ExecStart=%q run --config %q\n", resolved, setup.configFile())
	if !strings.Contains(string(content), wantLine) || !strings.HasPrefix(string(content), "# Written by sensordeck service install") {
		test.Errorf("unit file %q does not hold %q", content, wantLine)
	}
	if got := setup.systemctl.calls(test); !equalLines(got, installCalls) {
		test.Errorf("calls %q", got)
	}
}

func TestServiceInstallWithNoConfigFileOrAThemeThatDoesNotCompileWritesNothing(test *testing.T) {
	setup := newServiceSetup(test, "")
	missing := setup.run("service", "install")
	if missing.status != 1 || !strings.Contains(missing.stderr, "cannot read the config file "+setup.configFile()) ||
		!strings.Contains(missing.stderr, "sensordeck theme use <theme> creates it") {
		test.Errorf("no config file: %+v", missing)
	}
	setup.expectNoUnitFile()
	if got := setup.systemctl.calls(test); len(got) != 0 {
		test.Errorf("calls %q", got)
	}

	if err := os.MkdirAll(filepath.Dir(setup.configFile()), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(setup.configFile(), []byte("theme: no-such-theme\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	broken := setup.run("service", "install")
	if broken.status != 1 || !strings.Contains(broken.stderr, "config file "+setup.configFile()+": theme: ") {
		test.Errorf("a theme that does not compile: %+v", broken)
	}
	setup.expectNoUnitFile()
	if got := setup.systemctl.calls(test); len(got) != 0 {
		test.Errorf("calls %q", got)
	}
}

func TestServiceInstallWithACallThatFailsNamesTheCallAndStops(test *testing.T) {
	setup := newServiceSetup(test, "enable sensordeck.service")
	if used := setup.run("theme", "use", "minimal-dark"); used.status != 0 {
		test.Fatalf("theme use: %+v", used)
	}
	failed := setup.run("service", "install")
	if failed.status != 1 || !strings.Contains(failed.stderr, "systemctl --user enable sensordeck.service failed: ") ||
		!strings.Contains(failed.stderr, "fake failure of enable sensordeck.service") {
		test.Errorf("install: %+v", failed)
	}
	if got := setup.systemctl.calls(test); !equalLines(got, installCalls[:2]) {
		test.Errorf("calls %q", got)
	}
}

func TestServiceStatusBeforeAndAfterInstall(test *testing.T) {
	setup := newServiceSetup(test, "")
	before := setup.run("service", "status")
	if before.status != 0 || before.stdout != "installed: no\n" || before.stderr != "" {
		test.Errorf("before: %+v", before)
	}
	if got := setup.systemctl.calls(test); len(got) != 0 {
		test.Errorf("calls before: %q", got)
	}
	setup.install()
	after := setup.run("service", "status")
	want := "installed: yes, " + setup.unitFile() + "\nenabled: enabled\nactive: active (running)\n"
	if after.status != 0 || after.stdout != want || after.stderr != "" {
		test.Errorf("after: %+v", after)
	}
	show := "show --property=UnitFileState --property=ActiveState --property=SubState sensordeck.service"
	if got := setup.systemctl.calls(test); !equalLines(got, append(append([]string{}, installCalls...), show)) {
		test.Errorf("calls %q", got)
	}
}

func TestServiceUninstallRemovesTheUnitFileWithTwoCalls(test *testing.T) {
	setup := newServiceSetup(test, "")
	setup.install()
	removed := setup.run("service", "uninstall")
	if removed.status != 0 || removed.stdout != "sensordeck.service is stopped, disabled and removed\n" || removed.stderr != "" {
		test.Errorf("uninstall: %+v", removed)
	}
	setup.expectNoUnitFile()
	want := append(append([]string{}, installCalls...), "disable --now sensordeck.service", "daemon-reload")
	if got := setup.systemctl.calls(test); !equalLines(got, want) {
		test.Errorf("calls %q", got)
	}
}
