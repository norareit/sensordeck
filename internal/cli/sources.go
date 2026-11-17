package cli

import (
	"context"
	"errors"
	"net/url"
	"slices"

	"github.com/norareit/sensordeck/internal/commandercore"
	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/librehardwaremonitor"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/windowssensors"
)

// sourceSet is the sources of the readings, with the controller apart, because run reads it less often than the
// others.
type sourceSet struct {
	others     []sensors.Source
	controller sensors.Source
	// monitor makes the source of the monitor metrics for the URL, which tells report the outcome of each reading. It
	// is nil where the monitor is not read, which is every system but Windows.
	monitor func(address *url.URL, report func(found []sensors.Reading, err error)) sensors.Source
	// startSession makes a setup of a session that lights the LEDs of the controller in the colour and gives the
	// session and what the setup did with the LED port configuration, which it gives on an error too, or
	// errNoController when there is no controller. On an error the setup has ended.
	startSession func(stop context.Context, colour config.Colour) (heldSession, setupNotice, error)
	// reportController makes the source of the controller with refused called with the error of an opening that
	// another sensordeck process holds. It is nil where nothing can be reported, and sensors uses it in place of
	// controller.
	reportController func(refused func(err error)) sensors.Source
	// controllerProblem gives why a controller that seems to be there was not found, or nil.
	controllerProblem func() error
}

// problem gives the problem of the controller, nil when the set has no way to tell one.
func (set sourceSet) problem() error {
	if set.controllerProblem == nil {
		return nil
	}
	return set.controllerProblem()
}

// all gives every source, the controller last, in the order in which sensors reads them.
func (set sourceSet) all() []sensors.Source {
	return append(slices.Clip(set.others), set.controller)
}

// commanderSources gives the sources with the controller read through the finder and no other source, which the
// system's own file completes.
func commanderSources(finder commandercore.Finder) sourceSet {
	startSession := func(stop context.Context, colour config.Colour) (heldSession, setupNotice, error) {
		lighting := commandercore.Lighting{
			Finder: finder, TimeLimit: commandercore.ReplyTimeLimit,
			Red: colour.Red, Green: colour.Green, Blue: colour.Blue,
		}
		session, configuration, err := lighting.Start(stop)
		notice := setupNotice{
			configured: configuration.Written, unknownPort: configuration.Unknown, unknownLEDs: configuration.LEDs,
		}
		if errors.Is(err, commandercore.ErrNoController) {
			return nil, notice, errNoController
		}
		if err != nil {
			return nil, notice, err
		}
		return commanderSession{session: session}, notice, nil
	}
	return sourceSet{
		controller: commandercore.Source{Finder: finder, TimeLimit: commandercore.ReplyTimeLimit},
		reportController: func(refused func(err error)) sensors.Source {
			return commandercore.Source{Finder: finder, TimeLimit: commandercore.ReplyTimeLimit, Refused: refused}
		},
		startSession:      startSession,
		controllerProblem: finder.Problem,
	}
}

// withWindowsSensors gives the set the sources of Windows: the native metrics and the GPU over the seam, and the monitor.
func withWindowsSensors(set sourceSet, seam windowssensors.Seam) sourceSet {
	set.others = windowssensors.Sources(seam)
	set.monitor = func(address *url.URL, report func(found []sensors.Reading, err error)) sensors.Source {
		return librehardwaremonitor.Source{
			Reader: librehardwaremonitor.NewReader(address, librehardwaremonitor.TimeLimit), Report: report,
		}
	}
	return set
}

// commanderSession is the session of the driver as the keeper holds it.
type commanderSession struct{ session *commandercore.Session }

func (held commanderSession) read(stop context.Context) ([]sensors.Reading, error) {
	return held.session.Read(stop)
}

func (held commanderSession) end() { held.session.End() }
