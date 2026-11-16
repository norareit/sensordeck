package cli

import (
	"errors"
	"fmt"
	"testing"
)

var errTimeout = errors.New("fake timeout")

// timeoutPower says that errTimeout is a timeout and whether a recovery is possible.
type timeoutPower struct{ possible bool }

func (timeoutPower) timedOut(err error) bool { return errors.Is(err, errTimeout) }
func (power timeoutPower) recovery() (hangRecovery, bool) {
	return hangRecovery{suspended: func() bool { return true }}, power.possible
}

func TestHangErrorOfATimeoutWithARecoveryPossible(test *testing.T) {
	err := hangError(fmt.Errorf("write /dev/hidraw3: %w", errTimeout), timeoutPower{possible: true})
	want := "write /dev/hidraw3: fake timeout: the screen has hung. " +
		"It recovers by itself once no program has held it open for a few seconds, so run the command again"
	if err.Error() != want {
		test.Errorf("got %q", err)
	}
	if !errors.Is(err, errTimeout) {
		test.Error("the error of the write is not reachable")
	}
}

func TestHangErrorOfATimeoutWithNoRecoveryPossible(test *testing.T) {
	err := hangError(errTimeout, timeoutPower{})
	want := `fake timeout: the screen has hung. Installing the current udev rule lets sensordeck recover it, see ` +
		`"Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen. ` +
		"Unplugging the screen once recovers it now"
	if err.Error() != want {
		test.Errorf("got %q", err)
	}
	if !errors.Is(err, errTimeout) {
		test.Error("the error of the write is not reachable")
	}
}

func TestHangErrorGivesAnotherErrorBackAsItIs(test *testing.T) {
	if err := hangError(errFake, timeoutPower{possible: true}); err != errFake {
		test.Errorf("got %v", err)
	}
	if err := hangError(errTimeout, noScreenPower{}); err != errTimeout {
		test.Errorf("got %v", err)
	}
	if err := hangError(nil, timeoutPower{}); err != nil {
		test.Errorf("got %v", err)
	}
}
