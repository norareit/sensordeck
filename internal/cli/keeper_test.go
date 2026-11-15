package cli

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// fakeLighting stands in for the session of the controller. It records the moment and the colour of every setup and
// the moment of every reading on the clock of the harness, and takes the result of each setup and of each reading
// from a queue of errors, nil once the queue is empty. It ignores the stop, as a setup that was already under way
// does.
type fakeLighting struct {
	clock       *fakeClock
	setupErrors []error
	readErrors  []error
	setups      []fakeSetup
	reads       []time.Time
	teardowns   int
}

type fakeSetup struct {
	at     time.Time
	colour config.Colour
}

type fakeHeldSession struct{ lighting *fakeLighting }

func (lighting *fakeLighting) start(_ context.Context, colour config.Colour) (heldSession, error) {
	lighting.setups = append(lighting.setups, fakeSetup{at: lighting.clock.now(), colour: colour})
	if len(lighting.setupErrors) > 0 {
		err := lighting.setupErrors[0]
		lighting.setupErrors = lighting.setupErrors[1:]
		if err != nil {
			return nil, err
		}
	}
	return fakeHeldSession{lighting: lighting}, nil
}

func (held fakeHeldSession) read(context.Context) ([]sensors.Reading, error) {
	lighting := held.lighting
	lighting.reads = append(lighting.reads, lighting.clock.now())
	if len(lighting.readErrors) > 0 {
		err := lighting.readErrors[0]
		lighting.readErrors = lighting.readErrors[1:]
		if err != nil {
			lighting.teardowns++
			return nil, err
		}
	}
	return []sensors.Reading{{Name: "commander.1.temp", Value: 37}, {Name: "commander.2.temp", Value: 23}}, nil
}

func (held fakeHeldSession) end() { held.lighting.teardowns++ }

// colourTags is the config of a run that lights the LEDs, with the tags of controllerTags and an interval before them.
func colourTags(interval string) string {
	return "interval: " + interval + "\ncommander_color: \"#ff0000\"\n" + controllerTags
}

// runKeeping gives a harness with the stand-in, the clock that waits for the keeper, and a stop after the waits.
func runKeeping(lighting *fakeLighting, stopAfterWaits int) *harness {
	harness := runWith(twoMetrics(), stopAfterWaits)
	lighting.clock = harness.clock
	harness.startSession = lighting.start
	harness.clock.keeping = true
	return harness
}

func startOfThe(lighting *fakeLighting) time.Time { return lighting.setups[0].at }

func TestRunKeepsTheControllerAwakeWithAnIntervalOfTenSeconds(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 6)
	harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	checkTheKeepalive(test, lighting)
}

func TestRunKeepsTheControllerAwakeWhileTheScreenIsLost(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 6)
	harness.transport.failWrites = []bool{false, true}
	harness.transport.reopenErr = errFake
	harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	if harness.transport.reopens == 0 {
		test.Fatal("the screen was not lost")
	}
	checkTheKeepalive(test, lighting)
}

func checkTheKeepalive(test *testing.T, lighting *fakeLighting) {
	test.Helper()
	moments := append([]time.Time{startOfThe(lighting)}, lighting.reads...)
	for position := 1; position < len(moments); position++ {
		if gap := moments[position].Sub(moments[position-1]); gap > time.Second {
			test.Fatalf("%v passed between two readings, at %v", gap, moments[position])
		}
	}
	if last := moments[len(moments)-1].Sub(moments[0]); last < 59*time.Second {
		test.Fatalf("the last reading came %v after the start", last)
	}
}

func TestRunReadsTheControllerInEveryRoundAndSendsItsProbes(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 30)
	harness.run("run", "--config", writeConfig(test, colourTags("100ms")))
	if len(lighting.reads) != 30 {
		test.Fatalf("%d readings", len(lighting.reads))
	}
	for position := 1; position < len(lighting.reads); position++ {
		if gap := lighting.reads[position].Sub(lighting.reads[position-1]); gap != 100*time.Millisecond {
			test.Fatalf("%v between readings %d and %d", gap, position-1, position)
		}
	}
	want := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 4, Value: 37}, smartmonitor.Value{Tag: 5, Value: 23})
	for round, report := range valueReports(harness) {
		if report != want {
			test.Errorf("round %d: % X", round, report[:14])
		}
	}
}

func TestRunAsksForNoReadingWhenNoProbeIsMappedAndKeepsTheSession(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 3)
	harness.run("run", "--config", writeConfig(test, "interval: 10s\ncommander_color: \"#ff0000\"\ntags:\n  cpu.temp: 1\n"))
	if len(lighting.reads) != 30 {
		test.Fatalf("%d readings", len(lighting.reads))
	}
	for position, moment := range lighting.reads {
		if want := startOfThe(lighting).Add(time.Duration(position+1) * time.Second); !moment.Equal(want) {
			test.Fatalf("reading %d at %v, want %v", position, moment, want)
		}
	}
}

