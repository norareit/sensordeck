//go:build linux

package linuxsensors

import (
	"context"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/fakesensors"
	"github.com/norareit/sensordeck/internal/sensors"
)

func TestSourcesGiveTheMetricsInTheOrderOfTheTable(test *testing.T) {
	tree := newTree(test)
	tree.Hwmon(1, "k10temp", "pci/a",
		fakesensors.Sensor{Number: 1, Label: "Tctl", Input: "50000"},
		fakesensors.Sensor{Number: 2, Label: "Tccd1", Input: "40000"})
	tree.Hwmon(2, "spd5118", "pci/b", fakesensors.Sensor{Number: 1, Input: "30000"})
	tree.Hwmon(3, "nvme", "pci/c", fakesensors.Sensor{Number: 1, Label: "Composite", Input: "35000"})
	tree.CPUFrequency(0, "2000000\n")
	tree.Meminfo(hostMeminfo)
	tree.Stat("cpu  1 1 1 1 1 1 1 1 0 0\n")
	test.Setenv("PATH", fakesensors.NvidiaSMI(test, "echo '"+fullLine+"'\n"))
	got := sensors.ReadAll(context.Background(), Sources(tree.SysfsRoot, tree.ProcRoot), everything)
	want := []string{
		"cpu.temp", "cpu.ccd1.temp", "cpu.clock", "gpu.temp", "gpu.util", "gpu.mem.used", "gpu.mem.total",
		"ram.used", "ram.total", "ram.1.temp", "nvme.1.temp",
	}
	if !reflect.DeepEqual(names(got), want) {
		test.Errorf("got %v", names(got))
	}
}
