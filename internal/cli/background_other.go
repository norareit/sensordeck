//go:build !windows

package cli

import "errors"

var errNotWindows = errors.New("the stop event exists only on Windows")

// The functions below are called only when the system is windows, which no other build is.

func holdStopEvent() (heldStopEvent, error) { return nil, errNotWindows }

func findBackgroundRun() (bool, error) { return false, errNotWindows }

func signalBackgroundRun() error { return errNotWindows }

func detachConsole() error { return errNotWindows }
