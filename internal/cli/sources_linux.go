//go:build linux

package cli

import (
	"github.com/norareit/sensordeck/internal/linuxsensors"
	"github.com/norareit/sensordeck/internal/sensors"
)

// This file and sources_other.go do nothing but choose the sources of the readings.

func newSources(sysfsRoot, procRoot string) ([]sensors.Source, error) {
	return linuxsensors.Sources(sysfsRoot, procRoot), nil
}
