//go:build linux

package e2e

import (
	"bytes"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakecommander"
	"github.com/norareit/sensordeck/internal/fakescreen"
)

// startRunWithColour starts run with the tree of runningTree, a fake controller and a config that lights the LEDs red
// and maps cpu.temp and the first probe.
func startRunWithColour(test *testing.T, interval string) (*fakescreen.Screen, *fakecommander.Controller, *process) {
	test.Helper()
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	tree, _ := runningTree(test, fake)
	controller := fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)
	config := writeConfigFile(test, "interval: "+interval+"\ncommander_color: \"#ff0000\"\ntags:\n  cpu.temp: 1\n  commander.1.temp: 4\n")
	return screen, controller, start(test, fake, environmentFor(test, tree, ""), "run", "--config", config)
}

// encoded is a command as the driver writes it: the report ID byte, the command and zeros up to 97 bytes.
func encoded(command ...byte) []byte {
	written := make([]byte, 97)
	copy(written[1:], command)
	return written
}

func TestRunWithAColourSendsTheProbesInEveryRound(test *testing.T) {
	screen, controller, running := startRunWithColour(test, "100ms")
	first, reached := waitForReport(screen, 0, reportLimit, holdsPairs(map[uint8]uint16{1: 42, 4: 37}))
	if !reached {
		test.Fatalf("no value report holds (1, 42) and (4, 37): %v", screen.Reports())
	}
	controller.SetProbes(fakecommander.Probe{Tenths: 400}, fakecommander.Probe{Tenths: 227})
	if _, reached := waitForReport(screen, first, time.Second, holdsPairs(map[uint8]uint16{1: 42, 4: 40})); !reached {
		test.Errorf("no value report holds (1, 42) and (4, 40) within 1 second: %v", screen.Reports())
	}
	running.stop(syscall.SIGTERM)
}

func TestRunWithAColourReturnsTheControllerToHardwareModeOnSIGTERM(test *testing.T) {
	_, controller, running := startRunWithColour(test, "100ms")
	deadline := time.Now().Add(reportLimit)
	for len(controller.ColourWrites()) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	outcome, took := running.stop(syscall.SIGTERM)
	if outcome.status != 0 || took > 3*time.Second {
		test.Errorf("status %d after %v, stderr %q", outcome.status, took, outcome.stderr)
	}
	if controller.Mode() != fakecommander.HardwareMode || controller.OpenEndpoint(0) != 0 {
		test.Error("the controller is not in hardware mode with channel 0 closed")
	}
	enters := 0
	for _, written := range controller.Writes() {
		if bytes.Equal(written, encoded(0x08, 0x01, 0x03, 0x00, 0x02)) {
			enters++
		}
	}
	writes := controller.ColourWrites()
	if enters != 1 || len(writes) != 1 || !bytes.Equal(writes[0][9:9+48], bytes.Repeat([]byte{0xFF, 0x00, 0x00}, 16)) {
		test.Errorf("%d software modes, %d colour writes", enters, len(writes))
	}
	if lighting, _ := controller.Lighting(); lighting != fakecommander.StoredLighting {
		test.Errorf("lighting %d", lighting)
	}
}

func TestRunWithAColourKeepsTheControllerInSoftwareModeBetweenRounds(test *testing.T) {
	_, controller, running := startRunWithColour(test, "5s")
	controller.SetSoftwareModeTimeout(1500 * time.Millisecond)
	deadline := time.Now().Add(reportLimit)
	for len(controller.ColourWrites()) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if len(controller.ColourWrites()) != 1 {
		test.Fatal("no colour write arrived")
	}
	time.Sleep(4 * time.Second)
	lighting, colour := controller.Lighting()
	if controller.Mode() != fakecommander.SoftwareMode || lighting != fakecommander.SentColour || colour != [3]byte{0xFF, 0, 0} {
		test.Errorf("mode %d, lighting %d, colour %v", controller.Mode(), lighting, colour)
	}
	reads := 0
	for _, written := range controller.Writes() {
		if bytes.Equal(written, encoded(0x08, 0x08, 0x01, 0x21)) {
			reads++
		}
	}
	if reads < 3 {
		test.Errorf("%d reads of 0x21", reads)
	}
	running.stop(syscall.SIGTERM)
}

func TestRunWithAColourGoesOnWhenTheControllerIsRemoved(test *testing.T) {
	screen, controller, running := startRunWithColour(test, "100ms")
	if _, reached := waitForReport(screen, 0, reportLimit, holdsPairs(map[uint8]uint16{1: 42, 4: 37})); !reached {
		test.Fatal("no value report with the probe")
	}
	controller.RemoveNode()
	time.Sleep(200 * time.Millisecond)
	mark := len(screen.Reports())
	time.Sleep(2 * time.Second)
	reports := screen.Reports()
	if len(reports) <= mark {
		test.Fatal("no report arrived after the node was removed")
	}
	// The clock goes out once a second beside the values, and a clock report holds no value pairs.
	valueReports := 0
	for _, report := range reports[mark:] {
		pairs := valuePairs(report)
		if pairs == nil {
			continue
		}
		valueReports++
		if pairs[1] != 42 {
			test.Errorf("a report holds %v", pairs)
		}
	}
	if valueReports == 0 {
		test.Error("no value report arrived after the node was removed")
	}
	outcome, _ := running.stop(syscall.SIGTERM)
	if !strings.Contains(outcome.stderr, "controller lost: ") {
		test.Errorf("stderr %q", outcome.stderr)
	}
}

func TestRunWithAnUnquotedColourFailsBeforeTheScreen(test *testing.T) {
	fake := fakescreen.New(test)
	config := writeConfigFile(test, "commander_color: #ff0000\ntags:\n  cpu.temp: 1\n")
	outcome := runWith(test, fake, nil, "run", "--config", config)
	want := "sensordeck: config file " + config + `: commander_color: no colour given, write it in quotes such as "#00c8ff", because # starts a comment in YAML` + "\n"
	if outcome.status != 1 || outcome.stderr != want {
		test.Errorf("status %d, stderr %q", outcome.status, outcome.stderr)
	}
}
