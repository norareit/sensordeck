package custommetrics

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runText(test *testing.T, text, directory string, limit time.Duration) outcome {
	test.Helper()
	result, finished := runCommand(context.Background(), text, directory, limit)
	if !finished {
		test.Fatal("the run was reported as not finished")
	}
	return result
}

func TestACommandGivesItsNumber(test *testing.T) {
	if got := runText(test, "echo 'temp 21.5'", test.TempDir(), 5*time.Second); got.failed() || got.value != 21.5 {
		test.Errorf("got %+v", got)
	}
}

func TestACommandThatIsAPipeline(test *testing.T) {
	if got := runText(test, "echo 7 8 | cut -d' ' -f2", test.TempDir(), 5*time.Second); got.value != 8 {
		test.Errorf("got %+v", got)
	}
}

func TestACommandThatExitsWithAStatus(test *testing.T) {
	got := runText(test, "echo 5; echo boom >&2; echo second >&2; exit 3", test.TempDir(), 5*time.Second)
	if got.cause != "the command exited with status 3: boom" {
		test.Errorf("cause %q", got.cause)
	}
	got = runText(test, "exit 4", test.TempDir(), 5*time.Second)
	if got.cause != "the command exited with status 4" {
		test.Errorf("cause %q", got.cause)
	}
}

func TestTheFirstLineOfStandardErrorIsTheFirstLineEvenWhenItIsEmptyOrIndented(test *testing.T) {
	got := runText(test, "printf '\\nsecond\\n' >&2; exit 3", test.TempDir(), 5*time.Second)
	if got.cause != "the command exited with status 3" {
		test.Errorf("cause %q", got.cause)
	}
	got = runText(test, "printf '  indented\\r\\nsecond\\n' >&2; exit 3", test.TempDir(), 5*time.Second)
	if got.cause != "the command exited with status 3:   indented" {
		test.Errorf("cause %q", got.cause)
	}
}

func TestACommandWithNoNumber(test *testing.T) {
	if got := runText(test, "echo none", test.TempDir(), 5*time.Second); got.cause != "no number in the output of the command" {
		test.Errorf("cause %q", got.cause)
	}
}

func TestACommandRunsInTheDirectoryGiven(test *testing.T) {
	directory := test.TempDir()
	writeFile(test, filepath.Join(directory, "value"), "13\n")
	if got := runText(test, "cat value", directory, 5*time.Second); got.value != 13 {
		test.Errorf("got %+v", got)
	}
}

func TestACommandInAMissingDirectoryCannotStart(test *testing.T) {
	got := runText(test, "echo 1", filepath.Join(test.TempDir(), "missing"), 5*time.Second)
	if !strings.HasPrefix(got.cause, "the command could not start: ") {
		test.Errorf("cause %q", got.cause)
	}
}

func TestACommandPastItsLimitIsKilledWithItsChildren(test *testing.T) {
	marker := filepath.Join(test.TempDir(), "marker")
	started := time.Now()
	got := runText(test, "(sleep 2; : > "+marker+") & sleep 5", test.TempDir(), 300*time.Millisecond)
	if got.cause != "the command outlasted its time limit of 300ms" {
		test.Errorf("cause %q", got.cause)
	}
	if elapsed := time.Since(started); elapsed > 1300*time.Millisecond {
		test.Errorf("the run lasted %v", elapsed)
	}
	time.Sleep(2500 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		test.Error("the child of the command was not killed")
	}
}

func TestAStopDuringARunGivesNothing(test *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	if _, finished := runCommand(stop, "sleep 5", test.TempDir(), 10*time.Second); finished {
		test.Error("the run was reported as finished")
	}
}
