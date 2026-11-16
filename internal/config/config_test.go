package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseTheExample(test *testing.T) {
	got, err := Parse([]byte("interval: 1s\ntags:\n  cpu.temp: 1\n  cpu.clock: 2\n"))
	if err != nil {
		test.Fatal(err)
	}
	want := Config{Interval: time.Second, LibreHardwareMonitorURL: DefaultMonitorURL(), Mappings: []Mapping{{"cpu.temp", 1}, {"cpu.clock", 2}}}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %+v", got)
	}
}

func TestParseIntervals(test *testing.T) {
	cases := map[string]time.Duration{
		"interval: 500ms\n": 500 * time.Millisecond,
		"interval: 100ms\n": 100 * time.Millisecond,
		"":                  time.Second,
	}
	for interval, want := range cases {
		got, err := Parse([]byte(interval + "tags:\n  cpu.temp: 1\n"))
		if err != nil || got.Interval != want {
			test.Errorf("%q: %v, %v", interval, got.Interval, err)
		}
	}
}

func TestParseOrdersMappingsByTag(test *testing.T) {
	got, err := Parse([]byte("tags: {gpu.temp: 9, cpu.temp: 0, ram.used: 255, cpu.clock: 0x10}\n"))
	if err != nil {
		test.Fatal(err)
	}
	want := []Mapping{{"cpu.temp", 0}, {"gpu.temp", 9}, {"cpu.clock", 16}, {"ram.used", 255}}
	if !reflect.DeepEqual(got.Mappings, want) {
		test.Errorf("got %+v", got.Mappings)
	}
}

func TestParseTagSpellings(test *testing.T) {
	cases := map[string]uint8{"0xFF": 255, "0o17": 15, "017": 15, "0b11": 3, "1_0": 10, "0x1_0": 16, "+5": 5, "0": 0, "255": 255}
	for spelling, want := range cases {
		got, err := Parse([]byte("tags: {cpu.temp: " + spelling + "}\n"))
		if err != nil || got.Mappings[0].Tag != want {
			test.Errorf("%s: %+v, %v", spelling, got, err)
		}
	}
}

const colourEmpty = `commander_color: no colour given, write it in quotes such as "#00c8ff", because # starts a comment in YAML`

func monitorURLProblem(text string) string {
	return `librehardwaremonitor_url: "` + text + `" is not an http URL such as http://localhost:8085/data.json`
}

