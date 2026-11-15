package commandercore

import (
	"errors"
	"time"
)

// errNoAnswer is a receive whose time limit passed with no answer.
var errNoAnswer = errors.New("no answer arrived")

// node is the interface 0 of the controller as the protocol uses it, whatever the system: it writes one command and
// reads one answer. The error of send and of receive, but for errNoAnswer, has the whole problem as its text.
type node interface {
	// send writes the command as one output report.
	send(command []byte) error
	// receive gives the bytes of one input report after its report ID, within the time limit.
	receive(timeLimit time.Duration) ([]byte, error)
	close() error
}

// Finder finds interface 0 of the controller and opens it, and is the one place where the systems differ.
type Finder struct {
	// find gives the path of interface 0, ErrNoController when there is none, or another error that says why nothing
	// was found.
	find func() (string, error)
	// open gives the node of the path, or an error worded cannot open <path>: ...
	open func(path string) (node, error)
}

// Problem gives the error of the finding when it is neither a found path nor ErrNoController, which is what the
// human needs to read when a controller seems to be there and is not found.
func (finder Finder) Problem() error {
	_, err := finder.find()
	if errors.Is(err, ErrNoController) {
		return nil
	}
	return err
}
