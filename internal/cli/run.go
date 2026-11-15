package cli

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

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
			// The first stamp is taken before the file is read, so that a write in between is seen as a change.
			stamp := stampOf(path)
			content, err := readNamedFile("config file", path)
			if err != nil {
				return err
			}
			started, err := current.loadConfigFile(path, content)
			if err != nil {
				return err
			}
			settings := started.settings
			// The keeper and the feeder write their lines from two goroutines, and so may the trace.
			current.env.stderr = &lockedWriter{writer: current.env.stderr}
			state, stateErr := current.stateFile()
			if started.theme != nil && stateErr != nil {
				return stateErr
			}
			sources, err := current.env.newSources(current.env.sysfsRoot, current.env.devRoot, current.env.procRoot)
			if err != nil {
				return err
			}
			stop, cancel := current.env.notifyStop(context.Background())
			defer cancel()
			opened, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			current.say(fmt.Sprintf("running: %d metrics every %s", len(settings.Mappings), settings.Interval))
			feeding := &feeder{
				transport:  opened,
				others:     current.withMonitor(sources, settings.LibreHardwareMonitorURL),
				controller: &periodicSource{source: sources.controller, period: settings.CommanderPeriod()},
				config:     settings,
				holder:     newThemeHolder(started.theme, state, current.env.sleep, current.env.now, current.say, current.env.stderr),
				watcher: &configWatcher{path: path, last: stamp, stderr: current.env.stderr, load: func(content []byte) (runConfig, error) {
					reloaded, err := current.loadConfig(path, content)
					// A theme needs the state file, and a run that started with tags may have none.
					if err == nil && reloaded.theme != nil && stateErr != nil {
						return runConfig{}, stateErr
					}
					return reloaded, err
				}},
				now:    current.env.now,
				wait:   current.env.wait,
				stderr: current.env.stderr,
			}
			endKeeping := func() {}
			if settings.CommanderColour != nil {
				keeper := current.newKeeper(sources, settings, feeding.probesMapped)
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

// withMonitor gives the other sources of the set with the monitor of the URL after them, when the set has a monitor.
// The monitor writes a line for each outage to standard error.
func (current *session) withMonitor(sources sourceSet, address *url.URL) []sensors.Source {
	if sources.monitor == nil {
		return sources.others
	}
	outage := &monitorOutage{stderr: current.env.stderr, address: address}
	return append(slices.Clip(sources.others), sources.monitor(address, outage.observe))
}

// newKeeper makes the keeper of the session that lights the LEDs in the colour of the config.
func (current *session) newKeeper(sources sourceSet, settings config.Config, probesMapped func() bool) *sessionKeeper {
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
	directory, err := current.ownConfigDirectory()
	if err != nil {
		return "", fmt.Errorf("cannot find the config directory, name the file with --config: %w", err)
	}
	return filepath.Join(directory, "config.yaml"), nil
}

// ownConfigDirectory gives the directory sensordeck in the config directory of the user, with the error of
// userConfigDir unwrapped, so that each caller writes its own message.
func (current *session) ownConfigDirectory() (string, error) {
	directory, err := current.env.userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "sensordeck"), nil
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

// feeder sends the clock and the values of the mapped metrics to the screen in rounds, and the clock alone in ticks at
// each whole second between them, and reopens the transport
// when the screen is lost. Without commander_color it reads the controller at most once per period and sends the kept
// readings of its last reading in the other rounds. With it the controller is read through the keeper, which holds a
// session of the controller, and a round with a probe mapped asks it for a reading of its own. The holder keeps the
// theme of the config on the screen, and the watcher gives the config again when its file changed.
type feeder struct {
	transport  transport
	others     []sensors.Source
	controller controllerReader
	config     config.Config
	holder     *themeHolder
	watcher    *configWatcher
	now        func() time.Time
	wait       func(stop context.Context, duration time.Duration) bool
	stderr     io.Writer
	lastClock  time.Time
	clockDue   bool // the clock is to be sent in the next round that sends anything, whatever its age
	lost       bool
}

// probesMapped tells whether a metric of the controller is mapped now. The keeper asks it from the goroutine of the
// feeder, which is the one that replaces the mappings.
func (feeding *feeder) probesMapped() bool {
	for _, mapping := range feeding.config.Mappings {
		if sensors.IsCommanderTemperature(mapping.Name) {
			return true
		}
	}
	return false
}

// feed works until stop is done, and gives an error only when the clock cannot be encoded. Besides its rounds it ticks
// at each whole second of the time, and a tick writes the clock alone.
func (feeding *feeder) feed(stop context.Context) error {
	feeding.clockDue = true
	first := feeding.now()
	interval := feeding.config.Interval
	nextRound := first
	index := 0
	for stop.Err() == nil {
		start := feeding.now()
		var err error
		if !start.Before(nextRound) {
			err = feeding.round(stop, start)
			// A round that ran into the next one is followed at once, and the rounds missed are not made up.
			index = max(index+1, int(feeding.now().Sub(first)/interval))
			nextRound = first.Add(time.Duration(index) * interval)
		} else {
			err = feeding.tick(stop, start)
		}
		if err != nil {
			return err
		}
		next := start.Truncate(time.Second).Add(time.Second)
		if nextRound.Before(next) {
			next = nextRound
		}
		if end := feeding.now(); next.After(end) && !feeding.wait(stop, next.Sub(end)) {
			return nil
		}
	}
	return nil
}

// round does one round, which started at start.
func (feeding *feeder) round(stop context.Context, start time.Time) error {
	feeding.reload()
	if feeding.lost {
		if feeding.transport.Reopen() != nil {
			return nil
		}
		feeding.lost = false
		_, _ = fmt.Fprintln(feeding.stderr, "screen back")
		feeding.holder.screenBack()
		feeding.clockDue = true
	}
	ready, uploaded, failed := feeding.holder.ensure(stop, feeding.transport)
	if failed {
		feeding.lost = true
	}
	if !ready {
		return nil
	}
	feeding.clockDue = feeding.clockDue || uploaded
	if written, err := feeding.writeClockWhenDue(stop, start); !written || err != nil {
		return err
	}
	for _, report := range smartmonitor.EncodeValues(feeding.read(stop, start)) {
		if !feeding.write(stop, report) {
			return nil
		}
	}
	return nil
}

// tick writes the clock between the rounds, and nothing else. It leaves a lost screen and a theme that is not known
// to be on the screen to the next round.
func (feeding *feeder) tick(stop context.Context, start time.Time) error {
	if feeding.lost || !feeding.holder.onScreen() {
		return nil
	}
	_, err := feeding.writeClockWhenDue(stop, start)
	return err
}

// writeClockWhenDue writes the clock when it is due or when the whole second of start differs from that of the last
// clock report. The screen shows the time of the last clock report it received and does not advance it, so the clock
// is written once a second for the seconds on the screen to change. A second that differs, and not one that is later,
// makes a time set back count. It tells whether the work may go on, which is not so after a failed write.
func (feeding *feeder) writeClockWhenDue(stop context.Context, start time.Time) (bool, error) {
	if !feeding.clockDue && start.Truncate(time.Second).Equal(feeding.lastClock.Truncate(time.Second)) {
		return true, nil
	}
	written, err := feeding.writeClock(stop)
	if written {
		feeding.clockDue = false
	}
	return written, err
}

// reload applies the config file when it changed: the mappings and the theme at once, and for the other keys a line
// that a restart applies them.
func (feeding *feeder) reload() {
	reloaded, changed := feeding.watcher.poll()
	if !changed {
		return
	}
	for _, key := range restartKeys(feeding.config, reloaded.settings) {
		_, _ = fmt.Fprintf(feeding.stderr, "config file %s: %s changed, restart run to apply it\n", feeding.watcher.path, key)
	}
	feeding.config.Mappings = reloaded.settings.Mappings
	feeding.holder.setTheme(reloaded.theme)
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
