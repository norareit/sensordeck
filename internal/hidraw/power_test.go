//go:build linux

package hidraw

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakeusb"
)

func screenWithPower(test *testing.T) (*fakeusb.Device, string, string) {
	test.Helper()
	root := test.TempDir()
	sysfsRoot, devRoot := filepath.Join(root, "sys"), filepath.Join(root, "dev")
	return fakeusb.New(test, sysfsRoot, devRoot, 3), sysfsRoot, devRoot
}

func TestUSBDeviceOfADiscoveredNode(test *testing.T) {
	_, sysfsRoot, devRoot := screenWithPower(test)
	screens, err := Discover(sysfsRoot, devRoot)
	if err != nil || len(screens) != 1 {
		test.Fatalf("got %v, %v", screens, err)
	}
	usbDevice, found := USBDevice(sysfsRoot, devRoot, screens[0].Node)
	if !found || filepath.Base(usbDevice) != "1-4" {
		test.Fatalf("got %q, %v", usbDevice, found)
	}
	if _, err := os.Stat(filepath.Join(usbDevice, "power", "control")); err != nil {
		test.Fatal(err)
	}
}

func TestUSBDeviceOfANodeGivenByALink(test *testing.T) {
	device, sysfsRoot, devRoot := screenWithPower(test)
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
	_, sysfsRoot, devRoot := screenWithPower(test)
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

func TestRecovery(test *testing.T) {
	cases := []struct {
		name     string
		prepare  func(*fakeusb.Device)
		delay    time.Duration
		possible bool
	}{
		{"auto and 2000", func(*fakeusb.Device) {}, 2 * time.Second, true},
		{"control on", func(device *fakeusb.Device) { device.Set("control", "on\n") }, 0, false},
		{"delay -1", func(device *fakeusb.Device) { device.Set("autosuspend_delay_ms", "-1\n") }, 0, false},
		{"no control", func(device *fakeusb.Device) { device.Remove("control") }, 0, false},
		{"no delay", func(device *fakeusb.Device) { device.Remove("autosuspend_delay_ms") }, 0, false},
		{"delay 0", func(device *fakeusb.Device) { device.Set("autosuspend_delay_ms", "0\n") }, 0, true},
		{"delay not a number", func(device *fakeusb.Device) { device.Set("autosuspend_delay_ms", "soon\n") }, 0, false},
	}
	for _, testCase := range cases {
		test.Run(testCase.name, func(test *testing.T) {
			device, sysfsRoot, devRoot := screenWithPower(test)
			testCase.prepare(device)
			usbDevice, found := USBDevice(sysfsRoot, devRoot, device.Node)
			if !found {
				test.Fatal("no USB device")
			}
			delay, possible := Recovery(usbDevice)
			if delay != testCase.delay || possible != testCase.possible {
				test.Fatalf("got %v, %v", delay, possible)
			}
		})
	}
}

func TestSuspended(test *testing.T) {
	device, sysfsRoot, devRoot := screenWithPower(test)
	usbDevice, _ := USBDevice(sysfsRoot, devRoot, device.Node)
	if Suspended(usbDevice) {
		test.Fatal("active read as suspended")
	}
	device.Set("runtime_status", "suspended\n")
	if !Suspended(usbDevice) {
		test.Fatal("suspended not read")
	}
	device.Remove("runtime_status")
	if Suspended(usbDevice) {
		test.Fatal("a missing file read as suspended")
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
