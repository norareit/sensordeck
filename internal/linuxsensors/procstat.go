//go:build linux

package linuxsensors

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// sampleInterval is how long a first reading waits between its two samples.
const sampleInterval = 200 * time.Millisecond

// sample is the counters of /proc/stat that matter: all time, and the idle time.
type sample struct{ total, idle uint64 }

// parseStat reads the first line that starts with "cpu" and a space. The total is the sum of the first eight
// counters, because guest and guest_nice are already counted in user and nice, and iowait counts as idle.
func parseStat(content string) (sample, bool) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		counters := strings.Fields(line)[1:]
		if len(counters) < 8 {
			return sample{}, false
		}
		var taken sample
		for position, text := range counters[:8] {
			if !isDigits(text) {
				return sample{}, false
			}
			counter, err := strconv.ParseUint(text, 10, 64)
			if err != nil {
				return sample{}, false
			}
			taken.total += counter
			if position == 3 || position == 4 {
				taken.idle += counter
			}
		}
		return taken, true
	}
	return sample{}, false
}

// cpuUtilisation reads cpu.util as the share of non-idle time between two samples. It keeps its previous sample from
// one reading to the next, so that only the first reading waits.
type cpuUtilisation struct {
	procRoot string
	wait     func(time.Duration)
	previous *sample
}

func (source *cpuUtilisation) take() (sample, bool) {
	content, err := os.ReadFile(filepath.Join(source.procRoot, "stat"))
	if err != nil {
		return sample{}, false
	}
	return parseStat(string(content))
}

func (source *cpuUtilisation) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wanted(sensors.CPUUtilisation) {
		return nil
	}
	if source.previous == nil {
		first, ok := source.take()
		if !ok {
			return nil
		}
		source.previous = &first
		source.wait(sampleInterval)
	}
	current, ok := source.take()
	if !ok {
		return nil
	}
	previous := *source.previous
	source.previous = &current
	// Differences as floats, so that a counter that went back cannot wrap.
	growth := float64(current.total) - float64(previous.total)
	if growth <= 0 {
		return nil
	}
	busy := growth - (float64(current.idle) - float64(previous.idle))
	return []sensors.Reading{{Name: sensors.CPUUtilisation, Value: min(max(100*busy/growth, 0), 100)}}
}
