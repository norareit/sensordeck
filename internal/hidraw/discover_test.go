//go:build linux

package hidraw

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const screenUevent = "DRIVER=hid-generic\nHID_ID=0003:00000483:00000065\nHID_NAME=llhmi.com llTechCo.,Ltd\n" +
	"HID_PHYS=usb-0000:00:14.0-3/input0\nHID_UNIQ=\nMODALIAS=hid:b0003g0001v00000483p00000065\n"

// writeEntry makes class/hidraw/<name> under root, with a uevent file unless the content is nil.
func writeEntry(test *testing.T, root, name string, uevent *string) {
	test.Helper()
	directory := filepath.Join(root, "class", "hidraw", name, "device")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	if uevent == nil {
		return
	}
	if err := os.WriteFile(filepath.Join(directory, "uevent"), []byte(*uevent), 0o644); err != nil {
		test.Fatal(err)
	}
}

func text(content string) *string { return &content }

// discover runs the discovery on a tree in which no node exists under the dev root, so that a discovery that opened
// a node would fail.
func discover(test *testing.T, sysfsRoot string) ([]Screen, error) {
	test.Helper()
	return Discover(sysfsRoot, "/dev")
}

func TestDiscoverWithoutTheDirectory(test *testing.T) {
	screens, err := discover(test, test.TempDir())
	if err != nil || len(screens) != 0 {
		test.Fatalf("got %v, %v", screens, err)
	}
}

func TestDiscoverWithNoEntries(test *testing.T) {
	root := test.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "class", "hidraw"), 0o755); err != nil {
		test.Fatal(err)
	}
	screens, err := discover(test, root)
	if err != nil || len(screens) != 0 {
		test.Fatalf("got %v, %v", screens, err)
	}
}

func TestDiscoverPassesOverOtherDevices(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", text("HID_ID=0003:0000046D:0000C52B\nHID_NAME=Logitech\n"))
	writeEntry(test, root, "hidraw1", text("HID_ID=0003:00000483:00000066\n"))
	writeEntry(test, root, "hidraw4", text("HID_ID=0003:00000484:00000065\n"))
	screens, err := discover(test, root)
	if err != nil || len(screens) != 0 {
		test.Fatalf("got %v, %v", screens, err)
	}
}

func TestDiscoverOneScreen(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw3", text("DRIVER=hid-generic\nHID_ID=0003:00000483:00000065\nHID_NAME=llhmi.com llTechCo.,Ltd\nHID_UNIQ=ABC123\n"))
	screens, err := Discover(root, "/devroot")
	if err != nil {
		test.Fatal(err)
	}
	want := []Screen{{Node: "/devroot/hidraw3", Vendor: 0x0483, Product: 0x0065, Name: "llhmi.com llTechCo.,Ltd", Serial: "ABC123"}}
	if !reflect.DeepEqual(screens, want) {
		test.Fatalf("got %+v, want %+v", screens, want)
	}
}

func TestDiscoverOrdersByTheNumber(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw10", text(screenUevent))
	writeEntry(test, root, "hidraw2", text(screenUevent))
	screens, err := discover(test, root)
	if err != nil {
		test.Fatal(err)
	}
	if len(screens) != 2 || screens[0].Node != "/dev/hidraw2" || screens[1].Node != "/dev/hidraw10" {
		test.Fatalf("got %+v", screens)
	}
}

func TestDiscoverReadsLowerCaseHexadecimal(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", text("HID_ID=0003:0000a1b2:0000c3d4\n"))
	writeEntry(test, root, "hidraw1", text("HID_ID=0003:00000483:00000065\n"))
	screens, err := discover(test, root)
	if err != nil || len(screens) != 1 || screens[0].Node != "/dev/hidraw1" {
		test.Fatalf("got %v, %v", screens, err)
	}
}

func TestDiscoverWithoutNameAndWithEmptySerial(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", text("HID_ID=0003:00000483:00000065\nHID_UNIQ=\n"))
	screens, err := discover(test, root)
	if err != nil || len(screens) != 1 {
		test.Fatalf("got %v, %v", screens, err)
	}
	if screens[0].Name != "" || screens[0].Serial != "" {
		test.Fatalf("got %+v", screens[0])
	}
}

