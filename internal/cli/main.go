// Package cli is the command line of sensordeck.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// transport is what the commands need of a device: the transport of the protocol and a close.
type transport interface {
	smartmonitor.Transport
	Close() error
}

// environment is what the commands take from the outside world. Main fills it from the process, and the unit tests
// fill it with fakes.
type environment struct {
	stdout, stderr io.Writer
	sysfsRoot      string
	devRoot        string
	procRoot       string
	now            func() time.Time
	sleep          func(time.Duration)
	listScreens    func(sysfsRoot, devRoot string) ([]string, error)
	openTransport  func(sysfsRoot, devRoot, device string) (transport, error)
	userConfigDir  func() (string, error)
	newSources     func(sysfsRoot, devRoot, procRoot string) (sourceSet, error)
	// notifyStop gives a context that ends at SIGINT or SIGTERM. Only run and the reading of sensors call it, so that
	// the other commands keep the default handling of Ctrl+C.
	notifyStop func(parent context.Context) (context.Context, context.CancelFunc)
	// after gives a channel that receives the time once the duration has passed.
	after func(duration time.Duration) <-chan time.Time
	// wait waits for the duration and gives false when stop ended the wait.
	wait func(stop context.Context, duration time.Duration) bool
}

const (
	sysfsRootVariable = "SENSORDECK_SYSFS_ROOT"
	devRootVariable   = "SENSORDECK_DEV_ROOT"
	procRootVariable  = "SENSORDECK_PROC_ROOT"
	defaultSysfsRoot  = "/sys"
	defaultDevRoot    = "/dev"
	defaultProcRoot   = "/proc"
)

// Main runs the command line of the process and gives its exit status. It is the one place where the environment
// variables for the tests are read.
func Main() int {
	return run(os.Args[1:], environment{
		stdout:        os.Stdout,
		stderr:        os.Stderr,
		sysfsRoot:     valueOrDefault(os.Getenv(sysfsRootVariable), defaultSysfsRoot),
		devRoot:       valueOrDefault(os.Getenv(devRootVariable), defaultDevRoot),
		procRoot:      valueOrDefault(os.Getenv(procRootVariable), defaultProcRoot),
		now:           time.Now,
		sleep:         time.Sleep,
		listScreens:   listScreens,
		openTransport: openTransport,
		userConfigDir: os.UserConfigDir,
		newSources:    newSources,
		notifyStop: func(parent context.Context) (context.Context, context.CancelFunc) {
			return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
		},
		after: time.After,
		wait:  waitFor,
	})
}

// waitFor waits for the duration and gives false when stop ended before it passed.
func waitFor(stop context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-stop.Done():
		return false
	}
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// run executes the command line and gives the exit status: 0, or 1 after writing the error to standard error.
func run(args []string, env environment) int {
	root := newRootCommand(env, env.now())
	root.SetArgs(args)
	root.SetOut(env.stdout)
	root.SetErr(env.stderr)
	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "sensordeck: %v\n", err)
		return 1
	}
	return 0
}

// options are the two options that hold for every command.
type options struct {
	device string
	trace  bool
}

func newRootCommand(env environment, start time.Time) *cobra.Command {
	chosen := &options{}
	root := &cobra.Command{
		Use:           "sensordeck",
		Short:         "Drive a SmartMonitor screen",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().StringVar(&chosen.device, "device", "", "open the transport on this node instead of looking for the screen")
	root.PersistentFlags().BoolVar(&chosen.trace, "trace", false, "write every report that goes to or comes from the device to standard error")
	session := &session{env: env, options: chosen, start: start}
	root.AddCommand(
		session.devicesCommand(),
		session.clockCommand(),
		session.valuesCommand(),
		session.uploadCommand(),
		session.listenCommand(),
		session.rawCommand(),
		session.sensorsCommand(),
		session.runCommand(),
	)
	return root
}
