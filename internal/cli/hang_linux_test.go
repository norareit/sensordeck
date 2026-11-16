//go:build linux

package cli

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/norareit/sensordeck/internal/hidraw"
)

// timeoutError is what a write to a hung screen gives.
var timeoutError = &os.PathError{Op: "write", Path: "/dev/hidraw3", Err: syscall.ETIMEDOUT}

// hungHarness is a harness whose writes time out, with a restarter that tells a timeout as the Linux one does.
func hungHarness(test *testing.T, outcomes ...restartResult) (*harness, *fakeRestarter) {
	test.Helper()
	harness := runWith(twoMetrics(), 1<<30)
	harness.transport.failErr = timeoutError
	harness.transport.writeErr = timeoutError
	restarter := newFakeRestarter(harness, outcomes...)
	restarter.timedOutBy = hidraw.TimedOut
	return harness, restarter
}

func TestClockOfAHungScreenRestartsItAndSaysSo(test *testing.T) {
	cases := []struct {
		name    string
		outcome restartResult
		want    string
	}{
		{"left the bus", restartResult{outcome: restartedOpen, index: 33},
			"sensordeck: write /dev/hidraw3: connection timed out: the screen has hung. sensordeck restarted it, so run the command again\n"},
		{"did not restart", restartResult{outcome: notRestarted},
			"sensordeck: write /dev/hidraw3: connection timed out: the screen has hung and did not restart. Unplugging the screen once recovers it\n"},
		{"not possible", restartResult{outcome: restartNotPossible, err: errors.New("open /dev/bus/usb/001/005: permission denied")},
			"sensordeck: write /dev/hidraw3: connection timed out: the screen has hung and sensordeck cannot restart it: " +
				"open /dev/bus/usb/001/005: permission denied. Installing the current udev rule lets sensordeck recover it, see " +
				`"Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen. ` +
				"Unplugging the screen once recovers it now\n"},
	}
	for _, testCase := range cases {
		test.Run(testCase.name, func(test *testing.T) {
			harness, restarter := hungHarness(test, testCase.outcome)
			if status := harness.run("clock"); status != 1 {
				test.Fatalf("status %d", status)
			}
			if got := harness.stderr.String(); got != testCase.want {
				test.Errorf("stderr %q", got)
			}
			if len(restarter.began) != 1 {
				test.Errorf("%d restarts", len(restarter.began))
			}
			if len(harness.opened) != 1 {
				test.Errorf("opened %d times", len(harness.opened))
			}
		})
	}
}

func TestClockOfAHungScreenWritesOnlyOnce(test *testing.T) {
	harness, _ := hungHarness(test, restartResult{outcome: restartedOpen, index: 33})
	harness.transport.writeErr = nil
	harness.transport.failWrites = []bool{true}
	harness.run("clock")
	if len(harness.transport.writes) != 0 || len(harness.transport.failWrites) != 0 {
		test.Errorf("writes %d, unused failures %d", len(harness.transport.writes), len(harness.transport.failWrites))
	}
}

func TestAnotherFailureOfAWriteIsGivenAsItIs(test *testing.T) {
	harness, restarter := hungHarness(test, restartResult{outcome: restartedOpen})
	harness.transport.writeErr = errors.New("device gone")
	if status := harness.run("clock"); status != 1 {
		test.Fatalf("status %d", status)
	}
	if got := harness.stderr.String(); got != "sensordeck: device gone\n" || strings.Contains(got, "hung") {
		test.Errorf("stderr %q", got)
	}
	if len(restarter.began) != 0 {
		test.Errorf("%d restarts", len(restarter.began))
	}
}
