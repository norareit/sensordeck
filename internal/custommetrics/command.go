package custommetrics

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/config"
)

// TimeLimit is how long a run of a command may last, or its time between runs when that is shorter.
const TimeLimit = 10 * time.Second

// pipeGrace is how long a run that was killed may keep a reading waiting for a child that holds its output open.
const pipeGrace = 500 * time.Millisecond

func timeLimit(definition config.Metric) time.Duration {
	return min(TimeLimit, definition.Every)
}

// runCommand runs the text through the shell in the directory. Its number is not scaled. The bool is false when stop
// was done before the run ended, so that the outcome of a run that nobody waits for any more is dropped.
func runCommand(stop context.Context, text, directory string, limit time.Duration) (outcome, bool) {
	limited, cancel := context.WithTimeout(stop, limit)
	defer cancel()
	command := shellCommand(limited, text)
	command.Dir = directory
	var output, standardError bytes.Buffer
	command.Stdout = &output
	command.Stderr = &standardError
	command.WaitDelay = pipeGrace
	err := command.Run()
	if stop.Err() != nil {
		return outcome{}, false
	}
	if limited.Err() != nil {
		return outcome{cause: fmt.Sprintf("the command outlasted its time limit of %v", limit)}, true
	}
	return finished(command, err, output.String(), standardError.String()), true
}

func finished(command *exec.Cmd, err error, output, standardError string) outcome {
	if err != nil && !isOnlyAHeldPipe(command, err) {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return outcome{cause: "the command could not start: " + err.Error()}
		}
		cause := fmt.Sprintf("the command exited with status %d", exit.ExitCode())
		if line := firstLine(standardError); line != "" {
			cause += ": " + line
		}
		return outcome{cause: cause}
	}
	number, found := Number(output)
	if !found {
		return outcome{cause: "no number in the output of the command"}
	}
	return outcome{value: number}
}

// isOnlyAHeldPipe tells an error that is a child holding the output open after the shell exited with status 0.
func isOnlyAHeldPipe(command *exec.Cmd, err error) bool {
	return errors.Is(err, exec.ErrWaitDelay) && command.ProcessState != nil && command.ProcessState.ExitCode() == 0
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.TrimSuffix(line, "\r")
}
