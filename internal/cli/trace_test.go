package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

func newTracing(inner transport, elapsed time.Duration) (*tracingTransport, *bytes.Buffer) {
	var out bytes.Buffer
	return &tracingTransport{inner: inner, out: &out, elapsed: func() time.Duration { return elapsed }}, &out
}

func TestTraceExampleLinesToTheCharacter(test *testing.T) {
	clock := &fakeClock{}
	fake := &fakeTransport{clock: clock, queue: []readResult{{report: reportOf(0x06)}}}
	tracing, out := newTracing(fake, 412*time.Microsecond)
	if err := tracing.Write(smartmonitor.ResetCommand()); err != nil {
		test.Fatal(err)
	}
	tracing.elapsed = func() time.Duration { return 1503377 * time.Microsecond }
	if _, err := tracing.Read(time.Second); err != nil {
		test.Fatal(err)
	}
	want := "    0.000412 > 01 72 65 73 65 74" + strings.Repeat(" 00", 58) + "\n" +
		"    1.503377 < 06" + strings.Repeat(" 00", 63) + "\n"
	if got := out.String(); got != want {
		test.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	if lines := strings.Split(strings.TrimSpace(out.String()), "\n"); len(strings.Fields(lines[0])) != 2+64 {
		test.Errorf("the line has %d fields", len(strings.Fields(lines[0])))
	}
}

func TestTraceLineFormsOfFailures(test *testing.T) {
	clock := &fakeClock{}
	fake := &fakeTransport{clock: clock, writeErr: errFake, reopenErr: errFake, queue: []readResult{{err: errors.New("gone")}}}
	tracing, out := newTracing(fake, time.Second)
	_ = tracing.Write(reportOf(0xAB))
	_, _ = tracing.Read(time.Second)
	_, _ = tracing.Read(time.Second)
	_ = tracing.Reopen()
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 4 {
		test.Fatalf("got %d lines: %q", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "    1.000000 > AB 00") || !strings.HasSuffix(lines[0], "00 ! fake failure") {
		test.Errorf("write failure: %q", lines[0])
	}
	if lines[1] != "    1.000000 < ! gone" {
		test.Errorf("read failure: %q", lines[1])
	}
	if lines[2] != "    1.000000 < ! "+smartmonitor.ErrReadTimeout.Error() {
		test.Errorf("passed time limit: %q", lines[2])
	}
	if lines[3] != "    1.000000 * reopen ! fake failure" {
		test.Errorf("reopen failure: %q", lines[3])
	}
}

func TestTraceReopenSucceeds(test *testing.T) {
	tracing, out := newTracing(&fakeTransport{clock: &fakeClock{}}, 2*time.Second)
	if err := tracing.Reopen(); err != nil {
		test.Fatal(err)
	}
	if got := out.String(); got != "    2.000000 * reopen\n" {
		test.Errorf("got %q", got)
	}
}

func TestTraceCloseWritesNothingAndCallsReachTheInnerTransport(test *testing.T) {
	fake := &fakeTransport{clock: &fakeClock{}}
	tracing, out := newTracing(fake, 0)
	report := reportOf(1, 2, 3)
	_ = tracing.Write(report)
	_ = tracing.Reopen()
	out.Reset()
	if err := tracing.Close(); err != nil {
		test.Fatal(err)
	}
	if out.Len() != 0 {
		test.Errorf("close wrote %q", out.String())
	}
	if len(fake.writes) != 1 || fake.writes[0] != report || fake.reopens != 1 || fake.closed != 1 {
		test.Errorf("the inner transport saw %+v", fake)
	}
}

func TestTraceGivesTheSerialOfTheInnerTransportAndWritesNoLine(test *testing.T) {
	fake := &fakeTransport{clock: &fakeClock{}, serial: "2022100600"}
	tracing, out := newTracing(fake, 0)
	if got := tracing.Serial(); got != "2022100600" || out.Len() != 0 {
		test.Errorf("Serial = %q, trace %q", got, out.String())
	}
}
