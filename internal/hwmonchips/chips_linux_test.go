//go:build linux

package hwmonchips

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/norareit/sensordeck/internal/fakesensors"
)

func numbers(chips []Chip) []int {
	var found []int
	for _, chip := range chips {
		found = append(found, chip.Number)
	}
	return found
}

func TestNamedOrdersByDevicePathAndNotByNumber(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	tree.Hwmon(1, "amdgpu", "pci0000:00/0000:0b:00.0")
	tree.Hwmon(2, "amdgpu", "pci0000:00/0000:03:00.0")
	got := numbers(Named(tree.SysfsRoot, "amdgpu"))
	if want := []int{2, 1}; !slices.Equal(got, want) {
		test.Fatalf("order = %v, want %v", got, want)
	}
}

func TestNamedOrdersChipsWithOneDevicePathByNumber(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	tree.Hwmon(5, "amdgpu", "same")
	tree.Hwmon(3, "amdgpu", "same")
	got := numbers(Named(tree.SysfsRoot, "amdgpu"))
	if want := []int{3, 5}; !slices.Equal(got, want) {
		test.Fatalf("order = %v, want %v", got, want)
	}
}

func TestNamedPutsChipsWithoutDeviceLastByNumber(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	tree.Hwmon(1, "amdgpu", "")
	tree.Hwmon(2, "amdgpu", "z")
	tree.Hwmon(0, "amdgpu", "")
	got := Named(tree.SysfsRoot, "amdgpu")
	if want := []int{2, 0, 1}; !slices.Equal(numbers(got), want) {
		test.Fatalf("order = %v, want %v", numbers(got), want)
	}
	if got[1].Device != "" || got[0].Device == "" {
		test.Fatalf("devices = %q and %q", got[0].Device, got[1].Device)
	}
}

func TestNamedLeavesOutOtherNamesAndOtherDirectories(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	tree.Hwmon(1, "amdgpu", "a")
	tree.Hwmon(2, "nvme", "b")
	other := filepath.Join(tree.SysfsRoot, "class", "hwmon", "hwmonX")
	if err := os.MkdirAll(other, 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "name"), []byte("amdgpu\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	got := numbers(Named(tree.SysfsRoot, "amdgpu"))
	if want := []int{1}; !slices.Equal(got, want) {
		test.Fatalf("found %v, want %v", got, want)
	}
}

func TestNamedGivesNothingWithoutClassHwmon(test *testing.T) {
	if got := Named(test.TempDir(), "amdgpu"); got != nil {
		test.Fatalf("found %v", got)
	}
}
