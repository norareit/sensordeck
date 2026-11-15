package fakewinsensors

import (
	"errors"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/windowssensors"
)

func TestTheMemoryIsGivenAndCounted(test *testing.T) {
	fake := &Fake{Total: 8, Available: 3}
	total, available, err := fake.Memory()
	if total != 8 || available != 3 || err != nil || fake.MemoryCalls() != 1 {
		test.Errorf("got %d %d %v, %d calls", total, available, err, fake.MemoryCalls())
	}
	failing := &Fake{MemoryError: errors.New("failed")}
	if _, _, err := failing.Memory(); err == nil {
		test.Error("no error")
	}
}

func TestTheTimesAreTakenOneForEachCallAndTheLastIsRepeated(test *testing.T) {
	fake := &Fake{TimesList: []Times{{1, 2, 3}, {4, 5, 6}}}
	var got [][3]uint64
	for range 3 {
		idle, kernel, user, err := fake.SystemTimes()
		if err != nil {
			test.Fatal(err)
		}
		got = append(got, [3]uint64{idle, kernel, user})
	}
	want := [][3]uint64{{1, 2, 3}, {4, 5, 6}, {4, 5, 6}}
	if !reflect.DeepEqual(got, want) || fake.TimesCalls() != 3 {
		test.Errorf("got %v, %d calls", got, fake.TimesCalls())
	}
}

func TestTheTablesAreTakenOneForEachCallAndAFailureIsGiven(test *testing.T) {
	first := []windowssensors.Interface{{LUID: 1}}
	second := []windowssensors.Interface{{LUID: 2}}
	fake := &Fake{Tables: [][]windowssensors.Interface{first, second}}
	var got [][]windowssensors.Interface
	for range 3 {
		rows, err := fake.Interfaces()
		if err != nil {
			test.Fatal(err)
		}
		got = append(got, rows)
	}
	if !reflect.DeepEqual(got, [][]windowssensors.Interface{first, second, second}) || fake.InterfaceCalls() != 3 {
		test.Errorf("got %v, %d calls", got, fake.InterfaceCalls())
	}
	failing := &Fake{InterfacesError: errors.New("failed")}
	if _, err := failing.Interfaces(); err == nil || failing.InterfaceCalls() != 1 {
		test.Errorf("error %v", err)
	}
}
