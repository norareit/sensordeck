//go:build linux

package commandercore

import (
	"strings"

	"github.com/norareit/sensordeck/internal/hidraw"
)

const (
	busUSB  = 3
	vendor  = 0x1B1C
	product = 0x0C2A
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
