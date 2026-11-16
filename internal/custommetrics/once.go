package custommetrics

import (
	"context"
	"sync"

	"github.com/norareit/sensordeck/internal/config"
)

// Result is the reading of one metric by ReadOnce. The cause is empty when the metric has a value.
type Result struct {
	Name, Unit string
	Value      float64
	Cause      string
}

// ReadOnce reads each metric once, in the order of the definitions. Every command is run, concurrently, and waited for
// up to its time limit. A metric whose run stop ended is left out.
func ReadOnce(stop context.Context, definitions []config.Metric, sysfsRoot, directory string) []Result {
	outcomes := make([]*outcome, len(definitions))
	var runs sync.WaitGroup
	for position, definition := range definitions {
		runs.Add(1)
		go func() {
			defer runs.Done()
			outcomes[position] = readOnce(stop, definition, sysfsRoot, directory)
		}()
	}
	runs.Wait()
	var results []Result
	for position, result := range outcomes {
		if result == nil {
			continue
		}
		definition := definitions[position]
		results = append(results, Result{Name: definition.Name, Unit: definition.Unit, Value: result.value, Cause: result.cause})
	}
	return results
}

func readOnce(stop context.Context, definition config.Metric, sysfsRoot, directory string) *outcome {
	var result outcome
	switch {
	case definition.File != "":
		result = readFile(definition)
	case definition.Hwmon != "":
		result = readHwmon(sysfsRoot, definition)
	default:
		ran, finished := runCommand(stop, definition.Command, directory, timeLimit(definition))
		if !finished {
			return nil
		}
		result = scaled(ran, definition)
	}
	return &result
}
