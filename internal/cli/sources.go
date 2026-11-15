package cli

import (
	"context"
	"errors"
	"slices"

	"github.com/norareit/sensordeck/internal/commandercore"
	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
)

// sourceSet is the sources of the readings, with the controller apart, because run reads it less often than the
// others.
type sourceSet struct {
	others     []sensors.Source
	controller sensors.Source
	// startSession makes a setup of a session that lights the LEDs of the controller in the colour and gives the
	// session, or errNoController when there is no controller. On an error the setup has ended.
	startSession func(stop context.Context, colour config.Colour) (heldSession, error)
}

// all gives every source, the controller last, in the order in which sensors reads them.
func (set sourceSet) all() []sensors.Source {
	return append(slices.Clip(set.others), set.controller)
}

// commanderSources gives the sources with the controller read through the finder and no other source, which the
// system's own file completes.
func commanderSources(finder commandercore.Finder) sourceSet {
	startSession := func(stop context.Context, colour config.Colour) (heldSession, error) {
		lighting := commandercore.Lighting{
			Finder: finder, TimeLimit: commandercore.ReplyTimeLimit,
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
	return sourceSet{
		controller:   commandercore.Source{Finder: finder, TimeLimit: commandercore.ReplyTimeLimit},
		startSession: startSession,
	}
}

// commanderSession is the session of the driver as the keeper holds it.
type commanderSession struct{ session *commandercore.Session }

func (held commanderSession) read(stop context.Context) ([]sensors.Reading, error) {
	return held.session.Read(stop)
}

func (held commanderSession) end() { held.session.End() }
