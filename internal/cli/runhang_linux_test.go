//go:build linux

package cli

import (
	"os"
	"strings"
	"testing"
	"time"
)

const (
	hungLine        = "screen hung: write /dev/hidraw3: connection timed out\n"
	restartingText  = "restarting the screen to recover it\n"
	restarted33Line = "screen restarted at string descriptor 33\n"
	recoveredLine   = "screen recovered\n"
	notRestartLine  = "the screen did not restart, trying again every minute. Unplugging the screen once recovers it\n"
	stillHungText   = "the screen is still hung after its restart, trying again every minute. Unplugging the screen once recovers it\n"
	permissionError = "open /dev/bus/usb/001/005: permission denied"
	cannotRestart   = "the screen has hung and sensordeck cannot restart it: " + permissionError +
		`. Installing the current udev rule lets sensordeck recover it, see "Linux: access to the screen" in ` +
		"https://github.com/norareit/sensordeck#linux-access-to-the-screen. Unplugging the screen once recovers it now\n"
)

var (
	openedAt33     = restartResult{outcome: restartedOpen, index: 33}
	notOpenedAt33  = restartResult{outcome: restartedNotOpen, index: 33}
	didNotRestart  = restartResult{outcome: notRestarted}
	notPossibleNow = restartResult{outcome: restartNotPossible, err: errPermission}
)

var errPermission = &os.PathError{Op: "open", Path: "/dev/bus/usb/001/005", Err: os.ErrPermission}

// hungRun is a run whose first writes time out, as the entries of failWrites say, with a restarter whose restarts take
// 2 seconds. The run stops after the given number of waits.
type hungRun struct {
	harness   *harness
	restarter *fakeRestarter
}

func newHungRun(test *testing.T, failures []bool, stopAfterWaits int, outcomes ...restartResult) *hungRun {
	test.Helper()
	harness, restarter := hungHarness(test, outcomes...)
	harness.transport.writeErr = nil
	harness.transport.failWrites = failures
	restarter.duration = 2 * time.Second
	harness.onWait = func(count int) {
		if count >= stopAfterWaits {
			harness.cancelStop()
		}
	}
	return &hungRun{harness: harness, restarter: restarter}
}

func (hung *hungRun) run(test *testing.T, config string) {
	test.Helper()
	if status := hung.harness.run("run", "--config", config); status != 0 {
		test.Fatalf("status %d, stderr %q", status, hung.harness.stderr.String())
	}
}

func (hung *hungRun) stderr() string { return hung.harness.stderr.String() }

// restartTimes gives when each restart began, in seconds after the first.
func (hung *hungRun) restartTimes() []int {
	var seconds []int
	for _, began := range hung.restarter.began {
		seconds = append(seconds, int(began.Sub(hung.restarter.began[0])/time.Second))
	}
	return seconds
}

func (hung *hungRun) checkNoSuspendLines(test *testing.T) {
	test.Helper()
	for _, forbidden := range []string{"suspend", "power/"} {
		if strings.Contains(hung.stderr(), forbidden) {
			test.Errorf("stderr holds %q: %q", forbidden, hung.stderr())
		}
	}
}

func TestRunRestartsAHungScreenAndGoesOn(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 3, openedAt33)
	hung.run(test, writeConfig(test, twoTags))
	want := hungLine + restartingText + restarted33Line + recoveredLine
	if hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if got := kinds(test, hung.harness.transport.writes); !strings.HasPrefix(got, "CV") {
		test.Errorf("writes %s", got)
	}
	if len(hung.restarter.began) != 1 || hung.harness.transport.reopens != 1 {
		test.Errorf("restarts %d, reopenings %d", len(hung.restarter.began), hung.harness.transport.reopens)
	}
	hung.checkNoSuspendLines(test)
}

func TestRunUploadsNothingAfterTheRestartOfAHungScreen(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	hung := newHungRun(test, nil, 4, openedAt33)
	hung.harness.transport.ackWhenEmpty = true
	stopping := hung.harness.onWait
	hung.harness.onWait = func(count int) {
		if count == 1 {
			// the upload of the start is done, so the next write, a clock, times out
			hung.harness.transport.failWrites = []bool{true}
		}
		stopping(count)
	}
	hung.run(test, themeConfig(test, folder))
	got := themeKinds(hung.harness.transport.writes)
	if !strings.HasPrefix(got, "UC") || strings.Count(got, "U") != 1 || len(hung.restarter.began) != 1 {
		test.Errorf("writes %s, %d restarts", got, len(hung.restarter.began))
	}
}

