package custommetrics

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/fakesensors"
)

func TestReadOnceReadsTheThreeFormsInTheOrderOfTheFile(test *testing.T) {
	tree := fakesensors.New(test, test.TempDir())
	writeFile(test, filepath.Join(tree.Hwmon(1, "nct6799", "pci/a"), "fan2_input"), "900\n")
	file := writeFile(test, filepath.Join(test.TempDir(), "t"), "40000\n")
	definitions := []config.Metric{
		commandMetric("c.c", "sleep 0.3; echo 2", time.Minute),
		{Name: "h.h", Hwmon: "nct6799", Input: "fan2_input", Index: 1, Scale: 1, Unit: "RPM"},
		{Name: "f.f", File: file, Scale: 0.001, Unit: "°C"},
		commandMetric("bad.bad", "exit 1", time.Minute),
	}
	results := ReadOnce(context.Background(), definitions, tree.SysfsRoot, "/")
	want := []Result{
		{Name: "c.c", Value: 2},
		{Name: "h.h", Unit: "RPM", Value: 900},
		{Name: "f.f", Unit: "°C", Value: 40},
		{Name: "bad.bad", Cause: "the command exited with status 1"},
	}
	if len(results) != len(want) {
		test.Fatalf("results %+v", results)
	}
	for position := range want {
		if results[position] != want[position] {
			test.Errorf("result %d = %+v, want %+v", position, results[position], want[position])
		}
	}
}

func TestReadOnceLeavesOutARunThatStopEnded(test *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	results := ReadOnce(stop, []config.Metric{commandMetric("c.c", "sleep 5", time.Minute)}, "", "/")
	if len(results) != 0 {
		test.Errorf("results %+v", results)
	}
}
