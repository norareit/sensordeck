//go:build linux

package e2e

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/fakesensors"
)

// fullTree makes the tree of the sensors of the host on the sysfs root of the fake screen, and gives the environment
// that points the executable at it. The PATH holds the fake nvidia-smi of the given script, or nothing when it is
// empty, so that no real nvidia-smi is ever run.
func fullTree(test *testing.T, fake *fakescreen.Fake, nvidiaSMI string) []string {
	test.Helper()
	tree := fakesensors.New(test, fake.SysfsRoot)
	const pci = "pci0000:00/"
	composite := func(input string) fakesensors.Sensor {
		return fakesensors.Sensor{Number: 1, Label: "Composite", Input: input}
	}
	tree.Hwmon(0, "nvme", pci+"0000:00:02.1/0000:06:00.0/nvme/nvme0", composite("38800"))
	tree.Hwmon(1, "nvme", pci+"0000:00:01.2/0000:02:00.0/nvme/nvme1", composite("33900"))
	tree.Hwmon(2, "k10temp", pci+"0000:00:18.3",
		fakesensors.Sensor{Number: 1, Label: "Tctl", Input: "51375"},
		fakesensors.Sensor{Number: 2, Label: "Tccd1", Input: "40000"},
		fakesensors.Sensor{Number: 3, Label: "Tccd2", Input: "38125"})
	tree.Hwmon(3, "spd5118", pci+"0000:00:14.0/i2c-1/1-0053", fakesensors.Sensor{Number: 1, Input: "37875"})
	tree.Hwmon(4, "spd5118", pci+"0000:00:14.0/i2c-1/1-0051", fakesensors.Sensor{Number: 1, Input: "40000"})
	tree.CPUFrequency(0, "4425361\n")
	tree.CPUFrequency(1, "3000000\n")
	tree.Stat("cpu  100 0 100 700 100 0 0 0 0 0\n")
	tree.Meminfo("MemTotal:       96187828 kB\nMemFree:         1000 kB\nMemAvailable:   73300372 kB\n")
	addNetwork(tree)
	return environmentFor(test, tree, nvidiaSMI)
}

// netDevHeader and netDevLine write /proc/net/dev as the kernel does, with 0 in every field but the two byte counters.
const netDevHeader = "Inter-|   Receive                                                |  Transmit\n" +
	" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n"

func netDevLine(name string, received, transmitted uint64) string {
	return fmt.Sprintf("%6s: %d 0 0 0 0 0 0 0 %d 0 0 0 0 0 0 0\n", name, received, transmitted)
}

// addNetwork gives the tree a net/dev that does not change, with loopback, a card and a bridge, of which only the
// card belongs to a device.
func addNetwork(tree *fakesensors.Tree) {
	tree.NetDev(netDevHeader + netDevLine("lo", 5000, 5000) + netDevLine("enp6s0", 123456, 654321) +
		netDevLine("docker0", 777, 888))
	tree.NetInterface("lo", "")
	tree.NetInterface("enp6s0", "pci0000:00/0000:00:02.2/0000:06:00.0")
	tree.NetInterface("docker0", "")
}

func environmentFor(test *testing.T, tree *fakesensors.Tree, nvidiaSMI string) []string {
	test.Helper()
	path := test.TempDir()
	if nvidiaSMI != "" {
		path = fakesensors.NvidiaSMI(test, nvidiaSMI)
	}
	return []string{"SENSORDECK_PROC_ROOT=" + tree.ProcRoot, "PATH=" + path}
}

func TestSensorsListsTheReadingsOfTheHost(test *testing.T) {
	fake := fakescreen.New(test)
	environment := fullTree(test, fake, "echo '44, 17, 3256, 32607'\n")
	want := "cpu.temp\t51.4\t°C\n" +
		"cpu.ccd1.temp\t40.0\t°C\n" +
		"cpu.ccd2.temp\t38.1\t°C\n" +
		"cpu.clock\t4425.4\tMHz\n" +
		"gpu.temp\t44.0\t°C\n" +
		"gpu.util\t17.0\t%\n" +
		"gpu.mem.used\t3256.0\tMiB\n" +
		"gpu.mem.total\t32607.0\tMiB\n" +
		"ram.used\t21.8\tGiB\n" +
		"ram.total\t91.7\tGiB\n" +
		"ram.util\t23.8\t%\n" +
		"ram.1.temp\t40.0\t°C\n" +
		"ram.2.temp\t37.9\t°C\n" +
		"nvme.1.temp\t33.9\t°C\n" +
		"nvme.2.temp\t38.8\t°C\n" +
		"net.up\t0.0\tKiB/s\n" +
		"net.down\t0.0\tKiB/s\n"
	runWith(test, fake, environment, "sensors").expect(test, 0, want, "")
}

func TestSensorsWithAnEmptyTreeSaysSo(test *testing.T) {
	fake := fakescreen.New(test)
	empty := filepath.Join(test.TempDir(), "proc")
	environment := []string{"SENSORDECK_PROC_ROOT=" + empty, "PATH=" + test.TempDir()}
	runWith(test, fake, environment, "sensors").expect(test, 0, "", "no sensor could be read\n")
}
