package sensors

import (
	"reflect"
	"strings"
	"testing"
)

func TestIsName(test *testing.T) {
	accepted := []string{
		"cpu.temp", "cpu.util", "cpu.clock", "gpu.temp", "gpu.util", "gpu.mem.used", "gpu.mem.total", "ram.used",
		"ram.total", "cpu.ccd1.temp", "ram.2.temp", "nvme.10.temp", "nvme.9.temp",
		"commander.1.temp", "commander.10.temp", "ram.util", "net.up", "net.down",
	}
	for _, name := range accepted {
		if !IsName(name) {
			test.Errorf("%q is refused", name)
		}
	}
	refused := []string{
		"ram.0.temp", "ram.01.temp", "CPU.temp", "cpu.tmp", "ram..temp", "ram.-1.temp", "ram.1x.temp", "",
		"ram.1.2.temp", "ram.1.temp.", "cpu.ccd.temp", "commander.0.temp", "commander.01.temp", "ram.util.x", "net.upload", "NET.UP",
	}
	for _, name := range refused {
		if IsName(name) {
			test.Errorf("%q is accepted", name)
		}
	}
}

func TestUnitOfEachRow(test *testing.T) {
	want := map[string]string{
		"cpu.temp": "°C", "cpu.ccd1.temp": "°C", "cpu.util": "%", "cpu.clock": "MHz", "gpu.temp": "°C",
		"gpu.util": "%", "gpu.mem.used": "MiB", "gpu.mem.total": "MiB", "ram.used": "GiB", "ram.total": "GiB",
		"ram.1.temp": "°C", "nvme.1.temp": "°C",
		"commander.1.temp": "°C", "ram.util": "%", "net.up": "KiB/s", "net.down": "KiB/s",
	}
	if len(want) != len(metricKinds) {
		test.Fatalf("%d names for %d rows", len(want), len(metricKinds))
	}
	for name, unit := range want {
		if got := Unit(name); got != unit {
			test.Errorf("Unit(%q) = %q, want %q", name, got, unit)
		}
	}
	if got := Unit("nothing"); got != "" {
		test.Errorf("Unit of no name = %q", got)
	}
}

func TestNumberedNames(test *testing.T) {
	if CCDTemperature(2) != "cpu.ccd2.temp" || RAMTemperature(1) != "ram.1.temp" || NVMeTemperature(12) != "nvme.12.temp" ||
		CommanderTemperature(2) != "commander.2.temp" {
		test.Error("a numbered name is wrong")
	}
}

func TestTableOrdersTheNewRows(test *testing.T) {
	position := func(prefix string) int {
		for index, kind := range metricKinds {
			if kind.prefix == prefix {
				return index
			}
		}
		test.Fatalf("no row for %q", prefix)
		return -1
	}
	if position(RAMUtilisation) != position(RAMTotal)+1 {
		test.Error("ram.util is not right after ram.total")
	}
	if position(NetworkUp) != position("nvme.")+1 || position(NetworkDown) != position(NetworkUp)+1 ||
		position("commander.") != position(NetworkDown)+1 {
		test.Error("the network rows are not between nvme.<n>.temp and commander.<n>.temp")
	}
}

func TestIsCommanderTemperature(test *testing.T) {
	for name, want := range map[string]bool{
		"commander.1.temp": true, "commander.12.temp": true, "commander.0.temp": false, "commander.temp": false,
		"cpu.temp": false, "ram.1.temp": false,
	} {
		if got := IsCommanderTemperature(name); got != want {
			test.Errorf("%s: %v", name, got)
		}
	}
}

func TestInTableOrderSortsByRowAndNumber(test *testing.T) {
	given := []Reading{
		{Name: "commander.1.temp", Value: 1}, {Name: "net.down", Value: 2}, {Name: "nvme.10.temp", Value: 3},
		{Name: "nvme.2.temp", Value: 4}, {Name: "cpu.clock", Value: 5}, {Name: "ram.used", Value: 6},
		{Name: "cpu.ccd10.temp", Value: 7}, {Name: "cpu.ccd2.temp", Value: 8}, {Name: "cpu.temp", Value: 9},
		{Name: "ram.1.temp", Value: 10}, {Name: "net.up", Value: 11}, {Name: "commander.2.temp", Value: 12},
	}
	before := append([]Reading(nil), given...)
	var names []string
	for _, reading := range InTableOrder(given) {
		names = append(names, reading.Name)
	}
	want := "cpu.temp cpu.ccd2.temp cpu.ccd10.temp cpu.clock ram.used ram.1.temp nvme.2.temp nvme.10.temp net.up net.down commander.1.temp commander.2.temp"
	if got := strings.Join(names, " "); got != want {
		test.Errorf("got %s", got)
	}
	if !reflect.DeepEqual(given, before) {
		test.Error("the input was changed")
	}
}

func TestInTableOrderKeepsTheOrderOfEqualReadings(test *testing.T) {
	given := []Reading{{Name: "cpu.temp", Value: 1}, {Name: "cpu.temp", Value: 2}, {Name: "net.up", Value: 3}, {Name: "cpu.temp", Value: 4}}
	got := InTableOrder(given)
	if got[0].Value != 1 || got[1].Value != 2 || got[2].Value != 4 || got[3].Name != "net.up" {
		test.Errorf("got %v", got)
	}
}