func TestRunLeavesTheControllerOutAfterAFailedReading(test *testing.T) {
	lighting := &fakeLighting{readErrors: []error{nil, nil, errFake}}
	harness := runKeeping(lighting, 5)
	harness.run("run", "--config", writeConfig(test, colourTags("1s")))
	reports := valueReports(harness)
	whole := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 4, Value: 37}, smartmonitor.Value{Tag: 5, Value: 23})
	alone := valueReport(smartmonitor.Value{Tag: 1, Value: 42})
	if len(reports) != 5 || reports[0] != whole {
		test.Fatalf("%d value reports", len(reports))
	}
	for _, report := range reports[2:] {
		if report != alone {
			test.Errorf("a report after the failure: % X", report[:14])
		}
	}
	if !strings.Contains(harness.stderr.String(), "controller lost: fake failure\n") {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestRunWaitsLongerAfterEachFailedSetup(test *testing.T) {
	lighting := &fakeLighting{setupErrors: make([]error, 8)}
	for position := range lighting.setupErrors {
		lighting.setupErrors[position] = errFake
	}
	harness := runKeeping(lighting, 100)
	harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	var gaps []time.Duration
	for position := 1; position < len(lighting.setups); position++ {
		gaps = append(gaps, lighting.setups[position].at.Sub(lighting.setups[position-1].at))
	}
	want := []time.Duration{10 * time.Second, 20 * time.Second, 40 * time.Second, 80 * time.Second, 160 * time.Second, 300 * time.Second, 300 * time.Second}
	if !reflect.DeepEqual(gaps, want) || len(lighting.setups) != 8 {
		test.Errorf("%d setups, gaps %v", len(lighting.setups), gaps)
	}
}

func TestRunSetsUpAgainTenSecondsAfterALostSessionAndLightsTheColourAgain(test *testing.T) {
	lighting := &fakeLighting{setupErrors: []error{errFake, errFake, errFake}, readErrors: []error{errFake}}
	harness := runKeeping(lighting, 10)
	harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	if len(lighting.setups) < 5 || len(lighting.reads) == 0 {
		test.Fatalf("%d setups, %d readings", len(lighting.setups), len(lighting.reads))
	}
	if gap := lighting.setups[4].at.Sub(lighting.reads[0]); gap != 10*time.Second {
		test.Errorf("the setup came %v after the loss", gap)
	}
	for position, setup := range lighting.setups {
		if setup.colour != (config.Colour{Red: 255}) {
			test.Errorf("setup %d lit %v", position, setup.colour)
		}
	}
}

func TestRunWritesTheLinesOfTheControllerOnceEach(test *testing.T) {
	lighting := &fakeLighting{setupErrors: []error{errFake, errFake, errFake}, readErrors: []error{errFake}}
	harness := runKeeping(lighting, 10)
	harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	want := "controller setup failed: fake failure\ncontroller lost: fake failure\ncontroller back\n"
	if got := harness.stderr.String(); got != want {
		test.Errorf("stderr %q", got)
	}
}

func TestRunWritesNoLineWhenThereIsNoController(test *testing.T) {
	lighting := &fakeLighting{setupErrors: []error{errNoController, errNoController, errNoController}}
	harness := runKeeping(lighting, 10)
	harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	if got := harness.stderr.String(); got != "" || len(lighting.setups) < 4 {
		test.Errorf("stderr %q after %d setups", got, len(lighting.setups))
	}
}

func TestRunEndsTheSessionWithOneTeardownOnAStop(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 3)
	status := harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	if status != 0 || lighting.teardowns != 1 || !strings.HasSuffix(harness.stdout.String(), "stopped\n") {
		test.Errorf("status %d, %d teardowns, stdout %q", status, lighting.teardowns, harness.stdout.String())
	}
}

func TestRunEndsTheSessionWithOneTeardownOnAFeedingError(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 3)
	harness.clock.current = time.Date(2256, 1, 1, 0, 0, 0, 0, time.UTC)
	status := harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	if status != 1 || lighting.teardowns != 1 {
		test.Errorf("status %d, %d teardowns", status, lighting.teardowns)
	}
}

func TestRunWithoutAColourMakesNoSetupAndReadsTheControllerOfTheSources(test *testing.T) {
	lighting := &fakeLighting{}
	controller := twoProbes()
	harness := runKeeping(lighting, 3)
	harness.controller = controller
	harness.clock.keeping = false
	harness.run("run", "--config", writeConfig(test, controllerTags))
	if len(lighting.setups) != 0 || controller.readsAt != 1 {
		test.Errorf("%d setups, %d reads of the controller", len(lighting.setups), controller.readsAt)
	}
}
