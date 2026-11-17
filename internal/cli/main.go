// Package cli is the command line of sensordeck.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
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
	// listScreens gives the lines of the supported screens and, when there is none, the lines of the devices that
	// almost matched.
	listScreens   func(sysfsRoot, devRoot string) (screens, nearMisses []string, err error)
	openTransport func(sysfsRoot, devRoot, procRoot, device string) (transport, screenRestarter, error)
	userConfigDir func() (string, error)
	userCacheDir  func() (string, error)
	newSources    func(sysfsRoot, devRoot, procRoot string) (sourceSet, error)
	// notifyStop gives a context that ends at SIGINT or SIGTERM. Only run and the reading of sensors call it, so that
	// the other commands keep the default handling of Ctrl+C.
	notifyStop func(parent context.Context) (context.Context, context.CancelFunc)
	// after gives a channel that receives the time once the duration has passed.
	after func(duration time.Duration) <-chan time.Time
	// wait waits for the duration and gives false when stop ended the wait.
	wait func(stop context.Context, duration time.Duration) bool
	// system is the name of the system, as runtime.GOOS gives it.
	system string
	// executable gives the path of the running executable, as os.Executable does.
	executable func() (string, error)
	// systemctl runs systemctl --user with the arguments and gives its standard output and standard error. Its error is
	// nil when the call succeeded, an *exec.ExitError when it ended with another status, and the error of the start when
	// it could not start.
	systemctl func(arguments ...string) (stdout, stderr string, err error)
	// schtasks runs schtasks.exe with the arguments, as systemctl runs systemctl --user.
	schtasks func(arguments ...string) (stdout, stderr string, err error)
	// currentUser gives the security identifier of the user on Windows, the Uid of os/user.Current.
	currentUser func() (string, error)
	// holdStopEvent creates the stop event and holds it until its close, or gives errStopEventExists.
	holdStopEvent func() (heldStopEvent, error)
	// findBackgroundRun tells whether the stop event can be opened, and closes it at once.
	findBackgroundRun func() (bool, error)
	// signalBackgroundRun signals the stop event. An event that does not exist is no error.
	signalBackgroundRun func() error
	// detachConsole detaches the process from its console.
	detachConsole func() error
	// version is what --version prints.
	version string
}

// heldStopEvent is the stop event that a background run holds.
type heldStopEvent interface {
	signalled() <-chan struct{}
	close()
}

var (
	// errStopEventExists is the error of holdStopEvent when another background run holds the event.
	errStopEventExists = errors.New("the stop event exists already")
	// errLogged is the error of a background run that wrote its error to the log, so that run writes nothing for it.
	errLogged = errors.New("the error is in the log")
)

const (
	sysfsRootVariable = "SENSORDECK_SYSFS_ROOT"
	devRootVariable   = "SENSORDECK_DEV_ROOT"
	procRootVariable  = "SENSORDECK_PROC_ROOT"
	defaultSysfsRoot  = "/sys"
	defaultDevRoot    = "/dev"
	defaultProcRoot   = "/proc"
)

// Main runs the command line of the process and gives its exit status. It is the one place where the environment
// variables for the tests are read. The linked version is the one the release build sets, empty for any other build.
func Main(linked string) int {
	info, ok := debug.ReadBuildInfo()
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
		userCacheDir:  os.UserCacheDir,
		newSources:    newSources,
		notifyStop: func(parent context.Context) (context.Context, context.CancelFunc) {
			return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
		},
		after:               time.After,
		wait:                waitFor,
		system:              runtime.GOOS,
		executable:          os.Executable,
		systemctl:           runSystemctl,
		schtasks:            runSchtasks,
		currentUser:         currentUserIdentifier,
		holdStopEvent:       holdStopEvent,
		findBackgroundRun:   findBackgroundRun,
		signalBackgroundRun: signalBackgroundRun,
		detachConsole:       detachConsole,
		version:             resolveVersion(linked, info, ok),
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
		if errors.Is(err, errLogged) {
			return 1
		}
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
		Version:       env.version,
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
		session.themeCommand(),
		session.serviceCommand(),
	)
	return root
}
