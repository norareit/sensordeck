package winbackground

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

// ErrExists is the error of Create when an event of the name exists already.
var ErrExists = errors.New("an event of the name exists already")

// kernelLibrary is loaded from the system directory only, and lazily.
var (
	kernelLibrary   = windows.NewLazySystemDLL("kernel32.dll")
	procFreeConsole = kernelLibrary.NewProc("FreeConsole")
)

// Event is a named manual-reset event that this process created and holds until Close.
type Event struct {
	handle    windows.Handle
	closing   windows.Handle // an unnamed event that Close sets, so that the waiting goroutine ends
	signalled chan struct{}
	done      chan struct{}
}

// Create creates the event, not signalled. When one of the name exists it closes the handle CreateEvent gave and gives
// ErrExists. A goroutine waits on the event and on the unnamed event of Close, and closes the channel of Signalled
// when the event was signalled.
func Create(name string) (*Event, error) {
	pointer, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateEvent(nil, 1, 0, pointer)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}
		return nil, ErrExists
	}
	if err != nil {
		return nil, err
	}
	closing, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	event := &Event{handle: handle, closing: closing, signalled: make(chan struct{}), done: make(chan struct{})}
	go event.wait()
	return event, nil
}

func (event *Event) wait() {
	defer close(event.done)
	which, err := windows.WaitForMultipleObjects([]windows.Handle{event.handle, event.closing}, false, windows.INFINITE)
	if err == nil && which == windows.WAIT_OBJECT_0 {
		close(event.signalled)
	}
}

// Signalled is closed once the event was signalled.
func (event *Event) Signalled() <-chan struct{} { return event.signalled }

// Close sets the unnamed event, waits for the goroutine to end, and closes both handles.
func (event *Event) Close() error {
	setErr := windows.SetEvent(event.closing)
	if setErr != nil {
		return fmt.Errorf("cannot end the wait on the event: %w", setErr)
	}
	<-event.done
	return errors.Join(windows.CloseHandle(event.handle), windows.CloseHandle(event.closing))
}

// open opens the event with the access and closes it after use, which is false with no error when none of the name
// exists.
func open(name string, access uint32, use func(handle windows.Handle) error) (bool, error) {
	pointer, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return false, err
	}
	handle, err := windows.OpenEvent(access, false, pointer)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	useErr := use(handle)
	closeErr := windows.CloseHandle(handle)
	return useErr == nil, errors.Join(useErr, closeErr)
}

// Exists opens the event with SYNCHRONIZE and closes it at once. An event that does not exist is false with no error.
func Exists(name string) (bool, error) {
	return open(name, windows.SYNCHRONIZE, func(windows.Handle) error { return nil })
}

// Signal opens the event with EVENT_MODIFY_STATE, sets it and closes it. An event that does not exist is false with no
// error.
func Signal(name string) (bool, error) {
	return open(name, windows.EVENT_MODIFY_STATE, windows.SetEvent)
}

// FreeConsole detaches the process from its console.
func FreeConsole() error {
	result, _, err := procFreeConsole.Call()
	if result == 0 {
		if errors.Is(err, syscall.Errno(0)) {
			return errors.New("FreeConsole failed")
		}
		return err
	}
	return nil
}
