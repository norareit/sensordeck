package windowssensors_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakewinsensors"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/windowssensors"
)

// The flags of a row, as the spec names them.
const (
	hardware  = 0x01
	filter    = 0x02
	connector = 0x04
	adapter   = hardware | connector
)

// clock gives the moments in turn, and fails the test when it is asked for more.
type clock struct {
	test    *testing.T
	moments []time.Time
	asked   int
}

func (given *clock) now() time.Time {
	if given.asked >= len(given.moments) {
		given.test.Fatalf("the clock was asked %d times, for %d moments", given.asked+1, len(given.moments))
	}
	given.asked++
	return given.moments[given.asked-1]
}

// at is the moment that many seconds after an arbitrary start.
func at(seconds float64) time.Time {
	return time.Unix(1000, 0).Add(time.Duration(seconds * float64(time.Second)))
}

func row(luid uint64, flags uint8, received, sent uint64) windowssensors.Interface {
	return windowssensors.Interface{LUID: luid, Flags: flags, InOctets: received, OutOctets: sent}
}

type networkRun struct {
	fake   *fakewinsensors.Fake
	waits  waits
	clock  *clock
	source sensors.Source
}

func newNetworkRun(test *testing.T, tables [][]windowssensors.Interface, moments ...time.Time) *networkRun {
	run := &networkRun{fake: &fakewinsensors.Fake{Tables: tables}, clock: &clock{test: test, moments: moments}}
	run.source = windowssensors.NewNetwork(run.fake, run.waits.wait, run.clock.now)
	return run
}

func (run *networkRun) read(wanted func(string) bool) []sensors.Reading {
	return run.source.Read(context.Background(), wanted)
}

func rates(up, down float64) []sensors.Reading {
	return []sensors.Reading{{Name: "net.up", Value: up}, {Name: "net.down", Value: down}}
}

func TestTheFirstNetworkReadingTakesTwoSamplesAndWaitsOnce(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 1000, 500)}, {row(1, adapter, 3048, 1524)}}, at(0), at(2))
	// 2048 bytes received and 1024 sent in two seconds
	if got := run.read(everything); !reflect.DeepEqual(got, rates(0.5, 1)) {
		test.Errorf("got %v", got)
	}
	if run.fake.InterfaceCalls() != 2 || !reflect.DeepEqual(run.waits, waits{200 * time.Millisecond}) {
		test.Errorf("%d calls, waits %v", run.fake.InterfaceCalls(), run.waits)
	}
}

func TestTheSecondNetworkReadingUsesTheKeptSampleAndDoesNotWait(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 0, 0)}, {row(1, adapter, 1024, 0)}, {row(1, adapter, 1024+4096, 0)}}, at(0), at(1), at(3))
	run.read(everything)
	// 4096 bytes in the two seconds since the kept sample
	if got := run.read(everything); !reflect.DeepEqual(got, rates(0, 2)) {
		test.Errorf("got %v", got)
	}
	if run.fake.InterfaceCalls() != 3 || len(run.waits) != 1 {
		test.Errorf("%d calls, waits %v", run.fake.InterfaceCalls(), run.waits)
	}
}

func TestTheTimeBetweenTheSamplesDecidesTheValue(test *testing.T) {
	tables := [][]windowssensors.Interface{{row(1, adapter, 0, 0)}, {row(1, adapter, 10240, 0)}}
	fast := newNetworkRun(test, tables, at(0), at(1)).read(only("net.down"))
	slow := newNetworkRun(test, tables, at(0), at(5)).read(only("net.down"))
	if fast[0].Value != 10 || slow[0].Value != 2 {
		test.Errorf("fast %v, slow %v", fast, slow)
	}
}

func TestTwoRowsThatCountAreSummed(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 0, 0), row(2, adapter, 0, 0)}, {row(1, adapter, 1024, 2048), row(2, adapter, 3072, 4096)}}, at(0), at(1))
	if got := run.read(everything); !reflect.DeepEqual(got, rates(6, 4)) {
		test.Errorf("got %v", got)
	}
}

