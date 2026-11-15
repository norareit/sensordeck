package winhid

import (
	"errors"
	"fmt"
	"syscall"
)

// The codes of Windows that the wording of an error and the classification of a completion name. They are
// syscall.Errno, which golang.org/x/sys/windows aliases, so that no file for Windows is needed to name them.
const (
	errorAccessDenied     syscall.Errno = 5
	errorSharingViolation syscall.Errno = 32
	errorOperationAborted syscall.Errno = 995
)

const heldByAnotherProgram = ". Another program may hold the device, such as Corsair iCUE or the software of the " +
	"screen's vendor; close it and try again"

// openErrorValue is the error of a failed opening. Its text is exact while the error of the system stays reachable.
type openErrorValue struct {
	text string
	err  error
}

func (openFailure *openErrorValue) Error() string { return openFailure.text }
func (openFailure *openErrorValue) Unwrap() error { return openFailure.err }

// openError words the failure of the opening of the path for reading and writing.
func openError(path string, err error) error {
	text := fmt.Sprintf("cannot open %s: %v", path, err)
	var code syscall.Errno
	if errors.As(err, &code) && (code == errorAccessDenied || code == errorSharingViolation) {
		text += heldByAnotherProgram
	}
	return &openErrorValue{text: text, err: err}
}
