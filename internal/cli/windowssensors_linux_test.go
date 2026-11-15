//go:build linux

package cli

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakesensors"
	"github.com/norareit/sensordeck/internal/fakewinsensors"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/windowssensors"
)

// The tests of this file run the commands over the sources of Windows, with the calls of Windows faked, a fake
// nvidia-smi on a PATH they set and a real HTTP server for LibreHardwareMonitor.

const gibibyte = 1 << 30

// monitorDocument is a tree in the shape of the web server, with an AMD CPU, a drive and a memory module.
const monitorDocument = `{"id":0,"Text":"Sensor","Min":"Min","Value":"Value","Max":"Max","ImageURL":"","Children":[
 {"id":1,"Text":"DESKTOP","Min":"","Value":"","Max":"","ImageURL":"","Children":[
  {"id":2,"Text":"CPU","Min":"","Value":"","Max":"","ImageURL":"","Children":[
   {"id":3,"Text":"Temperatures","Min":"","Value":"","Max":"","ImageURL":"","Children":[
    {"id":4,"Text":"Core (Tctl/Tdie)","Min":"40.0 °C","Value":"46.9 °C","Max":"60.0 °C","ImageURL":"","SensorId":"/amdcpu/0/temperature/2","Type":"Temperature","Children":[]},
    {"id":5,"Text":"CCD1 (Tdie)","Min":"40.0 °C","Value":"41.5 °C","Max":"60.0 °C","ImageURL":"","SensorId":"/amdcpu/0/temperature/3","Type":"Temperature","Children":[]}]},
   {"id":6,"Text":"Clocks","Min":"","Value":"","Max":"","ImageURL":"","Children":[
    {"id":7,"Text":"Core #1","Min":"","Value":"4425.4 MHz","Max":"","ImageURL":"","SensorId":"/amdcpu/0/clock/1","Type":"Clock","Children":[]}]}]},
  {"id":8,"Text":"Samsung SSD","Min":"","Value":"","Max":"","ImageURL":"","Children":[
   {"id":9,"Text":"Temperatures","Min":"","Value":"","Max":"","ImageURL":"","Children":[
    {"id":10,"Text":"Temperature","Min":"","Value":"35.0 °C","Max":"","ImageURL":"","SensorId":"/nvme/0/temperature/0","Type":"Temperature","Children":[]}]}]},
  {"id":11,"Text":"DIMM","Min":"","Value":"","Max":"","ImageURL":"","Children":[
   {"id":12,"Text":"Temperatures","Min":"","Value":"","Max":"","ImageURL":"","Children":[
    {"id":13,"Text":"DIMM #0","Min":"","Value":"36.0 °C","Max":"","ImageURL":"","SensorId":"/memory/dimm/0/temperature/0","Type":"Temperature","Children":[]}]}]}]}]}`

const noMonitorMetricDocument = `{"Text":"Sensor","Children":[{"Text":"DESKTOP","Children":[]}]}`

// monitor is a server for the document that can be made to fail, and counts its requests.
type monitor struct {
	server   *httptest.Server
	document atomic.Value
	failing  atomic.Bool
	requests atomic.Int32
	holding  chan struct{} // when set, a request is held until the client goes and this is told of it
}

func newMonitor(test *testing.T, document string) *monitor {
	test.Helper()
	served := &monitor{}
	served.document.Store(document)
	served.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		served.requests.Add(1)
		if served.holding != nil {
			served.holding <- struct{}{}
			<-request.Context().Done()
			return
		}
		if served.failing.Load() {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = writer.Write([]byte(served.document.Load().(string)))
	}))
	test.Cleanup(func() {
		served.server.CloseClientConnections()
		served.server.Close()
	})
	return served
}

func (served *monitor) url() string { return served.server.URL + "/data.json" }