func TestDiscoverPassesOverBrokenEntries(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", nil)
	writeEntry(test, root, "hidraw1", text("HID_ID=0003:00000483\n"))
	writeEntry(test, root, "hidraw2", text("HID_ID=0003:0000048G:00000065\n"))
	writeEntry(test, root, "hidraw3", text("HID_ID=0003:00000483:00000065:0\n"))
	writeEntry(test, root, "hidraw4", text("NOTHING\n"))
	writeEntry(test, root, "hidraw5", text("HID_ID=0005:00000483:00000065\n"))
	writeEntry(test, root, "hidraw", text(screenUevent))
	writeEntry(test, root, "hidrawx1", text(screenUevent))
	writeEntry(test, root, "hidraw1x", text(screenUevent))
	screens, err := discover(test, root)
	if err != nil || len(screens) != 0 {
		test.Fatalf("got %v, %v", screens, err)
	}
}

func TestDiscoverReturnsOtherErrorsOfReadingTheDirectory(test *testing.T) {
	root := test.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "class"), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "class", "hidraw"), nil, 0o644); err != nil {
		test.Fatal(err)
	}
	if _, err := discover(test, root); err == nil {
		test.Fatal("no error for a regular file")
	}
}

func TestScreenLine(test *testing.T) {
	spec := Screen{Node: "/dev/hidraw3", Vendor: 0x0483, Product: 0x0065, Name: "llhmi.com llTechCo.,Ltd"}
	if got, want := spec.Line(), "/dev/hidraw3\t0483:0065\tllhmi.com llTechCo.,Ltd\t-"; got != want {
		test.Errorf("got %q, want %q", got, want)
	}
	withSerial := Screen{Node: "/dev/hidraw10", Vendor: 0x0483, Product: 0x0065, Serial: "S1"}
	if got, want := withSerial.Line(), "/dev/hidraw10\t0483:0065\t-\tS1"; got != want {
		test.Errorf("got %q, want %q", got, want)
	}
}

func TestListGivesTheFieldsOfAnEntry(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw4", text("HID_ID=0003:00001B1C:00000C2A\nHID_NAME=Corsair\n"+
		"HID_PHYS=usb-0000:00:14.0-5/input0\nHID_UNIQ=ABC\n"))
	entries, err := List(root, "/dev")
	want := []Entry{{Number: 4, Node: "/dev/hidraw4", Bus: 3, Vendor: 0x1B1C, Product: 0x0C2A, ValidID: true,
		Name: "Corsair", Phys: "usb-0000:00:14.0-5/input0", Serial: "ABC"}}
	if err != nil || !reflect.DeepEqual(entries, want) {
		test.Fatalf("got %+v, %v", entries, err)
	}
}

func TestListKeepsAnEntryWithAMalformedID(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", text("HID_ID=zz\nHID_NAME=Odd\n"))
	entries, err := List(root, "/dev")
	if err != nil || len(entries) != 1 || entries[0].ValidID || entries[0].Name != "Odd" {
		test.Fatalf("got %+v, %v", entries, err)
	}
}

func TestListLeavesOutAnEntryWithoutUevent(test *testing.T) {
	root := test.TempDir()
	writeEntry(test, root, "hidraw0", nil)
	entries, err := List(root, "/dev")
	if err != nil || len(entries) != 0 {
		test.Fatalf("got %+v, %v", entries, err)
	}
}

func TestListOrdersByNumber(test *testing.T) {
	root := test.TempDir()
	for _, name := range []string{"hidraw10", "hidraw2", "hidraw1"} {
		writeEntry(test, root, name, text("HID_ID=0003:00000001:00000002\n"))
	}
	entries, err := List(root, "/dev")
	if err != nil || len(entries) != 3 || entries[0].Number != 1 || entries[1].Number != 2 || entries[2].Number != 10 {
		test.Fatalf("got %+v, %v", entries, err)
	}
}

func TestListWithoutTheDirectory(test *testing.T) {
	entries, err := List(test.TempDir(), "/dev")
	if err != nil || len(entries) != 0 {
		test.Fatalf("got %+v, %v", entries, err)
	}
}
