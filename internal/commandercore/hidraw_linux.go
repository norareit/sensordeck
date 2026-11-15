//go:build linux

package commandercore

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/hidraw"
)

const (
	busUSB = 3
	// writeSize is a write on Linux: the report ID byte 0x00, the 96 bytes the driver writes, and no more.
	writeSize = 1 + answerSize
	// interfaceZero ends the HID_PHYS of the node of USB interface 0, the one that is used. The node of interface 1
	// is never opened.
	interfaceZero = "/input0"
)

// findNode gives the node of the lowest numbered entry with a valid HID_ID of bus 3, vendor 0x1B1C and product
// 0x0C2A, and a HID_PHYS ending in /input0, or false. It opens no node.
func findNode(sysfsRoot, devRoot string) (string, bool) {
	entries, err := hidraw.List(sysfsRoot, devRoot)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if entry.ValidID && entry.Bus == busUSB && entry.Vendor == vendor && entry.Product == product &&
			strings.HasSuffix(entry.Phys, interfaceZero) {
			return entry.Node, true
		}
	}
	return "", false
}

// Hidraw is the finder of Linux: the node of interface 0 under the sysfs and dev roots, opened as hidraw does.
func Hidraw(sysfsRoot, devRoot string) Finder {
	return Finder{
		find: func() (string, error) {
			path, found := findNode(sysfsRoot, devRoot)
			if !found {
				return "", ErrNoController
			}
			return path, nil
		},
		open: func(path string) (node, error) {
			file, err := hidraw.OpenNode(path)
			if err != nil {
				return nil, fmt.Errorf("cannot open %s: %w", path, err)
			}
			return hidrawNode{file: file}, nil
		},
	}
}

// hidrawNode is a node of hidraw. It writes the 97 bytes of the trace checked on the device and reads into the whole
// input report, so that hidraw discards nothing of it.
type hidrawNode struct{ file *os.File }

// encode gives the 97 bytes of a write: the report ID byte 0x00, the command, and zeros.
func encode(command []byte) []byte {
	written := make([]byte, writeSize)
	copy(written[1:], command)
	return written
}

func (node hidrawNode) send(command []byte) error {
	written, err := node.file.Write(encode(command))
	if err != nil {
		return fmt.Errorf("write failed: %v", err)
	}
	if written < writeSize {
		return fmt.Errorf("short write of %d bytes", written)
	}
	return nil
}

func (node hidrawNode) receive(timeLimit time.Duration) ([]byte, error) {
	if err := node.file.SetReadDeadline(time.Now().Add(timeLimit)); err != nil {
		return nil, fmt.Errorf("read failed: %v", err)
	}
	answer := make([]byte, reportSize)
	count, err := node.file.Read(answer)
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return nil, errNoAnswer
	}
	if err != nil {
		return nil, fmt.Errorf("read failed: %v", err)
	}
	return answer[:count], nil
}

func (node hidrawNode) close() error { return node.file.Close() }
