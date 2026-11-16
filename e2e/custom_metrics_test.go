//go:build linux

package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"testing"

	"github.com/norareit/sensordeck/internal/fakesensors"
)

// customTheme shows the three custom metrics of customMetricsConfig.
const customTheme = "name: custom\nbackground: \"#102030\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\n" +
	"widgets:\n" +
	"  - {type: number, font: body, metric: water.temp, x: 4, y: 4}\n" +
	"  - {type: number, font: body, metric: fan.pump, x: 4, y: 40}\n" +
	"  - {type: number, font: body, metric: out.value, x: 4, y: 80}\n"

// customMetricsSetup gives a setup whose config defines a file metric, a hwmon metric read with index 2 among two chips
// of one name, and a command metric, and the theme that names them.
func customMetricsSetup(test *testing.T) (*themeSetup, string) {
	test.Helper()
	setup := newThemeSetup(test)
	tree := fakesensors.New(test, setup.fake.SysfsRoot)
	// The numbers of the chips are in the opposite order of their device paths, so index 2 is hwmon2.
	writeFile(test, filepath.Join(tree.Hwmon(2, "nct6799", "pci0000:00/b"), "fan2_input"), "2500\n")
	writeFile(test, filepath.Join(tree.Hwmon(3, "nct6799", "pci0000:00/a"), "fan2_input"), "1500\n")
	probe := filepath.Join(test.TempDir(), "temp1_input")
	writeFile(test, probe, "42000\n")
	folder := writeThemeFolder(test, customTheme)
	config := "interval: 100ms\ntheme: '" + folder + "'\nmetrics:\n" +
		"  water.temp: {file: " + probe + ", scale: 0.001, unit: \"°C\"}\n" +
		"  fan.pump: {hwmon: nct6799, input: fan2_input, index: 2, unit: RPM}\n" +
		"  out.value: {command: \"echo value 77\"}\n"
	return setup, writeConfigFile(test, config)
}

func writeFile(test *testing.T, path, content string) {
	test.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
}

func TestSensorsListsCustomMetricsAfterTheBuiltInOnes(test *testing.T) {
	setup, config := customMetricsSetup(test)
	outcome := runWith(test, setup.fake, setup.environment, "sensors", "--config", config)
	want := "cpu.temp\t42.0\t°C\ncpu.clock\t1234.0\tMHz\nwater.temp\t42.0\t°C\nfan.pump\t2500.0\tRPM\nout.value\t77.0\t\n"
	outcome.expect(test, 0, want, "")
}

// compileTheme compiles the theme folder and gives the compiled file with a function that gives the tag of a metric.
func compileTheme(test *testing.T, setup *themeSetup, folder string) ([]byte, func(string) uint8) {
	test.Helper()
	out := filepath.Join(test.TempDir(), "custom.dat")
	if compiling := runWith(test, setup.fake, setup.environment, "theme", "compile", folder, out); compiling.status != 0 {
		test.Fatalf("theme compile: %q", compiling.stderr)
	}
	tags, err := os.ReadFile(tagsFilePath(out))
	if err != nil {
		test.Fatal(err)
	}
	tagOf := func(name string) uint8 {
		match := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(name) + `:\s*(\d+)`).FindSubmatch(tags)
		if match == nil {
			test.Fatalf("no tag of %s in %q", name, tags)
		}
		number, _ := strconv.Atoi(string(match[1]))
		return uint8(number)
	}
	compiled, err := os.ReadFile(out)
	if err != nil {
		test.Fatal(err)
	}
	return compiled, tagOf
}

func TestRunSendsCustomMetricsToTheTagsOfTheTheme(test *testing.T) {
	setup, config := customMetricsSetup(test)
	compiled, tagOf := compileTheme(test, setup, writeThemeFolder(test, customTheme))
	setup.screen.AcceptUpload(compiled)
	running := start(test, setup.fake, setup.environment, "run", "--config", config)
	position := setup.expectUploadAt(0, compiled)
	_, found := waitForReport(setup.screen, position, uploadLimit, func(pairs map[uint8]uint16) bool {
		return pairs != nil && pairs[tagOf("water.temp")] == 42 && pairs[tagOf("fan.pump")] == 2500 && pairs[tagOf("out.value")] == 77
	})
	if !found {
		test.Error("no value report holds the three custom values")
	}
	running.stop(syscall.SIGTERM)
}

func TestRunSendsNegativeAndClampedCustomMetrics(test *testing.T) {
	setup := newThemeSetup(test)
	below := filepath.Join(test.TempDir(), "below")
	above := filepath.Join(test.TempDir(), "above")
	writeFile(test, below, "-5\n")
	writeFile(test, above, "40000\n")
	folder := writeThemeFolder(test, "name: signed\nbackground: \"#102030\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\n"+
		"widgets:\n  - {type: number, font: body, metric: below, x: 4, y: 4}\n  - {type: number, font: body, metric: above, x: 4, y: 40}\n")
	config := writeConfigFile(test, "interval: 100ms\ntheme: '"+folder+"'\nmetrics:\n  below: {file: "+below+"}\n  above: {file: "+above+"}\n")
	compiled, tagOf := compileTheme(test, setup, folder)
	setup.screen.AcceptUpload(compiled)
	running := start(test, setup.fake, setup.environment, "run", "--config", config)
	position := setup.expectUploadAt(0, compiled)
	_, found := waitForReport(setup.screen, position, uploadLimit, func(pairs map[uint8]uint16) bool {
		return pairs != nil && pairs[tagOf("below")] == 0xFFFB && pairs[tagOf("above")] == 0x7FFF
	})
	if !found {
		test.Error("no value report holds -5 as FF FB and 40000 as 7F FF")
	}
	running.stop(syscall.SIGTERM)
}
