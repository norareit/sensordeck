//go:build linux

// Package fakesensors is test support: hwmon, cpufreq and proc trees in a temporary directory, and a fake nvidia-smi.
// It is shared by the tests of the Linux sources and the end-to-end tests.
package fakesensors

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Tree holds the two roots that the sources read.
type Tree struct {
	SysfsRoot string // given, so that the end-to-end tests can share the root of the fake screen
	ProcRoot  string // <temp>/proc

	test testing.TB
}

// New makes a tree whose sysfs root is the given one and whose proc root is in a temporary directory of the test.
func New(test testing.TB, sysfsRoot string) *Tree {
	test.Helper()
	tree := &Tree{SysfsRoot: sysfsRoot, ProcRoot: filepath.Join(test.TempDir(), "proc"), test: test}
	tree.mkdir(tree.SysfsRoot)
	tree.mkdir(tree.ProcRoot)
	return tree
}

// Sensor is temp<Number>_input with Input and, when Label is not empty, temp<Number>_label with Label. Both are
// written with a trailing newline.
type Sensor struct {
	Number       int
	Label, Input string
}

// Hwmon makes <SysfsRoot>/class/hwmon/hwmon<number> with name and the sensors. With device not empty,
// <entry>/device is a link to <SysfsRoot>/devices/<device>, which it makes. It gives the path of the entry.
func (tree *Tree) Hwmon(number int, name, device string, sensors ...Sensor) string {
	tree.test.Helper()
	entry := filepath.Join(tree.SysfsRoot, "class", "hwmon", fmt.Sprintf("hwmon%d", number))
	tree.mkdir(entry)
	tree.write(filepath.Join(entry, "name"), name+"\n")
	for _, sensor := range sensors {
		prefix := filepath.Join(entry, fmt.Sprintf("temp%d_", sensor.Number))
		tree.write(prefix+"input", sensor.Input+"\n")
		if sensor.Label != "" {
			tree.write(prefix+"label", sensor.Label+"\n")
		}
	}
	if device != "" {
		target := filepath.Join(tree.SysfsRoot, "devices", device)
		tree.mkdir(target)
		if err := os.Symlink(target, filepath.Join(entry, "device")); err != nil {
			tree.test.Fatalf("fake sensors: %v", err)
		}
	}
	return entry
}

// CPUFrequency writes devices/system/cpu/cpu<cpu>/cpufreq/scaling_cur_freq with the content as given.
func (tree *Tree) CPUFrequency(cpu int, content string) {
	tree.test.Helper()
	directory := filepath.Join(tree.SysfsRoot, "devices", "system", "cpu", fmt.Sprintf("cpu%d", cpu), "cpufreq")
	tree.mkdir(directory)
	tree.write(filepath.Join(directory, "scaling_cur_freq"), content)
}

// Stat writes <ProcRoot>/stat as given.
func (tree *Tree) Stat(content string) {
	tree.test.Helper()
	tree.write(filepath.Join(tree.ProcRoot, "stat"), content)
}

// Meminfo writes <ProcRoot>/meminfo as given.
func (tree *Tree) Meminfo(content string) {
	tree.test.Helper()
	tree.write(filepath.Join(tree.ProcRoot, "meminfo"), content)
}

// NvidiaSMI makes a new temporary directory with an executable nvidia-smi, "#!/bin/sh\n" and the script, and gives
// the directory, for the PATH of a test.
func NvidiaSMI(test testing.TB, script string) string {
	test.Helper()
	directory := test.TempDir()
	path := filepath.Join(directory, "nvidia-smi")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		test.Fatalf("fake sensors: %v", err)
	}
	return directory
}

func (tree *Tree) mkdir(path string) {
	tree.test.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		tree.test.Fatalf("fake sensors: %v", err)
	}
}

func (tree *Tree) write(path, content string) {
	tree.test.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tree.test.Fatalf("fake sensors: %v", err)
	}
}
