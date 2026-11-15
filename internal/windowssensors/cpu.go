package windowssensors

import (
	"time"

	"github.com/norareit/sensordeck/internal/sampling"
	"github.com/norareit/sensordeck/internal/sensors"
)

// NewCPU gives the source of cpu.util. A sample is the kernel time plus the user time as the total, because the kernel
// time includes the idle time as the total of /proc/stat does on Linux, and the idle time. A failed call makes no
// sample.
func NewCPU(seam Seam, wait func(time.Duration)) sensors.Source {
	return &sampling.CPU{
		Take: func() (sampling.CPUSample, bool) {
			idle, kernel, user, err := seam.SystemTimes()
			if err != nil {
				return sampling.CPUSample{}, false
			}
			return sampling.CPUSample{Total: kernel + user, Idle: idle}, true
		},
		Wait: wait,
	}
}
