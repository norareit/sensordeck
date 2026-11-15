package cli

import (
	"slices"

	"github.com/norareit/sensordeck/internal/sensors"
)

// sourceSet is the sources of the readings, with the controller apart, because run reads it less often than the
// others.
type sourceSet struct {
	others     []sensors.Source
	controller sensors.Source
}

// all gives every source, the controller last, in the order in which sensors reads them.
func (set sourceSet) all() []sensors.Source {
	return append(slices.Clip(set.others), set.controller)
}
