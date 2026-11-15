//go:build linux

package commandercore

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakecommander"
	"github.com/norareit/sensordeck/internal/hidraw"
	"github.com/norareit/sensordeck/internal/sensors"
)

const testTimeLimit = 50 * time.Millisecond

type harness struct {
	fake     *fakecommander.Fake
	source   Source
	stop     context.Context
	cancel   context.CancelFunc
	wantedBy func(string) bool
	devRoot  string
}

func newHarness(test *testing.T) *harness {
	test.Helper()
	root := test.TempDir()
	sysfsRoot, devRoot := filepath.Join(root, "sys"), filepath.Join(root, "dev")
	stop, cancel := context.WithCancel(context.Background())
	test.Cleanup(cancel)
	return &harness{
		fake:     fakecommander.New(test, sysfsRoot, devRoot),
		source:   Source{Finder: Hidraw(sysfsRoot, devRoot), TimeLimit: testTimeLimit},
		stop:     stop,
		cancel:   cancel,
		wantedBy: func(name string) bool { return strings.HasPrefix(name, "commander.") },
		devRoot:  devRoot,
	}
}

func (harness *harness) read() []sensors.Reading {
	return harness.source.Read(harness.stop, harness.wantedBy)
}

// sequence gives the writes of the steps with the numbers, 1 to 8, each as the driver writes it.
func sequence(numbers ...int) [][]byte {
	writes := make([][]byte, len(numbers))
	for position, number := range numbers {
		writes[position] = encode(steps[number-1])
	}
	return writes
}

var bothProbes = []sensors.Reading{{Name: "commander.1.temp", Value: 37.3}, {Name: "commander.2.temp", Value: 22.7}}

func TestBothProbesAreReadWithTheEightWrites(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	readings := harness.read()
	if !reflect.DeepEqual(readings, bothProbes) {
		test.Fatalf("got %v", readings)
	}
	if writes := controller.Writes(); !reflect.DeepEqual(writes, sequence(1, 2, 3, 4, 5, 6, 7, 8)) {
		test.Fatalf("got %d writes: % x", len(writes), writes)
	}
	if got := controller.WritesOnInterfaceOne(); len(got) != 0 {
		test.Fatalf("interface 1 got %d writes", len(got))
	}
	if controller.Mode() != fakecommander.HardwareMode {
		test.Fatal("the controller is not in hardware mode")
	}
}

// The controller answers with a full input report of 384 bytes. A driver that took an answer for exactly 96 bytes, as
// unit 0005 did, rejects the answer of step 1 and reads no probe.
func TestAnAnswerOfAFullInputReportIsRead(test *testing.T) {
	harness := newHarness(test)
	harness.fake.Add(9, 10)
	if got := harness.read(); !reflect.DeepEqual(got, bothProbes) {
		test.Fatalf("got %v", got)
	}
}

func TestAProbeWithAStatusKeepsTheNumberOfTheOther(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	controller.SetProbes(fakecommander.Probe{Status: 1, Tenths: 373}, fakecommander.Probe{Tenths: 227})
	want := []sensors.Reading{{Name: "commander.2.temp", Value: 22.7}}
	if got := harness.read(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %v", got)
	}
}

func TestANegativeTemperature(test *testing.T) {
	harness := newHarness(test)
	harness.fake.Add(9, 10).SetProbes(fakecommander.Probe{Tenths: -100}, fakecommander.Probe{Tenths: 227})
	if got := harness.read(); got[0] != (sensors.Reading{Name: "commander.1.temp", Value: -10}) {
		test.Fatalf("got %v", got)
	}
}

func TestOnlyTheWantedProbeIsGivenAfterAFullSequence(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	harness.wantedBy = func(name string) bool { return name == "commander.2.temp" }
	if got := harness.read(); !reflect.DeepEqual(got, bothProbes[1:]) {
		test.Fatalf("got %v", got)
	}
	if writes := controller.Writes(); len(writes) != 8 {
		test.Fatalf("got %d writes", len(writes))
	}
}

func TestNoControllerGivesNoReadings(test *testing.T) {
	harness := newHarness(test)
	if got := harness.read(); got != nil {
		test.Fatalf("without class/hidraw: %v", got)
	}
	harness.fake.Add(9, 10).RemoveNode()
	if got := harness.read(); got != nil {
		test.Fatalf("without the node: %v", got)
	}
}

func TestOnlyInterfaceOneGivesNoReadingsAndNoWrites(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.AddInterfaceOne(10)
	if got := harness.read(); got != nil {
		test.Fatalf("got %v", got)
	}
	if writes := controller.WritesOnInterfaceOne(); len(writes) != 0 {
		test.Fatalf("got %d writes", len(writes))
	}
}

