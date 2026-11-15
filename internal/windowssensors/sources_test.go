package windowssensors_test

import (
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/fakewinsensors"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/windowssensors"
)

func TestSourcesAreFourAndTheMemoryOneReadsTheSeam(test *testing.T) {
	fake := &fakewinsensors.Fake{Total: 4 * gibibyte, Available: 3 * gibibyte}
	sources := windowssensors.Sources(fake)
	if len(sources) != 4 {
		test.Fatalf("%d sources", len(sources))
	}
	got := sensors.ReadAll(test.Context(), sources[2:3], only("ram.total"))
	if !reflect.DeepEqual(got, []sensors.Reading{{Name: "ram.total", Value: 4}}) {
		test.Errorf("got %v", got)
	}
}