func TestARowThatIsNotAPhysicalAdapterAddsNothing(test *testing.T) {
	notCounting := map[string]uint8{
		"no hardware flag":  connector,
		"no connector":      hardware,
		"a filter module":   adapter | filter,
		"no flag":           0,
		"another flag only": 0x80,
	}
	for name, flags := range notCounting {
		run := newNetworkRun(test, [][]windowssensors.Interface{
			{row(1, adapter, 0, 0), row(2, flags, 0, 0)}, {row(1, adapter, 1024, 1024), row(2, flags, 99999, 99999)}}, at(0), at(1))
		if got := run.read(everything); !reflect.DeepEqual(got, rates(1, 1)) {
			test.Errorf("%s: got %v", name, got)
		}
	}
}

func TestWhetherARowCountsIsDecidedAtEverySample(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 0, 0), row(2, hardware, 0, 0)},
		{row(1, adapter, 1024, 1024), row(2, adapter, 5000, 5000)}}, at(0), at(1))
	// the second row counts only in the second sample, so it is in one sample and adds nothing
	if got := run.read(everything); !reflect.DeepEqual(got, rates(1, 1)) {
		test.Errorf("got %v", got)
	}
}

func TestAnInterfaceThatAppearsOrVanishesAddsNothing(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 0, 0), row(2, adapter, 100, 100)},
		{row(1, adapter, 1024, 1024), row(3, adapter, 9999, 9999)}}, at(0), at(1))
	if got := run.read(everything); !reflect.DeepEqual(got, rates(1, 1)) {
		test.Errorf("got %v", got)
	}
}

func TestACounterThatWentBackAddsNothingToItsDirectionForOneReading(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 5000, 0), row(2, adapter, 0, 0)},
		{row(1, adapter, 100, 1024), row(2, adapter, 2048, 0)},
		{row(1, adapter, 1124, 2048), row(2, adapter, 3072, 0)}}, at(0), at(1), at(2))
	// received: the first interface went back and adds nothing, the second adds 2 KiB
	if got := run.read(everything); !reflect.DeepEqual(got, rates(1, 2)) {
		test.Errorf("got %v", got)
	}
	// the next reading counts from the sample that went back
	if got := run.read(everything); !reflect.DeepEqual(got, rates(1, 2)) {
		test.Errorf("got %v", got)
	}
}

func TestAFailedNetworkCallGivesNothingAndDoesNotWait(test *testing.T) {
	run := newNetworkRun(test, nil)
	run.fake.InterfacesError = errors.New("failed")
	if got := run.read(everything); len(got) != 0 || len(run.waits) != 0 || run.fake.InterfaceCalls() != 1 || run.clock.asked != 0 {
		test.Errorf("got %v, waits %v, %d calls, clock %d", got, run.waits, run.fake.InterfaceCalls(), run.clock.asked)
	}
}

func TestNoRowThatCountsGivesNothing(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, hardware, 0, 0)}, {row(1, hardware, 1024, 1024)}}, at(0), at(1))
	if got := run.read(everything); len(got) != 0 {
		test.Errorf("got %v", got)
	}
	run = newNetworkRun(test, nil, at(0), at(1))
	if got := run.read(everything); len(got) != 0 {
		test.Errorf("no row at all: got %v", got)
	}
}

func TestATimeOfZeroGivesNothing(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 0, 0)}, {row(1, adapter, 1024, 1024)}}, at(1), at(1))
	if got := run.read(everything); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestNetDownAloneGivesOneReadingFromOneSample(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{
		{row(1, adapter, 0, 0)}, {row(1, adapter, 2048, 4096)}}, at(0), at(1))
	if got := run.read(only("net.down")); !reflect.DeepEqual(got, []sensors.Reading{{Name: "net.down", Value: 2}}) {
		test.Errorf("got %v", got)
	}
	if run.fake.InterfaceCalls() != 2 {
		test.Errorf("%d calls", run.fake.InterfaceCalls())
	}
}

func TestNoNetworkMetricWantedMakesNoCall(test *testing.T) {
	run := newNetworkRun(test, [][]windowssensors.Interface{{row(1, adapter, 0, 0)}})
	if got := run.read(only("cpu.util")); len(got) != 0 || run.fake.InterfaceCalls() != 0 || run.clock.asked != 0 || len(run.waits) != 0 {
		test.Errorf("got %v, %d calls", got, run.fake.InterfaceCalls())
	}
}
