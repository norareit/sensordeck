//go:build linux

// Package hidraw is the Linux transport of the screen: the discovery of its hidraw node through sysfs, and the
// transport that reads and writes the node. It also holds the listing of hidraw entries and the opening of a node,
// which the driver of the controller shares. It is not a general hidraw library.
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

// Entry is one hidraw<N> entry of <sysfs root>/class/hidraw whose device/uevent could be read.
type Entry struct {
	Number          int
	Node            string // <dev root>/hidraw<N>
	Bus             uint64 // from HID_ID, like Vendor and Product; ValidID is false when HID_ID is missing or malformed
	Vendor, Product uint64
	ValidID         bool
	Name, Phys      string // HID_NAME, HID_PHYS
	Serial          string // HID_UNIQ
}

// List gives the entries under the roots ordered by Number. A missing directory gives none and no error. An entry
// whose uevent cannot be read is left out. It opens no node.
func List(sysfsRoot, devRoot string) ([]Entry, error) {
	directory := filepath.Join(sysfsRoot, "class", "hidraw")
	directoryEntries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", directory, err)
	}
	var entries []Entry
	for _, directoryEntry := range directoryEntries {
		number, isHidraw := entryNumber(directoryEntry.Name())
		if !isHidraw {
			continue
		}
		content, err := os.ReadFile(filepath.Join(directory, directoryEntry.Name(), "device", "uevent"))
		if err != nil {
			continue
		}
		entry := parseUevent(string(content))
		entry.Number = number
		entry.Node = filepath.Join(devRoot, directoryEntry.Name())
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(left, right Entry) int { return left.Number - right.Number })
	return entries, nil
}

// Discover gives the supported screens under the roots, ordered by the number of their hidraw entry. It opens no
// node.
func Discover(sysfsRoot, devRoot string) ([]Screen, error) {
	entries, err := List(sysfsRoot, devRoot)
	if err != nil {
		return nil, err
	}
	var screens []Screen
	for _, entry := range entries {
		if !entry.ValidID || entry.Bus != busUSB || entry.Vendor != uint64(smartmonitor.VendorID) ||
			entry.Product != uint64(smartmonitor.ProductID) {
			continue
		}
		screens = append(screens, Screen{Node: entry.Node, Vendor: smartmonitor.VendorID,
			Product: smartmonitor.ProductID, Name: entry.Name, Serial: entry.Serial})
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

// parseUevent reads the lines KEY=value of a uevent file. The number and the node are left for the caller.
func parseUevent(content string) Entry {
	var entry Entry
	for _, line := range strings.Split(content, "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch key {
		case "HID_ID":
			entry.Bus, entry.Vendor, entry.Product, entry.ValidID = parseID(value)
		case "HID_NAME":
			entry.Name = value
		case "HID_PHYS":
			entry.Phys = value
		case "HID_UNIQ":
			entry.Serial = value
		}
	}
	return entry
}

// parseID reads the three hexadecimal numbers of a HID_ID value.
func parseID(value string) (bus, vendor, product uint64, valid bool) {
	fields := strings.Split(value, ":")
	if len(fields) != 3 {
		return 0, 0, 0, false
	}
	var numbers [3]uint64
	for position, field := range fields {
		number, err := strconv.ParseUint(field, 16, 32)
		if err != nil {
			return 0, 0, 0, false
		}
		numbers[position] = number
	}
	return numbers[0], numbers[1], numbers[2], true
}
