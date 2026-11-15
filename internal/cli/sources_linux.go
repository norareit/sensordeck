//go:build linux

package cli

import (
	"github.com/norareit/sensordeck/internal/commandercore"
	"github.com/norareit/sensordeck/internal/linuxsensors"
)

// This file and sources_other.go do nothing but choose the sources of the readings.

func newSources(sysfsRoot, devRoot, procRoot string) (sourceSet, error) {
	controller := commandercore.Source{SysfsRoot: sysfsRoot, DevRoot: devRoot, TimeLimit: commandercore.ReplyTimeLimit}
	return sourceSet{others: linuxsensors.Sources(sysfsRoot, procRoot), controller: controller}, nil
}
