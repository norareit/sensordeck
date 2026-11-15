//go:build linux

package nvidiasmi

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakesensors"
	"github.com/norareit/sensordeck/internal/sensors"
)

const fullLine = "44, 17, 3256, 32607"

func everything(string) bool { return true }

func read(source sensors.Source, wanted func(string) bool) []sensors.Reading {
	return source.Read(context.Background(), wanted)
}

func readGPU(test *testing.T, script string, wanted func(string) bool) []sensors.Reading {
	test.Helper()
	test.Setenv("PATH", fakesensors.NvidiaSMI(test, script))
	return read(Source{TimeLimit: 5 * time.Second}, wanted)
}

func allFour() []sensors.Reading {
	return []sensors.Reading{
		{Name: "gpu.temp", Value: 44}, {Name: "gpu.util", Value: 17},
		{Name: "gpu.mem.used", Value: 3256}, {Name: "gpu.mem.total", Value: 32607},
	}
}

func TestCompleteLine(test *testing.T) {
	if got := readGPU(test, "echo '"+fullLine+"'\n", everything); !reflect.DeepEqual(got, allFour()) {
		test.Errorf("got %v", got)
	}
}

func TestTheArgumentsAreExactlyTheTwoOfTheSpec(test *testing.T) {
	script := `[ "$#" = 2 ] && [ "$1" = "--query-gpu=temperature.gpu,utilization.gpu,memory.used,memory.total" ] && ` +
		`[ "$2" = "--format=csv,noheader,nounits" ] || exit 1
echo '` + fullLine + "'\n"
	if got := readGPU(test, script, everything); !reflect.DeepEqual(got, allFour()) {
		test.Errorf("got %v", got)
	}
}

func TestLineEndingInCRLFGivesTheFourMetrics(test *testing.T) {
	if got := readGPU(test, "printf '"+fullLine+"\\r\\n'\n", everything); !reflect.DeepEqual(got, allFour()) {
		test.Errorf("got %v", got)
	}
}

func TestOnlyTheFirstLineCounts(test *testing.T) {
	if got := readGPU(test, "echo '"+fullLine+"'\necho '1, 2, 3, 4'\n", everything); !reflect.DeepEqual(got, allFour()) {
		test.Errorf("got %v", got)
	}
}

func TestFieldsThatAreNotNumbersAreLeftOut(test *testing.T) {
	got := readGPU(test, "echo '[N/A], 17, [Not Supported], 32607'\n", everything)
	if want := []string{"gpu.util", "gpu.mem.total"}; !reflect.DeepEqual(names(got), want) {
		test.Errorf("got %v", got)
	}
	got = readGPU(test, "echo '44, 17, 4a, 32607.5'\n", everything)
	if want := []string{"gpu.temp", "gpu.util", "gpu.mem.total"}; !reflect.DeepEqual(names(got), want) {
		test.Errorf("got %v", got)
	}
}

