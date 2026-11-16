package custommetrics

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
)

// Set is the source of run for the custom metrics. Only one goroutine calls Read, Replace and Close, so the state
// needs no lock. The runs of commands are the other goroutines, and each gives its outcome through a channel.
type Set struct {
	sysfsRoot string
	directory string
	now       func() time.Time
	stderr    io.Writer

	lifetime context.Context
	stop     context.CancelFunc
	runs     sync.WaitGroup
	metrics  []*tracked
}

// tracked is a metric with what is known of its readings.
type tracked struct {
	definition config.Metric
	value      float64
	hasValue   bool
	streak     streak
	lastStart  time.Time
	started    bool
	running    *run
}

// run is a run of a command in progress.
type run struct {
	cancel context.CancelFunc
	done   chan outcome // holds the outcome of a run that finished, and is never waited for
}

// NewSet makes the set of the definitions. A command runs in the directory, and the chips are looked for under the
// sysfs root.
func NewSet(definitions []config.Metric, sysfsRoot, directory string, now func() time.Time, stderr io.Writer) *Set {
	lifetime, stop := context.WithCancel(context.Background())
	set := &Set{sysfsRoot: sysfsRoot, directory: directory, now: now, stderr: stderr, lifetime: lifetime, stop: stop}
	for _, definition := range definitions {
		set.metrics = append(set.metrics, &tracked{definition: definition})
	}
	return set
}

// Read gives the value of each wanted metric that has one, in the order of the file. A command is never waited for.
func (set *Set) Read(_ context.Context, wanted func(name string) bool) []sensors.Reading {
	var readings []sensors.Reading
	for _, each := range set.metrics {
		if !wanted(each.definition.Name) {
			continue
		}
		set.refresh(each)
		if each.hasValue {
			readings = append(readings, sensors.Reading{Name: each.definition.Name, Value: each.value})
		}
	}
	return readings
}

func (set *Set) refresh(each *tracked) {
	definition := each.definition
	switch {
	case definition.File != "":
		set.take(each, readFile(definition))
	case definition.Hwmon != "":
		set.take(each, readHwmon(set.sysfsRoot, definition))
	default:
		set.pollCommand(each)
		set.startCommand(each)
	}
}

// take keeps the outcome as the state of the metric. A failure clears the value, so that nothing presents an old
// value as current.
func (set *Set) take(each *tracked, result outcome) {
	each.streak.observe(set.stderr, each.definition.Name, result)
	each.value, each.hasValue = result.value, !result.failed()
}

func (set *Set) pollCommand(each *tracked) {
	if each.running == nil {
		return
	}
	select {
	case result := <-each.running.done:
		each.running = nil
		set.take(each, scaled(result, each.definition))
	default:
	}
}

func (set *Set) startCommand(each *tracked) {
	now := set.now()
	if each.running != nil || (each.started && now.Sub(each.lastStart) < each.definition.Every) {
		return
	}
	each.started, each.lastStart = true, now
	runContext, cancel := context.WithCancel(set.lifetime)
	started := &run{cancel: cancel, done: make(chan outcome, 1)}
	each.running = started
	definition := each.definition
	set.runs.Add(1)
	go func() {
		defer set.runs.Done()
		defer cancel()
		if result, finished := runCommand(runContext, definition.Command, set.directory, timeLimit(definition)); finished {
			started.done <- result
		}
	}()
}

// Replace makes the metrics of the new definitions. A metric whose definition is unchanged keeps its value, its streak
// and its run in progress. A run of a definition that changed or was removed is killed and its outcome dropped.
func (set *Set) Replace(definitions []config.Metric) {
	var next []*tracked
	kept := map[*tracked]bool{}
	for _, definition := range definitions {
		next = append(next, set.keptOrNew(definition, kept))
	}
	for _, old := range set.metrics {
		if !kept[old] && old.running != nil {
			old.running.cancel()
		}
	}
	set.metrics = next
}

func (set *Set) keptOrNew(definition config.Metric, kept map[*tracked]bool) *tracked {
	for _, old := range set.metrics {
		if old.definition == definition {
			kept[old] = true
			return old
		}
	}
	return &tracked{definition: definition}
}

// Close kills the runs in progress and waits for them to end.
func (set *Set) Close() {
	set.stop()
	set.runs.Wait()
}
