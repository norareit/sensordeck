//go:build linux

package hidraw

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/norareit/sensordeck/internal/fakeusb"
)

func usbScreen(test *testing.T) (*fakeusb.Device, string, string) {
	test.Helper()
	root := test.TempDir()
	sysfsRoot, devRoot := filepath.Join(root, "sys"), filepath.Join(root, "dev")
	return fakeusb.New(test, sysfsRoot, devRoot, 3), sysfsRoot, devRoot
}

func TestUSBDeviceOfADiscoveredNode(test *testing.T) {
	_, sysfsRoot, devRoot := usbScreen(test)
	screens, err := Discover(sysfsRoot, devRoot)
	if err != nil || len(screens) != 1 {
		test.Fatalf("got %v, %v", screens, err)
	}
	usbDevice, found := USBDevice(sysfsRoot, devRoot, screens[0].Node)
	if !found || filepath.Base(usbDevice) != "1-4" {
		test.Fatalf("got %q, %v", usbDevice, found)
	}
}

func TestUSBDeviceOfANodeGivenByALink(test *testing.T) {
	device, sysfsRoot, devRoot := usbScreen(test)
	link := filepath.Join(test.TempDir(), "screen")
	if err := os.Symlink(device.Node, link); err != nil {
		test.Fatal(err)
	}
	usbDevice, found := USBDevice(sysfsRoot, devRoot, link)
	if !found || filepath.Base(usbDevice) != "1-4" {
		test.Fatalf("got %q, %v", usbDevice, found)
	}
}

func TestUSBDeviceIsNotFoundForANodeWithNoEntry(test *testing.T) {
	_, sysfsRoot, devRoot := usbScreen(test)
	other := filepath.Join(devRoot, "hidraw9")
	if err := os.WriteFile(other, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	if usbDevice, found := USBDevice(sysfsRoot, devRoot, other); found {
		test.Fatalf("got %q", usbDevice)
	}
	if usbDevice, found := USBDevice(sysfsRoot, devRoot, filepath.Join(devRoot, "missing")); found {
		test.Fatalf("got %q", usbDevice)
	}
}

func TestUSBDeviceIsNotFoundForAFlatEntry(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", text(screenUevent))
	devRoot := filepath.Join(root, "dev")
	if err := os.MkdirAll(devRoot, 0o755); err != nil {
		test.Fatal(err)
	}
	node := filepath.Join(devRoot, "hidraw0")
	if err := os.WriteFile(node, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	if usbDevice, found := USBDevice(root, devRoot, node); found {
		test.Fatalf("got %q", usbDevice)
	}
}

func TestTimedOut(test *testing.T) {
	timeout := &os.PathError{Op: "write", Path: "/dev/hidraw3", Err: syscall.ETIMEDOUT}
	failure := &os.PathError{Op: "write", Path: "/dev/hidraw3", Err: syscall.EIO}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"timeout", timeout, true},
		{"wrapped timeout", fmt.Errorf("write failed: %w", timeout), true},
		{"other failure", failure, false},
		{"wrapped other failure", fmt.Errorf("write failed: %w", failure), false},
		{"nil", nil, false},
	}
	for _, testCase := range cases {
		if got := TimedOut(testCase.err); got != testCase.want {
			test.Errorf("%s: got %v", testCase.name, got)
		}
	}
}
