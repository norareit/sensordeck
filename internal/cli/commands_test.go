package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// harness runs the command line against a fake environment.
type harness struct {
	stdout, stderr bytes.Buffer
	clock          *fakeClock
	transport      *fakeTransport
	opened         []string // the device asked for by each opening
	screens        []string
	openErr        error
	sources        []sensors.Source
	sourcesErr     error
	configDir      string // what userConfigDir gives
	configDirErr   error
	stopped        context.Context // ends when cancelStop is called, which is what a signal does in the real command
	cancelStop     context.CancelFunc
	waits          []time.Duration
	onWait         func(count int) // called after each wait has moved the clock, with the count of waits so far
}

func newHarness() *harness {
	clock := &fakeClock{current: time.Date(2026, 11, 14, 15, 4, 5, 0, time.UTC)}
	stopped, cancelStop := context.WithCancel(context.Background())
	return &harness{clock: clock, transport: &fakeTransport{clock: clock}, stopped: stopped, cancelStop: cancelStop}
}

func (harness *harness) run(args ...string) int {
	return run(args, environment{
		stdout:    &harness.stdout,
		stderr:    &harness.stderr,
		sysfsRoot: "/sysroot",
		devRoot:   "/devroot",
		procRoot:  "/procroot",
		now:       harness.clock.now,
		sleep:     harness.clock.sleep,
		listScreens: func(sysfsRoot, devRoot string) ([]string, error) {
			return harness.screens, nil
		},
		openTransport: func(sysfsRoot, devRoot, device string) (transport, error) {
			harness.opened = append(harness.opened, device)
			if harness.openErr != nil {
				return nil, harness.openErr
			}
			return harness.transport, nil
		},
		userConfigDir: func() (string, error) { return harness.configDir, harness.configDirErr },
		newSources: func(sysfsRoot, procRoot string) ([]sensors.Source, error) {
			if sysfsRoot != "/sysroot" || procRoot != "/procroot" {
				return nil, errors.New("the roots were not passed on")
			}
			return harness.sources, harness.sourcesErr
		},
		notifyStop: func(parent context.Context) (context.Context, context.CancelFunc) {
			return harness.stopped, func() {}
		},
		wait: func(stop context.Context, duration time.Duration) bool {
			if stop.Err() != nil {
				return false
			}
			harness.clock.sleep(duration)
			harness.waits = append(harness.waits, duration)
			if harness.onWait != nil {
				harness.onWait(len(harness.waits))
			}
			return stop.Err() == nil
		},
	})
}

func (harness *harness) expectFailure(test *testing.T, want string, args ...string) {
	test.Helper()
	if status := harness.run(args...); status != 1 {
		test.Errorf("%v: status %d", args, status)
	}
	if got := harness.stderr.String(); got != "sensordeck: "+want+"\n" {
		test.Errorf("%v: stderr %q, want %q", args, got, want)
	}
	if harness.stdout.Len() != 0 {
		test.Errorf("%v: stdout %q", args, harness.stdout.String())
	}
}

func TestHelpNamesTheCommandsAndTheOptions(test *testing.T) {
	for _, args := range [][]string{{}, {"--help"}} {
		harness := newHarness()
		if status := harness.run(args...); status != 0 {
			test.Errorf("%v: status %d", args, status)
		}
		help := harness.stdout.String()
		for _, name := range []string{"devices", "clock", "values", "upload", "listen", "raw", "sensors", "--device", "--trace"} {
			if !strings.Contains(help, name) {
				test.Errorf("%v: the help does not name %s", args, name)
			}
		}
		if strings.Contains(help, "completion") {
			test.Errorf("%v: the help names completion", args)
		}
		if harness.stderr.Len() != 0 {
			test.Errorf("%v: stderr %q", args, harness.stderr.String())
		}
	}
	harness := newHarness()
	if status := harness.run("values", "--help"); status != 0 || !strings.Contains(harness.stdout.String(), "values") {
		test.Errorf("values --help: status %d, %q", status, harness.stdout.String())
	}
}

func TestUnknownCommandAndOption(test *testing.T) {
	for _, args := range [][]string{{"nothing"}, {"completion"}, {"--nothing"}, {"clock", "--nothing"}} {
		harness := newHarness()
		if status := harness.run(args...); status != 1 {
			test.Errorf("%v: status %d", args, status)
		}
		stderr := harness.stderr.String()
		if !strings.HasPrefix(stderr, "sensordeck: ") || strings.Count(stderr, "\n") != 1 || harness.stdout.Len() != 0 {
			test.Errorf("%v: stderr %q stdout %q", args, stderr, harness.stdout.String())
		}
	}
}

