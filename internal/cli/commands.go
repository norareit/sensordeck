package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const (
	clockLayout      = "2006-01-02 15:04:05"
	noScreenMessage  = "no supported screen found"
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
func (current *session) open() (transport, func(), error) {
	opened, err := current.env.openTransport(current.env.sysfsRoot, current.env.devRoot, current.options.device)
	if err != nil {
		return nil, nil, err
	}
	var used transport = opened
	if current.options.trace {
		used = &tracingTransport{inner: opened, out: current.env.stderr, elapsed: current.elapsed}
	}
	return used, func() { _ = opened.Close() }, nil
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
			lines, err := current.env.listScreens(current.env.sysfsRoot, current.env.devRoot)
			if err != nil {
				return err
			}
			if len(lines) == 0 {
				_, _ = fmt.Fprintln(current.env.stderr, noScreenMessage)
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
			opened, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			if err := opened.Write(report); err != nil {
				return err
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
			opened, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			for _, report := range reports {
				if err := opened.Write(report); err != nil {
					return err
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
			opened, closeTransport, err := current.open()
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
			if err := smartmonitor.Upload(opened, theme, current.env.sleep, progress); err != nil {
				return err
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
		// The inner error, so that the path is not written twice.
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			err = pathError.Err
		}
		return nil, fmt.Errorf("cannot read the %s %s: %w", description, path, err)
	}
	return content, nil
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
			opened, closeTransport, err := current.open()
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
			opened, closeTransport, err := current.open()
			if err != nil {
				return err
			}
			defer closeTransport()
			if err := opened.Write(report); err != nil {
				return err
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
