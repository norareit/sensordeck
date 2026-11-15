//go:build linux

package commandercore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeEntry(test *testing.T, root, name, uevent string) {
	test.Helper()
	directory := filepath.Join(root, "class", "hidraw", name, "device")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "uevent"), []byte(uevent), 0o644); err != nil {
		test.Fatal(err)
	}
}

func controllerUevent(interfaceNumber string) string {
	return "HID_ID=0003:00001B1C:00000C2A\nHID_PHYS=usb-0000:00:14.0-5/input" + interfaceNumber + "\n"
}

func TestFindNodeWithoutTheDirectory(test *testing.T) {
	if node, found := findNode(test.TempDir(), "/dev"); found {
		test.Fatalf("found %s", node)
	}
}

func TestFindNodePassesOverTheScreen(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", "HID_ID=0003:00000483:00000065\nHID_PHYS=usb-0000:00:14.0-3/input0\n")
	if node, found := findNode(root, "/dev"); found {
		test.Fatalf("found %s", node)
	}
}

func TestFindNodeNeverGivesInterfaceOne(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw10", controllerUevent("1"))
	if node, found := findNode(root, "/dev"); found {
		test.Fatalf("found %s", node)
	}
}

func TestFindNodePassesOverAMalformedID(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", "HID_ID=0003:1B1C\nHID_PHYS=usb/input0\n")
	writeEntry(test, root, "hidraw1", "HID_ID=0001:00001B1C:00000C2A\nHID_PHYS=usb/input0\n")
	if node, found := findNode(root, "/dev"); found {
		test.Fatalf("found %s", node)
	}
}

func TestFindNodeGivesTheInterfaceZeroNode(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw9", controllerUevent("0"))
	writeEntry(test, root, "hidraw10", controllerUevent("1"))
	if node, found := findNode(root, "/devroot"); !found || node != "/devroot/hidraw9" {
		test.Fatalf("got %q, %v", node, found)
	}
}

func TestFindNodeGivesTheLowestNumber(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw7", controllerUevent("0"))
	writeEntry(test, root, "hidraw3", controllerUevent("0"))
	if node, found := findNode(root, "/dev"); !found || node != "/dev/hidraw3" {
		test.Fatalf("got %q, %v", node, found)
	}
}

func TestEncodeGivesTheReportIDTheCommandAndZeros(test *testing.T) {
	for position, command := range steps {
		written := encode(command)
		if len(written) != 97 || written[0] != 0x00 || !bytes.Equal(written[1:1+len(command)], command) {
			test.Fatalf("step %d: % x", position+1, written)
		}
		if !bytes.Equal(written[1+len(command):], make([]byte, 96-len(command))) {
			test.Fatalf("step %d: the bytes after the command are not zero", position+1)
		}
	}
}
