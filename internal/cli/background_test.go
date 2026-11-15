package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// backgroundHarness gives a harness of the system with a cache directory of its own and a run that is stopped after
// the waits.
func backgroundHarness(test *testing.T, system string, stopAfterWaits int) *harness {
	test.Helper()
	harness := runWith(twoMetrics(), stopAfterWaits)
	harness.system = system
	harness.cacheDir = test.TempDir()
	harness.cacheDirErr = nil
	return harness
}

func logLinesOf(test *testing.T, harness *harness) []string {
	test.Helper()
	content := readTestFile(test, logPath(harness.cacheDir))
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

// withoutTimes gives the lines of the log with the time at the front of each cut off, and checks that it is there.
func withoutTimes(test *testing.T, lines []string) []string {
	test.Helper()
	var texts []string
	for _, line := range lines {
		stamp, text, found := strings.Cut(line, " ")
		if _, err := time.Parse(logTimeForm, stamp); !found || err != nil {
			test.Errorf("the line %q has no time in front", line)
		}
		texts = append(texts, text)
	}
	return texts
}

func TestTheHelpOfRunDoesNotNameTheBackgroundFlag(test *testing.T) {
	harness := newHarness()
	if status := harness.run("run", "--help"); status != 0 || strings.Contains(harness.stdout.String(), "background") {
		test.Errorf("status %d, %q", status, harness.stdout.String())
	}
}

func TestABackgroundRunWritesItsLinesToTheLogAndNothingElseOnEverySystem(test *testing.T) {
	for _, system := range []string{"linux", "windows"} {
		harness := backgroundHarness(test, system, 2)
		if status := harness.run("run", "--background", "--config", writeConfig(test, twoTags)); status != 0 {
			test.Fatalf("%s: status %d, stderr %q", system, status, harness.stderr.String())
		}
		if harness.stdout.Len() != 0 || harness.stderr.Len() != 0 {
			test.Errorf("%s: stdout %q, stderr %q", system, harness.stdout.String(), harness.stderr.String())
		}
		lines := logLinesOf(test, harness)
		if !strings.HasPrefix(lines[0], "2026-11-14T15:04:05+00:00 running: 2 metrics every 1s") || !strings.HasSuffix(lines[len(lines)-1], " stopped") {
			test.Errorf("%s: log %q", system, lines)
		}
		withoutTimes(test, lines)
		wantEvent := 0
		if system == "windows" {
			wantEvent = 1
		}
		if harness.holds != wantEvent || harness.detaches != wantEvent {
			test.Errorf("%s: %d holds, %d detaches", system, harness.holds, harness.detaches)
		}
		if harness.stopEvent.closed != wantEvent {
			test.Errorf("%s: the event was closed %d times", system, harness.stopEvent.closed)
		}
	}
}

func TestABackgroundRunWithNoCacheDirectoryOrNoLogFailsOnStandardError(test *testing.T) {
	harness := newHarness()
	harness.expectFailure(test, "cannot find the cache directory: no cache directory in this test", "run", "--background", "--config", writeConfig(test, twoTags))

	harness = newHarness()
	harness.cacheDir = test.TempDir()
	harness.cacheDirErr = nil
	writeTestFile(test, filepath.Join(harness.cacheDir, "sensordeck"), "a file\n")
	path := logPath(harness.cacheDir)
	if status := harness.run("run", "--background", "--config", writeConfig(test, twoTags)); status != 1 ||
		!strings.HasPrefix(harness.stderr.String(), "sensordeck: cannot open the log "+path+": ") || harness.stdout.Len() != 0 {
		test.Errorf("status %d, stderr %q", status, harness.stderr.String())
	}
	if harness.holds != 0 || harness.detaches != 0 || len(harness.opened) != 0 {
		test.Errorf("%d holds, %d detaches, opened %v", harness.holds, harness.detaches, harness.opened)
	}
}

func TestOnWindowsTheEventIsHeldBeforeTheDetachAndASignalEndsTheRunWithTheTeardown(test *testing.T) {
	lighting := &fakeLighting{}
	harness := runKeeping(lighting, 1<<30)
	harness.system = "windows"
	harness.cacheDir = test.TempDir()
	harness.cacheDirErr = nil
	harness.onWait = func(count int) {
		if count == 2 {
			close(harness.stopEvent.channel)
		}
	}
	if status := harness.run("run", "--background", "--config", writeConfig(test, colourTags("10s"))); status != 0 {
		test.Fatalf("status %d, log %q", status, logLinesOf(test, harness))
	}
	if !reflect.DeepEqual(harness.steps, []string{"hold", "detach"}) {
		test.Errorf("steps %v", harness.steps)
	}
	lines := logLinesOf(test, harness)
	if lighting.teardowns != 1 || !strings.HasSuffix(lines[len(lines)-1], " stopped") || harness.stopEvent.closed != 1 {
		test.Errorf("%d teardowns, log %q, closed %d", lighting.teardowns, lines, harness.stopEvent.closed)
	}
	if harness.stdout.Len() != 0 || harness.stderr.Len() != 0 {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
}

func TestAFailingDetachIsIgnored(test *testing.T) {
	harness := backgroundHarness(test, "windows", 1)
	harness.detachErr = errors.New("no console")
	if status := harness.run("run", "--background", "--config", writeConfig(test, twoTags)); status != 0 || harness.detaches != 1 {
		test.Errorf("status %d, %d detaches, log %q", status, harness.detaches, logLinesOf(test, harness))
	}
}

func TestAStopEventThatExistsEndsTheRunWithTheLineOfTheLogAndNoDetach(test *testing.T) {
	harness := backgroundHarness(test, "windows", 1)
	harness.stopEventErr = errStopEventExists
	if status := harness.run("run", "--background", "--config", writeConfig(test, twoTags)); status != 1 {
		test.Errorf("status %d", status)
	}
	if got := withoutTimes(test, logLinesOf(test, harness)); !reflect.DeepEqual(got, []string{"another background run of sensordeck is running"}) {
		test.Errorf("log %q", got)
	}
	if harness.stdout.Len() != 0 || harness.stderr.Len() != 0 || harness.detaches != 0 || len(harness.opened) != 0 {
		test.Errorf("stdout %q, stderr %q, %d detaches, opened %v", harness.stdout.String(), harness.stderr.String(), harness.detaches, harness.opened)
	}
}

func TestAStopEventThatCannotBeCreatedEndsTheRunWithItsErrorInTheLog(test *testing.T) {
	harness := backgroundHarness(test, "windows", 1)
	harness.stopEventErr = errors.New("cannot create the stop event x: denied")
	if status := harness.run("run", "--background", "--config", writeConfig(test, twoTags)); status != 1 {
		test.Errorf("status %d", status)
	}
	if got := withoutTimes(test, logLinesOf(test, harness)); !reflect.DeepEqual(got, []string{"sensordeck: cannot create the stop event x: denied"}) {
		test.Errorf("log %q", got)
	}
	if harness.stderr.Len() != 0 || harness.detaches != 0 {
		test.Errorf("stderr %q, %d detaches", harness.stderr.String(), harness.detaches)
	}
}

func TestAStartThatFailsIsTriedAgainAfterTenSecondsWithTheConfigFileReadAnew(test *testing.T) {
	harness := backgroundHarness(test, "linux", 3)
	harness.openErr = errors.New("no screen")
	configFile := writeConfig(test, twoTags)
	changing := harness.onWait
	harness.onWait = func(count int) {
		if count == 1 {
			harness.openErr = nil
			if err := os.WriteFile(configFile, []byte("tags:\n  cpu.temp: 1\n"), 0o644); err != nil {
				test.Fatal(err)
			}
		}
		changing(count)
	}
	if status := harness.run("run", "--background", "--config", configFile); status != 0 {
		test.Fatalf("status %d, log %q", status, logLinesOf(test, harness))
	}
	want := []string{"sensordeck: no screen", "starting again in 10s", "running: 1 metrics every 1s", "stopped"}
	if got := withoutTimes(test, logLinesOf(test, harness)); !reflect.DeepEqual(got, want) {
		test.Errorf("log %q, want %q", got, want)
	}
	if harness.waits[0] != 10*time.Second {
		test.Errorf("waits %v", harness.waits)
	}
	if harness.stderr.Len() != 0 {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestAConfigThatDoesNotLoadAtTheFirstStartIsTriedAgain(test *testing.T) {
	harness := backgroundHarness(test, "linux", 3)
	configFile := writeConfig(test, "tags: [")
	changing := harness.onWait
	harness.onWait = func(count int) {
		if count == 1 {
			if err := os.WriteFile(configFile, []byte(twoTags), 0o644); err != nil {
				test.Fatal(err)
			}
		}
		changing(count)
	}
	if status := harness.run("run", "--background", "--config", configFile); status != 0 {
		test.Fatalf("status %d, log %q", status, logLinesOf(test, harness))
	}
	lines := withoutTimes(test, logLinesOf(test, harness))
	if len(lines) < 4 || !strings.HasPrefix(lines[0], "sensordeck: config file "+configFile) || lines[1] != "starting again in 10s" || lines[2] != "running: 2 metrics every 1s" {
		test.Errorf("log %q", lines)
	}
}

func TestAStopDuringTheWaitEndsTheRunAtOnce(test *testing.T) {
	for name, stopper := range map[string]func(harness *harness){
		"a signal of the process": func(harness *harness) { harness.cancelStop() },
		"the stop event":          func(harness *harness) { close(harness.stopEvent.channel) },
	} {
		harness := backgroundHarness(test, "windows", 1<<30)
		harness.openErr = errors.New("no screen")
		harness.onWait = func(count int) {
			if count == 1 {
				stopper(harness)
			}
		}
		if status := harness.run("run", "--background", "--config", writeConfig(test, twoTags)); status != 0 {
			test.Fatalf("%s: status %d", name, status)
		}
		lines := logLinesOf(test, harness)
		if !strings.HasSuffix(lines[len(lines)-1], " stopped") || harness.stopEvent.closed != 1 {
			test.Errorf("%s: log %q", name, lines)
		}
	}
}
