//go:build linux

package hidraw

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const usbDeviceType = "DEVTYPE=usb_device"

// TimedOut tells whether the error of a write is the timeout of the kernel, which is what a screen whose firmware has
// hung gives.
func TimedOut(err error) bool { return errors.Is(err, syscall.ETIMEDOUT) }

// USBDevice gives the directory in sysfs of the USB device that the hidraw entry of the node belongs to, found by
// walking up from the HID device of the entry to the first directory that says it is a USB device. It gives false
// when no entry has that node, or the entry has no USB ancestor under the sysfs root.
func USBDevice(sysfsRoot, devRoot, node string) (string, bool) {
	entry, found := entryOfNode(sysfsRoot, devRoot, node)
	if !found {
		return "", false
	}
	root, err := filepath.EvalSymlinks(sysfsRoot)
	if err != nil {
		return "", false
	}
	hidDevice := filepath.Join(sysfsRoot, "class", "hidraw", entryPrefix+strconv.Itoa(entry.Number), "device")
	directory, err := filepath.EvalSymlinks(hidDevice)
	if err != nil {
		return "", false
	}
	for directory != root && directory != filepath.Dir(directory) {
		if isUSBDevice(directory) {
			return directory, true
		}
		directory = filepath.Dir(directory)
	}
	return "", false
}

func isUSBDevice(directory string) bool {
	content, err := os.ReadFile(filepath.Join(directory, "uevent"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(content), "\n") {
		if line == usbDeviceType {
			return true
		}
	}
	return false
}
