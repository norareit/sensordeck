//go:build linux

package e2e

import (
	"bytes"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakecommander"
	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/fakesensors"
)

// controllerCommands are the eight commands of a reading, after the report ID byte.
var controllerCommands = [8][]byte{
	{0x08, 0x01, 0x03, 0x00, 0x02}, {0x08, 0x0D, 0x00, 0x17}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01},
	{0x08, 0x0D, 0x00, 0x21}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01}, {0x08, 0x01, 0x03, 0x00, 0x01},
}

// controllerWrite is a command as the driver writes it: 97 bytes, the report ID byte first.
func controllerWrite(step int) []byte {
	written := make([]byte, 97)
	copy(written[1:], controllerCommands[step-1])
	return written
}

func expectWrites(test *testing.T, got [][]byte, steps ...int) {
	test.Helper()
	if len(got) != len(steps) {
		test.Fatalf("%d writes, want %d", len(got), len(steps))
	}
	for position, step := range steps {
		if !bytes.Equal(got[position], controllerWrite(step)) {
			test.Errorf("write %d is not step %d: % x", position+1, step, got[position][:8])
		}
	}
}

func TestSensorsListsTheProbesOfTheController(test *testing.T) {
	fake := fakescreen.New(test)
	environment := fullTree(test, fake, "echo '44, 17, 3256, 32607'\n")
	controller := fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)
	want := "cpu.temp\t51.4\t°C\n" +
		"cpu.ccd1.temp\t40.0\t°C\n" +
		"cpu.ccd2.temp\t38.1\t°C\n" +
		"cpu.clock\t4425.4\tMHz\n" +
		"gpu.temp\t44.0\t°C\n" +
		"gpu.util\t17.0\t%\n" +
		"gpu.mem.used\t3256.0\tMiB\n" +
		"gpu.mem.total\t32607.0\tMiB\n" +
		"ram.used\t21.8\tGiB\n" +
		"ram.total\t91.7\tGiB\n" +
		"ram.util\t23.8\t%\n" +
		"ram.1.temp\t40.0\t°C\n" +
		"ram.2.temp\t37.9\t°C\n" +
		"nvme.1.temp\t33.9\t°C\n" +
		"nvme.2.temp\t38.8\t°C\n" +
		"net.up\t0.0\tKiB/s\n" +
		"net.down\t0.0\tKiB/s\n" +
		"commander.1.temp\t37.3\t°C\n" +
		"commander.2.temp\t22.7\t°C\n"
	runWith(test, fake, environment, "sensors").expect(test, 0, want, "")
	expectWrites(test, controller.Writes(), 1, 2, 3, 4, 5, 6, 7, 8)
	if got := controller.WritesOnInterfaceOne(); len(got) != 0 {
		test.Errorf("interface 1 got %d writes", len(got))
	}
	if controller.Mode() != fakecommander.HardwareMode {
		test.Error("the controller is not in hardware mode")
	}
}

func TestRunSendsTheProbesOfTheController(test *testing.T) {
	fake, screen, environment, _ := runningSetup(test)
	controller := fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)
	config := writeConfigFile(test, "interval: 100ms\ntags:\n  commander.1.temp: 4\n  commander.2.temp: 5\n")
	running := start(test, fake, environment, "run", "--config", config)
	want := append([]byte{0x02, 0x02, 0x04, 0x00, 0x25, 0x05, 0x00, 0x17}, make([]byte, 56)...)
	found := false
	deadline := time.Now().Add(reportLimit)
	for !found && time.Now().Before(deadline) {
		reports, _ := screen.WaitReports(2, reportLimit)
		for _, report := range reports {
			found = found || bytes.Equal(report[:], want)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !found {
		test.Error("no value report holds the pairs (4, 37) and (5, 23)")
	}
	outcome, _ := running.stop(syscall.SIGTERM)
	if outcome.status != 0 {
		test.Errorf("status %d, stderr %q", outcome.status, outcome.stderr)
	}
	if controller.Mode() != fakecommander.HardwareMode {
		test.Error("the controller is not in hardware mode")
	}
}

func TestSensorsInterruptedDuringTheReadingRestoresHardwareMode(test *testing.T) {
	fake := fakescreen.New(test)
	controller := fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)
	controller.Fail(6, fakecommander.NoAnswer)
	environment := []string{"SENSORDECK_PROC_ROOT=" + filepath.Join(test.TempDir(), "proc"), "PATH=" + test.TempDir()}
	running := start(test, fake, environment, "sensors")
	if _, reached := controller.WaitWrites(6, reportLimit); !reached {
		test.Fatal("step 6 did not arrive")
	}
	outcome, took := running.stop(syscall.SIGINT)
	outcome.expect(test, 1, "", "sensordeck: interrupted\n")
	if took > 3*time.Second {
		test.Errorf("the process ended %v after the signal", took)
	}
	expectWrites(test, controller.Writes(), 1, 2, 3, 4, 5, 6, 8)
	if controller.Mode() != fakecommander.HardwareMode {
		test.Error("the controller is not in hardware mode")
	}
}

