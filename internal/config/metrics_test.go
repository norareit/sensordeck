package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func parseMetricsFile(text string) (Config, error) {
	return Parse([]byte("tags: {cpu.temp: 1}\n" + text))
}

func TestMetricsDefaultsOfEachForm(test *testing.T) {
	got, err := parseMetricsFile(`metrics:
  water.temp: {file: /sys/a}
  gpu.amd.temp: {hwmon: amdgpu, input: temp1_input}
  outside.temp: {command: "echo 1"}
`)
	if err != nil {
		test.Fatal(err)
	}
	want := []Metric{
		{Name: "water.temp", File: "/sys/a", Scale: 1},
		{Name: "gpu.amd.temp", Hwmon: "amdgpu", Input: "temp1_input", Index: 1, Scale: 1},
		{Name: "outside.temp", Command: "echo 1", Every: DefaultEvery, Scale: 1},
	}
	if !reflect.DeepEqual(got.Metrics, want) {
		test.Errorf("got %+v", got.Metrics)
	}
}

func TestMetricsWithEveryKey(test *testing.T) {
	got, err := parseMetricsFile(`metrics:
  fan.pump: {hwmon: nct6799, input: fan2_input, index: 2, scale: 0.5, offset: -3, unit: RPM}
  outside.temp: {command: "echo 1", every: 5m, unit: "°C"}
`)
	if err != nil {
		test.Fatal(err)
	}
	want := []Metric{
		{Name: "fan.pump", Hwmon: "nct6799", Input: "fan2_input", Index: 2, Scale: 0.5, Offset: -3, Unit: "RPM"},
		{Name: "outside.temp", Command: "echo 1", Every: 5 * time.Minute, Scale: 1, Unit: "°C"},
	}
	if !reflect.DeepEqual(got.Metrics, want) {
		test.Errorf("got %+v", got.Metrics)
	}
}

func TestMetricsKeepTheOrderOfTheFile(test *testing.T) {
	got, err := parseMetricsFile("metrics: {z.a: {file: /a}, a.z: {file: /b}, m.m: {file: /c}}\n")
	if err != nil {
		test.Fatal(err)
	}
	var names []string
	for _, each := range got.Metrics {
		names = append(names, each.Name)
	}
	if !reflect.DeepEqual(names, []string{"z.a", "a.z", "m.m"}) {
		test.Errorf("names = %v", names)
	}
}

