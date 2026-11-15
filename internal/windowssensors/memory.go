package windowssensors

import (
	"context"

	"github.com/norareit/sensordeck/internal/sensors"
)

const bytesPerGibibyte = 1 << 30

// memory reads ram.used, ram.total and ram.util from one call.
type memory struct{ seam Seam }

// NewMemory gives the source of the three memory metrics. ram.used is the total minus the available part, as on Linux,
// and ram.util is computed from the two and not clamped. dwMemoryLoad is not used, because it is a whole percentage.
func NewMemory(seam Seam) sensors.Source { return memory{seam: seam} }

func (source memory) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wanted(sensors.RAMUsed) && !wanted(sensors.RAMTotal) && !wanted(sensors.RAMUtilisation) {
		return nil
	}
	total, available, err := source.seam.Memory()
	if err != nil || total == 0 || available > total {
		return nil
	}
	used := float64(total-available) / bytesPerGibibyte
	totalGibibytes := float64(total) / bytesPerGibibyte
	var readings []sensors.Reading
	if wanted(sensors.RAMUsed) {
		readings = append(readings, sensors.Reading{Name: sensors.RAMUsed, Value: used})
	}
	if wanted(sensors.RAMTotal) {
		readings = append(readings, sensors.Reading{Name: sensors.RAMTotal, Value: totalGibibytes})
	}
	if wanted(sensors.RAMUtilisation) {
		readings = append(readings, sensors.Reading{Name: sensors.RAMUtilisation, Value: 100 * used / totalGibibytes})
	}
	return readings
}
