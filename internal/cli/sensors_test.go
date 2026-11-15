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
	stops   []context.Context
}

func (source *fakeSource) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	source.readsAt++
	source.stops = append(source.stops, stop)
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

func TestSensorsWithNoReadingAndAProblemWritesTheProblemFirst(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{&fakeSource{}}
	harness.controllerProblem = errors.New("no collection of interface 0 matches: p")
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	want := "no collection of interface 0 matches: p\nno sensor could be read\n"
	if harness.stdout.Len() != 0 || harness.stderr.String() != want {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
}

func TestSensorsWithAReadingWritesNoProblem(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{&fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 1}}}
	harness.controllerProblem = errors.New("a problem")
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if harness.stderr.Len() != 0 {
		test.Errorf("stderr %q", harness.stderr.String())
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

func TestSensorsInterruptedWhileReadingWritesNothingAndFails(test *testing.T) {
	harness := newHarness()
	source := &fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 50}}
	source.onRead = harness.cancelStop
	harness.sources = []sensors.Source{source}
	harness.expectFailure(test, "interrupted", "sensors")
	if len(source.stops) != 1 || source.stops[0].Err() == nil {
		test.Errorf("the source did not get a stop that was done: %v", source.stops)
	}
}

func TestSensorsGivesTheSourcesAStopThatIsNotDone(test *testing.T) {
	harness := newHarness()
	source := &fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 50}}
	harness.sources = []sensors.Source{source}
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if len(source.stops) != 1 || source.stops[0].Err() != nil {
		test.Errorf("stops %v", source.stops)
	}
}

func TestSensorsReadsTheControllerOnceAfterTheOthers(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{&fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 51}}}
	controller := &fakeSource{names: []string{"commander.1.temp"}, values: map[string]float64{"commander.1.temp": 37.3}}
	harness.controller = controller
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	want := "cpu.temp\t51.0\t°C\ncommander.1.temp\t37.3\t°C\n"
	if got := harness.stdout.String(); got != want {
		test.Errorf("stdout %q", got)
	}
	if controller.readsAt != 1 {
		test.Errorf("the controller got %d calls", controller.readsAt)
	}
}

func TestSensorsOnASetWithoutAMonitorNeverLooksForTheConfig(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{&fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 1}}}
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if harness.configDirCalls != 0 {
		test.Errorf("userConfigDir was called %d times", harness.configDirCalls)
	}
}

func TestSensorsWritesReadingsGivenOutOfOrderInTheOrderOfTheTable(test *testing.T) {
	harness := newHarness()
	harness.sources = []sensors.Source{
		&fakeSource{names: []string{"nvme.2.temp", "net.up", "cpu.temp", "nvme.1.temp"},
			values: map[string]float64{"nvme.2.temp": 2, "net.up": 3, "cpu.temp": 4, "nvme.1.temp": 5}},
	}
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	want := "cpu.temp\t4.0\t°C\nnvme.1.temp\t5.0\t°C\nnvme.2.temp\t2.0\t°C\nnet.up\t3.0\tKiB/s\n"
	if got := harness.stdout.String(); got != want {
		test.Errorf("stdout %q", got)
	}
}
