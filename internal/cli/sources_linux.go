//go:build linux

package cli

import (
	"github.com/norareit/sensordeck/internal/commandercore"
	"github.com/norareit/sensordeck/internal/linuxsensors"
)

// This file and sources_other.go do nothing but choose the sources of the readings.

func newSources(sysfsRoot, devRoot, procRoot string) (sourceSet, error) {
	set := commanderSources(commandercore.Hidraw(sysfsRoot, devRoot))
	set.others = linuxsensors.Sources(sysfsRoot, procRoot)
	return set, nil
}
