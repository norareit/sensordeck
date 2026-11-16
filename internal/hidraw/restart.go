//go:build linux

package hidraw

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// The screen has the string descriptors 0 to 3, so the first it does not have is 4. Asking for the ones it lacks
	// makes its firmware restart, and the index is one byte of wValue, so 255 is the last.
	firstMissingString = 4
	lastStringIndex    = 255
	requestTimeout     = time.Second
	scanLimit          = 10 * time.Second
	reopenPoll         = 100 * time.Millisecond
	reopenLimit        = 10 * time.Second

	getDescriptor    = 6
	stringDescriptor = 0x0300
	deviceToHost     = 0x80
	englishUS        = 0x0409
	stringLength     = 255
)

// RestartOutcome is how far a restart of the screen got.
type RestartOutcome int

const (
	RestartNotPossible RestartOutcome = iota // nothing was sent, and the result holds the error
	NotRestarted                             // the screen did not leave the bus
	RestartedOpen                            // the screen left the bus and is open again
	RestartedNotOpen                         // the screen left the bus and is not open again
)

// RestartResult is the outcome of a restart. Index is the index of the string descriptor whose request the screen left
// the bus at, set for the two restarted outcomes, and Err the reason the restart was not possible.
type RestartResult struct {
	Outcome RestartOutcome
	Index   int
	Err     error
}

// ControlRequest is a request on the control endpoint that has data coming from the device.
type ControlRequest struct {
	RequestType, Request uint8
	Value, Index, Length uint16
	Timeout              time.Duration
}

// Usbfs opens the node of a USB device in usbfs.
type Usbfs interface {
	Open(path string) (UsbfsNode, error)
}

// UsbfsNode is an open node of a USB device in usbfs.
type UsbfsNode interface {
	Control(request ControlRequest) error
	Close() error
}

// Reopenable is the transport that a restart closes and opens again. *Transport satisfies it. It is not called Screen,
// which is the name of a screen that discovery found.
type Reopenable interface {
	Close() error
	Reopen() error
}

// Restart restarts the firmware of a screen by asking its USB device for string descriptors it does not have, through
// the usbfs node of the device with usbhid still bound. It sends nothing else, and writes no file.
type Restart struct {
	SysfsRoot, DevRoot string
	Node               string // the hidraw node of the screen, as it was opened last
	Usbfs              Usbfs
	Screen             Reopenable
	Now                func() time.Time
	Wait               func(stop context.Context, duration time.Duration) bool // false when stop ended the wait
}

// Run makes the restart. A stop ends the requests before the next one and the wait before the next opening.
func (restart Restart) Run(stop context.Context) RestartResult {
	device, devnum, node, err := restart.prepare()
	if err != nil {
		return RestartResult{Outcome: RestartNotPossible, Err: err}
	}
	_ = restart.Screen.Close()
	index, left := restart.scan(stop, device, devnum, node)
	_ = node.Close()
	if !left {
		return RestartResult{Outcome: NotRestarted}
	}
	return restart.awaitScreen(stop, index)
}

// prepare finds the USB device of the screen, reads its number and opens its usbfs node, so that a restart that
// cannot be made sends nothing and closes nothing.
func (restart Restart) prepare() (device string, devnum int, node UsbfsNode, err error) {
	device, found := USBDevice(restart.SysfsRoot, restart.DevRoot, restart.Node)
	if !found {
		return "", 0, nil, fmt.Errorf("no USB device in sysfs for %s", restart.Node)
	}
	busnum, err := readNumber(device, "busnum")
	if err != nil {
		return "", 0, nil, err
	}
	devnum, err = readNumber(device, "devnum")
	if err != nil {
		return "", 0, nil, err
	}
	path := filepath.Join(restart.DevRoot, "bus", "usb", fmt.Sprintf("%03d", busnum), fmt.Sprintf("%03d", devnum))
	node, err = restart.Usbfs.Open(path)
	if err != nil {
		return "", 0, nil, err
	}
	return device, devnum, node, nil
}

func readNumber(device, name string) (int, error) {
	path := filepath.Join(device, name)
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	number, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("%s does not hold a number: %w", path, err)
	}
	return number, nil
}

// scan asks for the string descriptors from the first the screen lacks until it has left the bus, and tells the index
// whose request it left at. It gives false when the indexes ran out, the time ran out or the stop ended it.
func (restart Restart) scan(stop context.Context, device string, devnum int, node UsbfsNode) (int, bool) {
	begin := restart.Now()
	for index := firstMissingString; index <= lastStringIndex; index++ {
		if stop.Err() != nil || restart.Now().Sub(begin) >= scanLimit {
			return 0, false
		}
		err := node.Control(ControlRequest{
			RequestType: deviceToHost,
			Request:     getDescriptor,
			Value:       stringDescriptor | uint16(index),
			Index:       englishUS,
			Length:      stringLength,
			Timeout:     requestTimeout,
		})
		if errors.Is(err, syscall.ENODEV) || deviceChanged(device, devnum) {
			return index, true
		}
	}
	return 0, false
}

// deviceChanged tells whether the USB device is gone from sysfs or has another number, which is what a screen that
// left the bus looks like before its request has given ENODEV.
func deviceChanged(device string, devnum int) bool {
	current, err := readNumber(device, "devnum")
	return err != nil || current != devnum
}

// awaitScreen opens the screen again once it has enumerated, polling until the limit. The poll also covers the moment
// between the node appearing and udev granting access to it, in which an opening fails.
func (restart Restart) awaitScreen(stop context.Context, index int) RestartResult {
	left := restart.Now()
	for stop.Err() == nil {
		if restart.Screen.Reopen() == nil {
			return RestartResult{Outcome: RestartedOpen, Index: index}
		}
		if restart.Now().Sub(left) >= reopenLimit || !restart.Wait(stop, reopenPoll) {
			break
		}
	}
	return RestartResult{Outcome: RestartedNotOpen, Index: index}
}
