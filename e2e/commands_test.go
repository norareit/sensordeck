//go:build linux

// The suite runs the built executable against a fake screen. What it leaves out, and where that is covered instead:
//
//   - a reopening that fails all 20 times would take over 10 seconds, and is covered by the tests of the upload in
//     internal/smartmonitor
//   - a passed time limit of an answer, 10 seconds, is covered by the transport tests with a short limit and by the
//     tests of the upload in internal/smartmonitor
//   - the denied permission is covered by the transport tests, which test the text on the function that makes it,
//     because the tests may run as root
//   - the build for Windows is covered by the two commands of the suite that build and vet for Windows, and its
//     message is not run
//   - a reopening that fails before it succeeds is covered by the tests of the feeder in internal/cli, which move the
//     time themselves and fail the reopenings as they like
//   - the GPU metrics in `run`, with no nvidia-smi in these tests, are covered by the tests of the feeder in
//     internal/cli and of the GPU source in internal/nvidiasmi
//   - `cpu.util`, which needs two samples that differ, is covered by the tests of the sources in
//     internal/linuxsensors, and the time limit of nvidia-smi by the tests of internal/nvidiasmi
//   - the exact value of a network rate other than 0, which depends on the time between two rounds, counters that go
//     back, and interfaces that appear or vanish, are covered by the tests of the network source in
//     internal/linuxsensors
package e2e

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

func reportOf(data ...byte) smartmonitor.Report {
	var report smartmonitor.Report
	copy(report[:], data)
	return report
}

func (outcome result) expect(test *testing.T, status int, stdout, stderr string) {
	test.Helper()
	if outcome.status != status || outcome.stdout != stdout || outcome.stderr != stderr {
		test.Errorf("status %d, stdout %q, stderr %q\nwant status %d, stdout %q, stderr %q",
			outcome.status, outcome.stdout, outcome.stderr, status, stdout, stderr)
	}
}

func TestDevicesWithNoScreen(test *testing.T) {
	run(test, fakescreen.New(test), "devices").expect(test, 0, "", "no supported screen found\n")
}

func TestDevicesWithOneScreen(test *testing.T) {
	fake := fakescreen.New(test)
	fake.Add(3, "llhmi.com llTechCo.,Ltd", "")
	run(test, fake, "devices").expect(test, 0, fake.DevRoot+"/hidraw3\t0483:0065\tllhmi.com llTechCo.,Ltd\t-\n", "")
}

func TestDevicesWithTwoScreens(test *testing.T) {
	fake := fakescreen.New(test)
	fake.Add(10, "other", "S1")
	fake.Add(2, "llhmi.com llTechCo.,Ltd", "")
	want := fake.DevRoot + "/hidraw2\t0483:0065\tllhmi.com llTechCo.,Ltd\t-\n" +
		fake.DevRoot + "/hidraw10\t0483:0065\tother\tS1\n"
	run(test, fake, "devices").expect(test, 0, want, "")
}

func TestValuesExample(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	run(test, fake, "values", "1=0x1234", "255=1").expect(test, 0, "values sent: 2, reports written: 1\n", "")
	want := append([]byte{0x00, 0x02, 0x02, 0x01, 0x12, 0x34, 0xFF, 0x00, 0x01}, make([]byte, 56)...)
	if got := screen.Received(); !bytes.Equal(got, want) {
		test.Errorf("arrived % X", got)
	}
}

func TestValuesWith21PairsWritesTwoReports(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	args := []string{"values"}
	for tag := 1; tag <= 21; tag++ {
		args = append(args, fmt.Sprintf("%d=%d", tag, tag))
	}
	run(test, fake, args...).expect(test, 0, "values sent: 21, reports written: 2\n", "")
	got := screen.Received()
	if len(got) != 130 || got[0] != 0x00 || got[1] != 0x02 || got[2] != 20 || got[65] != 0x00 || got[66] != 0x02 || got[67] != 1 {
		test.Errorf("arrived %d bytes, % X", len(got), got[:min(len(got), 4)])
	}
}

func TestWrongValuesArgumentWithNoScreen(test *testing.T) {
	run(test, fakescreen.New(test), "values", "1").expect(test, 1, "", "sensordeck: invalid argument \"1\": want <tag>=<value>\n")
}

func TestClock(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	before := time.Now().Truncate(time.Second)
	outcome := run(test, fake, "clock")
	after := time.Now()
	if outcome.status != 0 || !strings.HasPrefix(outcome.stdout, "clock sent: ") || outcome.stderr != "" {
		test.Fatalf("status %d, stdout %q, stderr %q", outcome.status, outcome.stdout, outcome.stderr)
	}
	got := screen.Received()
	if len(got) != 65 {
		test.Fatalf("arrived %d bytes", len(got))
	}
	expectClockReport(test, got[1:], before, after)
}

