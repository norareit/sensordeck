package cli

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

var errTimeout = errors.New("fake timeout")

// fakeRestarter is a restarter whose outcomes the test sets, one for each restart and the last repeated. A restart
// that is possible closes the transport and takes the time of duration, and one that opens the screen again reopens it.
type fakeRestarter struct {
	timedOutBy func(err error) bool
	outcomes   []restartResult
	duration   time.Duration
	clock      *fakeClock
	transport  *fakeTransport
	began      []time.Time // when each restart began
	onRestart  func(count int)
}

func newFakeRestarter(harness *harness, outcomes ...restartResult) *fakeRestarter {
	restarter := &fakeRestarter{
		timedOutBy: func(err error) bool { return errors.Is(err, errTimeout) },
		outcomes:   outcomes,
		clock:      harness.clock,
		transport:  harness.transport,
	}
	harness.restarter = restarter
	return restarter
}

func (restarter *fakeRestarter) timedOut(err error) bool { return restarter.timedOutBy(err) }

func (restarter *fakeRestarter) restart(stop context.Context) restartResult {
	restarter.began = append(restarter.began, restarter.clock.now())
	count := len(restarter.began)
	result := restarter.outcomes[min(count, len(restarter.outcomes))-1]
	if restarter.onRestart != nil {
		restarter.onRestart(count)
	}
	if result.outcome == restartNotPossible {
		return result
	}
	_ = restarter.transport.Close()
	restarter.clock.sleep(restarter.duration)
	if result.outcome == restartedOpen {
		_ = restarter.transport.Reopen()
	}
	return result
}

func TestRestartAfterHangOfATimeoutThatRestartedTheScreen(test *testing.T) {
	for _, outcome := range []restartOutcome{restartedOpen, restartedNotOpen} {
		harness := newHarness()
		restarter := newFakeRestarter(harness, restartResult{outcome: outcome, index: 33})
		err := restartAfterHang(fmt.Errorf("write /dev/hidraw3: %w", errTimeout), restarter)
		want := "write /dev/hidraw3: fake timeout: the screen has hung. sensordeck restarted it, so run the command again"
		if err.Error() != want {
			test.Errorf("outcome %d: got %q", outcome, err)
		}
		if !errors.Is(err, errTimeout) {
			test.Error("the error of the write is not reachable")
		}
		if len(restarter.began) != 1 {
			test.Errorf("%d restarts", len(restarter.began))
		}
	}
}

func TestRestartAfterHangOfATimeoutThatDidNotRestartTheScreen(test *testing.T) {
	harness := newHarness()
	restarter := newFakeRestarter(harness, restartResult{outcome: notRestarted})
	err := restartAfterHang(errTimeout, restarter)
	want := "fake timeout: the screen has hung and did not restart. Unplugging the screen once recovers it"
	if err.Error() != want || !errors.Is(err, errTimeout) {
		test.Errorf("got %q", err)
	}
}

func TestRestartAfterHangOfATimeoutThatCannotBeRestarted(test *testing.T) {
	harness := newHarness()
	restarter := newFakeRestarter(harness, restartResult{outcome: restartNotPossible, err: errors.New("permission denied")})
	err := restartAfterHang(errTimeout, restarter)
	want := `fake timeout: the screen has hung and sensordeck cannot restart it: permission denied. Installing the current ` +
		`udev rule lets sensordeck recover it, see "Linux: access to the screen" in ` +
		"https://github.com/norareit/sensordeck#linux-access-to-the-screen. Unplugging the screen once recovers it now"
	if err.Error() != want || !errors.Is(err, errTimeout) {
		test.Errorf("got %q", err)
	}
}

func TestRestartAfterHangGivesAnotherErrorBackAsItIsWithNoRestart(test *testing.T) {
	harness := newHarness()
	restarter := newFakeRestarter(harness, restartResult{outcome: restartedOpen})
	if err := restartAfterHang(errFake, restarter); err != errFake {
		test.Errorf("got %v", err)
	}
	if err := restartAfterHang(nil, restarter); err != nil {
		test.Errorf("got %v", err)
	}
	if err := restartAfterHang(errTimeout, noScreenRestarter{}); err != errTimeout {
		test.Errorf("got %v", err)
	}
	if len(restarter.began) != 0 {
		test.Errorf("%d restarts", len(restarter.began))
	}
}
