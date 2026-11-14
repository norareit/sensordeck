//go:build linux

package linuxsensors

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/fakesensors"
	"github.com/norareit/sensordeck/internal/sensors"
)

func everything(string) bool { return true }

func read(source sensors.Source, wanted func(string) bool) []sensors.Reading {
	return source.Read(context.Background(), wanted)
}

func newTree(test *testing.T) *fakesensors.Tree {
	test.Helper()
	return fakesensors.New(test, filepath.Join(test.TempDir(), "sys"))
}

func TestNoHwmonDirectoryGivesNothing(test *testing.T) {
	tree := newTree(test)
	for _, source := range []sensors.Source{cpuTemperatures{tree.SysfsRoot}, memoryModules(tree.SysfsRoot), drives(tree.SysfsRoot)} {
		if got := read(source, everything); len(got) != 0 {
			test.Errorf("%T gave %v", source, got)
		}
	}
}

func TestK10tempGivesControlAndCCDs(test *testing.T) {
	tree := newTree(test)
	tree.Hwmon(2, "k10temp", "pci0000:00/0000:00:18.3",
		fakesensors.Sensor{Number: 1, Label: "Tctl", Input: "51375"},
		fakesensors.Sensor{Number: 3, Label: "Tccd1", Input: "40000"},
		fakesensors.Sensor{Number: 4, Label: "Tccd2", Input: "-1500"})
	want := []sensors.Reading{{Name: "cpu.temp", Value: 51.375}, {Name: "cpu.ccd1.temp", Value: 40}, {Name: "cpu.ccd2.temp", Value: -1.5}}
	if got := read(cpuTemperatures{tree.SysfsRoot}, everything); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestK10tempMalformedInputMissingLabelAndOtherDriver(test *testing.T) {
	tree := newTree(test)
	tree.Hwmon(1, "amdgpu", "pci0000:00/a", fakesensors.Sensor{Number: 1, Label: "edge", Input: "30000"})
	tree.Hwmon(2, "k10temp", "pci0000:00/b",
		fakesensors.Sensor{Number: 1, Label: "Tctl", Input: "4x000"},
		fakesensors.Sensor{Number: 2, Input: "40000"},
		fakesensors.Sensor{Number: 3, Label: "Tccd1", Input: "+5"},
		fakesensors.Sensor{Number: 4, Label: "Tccd2", Input: "38125"})
	want := []sensors.Reading{{Name: "cpu.ccd2.temp", Value: 38.125}}
	if got := read(cpuTemperatures{tree.SysfsRoot}, everything); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestK10tempReadsOnlyWantedMetrics(test *testing.T) {
	tree := newTree(test)
	tree.Hwmon(2, "k10temp", "pci0000:00/b",
		fakesensors.Sensor{Number: 1, Label: "Tctl", Input: "51000"},
		fakesensors.Sensor{Number: 3, Label: "Tccd1", Input: "40000"})
	got := read(cpuTemperatures{tree.SysfsRoot}, func(name string) bool { return name == "cpu.ccd1.temp" })
	if want := []sensors.Reading{{Name: "cpu.ccd1.temp", Value: 40}}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestNumberedEntriesFollowTheirDevicePathsNotTheirNumbers(test *testing.T) {
	tree := newTree(test)
	tree.Hwmon(5, "spd5118", "pci0000:00/0000:00:14.0/i2c-1/1-0051", fakesensors.Sensor{Number: 1, Input: "40000"})
	tree.Hwmon(3, "spd5118", "pci0000:00/0000:00:14.0/i2c-1/1-0053", fakesensors.Sensor{Number: 1, Input: "37875"})
	tree.Hwmon(7, "nvme", "pci0000:00/0000:00:02.1/0000:06:00.0/nvme/nvme0", fakesensors.Sensor{Number: 1, Label: "Composite", Input: "38800"})
	tree.Hwmon(2, "nvme", "pci0000:00/0000:00:01.2/0000:02:00.0/nvme/nvme1", fakesensors.Sensor{Number: 1, Label: "Composite", Input: "33900"})
	wantRAM := []sensors.Reading{{Name: "ram.1.temp", Value: 40}, {Name: "ram.2.temp", Value: 37.875}}
	if got := read(memoryModules(tree.SysfsRoot), everything); !reflect.DeepEqual(got, wantRAM) {
		test.Errorf("ram: %v", got)
	}
	wantDrives := []sensors.Reading{{Name: "nvme.1.temp", Value: 33.9}, {Name: "nvme.2.temp", Value: 38.8}}
	if got := read(drives(tree.SysfsRoot), everything); !reflect.DeepEqual(got, wantDrives) {
		test.Errorf("nvme: %v", got)
	}
	got := read(memoryModules(tree.SysfsRoot), func(name string) bool { return name == "ram.2.temp" })
	if want := []sensors.Reading{{Name: "ram.2.temp", Value: 37.875}}; !reflect.DeepEqual(got, want) {
		test.Errorf("wanted only ram.2.temp: %v", got)
	}
}

func TestDriveWithoutCompositeKeepsItsNumber(test *testing.T) {
	tree := newTree(test)
	tree.Hwmon(1, "nvme", "pci0000:00/a", fakesensors.Sensor{Number: 2, Label: "Sensor 1", Input: "30000"})
	tree.Hwmon(2, "nvme", "pci0000:00/b", fakesensors.Sensor{Number: 1, Label: "Composite", Input: "35000"})
	want := []sensors.Reading{{Name: "nvme.2.temp", Value: 35}}
	if got := read(drives(tree.SysfsRoot), everything); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestEntryWithoutDeviceLinkIsPassedOverAndNotCounted(test *testing.T) {
	tree := newTree(test)
	tree.Hwmon(1, "spd5118", "", fakesensors.Sensor{Number: 1, Input: "30000"})
	tree.Hwmon(2, "spd5118", "pci0000:00/b", fakesensors.Sensor{Number: 1, Input: "35000"})
	want := []sensors.Reading{{Name: "ram.1.temp", Value: 35}}
	if got := read(memoryModules(tree.SysfsRoot), everything); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestEqualDevicePathsAreOrderedByHwmonNumber(test *testing.T) {
	tree := newTree(test)
	first := tree.Hwmon(10, "spd5118", "pci0000:00/same", fakesensors.Sensor{Number: 1, Input: "10000"})
	second := tree.Hwmon(9, "spd5118", "", fakesensors.Sensor{Number: 1, Input: "9000"})
	if err := os.Symlink(filepath.Join(first, "device"), filepath.Join(second, "device")); err != nil {
		test.Fatal(err)
	}
	want := []sensors.Reading{{Name: "ram.1.temp", Value: 9}, {Name: "ram.2.temp", Value: 10}}
	if got := read(memoryModules(tree.SysfsRoot), everything); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestParseMillidegrees(test *testing.T) {
	good := map[string]float64{"51375\n": 51.375, "-1500\n": -1.5, "40000": 40}
	for content, want := range good {
		if got, ok := parseMillidegrees(content); !ok || got != want {
			test.Errorf("%q: %v %v", content, got, ok)
		}
	}
	for _, content := range []string{"+5", "", "4.5", "4x000", "1 2", "\n", "-"} {
		if _, ok := parseMillidegrees(content); ok {
			test.Errorf("%q was accepted", content)
		}
	}
}
