//go:build linux

package linuxsensors

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/sampling"
)

// parseStat reads the first line that starts with "cpu" and a space. The total is the sum of the first eight
// counters, and every counter must be a whole number, because guest and guest_nice are already counted in user and nice, and iowait counts as idle.
func parseStat(content string) (sampling.CPUSample, bool) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		counters := strings.Fields(line)[1:]
		if len(counters) < 8 {
			return sampling.CPUSample{}, false
		}
		var taken sampling.CPUSample
		for position, text := range counters {
			counter, err := strconv.ParseUint(text, 10, 64)
			if err != nil || !isDigits(text) {
				return sampling.CPUSample{}, false
			}
			if position >= 8 {
				continue
			}
			taken.Total += counter
			if position == 3 || position == 4 {
				taken.Idle += counter
			}
		}
		return taken, true
	}
	return sampling.CPUSample{}, false
}

// newCPUUtilisation reads cpu.util from /proc/stat under procRoot, with the share computed by internal/sampling.
func newCPUUtilisation(procRoot string, wait func(time.Duration)) *sampling.CPU {
	return &sampling.CPU{
		Take: func() (sampling.CPUSample, bool) {
			content, err := os.ReadFile(filepath.Join(procRoot, "stat"))
			if err != nil {
				return sampling.CPUSample{}, false
			}
			return parseStat(string(content))
		},
		Wait: wait,
	}
}
