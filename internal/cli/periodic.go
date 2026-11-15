package cli

import (
	"context"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// periodicSource reads a source at most once per period and gives the readings of its last reading in between.
type periodicSource struct {
	source   sensors.Source
	period   time.Duration
	read     bool              // whether a reading was made
	lastRead time.Time         // the start of the round that made the last reading
	kept     []sensors.Reading // exactly what the last reading gave
}

// readings reads the source when no reading was made yet or when roundStart is at least the period after the start
// of the round that made the last one, and gives the kept readings. What a reading gave replaces the kept readings as
// a whole, even when it gave nothing, so that a reading that failed counts as a reading and the next one comes a
// full period later.
func (periodic *periodicSource) readings(stop context.Context, roundStart time.Time, wanted func(name string) bool) []sensors.Reading {
	if !periodic.read || roundStart.Sub(periodic.lastRead) >= periodic.period {
		periodic.kept = periodic.source.Read(stop, wanted)
		periodic.lastRead = roundStart
		periodic.read = true
	}
	return periodic.kept
}
