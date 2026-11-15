package windowssensors

import (
	"time"

	"github.com/norareit/sensordeck/internal/nvidiasmi"
	"github.com/norareit/sensordeck/internal/sensors"
)

// Sources gives the sources of the metrics that Windows gives, in the order of the table of metrics: the CPU, the GPU,
// the memory and the network.
func Sources(seam Seam) []sensors.Source {
	return []sensors.Source{
		NewCPU(seam, time.Sleep),
		nvidiasmi.Source{TimeLimit: nvidiasmi.TimeLimit},
		NewMemory(seam),
		NewNetwork(seam, time.Sleep, time.Now),
	}
}