// windowsHarness is a harness whose sources are those of Windows over a fake seam, with a controller that gives one
// probe, and a fake nvidia-smi.
func windowsHarness(test *testing.T) (*harness, *fakewinsensors.Fake) {
	test.Helper()
	test.Setenv("PATH", fakesensors.NvidiaSMI(test, "echo '44, 17, 3256, 32607'\n"))
	seam := &fakewinsensors.Fake{
		Total: 32 * gibibyte, Available: 12 * gibibyte,
		TimesList: []fakewinsensors.Times{{Idle: 100, Kernel: 400, User: 100}, {Idle: 150, Kernel: 500, User: 200}},
		// The counters do not change, so that the rates are 0.0 whatever the time between the samples.
		Tables: [][]windowssensors.Interface{{{LUID: 1, Flags: 0x05, InOctets: 5000, OutOctets: 7000}}},
	}
	harness := newHarness()
	harness.controller = &fakeSource{names: []string{"commander.1.temp"}, values: map[string]float64{"commander.1.temp": 37.3}}
	set := withWindowsSensors(sourceSet{controller: harness.controller, controllerProblem: func() error { return harness.controllerProblem }}, seam)
	harness.set = &set
	harness.configDir = test.TempDir()
	return harness, seam
}

// writeDefaultConfig writes the config file at the place that run reads by default.
func writeDefaultConfig(test *testing.T, harness *harness, content string) string {
	test.Helper()
	directory := filepath.Join(harness.configDir, "sensordeck")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	path := filepath.Join(directory, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

func writeMonitorURL(test *testing.T, harness *harness, served *monitor) {
	test.Helper()
	writeDefaultConfig(test, harness, "librehardwaremonitor_url: "+served.url()+"\ntags: {cpu.temp: 1}\n")
}

const allMetricsOnWindows = "cpu.temp\t46.9\t°C\n" +
	"cpu.ccd1.temp\t41.5\t°C\n" +
	"cpu.util\t75.0\t%\n" +
	"cpu.clock\t4425.4\tMHz\n" +
	"gpu.temp\t44.0\t°C\n" +
	"gpu.util\t17.0\t%\n" +
	"gpu.mem.used\t3256.0\tMiB\n" +
	"gpu.mem.total\t32607.0\tMiB\n" +
	"ram.used\t20.0\tGiB\n" +
	"ram.total\t32.0\tGiB\n" +
	"ram.util\t62.5\t%\n" +
	"ram.1.temp\t36.0\t°C\n" +
	"nvme.1.temp\t35.0\t°C\n" +
	"net.up\t0.0\tKiB/s\n" +
	"net.down\t0.0\tKiB/s\n" +
	"commander.1.temp\t37.3\t°C\n"

func TestSensorsOnWindowsListsEveryMetricInTheOrderOfTheTable(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, monitorDocument)
	writeMonitorURL(test, harness, served)
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	got, want := strings.Split(harness.stdout.String(), "\n"), strings.Split(allMetricsOnWindows, "\n")
	if len(got) != len(want) {
		test.Fatalf("stdout %q", harness.stdout.String())
	}
	for position := range want {
		if got[position] != want[position] {
			test.Errorf("line %d: %q, want %q", position, got[position], want[position])
		}
	}
	if harness.stderr.Len() != 0 || served.requests.Load() != 1 {
		test.Errorf("stderr %q, %d requests", harness.stderr.String(), served.requests.Load())
	}
}

func refusedLine(address string, host string) string {
	return "LibreHardwareMonitor cannot be read at " + address + ": dial tcp " + host +
		": connect: connection refused. The temperatures and cpu.clock need it running, with Options > Remote Web Server > Run checked\n"
}

func TestSensorsOnWindowsWithTheServerStoppedListsTheOthersAndWritesTheLine(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, monitorDocument)
	writeMonitorURL(test, harness, served)
	address, host := served.url(), served.server.Listener.Addr().String()
	served.server.Close()
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	var want strings.Builder
	for _, line := range strings.SplitAfter(allMetricsOnWindows, "\n") {
		if !strings.HasPrefix(line, "cpu.temp") && !strings.HasPrefix(line, "cpu.ccd") && !strings.HasPrefix(line, "cpu.clock") &&
			!strings.HasPrefix(line, "ram.1") && !strings.HasPrefix(line, "nvme.") {
			want.WriteString(line)
		}
	}
	if harness.stdout.String() != want.String() {
		test.Errorf("stdout %q", harness.stdout.String())
	}
	if harness.stderr.String() != refusedLine(address, host) {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestSensorsOnWindowsWithADocumentThatGivesNoMonitorMetricSaysSo(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, noMonitorMetricDocument)
	writeMonitorURL(test, harness, served)
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	want := "LibreHardwareMonitor at " + served.url() + " gives no temperature and no clock that sensordeck knows\n"
	if harness.stderr.String() != want || strings.Contains(harness.stdout.String(), "cpu.temp") || !strings.Contains(harness.stdout.String(), "ram.used") {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
}

func TestSensorsOnWindowsLeavesOutTheUserInformationOfTheURLInTheLine(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, monitorDocument)
	served.failing.Store(true)
	writeDefaultConfig(test, harness, "librehardwaremonitor_url: "+strings.Replace(served.url(), "http://", "http://monitor:s3cret@", 1)+"\ntags: {cpu.temp: 1}\n")
	harness.run("sensors")
	want := "LibreHardwareMonitor cannot be read at " + served.url() + ": status 500 Internal Server Error. " +
		"The temperatures and cpu.clock need it running, with Options > Remote Web Server > Run checked\n"
	if harness.stderr.String() != want || strings.Contains(harness.stderr.String(), "s3cret") {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestSensorsOnWindowsTakesTheDefaultURLWithNoConfig(test *testing.T) {
	cases := map[string]func(*harness){
		"no file": func(*harness) {},
		"no config directory": func(harness *harness) {
			harness.configDirErr = errors.New("no directory")
		},
	}
	for name, setup := range cases {
		harness, _ := windowsHarness(test)
		setup(harness)
		if status := harness.run("sensors"); status != 0 {
			test.Fatalf("%s: status %d", name, status)
		}
		// Nothing is to listen at the default address in the test, so the line names it.
		if !strings.HasPrefix(harness.stderr.String(), "LibreHardwareMonitor cannot be read at http://localhost:8085/data.json: ") ||
			strings.Count(harness.stderr.String(), "\n") != 1 {
			test.Errorf("%s: stderr %q", name, harness.stderr.String())
		}
	}
}

func TestSensorsOnWindowsWithAConfigThatCannotBeReadOrIsNotValidSaysSoAndGoesOnWithTheDefault(test *testing.T) {
	harness, _ := windowsHarness(test)
	path := writeDefaultConfig(test, harness, "")
	if err := os.Remove(path); err != nil {
		test.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		test.Fatal(err)
	}
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	lines := strings.Split(harness.stderr.String(), "\n")
	if lines[0] != "cannot read the config file "+path+": is a directory" ||
		!strings.HasPrefix(lines[1], "LibreHardwareMonitor cannot be read at http://localhost:8085/data.json: ") {
		test.Errorf("stderr %q", harness.stderr.String())
	}

	invalid, _ := windowsHarness(test)
	path = writeDefaultConfig(test, invalid, "librehardwaremonitor_url: ftp://host/\ntags: {cpu.temp: 1}\n")
	if status := invalid.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	lines = strings.Split(invalid.stderr.String(), "\n")
	wantFirst := "config file " + path + `: librehardwaremonitor_url: "ftp://host/" is not an http URL such as http://localhost:8085/data.json`
	if lines[0] != wantFirst || !strings.HasPrefix(lines[1], "LibreHardwareMonitor cannot be read at http://localhost:8085/data.json: ") ||
		!strings.Contains(invalid.stdout.String(), "ram.used") {
		test.Errorf("stderr %q", invalid.stderr.String())
	}
}

func TestSensorsOnWindowsWithNothingReadableWritesTheLineOfTheMonitorBeforeTheOthers(test *testing.T) {
	harness, seam := windowsHarness(test)
	test.Setenv("PATH", test.TempDir())
	seam.MemoryError = errors.New("failed")
	seam.TimesError = errors.New("failed")
	seam.InterfacesError = errors.New("failed")
	harness.controller.(*fakeSource).values = nil
	harness.controllerProblem = errors.New("no collection of interface 0 matches: p")
	served := newMonitor(test, monitorDocument)
	writeMonitorURL(test, harness, served)
	address, host := served.url(), served.server.Listener.Addr().String()
	served.server.Close()
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	want := refusedLine(address, host) + "no collection of interface 0 matches: p\nno sensor could be read\n"
	if harness.stdout.Len() != 0 || harness.stderr.String() != want {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
}

// runOnWindows runs with a config that maps the tags and a stop after the given number of waits, calling change after
// each wait so that a test can make the server fail and recover.
func runOnWindows(test *testing.T, harness *harness, config string, stopAfter int, change func(count int)) {
	test.Helper()
	harness.onWait = func(count int) {
		if change != nil {
			change(count)
		}
		if count >= stopAfter {
			harness.cancelStop()
		}
	}
	if status := harness.run("run", "--config", writeConfig(test, config)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
}

func TestRunOnWindowsSendsTheNativeAndTheMonitorMetricsAndWritesOneLinePerOutage(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, monitorDocument)
	config := "librehardwaremonitor_url: " + served.url() + "\ntags: {ram.used: 1, cpu.temp: 2}\n"
	runOnWindows(test, harness, config, 4, func(count int) {
		switch count {
		case 1:
			served.failing.Store(true) // the rounds 2 and 3 fail
		case 3:
			served.failing.Store(false)
		}
	})
	both := valueReport(smartmonitor.Value{Tag: 1, Value: sensors.ToValue(20)}, smartmonitor.Value{Tag: 2, Value: sensors.ToValue(46.9)})
	native := valueReport(smartmonitor.Value{Tag: 1, Value: sensors.ToValue(20)})
	reports := valueReports(harness)
	if len(reports) != 4 || reports[0] != both || reports[1] != native || reports[2] != native || reports[3] != both {
		test.Errorf("%d reports", len(reports))
	}
	want := "LibreHardwareMonitor cannot be read at " + served.url() + ": status 500 Internal Server Error. " +
		"The temperatures and cpu.clock need it running, with Options > Remote Web Server > Run checked\n" +
		"LibreHardwareMonitor read again at " + served.url() + "\n"
	if harness.stderr.String() != want {
		test.Errorf("stderr %q", harness.stderr.String())
	}
	if served.requests.Load() != 4 {
		test.Errorf("%d requests", served.requests.Load())
	}
}

func TestRunOnWindowsWithOnlyNativeMetricsMakesNoRequest(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, monitorDocument)
	config := "librehardwaremonitor_url: " + served.url() + "\ntags: {ram.used: 1, net.up: 2}\n"
	runOnWindows(test, harness, config, 3, nil)
	if served.requests.Load() != 0 || harness.stderr.Len() != 0 {
		test.Errorf("%d requests, stderr %q", served.requests.Load(), harness.stderr.String())
	}
	if len(valueReports(harness)) == 0 {
		test.Error("no value was sent")
	}
}

func TestRunOnWindowsEndsWhenStoppedWhileTheMonitorHoldsARequest(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, monitorDocument)
	served.holding = make(chan struct{}, 1)
	config := "librehardwaremonitor_url: " + served.url() + "\ntags: {cpu.temp: 1}\n"
	go func() {
		<-served.holding
		harness.cancelStop()
	}()
	started := time.Now()
	if status := harness.run("run", "--config", writeConfig(test, config)); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if took := time.Since(started); took > 3*time.Second {
		test.Errorf("run took %v", took)
	}
}

func TestSensorsOnWindowsInterruptedWhileTheMonitorHoldsARequestWritesNoLine(test *testing.T) {
	harness, _ := windowsHarness(test)
	served := newMonitor(test, monitorDocument)
	served.holding = make(chan struct{}, 1)
	writeMonitorURL(test, harness, served)
	go func() {
		<-served.holding
		harness.cancelStop()
	}()
	harness.expectFailure(test, "interrupted", "sensors")
}
