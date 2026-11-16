//go:build linux

package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakeusb"
	"github.com/norareit/sensordeck/internal/uploadstate"
)

const (
	hungLine       = "screen hung: write /dev/hidraw3: connection timed out\n"
	suspendingLine = "suspending the screen to recover it\n"
	recoveredLine  = "screen recovered\n"
	noSuspendLine  = "the screen did not suspend within 7s, another program may hold it open\n"
	noRuleLine     = "the screen has hung. Installing the current udev rule lets sensordeck recover it, see " +
		`"Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen. ` +
		"Unplugging the screen once recovers it now\n"
)

// hungRun is a run whose first writes time out, as the entries of failWrites say, with the USB device in a temporary
// tree. The run stops after the given number of waits, and steps is called after each wait first.
type hungRun struct {
	harness *harness
	device  *fakeusb.Device
	steps   func(count int)
}

func newHungRun(test *testing.T, failures []bool, stopAfterWaits int) *hungRun {
	test.Helper()
	harness, device := hungHarness(test)
	harness.transport.writeErr = nil
	harness.transport.failWrites = failures
	hung := &hungRun{harness: harness, device: device}
	harness.onWait = func(count int) {
		if hung.steps != nil {
			hung.steps(count)
		}
		if count >= stopAfterWaits {
			harness.cancelStop()
		}
	}
	return hung
}

func (hung *hungRun) run(test *testing.T, config string) {
	test.Helper()
	if status := hung.harness.run("run", "--config", config); status != 0 {
		test.Fatalf("status %d, stderr %q", status, hung.harness.stderr.String())
	}
}

func (hung *hungRun) stderr() string { return hung.harness.stderr.String() }

// pollWaits counts the waits of 100 milliseconds, which are the reads of the status of the device.
func pollWaits(waits []time.Duration) int {
	count := 0
	for _, wait := range waits {
		if wait == 100*time.Millisecond {
			count++
		}
	}
	return count
}

func TestRunRecoversAHungScreenBySuspendingAndReopeningIt(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 5)
	hung.steps = func(count int) {
		if count == 4 {
			hung.device.Set("runtime_status", "suspended\n")
		}
	}
	hung.run(test, writeConfig(test, twoTags))
	harness := hung.harness
	if want := hungLine + suspendingLine + recoveredLine; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if got := kinds(test, harness.transport.writes); got != "CV" {
		test.Errorf("writes %s", got)
	}
	if want := []time.Duration{time.Second, 100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond}; len(harness.waits) < 4 ||
		strings.Join(durations(harness.waits[:4]), " ") != strings.Join(durations(want), " ") {
		test.Errorf("waits %v", harness.waits)
	}
	// one close for the recovery and one at the end of run
	if harness.transport.closed != 2 || harness.transport.reopens != 1 {
		test.Errorf("closed %d, reopened %d", harness.transport.closed, harness.transport.reopens)
	}
}

func durations(values []time.Duration) []string {
	texts := make([]string, len(values))
	for position, value := range values {
		texts[position] = value.String()
	}
	return texts
}

func TestRunChecksTheThemeAfterTheRecoveryOfAHungScreen(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	compiled := compileExpected(test, folder)
	hung := newHungRun(test, []bool{true}, 4)
	harness := hung.harness
	statePath := harness.useCacheDirectory(test)
	harness.transport.ackWhenEmpty, harness.transport.serial = true, "S1"
	if err := uploadstate.Write(statePath, map[string]string{"S1": uploadstate.Hash(compiled.File)}); err != nil {
		test.Fatal(err)
	}
	hung.steps = func(count int) {
		switch count {
		case 2:
			if err := os.Remove(statePath); err != nil {
				test.Fatal(err)
			}
		case 3:
			hung.device.Set("runtime_status", "suspended\n")
		}
	}
	hung.run(test, themeConfig(test, folder))
	if got := themeKinds(harness.transport.writes); got != "UCV" {
		test.Errorf("writes %s", got)
	}
	if want := hungLine + suspendingLine + recoveredLine; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
}

func TestRunOpensTheNodeAfterTheLimitWhenTheDeviceNeverSuspends(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 72)
	hung.run(test, writeConfig(test, twoTags))
	if want := hungLine + suspendingLine + noSuspendLine + recoveredLine; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if got := pollWaits(hung.harness.waits); got != 70 {
		test.Errorf("%d reads of the status", got)
	}
	if hung.harness.transport.reopens != 1 {
		test.Errorf("reopened %d times", hung.harness.transport.reopens)
	}
}

func TestRunWritesTheLinesOfARecoveryThatDoesNotHelpOnce(test *testing.T) {
	hung := newHungRun(test, []bool{true, true, true}, 3)
	hung.device.Set("runtime_status", "suspended\n")
	hung.run(test, writeConfig(test, twoTags))
	if want := hungLine + suspendingLine; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	// two closes of the recoveries and one at the end of run
	if hung.harness.transport.closed != 3 || hung.harness.transport.reopens != 2 {
		test.Errorf("closed %d, reopened %d", hung.harness.transport.closed, hung.harness.transport.reopens)
	}
}

