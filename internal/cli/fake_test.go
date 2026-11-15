package cli

import (
	"errors"
	"sync"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// fakeClock is the time of the tests. It moves only when something waits. The keeper of the session of the controller
// arms one timer at a time, and the clock fires it at its moment when a sleep passes it, then waits until the keeper
// has armed the next one, so that the keeper works at the moments it asked for.
type fakeClock struct {
	mutex   sync.Mutex
	current time.Time
	timer   *fakeTimer
	armed   chan struct{}   // gets a signal when the keeper arms a timer, if the test made it
	release <-chan struct{} // ends the wait for the keeper when the run is stopped
	keeping bool            // the first sleep waits for the first timer, which the keeper may not have armed yet
}

type fakeTimer struct {
	due     time.Time
	channel chan time.Time
}

func (clock *fakeClock) now() time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	return clock.current
}

// after arms a timer due after the duration, which replaces the one that is pending.
func (clock *fakeClock) after(duration time.Duration) <-chan time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	timer := &fakeTimer{due: clock.current.Add(duration), channel: make(chan time.Time, 1)}
	clock.timer = timer
	select {
	case clock.armed <- struct{}{}:
	default:
	}
	return timer.channel
}

func (clock *fakeClock) sleep(duration time.Duration) {
	if clock.keeping {
		clock.keeping = false
		select {
		case <-clock.armed:
		case <-clock.release:
		}
	}
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	select {
	case <-clock.armed:
	default:
	}
	target := clock.current.Add(duration)
	for clock.timer != nil && !clock.timer.due.After(target) {
		timer := clock.timer
		clock.timer = nil
		if timer.due.After(clock.current) {
			clock.current = timer.due
		}
		timer.channel <- clock.current
		clock.mutex.Unlock()
		select {
		case <-clock.armed:
		case <-clock.release:
		}
		clock.mutex.Lock()
	}
	clock.current = target
}

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
	serial        string
	afterWrite    func(count int) // called after each write that succeeded, with the count of writes so far
}

func (fake *fakeTransport) Serial() string { return fake.serial }

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
	if fake.afterWrite != nil {
		fake.afterWrite(len(fake.writes))
	}
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
