// Package winhid is the transport of the screen on Windows, and the layer of the Commander on it: the listing of the
// HID collections, the opening of one, and the reports read and written with a time limit. What touches the system is
// behind the Seam, so that all the rest runs and is tested on any system.
package winhid

import "syscall"

// Handle is an open collection of the HID class.
type Handle uintptr

// Caps is what the caps of a collection say of it.
type Caps struct {
	UsagePage, Usage          uint16
	InputLength, OutputLength uint16 // each counts the report ID byte
}

// Operation is an overlapped read or write that was started. Its buffer and what the system needs for it belong to
// the system until Reap returned.
type Operation interface {
	// Wait waits at most the milliseconds for the completion. It gives true when the operation completed within the
	// wait, and false when the wait ended without it or failed.
	Wait(milliseconds uint32) bool
	// Cancel asks for the cancellation of the operation, which then completes with ERROR_OPERATION_ABORTED unless it
	// had completed already.
	Cancel()
	// Reap waits for the completion and gives its byte count and its code, 0 for success, and frees what the
	// operation held.
	Reap() (count uint32, code syscall.Errno)
}

// Seam is the calls into the system that the transport rests on, and nothing else.
type Seam interface {
	// Paths gives the paths of the present collections of the HID class.
	Paths() ([]string, error)
	// OpenQuery opens the path with no access, which reads no reports and gives the attributes, the strings and the
	// caps. OpenReadWrite opens it for reading and writing, shared, for overlapped operations.
	OpenQuery(path string) (Handle, error)
	OpenReadWrite(path string) (Handle, error)
	Attributes(handle Handle) (vendor, product uint16, err error)
	Product(handle Handle) (string, error)
	Serial(handle Handle) (string, error)
	Caps(handle Handle) (Caps, error)
	// StartRead and StartWrite start an operation on the buffer. An error, which is that of the system, means that
	// nothing is pending.
	StartRead(handle Handle, buffer []byte) (Operation, error)
	StartWrite(handle Handle, buffer []byte) (Operation, error)
	Close(handle Handle) error
}
