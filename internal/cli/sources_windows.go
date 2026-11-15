package cli

import (
	"github.com/norareit/sensordeck/internal/commandercore"
	"github.com/norareit/sensordeck/internal/windowssensors"
)

// This file and sources_linux.go do nothing but choose the sources of the readings. The roots of the tests are not used.

func newSources(sysfsRoot, devRoot, procRoot string) (sourceSet, error) {
	return withWindowsSensors(commanderSources(commandercore.Windows()), windowssensors.System()), nil
}
