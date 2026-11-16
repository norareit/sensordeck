//go:build linux

// Package fakeusb is test support: a sysfs tree laid out as the kernel lays out a USB screen, with the hidraw entry
// under the HID device of an interface of the USB device, as internal/fakescreen is test support for the flat tree.
// The node is a regular file, since the tests that use it never write to the screen through it.
package fakeusb

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

const (
	hidUevent = "DRIVER=hid-generic\nHID_ID=0003:00000483:00000065\nHID_NAME=llhmi.com llTechCo.,Ltd\n" +
		"HID_PHYS=usb-0000:00:14.0-4/input0\nHID_UNIQ=\n"
	interfaceUevent = "DEVTYPE=usb_interface\nDRIVER=usbhid\n"
	deviceUevent    = "DEVTYPE=usb_device\nDRIVER=usb\nPRODUCT=483/65/100\n"
)

// Device is a USB screen in a sysfs tree.
type Device struct {
	Node string // <dev root>/hidraw<n>

	powerDirectory string
	usbDevice      string
	sysfsRoot      string
	hidrawName     string
	test           testing.TB
}

// New builds the tree under the roots with busnum 1 and devnum 5 as the kernel writes them, power/control at auto,
// power/autosuspend_delay_ms at 2000 and power/runtime_status at active, and makes the node.
func New(test testing.TB, sysfsRoot, devRoot string, number int) *Device {
	test.Helper()
	name := "hidraw" + strconv.Itoa(number)
	usbDevice := filepath.Join(sysfsRoot, "devices", "pci0000:00", "0000:00:14.0", "usb1", "1-4")
	usbInterface := filepath.Join(usbDevice, "1-4:1.0")
	hidDevice := filepath.Join(usbInterface, "0003:0483:0065.0001")
	entry := filepath.Join(hidDevice, "hidraw", name)
	device := &Device{
		Node:           filepath.Join(devRoot, name),
		powerDirectory: filepath.Join(usbDevice, "power"),
		usbDevice:      usbDevice,
		sysfsRoot:      sysfsRoot,
		hidrawName:     name,
		test:           test,
	}

	for _, directory := range []string{entry, device.powerDirectory, filepath.Join(sysfsRoot, "class", "hidraw"), devRoot} {
		device.must(os.MkdirAll(directory, 0o755))
	}
	device.must(os.WriteFile(filepath.Join(hidDevice, "uevent"), []byte(hidUevent), 0o644))
	device.must(os.WriteFile(filepath.Join(usbInterface, "uevent"), []byte(interfaceUevent), 0o644))
	device.must(os.WriteFile(filepath.Join(usbDevice, "uevent"), []byte(deviceUevent), 0o644))
	device.must(os.Symlink("../..", filepath.Join(entry, "device")))
	device.must(os.Symlink(entry, filepath.Join(sysfsRoot, "class", "hidraw", name)))
	device.must(os.WriteFile(filepath.Join(usbDevice, "busnum"), []byte("1\n"), 0o644))
	device.must(os.WriteFile(filepath.Join(usbDevice, "devnum"), []byte("5\n"), 0o644))
	device.must(os.WriteFile(device.Node, nil, 0o644))
	device.Set("control", "auto\n")
	device.Set("autosuspend_delay_ms", "2000\n")
	device.Set("runtime_status", "active\n")
	return device
}

// Set writes a file of power/ of the USB device, as the kernel would hold it.
func (device *Device) Set(name, content string) {
	device.test.Helper()
	device.must(os.WriteFile(filepath.Join(device.powerDirectory, name), []byte(content), 0o644))
}

// Remove deletes a file of power/ of the USB device.
func (device *Device) Remove(name string) {
	device.test.Helper()
	device.must(os.Remove(filepath.Join(device.powerDirectory, name)))
}

// SetDevnum rewrites devnum of the USB device with the content, as the kernel holds it after the device enumerated
// again.
func (device *Device) SetDevnum(content string) {
	device.test.Helper()
	device.must(os.WriteFile(filepath.Join(device.usbDevice, "devnum"), []byte(content), 0o644))
}

// Unplug removes the USB device from the tree, the entry of the hidraw node and the node, as the kernel does when the
// device leaves the bus.
func (device *Device) Unplug() {
	device.test.Helper()
	device.must(os.RemoveAll(filepath.Join(device.sysfsRoot, "devices", "pci0000:00")))
	device.must(os.Remove(filepath.Join(device.sysfsRoot, "class", "hidraw", device.hidrawName)))
	device.must(os.Remove(device.Node))
}

func (device *Device) must(err error) {
	device.test.Helper()
	if err != nil {
		device.test.Fatal(err)
	}
}