func TestParseErrors(test *testing.T) {
	cases := []struct{ name, content, want string }{
		{"empty", "", "want a mapping with the keys commander_color, commander_interval, interval, librehardwaremonitor_url, metrics, tags and theme"},
		{"comment", "# nothing\n", "want a mapping with the keys commander_color, commander_interval, interval, librehardwaremonitor_url, metrics, tags and theme"},
		{"list", "- tags\n", "want a mapping with the keys commander_color, commander_interval, interval, librehardwaremonitor_url, metrics, tags and theme"},
		{"unknown key", "colour: red\ntags: {cpu.temp: 1}\n", `unknown key "colour", the keys are commander_color, commander_interval, interval, librehardwaremonitor_url, metrics, tags and theme`},
		{"neither theme nor tags", "interval: 1s\n", "one of theme and tags is needed"},
		{"theme and tags", "theme: minimal-dark\ntags: {cpu.temp: 1}\n", "theme and tags cannot both be given, the theme gives the tags"},
		{"theme and tags without value", "theme: minimal-dark\ntags:\n", "theme and tags cannot both be given, the theme gives the tags"},
		{"theme number", "theme: 2024\n", "theme: want a theme name or folder path as a string"},
		{"theme list", "theme: [a]\n", "theme: want a theme name or folder path as a string"},
		{"theme mapping", "theme: {a: 1}\n", "theme: want a theme name or folder path as a string"},
		{"theme empty", "theme: \"\"\n", "theme: the theme name or folder path is empty"},
		{"theme without value", "theme:\n", "theme: the theme name or folder path is empty"},
		{"tags without value", "tags:\n", "tags: want a mapping from metric name to tag, with at least one entry"},
		{"empty tags", "tags: {}\n", "tags: want a mapping from metric name to tag, with at least one entry"},
		{"list tags", "tags: [1]\n", "tags: want a mapping from metric name to tag, with at least one entry"},
		{"not a metric", "tags: {cpu.tmp: 1}\n", `tags: "cpu.tmp" is not a metric name`},
		{"float tag", "tags: {cpu.temp: 1.0}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"string tag", "tags: {cpu.temp: \"1\"}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"word tag", "tags: {cpu.temp: x}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"mapping tag", "tags: {cpu.temp: {a: 1}}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"big tag", "tags: {cpu.temp: 256}\n", "tags: cpu.temp: the tag 256 is outside 0 to 255"},
		{"huge tag", "tags: {cpu.temp: 9223372036854775808}\n", "tags: cpu.temp: the tag 9223372036854775808 is outside 0 to 255"},
		{"negative huge tag", "tags: {cpu.temp: -9223372036854775809}\n", "tags: cpu.temp: the tag -9223372036854775809 is outside 0 to 255"},
		{"hex huge tag", "tags: {cpu.temp: 0x8000000000000000}\n", "tags: cpu.temp: the tag 0x8000000000000000 is outside 0 to 255"},
		{"underscore huge tag", "tags: {cpu.temp: 9_223_372_036_854_775_808}\n", "tags: cpu.temp: the tag 9_223_372_036_854_775_808 is outside 0 to 255"},
		{"negative underscore huge tag", "tags: {cpu.temp: -9_223_372_036_854_775_809}\n", "tags: cpu.temp: the tag -9_223_372_036_854_775_809 is outside 0 to 255"},
		{"hex big tag", "tags: {cpu.temp: 0x100}\n", "tags: cpu.temp: the tag 0x100 is outside 0 to 255"},
		{"octal big tag", "tags: {cpu.temp: 0o400}\n", "tags: cpu.temp: the tag 0o400 is outside 0 to 255"},
		{"legacy octal big tag", "tags: {cpu.temp: 0400}\n", "tags: cpu.temp: the tag 0400 is outside 0 to 255"},
		{"binary big tag", "tags: {cpu.temp: 0b100000000}\n", "tags: cpu.temp: the tag 0b100000000 is outside 0 to 255"},
		{"underscore big tag", "tags: {cpu.temp: 2_56}\n", "tags: cpu.temp: the tag 2_56 is outside 0 to 255"},
		{"hex negative tag", "tags: {cpu.temp: -0x1}\n", "tags: cpu.temp: the tag -0x1 is outside 0 to 255"},
		{"octal negative tag", "tags: {cpu.temp: -0o1}\n", "tags: cpu.temp: the tag -0o1 is outside 0 to 255"},
		{"exponent tag", "tags: {cpu.temp: 1e3}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"negative tag", "tags: {cpu.temp: -1}\n", "tags: cpu.temp: the tag -1 is outside 0 to 255"},
		{"same tag", "tags: {cpu.temp: 1, cpu.clock: 1}\n", "tags: cpu.clock and cpu.temp both have the tag 1"},
		{"same tag other order", "tags: {cpu.clock: 1, cpu.temp: 1}\n", "tags: cpu.clock and cpu.temp both have the tag 1"},
		{"interval number", "interval: 1\ntags: {cpu.temp: 1}\n", `interval: "1" is not a duration such as 1s or 500ms`},
		{"interval word", "interval: soon\ntags: {cpu.temp: 1}\n", `interval: "soon" is not a duration such as 1s or 500ms`},
		{"interval list", "interval: [1s]\ntags: {cpu.temp: 1}\n", `interval: "" is not a duration such as 1s or 500ms`},
		{"interval short", "interval: 99ms\ntags: {cpu.temp: 1}\n", "interval: 99ms is shorter than 100ms"},
		{"interval negative", "interval: -1s\ntags: {cpu.temp: 1}\n", "interval: -1s is shorter than 100ms"},
		{"commander interval below interval", "interval: 2s\ncommander_interval: 1s\ntags: {cpu.temp: 1}\n", "commander_interval: 1s is shorter than the interval 2s"},
		{"commander interval below default interval", "commander_interval: 500ms\ntags: {cpu.temp: 1}\n", "commander_interval: 500ms is shorter than the interval 1s"},
		{"commander interval word", "commander_interval: soon\ntags: {cpu.temp: 1}\n", `commander_interval: "soon" is not a duration such as 1m or 30s`},
		{"commander interval mapping", "commander_interval: {a: 1}\ntags: {cpu.temp: 1}\n", `commander_interval: "" is not a duration such as 1m or 30s`},
		{"commander interval negative", "commander_interval: -1m\ntags: {cpu.temp: 1}\n", "commander_interval: -1m is shorter than the interval 1s"},
		{"unquoted colour", "commander_color: #00c8ff\ntags: {cpu.temp: 1}\n", colourEmpty},
		{"empty colour", "commander_color: \"\"\ntags: {cpu.temp: 1}\n", colourEmpty},
		{"null colour", "commander_color: null\ntags: {cpu.temp: 1}\n", colourEmpty},
		{"short colour", "commander_color: \"#00c8f\"\ntags: {cpu.temp: 1}\n", `commander_color: "#00c8f" is not a colour such as "#00c8ff"`},
		{"colour without hash", "commander_color: \"00c8ff\"\ntags: {cpu.temp: 1}\n", `commander_color: "00c8ff" is not a colour such as "#00c8ff"`},
		{"colour not hexadecimal", "commander_color: \"#00c8fg\"\ntags: {cpu.temp: 1}\n", `commander_color: "#00c8fg" is not a colour such as "#00c8ff"`},
		{"colour mapping", "commander_color: {a: 1}\ntags: {cpu.temp: 1}\n", `commander_color: "" is not a colour such as "#00c8ff"`},
		{"url number", "librehardwaremonitor_url: 8085\ntags: {cpu.temp: 1}\n", monitorURLProblem("8085")},
		{"url mapping", "librehardwaremonitor_url: {a: 1}\ntags: {cpu.temp: 1}\n", monitorURLProblem("")},
		{"url null", "librehardwaremonitor_url:\ntags: {cpu.temp: 1}\n", monitorURLProblem("")},
		{"url not parsable", "librehardwaremonitor_url: \"http://[::1\"\ntags: {cpu.temp: 1}\n", monitorURLProblem("http://[::1")},
		{"url relative", "librehardwaremonitor_url: /data.json\ntags: {cpu.temp: 1}\n", monitorURLProblem("/data.json")},
		{"url without scheme", "librehardwaremonitor_url: localhost:8085/data.json\ntags: {cpu.temp: 1}\n", monitorURLProblem("localhost:8085/data.json")},
		{"url ftp", "librehardwaremonitor_url: ftp://host/\ntags: {cpu.temp: 1}\n", monitorURLProblem("ftp://host/")},
		{"url without host", "librehardwaremonitor_url: http:///data.json\ntags: {cpu.temp: 1}\n", monitorURLProblem("http:///data.json")},
		{"url with a port only", "librehardwaremonitor_url: http://:8085/data.json\ntags: {cpu.temp: 1}\n", monitorURLProblem("http://:8085/data.json")},
		{"colour and commander interval", "commander_color: \"#00c8ff\"\ncommander_interval: 1m\ntags: {cpu.temp: 1}\n", "commander_interval: cannot be used with commander_color, which reads the controller in every round"},
	}
	for _, each := range cases {
		_, err := Parse([]byte(each.content))
		if err == nil || err.Error() != each.want {
			test.Errorf("%s: error %v, want %q", each.name, err, each.want)
		}
	}
}

func TestParseRepeatedKeys(test *testing.T) {
	for name, content := range map[string]string{
		"tags twice":   "tags: {cpu.temp: 1}\ntags: {cpu.clock: 2}\n",
		"metric twice": "tags:\n  cpu.temp: 1\n  cpu.temp: 2\n",
	} {
		_, err := Parse([]byte(content))
		if err == nil || !strings.Contains(err.Error(), "already defined") {
			test.Errorf("%s: error %v", name, err)
		}
	}
}

func TestParseSyntaxError(test *testing.T) {
	_, err := Parse([]byte("tags: {cpu.temp: 1\n  - x: ["))
	if err == nil || !strings.HasPrefix(err.Error(), "yaml: ") {
		test.Errorf("error %v", err)
	}
}

func TestParseCommanderInterval(test *testing.T) {
	settings, err := Parse([]byte("interval: 1s\ncommander_interval: 30s\ntags: {cpu.temp: 1}\n"))
	if err != nil || settings.CommanderInterval != 30*time.Second || settings.CommanderPeriod() != 30*time.Second {
		test.Errorf("got %+v, %v", settings, err)
	}
}

func TestCommanderPeriodDefault(test *testing.T) {
	cases := []struct {
		content string
		want    time.Duration
	}{
		{"tags: {cpu.temp: 1}\n", time.Minute},
		{"interval: 2m\ntags: {cpu.temp: 1}\n", 2 * time.Minute},
	}
	for _, each := range cases {
		settings, err := Parse([]byte(each.content))
		if err != nil || settings.CommanderInterval != 0 || settings.CommanderPeriod() != each.want {
			test.Errorf("%q: got %+v, %v, want period %v", each.content, settings, err, each.want)
		}
	}
}

func TestParseCommanderIntervalEqualToInterval(test *testing.T) {
	settings, err := Parse([]byte("interval: 500ms\ncommander_interval: 500ms\ntags: {cpu.temp: 1}\n"))
	if err != nil || settings.CommanderPeriod() != 500*time.Millisecond {
		test.Errorf("got %+v, %v", settings, err)
	}
}

func TestParseColour(test *testing.T) {
	for _, spelling := range []string{"#00c8ff", "#00C8FF"} {
		got, err := Parse([]byte("commander_color: \"" + spelling + "\"\ntags: {cpu.temp: 1}\n"))
		want := &Colour{Red: 0, Green: 200, Blue: 255}
		if err != nil || !reflect.DeepEqual(got.CommanderColour, want) {
			test.Errorf("%s: %+v, %v", spelling, got.CommanderColour, err)
		}
	}
}

func TestParseWithoutColour(test *testing.T) {
	got, err := Parse([]byte("tags: {cpu.temp: 1}\n"))
	if err != nil || got.CommanderColour != nil {
		test.Errorf("%+v, %v", got.CommanderColour, err)
	}
}

func TestDecodeColour(test *testing.T) {
	for _, text := range []string{"#00c8ff", "#00C8FF"} {
		got, ok := DecodeColour(text)
		if !ok || got != (Colour{Red: 0, Green: 200, Blue: 255}) {
			test.Errorf("%s: got %+v, %v", text, got, ok)
		}
	}
	for _, text := range []string{"00c8ff", "#00c8f", "#00c8fff", "#00c8fg", ""} {
		if _, ok := DecodeColour(text); ok {
			test.Errorf("%q: want false", text)
		}
	}
}

func TestParseThemeAlone(test *testing.T) {
	got, err := Parse([]byte("interval: 2s\ntheme: ./my theme\n"))
	want := Config{Interval: 2 * time.Second, Theme: "./my theme", LibreHardwareMonitorURL: DefaultMonitorURL()}
	if err != nil || !reflect.DeepEqual(got, want) {
		test.Errorf("got %+v, %v", got, err)
	}
}

func TestParseWithoutMonitorURLGivesTheDefault(test *testing.T) {
	got, err := Parse([]byte("tags: {cpu.temp: 1}\n"))
	if err != nil || got.LibreHardwareMonitorURL.String() != "http://localhost:8085/data.json" {
		test.Errorf("got %v, %v", got.LibreHardwareMonitorURL, err)
	}
	if DefaultMonitorURL().String() != DefaultLibreHardwareMonitorURL {
		test.Errorf("the default URL is %s, the constant %s", DefaultMonitorURL(), DefaultLibreHardwareMonitorURL)
	}
	if DefaultMonitorURL() == DefaultMonitorURL() {
		test.Error("the default URL is shared between callers")
	}
}

func TestParseMonitorURLs(test *testing.T) {
	for _, text := range []string{
		"https://host/data.json",
		"http://user:secret@192.168.1.5:9000/data.json?a=b",
		"http://host",
		"HTTP://Host:8085/x",
	} {
		got, err := Parse([]byte("librehardwaremonitor_url: \"" + text + "\"\ntags: {cpu.temp: 1}\n"))
		if err != nil || got.LibreHardwareMonitorURL.Host == "" {
			test.Errorf("%s: %v, %v", text, got.LibreHardwareMonitorURL, err)
		}
	}
	got, _ := Parse([]byte("librehardwaremonitor_url: http://user:secret@host:9000/data.json?a=b\ntags: {cpu.temp: 1}\n"))
	address := got.LibreHardwareMonitorURL
	password, _ := address.User.Password()
	if address.User.Username() != "user" || password != "secret" || address.Port() != "9000" || address.Path != "/data.json" || address.RawQuery != "a=b" {
		test.Errorf("got %v", address)
	}
}
