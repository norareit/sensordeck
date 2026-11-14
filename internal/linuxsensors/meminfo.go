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

const kibibytesPerGibibyte = 1048576

// parseMeminfo gives the value in KiB of the line "<key>:", which must have exactly a number and the unit kB.
func parseMeminfo(content string, key string) (uint64, bool) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != key+":" {
			continue
		}
		if len(fields) != 3 || fields[2] != "kB" || !isDigits(fields[1]) {
			return 0, false
		}
		kibibytes, err := strconv.ParseUint(fields[1], 10, 64)
		return kibibytes, err == nil
	}
	return 0, false
}

// memory reads ram.used, as the total minus what is available, and ram.total, both in GiB.
type memory struct{ procRoot string }

func (source memory) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wanted(sensors.RAMUsed) && !wanted(sensors.RAMTotal) {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(source.procRoot, "meminfo"))
	if err != nil {
		return nil
	}
	total, haveTotal := parseMeminfo(string(content), "MemTotal")
	if !haveTotal {
		return nil
	}
	var readings []sensors.Reading
	if available, haveAvailable := parseMeminfo(string(content), "MemAvailable"); haveAvailable && wanted(sensors.RAMUsed) {
		used := float64(total) - float64(available)
		readings = append(readings, sensors.Reading{Name: sensors.RAMUsed, Value: used / kibibytesPerGibibyte})
	}
	if wanted(sensors.RAMTotal) {
		readings = append(readings, sensors.Reading{Name: sensors.RAMTotal, Value: float64(total) / kibibytesPerGibibyte})
	}
	return readings
}
