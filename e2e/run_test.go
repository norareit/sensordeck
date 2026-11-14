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
	tree := fakesensors.New(test, fake.SysfsRoot)
	tree.Hwmon(1, "k10temp", "pci0000:00/0000:00:18.3", fakesensors.Sensor{Number: 1, Label: "Tctl", Input: "42000"})
	tree.CPUFrequency(0, "1234000\n")
	return fake, screen, environmentFor(test, tree, ""), writeConfigFile(test, runConfig)
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
