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
	"github.com/norareit/sensordeck/internal/custommetrics"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

func (current *session) runCommand() *cobra.Command {
	var configPath string
	var background bool
	command := &cobra.Command{
		Use:   "run",
		Short: "Send the sensor values named in the config file to the screen until stopped",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			if background {
				return current.runInBackground(configPath)
			}
			// The keeper and the feeder write their lines from two goroutines, and so may the trace.
			current.env.stderr = &lockedWriter{writer: current.env.stderr}
			prepared, err := current.prepareRun(configPath)
			if err != nil {
				return err
			}
			stop, cancel := current.env.notifyStop(context.Background())
			defer cancel()
			opened, restarter, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			return current.serveRun(stop, prepared, opened, restarter, closeTransport)
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "read the config from this file instead of the default place")
	command.Flags().BoolVar(&background, "background", false, "write to the log and start again after a failed start, as the startup task does")
	_ = command.Flags().MarkHidden("background")
	return command
}

// preparedRun is what a start of run made before it opens the screen.
type preparedRun struct {
	path    string
	stamp   fileStamp
	started runConfig
	sources sourceSet
}

// prepareRun reads and loads the config file and makes the sources, as run does at its start.
func (current *session) prepareRun(named string) (preparedRun, error) {
	path, err := current.configPath(named)
	if err != nil {
		return preparedRun{}, err
	}
	// The first stamp is taken before the file is read, so that a write in between is seen as a change.
	stamp := stampOf(path)
	content, err := readNamedFile("config file", path)
	if err != nil {
		return preparedRun{}, err
	}
	started, err := current.loadConfigFile(path, content)
	if err != nil {
		return preparedRun{}, err
	}
	sources, err := current.env.newSources(current.env.sysfsRoot, current.env.devRoot, current.env.procRoot)
	if err != nil {
		return preparedRun{}, err
	}
	return preparedRun{path: path, stamp: stamp, started: started, sources: sources}, nil
}

// serveRun writes the line running, feeds the screen until stop is done, ends the keeper and closes the transport, and
// writes stopped. Its error is the error of the feeding.
func (current *session) serveRun(stop context.Context, prepared preparedRun, opened transport, restarter screenRestarter, closeTransport func()) error {
	settings := prepared.started.settings
	sources := prepared.sources
	current.say(fmt.Sprintf("running: %d metrics every %s", len(settings.MetricNames()), settings.Interval))
	observed := &observedTransport{transport: opened}
	custom := custommetrics.NewSet(settings.Metrics, current.env.sysfsRoot, configDirectory(prepared.path), current.env.now, current.env.stderr)
	feeding := &feeder{
		transport:  observed,
		restarter:  restarter,
		others:     append(current.withMonitor(sources, settings.LibreHardwareMonitorURL), custom),
		custom:     custom,
		controller: &periodicSource{source: sources.controller, period: settings.CommanderPeriod()},
		config:     settings,
		holder:     newThemeHolder(prepared.started.theme, current.env.sleep, current.env.now, current.say, current.env.stderr),
		watcher:    &configWatcher{path: prepared.path, last: prepared.stamp, stderr: current.env.stderr, load: func(content []byte) (runConfig, error) { return current.loadConfig(prepared.path, content) }},
		now:        current.env.now,
		wait:       current.env.wait,
		stderr:     current.env.stderr,
	}
	observed.written = feeding.writeSucceeded
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
	custom.Close()
	closeTransport()
	if feedErr != nil {
		return feedErr
	}
	current.say("stopped")
	return nil
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
	start := func(stop context.Context) (heldSession, setupNotice, error) {
		return sources.startSession(stop, *settings.CommanderColour)
	}
	return newSessionKeeper(start, probesMapped, current.env.now, current.env.after, current.env.stderr)
}

