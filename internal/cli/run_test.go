package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const twoTags = "tags:\n  cpu.temp: 1\n  cpu.clock: 2\n"

func writeConfig(test *testing.T, content string) string {
	test.Helper()
	path := filepath.Join(test.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

// runWith gives a harness with one fake source and a stop after the given number of waits.
func runWith(source *fakeSource, stopAfterWaits int) *harness {
	harness := newHarness()
	harness.sources = []sensors.Source{source}
	harness.onWait = func(count int) {
		if count >= stopAfterWaits {
			harness.cancelStop()
		}
	}
	return harness
}

func twoMetrics() *fakeSource {
	return &fakeSource{names: []string{"cpu.temp", "cpu.clock"}, values: map[string]float64{"cpu.temp": 42, "cpu.clock": 1234.4}}
}

// kinds writes C for a clock report and V for a value report, for each report written.
func kinds(test *testing.T, writes []smartmonitor.Report) string {
	test.Helper()
	var result strings.Builder
	for _, report := range writes {
		switch report[0] {
		case 0x03:
			result.WriteString("C")
		case 0x02:
			result.WriteString("V")
		default:
			test.Fatalf("report % X", report[:4])
		}
	}
	return result.String()
}

func valueReport(pairs ...smartmonitor.Value) smartmonitor.Report {
	return smartmonitor.EncodeValues(pairs)[0]
}

func TestRunConfigErrorsOpenNoTransport(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "none.yaml")
	cases := []struct {
		name  string
		setup func(*harness)
		args  []string
		want  string
	}{
		{"no config directory", func(harness *harness) { harness.configDirErr = os.ErrNotExist }, []string{"run"},
			"cannot find the config directory, name the file with --config: file does not exist"},
		{"missing file", nil, []string{"run", "--config", missing},
			"cannot read the config file " + missing + ": no such file or directory"},
		{"tag out of range", nil, []string{"run", "--config", writeConfig(test, "tags: {cpu.temp: 256}\n")},
			"config file %s: tags: cpu.temp: the tag 256 is outside 0 to 255"},
		{"argument", nil, []string{"run", "x"}, `unknown command "x" for "sensordeck run"`},
	}
	for _, each := range cases {
		harness := newHarness()
		if each.setup != nil {
			each.setup(harness)
		}
		want := each.want
		if strings.Contains(want, "%s") {
			want = fmt.Sprintf(want, each.args[2])
		}
		harness.expectFailure(test, want, each.args...)
		if len(harness.opened) != 0 || len(harness.transport.writes) != 0 {
			test.Errorf("%s: opened %v, wrote %d", each.name, harness.opened, len(harness.transport.writes))
		}
	}
}

func TestRunReadsTheDefaultConfigFile(test *testing.T) {
	directory := test.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "sensordeck"), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "sensordeck", "config.yaml"), []byte(twoTags), 0o644); err != nil {
		test.Fatal(err)
	}
	harness := runWith(twoMetrics(), 1)
	harness.configDir = directory
	if status := harness.run("run"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if !strings.HasPrefix(harness.stdout.String(), "running: 2 metrics every 1s\n") {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestRunStartsWithTheClockAndTheFirstRound(test *testing.T) {
	harness := runWith(twoMetrics(), 1)
	if status := harness.run("run", "--config", writeConfig(test, twoTags)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got, want := harness.stdout.String(), "running: 2 metrics every 1s\nstopped\n"; got != want {
		test.Errorf("stdout %q", got)
	}
	clock, _ := smartmonitor.EncodeClock(harness.clock.current.Add(-time.Second))
	writes := harness.transport.writes
	if len(writes) != 2 || writes[0] != clock {
		test.Fatalf("writes %d, first % X", len(writes), writes[0][:12])
	}
	if want := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 2, Value: 1234}); writes[1] != want {
		test.Errorf("values % X", writes[1][:10])
	}
	if harness.transport.closed != 1 || harness.stderr.Len() != 0 {
		test.Errorf("closed %d, stderr %q", harness.transport.closed, harness.stderr.String())
	}
}

func TestRunAsksTheSourcesOnlyForTheMappedNames(test *testing.T) {
	source := &fakeSource{names: []string{"cpu.temp", "cpu.clock", "ram.used"}, values: map[string]float64{"cpu.temp": 1, "ram.used": 2}}
	runWith(source, 1).run("run", "--config", writeConfig(test, "tags: {cpu.temp: 1, ram.used: 2}\n"))
	if want := [][]string{{"cpu.temp", "ram.used"}}; !reflect.DeepEqual(source.asked, want) {
		test.Errorf("asked %v", source.asked)
	}
}

func TestRunSetsTheClockAgainAfterSixtySecondsAndNotBefore(test *testing.T) {
	harness := runWith(twoMetrics(), 61) // rounds 0 to 60 of one second each
	harness.run("run", "--config", writeConfig(test, twoTags))
	want := "C" + strings.Repeat("V", 60) + "CV"
	if got := kinds(test, harness.transport.writes); got != want {
		test.Errorf("writes %s", got)
	}
}

// The clock written at 2.5 seconds, after the reopening, is 60 seconds old at 62.5, so the round that starts at 62.6
// after a long round sets it, although the place of that round on the grid of one second is 62.
func TestRunSetsTheClockByTheTimeARoundStartsNotByItsPlaceOnTheGrid(test *testing.T) {
	source := twoMetrics()
	var harness *harness
	reads := 0
	var writtenBeforeRead62 string
	source.onRead = func() {
		reads++
		switch reads {
		case 1:
			harness.clock.sleep(2500 * time.Millisecond)
		case 61:
			harness.clock.sleep(1600 * time.Millisecond)
		case 62:
			writtenBeforeRead62 = kinds(test, harness.transport.writes)
		case 63:
			harness.cancelStop()
		}
	}
	harness = newHarness()
	harness.sources = []sensors.Source{source}
	harness.transport.failWrites = []bool{false, true}
	harness.transport.reopenResults = []error{nil}
	harness.run("run", "--config", writeConfig(test, twoTags))
	if reads < 62 || !strings.HasSuffix(writtenBeforeRead62, "VC") {
		test.Errorf("reads %d, writes before the 62nd read end in %q", reads, writtenBeforeRead62[max(0, len(writtenBeforeRead62)-4):])
	}
}

func TestRunLeavesAnUnavailableMetricOut(test *testing.T) {
	source := twoMetrics()
	delete(source.values, "cpu.temp")
	harness := runWith(source, 1)
	harness.run("run", "--config", writeConfig(test, twoTags))
	if want := valueReport(smartmonitor.Value{Tag: 2, Value: 1234}); harness.transport.writes[1] != want {
		test.Errorf("values % X", harness.transport.writes[1][:10])
	}
}

func TestRunWritesNoReportWhenEveryMetricIsUnavailable(test *testing.T) {
	harness := runWith(&fakeSource{names: []string{"cpu.temp", "cpu.clock"}}, 2)
	harness.run("run", "--config", writeConfig(test, twoTags))
	if got := kinds(test, harness.transport.writes); got != "C" {
		test.Errorf("writes %s", got)
	}
}

func TestRun21MetricsGiveTwoReportsOrderedByTag(test *testing.T) {
	source := &fakeSource{values: map[string]float64{}}
	var config strings.Builder
	config.WriteString("tags:\n")
	for number := 1; number <= 21; number++ {
		name := fmt.Sprintf("nvme.%d.temp", number)
		source.names = append(source.names, name)
		source.values[name] = float64(number)
		fmt.Fprintf(&config, "  %s: %d\n", name, 22-number)
	}
	harness := runWith(source, 1)
	harness.run("run", "--config", writeConfig(test, config.String()))
	writes := harness.transport.writes
	if got := kinds(test, writes); got != "CVV" {
		test.Fatalf("writes %s", got)
	}
	if writes[1][1] != 20 || writes[1][2] != 1 || writes[1][2+3*19] != 20 || writes[2][1] != 1 || writes[2][2] != 21 {
		test.Errorf("first % X, second % X", writes[1][:8], writes[2][:8])
	}
	// the tag 1 is nvme.21.temp, with the value 21
	if writes[1][4] != 21 {
		test.Errorf("value of tag 1 is %d", writes[1][4])
	}
}

func TestRunDoesNotMakeUpMissedRounds(test *testing.T) {
	source := twoMetrics()
	var harness *harness
	var readTimes []time.Time
	source.onRead = func() {
		readTimes = append(readTimes, harness.clock.current)
		if len(readTimes) == 1 {
			harness.clock.sleep(2500 * time.Millisecond)
		}
	}
	harness = runWith(source, 1)
	harness.run("run", "--config", writeConfig(test, twoTags))
	start := time.Date(2026, 11, 14, 15, 4, 5, 0, time.UTC)
	want := []time.Time{start, start.Add(2500 * time.Millisecond)}
	if !reflect.DeepEqual(readTimes, want) {
		test.Errorf("rounds at %v", readTimes)
	}
	// the round that ran into the next was followed at once, and the next wait ends on the grid at 3 seconds
	if !reflect.DeepEqual(harness.waits, []time.Duration{500 * time.Millisecond}) {
		test.Errorf("waits %v", harness.waits)
	}
}

func TestRunReopensALostScreenOncePerRound(test *testing.T) {
	harness := runWith(twoMetrics(), 4)
	harness.transport.failWrites = []bool{false, true}
	harness.transport.reopenResults = []error{errFake, errFake, nil}
	harness.run("run", "--config", writeConfig(test, twoTags))
	if got := harness.stderr.String(); got != "screen lost: fake failure\nscreen back\n" {
		test.Errorf("stderr %q", got)
	}
	if got := kinds(test, harness.transport.writes); got != "CCV" {
		test.Errorf("writes %s", got)
	}
	if harness.transport.reopens != 3 {
		test.Errorf("reopens %d", harness.transport.reopens)
	}
	if !strings.HasSuffix(harness.stdout.String(), "stopped\n") {
		test.Errorf("stdout %q", harness.stdout.String())
	}
}

func TestRunStopsWhileTheScreenIsLost(test *testing.T) {
	harness := runWith(twoMetrics(), 3)
	harness.transport.failWrites = []bool{false, true}
	harness.transport.reopenErr = errFake
	if status := harness.run("run", "--config", writeConfig(test, twoTags)); status != 0 {
		test.Fatalf("status %d", status)
	}
	if got := kinds(test, harness.transport.writes); got != "C" {
		test.Errorf("writes %s", got)
	}
	if harness.transport.closed != 1 || harness.stdout.String() != "running: 2 metrics every 1s\nstopped\n" {
		test.Errorf("closed %d, stdout %q", harness.transport.closed, harness.stdout.String())
	}
}

func TestRunWritesNothingAfterTheStopWhileAReadIsInProgress(test *testing.T) {
	source := twoMetrics()
	var harness *harness
	source.onRead = func() { harness.cancelStop() }
	harness = runWith(source, 100)
	if status := harness.run("run", "--config", writeConfig(test, twoTags)); status != 0 {
		test.Fatalf("status %d", status)
	}
	if got := kinds(test, harness.transport.writes); got != "C" || harness.transport.closed != 1 {
		test.Errorf("writes %s, closed %d", got, harness.transport.closed)
	}
}

func TestRunWithTraceWritesALineForEveryWriteAndReopening(test *testing.T) {
	harness := runWith(twoMetrics(), 3)
	harness.transport.failWrites = []bool{false, true}
	harness.transport.reopenResults = []error{errFake, nil}
	harness.run("--trace", "run", "--config", writeConfig(test, twoTags))
	output := harness.stderr.String()
	for _, line := range []string{"> 03 01 15", "> 02 02 01", "! fake failure\n", "screen lost: fake failure\n",
		"* reopen ! fake failure\n", "* reopen\n", "screen back\n"} {
		if !strings.Contains(output, line) {
			test.Errorf("the output has no %q: %q", line, output)
		}
	}
	if count := strings.Count(output, "> 03 01 15"); count != 2 {
		test.Errorf("%d clock lines", count)
	}
	if count := strings.Count(output, "* reopen"); count != 2 {
		test.Errorf("%d reopening lines", count)
	}
}

const controllerTags = "tags:\n  cpu.temp: 1\n  commander.1.temp: 4\n  commander.2.temp: 5\n"

func twoProbes() *fakeSource {
	return &fakeSource{
		names:  []string{"commander.1.temp", "commander.2.temp"},
		values: map[string]float64{"commander.1.temp": 37, "commander.2.temp": 23},
	}
}

// runWithController gives a harness with one other source, a controller and a stop after the given number of waits.
func runWithController(others, controller *fakeSource, stopAfterWaits int) *harness {
	harness := runWith(others, stopAfterWaits)
	harness.controller = controller
	return harness
}

// valueReports gives the value report of each round, in order.
func valueReports(harness *harness) []smartmonitor.Report {
	var reports []smartmonitor.Report
	for _, report := range harness.transport.writes {
		if report[0] == 0x02 {
			reports = append(reports, report)
		}
	}
	return reports
}

func startOfTheRun() time.Time { return time.Date(2026, 11, 14, 15, 4, 5, 0, time.UTC) }

func TestRunReadsTheControllerOncePerPeriod(test *testing.T) {
	controller := twoProbes()
	harness := runWithController(twoMetrics(), controller, 121) // rounds at 0 to 120 seconds
	var calls []time.Time
	controller.onRead = func() { calls = append(calls, harness.clock.current) }
	harness.run("run", "--config", writeConfig(test, controllerTags))
	start := startOfTheRun()
	want := []time.Time{start, start.Add(60 * time.Second), start.Add(120 * time.Second)}
	if !reflect.DeepEqual(calls, want) {
		test.Errorf("the controller was read at %v", calls)
	}
}

func TestRunSendsTheKeptReadingsAndReadsTheOtherSourcesInEveryRound(test *testing.T) {
	cpu := twoMetrics()
	controller := twoProbes()
	harness := runWithController(cpu, controller, 3)
	calls := 0
	cpu.onRead = func() {
		calls++
		cpu.values["cpu.temp"] = float64(calls)
	}
	harness.run("run", "--config", writeConfig(test, controllerTags))
	reports := valueReports(harness)
	if len(reports) != 3 || controller.readsAt != 1 {
		test.Fatalf("%d value reports, %d reads of the controller", len(reports), controller.readsAt)
	}
	for round, report := range reports {
		want := valueReport(smartmonitor.Value{Tag: 1, Value: uint16(round + 1)}, smartmonitor.Value{Tag: 4, Value: 37}, smartmonitor.Value{Tag: 5, Value: 23})
		if report != want {
			test.Errorf("round %d: % X", round, report[:12])
		}
	}
}

func TestRunWaitsAFullPeriodAfterAReadingThatGivesNothing(test *testing.T) {
	controller := &fakeSource{names: []string{"commander.1.temp", "commander.2.temp"}}
	harness := runWithController(twoMetrics(), controller, 60) // rounds at 0 to 59 seconds
	harness.run("run", "--config", writeConfig(test, controllerTags))
	reports := valueReports(harness)
	if len(reports) != 60 || controller.readsAt != 1 {
		test.Fatalf("%d value reports, %d reads of the controller", len(reports), controller.readsAt)
	}
	for round, report := range reports {
		if report != valueReport(smartmonitor.Value{Tag: 1, Value: 42}) {
			test.Errorf("round %d: % X", round, report[:8])
		}
	}
}

func TestRunKeepsTheProbesOfAReadingThatGivesOnlySome(test *testing.T) {
	controller := twoProbes()
	delete(controller.values, "commander.2.temp")
	harness := runWithController(twoMetrics(), controller, 3) // rounds at 0, 1 and 2 seconds
	controller.onRead = func() {
		if controller.readsAt == 2 {
			controller.values["commander.2.temp"] = 23
		}
	}
	harness.run("run", "--config", writeConfig(test, "interval: 1s\ncommander_interval: 2s\n"+controllerTags))
	reports := valueReports(harness)
	only := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 4, Value: 37})
	both := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 4, Value: 37}, smartmonitor.Value{Tag: 5, Value: 23})
	if len(reports) != 3 || reports[0] != only || reports[1] != only || reports[2] != both {
		test.Errorf("reports %d", len(reports))
	}
}