// expectClockReport checks the 64 bytes of a clock report: the fixed bytes, a time between before and after, and the
// weekday that belongs to it.
func expectClockReport(test *testing.T, report []byte, before, after time.Time) {
	test.Helper()
	if !bytes.Equal(report[0:3], []byte{0x03, 0x01, 0x15}) || report[10] != 0x64 || !bytes.Equal(report[11:], make([]byte, 53)) {
		test.Errorf("fixed bytes % X", report)
	}
	moment := time.Date(2000+int(report[3]), time.Month(report[4]), int(report[5]), int(report[6]), int(report[7]), int(report[8]), 0, time.Local)
	if moment.Before(before) || moment.After(after) {
		test.Errorf("time %v is not between %v and %v", moment, before, after)
	}
	weekday := int(moment.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	if int(report[9]) != weekday {
		test.Errorf("weekday %d, want %d", report[9], weekday)
	}
}

// theme is the file of the upload tests: byte i is i modulo 256.
func writeTheme(test *testing.T, size int) string {
	test.Helper()
	theme := make([]byte, size)
	for position := range theme {
		theme[position] = byte(position % 256)
	}
	path := filepath.Join(test.TempDir(), "theme.dat")
	if err := os.WriteFile(path, theme, 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

// recordingTransport answers ACK to every read and reopens at once, to learn the reports that Upload writes.
type recordingTransport struct{ written []smartmonitor.Report }

func (recording *recordingTransport) Write(report smartmonitor.Report) error {
	recording.written = append(recording.written, report)
	return nil
}
func (recording *recordingTransport) Read(time.Duration) (smartmonitor.Report, error) {
	return reportOf(0x06), nil
}
func (recording *recordingTransport) Reopen() error { return nil }

// expectedUploadReports are the reports of the upload as the package of the protocol writes them. Its own tests prove
// that they are the reports of the protocol, so this test checks that the executable carries them to the node.
func expectedUploadReports(test *testing.T, themePath string) []smartmonitor.Report {
	test.Helper()
	theme, err := os.ReadFile(themePath)
	if err != nil {
		test.Fatal(err)
	}
	recording := &recordingTransport{}
	noPause := func(time.Duration) {}
	if err := smartmonitor.Upload(recording, theme, noPause, func(smartmonitor.Stage, int, int) {}); err != nil {
		test.Fatal(err)
	}
	return recording.written
}

var (
	traceWritePattern  = regexp.MustCompile(`^ *\d+\.\d{6} > ([0-9A-F]{2}( [0-9A-F]{2}){63})$`)
	traceReadPattern   = regexp.MustCompile(`^ *\d+\.\d{6} < ([0-9A-F]{2}( [0-9A-F]{2}){63})$`)
	traceReopenPattern = regexp.MustCompile(`^ *\d+\.\d{6} \* reopen$`)
)

func hexadecimal(report smartmonitor.Report) string { return fmt.Sprintf("% X", report[:]) }

func TestUploadComplete(test *testing.T) {
	themePath := writeTheme(test, 2500)
	expected := expectedUploadReports(test, themePath)
	if len(expected) != 60 {
		test.Fatalf("the expectation has %d reports", len(expected))
	}
	if expected[0] != smartmonitor.ResetCommand() || expected[1] != smartmonitor.YmodemCommand() ||
		!strings.HasPrefix(hexadecimal(expected[2]), "01 00 FF 69 6D 67 2E 64 61 74") {
		test.Fatal("the first reports are not reset, ymodem command and the header")
	}

	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	screen.MoveAfter(1, 7)
	for _, count := range []int{2, 5, 22, 39, 56, 57} {
		screen.AnswerAfter(count, reportOf(0x06))
	}
	outcome := run(test, fake, "--trace", "upload", themePath)

	wantStdout := `reset
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
	if outcome.status != 0 || outcome.stdout != wantStdout {
		test.Errorf("status %d, stdout\n%s\nstderr\n%s", outcome.status, outcome.stdout, outcome.stderr)
	}
	var want []byte
	for _, report := range expected {
		want = append(append(want, 0x00), report[:]...)
	}
	if got := screen.Received(); !bytes.Equal(got, want) {
		test.Errorf("%d bytes arrived, want %d", len(got), len(want))
	}

	writes, reads, reopens := 0, 0, 0
	for _, line := range strings.Split(strings.TrimSuffix(outcome.stderr, "\n"), "\n") {
		switch {
		case traceWritePattern.MatchString(line):
			if got, want := traceWritePattern.FindStringSubmatch(line)[1], hexadecimal(expected[writes]); got != want {
				test.Errorf("write line %d carries %s", writes+1, got)
			}
			writes++
		case traceReadPattern.MatchString(line):
			reads++
		case traceReopenPattern.MatchString(line):
			reopens++
		default:
			test.Errorf("a trace line matches nothing: %q", line)
		}
	}
	if writes != 60 || reads != 6 || reopens != 2 {
		test.Errorf("trace has %d writes, %d reads, %d reopens", writes, reads, reopens)
	}
}

func TestUploadFailsOnAnUnexpectedAnswer(test *testing.T) {
	themePath := writeTheme(test, 2500)
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	screen.MoveAfter(1, 7)
	screen.AnswerAfter(2, reportOf(0x43))
	outcome := run(test, fake, "--trace", "upload", themePath)

	if outcome.status != 1 || !strings.HasSuffix(outcome.stdout, "ymodem command\n") {
		test.Errorf("status %d, stdout %q", outcome.status, outcome.stdout)
	}
	lines := strings.Split(strings.TrimSuffix(outcome.stderr, "\n"), "\n")
	if last := lines[len(lines)-1]; last != "sensordeck: upload failed at the ymodem command: unexpected answer 0x43" {
		test.Errorf("last line of stderr %q", last)
	}
	foundAnswer := false
	for _, line := range lines {
		if traceReadPattern.MatchString(line) && strings.HasPrefix(traceReadPattern.FindStringSubmatch(line)[1], "43") {
			foundAnswer = true
		}
	}
	if !foundAnswer {
		test.Error("the trace holds no read line that starts with 43")
	}
	if got := len(screen.Received()); got != 130 {
		test.Errorf("%d bytes arrived, want the reset and the ymodem command only", got)
	}
}

func TestListen(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	screen.Send(reportOf(0x11, 0x22))
	screen.Send(reportOf(0x33))
	outcome := run(test, fake, "listen", "--seconds", "1")
	lines := strings.Split(strings.TrimSuffix(outcome.stdout, "\n"), "\n")
	if outcome.status != 0 || outcome.stderr != "" || len(lines) != 3 {
		test.Fatalf("status %d, stdout %q, stderr %q", outcome.status, outcome.stdout, outcome.stderr)
	}
	for position, report := range []smartmonitor.Report{reportOf(0x11, 0x22), reportOf(0x33)} {
		if match := traceReadPattern.FindStringSubmatch(lines[position]); match == nil || match[1] != hexadecimal(report) {
			test.Errorf("line %d is %q", position+1, lines[position])
		}
	}
	if lines[2] != "reports received: 2" {
		test.Errorf("last line %q", lines[2])
	}
	if len(screen.Received()) != 0 {
		test.Error("listen wrote to the device")
	}
}

func TestRaw(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	answer := reportOf(0x43, 0x01)
	screen.AnswerAfter(1, answer)
	outcome := run(test, fake, "raw", "03", "01", "15")
	lines := strings.Split(strings.TrimSuffix(outcome.stdout, "\n"), "\n")
	if outcome.status != 0 || outcome.stderr != "" || len(lines) != 2 {
		test.Fatalf("status %d, stdout %q, stderr %q", outcome.status, outcome.stdout, outcome.stderr)
	}
	if match := traceReadPattern.FindStringSubmatch(lines[0]); match == nil || match[1] != hexadecimal(answer) {
		test.Errorf("read line %q", lines[0])
	}
	if lines[1] != "reports received: 1" {
		test.Errorf("last line %q", lines[1])
	}
	want := append([]byte{0x00, 0x03, 0x01, 0x15}, make([]byte, 61)...)
	if got := screen.Received(); !bytes.Equal(got, want) {
		test.Errorf("arrived % X", got)
	}
}

func TestTwoScreens(test *testing.T) {
	fake := fakescreen.New(test)
	first := fake.Add(2, "first", "")
	second := fake.Add(4, "second", "S2")

	wantError := "sensordeck: more than one supported screen is connected, name one with --device:\n" +
		fake.DevRoot + "/hidraw2\t0483:0065\tfirst\t-\n" +
		fake.DevRoot + "/hidraw4\t0483:0065\tsecond\tS2\n"
	run(test, fake, "values", "1=1").expect(test, 1, "", wantError)

	run(test, fake, "--device", second.Node(), "values", "1=1").expect(test, 0, "values sent: 1, reports written: 1\n", "")
	if got := second.Received(); len(got) != 65 || got[1] != 0x02 {
		test.Errorf("the named screen got %d bytes", len(got))
	}
	if got := first.Received(); len(got) != 0 {
		test.Errorf("the other screen got %d bytes", len(got))
	}
}
