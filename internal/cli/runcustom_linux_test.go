package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// customTheme is the theme of the tests of the file with a number widget of the metric.
func customTheme(metric string) string {
	return strings.Replace(compiledTheme, "cpu.temp", metric, 1)
}

func TestLoadConfigTakesATagForADefinedCustomMetric(test *testing.T) {
	folder := writeThemeFolder(test, customTheme("water.temp"))
	content := "metrics: {water.temp: {file: /sys/a}}\ntheme: '" + folder + "'\n"
	loaded, err := newSession().loadConfig("/c/config.yaml", []byte(content))
	if err != nil || len(loaded.settings.Mappings) != 1 || loaded.settings.Mappings[0].Name != "water.temp" {
		test.Errorf("got %+v, %v", loaded.settings.Mappings, err)
	}
}

func TestLoadConfigRefusesAThemeMetricThatIsNeitherBuiltInNorDefined(test *testing.T) {
	folder := writeThemeFolder(test, customTheme("water.temp"))
	_, err := newSession().loadConfig("/c/config.yaml", []byte("theme: '"+folder+"'\n"))
	want := "theme: " + folder + `: the metric "water.temp" is neither built in nor defined under metrics`
	if err == nil || err.Error() != want {
		test.Errorf("error %v, want %q", err, want)
	}
}

func TestLoadConfigRefusesABarMetricThatIsNeitherBuiltInNorDefined(test *testing.T) {
	theme := strings.Replace(compiledTheme, "  - {type: number, font: body, metric: cpu.temp, x: 4, y: 40}\n",
		"  - {type: bar, metric: water.temp, x: 4, y: 40, width: 100, height: 8, color: \"#00ff00\"}\n", 1)
	folder := writeThemeFolder(test, theme)
	_, err := newSession().loadConfig("/c/config.yaml", []byte("theme: '"+folder+"'\n"))
	want := "theme: " + folder + `: the metric "water.temp" is neither built in nor defined under metrics`
	if err == nil || err.Error() != want {
		test.Errorf("error %v, want %q", err, want)
	}
}

func TestLoadConfigAllowsABuiltInMetricThatTheMachineLacks(test *testing.T) {
	folder := writeThemeFolder(test, customTheme("nvme.9.temp"))
	if _, err := newSession().loadConfig("/c/config.yaml", []byte("theme: '"+folder+"'\n")); err != nil {
		test.Error(err)
	}
}

func TestRunRefusesAReloadWhoseThemeNamesAnUndefinedMetric(test *testing.T) {
	folder := writeThemeFolder(test, customTheme("water.temp"))
	harness := runWith(twoMetrics(), 3)
	path := writeConfig(test, "tags: {cpu.temp: 1}\n")
	changeAtWait(harness, 1, 3, func() { rewrite(test, path, "theme: '"+folder+"'\n", 1) })
	harness.run("run", "--config", path)
	want := "config file " + path + ": theme: " + folder + `: the metric "water.temp" is neither built in nor defined under metrics, keeping the running config` + "\n"
	if harness.stderr.String() != want {
		test.Errorf("stderr %q, want %q", harness.stderr.String(), want)
	}
	for _, report := range valueReports(harness) {
		if report != valueReport(smartmonitor.Value{Tag: 1, Value: 42}) {
			test.Errorf("a report changed: % X", report[:8])
		}
	}
}

func TestRunAppliesAReloadThatAddsACustomMetricAndItsDefinition(test *testing.T) {
	harness := runWith(twoMetrics(), 3)
	value := writeConfig(test, "40\n")
	path := writeConfig(test, "tags: {cpu.temp: 1}\n")
	changeAtWait(harness, 1, 3, func() {
		rewrite(test, path, "metrics: {water.temp: {file: "+value+"}}\ntags: {water.temp: 5}\n", 1)
	})
	harness.run("run", "--config", path)
	reports := valueReports(harness)
	if len(reports) != 3 || reports[2] != valueReport(smartmonitor.Value{Tag: 5, Value: 40}) {
		test.Errorf("%d reports", len(reports))
	}
}

func TestRunSendsAFileMetricThroughToValueAndSaysWhenItIsLostAndBack(test *testing.T) {
	harness := runWith(twoMetrics(), 5)
	value := writeConfig(test, "51.5\n")
	path := writeConfig(test, "metrics: {water.temp: {file: "+value+", scale: 1}}\ntags: {cpu.temp: 1, water.temp: 5}\n")
	harness.onWait = func(count int) {
		switch count {
		case 1:
			_ = os.WriteFile(value, []byte("none\n"), 0o644)
		case 2:
			_ = os.WriteFile(value, []byte("7\n"), 0o644)
		case 5:
			harness.cancelStop()
		}
	}
	harness.run("run", "--config", path)
	reports := valueReports(harness)
	if len(reports) != 5 {
		test.Fatalf("%d reports", len(reports))
	}
	both := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 5, Value: 52})
	only := valueReport(smartmonitor.Value{Tag: 1, Value: 42})
	back := valueReport(smartmonitor.Value{Tag: 1, Value: 42}, smartmonitor.Value{Tag: 5, Value: 7})
	if reports[0] != both || reports[1] != only || reports[2] != back {
		test.Error("the reports do not follow the file")
	}
	want := "metric water.temp: no number in " + value + "\nmetric water.temp: read again\n"
	if harness.stderr.String() != want {
		test.Errorf("stderr %q, want %q", harness.stderr.String(), want)
	}
}

func TestRunDoesNotWaitForACommandMetric(test *testing.T) {
	harness := runWith(twoMetrics(), 2)
	path := writeConfig(test, "metrics: {slow.slow: {command: 'sleep 1; echo 9'}}\ntags: {cpu.temp: 1, slow.slow: 5}\n")
	started := time.Now()
	harness.run("run", "--config", path)
	if time.Since(started) > 900*time.Millisecond {
		test.Errorf("run lasted %v", time.Since(started))
	}
	reports := valueReports(harness)
	if len(reports) != 2 || reports[0] != valueReport(smartmonitor.Value{Tag: 1, Value: 42}) {
		test.Errorf("%d reports", len(reports))
	}
}