func TestMetricsErrors(test *testing.T) {
	cases := []struct{ name, text, want string }{
		{"empty", "metrics: {}\n", "metrics: want a mapping from metric name to definition, with at least one entry"},
		{"null", "metrics:\n", "metrics: want a mapping from metric name to definition, with at least one entry"},
		{"list", "metrics: [a]\n", "metrics: want a mapping from metric name to definition, with at least one entry"},
		{"repeated name", "metrics:\n  a.b: {file: /a}\n  a.b: {file: /b}\n", "already defined"},
		{"malformed name", "metrics: {A.b: {file: /a}}\n", `metrics: "A.b" is not a metric name: segments of a to z, 0 to 9, _ and - joined by dots, starting with a letter`},
		{"built in", "metrics: {cpu.temp: {file: /a}}\n", "metrics: cpu.temp: is a built-in metric and cannot be replaced"},
		{"built in without hardware", "metrics: {nvme.9.temp: {file: /a}}\n", "metrics: nvme.9.temp: is a built-in metric and cannot be replaced"},
		{"scalar", "metrics: {a.b: 3}\n", "metrics: a.b: want a mapping with one of the keys file, hwmon and command"},
		{"repeated key", "metrics: {a.b: {file: /a, file: /b}}\n", "already defined"},
		{"unknown key", "metrics: {a.b: {file: /a, speed: 1}}\n", `metrics: a.b: unknown key "speed", the keys are command, every, file, hwmon, index, input, offset, scale and unit`},
		{"no source", "metrics: {a.b: {scale: 2}}\n", "metrics: a.b: want exactly one of the keys file, hwmon and command"},
		{"two sources", "metrics: {a.b: {file: /a, command: x}}\n", "metrics: a.b: want exactly one of the keys file, hwmon and command"},
		{"input without hwmon", "metrics: {a.b: {file: /a, input: x}}\n", "metrics: a.b: input: only with hwmon"},
		{"index without hwmon", "metrics: {a.b: {file: /a, index: 1}}\n", "metrics: a.b: index: only with hwmon"},
		{"every without command", "metrics: {a.b: {file: /a, every: 5s}}\n", "metrics: a.b: every: only with command"},
		{"hwmon without input", "metrics: {a.b: {hwmon: x}}\n", "metrics: a.b: input: needed with hwmon"},
		{"relative file", "metrics: {a.b: {file: a/b}}\n", "metrics: a.b: file: want an absolute path"},
		{"empty hwmon", "metrics: {a.b: {hwmon: \"\", input: x}}\n", "metrics: a.b: hwmon: want a chip name, a non-empty string without /"},
		{"hwmon with slash", "metrics: {a.b: {hwmon: a/b, input: x}}\n", "metrics: a.b: hwmon: want a chip name, a non-empty string without /"},
		{"input dot dot", "metrics: {a.b: {hwmon: x, input: ..}}\n", "metrics: a.b: input: want the name of a file of the chip, a non-empty string other than . and .. without /"},
		{"input with slash", "metrics: {a.b: {hwmon: x, input: a/b}}\n", "metrics: a.b: input: want the name of a file of the chip, a non-empty string other than . and .. without /"},
		{"index 0", "metrics: {a.b: {hwmon: x, input: y, index: 0}}\n", "metrics: a.b: index: want a whole number of 1 or more"},
		{"index negative", "metrics: {a.b: {hwmon: x, input: y, index: -1}}\n", "metrics: a.b: index: want a whole number of 1 or more"},
		{"index fraction", "metrics: {a.b: {hwmon: x, input: y, index: 1.5}}\n", "metrics: a.b: index: want a whole number of 1 or more"},
		{"index quoted", "metrics: {a.b: {hwmon: x, input: y, index: \"2\"}}\n", "metrics: a.b: index: want a whole number of 1 or more"},
		{"empty command", "metrics: {a.b: {command: \"\"}}\n", "metrics: a.b: command: want a non-empty string"},
		{"scale inf", "metrics: {a.b: {file: /a, scale: .inf}}\n", "metrics: a.b: scale: want a finite number"},
		{"scale nan", "metrics: {a.b: {file: /a, scale: .nan}}\n", "metrics: a.b: scale: want a finite number"},
		{"scale quoted", "metrics: {a.b: {file: /a, scale: \"1\"}}\n", "metrics: a.b: scale: want a finite number"},
		{"offset text", "metrics: {a.b: {file: /a, offset: x}}\n", "metrics: a.b: offset: want a finite number"},
		{"unit with tab", "metrics: {a.b: {file: /a, unit: \"a\\tb\"}}\n", "metrics: a.b: unit: want a string without tabs, line feeds or other control characters"},
		{"unit with line feed", "metrics: {a.b: {file: /a, unit: \"a\\nb\"}}\n", "metrics: a.b: unit: want a string without tabs, line feeds or other control characters"},
		{"every text", "metrics: {a.b: {command: x, every: soon}}\n", `metrics: a.b: every: "soon" is not a duration such as 30s or 5m`},
		{"every short", "metrics: {a.b: {command: x, every: 500ms}}\n", "metrics: a.b: every: 500ms is shorter than 1s"},
	}
	for _, each := range cases {
		_, err := parseMetricsFile(each.text)
		if err == nil || !strings.Contains(err.Error(), each.want) {
			test.Errorf("%s: error %v, want %q", each.name, err, each.want)
		}
	}
}

func TestTagsAcceptADefinedCustomNameAndRefuseAnUndefinedOne(test *testing.T) {
	got, err := Parse([]byte("metrics: {water.temp: {file: /a}}\ntags: {water.temp: 3}\n"))
	if err != nil || len(got.Mappings) != 1 || got.Mappings[0].Name != "water.temp" {
		test.Fatalf("%+v, %v", got, err)
	}
	_, err = Parse([]byte("metrics: {water.temp: {file: /a}}\ntags: {fan.pump: 3}\n"))
	if err == nil || err.Error() != `tags: "fan.pump" is not a metric name` {
		test.Fatalf("error %v", err)
	}
}

func TestIsMetric(test *testing.T) {
	settings := Config{Metrics: []Metric{{Name: "water.temp"}}}
	if !settings.IsMetric("cpu.temp") || !settings.IsMetric("nvme.9.temp") || !settings.IsMetric("water.temp") || settings.IsMetric("fan.pump") {
		test.Error("IsMetric is wrong")
	}
}
