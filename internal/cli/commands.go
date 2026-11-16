package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/screenlist"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const (
	clockLayout      = "2006-01-02 15:04:05"
	noScreenMessage  = screenlist.NoneFound
	noSensorMessage  = "no sensor could be read"
	listenSeconds    = "5"
	rawSeconds       = "1"
	secondsFlagUsage = "how many seconds to read input reports"
)

// session is what every command shares: the environment, the two options and the time the command started.
type session struct {
	env     environment
	options *options
	start   time.Time
}

func (current *session) elapsed() time.Duration { return current.env.now().Sub(current.start) }

// open opens the transport, traced when asked, and closes it when the returned function is called.
func (current *session) open() (transport, screenRestarter, func(), error) {
	opened, restarter, err := current.env.openTransport(current.env.sysfsRoot, current.env.devRoot, current.options.device)
	if err != nil {
		return nil, nil, nil, err
	}
	var used transport = opened
	if current.options.trace {
		used = &tracingTransport{inner: opened, out: current.env.stderr, elapsed: current.elapsed}
	}
	return used, restarter, func() { _ = opened.Close() }, nil
}

func (current *session) say(line string) {
	_, _ = fmt.Fprintln(current.env.stdout, line)
}

func (current *session) devicesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List the supported screens that are connected",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			lines, nearMisses, err := current.env.listScreens(current.env.sysfsRoot, current.env.devRoot)
			if err != nil {
				return err
			}
			if len(lines) == 0 {
				_, _ = fmt.Fprintln(current.env.stderr, noScreenMessage)
				for _, line := range nearMisses {
					_, _ = fmt.Fprintln(current.env.stderr, line)
				}
				return nil
			}
			for _, line := range lines {
				current.say(line)
			}
			return nil
		},
	}
}

func (current *session) clockCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clock",
		Short: "Set the clock of the screen to the current local time",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			moment := current.env.now()
			report, err := smartmonitor.EncodeClock(moment)
			if err != nil {
				return err
			}
			opened, restarter, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			if err := opened.Write(report); err != nil {
				return restartAfterHang(err, restarter)
			}
			current.say("clock sent: " + moment.Format(clockLayout))
			return nil
		},
	}
}

func (current *session) valuesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "values <tag>=<value>...",
		Short: "Send the values of numeric tags to the screen",
		RunE: func(command *cobra.Command, args []string) error {
			values, err := parseValues(args)
			if err != nil {
				return err
			}
			reports := smartmonitor.EncodeValues(values)
			opened, restarter, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			for _, report := range reports {
				if err := opened.Write(report); err != nil {
					return restartAfterHang(err, restarter)
				}
			}
			current.say(fmt.Sprintf("values sent: %d, reports written: %d", len(values), len(reports)))
			return nil
		},
	}
}

func (current *session) uploadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload a theme file to the screen as it is",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			theme, err := readTheme(args[0])
			if err != nil {
				return err
			}
			state, err := current.stateFile()
			if err != nil {
				return err
			}
			opened, restarter, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			blocks := 0
			progress := func(stage smartmonitor.Stage, block, count int) {
				blocks = count
				if stage == smartmonitor.StageData {
					current.say(fmt.Sprintf("data block %d of %d", block, count))
					return
				}
				current.say(stage.String())
			}
			if err := state.upload(opened, opened.Serial(), theme, current.env.sleep, progress); err != nil {
				return restartAfterHang(err, restarter)
			}
			current.say(fmt.Sprintf("upload complete: %d bytes in %d data blocks", len(theme), blocks))
			return nil
		},
	}
}

func readTheme(path string) ([]byte, error) { return readNamedFile("theme file", path) }

// readNamedFile reads a file the user named, and describes a failure with the description and the path once.
func readNamedFile(description, path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the %s %s: %w", description, path, innerError(err))
	}
	return content, nil
}

// innerError gives the cause of a failure of a file operation, so that the path is not written twice.
func innerError(err error) error {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return pathError.Err
	}
	return err
}

func (current *session) listenCommand() *cobra.Command {
	var seconds string
	command := &cobra.Command{
		Use:   "listen",
		Short: "Show the input reports that the screen sends",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			duration, err := parseSeconds(seconds)
			if err != nil {
				return err
			}
			opened, _, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			return current.receive(opened, duration)
		},
	}
	command.Flags().StringVar(&seconds, "seconds", listenSeconds, secondsFlagUsage)
	return command
}