func TestTheLowerNumberOfTwoControllersIsUsed(test *testing.T) {
	harness := newHarness(test)
	higher, lower := harness.fake.Add(7, 8), harness.fake.Add(3, 4)
	if got := harness.read(); !reflect.DeepEqual(got, bothProbes) {
		test.Fatalf("got %v", got)
	}
	if len(lower.Writes()) != 8 || len(higher.Writes()) != 0 {
		test.Fatal("the writes did not go to hidraw3 alone")
	}
}

func TestAFailureEndsTheStepsAndReturnsToHardwareMode(test *testing.T) {
	type failureCase struct {
		step    int
		failure fakecommander.Failure
	}
	var cases []failureCase
	for _, step := range []int{1, 3, 6} {
		for _, failure := range []fakecommander.Failure{fakecommander.NoAnswer, fakecommander.WrongEcho,
			fakecommander.BadStatus} {
			cases = append(cases, failureCase{step, failure})
		}
	}
	cases = append(cases, failureCase{6, fakecommander.WrongDataType}, failureCase{6, fakecommander.CountTooLarge})
	for _, testCase := range cases {
		harness := newHarness(test)
		controller := harness.fake.Add(9, 10)
		controller.Fail(testCase.step, testCase.failure)
		if got := harness.read(); got != nil {
			test.Fatalf("step %d, failure %d: got %v", testCase.step, testCase.failure, got)
		}
		var want []int
		for step := 1; step <= testCase.step; step++ {
			want = append(want, step)
		}
		want = append(want, 8)
		if writes := controller.Writes(); !reflect.DeepEqual(writes, sequence(want...)) {
			test.Fatalf("step %d, failure %d: got %d writes", testCase.step, testCase.failure, len(writes))
		}
		if controller.Mode() != fakecommander.HardwareMode {
			test.Fatalf("step %d, failure %d: not in hardware mode", testCase.step, testCase.failure)
		}
	}
}

func TestAFailedStepEightKeepsTheReadings(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	controller.Fail(8, fakecommander.NoAnswer)
	if got := harness.read(); !reflect.DeepEqual(got, bothProbes) {
		test.Fatalf("got %v", got)
	}
	if writes := controller.Writes(); len(writes) != 8 {
		test.Fatalf("got %d writes", len(writes))
	}
}

func TestStopBeforeTheReadingWritesNothing(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	harness.cancel()
	if got := harness.read(); got != nil {
		test.Fatalf("got %v", got)
	}
	if writes := controller.Writes(); len(writes) != 0 {
		test.Fatalf("got %d writes", len(writes))
	}
}

func TestStopDuringTheSequenceWritesStepEight(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	controller.OnStep(3, harness.cancel)
	if got := harness.read(); got != nil {
		test.Fatalf("got %v", got)
	}
	if writes := controller.Writes(); !reflect.DeepEqual(writes, sequence(1, 2, 3, 8)) {
		test.Fatalf("got %d writes", len(writes))
	}
	if controller.Mode() != fakecommander.HardwareMode {
		test.Fatal("the controller is not in hardware mode")
	}
}

func TestNothingWantedOpensNoNode(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	var asked []string
	harness.wantedBy = func(name string) bool {
		asked = append(asked, name)
		return false
	}
	if got := harness.read(); got != nil {
		test.Fatalf("got %v", got)
	}
	if writes := controller.Writes(); len(writes) != 0 {
		test.Fatalf("got %d writes", len(writes))
	}
	if len(asked) == 0 {
		test.Fatal("wanted was not asked")
	}
	for _, name := range asked {
		if !sensors.IsName(name) || !strings.HasPrefix(name, "commander.") {
			test.Fatalf("wanted was asked for %q", name)
		}
	}
}

func TestAStopDoneAtTheStartOfTheSequenceWritesNothing(test *testing.T) {
	harness := newHarness(test)
	controller := harness.fake.Add(9, 10)
	file, err := hidraw.OpenNode(filepath.Join(harness.devRoot, "hidraw9"))
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	harness.cancel()
	if got := runSequence(harness.stop, hidrawNode{file: file}, testTimeLimit); got != nil {
		test.Fatalf("got %v", got)
	}
	if writes := controller.Writes(); len(writes) != 0 {
		test.Fatalf("got %d writes", len(writes))
	}
	if controller.Mode() != fakecommander.NoMode {
		test.Fatal("a mode command was written")
	}
}
