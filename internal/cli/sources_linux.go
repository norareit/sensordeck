//go:build linux

package cli

import (
	"context"
	"errors"

	"github.com/norareit/sensordeck/internal/commandercore"
	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/linuxsensors"
	"github.com/norareit/sensordeck/internal/sensors"
)

// This file and sources_other.go do nothing but choose the sources of the readings.

func newSources(sysfsRoot, devRoot, procRoot string) (sourceSet, error) {
	controller := commandercore.Source{SysfsRoot: sysfsRoot, DevRoot: devRoot, TimeLimit: commandercore.ReplyTimeLimit}
	startSession := func(stop context.Context, colour config.Colour) (heldSession, error) {
		lighting := commandercore.Lighting{
			SysfsRoot: sysfsRoot, DevRoot: devRoot, TimeLimit: commandercore.ReplyTimeLimit,
			Red: colour.Red, Green: colour.Green, Blue: colour.Blue,
		}
		session, err := lighting.Start(stop)
		if errors.Is(err, commandercore.ErrNoController) {
			return nil, errNoController
		}
		if err != nil {
			return nil, err
		}
		return commanderSession{session: session}, nil
	}
	return sourceSet{others: linuxsensors.Sources(sysfsRoot, procRoot), controller: controller, startSession: startSession}, nil
}

// commanderSession is the session of the driver as the keeper holds it.
type commanderSession struct{ session *commandercore.Session }

func (held commanderSession) read(stop context.Context) ([]sensors.Reading, error) {
	return held.session.Read(stop)
}

func (held commanderSession) end() { held.session.End() }