func TestRunLeavesTheProbesOutAfterAReadingThatGivesNone(test *testing.T) {
	controller := twoProbes()
	harness := runWithController(twoMetrics(), controller, 4) // rounds at 0 to 3 seconds
	controller.onRead = func() {
		if controller.readsAt == 2 {
			clear(controller.values)
		}
	}
	harness.run("run", "--config", writeConfig(test, "interval: 1s\ncommander_interval: 2s\n"+controllerTags))
	reports := valueReports(harness)
	both := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 4, Value: 37}, smartmonitor.Value{Tag: 5, Value: 23})
	none := valueReport(smartmonitor.Value{Tag: 1, Value: 42})
	if len(reports) != 4 || reports[0] != both || reports[1] != both || reports[2] != none || reports[3] != none {
		test.Errorf("reports %d", len(reports))
	}
}

func TestRunReadsTheControllerInEveryRoundWhenThePeriodIsTheInterval(test *testing.T) {
	controller := twoProbes()
	harness := runWithController(twoMetrics(), controller, 3)
	harness.run("run", "--config", writeConfig(test, "interval: 1s\ncommander_interval: 1s\n"+controllerTags))
	if controller.readsAt != 3 {
		test.Errorf("the controller was read %d times", controller.readsAt)
	}
}