func TestThreeFieldsGiveNothing(test *testing.T) {
	if got := readGPU(test, "echo '44, 17, 3256'\n", everything); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestExitStatusOneGivesNothing(test *testing.T) {
	if got := readGPU(test, "echo '"+fullLine+"'\nexit 1\n", everything); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestMissingProgramGivesNothing(test *testing.T) {
	test.Setenv("PATH", test.TempDir())
	if got := read(Source{TimeLimit: time.Second}, everything); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestNoGPUMetricWantedDoesNotRunTheProgram(test *testing.T) {
	marker := filepath.Join(test.TempDir(), "ran")
	// a redirection is a shell builtin, so the marker needs no program of the PATH
	readGPU(test, ": > "+marker+"\n", func(name string) bool { return name == "cpu.temp" })
	if _, err := os.Stat(marker); err == nil {
		test.Error("the program ran")
	}
}

func TestTheMarkerShowsARunOfTheProgram(test *testing.T) {
	marker := filepath.Join(test.TempDir(), "ran")
	readGPU(test, ": > "+marker+"\n", everything)
	if _, err := os.Stat(marker); err != nil {
		test.Errorf("the marker was not written: %v", err)
	}
}

func TestTwoGPUMetricsRunTheProgramOnce(test *testing.T) {
	marker := filepath.Join(test.TempDir(), "runs")
	script := "echo run >> " + marker + "\necho '" + fullLine + "'\n"
	got := readGPU(test, script, func(name string) bool { return strings.HasPrefix(name, "gpu.mem") })
	if want := []string{"gpu.mem.used", "gpu.mem.total"}; !reflect.DeepEqual(names(got), want) {
		test.Errorf("got %v", got)
	}
	content, _ := os.ReadFile(marker)
	if string(content) != "run\n" {
		test.Errorf("runs: %q", content)
	}
}

func TestProgramBeyondTheTimeLimitIsKilledEvenWithAChildHoldingItsOutput(test *testing.T) {
	directory := test.TempDir()
	started, finished := filepath.Join(directory, "started"), filepath.Join(directory, "finished")
	// the child keeps the output open for 3 seconds and the program waits for it
	script := ": > " + started + "\nsleep 3 &\nwait\n: > " + finished + "\n"
	test.Setenv("PATH", fakesensors.NvidiaSMI(test, script))
	limit := 100 * time.Millisecond
	begun := time.Now()
	got := read(Source{TimeLimit: limit}, everything)
	elapsed := time.Since(begun)
	if _, err := os.Stat(started); err != nil {
		test.Fatalf("the program did not start: %v", err)
	}
	if len(got) != 0 || elapsed < limit || elapsed > limit+time.Second {
		test.Errorf("got %v after %v", got, elapsed)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(finished); err == nil {
		test.Error("the program ran to its end")
	}
}

func TestStopEndsTheRun(test *testing.T) {
	directory := test.TempDir()
	started := filepath.Join(directory, "started")
	test.Setenv("PATH", fakesensors.NvidiaSMI(test, ": > "+started+"\nexec sleep 3\n"))
	stop, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	begun := time.Now()
	got := Source{TimeLimit: 5 * time.Second}.Read(stop, everything)
	elapsed := time.Since(begun)
	if _, err := os.Stat(started); err != nil {
		test.Fatalf("the program did not start: %v", err)
	}
	if len(got) != 0 || elapsed < 300*time.Millisecond || elapsed > time.Second {
		test.Errorf("got %v after %v", got, elapsed)
	}
}

func TestFakeProgramLookupFindsOnlyTheFake(test *testing.T) {
	test.Setenv("PATH", fakesensors.NvidiaSMI(test, "exit 0\n"))
	if _, err := exec.LookPath("sleep"); err == nil {
		test.Error("sleep is on the PATH of the program under test")
	}
}

func TestParseGPULine(test *testing.T) {
	if fields, ok := parseGPULine(fullLine + "\n"); !ok || fields != [4]string{"44", "17", "3256", "32607"} {
		test.Errorf("%v %v", fields, ok)
	}
	if fields, ok := parseGPULine(fullLine + "\r\n"); !ok || fields != [4]string{"44", "17", "3256", "32607"} {
		test.Errorf("CR LF: %v %v", fields, ok)
	}
	if fields, ok := parseGPULine(fullLine + "\r\n1, 2, 3, 4\r\n"); !ok || fields[3] != "32607" {
		test.Errorf("two lines with CR LF: %v %v", fields, ok)
	}
	if fields, _ := parseGPULine("44, 17, 3256, 32607\r"); fields[3] != "32607\r" {
		test.Errorf("a CR with no LF after it was dropped: %q", fields[3])
	}
	if fields, ok := parseGPULine(fullLine + "\n1, 2, 3, 4\n"); !ok || fields[0] != "44" {
		test.Errorf("two lines: %v %v", fields, ok)
	}
	if _, ok := parseGPULine("a,b,c,d,e"); ok {
		test.Error("five fields were accepted")
	}
}

func names(readings []sensors.Reading) []string {
	var result []string
	for _, reading := range readings {
		result = append(result, reading.Name)
	}
	return result
}