func TestDeviceOptionBeforeAndAfterTheCommand(test *testing.T) {
	for _, args := range [][]string{{"--device", "x", "values", "1=1"}, {"values", "1=1", "--device", "x"}} {
		harness := newHarness()
		if status := harness.run(args...); status != 0 {
			test.Fatalf("%v: status %d, %q", args, status, harness.stderr.String())
		}
		if len(harness.opened) != 1 || harness.opened[0] != "x" {
			test.Errorf("%v: opened %v", args, harness.opened)
		}
	}
}

func TestTraceOptionBeforeAndAfterTheCommand(test *testing.T) {
	for _, args := range [][]string{{"--trace", "values", "1=1"}, {"values", "1=1", "--trace"}} {
		harness := newHarness()
		if status := harness.run(args...); status != 0 {
			test.Fatalf("%v: status %d", args, status)
		}
		if !strings.HasPrefix(harness.stderr.String(), "    0.000000 > 02 01 01 00 01 00") {
			test.Errorf("%v: stderr %q", args, harness.stderr.String())
		}
	}
	harness := newHarness()
	harness.run("values", "1=1")
	if harness.stderr.Len() != 0 {
		test.Errorf("without --trace: stderr %q", harness.stderr.String())
	}
}

func TestDevicesWithScreens(test *testing.T) {
	harness := newHarness()
	harness.screens = []string{"/dev/hidraw2\t0483:0065\ta\t-", "/dev/hidraw10\t0483:0065\t-\tS1"}
	if status := harness.run("devices", "--device", "ignored"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if got, want := harness.stdout.String(), harness.screens[0]+"\n"+harness.screens[1]+"\n"; got != want {
		test.Errorf("stdout %q", got)
	}
	if len(harness.opened) != 0 {
		test.Errorf("devices opened %v", harness.opened)
	}
}

func TestDevicesWithNoScreen(test *testing.T) {
	harness := newHarness()
	if status := harness.run("devices"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if harness.stdout.Len() != 0 || harness.stderr.String() != "no supported screen found\n" || len(harness.opened) != 0 {
		test.Errorf("stdout %q stderr %q opened %v", harness.stdout.String(), harness.stderr.String(), harness.opened)
	}
}

func TestWrongArgumentsFailBeforeTheTransportIsOpened(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "missing.dat")
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"clock", "now"}, `unknown command "now" for "sensordeck clock"`},
		{[]string{"values"}, "values needs at least one <tag>=<value>"},
		{[]string{"values", "1"}, `invalid argument "1": want <tag>=<value>`},
		{[]string{"upload"}, "accepts 1 arg(s), received 0"},
		{[]string{"upload", missing}, "cannot read the theme file " + missing + ": no such file or directory"},
		{[]string{"listen", "extra"}, `unknown command "extra" for "sensordeck listen"`},
		{[]string{"listen", "--seconds", "0"}, `invalid --seconds "0": want a whole number of 1 or more`},
		{[]string{"raw"}, "raw needs at least one byte"},
		{[]string{"raw", "0x03"}, `invalid argument "0x03": want pairs of hexadecimal digits`},
		{[]string{"raw", "03", "--seconds", "x"}, `invalid --seconds "x": want a whole number of 1 or more`},
	}
	for _, scenario := range cases {
		harness := newHarness()
		harness.expectFailure(test, scenario.want, scenario.args...)
		if len(harness.opened) != 0 {
			test.Errorf("%v: opened the transport", scenario.args)
		}
	}
}

func TestOpeningFailureIsReported(test *testing.T) {
	harness := newHarness()
	harness.openErr = errFake
	harness.expectFailure(test, "fake failure", "values", "1=1")
}

func TestClock(test *testing.T) {
	harness := newHarness()
	if status := harness.run("clock"); status != 0 {
		test.Fatalf("status %d: %q", status, harness.stderr.String())
	}
	want := reportOf(0x03, 0x01, 0x15, 26, 11, 14, 15, 4, 5, 6, 0x64)
	if len(harness.transport.writes) != 1 || harness.transport.writes[0] != want {
		test.Errorf("writes %v", harness.transport.writes)
	}
	if len(harness.transport.readLimits) != 0 {
		test.Error("clock read")
	}
	if got := harness.stdout.String(); got != "clock sent: 2026-11-14 15:04:05\n" {
		test.Errorf("stdout %q", got)
	}
	if harness.transport.closed != 1 {
		test.Errorf("closed %d times", harness.transport.closed)
	}
}

func TestValues(test *testing.T) {
	harness := newHarness()
	if status := harness.run("values", "1=0x1234", "255=1"); status != 0 {
		test.Fatalf("status %d: %q", status, harness.stderr.String())
	}
	if len(harness.transport.writes) != 1 || harness.transport.writes[0] != reportOf(0x02, 0x02, 0x01, 0x12, 0x34, 0xFF, 0x00, 0x01) {
		test.Errorf("writes %v", harness.transport.writes)
	}
	if got := harness.stdout.String(); got != "values sent: 2, reports written: 1\n" {
		test.Errorf("stdout %q", got)
	}
	if harness.transport.closed != 1 {
		test.Errorf("closed %d times", harness.transport.closed)
	}
}

