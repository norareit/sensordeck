//go:build linux

package hidraw

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"
	"time"

	"github.com/norareit/sensordeck/internal/devicelock"
	"github.com/norareit/sensordeck/internal/screenlist"
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
	open func() (opened, error) // the way it was opened first, used again by Reopen
	file *os.File               // nil while no node is open
	node string                 // the path of the node opened last
}

// opened is a node that an opening gave.
type opened struct {
	file *os.File
	node string
}

// OpenDiscovered opens the node of the one supported screen under the roots. A Reopen does the discovery anew.
func OpenDiscovered(sysfsRoot, devRoot, procRoot string) (*Transport, error) {
	return openWith(func() (opened, error) { return openDiscovered(sysfsRoot, devRoot, procRoot) })
}

// OpenPath opens the node at the path as it is. A Reopen opens the same path.
func OpenPath(path, procRoot string) (*Transport, error) {
	return openWith(func() (opened, error) {
		file, err := openNode(path, procRoot)
		if err != nil {
			return opened{}, err
		}
		return opened{file: file, node: path}, nil
	})
}

func openWith(open func() (opened, error)) (*Transport, error) {
	node, err := open()
	if err != nil {
		return nil, err
	}
	return &Transport{open: open, file: node.file, node: node.node}, nil
}

func openDiscovered(sysfsRoot, devRoot, procRoot string) (opened, error) {
	screens, err := Discover(sysfsRoot, devRoot)
	if err != nil {
		return opened{}, err
	}
	switch len(screens) {
	case 0:
		return opened{}, errors.New(screenlist.NoneFound)
	case 1:
		file, err := openNode(screens[0].Node, procRoot)
		return opened{file: file, node: screens[0].Node}, err
	}
	lines := make([]string, len(screens))
	for position, screen := range screens {
		lines[position] = screen.Line()
	}
	return opened{}, screenlist.Several(lines)
}

// entryOfNode gives the entry whose node is the same file as the path, and false when no entry is, or when the listing
// or a stat fails. os.SameFile follows the links of the fake screen and of udev alike.
func entryOfNode(sysfsRoot, devRoot, path string) (Entry, bool) {
	wanted, err := os.Stat(path)
	if err != nil {
		return Entry{}, false
	}
	entries, err := List(sysfsRoot, devRoot)
	if err != nil {
		return Entry{}, false
	}
	for _, entry := range entries {
		if candidate, err := os.Stat(entry.Node); err == nil && os.SameFile(wanted, candidate) {
			return entry, true
		}
	}
	return Entry{}, false
}

// Node is the path of the node opened last. A Reopen that failed leaves it as it was.
func (transport *Transport) Node() string { return transport.node }

// OpenNode opens the node for reading and writing, without making it the controlling terminal, and checks that it
// takes a time limit for reading, so that no read waits without one. O_NOCTTY matters only for the node of a fake
// device, which is a terminal that must not become the controlling terminal of the process. Its error is the error
// of the system, or the failure of that check.
func OpenNode(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, err
	}
	if err := file.SetReadDeadline(time.Time{}); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("the node does not support a time limit for reading: %w", err)
	}
	return file, nil
}

// openNode opens the node of the screen, takes its lock and words an opening that failed for the user. A lock held by
// another sensordeck process closes the node again and gives the error of the lock as it is, so that it is not taken
// for an opening that failed.
func openNode(path, procRoot string) (*os.File, error) {
	file, err := OpenNode(path)
	if err != nil {
		return nil, openError(path, err)
	}
	if err := devicelock.LockNode(file, procRoot); err != nil {
		_ = file.Close()
		return nil, err
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
	node, err := transport.open()
	if err != nil {
		return err
	}
	transport.file, transport.node = node.file, node.node
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
