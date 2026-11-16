package cli

import (
	"path/filepath"
	"testing"

	"github.com/norareit/sensordeck/internal/sensors"
)

func sensorsHarness(test *testing.T) *harness {
	test.Helper()
	harness := newHarness()
	harness.configDir = test.TempDir()
	harness.sources = []sensors.Source{&fakeSource{names: []string{"cpu.temp"}, values: map[string]float64{"cpu.temp": 51}}}
	return harness
}

func TestSensorsListsCustomMetricsAfterTheBuiltInOnesWithTheirUnits(test *testing.T) {
	harness := sensorsHarness(test)
	value := writeConfig(test, "40000\n")
	second := writeConfig(test, "7\n")
	writeDefaultConfig(test, harness, "metrics:\n  zz.first: {file: "+value+", scale: 0.001, unit: \"°C\"}\n  aa.second: {file: "+second+"}\ntags: {cpu.temp: 1}\n")
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	want := "cpu.temp\t51.0\t°C\nzz.first\t40.0\t°C\naa.second\t7.0\t\n"
	if got := harness.stdout.String(); got != want {
		test.Errorf("stdout %q, want %q", got, want)
	}
	if harness.stderr.Len() != 0 {
		test.Errorf("stderr %q", harness.stderr.String())
	}
}

func TestSensorsReadsTheConfigNamedWithTheFlag(test *testing.T) {
	harness := sensorsHarness(test)
	value := writeConfig(test, "5\n")
	writeDefaultConfig(test, harness, "metrics: {default.one: {file: "+value+"}}\ntags: {cpu.temp: 1}\n")
	named := writeConfig(test, "metrics: {named.one: {file: "+value+", unit: W}}\ntags: {cpu.temp: 1}\n")
	if status := harness.run("sensors", "--config", named); status != 0 {
		test.Fatalf("status %d", status)
	}
	if want := "cpu.temp\t51.0\t°C\nnamed.one\t5.0\tW\n"; harness.stdout.String() != want {
		test.Errorf("stdout %q, want %q", harness.stdout.String(), want)
	}
}

func TestSensorsWithAMissingDefaultConfigListsTheBuiltInMetricsWithNoLine(test *testing.T) {
	harness := sensorsHarness(test)
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	if harness.stdout.String() != "cpu.temp\t51.0\t°C\n" || harness.stderr.Len() != 0 {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
}

func TestSensorsWithAMissingNamedConfigOrOneThatIsNotValidWritesOneLineAndListsTheBuiltInMetrics(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "none.yaml")
	invalid := writeConfig(test, "tags: [\n")
	cases := map[string]string{
		missing: "cannot read the config file " + missing + ": no such file or directory\n",
		invalid: "",
	}
	for path, want := range cases {
		harness := sensorsHarness(test)
		if status := harness.run("sensors", "--config", path); status != 0 {
			test.Fatalf("status %d", status)
		}
		if harness.stdout.String() != "cpu.temp\t51.0\t°C\n" {
			test.Errorf("stdout %q", harness.stdout.String())
		}
		got := harness.stderr.String()
		if want != "" && got != want || want == "" && (len(got) == 0 || got[:len("config file "+path+": ")] != "config file "+path+": ") {
			test.Errorf("stderr %q", got)
		}
	}
}

func TestSensorsWritesTheLineOfACustomMetricWithNoValue(test *testing.T) {
	harness := sensorsHarness(test)
	missing := filepath.Join(test.TempDir(), "none")
	writeDefaultConfig(test, harness, "metrics: {gone.one: {file: "+missing+"}}\ntags: {cpu.temp: 1}\n")
	if status := harness.run("sensors"); status != 0 {
		test.Fatalf("status %d", status)
	}
	want := "metric gone.one: cannot read " + missing + ": no such file or directory\n"
	if harness.stderr.String() != want || harness.stdout.String() != "cpu.temp\t51.0\t°C\n" {
		test.Errorf("stdout %q, stderr %q", harness.stdout.String(), harness.stderr.String())
	}
}

func TestSensorsSaysNoSensorCanBeReadOnlyWhenNoMetricHasAValue(test *testing.T) {
	value := writeConfig(test, "3\n")
	for name, content := range map[string]string{
		"a custom value": "metrics: {a.b: {file: " + value + "}}\ntags: {cpu.temp: 1}\n",
		"no value":       "metrics: {a.b: {file: /nowhere/x}}\ntags: {cpu.temp: 1}\n",
	} {
		harness := newHarness()
		harness.configDir = test.TempDir()
		writeDefaultConfig(test, harness, content)
		harness.run("sensors")
		said := harness.stderr.String()
		hasMessage := len(said) >= len(noSensorMessage) && said[len(said)-len(noSensorMessage)-1:] == noSensorMessage+"\n"
		if hasMessage != (name == "no value") {
			test.Errorf("%s: stderr %q", name, said)
		}
	}
}
