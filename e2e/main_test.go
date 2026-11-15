//go:build linux

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakescreen"
)

const runTimeLimit = 30 * time.Second

// executable is the built sensordeck, which every test runs as a process.
var executable string

func TestMain(main *testing.M) {
	os.Exit(buildAndRun(main))
}

func buildAndRun(main *testing.M) int {
	directory, err := os.MkdirTemp("", "sensordeck-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(directory)
	executable = filepath.Join(directory, "sensordeck")
	build := exec.Command("go", "build", "-o", executable, ".")
	build.Dir = ".."
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building the executable failed: %v\n%s", err, output)
		return 1
	}
	return main.Run()
}

// result is what a run of the executable left behind.
type result struct {
	stdout, stderr string
	status         int
}

// run runs the executable against the fake screen through the two variables that replace the roots.
func run(test *testing.T, fake *fakescreen.Fake, args ...string) result {
	test.Helper()
	return runWith(test, fake, nil, args...)
}

// environmentOf gives the variables that put the executable on the fake screen, and the given ones after them, so that
// they override what the process inherits.
func environmentOf(fake *fakescreen.Fake, environment []string) []string {
	return append(append(os.Environ(),
		"SENSORDECK_SYSFS_ROOT="+fake.SysfsRoot,
		"SENSORDECK_DEV_ROOT="+fake.DevRoot), environment...)
}

// runWith is run with more environment variables, and waits for the end of the executable.
func runWith(test *testing.T, fake *fakescreen.Fake, environment []string, args ...string) result {
	test.Helper()
	return runIn(test, fake, "", environment, args...)
}

// runIn is runWith with the working directory of the executable set, or left as it is when it is empty.
func runIn(test *testing.T, fake *fakescreen.Fake, directory string, environment []string, args ...string) result {
	test.Helper()
	context, cancel := context.WithTimeout(context.Background(), runTimeLimit)
	defer cancel()
	command := exec.CommandContext(context, executable, args...)
	command.Env = environmentOf(fake, environment)
	command.Dir = directory
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	return finish(test, args, command.Run(), &stdout, &stderr)
}

func finish(test *testing.T, args []string, err error, stdout, stderr *bytes.Buffer) result {
	test.Helper()
	status := 0
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		status = exitError.ExitCode()
	} else if err != nil {
		test.Fatalf("running %v: %v", args, err)
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), status: status}
}

// process is an executable that is still running.
type process struct {
	test           *testing.T
	args           []string
	command        *exec.Cmd
	stdout, stderr bytes.Buffer
	done           chan error
}

// start starts the executable and does not wait for it. A cleanup kills it when the test ends and it still runs.
func start(test *testing.T, fake *fakescreen.Fake, environment []string, args ...string) *process {
	test.Helper()
	running := &process{test: test, args: args, done: make(chan error, 1)}
	running.command = exec.Command(executable, args...)
	running.command.Env = environmentOf(fake, environment)
	running.command.Stdout, running.command.Stderr = &running.stdout, &running.stderr
	if err := running.command.Start(); err != nil {
		test.Fatalf("starting %v: %v", args, err)
	}
	go func() { running.done <- running.command.Wait() }()
	test.Cleanup(func() { _ = running.command.Process.Kill() })
	return running
}

// stop sends the signal and waits for the end, and gives what the run left behind and the time from the signal to
// the end.
func (running *process) stop(signal os.Signal) (result, time.Duration) {
	running.test.Helper()
	if err := running.command.Process.Signal(signal); err != nil {
		running.test.Fatalf("signalling %v: %v", running.args, err)
	}
	sent := time.Now()
	select {
	case err := <-running.done:
		return finish(running.test, running.args, err, &running.stdout, &running.stderr), time.Since(sent)
	case <-time.After(runTimeLimit):
		running.test.Fatalf("%v did not end after the signal", running.args)
		return result{}, 0
	}
}
