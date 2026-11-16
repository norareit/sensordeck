//go:build linux

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/norareit/sensordeck/internal/fakeusb"
)

// timeoutError is what a write to a hung screen gives.
var timeoutError = &os.PathError{Op: "write", Path: "/dev/hidraw3", Err: syscall.ETIMEDOUT}

// hungHarness is a harness whose writes time out, with the power of a USB screen in a temporary sysfs tree.
func hungHarness(test *testing.T) (*harness, *fakeusb.Device) {
	test.Helper()
	root := test.TempDir()
	sysfsRoot, devRoot := filepath.Join(root, "sys"), filepath.Join(root, "dev")
	device := fakeusb.New(test, sysfsRoot, devRoot, 3)
	harness := newHarness()
	harness.transport.writeErr = timeoutError
	harness.power = linuxPower{sysfsRoot: sysfsRoot, devRoot: devRoot, node: func() string { return device.Node }}
	return harness, device
}

func TestClockOfAHungScreenTellsItWillRecover(test *testing.T) {
	harness, _ := hungHarness(test)
	if status := harness.run("clock"); status != 1 {
		test.Fatalf("status %d", status)
	}
	want := "sensordeck: write /dev/hidraw3: connection timed out: the screen has hung. " +
		"It recovers by itself once no program has held it open for a few seconds, so run the command again\n"
	if got := harness.stderr.String(); got != want {
		test.Errorf("stderr %q", got)
	}
	if len(harness.opened) != 1 || harness.transport.reopens != 0 {
		test.Errorf("opened %d times, reopened %d", len(harness.opened), harness.transport.reopens)
	}
}

func TestClockOfAHungScreenWithNoRuleTellsToInstallIt(test *testing.T) {
	harness, device := hungHarness(test)
	device.Set("control", "on\n")
	if status := harness.run("clock"); status != 1 {
		test.Fatalf("status %d", status)
	}
	got := harness.stderr.String()
	if !strings.Contains(got, "connection timed out: the screen has hung. Installing the current udev rule lets sensordeck recover it") {
		test.Errorf("stderr %q", got)
	}
	if len(harness.opened) != 1 || harness.transport.reopens != 0 {
		test.Errorf("opened %d times, reopened %d", len(harness.opened), harness.transport.reopens)
	}
}

func TestAnotherFailureOfAWriteIsGivenAsItIs(test *testing.T) {
	harness, _ := hungHarness(test)
	harness.transport.writeErr = errors.New("device gone")
	if status := harness.run("clock"); status != 1 {
		test.Fatalf("status %d", status)
	}
	if got := harness.stderr.String(); got != "sensordeck: device gone\n" {
		test.Errorf("stderr %q", got)
	}
}
