//go:build linux

package linuxsensors

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakesensors"
	"github.com/norareit/sensordeck/internal/sensors"
)

// utilisation makes the source of a tree with a wait that records its duration and writes the next content.
type utilisation struct {
	source  *cpuUtilisation
	waits   []time.Duration
	changes []string
}

func newUtilisation(test *testing.T, tree *fakesensors.Tree, first string) *utilisation {
	test.Helper()
	tree.Stat(first)
	result := &utilisation{}
	result.source = &cpuUtilisation{procRoot: tree.ProcRoot, wait: func(duration time.Duration) {
		result.waits = append(result.waits, duration)
		if len(result.changes) > 0 {
			tree.Stat(result.changes[0])
			result.changes = result.changes[1:]
		}
	}}
	return result
}

func readUtilisation(source *cpuUtilisation) []sensors.Reading {
	return source.Read(context.Background(), everything)
}

func TestFirstReadingTakesTwoSamplesAndWaitsOnce(test *testing.T) {
	tree := newTree(test)
	run := newUtilisation(test, tree, "cpu  100 0 100 700 100 0 0 0 5 5\n")
	run.changes = []string{"cpu  200 0 100 750 150 0 0 0 5 5\n"}
	// growth of the total 200 and of the idle time 100: half busy
	want := []sensors.Reading{{Name: "cpu.util", Value: 50}}
	if got := readUtilisation(run.source); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
	if !reflect.DeepEqual(run.waits, []time.Duration{200 * time.Millisecond}) {
		test.Errorf("waits %v", run.waits)
	}
}

func TestSecondReadingUsesTheKeptSampleAndDoesNotWait(test *testing.T) {
	tree := newTree(test)
	run := newUtilisation(test, tree, "cpu  100 0 100 700 100 0 0 0 0 0\n")
	run.changes = []string{"cpu  200 0 100 750 150 0 0 0 0 0\n"}
	readUtilisation(run.source)
	tree.Stat("cpu  300 0 100 800 200 0 0 0 0 0\n") // growth 200, idle 100 against the kept sample
	want := []sensors.Reading{{Name: "cpu.util", Value: 50}}
	if got := readUtilisation(run.source); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
	if len(run.waits) != 1 {
		test.Errorf("waits %v", run.waits)
	}
}

func TestNoGrowthGivesNothing(test *testing.T) {
	tree := newTree(test)
	run := newUtilisation(test, tree, "cpu  1 1 1 1 1 1 1 1 0 0\n")
	if got := readUtilisation(run.source); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestMalformedLinesGiveNothing(test *testing.T) {
	for _, content := range []string{"cpu  1 1 1 1 1 1 1\n", "cpu  1 1 1x 1 1 1 1 1\n", "cpu  1 1 1 1 1 1 1 1 bad 0\n", "cpu  1 1 1 1 1 1 1 1 0 -1\n", "cpu0 1 1 1 1 1 1 1 1\n", ""} {
		tree := newTree(test)
		run := newUtilisation(test, tree, content)
		if got := readUtilisation(run.source); len(got) != 0 {
			test.Errorf("%q gave %v", content, got)
		}
		if len(run.waits) != 0 {
			test.Errorf("%q waited", content)
		}
	}
}

func TestValueIsClampedTo100(test *testing.T) {
	tree := newTree(test)
	run := newUtilisation(test, tree, "cpu  100 0 0 1000 0 0 0 0 0 0\n")
	// the idle time went back, so that the busy share is above the total
	run.changes = []string{"cpu  300 0 0 900 0 0 0 0 0 0\n"}
	if got := readUtilisation(run.source); len(got) != 1 || got[0].Value != 100 {
		test.Errorf("got %v", got)
	}
}

func TestMissingFileGivesNothingAndDoesNotWait(test *testing.T) {
	waited := false
	source := &cpuUtilisation{procRoot: filepath.Join(test.TempDir(), "none"), wait: func(time.Duration) { waited = true }}
	if got := readUtilisation(source); len(got) != 0 || waited {
		test.Errorf("got %v, waited %v", got, waited)
	}
}

func TestUnwantedUtilisationReadsNothingAndDoesNotWait(test *testing.T) {
	tree := newTree(test)
	run := newUtilisation(test, tree, "cpu  1 1 1 1 1 1 1 1 0 0\n")
	got := run.source.Read(context.Background(), func(name string) bool { return name != "cpu.util" })
	if len(got) != 0 || len(run.waits) != 0 || run.source.previous != nil {
		test.Errorf("got %v, waits %v", got, run.waits)
	}
}

func TestParseStatSums(test *testing.T) {
	got, ok := parseStat("intr 5\ncpu  1 2 3 4 5 6 7 8 9 10\n")
	if !ok || got != (sample{total: 36, idle: 9}) {
		test.Errorf("got %+v %v", got, ok)
	}
}
