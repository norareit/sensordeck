//go:build linux

package linuxsensors

import (
	"os"
	"path/filepath"
	"testing"
)

func clockOf(test *testing.T, source cpuClock) (float64, bool) {
	test.Helper()
	got := read(source, everything)
	if len(got) == 0 {
		return 0, false
	}
	if len(got) != 1 || got[0].Name != "cpu.clock" {
		test.Fatalf("got %v", got)
	}
	return got[0].Value, true
}

func TestClockIsTheHighestOfAllCPUs(test *testing.T) {
	tree := newTree(test)
	tree.CPUFrequency(0, "3000000\n")
	tree.CPUFrequency(1, "4425361\n")
	tree.CPUFrequency(2, "1200000\n")
	if got, ok := clockOf(test, cpuClock{tree.SysfsRoot}); !ok || got != 4425.361 {
		test.Errorf("got %v %v", got, ok)
	}
}

func TestClockPassesOverMalformedAndOddlyNamedCPUs(test *testing.T) {
	tree := newTree(test)
	tree.CPUFrequency(0, "3000000\n")
	tree.CPUFrequency(1, "abc\n")
	tree.CPUFrequency(2, "9000000\n")
	tree.CPUFrequency(3, "-5\n")
	odd := filepath.Join(tree.SysfsRoot, "devices", "system", "cpu", "cpu0x", "cpufreq")
	if err := os.MkdirAll(odd, 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(odd, "scaling_cur_freq"), []byte("9999000\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	if got, ok := clockOf(test, cpuClock{tree.SysfsRoot}); !ok || got != 9000 {
		test.Errorf("got %v %v", got, ok)
	}
}

func TestClockWithNoCPUGivesNothing(test *testing.T) {
	if _, ok := clockOf(test, cpuClock{newTree(test).SysfsRoot}); ok {
		test.Error("a reading from no CPU")
	}
}
