package custommetrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/fakesensors"
)

func TestAFileMetric(test *testing.T) {
	path := writeFile(test, filepath.Join(test.TempDir(), "t"), "42000\n")
	got := readFile(config.Metric{File: path, Scale: 0.001})
	if got.failed() || got.value != 42 {
		test.Errorf("got %+v", got)
	}
}

func TestAFileMetricThatCannotBeRead(test *testing.T) {
	path := filepath.Join(test.TempDir(), "missing")
	got := readFile(config.Metric{File: path, Scale: 1})
	if want := "cannot read " + path + ": no such file or directory"; got.cause != want {
		test.Errorf("cause %q, want %q", got.cause, want)
	}
}

func TestAFileMetricWithNoNumber(test *testing.T) {
	path := writeFile(test, filepath.Join(test.TempDir(), "t"), "n/a\n")
	if got := readFile(config.Metric{File: path, Scale: 1}); got.cause != "no number in "+path {
		test.Errorf("cause %q", got.cause)
	}
}

func hwmonDefinition(index int) config.Metric {
	return config.Metric{Hwmon: "amdgpu", Input: "fan1_input", Index: index, Scale: 1}
}

func TestAHwmonMetricFollowsTheChipWhenItsNumberChanges(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	entry := tree.Hwmon(3, "nct6799", "pci/a")
	writeFile(test, filepath.Join(entry, "fan2_input"), "1200\n")
	definition := config.Metric{Hwmon: "nct6799", Input: "fan2_input", Index: 1, Scale: 1}
	if got := readHwmon(tree.SysfsRoot, definition); got.value != 1200 {
		test.Fatalf("got %+v", got)
	}
	if err := os.Rename(entry, filepath.Join(filepath.Dir(entry), "hwmon7")); err != nil {
		test.Fatal(err)
	}
	if got := readHwmon(tree.SysfsRoot, definition); got.value != 1200 {
		test.Errorf("after the change: %+v", got)
	}
}

func twoGPUs(test *testing.T) *fakesensors.Tree {
	tree := fakesensors.New(test, test.TempDir())
	// The numbers are in the opposite order of the device paths.
	writeFile(test, filepath.Join(tree.Hwmon(1, "amdgpu", "pci/0b"), "fan1_input"), "2000\n")
	writeFile(test, filepath.Join(tree.Hwmon(2, "amdgpu", "pci/03"), "fan1_input"), "1000\n")
	return tree
}

func TestHwmonIndexFollowsTheOrderOfTheDevicePaths(test *testing.T) {
	tree := twoGPUs(test)
	for index, want := range map[int]float64{1: 1000, 2: 2000} {
		if got := readHwmon(tree.SysfsRoot, hwmonDefinition(index)); got.value != want || got.failed() {
			test.Errorf("index %d: %+v, want %v", index, got, want)
		}
	}
}

func TestHwmonIndexBeyondTheChips(test *testing.T) {
	cause := readHwmon(twoGPUs(test).SysfsRoot, hwmonDefinition(3)).cause
	if cause != "2 hwmon chips named amdgpu, fewer than index 3" {
		test.Errorf("cause %q", cause)
	}
	cause = readHwmon(test.TempDir(), hwmonDefinition(3)).cause
	if cause != "0 hwmon chips named amdgpu, fewer than index 3" {
		test.Errorf("cause %q", cause)
	}
}

func TestAChipWithoutADeviceComesAfterOneWithIt(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	writeFile(test, filepath.Join(tree.Hwmon(1, "amdgpu", ""), "fan1_input"), "1\n")
	writeFile(test, filepath.Join(tree.Hwmon(2, "amdgpu", "pci/z"), "fan1_input"), "2\n")
	if got := readHwmon(tree.SysfsRoot, hwmonDefinition(1)); got.value != 2 {
		test.Errorf("first: %+v", got)
	}
	if got := readHwmon(tree.SysfsRoot, hwmonDefinition(2)); got.value != 1 {
		test.Errorf("second: %+v", got)
	}
}

func TestAHwmonInputThatIsMissing(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	entry := tree.Hwmon(1, "amdgpu", "pci/a")
	want := "cannot read " + filepath.Join(entry, "fan1_input") + ": no such file or directory"
	if got := readHwmon(tree.SysfsRoot, hwmonDefinition(1)); !strings.HasPrefix(got.cause, want) {
		test.Errorf("cause %q, want %q", got.cause, want)
	}
}