// uploadAfterAFailedStart runs a theme config whose upload of the start fails at its first write, as the harness says, and
// gives the moments, since the start of the run, at which the uploads that began after it did. The failed first write is
// not recorded.
func uploadAfterAFailedStart(test *testing.T, hung *hungRun) []time.Duration {
	test.Helper()
	hung.harness.transport.ackWhenEmpty = true
	written := stampWrites(hung.harness, startOfTheRun())
	hung.run(test, themeConfig(test, writeThemeFolder(test, compiledTheme)))
	return uploadStarts(*written)
}

func TestRunMakesTheUploadInTheRoundOfARestartThatRecoveredTheScreen(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 8, openedAt33)
	starts := uploadAfterAFailedStart(test, hung)
	if len(starts) != 1 || starts[0] >= 10*time.Second {
		test.Fatalf("uploads began at %v", starts)
	}
	if got := themeKinds(hung.harness.transport.writes); !strings.HasPrefix(got, "UCV") {
		test.Errorf("writes %s", got)
	}
}

func TestRunMakesTheUploadInTheRoundThatOpenedTheScreenAfterItsRestart(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 10, notOpenedAt33)
	hung.harness.transport.reopenResults = []error{errFake, nil}
	starts := uploadAfterAFailedStart(test, hung)
	if len(starts) != 1 || starts[0] >= 10*time.Second || len(hung.restarter.began) != 1 {
		test.Fatalf("uploads began at %v, %d restarts", starts, len(hung.restarter.began))
	}
	if got := themeKinds(hung.harness.transport.writes); !strings.HasPrefix(got, "UCV") {
		test.Errorf("writes %s", got)
	}
}

func TestRunWaitsTenSecondsForTheUploadWhenNoRestartRecoveredTheScreen(test *testing.T) {
	cases := []struct {
		name    string
		outcome restartResult
		failure error
	}{
		{"did not restart", didNotRestart, timeoutError},
		{"restart not possible", notPossibleNow, timeoutError},
		{"failed write without a hang", openedAt33, errFake},
	}
	for _, each := range cases {
		hung := newHungRun(test, []bool{true}, 14, each.outcome)
		hung.harness.transport.failErr = each.failure
		starts := uploadAfterAFailedStart(test, hung)
		if len(starts) != 1 || starts[0] < 10*time.Second {
			test.Errorf("%s: uploads began at %v", each.name, starts)
		}
	}
}

func TestRunWaitsTwentySecondsAfterTheUploadMadeAtOnceFailsToo(test *testing.T) {
	// the first upload fails, and so does the one made at once after the restart
	hung := newHungRun(test, []bool{true, true}, 40, openedAt33)
	starts := uploadAfterAFailedStart(test, hung)
	if len(hung.restarter.began) != 1 || len(starts) != 1 {
		test.Fatalf("%d restarts, uploads began at %v", len(hung.restarter.began), starts)
	}
	recovered := hung.restarter.began[0].Sub(startOfTheRun()) + hung.restarter.duration
	if starts[0] < recovered+20*time.Second {
		test.Errorf("the upload began at %v, the screen was recovered at %v", starts[0], recovered)
	}
}

func TestRunWritesTheLineOfARestartThatDidNotHelpOnceAndTriesAgainAfterAMinute(test *testing.T) {
	failures := make([]bool, 200)
	for position := range failures {
		failures[position] = true
	}
	hung := newHungRun(test, failures, 70, didNotRestart)
	hung.run(test, writeConfig(test, twoTags))
	want := hungLine + restartingText + notRestartLine
	if hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	times := hung.restartTimes()
	if len(times) < 2 || times[1] < 60 {
		test.Fatalf("restarts began at %v", times)
	}
	// every round between the restarts opens the node plainly
	if hung.harness.transport.reopens < 10 {
		test.Errorf("%d reopenings", hung.harness.transport.reopens)
	}
	hung.checkNoSuspendLines(test)
}

func TestRunWritesTheLineOfAScreenStillHungOnceAndRestartsAfterAMinute(test *testing.T) {
	failures := make([]bool, 200)
	for position := range failures {
		failures[position] = true
	}
	hung := newHungRun(test, failures, 70, openedAt33)
	hung.run(test, writeConfig(test, twoTags))
	got := hung.stderr()
	if !strings.HasPrefix(got, hungLine+restartingText+restarted33Line+stillHungText) || strings.Count(got, "still hung") != 1 {
		test.Errorf("stderr %q", got)
	}
	times := hung.restartTimes()
	if len(times) < 2 || times[1] < 60 {
		test.Fatalf("restarts began at %v", times)
	}
	if strings.Count(got, restartingText) != 1 || strings.Contains(got, "screen lost") || strings.Contains(got, "screen back") {
		test.Errorf("stderr %q", got)
	}
}

