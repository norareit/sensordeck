//go:build linux

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakecommander"
	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/fakesensors"
	"golang.org/x/sys/unix"
)

const controllerInUseLine = "the controller is in use by another sensordeck process, its probes are left out\n"

const serviceStopText = ", stop it first, or stop the service with systemctl --user stop sensordeck\n"

func softwareModeEntries(controller *fakecommander.Controller) (entries, returns int) {
	for _, written := range controller.Writes() {
		switch {
		case bytes.Equal(written, controllerWrite(1)):
			entries++
		case bytes.Equal(written, controllerWrite(8)):
			returns++
		}
	}
	return entries, returns
}

func TestSensorsWhileRunHoldsASessionLeavesTheProbesOutAndTheSessionAlone(test *testing.T) {
	fake := fakescreen.New(test)
	fake.Add(3, "name", "")
	tree, _ := runningTree(test, fake)
	controller := fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)
	config := writeConfigFile(test, "interval: 100ms\ncommander_color: \"#ff0000\"\ntags:\n  cpu.temp: 1\n  commander.1.temp: 4\n")
	environment := environmentFor(test, tree, "")
	running := start(test, fake, environment, "run", "--config", config)
	waitForColourWrite(controller)
	outcome := runWith(test, fake, environment, "sensors")
	if outcome.status != 0 || outcome.stderr != controllerInUseLine {
		test.Errorf("status %d, stderr %q", outcome.status, outcome.stderr)
	}
	if !strings.Contains(outcome.stdout, "cpu.temp\t42.0\t°C\n") || strings.Contains(outcome.stdout, "commander.") {
		test.Errorf("stdout %q", outcome.stdout)
	}
	if mode := controller.Mode(); mode != fakecommander.SoftwareMode {
		test.Errorf("the controller is in mode %v", mode)
	}
	// The writes are read after the run ended, since a run that holds a session never goes quiet. Its own teardown is
	// the one return to hardware mode, and sensordeck sensors added neither an entry nor a return.
	running.stop(syscall.SIGTERM)
	if entries, returns := softwareModeEntries(controller); entries != 1 || returns != 1 {
		test.Errorf("%d entries into software mode, %d returns to hardware mode", entries, returns)
	}
}

// writeHolder writes the fake proc root so that the node of the screen is locked by the process.
func writeHolder(test *testing.T, procRoot string, screen *fakescreen.Screen, running *process) string {
	test.Helper()
	var status unix.Stat_t
	if err := unix.Stat(screen.Node(), &status); err != nil {
		test.Fatal(err)
	}
	pid := running.command.Process.Pid
	line := fmt.Sprintf("1: FLOCK  ADVISORY  WRITE %d %02x:%02x:%d 0 EOF\n", pid, unix.Major(uint64(status.Dev)), unix.Minor(uint64(status.Dev)), status.Ino)
	if err := os.WriteFile(filepath.Join(procRoot, "locks"), []byte(line), 0o644); err != nil {
		test.Fatal(err)
	}
	directory := filepath.Join(procRoot, strconv.Itoa(pid))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	cmdline := strings.Join(running.command.Args, "\x00") + "\x00"
	if err := os.WriteFile(filepath.Join(directory, "cmdline"), []byte(cmdline), 0o644); err != nil {
		test.Fatal(err)
	}
	return fmt.Sprintf(" (process %d: %s)", pid, strings.Join(running.command.Args, " "))
}

func TestASecondRunRefusesWhileTheFirstHasTheScreen(test *testing.T) {
	fake, screen, environment, config := runningSetup(test)
	first := start(test, fake, environment, "run", "--config", config)
	if _, reached := screen.WaitReports(2, reportLimit); !reached {
		test.Fatal("the first run sent no report")
	}
	procRoot := strings.TrimPrefix(environment[0], "SENSORDECK_PROC_ROOT=")
	holder := writeHolder(test, procRoot, screen, first)
	outcome := runWith(test, fake, environment, "run", "--config", config)
	want := "sensordeck: the screen is in use by another sensordeck process" + holder + serviceStopText
	if outcome.status != 1 || outcome.stderr != want || strings.Contains(outcome.stdout, "running:") {
		test.Errorf("status %d, stdout %q, stderr %q, want %q", outcome.status, outcome.stdout, outcome.stderr, want)
	}
	before := len(screen.Reports())
	if _, reached := screen.WaitReports(before+1, reportLimit); !reached {
		test.Error("the first run sent no report after the refusal")
	}
	first.stop(syscall.SIGTERM)
}

