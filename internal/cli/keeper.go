package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// errNoController is the setup that found no node of the controller, which writes no line.
var errNoController = errors.New("no controller found")

// heldSession is a held session of the controller. After an error of read it has ended with its teardown.
type heldSession interface {
	read(stop context.Context) ([]sensors.Reading, error)
	end()
}

const (
	// keepalivePeriod is the longest time between two readings of a held session, so that the controller stays in
	// software mode. It is the rate of the trace that showed software mode held, as docs/commander-core-xt.md says.
	keepalivePeriod = time.Second
	firstRetry      = 10 * time.Second
	longestRetry    = 5 * time.Minute
)

// sessionKeeper holds the session of the controller in a goroutine of its own, beside the rounds of run, so that the
// keepalive does not wait for a round, which can wait for a source, for the screen or for its next start.
type sessionKeeper struct {
	start        func(stop context.Context) (heldSession, error) // a setup; on an error it has ended
	probesMapped func() bool                                     // asked from readings, so by the goroutine that replaces the mappings
	now          func() time.Time
	after        func(duration time.Duration) <-chan time.Time
	stderr       io.Writer
	requests     chan chan []sensors.Reading
	done         chan struct{}

	// The fields below belong to the goroutine of keep.
	held     heldSession // nil while no session is held
	next     time.Time   // the next keepalive while held, the next setup otherwise
	retry    time.Duration
	lost     bool // "controller lost" was written and "controller back" is owed
	reported bool // "controller setup failed" was written since the last setup that succeeded
}

func newSessionKeeper(
	start func(stop context.Context) (heldSession, error), probesMapped func() bool, now func() time.Time,
	after func(duration time.Duration) <-chan time.Time, stderr io.Writer,
) *sessionKeeper {
	return &sessionKeeper{
		start: start, probesMapped: probesMapped, now: now, after: after, stderr: stderr,
		requests: make(chan chan []sensors.Reading), done: make(chan struct{}), retry: firstRetry,
	}
}

// keep holds the session until stop is done, and ends a held session with its teardown. It closes done.
func (keeper *sessionKeeper) keep(stop context.Context) {
	defer close(keeper.done)
	keeper.setUp(stop)
	// The timer is armed before a reply is sent, so that the round that asked cannot move on before the keeper has
	// the timer that the reading made due.
	timer := keeper.arm()
	for {
		select {
		case <-stop.Done():
			if keeper.held != nil {
				keeper.held.end()
			}
			return
		case <-timer:
			if keeper.held == nil {
				keeper.setUp(stop)
			} else {
				keeper.read(stop)
			}
			timer = keeper.arm()
		case reply := <-keeper.requests:
			readings := keeper.readIfHeld(stop)
			timer = keeper.arm()
			reply <- readings
		}
	}
}

// arm gives a timer for the next keepalive or setup.
func (keeper *sessionKeeper) arm() <-chan time.Time {
	return keeper.after(max(0, keeper.next.Sub(keeper.now())))
}

// readings asks for a reading and gives its wanted probes, or none when no session is held or the reading failed. The
// reading starts after the call, so after the round that asks, which is why the start of the round is not needed. The
// parameter is there because periodicSource takes it.
func (keeper *sessionKeeper) readings(stop context.Context, _ time.Time, wanted func(name string) bool) []sensors.Reading {
	if !keeper.probesMapped() {
		return nil
	}
	reply := make(chan []sensors.Reading, 1)
	select {
	case keeper.requests <- reply:
	case <-keeper.done:
		return nil
	case <-stop.Done():
		return nil
	}
	var wantedReadings []sensors.Reading
	for _, reading := range <-reply {
		if wanted(reading.Name) {
			wantedReadings = append(wantedReadings, reading)
		}
	}
	return wantedReadings
}

func (keeper *sessionKeeper) readIfHeld(stop context.Context) []sensors.Reading {
	if keeper.held == nil {
		return nil
	}
	return keeper.read(stop)
}

// read makes a reading of the held session, and when it fails, the session is gone and the next setup is due.
func (keeper *sessionKeeper) read(stop context.Context) []sensors.Reading {
	start := keeper.now()
	readings, err := keeper.held.read(stop)
	if err != nil {
		keeper.held = nil
		if stop.Err() == nil {
			keeper.lost = true
			keeper.say("controller lost: " + err.Error())
		}
		keeper.scheduleRetry()
		return nil
	}
	keeper.next = start.Add(keepalivePeriod)
	return readings
}

// setUp makes a setup and holds the session it starts.
func (keeper *sessionKeeper) setUp(stop context.Context) {
	session, err := keeper.start(stop)
	if err != nil {
		if stop.Err() == nil && !errors.Is(err, errNoController) && !keeper.reported {
			keeper.reported = true
			keeper.say("controller setup failed: " + err.Error())
		}
		keeper.scheduleRetry()
		return
	}
	keeper.held = session
	keeper.next = keeper.now().Add(keepalivePeriod)
	keeper.retry = firstRetry
	keeper.reported = false
	if keeper.lost {
		keeper.lost = false
		keeper.say("controller back")
	}
}

// scheduleRetry makes the next setup due after the wait, which doubles for each failure up to the longest.
func (keeper *sessionKeeper) scheduleRetry() {
	keeper.next = keeper.now().Add(keeper.retry)
	keeper.retry = min(2*keeper.retry, longestRetry)
}

func (keeper *sessionKeeper) say(line string) {
	_, _ = fmt.Fprintln(keeper.stderr, line)
}