func TestRunWritesTheLineOfARestartThatIsNotPossibleOnceAndTriesInEachRound(test *testing.T) {
	hung := newHungRun(test, []bool{true, true, true}, 5, notPossibleNow)
	hung.run(test, writeConfig(test, twoTags))
	want := hungLine + restartingText + cannotRestart + recoveredLine
	if hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if len(hung.restarter.began) != 3 || hung.harness.transport.reopens != 3 {
		test.Errorf("restarts %d, reopenings %d", len(hung.restarter.began), hung.harness.transport.reopens)
	}
}

func TestRunOpensAScreenThatIsNotOpenAfterItsRestartWithNoNewRestart(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 4, notOpenedAt33)
	hung.harness.transport.reopenResults = []error{errFake, errFake, nil}
	hung.run(test, writeConfig(test, twoTags))
	want := hungLine + restartingText + restarted33Line + recoveredLine
	if hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if len(hung.restarter.began) != 1 || hung.harness.transport.reopens != 3 {
		test.Errorf("restarts %d, reopenings %d", len(hung.restarter.began), hung.harness.transport.reopens)
	}
	if got := kinds(test, hung.harness.transport.writes); !strings.HasPrefix(got, "CV") {
		test.Errorf("writes %s", got)
	}
}

func TestRunWritesThatAnUploadFailedBeforeThatTheScreenHung(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	hung := newHungRun(test, []bool{true}, 1, openedAt33)
	hung.harness.transport.ackWhenEmpty = true
	hung.run(test, themeConfig(test, folder))
	got := hung.stderr()
	failed, hungAt := strings.Index(got, "theme upload failed: "), strings.Index(got, "\nscreen hung: ")
	if failed != 0 || hungAt < 0 || strings.Count(got, "screen hung: ") != 1 || strings.Contains(got, "screen lost") {
		test.Errorf("stderr %q", got)
	}
}

func TestRunHandlesAnotherFailureOutsideAHangAsBefore(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 2, openedAt33)
	hung.harness.transport.failErr = errFake
	hung.run(test, writeConfig(test, twoTags))
	if want := "screen lost: fake failure\nscreen back\n"; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if len(hung.restarter.began) != 0 {
		test.Errorf("%d restarts", len(hung.restarter.began))
	}
}

func TestRunEndsAHangWithoutALineWhenAnotherFailureFollows(test *testing.T) {
	hung := newHungRun(test, []bool{true, true}, 3, didNotRestart)
	hung.restarter.onRestart = func(int) { hung.harness.transport.failErr = errFake }
	hung.run(test, writeConfig(test, twoTags))
	want := hungLine + restartingText + notRestartLine + "screen lost: fake failure\nscreen back\n"
	if hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
}

func TestRunWritesNothingInTheTicksOfAHang(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 6, openedAt33)
	written := stampWrites(hung.harness, startOfTheRun())
	hung.run(test, writeConfig(test, "interval: 5s\n"+twoTags))
	if got := timeline(*written); !strings.HasPrefix(got, "C7000") {
		test.Errorf("writes %s", got)
	}
}

func TestRunWritesTheLinesOfASecondHangAgain(test *testing.T) {
	// the clock and the values of a round make two writes, so the second hang begins at the first write of round 3
	// that comes more than a minute after the first restart, which the failures at the end of the list make
	failures := make([]bool, 0, 200)
	failures = append(failures, true)
	for len(failures) < 150 {
		failures = append(failures, false)
	}
	hung := newHungRun(test, failures, 80, openedAt33)
	hung.harness.onWait = func(count int) {
		if count == 70 {
			hung.harness.transport.failWrites = []bool{true}
		}
		if count >= 80 {
			hung.harness.cancelStop()
		}
	}
	hung.run(test, writeConfig(test, twoTags))
	want := hungLine + restartingText + restarted33Line + recoveredLine + hungLine + restartingText + restarted33Line + recoveredLine
	if hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if times := hung.restartTimes(); len(times) != 2 || times[1] < 60 {
		test.Errorf("restarts began at %v", times)
	}
}

func TestRunStopsDuringTheRestartWithNoLineOfItsResult(test *testing.T) {
	hung := newHungRun(test, []bool{true}, 100, openedAt33)
	hung.restarter.onRestart = func(int) { hung.harness.cancelStop() }
	hung.run(test, writeConfig(test, twoTags))
	if want := hungLine + restartingText; hung.stderr() != want {
		test.Errorf("stderr %q", hung.stderr())
	}
	if !strings.HasSuffix(hung.harness.stdout.String(), "stopped\n") {
		test.Errorf("stdout %q", hung.harness.stdout.String())
	}
}
