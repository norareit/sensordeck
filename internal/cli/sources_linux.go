//go:build linux

package cli

import (
	"github.com/norareit/sensordeck/internal/commandercore"
	"github.com/norareit/sensordeck/internal/linuxsensors"
	"github.com/norareit/sensordeck/internal/sensors"
)

// This file and sources_other.go do nothing but choose the sources of the readings.

func newSources(sysfsRoot, devRoot, procRoot string) ([]sensors.Source, error) {
	controller := commandercore.Source{SysfsRoot: sysfsRoot, DevRoot: devRoot, TimeLimit: commandercore.ReplyTimeLimit}
	return append(linuxsensors.Sources(sysfsRoot, procRoot), controller), nil
}