func TestValuesRefusesWhileRunHasTheScreen(test *testing.T) {
	fake, screen, environment, config := runningSetup(test)
	first := start(test, fake, environment, "run", "--config", config)
	if _, reached := screen.WaitReports(2, reportLimit); !reached {
		test.Fatal("the run sent no report")
	}
	outcome := runWith(test, fake, environment, "values", "7=1234")
	if outcome.status != 1 || outcome.stdout != "" || outcome.stderr != "sensordeck: the screen is in use by another sensordeck process"+serviceStopText {
		test.Errorf("status %d, stdout %q, stderr %q", outcome.status, outcome.stdout, outcome.stderr)
	}
	if _, found := waitForReport(screen, 0, 300*time.Millisecond, func(pairs map[uint8]uint16) bool {
		_, has := pairs[7]
		return has
	}); found {
		test.Error("the screen received a report with the tag 7")
	}
	first.stop(syscall.SIGTERM)
}

// expectFreeDevices checks that values and sensors find the screen and the controller free.
func expectFreeDevices(test *testing.T, fake *fakescreen.Fake, environment []string, after syscall.Signal) {
	test.Helper()
	if outcome := runWith(test, fake, environment, "values", "7=1234"); outcome.status != 0 {
		test.Errorf("after %v, values: status %d, stderr %q", after, outcome.status, outcome.stderr)
	}
	outcome := runWith(test, fake, environment, "sensors")
	if !strings.Contains(outcome.stdout, "commander.1.temp") {
		test.Errorf("after %v, sensors: stdout %q, stderr %q", after, outcome.stdout, outcome.stderr)
	}
}

func TestTheLocksEndWithTheHolderOnSIGTERMAndOnSIGKILL(test *testing.T) {
	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		fake := fakescreen.New(test)
		fake.Add(3, "name", "")
		tree, _ := runningTree(test, fake)
		controller := fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)
		config := writeConfigFile(test, "interval: 100ms\ncommander_color: \"#ff0000\"\ntags:\n  cpu.temp: 1\n  commander.1.temp: 4\n")
		environment := environmentFor(test, tree, "")
		running := start(test, fake, environment, "run", "--config", config)
		waitForColourWrite(controller)
		if signal == syscall.SIGKILL {
			_ = running.command.Process.Kill()
			<-running.done
			// A session killed in the middle of an exchange leaves the LED endpoint of channel 0 open, which makes the
			// next reading fail whatever the locks do, as the driver does not close channel 0 after a failed reading
			// (unit 0005). The endpoint is closed here so that the test checks the lock and not that old limit.
			controller.SetOpenEndpoint(0, 0)
		} else {
			running.stop(signal)
		}
		expectFreeDevices(test, fake, environment, signal)
	}
}

func TestACommandOfACustomMetricDoesNotInheritTheLocks(test *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		test.Skip("no sleep")
	}
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	tree := fakesensors.New(test, fake.SysfsRoot)
	environment := environmentFor(test, tree, "")
	config := writeConfigFile(test, "interval: 100ms\nmetrics:\n  background: {command: \""+sleep+" 5 >/dev/null 2>&1 & echo value 1\"}\ntags:\n  background: 7\n")
	running := start(test, fake, environment, "run", "--config", config)
	if _, reached := waitForReport(screen, 0, reportLimit, func(pairs map[uint8]uint16) bool { _, has := pairs[7]; return has }); !reached {
		test.Fatal("the run sent no value of the custom metric")
	}
	running.stop(syscall.SIGTERM)
	if outcome := runWith(test, fake, environment, "values", "7=1234"); outcome.status != 0 {
		test.Errorf("values: status %d, stderr %q", outcome.status, outcome.stderr)
	}
}