func writeTheme(test *testing.T, size int) string {
	test.Helper()
	theme := make([]byte, size)
	for position := range theme {
		theme[position] = byte(position)
	}
	path := filepath.Join(test.TempDir(), "theme.dat")
	if err := os.WriteFile(path, theme, 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

func TestUploadComplete(test *testing.T) {
	harness := newHarness()
	harness.transport.ackWhenEmpty = true
	if status := harness.run("upload", writeTheme(test, 2500)); status != 0 {
		test.Fatalf("status %d: %q", status, harness.stderr.String())
	}
	want := `reset
reopening after the reset
ymodem command
header block
data block 1 of 3
data block 2 of 3
data block 3 of 3
end of transmission
closing block
reopening after the upload
upload complete: 2500 bytes in 3 data blocks
`
	if got := harness.stdout.String(); got != want {
		test.Errorf("stdout\n%s", got)
	}
	if harness.transport.reopens != 2 || harness.transport.closed != 1 {
		test.Errorf("reopens %d closed %d", harness.transport.reopens, harness.transport.closed)
	}
	if harness.clock.current.Sub(time.Date(2026, 11, 14, 15, 4, 5, 0, time.UTC)) != 2*time.Second {
		test.Error("the pauses did not go through the sleep function of the environment")
	}
}

func TestUploadFailsOnAnUnexpectedAnswer(test *testing.T) {
	harness := newHarness()
	harness.transport.queue = []readResult{{report: reportOf(0x43)}}
	if status := harness.run("upload", writeTheme(test, 2500)); status != 1 {
		test.Fatalf("status %d", status)
	}
	if got := harness.stdout.String(); got != "reset\nreopening after the reset\nymodem command\n" {
		test.Errorf("stdout %q", got)
	}
	if got := harness.stderr.String(); got != "sensordeck: upload failed at the ymodem command: unexpected answer 0x43\n" {
		test.Errorf("stderr %q", got)
	}
	if harness.transport.closed != 1 {
		test.Errorf("closed %d times", harness.transport.closed)
	}
}

func TestListen(test *testing.T) {
	harness := newHarness()
	harness.transport.queue = []readResult{{report: reportOf(0x11)}, {report: reportOf(0x22)}}
	if status := harness.run("listen", "--seconds", "1"); status != 0 {
		test.Fatalf("status %d: %q", status, harness.stderr.String())
	}
	lines := strings.Split(strings.TrimSuffix(harness.stdout.String(), "\n"), "\n")
	if len(lines) != 3 || lines[0] != traceLine(0, readBody(reportOf(0x11))) ||
		lines[1] != traceLine(0, readBody(reportOf(0x22))) || lines[2] != "reports received: 2" {
		test.Errorf("stdout %q", lines)
	}
	if len(harness.transport.writes) != 0 {
		test.Error("listen wrote")
	}
	for _, limit := range harness.transport.readLimits {
		if limit > time.Second {
			test.Errorf("a read waited %v", limit)
		}
	}
	if harness.transport.closed != 1 {
		test.Errorf("closed %d times", harness.transport.closed)
	}
}

func TestListenDefaultsToFiveSecondsAndFailsOnAReadError(test *testing.T) {
	harness := newHarness()
	harness.run("listen")
	if got := harness.transport.readLimits[0]; got != 5*time.Second {
		test.Errorf("first limit %v", got)
	}
	failing := newHarness()
	failing.transport.queue = []readResult{{err: errFake}}
	failing.expectFailure(test, "fake failure", "listen")
}

func TestRaw(test *testing.T) {
	harness := newHarness()
	harness.transport.queue = []readResult{{report: reportOf(0x43)}}
	if status := harness.run("raw", "03", "01", "15"); status != 0 {
		test.Fatalf("status %d: %q", status, harness.stderr.String())
	}
	if len(harness.transport.writes) != 1 || harness.transport.writes[0] != reportOf(0x03, 0x01, 0x15) {
		test.Errorf("writes %v", harness.transport.writes)
	}
	want := traceLine(0, readBody(reportOf(0x43))) + "\nreports received: 1\n"
	if got := harness.stdout.String(); got != want {
		test.Errorf("stdout %q", got)
	}
	if got := harness.transport.readLimits[0]; got != time.Second {
		test.Errorf("first limit %v", got)
	}
	if harness.transport.closed != 1 {
		test.Errorf("closed %d times", harness.transport.closed)
	}
}

func TestRawFailsWhenTheWriteFails(test *testing.T) {
	harness := newHarness()
	harness.transport.writeErr = errFake
	harness.expectFailure(test, "fake failure", "raw", "03")
	if harness.transport.closed != 1 {
		test.Errorf("closed %d times", harness.transport.closed)
	}
}

var _ smartmonitor.Transport = (*fakeTransport)(nil)
