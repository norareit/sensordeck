//go:build linux

// Package hidraw is the Linux transport of the screen: the discovery of its hidraw node through sysfs, and the
// transport that reads and writes the node. It is not a general hidraw library.
package hidraw

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const (
	entryPrefix = "hidraw"
	busUSB      = 3
)

// Screen is a supported screen that the discovery found.
type Screen struct {
	Node            string
	Vendor, Product uint16
	Name, Serial    string
}

// Line is the screen as the command devices writes it, and as the error of several screens lists it.
func (screen Screen) Line() string {
	return fmt.Sprintf("%s\t%04x:%04x\t%s\t%s", screen.Node, screen.Vendor, screen.Product,
		orDash(screen.Name), orDash(screen.Serial))
}

func orDash(text string) string {
	if text == "" {
		return "-"
	}
	return text
}

// Discover gives the supported screens under the roots, ordered by the number of their hidraw entry. It opens no
// node.
func Discover(sysfsRoot, devRoot string) ([]Screen, error) {
	directory := filepath.Join(sysfsRoot, "class", "hidraw")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", directory, err)
	}
	type numbered struct {
		number int
		screen Screen
	}
	var found []numbered
	for _, entry := range entries {
		number, isHidraw := entryNumber(entry.Name())
		if !isHidraw {
			continue
		}
		content, err := os.ReadFile(filepath.Join(directory, entry.Name(), "device", "uevent"))
		if err != nil {
			continue
		}
		screen, supported := parseUevent(string(content))
		if !supported {
			continue
		}
		screen.Node = filepath.Join(devRoot, entry.Name())
		found = append(found, numbered{number: number, screen: screen})
	}
	slices.SortFunc(found, func(left, right numbered) int { return left.number - right.number })
	screens := make([]Screen, len(found))
	for position, item := range found {
		screens[position] = item.screen
	}
	return screens, nil
}

// entryNumber reads the number of an entry named hidraw followed by decimal digits only.
func entryNumber(name string) (int, bool) {
	digits, found := strings.CutPrefix(name, entryPrefix)
	if !found || digits == "" {
		return 0, false
	}
	for _, character := range digits {
		if character < '0' || character > '9' {
			return 0, false
		}
	}
	number, err := strconv.Atoi(digits)
	return number, err == nil
}

// parseUevent reads the lines KEY=value of a uevent file. It accepts the content only when its HID_ID line names a
// supported screen. The node is left for the caller.
func parseUevent(content string) (Screen, bool) {
	var screen Screen
	supported := false
	for _, line := range strings.Split(content, "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch key {
		case "HID_ID":
			supported = isSupportedID(value)
		case "HID_NAME":
			screen.Name = value
		case "HID_UNIQ":
			screen.Serial = value
		}
	}
	screen.Vendor, screen.Product = smartmonitor.VendorID, smartmonitor.ProductID
	return screen, supported
}

func isSupportedID(value string) bool {
	fields := strings.Split(value, ":")
	if len(fields) != 3 {
		return false
	}
	var numbers [3]uint64
	for position, field := range fields {
		number, err := strconv.ParseUint(field, 16, 32)
		if err != nil {
			return false
		}
		numbers[position] = number
	}
	return numbers[0] == busUSB && numbers[1] == uint64(smartmonitor.VendorID) && numbers[2] == uint64(smartmonitor.ProductID)
}
