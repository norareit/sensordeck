//go:build linux

package linuxsensors

import "github.com/norareit/sensordeck/internal/sensors"

const fullLine = "44, 17, 3256, 32607"

func names(readings []sensors.Reading) []string {
	var result []string
	for _, reading := range readings {
		result = append(result, reading.Name)
	}
	return result
}
