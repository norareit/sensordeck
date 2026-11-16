package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// A screen whose firmware has hung times out every write, and a restart of its firmware recovers it. The texts below
// are what the user is told of it.
const (
	hungText         = "the screen has hung"
	restartingLine   = "restarting the screen to recover it"
	restartedFormat  = "screen restarted at string descriptor %d"
	notRestartedLine = "the screen did not restart, trying again every minute. Unplugging the screen once recovers it"
	stillHungLine    = "the screen is still hung after its restart, trying again every minute. Unplugging the screen once recovers it"
	// cannotRestartFormat follows hungText in the line of run and in the error of a command, and takes the error that
	// made the restart impossible.
	cannotRestartFormat = " and sensordeck cannot restart it: %v. Installing the current udev rule lets sensordeck recover it, see " +
		`"Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen. ` +
		"Unplugging the screen once recovers it now"
	restartedSuffix    = ". sensordeck restarted it, so run the command again"
	notRestartedSuffix = " and did not restart. Unplugging the screen once recovers it"
)

// restartOutcome is how far a restart of the screen got.
type restartOutcome int

const (
	restartNotPossible restartOutcome = iota // nothing was sent, and the result holds the error
	notRestarted                             // the screen did not leave the bus
	restartedOpen                            // the screen left the bus and is open again
	restartedNotOpen                         // the screen left the bus and is not open again
)

// restartResult is what a restart gave. index is the string descriptor whose request the screen left the bus at, set
// for the two restarted outcomes, and err the reason the restart was not possible.
type restartResult struct {
	outcome restartOutcome
	index   int
	err     error
}

// screenRestarter is what the command line knows of hangs and restarts of the screen, which is Linux only.
type screenRestarter interface {
	// timedOut tells whether the error of a write is a timeout of the screen.
	timedOut(err error) bool
	// restart restarts the firmware of the screen, which closes the transport and, when the screen came back, opens it
	// again. A stop ends it before the next request or the next attempt to open.
	restart(stop context.Context) restartResult
}

// noScreenRestarter is the restarter of a system where no failure of a write is a timeout and no restart is possible.
type noScreenRestarter struct{}

func (noScreenRestarter) timedOut(error) bool { return false }
func (noScreenRestarter) restart(context.Context) restartResult {
	return restartResult{outcome: restartNotPossible, err: errors.New("restarting the screen is supported on Linux only")}
}

// hangedError is the error of a command whose write timed out. It keeps the error of the write reachable.
type hangedError struct {
	text string
	err  error
}

func (hanged *hangedError) Error() string { return hanged.text }
func (hanged *hangedError) Unwrap() error { return hanged.err }

// restartAfterHang makes a restart when the failure of a write of a command is a timeout, and words the failure for the
// user. It gives any other error as it is. The command does not write again what timed out. The restart is ended by
// the signal that ends the process, since the commands keep the default handling of Ctrl+C.
func restartAfterHang(err error, restarter screenRestarter) error {
	if err == nil || !restarter.timedOut(err) {
		return err
	}
	var advice string
	switch result := restarter.restart(context.Background()); result.outcome {
	case restartedOpen, restartedNotOpen:
		advice = restartedSuffix
	case notRestarted:
		advice = notRestartedSuffix
	default:
		advice = fmt.Sprintf(cannotRestartFormat, result.err)
	}
	return &hangedError{text: err.Error() + ": " + hungText + advice, err: err}
}

// observedTransport tells the function of every write that succeeded, so that the feeder sees the writes of an upload
// as well as its own.
type observedTransport struct {
	transport
	written func()
}

func (observed *observedTransport) Write(report smartmonitor.Report) error {
	if err := observed.transport.Write(report); err != nil {
		return err
	}
	observed.written()
	return nil
}
