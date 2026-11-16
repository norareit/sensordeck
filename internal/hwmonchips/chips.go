// Package hwmonchips finds the hwmon chips of a name in a sysfs tree. It has no build constraint, so that the
// sources of Linux and the custom metrics, which every system builds, share one lookup and one order.
package hwmonchips

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Chip is a directory hwmonN of class/hwmon.
type Chip struct {
	Number int    // N of hwmonN
	Path   string // <sysfs root>/class/hwmon/hwmonN
	Device string // <Path>/device resolved to its final path, or empty when it cannot be resolved
}

// Named gives the chips whose name file holds the name, without its trailing line feed. The order follows the
// hardware and not the boot: the chips with a device by the resolved path of the device, compared by bytes, and then
// by N, followed by the chips without one by N. The numbers follow the order in which the drivers load and can change
// after a reboot.
func Named(sysfsRoot, name string) []Chip {
	directory := filepath.Join(sysfsRoot, "class", "hwmon")
	listed, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	var chips []Chip
	for _, each := range listed {
		digits, found := strings.CutPrefix(each.Name(), "hwmon")
		number, err := strconv.Atoi(digits)
		if !found || !isDigits(digits) || err != nil {
			continue
		}
		path := filepath.Join(directory, each.Name())
		content, err := os.ReadFile(filepath.Join(path, "name"))
		if err != nil || strings.TrimSuffix(string(content), "\n") != name {
			continue
		}
		device, _ := filepath.EvalSymlinks(filepath.Join(path, "device"))
		chips = append(chips, Chip{Number: number, Path: path, Device: device})
	}
	slices.SortFunc(chips, compare)
	return chips
}

func compare(first, second Chip) int {
	if (first.Device == "") != (second.Device == "") {
		if first.Device == "" {
			return 1
		}
		return -1
	}
	if byDevice := strings.Compare(first.Device, second.Device); byDevice != 0 {
		return byDevice
	}
	return first.Number - second.Number
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
