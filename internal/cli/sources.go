package cli

import (
	"context"
	"slices"

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
