//go:build linux

package linuxsensors

import (
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/sensors"
)

const hostMeminfo = "MemTotal:       96187828 kB\nMemFree:         1000 kB\nMemAvailable:   73300372 kB\n"

func TestMemoryGivesUsedAndTotal(test *testing.T) {
	tree := newTree(test)
	tree.Meminfo(hostMeminfo)
	got := read(memory{tree.ProcRoot}, everything)
	want := []sensors.Reading{
		{Name: "ram.used", Value: float64(96187828-73300372) / 1048576},
		{Name: "ram.total", Value: float64(96187828) / 1048576},
	}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestMemoryWithoutAvailableGivesTotalOnly(test *testing.T) {
	tree := newTree(test)
	tree.Meminfo("MemTotal:       1048576 kB\n")
	want := []sensors.Reading{{Name: "ram.total", Value: 1}}
	if got := read(memory{tree.ProcRoot}, everything); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestMemoryMalformedOrMissingGivesNothing(test *testing.T) {
	tree := newTree(test)
	tree.Meminfo("MemTotal:       9x kB\nMemAvailable:   5 kB\n")
	if got := read(memory{tree.ProcRoot}, everything); len(got) != 0 {
		test.Errorf("malformed: %v", got)
	}
	if got := read(memory{newTree(test).ProcRoot}, everything); len(got) != 0 {
		test.Errorf("missing: %v", got)
	}
}

func TestMemoryReadsOnlyWantedMetrics(test *testing.T) {
	tree := newTree(test)
	tree.Meminfo(hostMeminfo)
	got := read(memory{tree.ProcRoot}, func(name string) bool { return name == "ram.total" })
	if len(got) != 1 || got[0].Name != "ram.total" {
		test.Errorf("got %v", got)
	}
}

func TestParseMeminfoNeedsThreeFieldsWithKB(test *testing.T) {
	if _, ok := parseMeminfo("MemTotal: 5 MB\n", "MemTotal"); ok {
		test.Error("the unit MB was accepted")
	}
	if _, ok := parseMeminfo("MemTotal: 5\n", "MemTotal"); ok {
		test.Error("a missing unit was accepted")
	}
}
