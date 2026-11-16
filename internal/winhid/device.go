package winhid

import (
	"errors"
	"fmt"
	"io"
	"time"
)

var errClosed = errors.New("the device is closed")

// Device is a collection open for reading and writing. It is not safe for use from several goroutines.
type Device struct {
	seam   Seam
	handle Handle
	caps   Caps
	closed bool
}

// Open opens the path for reading and writing, shared and overlapped, and reads its caps. The handle is closed when
// the caps cannot be read.
func Open(seam Seam, path string) (*Device, error) {
	handle, err := seam.OpenReadWrite(path)
	if err != nil {
		return nil, openError(path, err)
	}
	caps, err := seam.Caps(handle)
	if err != nil {
		_ = seam.Close(handle)
		return nil, fmt.Errorf("cannot open %s: its caps cannot be read: %w", path, err)
	}
	return &Device{seam: seam, handle: handle, caps: caps}, nil
}

// Caps gives the caps of the collection as read at the opening.
func (device *Device) Caps() Caps { return device.caps }

// WriteReport writes one output report: the report ID byte 0x00, the payload, and zeros up to the output length of
// the caps, which a write must have exactly.
func (device *Device) WriteReport(payload []byte, limit time.Duration) error {
	if device.closed {
		return errClosed
	}
	buffer := make([]byte, device.caps.OutputLength)
	copy(buffer[1:], payload)
	_, err := run(device.seam, writing, device.handle, buffer, limit)
	return err
}

// ReadReport reads one input report within the limit and gives its bytes after the report ID. A read of no bytes is
// io.EOF. The timeout is an error for which errors.Is with smartmonitor.ErrReadTimeout holds.
func (device *Device) ReadReport(limit time.Duration) ([]byte, error) {
	if device.closed {
		return nil, errClosed
	}
	buffer := make([]byte, device.caps.InputLength)
	count, err := run(device.seam, reading, device.handle, buffer, limit)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, io.EOF
	}
	return buffer[1:count], nil
}

// Close closes the handle. A second Close does nothing.
func (device *Device) Close() error {
	if device.closed {
		return nil
	}
	device.closed = true
	return device.seam.Close(device.handle)
}
