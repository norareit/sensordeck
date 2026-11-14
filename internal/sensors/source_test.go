package sensors

import (
	"context"
	"reflect"
	"testing"
)

type recordingSource struct {
	readings []Reading
	stops    []context.Context
	asked    []string
}

func (source *recordingSource) Read(stop context.Context, wanted func(name string) bool) []Reading {
	source.stops = append(source.stops, stop)
	for _, reading := range source.readings {
		if wanted(reading.Name) {
			source.asked = append(source.asked, reading.Name)
		}
	}
	return source.readings
}

func TestReadAllJoinsInOrderAndPassesWantedAndStop(test *testing.T) {
	first := &recordingSource{readings: []Reading{{Name: "cpu.temp", Value: 1}}}
	second := &recordingSource{readings: []Reading{{Name: "ram.used", Value: 2}, {Name: "ram.total", Value: 3}}}
	type key struct{}
	stop := context.WithValue(context.Background(), key{}, 1)
	got := ReadAll(stop, []Source{first, second}, func(name string) bool { return name != "ram.total" })
	want := []Reading{{"cpu.temp", 1}, {"ram.used", 2}, {"ram.total", 3}}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
	if first.stops[0] != stop || second.stops[0] != stop {
		test.Error("stop was not passed on")
	}
	if !reflect.DeepEqual(second.asked, []string{"ram.used"}) {
		test.Errorf("wanted: %v", second.asked)
	}
	if got := ReadAll(stop, nil, nil); len(got) != 0 {
		test.Errorf("no sources gave %v", got)
	}
}
