package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/norareit/sensordeck/internal/sensors"
)

// fakeSource gives the readings it holds for the names that are wanted, in the order of its list, and records what it
// was asked for.
type fakeSource struct {
	names   []string
	values  map[string]float64
	asked   [][]string
	onRead  func()
	readsAt int
}

func (source *fakeSource) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	source.readsAt++
	if source.onRead != nil {
		source.onRead()
	}
	var readings []sensors.Reading
	var asked []string
	for _, name := range source.names {
		if !wanted(name) {
			continue
		}
		asked = append(asked, name)
		if value, available := source.values[name]; available {
			readings = append(readings, sensors.Reading{Name: name, Value: value})
		}
	}
	source.asked = append(source.asked, asked)
	return readings
}

func TestSensorsWritesTheReadingsWithUnitAndOneDecimal(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{
		&fakeSource{names: []string{"cpu.temp", "cpu.ccd2.temp"}, values: map[string]float64{"cpu.temp": 51.375, "cpu.ccd2.temp": 38.125}},
		&fakeSource{names: []string{"cpu.clock", "ram.used"}, values: map[string]float64{"cpu.clock": 40, "ram.used": 21.84}},
	}
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	want := "cpu.temp\t51.4\t°C\ncpu.ccd2.temp\t38.1\t°C\ncpu.clock\t40.0\tMHz\nram.used\t21.8\tGiB\n"
	if got := harness.stdout.String(); got != want {
		test.Errorf("stdout %q", got)
	}
	if harness.stderr.Len() != 0 {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestSensorsWithNoReadingSaysSoOnStandardError(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{&fakeSource{}}
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if harness.stdout.Len() != 0 || harness.stderr.String() != "no sensor could be read\n" {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
}

func TestSensorsTakesNoArguments(test *testing.T) {
	newHarness().expectFailure(test, `unknown command "x" for "sensordeck sensors"`, "sensors", "x")
}

func TestSensorsFailsWhenTheSourcesAreNotSupported(test *testing.T) {
	harness := newHarness()
	harness.sourcesErr = errors.New("windows is not supported yet")
	harness.expectFailure(test, "windows is not supported yet", "sensors")
}

func TestSensorsNeverOpensATransport(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{&fakeSource{}}
	harness.run("--device", "x", "sensors")
	if len(harness.opened) != 0 {
		test.Errorf("opened %v", harness.opened)
	}
}
