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
	context, cancel := context.WithTimeout(context.Background(), runTimeLimit)
	defer cancel()
	command := exec.CommandContext(context, executable, args...)
	command.Env = append(os.Environ(),
		"SENSORDECK_SYSFS_ROOT="+fake.SysfsRoot,
		"SENSORDECK_DEV_ROOT="+fake.DevRoot)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	status := 0
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		status = exitError.ExitCode()
	} else if err != nil {
		test.Fatalf("running %v: %v", args, err)
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), status: status}
}