func (current *session) rawCommand() *cobra.Command {
	var seconds string
	command := &cobra.Command{
		Use:   "raw <hex bytes>...",
		Short: "Send one report of the bytes given and show what the screen answers",
		RunE: func(command *cobra.Command, args []string) error {
			report, err := parseRawBytes(args)
			if err != nil {
				return err
			}
			duration, err := parseSeconds(seconds)
			if err != nil {
				return err
			}
			opened, restarter, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			if err := opened.Write(report); err != nil {
				return restartAfterHang(err, restarter)
			}
			return current.receive(opened, duration)
		},
	}
	command.Flags().StringVar(&seconds, "seconds", rawSeconds, secondsFlagUsage)
	return command
}

// receive writes a line for every report read, for the given number of seconds counted from now, and then the count.
// A read whose time limit passed ends it and is no failure, and no read waits beyond the end.
func (current *session) receive(opened smartmonitor.Transport, seconds int) error {
	deadline := current.env.now().Add(time.Duration(seconds) * time.Second)
	count := 0
	for {
		remaining := deadline.Sub(current.env.now())
		if remaining <= 0 {
			break
		}
		report, err := opened.Read(remaining)
		if errors.Is(err, smartmonitor.ErrReadTimeout) {
			break
		}
		if err != nil {
			return err
		}
		current.say(traceLine(current.elapsed(), readBody(report)))
		count++
	}
	current.say(fmt.Sprintf("reports received: %d", count))
	return nil
}

// monitorOutcome keeps what the monitor told about the one reading that sensors makes.
type monitorOutcome struct {
	address *url.URL
	read    bool
	found   []sensors.Reading
	err     error
}

func (outcome *monitorOutcome) keep(found []sensors.Reading, err error) {
	outcome.read, outcome.found, outcome.err = true, found, err
}

// say writes the line of a monitor that could not be read, or that gave nothing that sensordeck knows. It writes
// nothing when the monitor was not read, and so none after an interrupted reading.
func (outcome *monitorOutcome) say(stderr io.Writer) {
	switch {
	case !outcome.read:
	case outcome.err != nil:
		_, _ = fmt.Fprintln(stderr, monitorFailureLine(outcome.address, outcome.err))
	case len(outcome.found) == 0:
		_, _ = fmt.Fprintln(stderr, monitorEmptyLine(outcome.address))
	}
}

// monitorURL gives the URL that sensors reads the monitor at: the key of the config file at the place where run reads
// it by default, or the default URL. A config file that cannot be read or is not valid does not stop sensors, which is
// the command that is run while the config is written, so the problem is written as a line and the default is taken.
func (current *session) monitorURL() *url.URL {
	path, err := current.configPath("")
	if err != nil {
		return config.DefaultMonitorURL()
	}
	content, err := readNamedFile("config file", path)
	if errors.Is(err, fs.ErrNotExist) {
		return config.DefaultMonitorURL()
	}
	if err != nil {
		_, _ = fmt.Fprintln(current.env.stderr, err)
		return config.DefaultMonitorURL()
	}
	settings, err := config.Parse(content)
	if err != nil {
		_, _ = fmt.Fprintln(current.env.stderr, configFileError(path, err))
		return config.DefaultMonitorURL()
	}
	return settings.LibreHardwareMonitorURL
}

func (current *session) sensorsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sensors",
		Short: "List the sensor readings that can be read now",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			sources, err := current.env.newSources(current.env.sysfsRoot, current.env.devRoot, current.env.procRoot)
			if err != nil {
				return err
			}
			var monitored monitorOutcome
			if sources.monitor != nil {
				address := current.monitorURL()
				sources.others = append(slices.Clip(sources.others), sources.monitor(address, monitored.keep))
				monitored.address = address
			}
			stop, cancel := current.env.notifyStop(context.Background())
			readings := sensors.InTableOrder(sensors.ReadAll(stop, sources.all(), func(string) bool { return true }))
			interrupted := stop.Err() != nil
			cancel()
			if interrupted {
				return errors.New("interrupted")
			}
			monitored.say(current.env.stderr)
			if len(readings) == 0 {
				if problem := sources.problem(); problem != nil {
					_, _ = fmt.Fprintln(current.env.stderr, problem)
				}
				_, _ = fmt.Fprintln(current.env.stderr, noSensorMessage)
				return nil
			}
			for _, reading := range readings {
				value := strconv.FormatFloat(reading.Value, 'f', 1, 64)
				current.say(reading.Name + "\t" + value + "\t" + sensors.Unit(reading.Name))
			}
			return nil
		},
	}
}
