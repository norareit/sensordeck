//go:build linux

package hidraw

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const (
	reportSize          = len(smartmonitor.Report{})
	permissionDeniedFmt = "cannot open %s: permission denied. Install the udev rule and replug the screen, " +
		`see "Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen`
)

var errNoNode = errors.New("no node is open")

// Transport reads and writes the hidraw node of a screen. It is not safe for use from several goroutines.
type Transport struct {
	open func() (*os.File, error) // the way it was opened first, used again by Reopen
	file *os.File                 // nil while no node is open
}

// OpenDiscovered opens the node of the one supported screen under the roots. A Reopen does the discovery anew.
func OpenDiscovered(sysfsRoot, devRoot string) (*Transport, error) {
	return openWith(func() (*os.File, error) { return openDiscovered(sysfsRoot, devRoot) })
}

// OpenPath opens the node at the path as it is. A Reopen opens the same path.
func OpenPath(path string) (*Transport, error) {
	return openWith(func() (*os.File, error) { return openNode(path) })
}

func openWith(open func() (*os.File, error)) (*Transport, error) {
	file, err := open()
	if err != nil {
		return nil, err
	}
	return &Transport{open: open, file: file}, nil
}

func openDiscovered(sysfsRoot, devRoot string) (*os.File, error) {
	screens, err := Discover(sysfsRoot, devRoot)
	if err != nil {
		return nil, err
	}
	switch len(screens) {
	case 0:
		return nil, errors.New("no supported screen found")
	case 1:
		return openNode(screens[0].Node)
	}
	lines := make([]string, len(screens))
	for position, screen := range screens {
		lines[position] = screen.Line()
	}
	return nil, errors.New("more than one supported screen is connected, name one with --device:\n" +
		strings.Join(lines, "\n"))
}

// openNode opens the node and checks that it can be given a time limit for reading, so that no read waits without
// one. O_NOCTTY matters only for the node of a fake screen, which is a terminal that must not become the controlling
// terminal of the process.
func openNode(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, openError(path, err)
	}
	if err := file.SetReadDeadline(time.Time{}); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("cannot open %s: the node does not support a time limit for reading: %w", path, err)
	}
	return file, nil
}

// openErrorValue is the error of a failed opening. Its text is exact while the error of the system stays reachable.
type openErrorValue struct {
	text string
	err  error
}

func (openFailure *openErrorValue) Error() string { return openFailure.text }
func (openFailure *openErrorValue) Unwrap() error { return openFailure.err }

func openError(path string, err error) error {
	cause := err
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		cause = pathError.Err
	}
	if errors.Is(err, fs.ErrPermission) {
		return &openErrorValue{text: fmt.Sprintf(permissionDeniedFmt, path), err: err}
	}
	return &openErrorValue{text: fmt.Sprintf("cannot open %s: %v", path, cause), err: err}
}

// Write writes the report as 65 bytes in one write: the report ID byte 0x00, then the 64 bytes.
func (transport *Transport) Write(report smartmonitor.Report) error {
	if transport.file == nil {
		return errNoNode
	}
	buffer := make([]byte, 0, 1+reportSize)
	buffer = append(buffer, 0x00)
	buffer = append(buffer, report[:]...)
	written, err := transport.file.Write(buffer)
	if err != nil {
		return err
	}
	if written < len(buffer) {
		return fmt.Errorf("short write: %d of %d bytes", written, len(buffer))
	}
	return nil
}

// Read returns one input report. A report shorter than 64 bytes is filled up with 0x00.
func (transport *Transport) Read(timeout time.Duration) (smartmonitor.Report, error) {
	var report smartmonitor.Report
	if transport.file == nil {
		return report, errNoNode
	}
	if err := transport.file.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return report, err
	}
	buffer := make([]byte, reportSize)
	count, err := transport.file.Read(buffer)
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return report, fmt.Errorf("%w: %w", smartmonitor.ErrReadTimeout, err)
	}
	if err != nil {
		return report, err
	}
	if count == 0 {
		return report, io.EOF
	}
	copy(report[:], buffer[:count])
	return report, nil
}

// Reopen closes the node and opens the device again the way it was opened first. When the opening fails the
// transport holds no node, and a later Reopen tries again.
func (transport *Transport) Reopen() error {
	if transport.file != nil {
		_ = transport.file.Close()
		transport.file = nil
	}
	file, err := transport.open()
	if err != nil {
		return err
	}
	transport.file = file
	return nil
}

// Close closes the node.
func (transport *Transport) Close() error {
	if transport.file == nil {
		return nil
	}
	err := transport.file.Close()
	transport.file = nil
	return err
}
