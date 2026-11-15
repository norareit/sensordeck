//go:build linux

package e2e

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakescreen"
)

// waitForLog waits until the log holds the text and gives the lines of the log.
func waitForLog(test *testing.T, path, text string) []string {
	test.Helper()
	deadline := time.Now().Add(reportLimit)
	for {
		content, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			test.Fatal(err)
		}
		if strings.Contains(string(content), text) {
			return strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
		}
		if time.Now().After(deadline) {
			test.Fatalf("the log %s does not hold %q: %q", path, text, content)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// expectTimes checks that each line begins with a time as the log writes it, and gives the lines without it.
func expectTimes(test *testing.T, lines []string) []string {
	test.Helper()
	var texts []string
	for _, line := range lines {
		stamp, text, found := strings.Cut(line, " ")
		if _, err := time.Parse("2006-01-02T15:04:05-07:00", stamp); !found || err != nil {
			test.Errorf("the line %q has no time in front", line)
		}
		texts = append(texts, text)
	}
	return texts
}

func TestRunInTheBackgroundWritesItsLinesToTheLogAndEndsOnSIGTERM(test *testing.T) {
	fake, screen, environment, config := runningSetup(test)
	cache := test.TempDir()
	environment = append(environment, "XDG_CACHE_HOME="+cache)
	log := filepath.Join(cache, "sensordeck", "run.log")
	running := start(test, fake, environment, "run", "--background", "--config", config)
	if _, reached := screen.WaitReports(2, reportLimit); !reached {
		test.Fatal("the screen got no report")
	}
	lines := waitForLog(test, log, "running: 3 metrics every 100ms")
	if got := expectTimes(test, lines); got[0] != "running: 3 metrics every 100ms" {
		test.Errorf("log %q", got)
	}
	ended, _ := running.stop(syscall.SIGTERM)
	if ended.status != 0 || ended.stdout != "" || ended.stderr != "" {
		test.Errorf("status %d, stdout %q, stderr %q", ended.status, ended.stdout, ended.stderr)
	}
	lines = expectTimes(test, waitForLog(test, log, "stopped"))
	if lines[len(lines)-1] != "stopped" {
		test.Errorf("log %q", lines)
	}
}

func TestRunInTheBackgroundWithNoScreenWritesTheErrorAndTriesAgain(test *testing.T) {
	fake := fakescreen.New(test)
	tree, _ := runningTree(test, fake)
	environment := environmentFor(test, tree, "")
	cache := test.TempDir()
	environment = append(environment, "XDG_CACHE_HOME="+cache)
	log := filepath.Join(cache, "sensordeck", "run.log")
	running := start(test, fake, environment, "run", "--background", "--config", writeConfigFile(test, runConfig))
	lines := expectTimes(test, waitForLog(test, log, "starting again in 10s"))
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "sensordeck: ") || lines[1] != "starting again in 10s" {
		test.Errorf("log %q", lines)
	}
	select {
	case err := <-running.done:
		test.Fatalf("the run ended after the failed start: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	ended, _ := running.stop(syscall.SIGTERM)
	if ended.status != 0 || ended.stdout != "" || ended.stderr != "" {
		test.Errorf("status %d, stdout %q, stderr %q", ended.status, ended.stdout, ended.stderr)
	}
	lines = expectTimes(test, waitForLog(test, log, "stopped"))
	if lines[len(lines)-1] != "stopped" {
		test.Errorf("log %q", lines)
	}
}
