package winhid

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"
)

func TestOpenErrorWordsTheCodesThatAnotherProgramGives(test *testing.T) {
	for _, code := range []syscall.Errno{errorAccessDenied, errorSharingViolation} {
		got := openError("p", code)
		want := fmt.Sprintf("cannot open p: %v. Another program may hold the device, such as Corsair iCUE or the software of the screen's vendor; close it and try again", code)
		if got.Error() != want {
			test.Errorf("code %d: got %q, want %q", code, got, want)
		}
		var reachable syscall.Errno
		if !errors.As(got, &reachable) || reachable != code {
			test.Errorf("code %d is not reachable with errors.As", code)
		}
	}
}

func TestOpenErrorOfAnotherCodeHasNoHint(test *testing.T) {
	const other syscall.Errno = 2
	got := openError("p", other)
	if want := fmt.Sprintf("cannot open p: %v", other); got.Error() != want {
		test.Errorf("got %q, want %q", got, want)
	}
	var reachable syscall.Errno
	if !errors.As(got, &reachable) || reachable != other {
		test.Error("the code is not reachable with errors.As")
	}
}

func TestOpenErrorKeepsTheErrorOfTheSystemReachable(test *testing.T) {
	cause := &fs.PathError{Op: "open", Path: "p", Err: syscall.EACCES}
	if !errors.Is(openError("p", cause), fs.ErrPermission) {
		test.Error("errors.Is with fs.ErrPermission does not hold")
	}
}
