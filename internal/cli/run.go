package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// clockPeriod is how often the clock of the screen is set again. The screen keeps time itself, so this only corrects
// drift.
const clockPeriod = time.Minute

func (current *session) runCommand() *cobra.Command {
	var configPath string
	command := &cobra.Command{
		Use:   "run",
		Short: "Send the sensor values named in the config file to the screen until stopped",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			path, err := current.configPath(configPath)
			if err != nil {
				return err
			}
			content, err := readNamedFile("config file", path)
			if err != nil {
				return err
			}
			settings, err := config.Parse(content)
			if err != nil {
				return fmt.Errorf("config file %s: %w", path, err)
			}
			sources, err := current.env.newSources(current.env.sysfsRoot, current.env.devRoot, current.env.procRoot)
			if err != nil {
				return err
			}
			stop, cancel := current.env.notifyStop(context.Background())
			defer cancel()
			// The keeper and the feeder write their lines from two goroutines, and so may the trace.
			current.env.stderr = &lockedWriter{writer: current.env.stderr}
			opened, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			current.say(fmt.Sprintf("running: %d metrics every %s", len(settings.Mappings), settings.Interval))
			feeding := &feeder{
				transport:  opened,
				others:     sources.others,
				controller: &periodicSource{source: sources.controller, period: settings.CommanderPeriod()},
				config:     settings,
				now:        current.env.now,
				wait:       current.env.wait,
				stderr:     current.env.stderr,
			}
			endKeeping := func() {}
			if settings.CommanderColour != nil {
				keeper := current.newKeeper(sources, settings)
				keeping, cancelKeeping := context.WithCancel(stop)
				go keeper.keep(keeping)
				feeding.controller = keeper
				// The teardown of the controller comes before the end of run, on a stop and on a feeding error.
				endKeeping = func() {
					cancelKeeping()
					<-keeper.done
				}
			}
			feedErr := feeding.feed(stop)
			endKeeping()
			closeTransport()
			if feedErr != nil {
				return feedErr
			}
			current.say("stopped")
			return nil
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "read the config from this file instead of the default place")
	return command
}

// newKeeper makes the keeper of the session that lights the LEDs in the colour of the config.
func (current *session) newKeeper(sources sourceSet, settings config.Config) *sessionKeeper {
	probesMapped := false
	for _, mapping := range settings.Mappings {
		probesMapped = probesMapped || sensors.IsCommanderTemperature(mapping.Name)
	}
	start := func(stop context.Context) (heldSession, error) {
		return sources.startSession(stop, *settings.CommanderColour)
	}
	return newSessionKeeper(start, probesMapped, current.env.now, current.env.after, current.env.stderr)
}

// configPath is the file named with --config, or the config file of the user.
func (current *session) configPath(named string) (string, error) {
	if named != "" {
		return named, nil
	}
	directory, err := current.env.userConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot find the config directory, name the file with --config: %w", err)
	}
	return filepath.Join(directory, "sensordeck", "config.yaml"), nil
}

// controllerReader gives the readings of the controller for a round that started at roundStart.
type controllerReader interface {
	readings(stop context.Context, roundStart time.Time, wanted func(name string) bool) []sensors.Reading
}

// lockedWriter lets two goroutines write to one writer.
type lockedWriter struct {
	mutex  sync.Mutex
	writer io.Writer
}

func (locked *lockedWriter) Write(data []byte) (int, error) {
	locked.mutex.Lock()
	defer locked.mutex.Unlock()
	return locked.writer.Write(data)
}

// feeder sends the clock and the values of the mapped metrics to the screen in rounds, and reopens the transport
// when the screen is lost. Without commander_color it reads the controller at most once per period and sends the kept
// readings of its last reading in the other rounds. With it the controller is read through the keeper, which holds a
// session of the controller, and a round with a probe mapped asks it for a reading of its own.
type feeder struct {
	transport  transport
	others     []sensors.Source
	controller controllerReader
	config     config.Config
	now        func() time.Time
	wait       func(stop context.Context, duration time.Duration) bool
	stderr     io.Writer
	lastClock  time.Time
	lost       bool
}

// feed works until stop is done, and gives an error only when the clock cannot be encoded.
func (feeding *feeder) feed(stop context.Context) error {
	if _, err := feeding.writeClock(stop); err != nil {
		return err
	}
	first := feeding.now()
	interval := feeding.config.Interval
	index := 0
	for stop.Err() == nil {
		if err := feeding.round(stop); err != nil {
			return err
		}
		end := feeding.now()
		// A round that ran into the next one is followed at once, and the rounds missed are not made up.
		index = max(index+1, int(end.Sub(first)/interval))
		next := first.Add(time.Duration(index) * interval)
		if next.After(end) && !feeding.wait(stop, next.Sub(end)) {
			return nil
		}
	}
	return nil
}

// round does one round. The age of the clock is measured at the time the round actually starts, because a round
// that follows a long one starts after its place on the grid.
func (feeding *feeder) round(stop context.Context) error {
	start := feeding.now()
	if feeding.lost {
		if feeding.transport.Reopen() != nil {
			return nil
		}
		feeding.lost = false
		_, _ = fmt.Fprintln(feeding.stderr, "screen back")
		if written, err := feeding.writeClock(stop); !written || err != nil {
			return err
		}
	} else if start.Sub(feeding.lastClock) >= clockPeriod {
		if written, err := feeding.writeClock(stop); !written || err != nil {
			return err
		}
	}
	for _, report := range smartmonitor.EncodeValues(feeding.read(stop, start)) {
		if !feeding.write(stop, report) {
			return nil
		}
	}
	return nil
}

// read gives the pairs of the mapped metrics that have a value, in the order of their tags. The other sources are
// read now. The controller gives the readings of its last reading, which is made when the period has passed since the
// start of the round that made the one before.
func (feeding *feeder) read(stop context.Context, start time.Time) []smartmonitor.Value {
	mapped := make(map[string]bool, len(feeding.config.Mappings))
	for _, mapping := range feeding.config.Mappings {
		mapped[mapping.Name] = true
	}
	readings := make(map[string]float64, len(mapped))
	wanted := func(name string) bool { return mapped[name] }
	for _, reading := range sensors.ReadAll(stop, feeding.others, wanted) {
		readings[reading.Name] = reading.Value
	}
	for _, reading := range feeding.controller.readings(stop, start, wanted) {
		readings[reading.Name] = reading.Value
	}
	var values []smartmonitor.Value
	for _, mapping := range feeding.config.Mappings {
		if value, found := readings[mapping.Name]; found {
			values = append(values, smartmonitor.Value{Tag: mapping.Tag, Value: sensors.ToValue(value)})
		}
	}
	return values
}

// writeClock sets the clock of the screen to the current time. It tells whether the report was written.
func (feeding *feeder) writeClock(stop context.Context) (bool, error) {
	moment := feeding.now()
	report, err := smartmonitor.EncodeClock(moment)
	if err != nil {
		return false, err
	}
	feeding.lastClock = moment
	return feeding.write(stop, report), nil
}

// write writes one report. It writes nothing once stop is done, and a failure of the write is the screen lost.
func (feeding *feeder) write(stop context.Context, report smartmonitor.Report) bool {
	if stop.Err() != nil {
		return false
	}
	if err := feeding.transport.Write(report); err != nil {
		feeding.lost = true
		_, _ = fmt.Fprintf(feeding.stderr, "screen lost: %v\n", err)
		return false
	}
	return true
}
