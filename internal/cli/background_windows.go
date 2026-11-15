package cli

import (
	"errors"
	"fmt"

	"github.com/norareit/sensordeck/internal/winbackground"
)

// stopEventName is in the namespace of the session of the user, so that the commands and the run they stop share it.
const stopEventName = `Local\sensordeck-background-run`

// stopEvent is the stop event of winbackground as the commands use it.
type stopEvent struct{ event *winbackground.Event }

func (held stopEvent) signalled() <-chan struct{} { return held.event.Signalled() }

func (held stopEvent) close() { _ = held.event.Close() }

func holdStopEvent() (heldStopEvent, error) {
	event, err := winbackground.Create(stopEventName)
	if errors.Is(err, winbackground.ErrExists) {
		return nil, errStopEventExists
	}
	if err != nil {
		return nil, fmt.Errorf("cannot create the stop event %s: %w", stopEventName, err)
	}
	return stopEvent{event: event}, nil
}

func findBackgroundRun() (bool, error) {
	found, err := winbackground.Exists(stopEventName)
	if err != nil {
		return false, fmt.Errorf("cannot check the stop event %s: %w", stopEventName, err)
	}
	return found, nil
}

func signalBackgroundRun() error {
	if _, err := winbackground.Signal(stopEventName); err != nil {
		return fmt.Errorf("cannot signal the stop event %s: %w", stopEventName, err)
	}
	return nil
}

func detachConsole() error { return winbackground.FreeConsole() }
