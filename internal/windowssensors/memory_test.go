package windowssensors_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/fakewinsensors"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/windowssensors"
)

const gibibyte = 1 << 30

func everything(string) bool { return true }

func only(names ...string) func(string) bool {
	return func(name string) bool {
		for _, each := range names {
			if each == name {
				return true
			}
		}
		return false
	}
}

func TestTheThreeMemoryMetricsComeFromOneCall(test *testing.T) {
	fake := &fakewinsensors.Fake{Total: 32 * gibibyte, Available: 12 * gibibyte}
	got := windowssensors.NewMemory(fake).Read(context.Background(), everything)
	want := []sensors.Reading{{Name: "ram.used", Value: 20}, {Name: "ram.total", Value: 32}, {Name: "ram.util", Value: 62.5}}
	if !reflect.DeepEqual(got, want) || fake.MemoryCalls() != 1 {
		test.Errorf("got %v, %d calls", got, fake.MemoryCalls())
	}
}

func TestRAMUtilisationAloneIsTheShareOfUsedToTotalNotAWholePercentage(test *testing.T) {
	fake := &fakewinsensors.Fake{Total: 3 * gibibyte, Available: 2 * gibibyte}
	got := windowssensors.NewMemory(fake).Read(context.Background(), only("ram.util"))
	if len(got) != 1 || got[0].Name != "ram.util" || got[0].Value < 33.333333 || got[0].Value > 33.333334 {
		test.Errorf("got %v", got)
	}
}

func TestTheMemoryIsUnavailableForAFailedCallATotalOfZeroAndMoreAvailableThanTotal(test *testing.T) {
	cases := map[string]*fakewinsensors.Fake{
		"failed call":        {MemoryError: errors.New("failed"), Total: 8 * gibibyte},
		"total of 0":         {Total: 0, Available: 0},
		"available > total ": {Total: 8 * gibibyte, Available: 9 * gibibyte},
	}
	for name, fake := range cases {
		if got := windowssensors.NewMemory(fake).Read(context.Background(), everything); len(got) != 0 {
			test.Errorf("%s: got %v", name, got)
		}
		if fake.MemoryCalls() != 1 {
			test.Errorf("%s: %d calls", name, fake.MemoryCalls())
		}
	}
}

func TestAnAvailableEqualToTheTotalIsAnEmptyMemoryNotAnError(test *testing.T) {
	fake := &fakewinsensors.Fake{Total: 8 * gibibyte, Available: 8 * gibibyte}
	got := windowssensors.NewMemory(fake).Read(context.Background(), everything)
	if len(got) != 3 || got[0].Value != 0 || got[2].Value != 0 {
		test.Errorf("got %v", got)
	}
}

func TestNoMemoryMetricWantedMakesNoCall(test *testing.T) {
	fake := &fakewinsensors.Fake{Total: gibibyte}
	if got := windowssensors.NewMemory(fake).Read(context.Background(), only("cpu.util", "net.up")); len(got) != 0 || fake.MemoryCalls() != 0 {
		test.Errorf("got %v, %d calls", got, fake.MemoryCalls())
	}
}
