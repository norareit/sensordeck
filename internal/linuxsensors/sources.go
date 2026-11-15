//go:build linux

package linuxsensors

import (
	"time"

	"github.com/norareit/sensordeck/internal/nvidiasmi"
	"github.com/norareit/sensordeck/internal/sensors"
)

// Sources gives the sources of every metric, in the order of the table of metrics.
func Sources(sysfsRoot, procRoot string) []sensors.Source {
	return []sensors.Source{
		cpuTemperatures{sysfsRoot: sysfsRoot},
		newCPUUtilisation(procRoot, time.Sleep),
		cpuClock{sysfsRoot: sysfsRoot},
		nvidiasmi.Source{TimeLimit: nvidiasmi.TimeLimit},
		memory{procRoot: procRoot},
		memoryModules(sysfsRoot),
		drives(sysfsRoot),
		newNetwork(sysfsRoot, procRoot, time.Sleep, time.Now),
	}
}