// The period is measured from the start of the round that read the controller, not from the end of the reading.
func TestRunMeasuresThePeriodFromTheStartOfTheRound(test *testing.T) {
	controller := twoProbes()
	harness := runWithController(twoMetrics(), controller, 1000)
	var calls []time.Time
	controller.onRead = func() {
		calls = append(calls, harness.clock.current)
		harness.clock.sleep(500 * time.Millisecond)
		if len(calls) == 3 {
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", writeConfig(test, controllerTags))
	start := startOfTheRun()
	want := []time.Time{start, start.Add(60 * time.Second), start.Add(120 * time.Second)}
	if !reflect.DeepEqual(calls, want) {
		test.Errorf("the controller was read at %v", calls)
	}
}

// The round at 59 seconds ends at 60.6, so the next round starts late and is the first that is a period after the
// start. The period that follows is measured from 60.6.
func TestRunMeasuresThePeriodFromTheStartOfALateRound(test *testing.T) {
	others := twoMetrics()
	controller := twoProbes()
	harness := runWithController(others, controller, 1000)
	otherCalls := 0
	others.onRead = func() {
		otherCalls++
		if otherCalls == 60 {
			harness.clock.sleep(1600 * time.Millisecond)
		}
	}
	var calls []time.Time
	controller.onRead = func() {
		calls = append(calls, harness.clock.current)
		if len(calls) == 3 {
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", writeConfig(test, controllerTags))
	start := startOfTheRun()
	want := []time.Time{start, start.Add(60600 * time.Millisecond), start.Add(121 * time.Second)}
	if !reflect.DeepEqual(calls, want) {
		test.Errorf("the controller was read at %v", calls)
	}
}

func TestRunSendsTheKeptReadingsAfterALostScreenWithoutReadingTheController(test *testing.T) {
	controller := twoProbes()
	harness := runWithController(twoMetrics(), controller, 3)
	harness.transport.failWrites = []bool{false, true}
	harness.transport.reopenResults = []error{errFake, nil}
	harness.run("run", "--config", writeConfig(test, controllerTags))
	if got := kinds(test, harness.transport.writes); got != "CCV" {
		test.Fatalf("writes %s", got)
	}
	want := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 4, Value: 37}, smartmonitor.Value{Tag: 5, Value: 23})
	if harness.transport.writes[2] != want || controller.readsAt != 1 {
		test.Errorf("last report % X, %d reads of the controller", harness.transport.writes[2][:12], controller.readsAt)
	}
}
