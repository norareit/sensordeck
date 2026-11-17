package commandercore

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/winhid"
)

const (
	// The usage page and usage of the report descriptor of interface 0, which tell its collection apart from the
	// others of the controller on Windows, where the report ID byte counts in the lengths of the caps.
	interfaceZeroUsagePage uint16 = 0xFF42
	interfaceZeroUsage     uint16 = 0x0001
	interfaceZeroLength           = uint16(1 + reportSize)
)

// HID is the finder of the systems with the HID API of Windows: interface 0 is the first collection, in the order of
// the listing, of the controller whose path holds the field mi_00 and whose usage and report lengths are those of
// interface 0. A collection of interface 0 that does not match is never opened.
//
// acquire takes the lock of the controller before each opening, and its release is called after the close of the node.
func HID(seam winhid.Seam, acquire func() (release func(), err error)) Finder {
	return Finder{
		find: func() (string, error) { return findCollection(seam) },
		open: func(path string) (node, error) {
			release, err := acquire()
			if err != nil {
				return nil, controllerOpenError(err)
			}
			device, err := winhid.Open(seam, path)
			if err != nil {
				release()
				return nil, err
			}
			return hidNode{device: device, release: release}, nil
		},
	}
}

func isInterfaceZero(collection winhid.Collection) bool {
	return collection.Caps.UsagePage == interfaceZeroUsagePage && collection.Caps.Usage == interfaceZeroUsage &&
		collection.Caps.InputLength == interfaceZeroLength && collection.Caps.OutputLength == interfaceZeroLength
}

func findCollection(seam winhid.Seam) (string, error) {
	collections, err := winhid.List(seam)
	if err != nil {
		return "", err
	}
	var nearMisses []winhid.Collection
	for _, collection := range collections {
		if collection.Vendor != vendor || collection.Product != product || !winhid.InterfaceZero(collection.Path) {
			continue
		}
		if isInterfaceZero(collection) {
			return collection.Path, nil
		}
		nearMisses = append(nearMisses, collection)
	}
	if len(nearMisses) == 0 {
		return "", ErrNoController
	}
	return "", nearMissError(nearMisses)
}

// nearMissError names what the collections of interface 0 that do not match gave, beside what is expected, since a
// device that is found and not matched is otherwise a silent "not found".
func nearMissError(nearMisses []winhid.Collection) error {
	found := make([]string, len(nearMisses))
	for position, collection := range nearMisses {
		found[position] = fmt.Sprintf("%s has usage page 0x%04X, usage 0x%04X and reports of %d and %d bytes",
			collection.Path, collection.Caps.UsagePage, collection.Caps.Usage, collection.Caps.InputLength,
			collection.Caps.OutputLength)
	}
	return fmt.Errorf("no collection of interface 0 matches: %s, interface 0 has usage page 0x%04X, usage 0x%04X "+
		"and reports of %d and %d bytes", strings.Join(found, "; "), interfaceZeroUsagePage, interfaceZeroUsage,
		interfaceZeroLength, interfaceZeroLength)
}

// hidNode is a node of the HID API: it writes the command as the output report of 385 bytes and reads the input
// report of 385 bytes, whose first byte, the report ID, is dropped, so that the answer is the 384 bytes that Linux
// reads.
type hidNode struct {
	device  *winhid.Device
	release func()
}

func (node hidNode) send(command []byte) error {
	return node.device.WriteReport(command, winhid.WriteTimeLimit)
}

func (node hidNode) receive(timeLimit time.Duration) ([]byte, error) {
	answer, err := node.device.ReadReport(timeLimit)
	switch {
	case errors.Is(err, smartmonitor.ErrReadTimeout):
		return nil, errNoAnswer
	case errors.Is(err, io.EOF):
		return nil, errors.New("read failed: EOF")
	case err != nil:
		return nil, err
	}
	return answer, nil
}

func (node hidNode) close() error {
	err := node.device.Close()
	node.release()
	return err
}
