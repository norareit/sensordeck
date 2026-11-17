package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/norareit/sensordeck/internal/config"
	"reflect"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/devicelock"
	"github.com/norareit/sensordeck/internal/sensors"
)

const otherHolder = " (process 42: sensordeck run)"

func TestTheCommandsThatWriteToTheScreenRefuseWhileAnotherProcessHasIt(test *testing.T) {
	stopText := map[string]string{
		"linux":   "stop the service with systemctl --user stop sensordeck",
		"windows": "stop the startup task with sensordeck service uninstall",
	}
	for system, stop := range stopText {
		for _, arguments := range [][]string{{"clock"}, {"values", "1=2"}, {"upload", writeConfig(test, "x")}, {"listen"}, {"raw", "01"}} {
			harness := newHarness()
			harness.system = system
			harness.openErr = &devicelock.InUseError{Holder: otherHolder}
			if status := harness.run(arguments...); status != 1 {
				test.Errorf("%s %v: status %d", system, arguments, status)
			}
			want := "sensordeck: the screen is in use by another sensordeck process" + otherHolder + ", stop it first, or " + stop + "\n"
			if got := harness.stderr.String(); got != want {
				test.Errorf("%s %v: stderr %q", system, arguments, got)
			}
			if harness.stdout.Len() != 0 || len(harness.transport.writes) != 0 {
				test.Errorf("%s %v: stdout %q, %d writes", system, arguments, harness.stdout.String(), len(harness.transport.writes))
			}
		}
	}
}

func TestRunRefusesBeforeItWritesAnythingWhileAnotherProcessHasTheScreen(test *testing.T) {
	harness := newHarness()
	harness.openErr = &devicelock.InUseError{Holder: otherHolder}
	harness.startSession = func(context.Context, config.Colour) (heldSession, setupNotice, error) {
		test.Error("a session was started")
		return nil, setupNotice{}, errFake
	}
	if status := harness.run("run", "--config", writeConfig(test, colourTags("10s"))); status != 1 {
		test.Fatalf("status %d", status)
	}
	want := "sensordeck: the screen is in use by another sensordeck process" + otherHolder +
		", stop it first, or stop the service with systemctl --user stop sensordeck\n"
	if got := harness.stderr.String(); got != want {
		test.Errorf("stderr %q", got)
	}
	if strings.Contains(harness.stdout.String(), "running:") || len(harness.transport.writes) != 0 {
		test.Errorf("stdout %q, %d writes", harness.stdout.String(), len(harness.transport.writes))
	}
}

func TestAnErrorOfTheOpeningThatIsNotALockKeepsItsText(test *testing.T) {
	harness := newHarness()
	harness.openErr = errors.New("no screen")
	harness.run("clock")
	if got := harness.stderr.String(); got != "sensordeck: no screen\n" {
		test.Errorf("stderr %q", got)
	}
}

func TestDevicesAndSensorsTakeNoLockOfTheScreen(test *testing.T) {
	for _, arguments := range [][]string{{"devices"}, {"sensors"}} {
		harness := newHarness()
		harness.openErr = &devicelock.InUseError{}
		harness.screens = []string{"a screen"}
		harness.sources = []sensors.Source{&fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 1}}}
		if status := harness.run(arguments...); status != 0 {
			test.Errorf("%v: status %d, stderr %q", arguments, status, harness.stderr.String())
		}
		if len(harness.opened) != 0 {
			test.Errorf("%v opened %v", arguments, harness.opened)
		}
	}
}

// refusingSource is a source of the controller that another sensordeck process holds.
type refusingSource struct{ refused func(err error) }

func (source refusingSource) Read(context.Context, func(string) bool) []sensors.Reading {
	source.refused(fmt.Errorf("the controller is %w", &devicelock.InUseError{Holder: otherHolder}))
	return nil
}

func TestSensorsLeavesOutTheProbesOfAControllerInUseAndSaysSoOnce(test *testing.T) {
	harness := newHarness()
	harness.set = &sourceSet{
		others:     []sensors.Source{&fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 42}}},
		controller: &fakeSource{},
		reportController: func(refused func(err error)) sensors.Source {
			return refusingSource{refused: refused}
		},
	}
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	want := "the controller is in use by another sensordeck process" + otherHolder + ", its probes are left out\n"
	if got := harness.stderr.String(); got != want {
		test.Errorf("stderr %q", got)
	}
	if got := harness.stdout.String(); got != "cpu.temp\t42.0\t°C\n" {
		test.Errorf("stdout %q", got)
	}
}

func TestSensorsWithNoOtherReadingSaysSoAfterTheLineOfTheController(test *testing.T) {
	harness := newHarness()
	harness.set = &sourceSet{
		controller:       &fakeSource{},
		reportController: func(refused func(err error)) sensors.Source { return refusingSource{refused: refused} },
	}
	harness.run("sensors")
	want := "the controller is in use by another sensordeck process" + otherHolder + ", its probes are left out\n" +
		"no sensor could be read\n"
	if got := harness.stderr.String(); got != want {
		test.Errorf("stderr %q", got)
	}
}

func TestTheKeeperWritesTheLineOfAControllerInUseOnceOverSeveralSetups(test *testing.T) {
	refusal := fmt.Errorf("the controller is %w", &devicelock.InUseError{Holder: otherHolder})
	lighting := &fakeLighting{setupErrors: []error{refusal, refusal, refusal}}
	harness := runKeeping(lighting, 100)
	harness.run("run", "--config", writeConfig(test, colourTags("10s")))
	want := "controller setup failed: the controller is in use by another sensordeck process" + otherHolder + "\n"
	if got := harness.stderr.String(); got != want {
		test.Errorf("stderr %q", got)
	}
	if len(lighting.setups) < 4 {
		test.Fatalf("%d setups", len(lighting.setups))
	}
	var gaps []int64
	for position := 1; position < 4; position++ {
		gaps = append(gaps, int64(lighting.setups[position].at.Sub(lighting.setups[position-1].at).Seconds()))
	}
	if !reflect.DeepEqual(gaps, []int64{10, 20, 40}) {
		test.Errorf("setups came %v seconds apart", gaps)
	}
}

func TestABackgroundRunWhoseOpeningIsRefusedLogsTheLineAndStartsAgain(test *testing.T) {
	harness := backgroundHarness(test, "windows", 3)
	harness.openErr = &devicelock.InUseError{Holder: " (process 42: sensordeck run)"}
	configFile := writeConfig(test, twoTags)
	changing := harness.onWait
	harness.onWait = func(count int) {
		if count == 1 {
			harness.openErr = nil
		}
		changing(count)
	}
	if status := harness.run("run", "--background", "--config", configFile); status != 0 {
		test.Fatalf("status %d, log %q", status, logLinesOf(test, harness))
	}
	got := withoutTimes(test, logLinesOf(test, harness))
	want := []string{
		"sensordeck: the screen is in use by another sensordeck process (process 42: sensordeck run), stop it first, " +
			"or stop the startup task with sensordeck service uninstall",
		"starting again in 10s", "running: 2 metrics every 1s", "stopped",
	}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("log %q, want %q", got, want)
	}
}
