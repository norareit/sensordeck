package cli

import (
	"errors"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// fakeClock is the time of the tests. It moves only when something waits.
type fakeClock struct{ current time.Time }

func (clock *fakeClock) now() time.Time { return clock.current }

func (clock *fakeClock) sleep(duration time.Duration) { clock.current = clock.current.Add(duration) }

type readResult struct {
	report smartmonitor.Report
	err    error
}

// fakeTransport records every call. A read takes the next queued result. With the queue empty it answers ACK when
// ackWhenEmpty is set, and otherwise it moves the clock forward by its time limit and gives the passed time limit.
type fakeTransport struct {
	clock         *fakeClock
	queue         []readResult
	ackWhenEmpty  bool
	writes        []smartmonitor.Report
	writeErr      error
	failWrites    []bool  // one entry for each write from the next on: true fails it with errFake
	reopenResults []error // one entry for each reopening from the next on, then reopenErr
	reopens       int
	reopenErr     error
	closed        int
	readLimits    []time.Duration
}

func (fake *fakeTransport) Write(report smartmonitor.Report) error {
	if len(fake.failWrites) > 0 {
		failing := fake.failWrites[0]
		fake.failWrites = fake.failWrites[1:]
		if failing {
			return errFake
		}
	}
	if fake.writeErr != nil {
		return fake.writeErr
	}
	fake.writes = append(fake.writes, report)
	return nil
}

func (fake *fakeTransport) Read(timeout time.Duration) (smartmonitor.Report, error) {
	fake.readLimits = append(fake.readLimits, timeout)
	if len(fake.queue) > 0 {
		result := fake.queue[0]
		fake.queue = fake.queue[1:]
		return result.report, result.err
	}
	if fake.ackWhenEmpty {
		return reportOf(0x06), nil
	}
	fake.clock.sleep(timeout)
	return smartmonitor.Report{}, smartmonitor.ErrReadTimeout
}

func (fake *fakeTransport) Reopen() error {
	fake.reopens++
	if len(fake.reopenResults) > 0 {
		result := fake.reopenResults[0]
		fake.reopenResults = fake.reopenResults[1:]
		return result
	}
	return fake.reopenErr
}

func (fake *fakeTransport) Close() error {
	fake.closed++
	return nil
}

var errFake = errors.New("fake failure")
