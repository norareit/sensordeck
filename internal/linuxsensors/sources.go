//go:build linux

package linuxsensors

import (
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// gpuTimeLimit leaves room for a slow first start of the driver and bounds a program that hangs.
const gpuTimeLimit = 2 * time.Second

// Sources gives the sources of every metric, in the order of the table of metrics.
func Sources(sysfsRoot, procRoot string) []sensors.Source {
	return []sensors.Source{
		cpuTemperatures{sysfsRoot: sysfsRoot},
		newCPUUtilisation(procRoot, time.Sleep),
		cpuClock{sysfsRoot: sysfsRoot},
		gpu{timeLimit: gpuTimeLimit},
		memory{procRoot: procRoot},
		memoryModules(sysfsRoot),
		drives(sysfsRoot),
		newNetwork(sysfsRoot, procRoot, time.Sleep, time.Now),
	}
}
