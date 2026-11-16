package winhid

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/screenlist"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

var errNoDevice = errors.New("no device is open")

// Transport reads and writes the collection of a screen. It is not safe for use from several goroutines.
type Transport struct {
	open   func() (*Device, error) // the way it was opened first, used again by Reopen
	device *Device                 // nil while none is open
}

// OpenDiscovered opens the one supported screen. A Reopen does the discovery anew.
func OpenDiscovered(seam Seam) (*Transport, error) {
	return openWith(func() (*Device, error) { return openDiscovered(seam) })
}

// OpenPath opens the path as it is, whatever its IDs, when its reports are those of the screen. A Reopen opens the
// same path.
func OpenPath(seam Seam, path string) (*Transport, error) {
	return openWith(func() (*Device, error) { return openScreen(seam, path) })
}

func openWith(open func() (*Device, error)) (*Transport, error) {
	device, err := open()
	if err != nil {
		return nil, err
	}
	return &Transport{open: open, device: device}, nil
}

func openDiscovered(seam Seam) (*Device, error) {
	collections, err := List(seam)
	if err != nil {
		return nil, err
	}
	screens, nearMisses := Screens(collections)
	switch len(screens) {
	case 0:
		return nil, errors.New(strings.Join(append([]string{screenlist.NoneFound}, NearMissLines(nearMisses)...), "\n"))
	case 1:
		return openScreen(seam, screens[0].Path)
	}
	lines := make([]string, len(screens))
	for position, screen := range screens {
		lines[position] = screen.Line()
	}
	return nil, screenlist.Several(lines)
}

// openScreen opens the path and checks that its reports are those of the screen, which a write needs exactly.
func openScreen(seam Seam, path string) (*Device, error) {
	device, err := Open(seam, path)
	if err != nil {
		return nil, err
	}
	caps := device.Caps()
	if caps.InputLength != screenReportLength || caps.OutputLength != screenReportLength {
		_ = device.Close()
		return nil, fmt.Errorf("cannot open %s: its reports are %d and %d bytes, those of the screen are %d",
			path, caps.InputLength, caps.OutputLength, screenReportLength)
	}
	return device, nil
}

// Write writes the report as one output report with the report ID byte 0x00 first.
func (transport *Transport) Write(report smartmonitor.Report) error {
	if transport.device == nil {
		return errNoDevice
	}
	return transport.device.WriteReport(report[:], WriteTimeLimit)
}

// Read returns one input report, filled up with 0x00 to 64 bytes when it is shorter.
func (transport *Transport) Read(timeout time.Duration) (smartmonitor.Report, error) {
	var report smartmonitor.Report
	if transport.device == nil {
		return report, errNoDevice
	}
	bytes, err := transport.device.ReadReport(timeout)
	if err != nil {
		return report, err
	}
	copy(report[:], bytes)
	return report, nil
}

// Reopen closes the collection and opens the device again the way it was opened first. When the opening fails the
// transport holds none, and a later Reopen tries again.
func (transport *Transport) Reopen() error {
	_ = transport.Close()
	device, err := transport.open()
	if err != nil {
		return err
	}
	transport.device = device
	return nil
}

// Close closes the collection. A second Close does nothing.
func (transport *Transport) Close() error {
	if transport.device == nil {
		return nil
	}
	err := transport.device.Close()
	transport.device = nil
	return err
}