func TestRunSaysThatTheDeviceDidNotSuspendOnceOverTwoRecoveries(test *testing.T) {
	hung := newHungRun(test, []bool{true, true, true}, 143)
	hung.run(test, writeConfig(test, twoTags))
	if want := hungLine + suspendingLine + noSuspendLine; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if hung.harness.transport.reopens != 2 {
		test.Errorf("reopened %d times", hung.harness.transport.reopens)
	}
}

func TestRunReopensAHungScreenPlainlyWhenNoRecoveryIsPossible(test *testing.T) {
	cases := []struct {
		name    string
		prepare func(*hungRun)
	}{
		{"control on", func(hung *hungRun) { hung.device.Set("control", "on\n") }},
		{"no USB device", func(hung *hungRun) {
			hung.harness.power = linuxPower{sysfsRoot: test.TempDir(), devRoot: test.TempDir(), node: func() string { return hung.device.Node }}
		}},
	}
	for _, each := range cases {
		test.Run(each.name, func(test *testing.T) {
			hung := newHungRun(test, []bool{true, true, true}, 4)
			each.prepare(hung)
			hung.run(test, writeConfig(test, twoTags))
			if want := hungLine + noRuleLine + recoveredLine; hung.stderr() != want {
				test.Errorf("stderr %q", hung.stderr())
			}
			// the plain reopenings of the rounds 2 to 4, and no close but the one at the end of run
			if hung.harness.transport.reopens != 3 || hung.harness.transport.closed != 1 {
				test.Errorf("reopened %d times, closed %d", hung.harness.transport.reopens, hung.harness.transport.closed)
			}
		})
	}
}

func TestRunReadsThePowerFilesAtEachRoundOfAHang(test *testing.T) {
	hung := newHungRun(test, []bool{true, true, true}, 4)
	hung.device.Set("control", "on\n")
	hung.device.Set("runtime_status", "suspended\n")
	hung.steps = func(count int) {
		if count == 2 {
			hung.device.Set("control", "auto\n")
		}
	}
	hung.run(test, writeConfig(test, twoTags))
	if want := hungLine + noRuleLine + suspendingLine + recoveredLine; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
}

func TestRunWritesThatAnUploadFailedBeforeThatTheScreenHung(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	hung := newHungRun(test, []bool{true}, 1)
	hung.harness.useCacheDirectory(test)
	hung.harness.transport.ackWhenEmpty, hung.harness.transport.serial = true, "S1"
	hung.run(test, themeConfig(test, folder))
	got := hung.stderr()
	failed, hungAt := strings.Index(got, "theme upload failed: "), strings.Index(got, "\nscreen hung: ")
	if failed != 0 || hungAt < 0 || strings.Count(got, "screen hung: ") != 1 || strings.Contains(got, "screen lost") {
		test.Errorf("stderr %q", got)
	}
}

func TestRunStopsDuringTheWaitForTheSuspend(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 3)
	hung.run(test, writeConfig(test, twoTags))
	if want := hungLine + suspendingLine; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if hung.harness.transport.reopens != 0 || len(hung.harness.waits) != 3 {
		test.Errorf("reopened %d times, waits %v", hung.harness.transport.reopens, hung.harness.waits)
	}
	if !strings.HasSuffix(hung.harness.stdout.String(), "stopped\n") {
		test.Errorf("stdout %q", hung.harness.stdout.String())
	}
}

func TestRunHandlesAnotherFailureOutsideAHangAsBefore(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 2)
	hung.harness.transport.failErr = errFake
	hung.run(test, writeConfig(test, twoTags))
	if want := "screen lost: fake failure\nscreen back\n"; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
}

func TestRunEndsAHangWithoutALineWhenAnotherFailureFollows(test *testing.T) {
	hung := newHungRun(test, []bool{true, true}, 3)
	hung.device.Set("runtime_status", "suspended\n")
	hung.steps = func(count int) {
		if count == 1 {
			hung.harness.transport.failErr = errFake
		}
	}
	hung.run(test, writeConfig(test, twoTags))
	if want := hungLine + suspendingLine + "screen lost: fake failure\nscreen back\n"; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
}

func TestRunWritesNothingInTheTicksOfAHang(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 6)
	hung.device.Set("runtime_status", "suspended\n")
	written := stampWrites(hung.harness, startOfTheRun())
	hung.run(test, writeConfig(test, "interval: 5s\n"+twoTags))
	if got, want := timeline(*written), "C5000 V5000"; got != want {
		test.Errorf("writes %s", got)
	}
}

func TestRunWritesTheLinesOfASecondHangAgain(test *testing.T) {
	// the clock and the values of a round make two writes, so the second hang begins at the first write of round 3
	hung := newHungRun(test, []bool{true, false, false, true, true}, 4)
	hung.device.Set("runtime_status", "suspended\n")
	hung.run(test, writeConfig(test, twoTags))
	want := hungLine + suspendingLine + recoveredLine + hungLine + suspendingLine
	if hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
}
