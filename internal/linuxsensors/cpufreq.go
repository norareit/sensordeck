//go:build linux

package linuxsensors

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/norareit/sensordeck/internal/sensors"
)

// cpuClock reads cpu.clock as the highest clock of all CPUs in MHz.
type cpuClock struct{ sysfsRoot string }

func (source cpuClock) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wanted(sensors.CPUClock) {
		return nil
	}
	files, err := filepath.Glob(filepath.Join(source.sysfsRoot, "devices", "system", "cpu", "cpu*", "cpufreq", "scaling_cur_freq"))
	if err != nil {
		return nil
	}
	var highest uint64
	found := false
	for _, file := range files {
		cpu := filepath.Base(filepath.Dir(filepath.Dir(file)))
		content, err := os.ReadFile(file)
		if !isDigits(strings.TrimPrefix(cpu, "cpu")) || err != nil {
			continue
		}
		kilohertz, ok := parseKilohertz(string(content))
		if ok && (!found || kilohertz > highest) {
			highest, found = kilohertz, true
		}
	}
	if !found {
		return nil
	}
	return []sensors.Reading{{Name: sensors.CPUClock, Value: float64(highest) / 1000}}
}

func parseKilohertz(content string) (uint64, bool) {
	text := strings.TrimSuffix(content, "\n")
	if !isDigits(text) {
		return 0, false
	}
	kilohertz, err := strconv.ParseUint(text, 10, 64)
	return kilohertz, err == nil
}