// startRunWithController starts run with the tree of runningTree, a fake controller and a config that reads the
// controller at the given period. It gives what the tests of the period need.
func startRunWithController(test *testing.T, period string) (*fakescreen.Screen, *fakecommander.Controller, *fakesensors.Tree, string, *process) {
	test.Helper()
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	tree, entry := runningTree(test, fake)
	controller := fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)
	config := writeConfigFile(test, "interval: 100ms\ncommander_interval: "+period+"\ntags:\n  cpu.temp: 1\n  commander.1.temp: 4\n")
	return screen, controller, tree, entry, start(test, fake, environmentFor(test, tree, ""), "run", "--config", config)
}

func holdsPairs(want map[uint8]uint16) func(map[uint8]uint16) bool {
	return func(pairs map[uint8]uint16) bool {
		for tag, value := range want {
			if got, found := pairs[tag]; !found || got != value {
				return false
			}
		}
		return len(pairs) == len(want)
	}
}

func TestRunReadsTheControllerAgainAfterItsPeriod(test *testing.T) {
	screen, controller, _, _, running := startRunWithController(test, "100ms")
	first, reached := waitForReport(screen, 0, reportLimit, holdsPairs(map[uint8]uint16{1: 42, 4: 37}))
	if !reached {
		test.Fatalf("no value report holds (1, 42) and (4, 37): %v", screen.Reports())
	}
	controller.SetProbes(fakecommander.Probe{Tenths: 400}, fakecommander.Probe{Tenths: 227})
	if _, reached := waitForReport(screen, first, reportLimit, holdsPairs(map[uint8]uint16{1: 42, 4: 40})); !reached {
		test.Errorf("no later value report holds (1, 42) and (4, 40): %v", screen.Reports())
	}
	outcome, _ := running.stop(syscall.SIGTERM)
	if outcome.status != 0 {
		test.Errorf("status %d, stderr %q", outcome.status, outcome.stderr)
	}
	starts := 0
	for _, written := range controller.Writes() {
		if bytes.Equal(written, controllerWrite(1)) {
			starts++
		}
	}
	if starts < 2 {
		test.Errorf("the controller got step 1 %d times", starts)
	}
	if controller.Mode() != fakecommander.HardwareMode {
		test.Error("the controller is not in hardware mode")
	}
}

func TestRunSendsTheKeptProbesBetweenReadingsOfTheController(test *testing.T) {
	screen, controller, tree, entry, running := startRunWithController(test, "1h")
	first, reached := waitForReport(screen, 0, reportLimit, holdsPairs(map[uint8]uint16{1: 42, 4: 37}))
	if !reached {
		test.Fatalf("no value report holds (1, 42) and (4, 37): %v", screen.Reports())
	}
	controller.SetProbes(fakecommander.Probe{Tenths: 400}, fakecommander.Probe{Tenths: 227})
	tree.ReplaceInput(entry, 1, "43000")
	changed, reached := waitForReport(screen, first, reportLimit, holdsPairs(map[uint8]uint16{1: 43, 4: 37}))
	if !reached {
		test.Fatalf("no later value report holds (1, 43) and (4, 37): %v", screen.Reports())
	}
	// A clock report may stand between value reports, once a second, so only the value reports are counted.
	var following []map[uint8]uint16
	deadline := time.Now().Add(reportLimit)
	for len(following) < 3 && time.Now().Before(deadline) {
		following = following[:0]
		for _, report := range screen.Reports()[changed+1:] {
			if pairs := valuePairs(report); pairs != nil {
				following = append(following, pairs)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(following) < 3 {
		test.Fatalf("fewer than three value reports followed: %v", screen.Reports())
	}
	for _, pairs := range following[:3] {
		if pairs[4] != 37 {
			test.Errorf("a report after the change holds %v", pairs)
		}
	}
	outcome, _ := running.stop(syscall.SIGTERM)
	if outcome.status != 0 {
		test.Errorf("status %d, stderr %q", outcome.status, outcome.stderr)
	}
	expectWrites(test, controller.Writes(), 1, 2, 3, 4, 5, 6, 7, 8)
	if controller.Mode() != fakecommander.HardwareMode {
		test.Error("the controller is not in hardware mode")
	}
}
