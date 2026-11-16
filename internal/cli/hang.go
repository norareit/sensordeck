package cli

import (
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// A screen whose firmware has hung times out every write. The texts below are what the user is told of it.
const (
	hungPrefix    = "the screen has hung. "
	hungByItself  = "It recovers by itself once no program has held it open for a few seconds, so run the command again"
	hungNoRecover = `Installing the current udev rule lets sensordeck recover it, see "Linux: access to the screen" in ` +
		`https://github.com/norareit/sensordeck#linux-access-to-the-screen. Unplugging the screen once recovers it now`
	hungSuspending = "suspending the screen to recover it"
)

// screenPower is what the command line knows of the power state of the screen's USB device, which is Linux only.
type screenPower interface {
	// timedOut tells whether the error of a write is a timeout of the screen.
	timedOut(err error) bool
	// recovery reads the files of the USB device anew, and gives how to recover a hung screen by suspending it, or false
	// when that is not possible.
	recovery() (hangRecovery, bool)
}

// hangRecovery is a recovery by a suspend and a resume of the screen's USB device.
type hangRecovery struct {
	delay     time.Duration // the autosuspend delay of the device
	suspended func() bool   // reads whether the device is suspended now
}

// noScreenPower is the power of a system where no failure of a write is a timeout and no recovery is possible.
type noScreenPower struct{}

func (noScreenPower) timedOut(error) bool            { return false }
func (noScreenPower) recovery() (hangRecovery, bool) { return hangRecovery{}, false }

// hangedError is the error of a command whose write timed out. It keeps the error of the write reachable.
type hangedError struct {
	text string
	err  error
}

func (hanged *hangedError) Error() string { return hanged.text }
func (hanged *hangedError) Unwrap() error { return hanged.err }

// hangError words the failure of a write of a command for the user when it is a timeout, and gives any other error
// as it is.
func hangError(err error, power screenPower) error {
	if err == nil || !power.timedOut(err) {
		return err
	}
	advice := hungNoRecover
	if _, possible := power.recovery(); possible {
		advice = hungByItself
	}
	return &hangedError{text: err.Error() + ": " + hungPrefix + advice, err: err}
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