// configDirectory gives the directory of the config file as an absolute path, where the commands of custom metrics run.
// A path that cannot be made absolute stays as it is.
func configDirectory(path string) string {
	directory := filepath.Dir(path)
	if absolute, err := filepath.Abs(directory); err == nil {
		return absolute
	}
	return directory
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

// restartInterval is the least time between the starts of two restarts of a hung screen. A restart that did not help
// costs a write that times out and a restart for nothing, and restarting every few seconds is what it prevents.
const restartInterval = time.Minute

// hangState is the hang of a screen whose writes time out, from the first write that timed out to the first one that
// succeeds, and the lines said in it, so that each is written once.
type hangState struct {
	active            bool
	restarted         bool // a restart of this hang made the screen leave the bus
	awaitingOpen      bool // that restart did not open the screen again
	restartingSaid    bool
	notRestartedSaid  bool
	stillHungSaid     bool
	cannotRestartSaid bool
}

// feeder sends the clock and the values of the mapped metrics to the screen in rounds, and the clock alone in ticks at
// each whole second between them, and reopens the transport
// when the screen is lost. A screen whose writes time out is hung, and the round recovers it by restarting its firmware,
// at the first round of the hang and at most once a minute after, with its lines written once in the hang. Without commander_color it reads the controller at most once per period and sends the kept
// readings of its last reading in the other rounds. With it the controller is read through the keeper, which holds a
// session of the controller, and a round with a probe mapped asks it for a reading of its own. The holder keeps the
// theme of the config on the screen, and the watcher gives the config again when its file changed.
type feeder struct {
	transport  transport
	others     []sensors.Source
	custom     *custommetrics.Set // the last of others; the reload replaces its definitions
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
	restarter  screenRestarter
	// lastRestart is when the last restart that sent a request began, kept across hangs so that no two are less than
	// restartInterval apart.
	lastRestart time.Time
	hang        hangState
}

// probesMapped tells whether a metric of the controller is mapped now. The keeper asks it from the goroutine of the
// feeder, which is the one that replaces the mappings.
func (feeding *feeder) probesMapped() bool {
	return slices.ContainsFunc(feeding.config.MetricNames(), sensors.IsCommanderTemperature)
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
	if feeding.lost && !feeding.reopen(stop) {
		return nil
	}
	ready, uploaded, failure := feeding.holder.ensure(stop, feeding.transport)
	if failure != nil {
		feeding.failed(failure, false)
	}
	if !ready {
		return nil
	}
	feeding.clockDue = feeding.clockDue || uploaded
	if written, err := feeding.writeClockWhenDue(stop, start); !written || err != nil {
		return err
	}
	for _, report := range smartmonitor.EncodeUnits(feeding.read(stop, start)) {
		if !feeding.write(stop, report) {
			return nil
		}
	}
	return nil
}

// reopen opens the node of a lost screen again and tells whether it is open. A screen that is hung is restarted first,
// when it is time for one. It is back the same way in both cases, and says so unless it was hung, which says "screen
// recovered" at its first write that succeeds. The screen keeps its theme in its flash, so a return makes no upload
// due, except that a restart that recovered a hung screen lets an upload that is due be made at once.
func (feeding *feeder) reopen(stop context.Context) bool {
	if feeding.hang.active {
		open, recovered := feeding.recoverHung(stop)
		if !open {
			return false
		}
		if recovered {
			feeding.holder.restartRecovered()
		}
	} else {
		if feeding.transport.Reopen() != nil {
			return false
		}
		_, _ = fmt.Fprintln(feeding.stderr, "screen back")
	}
	feeding.lost = false
	feeding.clockDue = true
	return true
}

// recoverHung opens the node of a hung screen again, after a restart of the screen when no restart began in the last
// minute, and tells whether the node is open, and whether a restart that made the screen leave the bus recovered it. A
// round that makes no restart reopens the node as for a lost screen, so that a screen that was unplugged is taken up at
// once, and that is no recovery.
func (feeding *feeder) recoverHung(stop context.Context) (open, recovered bool) {
	if feeding.hang.awaitingOpen {
		if feeding.transport.Reopen() != nil {
			return false, false
		}
		feeding.hang.awaitingOpen = false
		return true, true
	}
	if !feeding.lastRestart.IsZero() && feeding.now().Sub(feeding.lastRestart) < restartInterval {
		return feeding.transport.Reopen() == nil, false
	}
	return feeding.restartScreen(stop)
}

// restartScreen restarts the screen and tells whether the node is open again, and whether the restart recovered it,
// which is so when the screen left the bus and the node was opened again. A restart that is not possible sent nothing,
// so it does not count for restartInterval and the next round tries again, which uses a rule applied while run runs.
func (feeding *feeder) restartScreen(stop context.Context) (open, recovered bool) {
	feeding.sayOnce(&feeding.hang.restartingSaid, restartingLine)
	began := feeding.now()
	result := feeding.restarter.restart(stop)
	if stop.Err() != nil {
		return false, false
	}
	switch result.outcome {
	case restartNotPossible:
		feeding.sayOnce(&feeding.hang.cannotRestartSaid, hungText+fmt.Sprintf(cannotRestartFormat, result.err))
		return feeding.transport.Reopen() == nil, false
	case notRestarted:
		feeding.lastRestart = began
		feeding.sayOnce(&feeding.hang.notRestartedSaid, notRestartedLine)
		return feeding.transport.Reopen() == nil, false
	}
	feeding.lastRestart = began
	feeding.hang.restarted = true
	_, _ = fmt.Fprintf(feeding.stderr, restartedFormat+"\n", result.index)
	if result.outcome == restartedNotOpen {
		feeding.hang.awaitingOpen = true
		return false, false
	}
	return true, true
}

// sayOnce writes the line to standard error unless it was written in this hang, which the flag records.
func (feeding *feeder) sayOnce(said *bool, line string) {
	if *said {
		return
	}
	*said = true
	_, _ = fmt.Fprintln(feeding.stderr, line)
}

// writeSucceeded ends a hang, which only a write that succeeds does: the node of a hung screen opens fine.
func (feeding *feeder) writeSucceeded() {
	if !feeding.hang.active {
		return
	}
	feeding.hang = hangState{}
	_, _ = fmt.Fprintln(feeding.stderr, "screen recovered")
}

// failed makes the screen lost after a failed write or upload. A timeout begins a hang, with its line once, and
// another failure ends one. A failed write says that the screen is lost, and a failed upload has said it failed.
func (feeding *feeder) failed(err error, write bool) {
	feeding.lost = true
	if feeding.restarter.timedOut(err) {
		if !feeding.hang.active {
			feeding.hang.active = true
			_, _ = fmt.Fprintf(feeding.stderr, "screen hung: %v\n", err)
		} else if feeding.hang.restarted {
			feeding.sayOnce(&feeding.hang.stillHungSaid, stillHungLine)
		}
		return
	}
	feeding.hang = hangState{}
	if write {
		_, _ = fmt.Fprintf(feeding.stderr, "screen lost: %v\n", err)
	}
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
	feeding.config.Alarms = reloaded.settings.Alarms
	feeding.config.Metrics = reloaded.settings.Metrics
	feeding.custom.Replace(reloaded.settings.Metrics)
	feeding.holder.setTheme(reloaded.theme)
}

// read gives the units of the mapped metrics that have a value, in the order of their lowest tags. The other sources are
// read now. The controller gives the readings of its last reading, which is made when the period has passed since the
// start of the round that made the one before.
func (feeding *feeder) read(stop context.Context, start time.Time) [][]smartmonitor.Value {
	names := feeding.config.MetricNames()
	wanted := func(name string) bool { return slices.Contains(names, name) }
	readings := make(map[string]float64, len(names))
	for _, reading := range sensors.ReadAll(stop, feeding.others, wanted) {
		readings[reading.Name] = reading.Value
	}
	for _, reading := range feeding.controller.readings(stop, start, wanted) {
		readings[reading.Name] = reading.Value
	}
	return roundUnits(feeding.config, readings)
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

// write writes one report. It writes nothing once stop is done, and a failure of the write is the screen lost or hung.
func (feeding *feeder) write(stop context.Context, report smartmonitor.Report) bool {
	if stop.Err() != nil {
		return false
	}
	if err := feeding.transport.Write(report); err != nil {
		feeding.failed(err, true)
		return false
	}
	return true
}
