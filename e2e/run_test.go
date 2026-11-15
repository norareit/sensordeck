//go:build linux

package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/fakesensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const (
	runConfig   = "interval: 100ms\ntags:\n  cpu.temp: 1\n  cpu.clock: 2\n  gpu.temp: 3\n"
	reportLimit = 3 * time.Second
)

// valueReportOf42And1234 is the report of the first round: the tag 3 is left out because there is no nvidia-smi.
var valueReportOf42And1234 = append([]byte{0x02, 0x02, 0x01, 0x00, 0x2A, 0x02, 0x04, 0xD2}, make([]byte, 56)...)

func writeConfigFile(test *testing.T, content string) string {
	test.Helper()
	path := filepath.Join(test.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

// runningSetup gives a fake screen as hidraw3, the tree of the spec, the config file and the environment, with no
// nvidia-smi on the PATH.
func runningSetup(test *testing.T) (*fakescreen.Fake, *fakescreen.Screen, []string, string) {
	test.Helper()
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	tree, _ := runningTree(test, fake)
	return fake, screen, environmentFor(test, tree, ""), writeConfigFile(test, runConfig)
}

// runningTree makes the tree of the spec of unit 0004 for the fake screen, and gives it with the path of the entry of
// k10temp.
func runningTree(test *testing.T, fake *fakescreen.Fake) (*fakesensors.Tree, string) {
	test.Helper()
	tree := fakesensors.New(test, fake.SysfsRoot)
	entry := tree.Hwmon(1, "k10temp", "pci0000:00/0000:00:18.3", fakesensors.Sensor{Number: 1, Label: "Tctl", Input: "42000"})
	tree.CPUFrequency(0, "1234000\n")
	return tree, entry
}

func TestRunSendsTheClockAndThenTheValues(test *testing.T) {
	fake, screen, environment, config := runningSetup(test)
	before := time.Now().Truncate(time.Second)
	running := start(test, fake, environment, "run", "--config", config)
	reports, reached := screen.WaitReports(2, reportLimit)
	after := time.Now()
	if !reached {
		test.Fatalf("%d reports arrived", len(reports))
	}
	expectClockReport(test, reports[0][:], before, after)
	if !bytes.Equal(reports[1][:], valueReportOf42And1234) {
		test.Errorf("second report % X", reports[1])
	}
	running.stop(syscall.SIGTERM)
}

// valuePairs decodes a value report into its tags and values, and gives nil for a report of another kind.
func valuePairs(report smartmonitor.Report) map[uint8]uint16 {
	if report[0] != 0x02 {
		return nil
	}
	pairs := map[uint8]uint16{}
	for position := range int(report[1]) {
		offset := 2 + position*3
		pairs[report[offset]] = uint16(report[offset+1])<<8 | uint16(report[offset+2])
	}
	return pairs
}

// waitForReport polls the reports from the given position on until one satisfies the test, and gives its position.
func waitForReport(screen *fakescreen.Screen, from int, limit time.Duration, holds func(map[uint8]uint16) bool) (int, bool) {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		reports := screen.Reports()
		for position := from; position < len(reports); position++ {
			if holds(valuePairs(reports[position])) {
				return position, true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return 0, false
}

func TestRunSendsTheShareOfRAMAndTheNetworkRates(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	tree := fakesensors.New(test, fake.SysfsRoot)
	tree.Meminfo("MemTotal:       96187828 kB\nMemFree:         1000 kB\nMemAvailable:   73300372 kB\n")
	addNetwork(tree)
	config := writeConfigFile(test, "interval: 100ms\ntags:\n  ram.util: 12\n  net.up: 18\n  net.down: 19\n")
	running := start(test, fake, environmentFor(test, tree, ""), "run", "--config", config)

	first, reached := waitForReport(screen, 0, reportLimit, func(pairs map[uint8]uint16) bool {
		value12, have12 := pairs[12]
		value18, have18 := pairs[18]
		value19, have19 := pairs[19]
		return have12 && have18 && have19 && value12 == 24 && value18 == 0 && value19 == 0
	})
	if !reached {
		test.Fatalf("no report held (12, 24), (18, 0) and (19, 0): %v", screen.Reports())
	}
	// 1048576 bytes more received, through a new file renamed into place, which gives at least 1 KiB/s for any time
	// between two rounds below about 34 minutes
	tree.NetDev(netDevHeader + netDevLine("lo", 5000, 5000) + netDevLine("enp6s0", 123456+1048576, 654321) +
		netDevLine("docker0", 777, 888))
	if _, reached := waitForReport(screen, first+1, reportLimit, func(pairs map[uint8]uint16) bool {
		return pairs[19] >= 1
	}); !reached {
		test.Errorf("no later report held a download rate of 1 or more: %v", screen.Reports())
	}
	outcome, _ := running.stop(syscall.SIGTERM)
	if outcome.status != 0 {
		test.Errorf("status %d", outcome.status)
	}
}

func TestRunEndsOnSIGTERM(test *testing.T) {
	fake, screen, environment, config := runningSetup(test)
	running := start(test, fake, environment, "run", "--config", config)
	if _, reached := screen.WaitReports(4, reportLimit); !reached {
		test.Fatal("fewer than 4 reports arrived")
	}
	outcome, took := running.stop(syscall.SIGTERM)
	outcome.expect(test, 0, "running: 3 metrics every 100ms\nstopped\n", "")
	if took > 3*time.Second {
		test.Errorf("the process ended %v after the signal", took)
	}
}

func TestRunFindsTheScreenAgainUnderAnotherNumber(test *testing.T) {
	fake, screen, environment, config := runningSetup(test)
	screen.MoveAfter(2, 7)
	running := start(test, fake, environment, "run", "--config", config)
	reports, reached := screen.WaitReports(4, reportLimit)
	if !reached {
		test.Fatalf("%d reports arrived", len(reports))
	}
	if reports[2][0] != 0x03 || reports[3][0] != 0x02 {
		test.Errorf("after the move: % X and % X", reports[2][:4], reports[3][:4])
	}
	outcome, _ := running.stop(syscall.SIGTERM)
	if outcome.status != 0 {
		test.Errorf("status %d", outcome.status)
	}
	var lost bool
	for _, line := range strings.Split(outcome.stderr, "\n") {
		lost = lost || strings.HasPrefix(line, "screen lost: ")
	}
	if !lost || !strings.Contains("\n"+outcome.stderr, "\nscreen back\n") {
		test.Errorf("stderr %q", outcome.stderr)
	}
}

func TestRunWithATagOutOfRangeFailsBeforeOpeningTheScreen(test *testing.T) {
	config := writeConfigFile(test, "tags:\n  cpu.temp: 256\n")
	want := "sensordeck: config file " + config + ": tags: cpu.temp: the tag 256 is outside 0 to 255\n"
	run(test, fakescreen.New(test), "run", "--config", config).expect(test, 1, "", want)
}

func TestRunWithNoConfigFileNamesTheDefaultPlace(test *testing.T) {
	directory := test.TempDir()
	want := "sensordeck: cannot read the config file " + directory + "/sensordeck/config.yaml: no such file or directory\n"
	runWith(test, fakescreen.New(test), []string{"XDG_CONFIG_HOME=" + directory}, "run").expect(test, 1, "", want)
}
