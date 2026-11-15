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

// waits records the durations a source waits for, without waiting.
type waits []time.Duration

func (recorded *waits) wait(duration time.Duration) { *recorded = append(*recorded, duration) }

func readCPU(source sensors.Source) []sensors.Reading {
	return source.Read(context.Background(), everything)
}

func TestTheFirstCPUReadingTakesTwoSamplesAndWaitsOnce(test *testing.T) {
	// the kernel time includes the idle time: the total grows by 200 and the idle time by 50, so three quarters are busy
	fake := &fakewinsensors.Fake{TimesList: []fakewinsensors.Times{{Idle: 100, Kernel: 400, User: 100}, {Idle: 150, Kernel: 500, User: 200}}}
	var recorded waits
	got := readCPU(windowssensors.NewCPU(fake, recorded.wait))
	if want := []sensors.Reading{{Name: "cpu.util", Value: 75}}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
	if fake.TimesCalls() != 2 || !reflect.DeepEqual(recorded, waits{200 * time.Millisecond}) {
		test.Errorf("%d calls, waits %v", fake.TimesCalls(), recorded)
	}
}

func TestTheSecondCPUReadingMakesOneCallAndDoesNotWait(test *testing.T) {
	fake := &fakewinsensors.Fake{TimesList: []fakewinsensors.Times{
		{Idle: 100, Kernel: 400, User: 100}, {Idle: 150, Kernel: 500, User: 200}, {Idle: 150, Kernel: 600, User: 300}}}
	var recorded waits
	source := windowssensors.NewCPU(fake, recorded.wait)
	readCPU(source)
	got := readCPU(source) // growth 200 of the total, none of the idle time
	if want := []sensors.Reading{{Name: "cpu.util", Value: 100}}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
	if fake.TimesCalls() != 3 || len(recorded) != 1 {
		test.Errorf("%d calls, waits %v", fake.TimesCalls(), recorded)
	}
}

func TestAGrowthOfZeroGivesNothing(test *testing.T) {
	fake := &fakewinsensors.Fake{TimesList: []fakewinsensors.Times{{Idle: 1, Kernel: 4, User: 1}}}
	var recorded waits
	if got := readCPU(windowssensors.NewCPU(fake, recorded.wait)); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestACPUShareAboveAHundredIsClamped(test *testing.T) {
	// the idle time went back, so that the busy time is more than the total
	fake := &fakewinsensors.Fake{TimesList: []fakewinsensors.Times{{Idle: 100, Kernel: 400, User: 100}, {Idle: 50, Kernel: 500, User: 200}}}
	var recorded waits
	got := readCPU(windowssensors.NewCPU(fake, recorded.wait))
	if want := []sensors.Reading{{Name: "cpu.util", Value: 100}}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestAFailedCPUCallGivesNothingAndDoesNotWait(test *testing.T) {
	fake := &fakewinsensors.Fake{TimesError: errors.New("failed")}
	var recorded waits
	if got := readCPU(windowssensors.NewCPU(fake, recorded.wait)); len(got) != 0 || len(recorded) != 0 || fake.TimesCalls() != 1 {
		test.Errorf("got %v, waits %v, %d calls", got, recorded, fake.TimesCalls())
	}
}

func TestCPUUtilisationNotWantedMakesNoCall(test *testing.T) {
	fake := &fakewinsensors.Fake{TimesList: []fakewinsensors.Times{{Kernel: 1}}}
	var recorded waits
	source := windowssensors.NewCPU(fake, recorded.wait)
	if got := source.Read(context.Background(), only("ram.used")); len(got) != 0 || fake.TimesCalls() != 0 || len(recorded) != 0 {
		test.Errorf("got %v, %d calls", got, fake.TimesCalls())
	}
}
