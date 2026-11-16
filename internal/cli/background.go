package cli

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// startRetry is how long a background run waits after a failed start, as systemd does on Linux.
const startRetry = 10 * time.Second

// runInBackground does what run does with --background: it opens the log, holds the stop event and detaches on
// Windows, and repeats the start until it succeeds or a stop comes.
func (current *session) runInBackground(named string) error {
	cacheDirectory, err := current.env.userCacheDir()
	if err != nil {
		return fmt.Errorf("cannot find the cache directory: %w", err)
	}
	log, err := openRunLog(logPath(cacheDirectory), current.env.now)
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }()
	current.env.stdout, current.env.stderr = log, log
	stop, release, err := current.holdBackgroundStop()
	if err != nil {
		return err
	}
	defer release()
	for {
		err := current.startRun(stop, named)
		if err == nil {
			return nil
		}
		if errors.Is(err, errLogged) {
			return err
		}
		current.say("sensordeck: " + err.Error())
		current.say(fmt.Sprintf("starting again in %s", startRetry))
		if !current.env.wait(stop, startRetry) {
			current.say("stopped")
			return nil
		}
	}
}

// startRun makes one start of run. A failure before the run is feeding is an error to try again after, and a failure
// of the feeding is written to the log and ends the run.
func (current *session) startRun(stop context.Context, named string) error {
	prepared, err := current.prepareRun(named)
	if err != nil {
		return err
	}
	opened, restarter, closeTransport, err := current.open()
	if err != nil {
		return err
	}
	if err := current.serveRun(stop, prepared, opened, restarter, closeTransport); err != nil {
		current.say("sensordeck: " + err.Error())
		return errLogged
	}
	return nil
}

// holdBackgroundStop holds the stop event and detaches from the console on Windows, and gives the context that ends at
// a stop of the process or a signal of the event, with the function that ends the hold. Its error is written to the log
// as errLogged.
func (current *session) holdBackgroundStop() (context.Context, func(), error) {
	stop, cancel := current.env.notifyStop(context.Background())
	if current.env.system != "windows" {
		return stop, cancel, nil
	}
	held, err := current.env.holdStopEvent()
	if errors.Is(err, errStopEventExists) {
		cancel()
		current.say("another background run of sensordeck is running")
		return nil, nil, errLogged
	}
	if err != nil {
		cancel()
		current.say("sensordeck: " + err.Error())
		return nil, nil, errLogged
	}
	// The detach comes after the log and the event, so that a log that cannot be opened is still reported on the console.
	_ = current.env.detachConsole()
	stopping, cancelStopping := context.WithCancel(stop)
	go func() {
		select {
		case <-held.signalled():
			cancelStopping()
		case <-stopping.Done():
		}
	}()
	return stopping, func() {
		cancelStopping()
		cancel()
		held.close()
	}, nil
}
