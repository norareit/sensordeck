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
	want := Config{Interval: time.Second, Mappings: []Mapping{{"cpu.temp", 1}, {"cpu.clock", 2}}}
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

func TestParseErrors(test *testing.T) {
	cases := []struct{ name, content, want string }{
		{"empty", "", "want a mapping with the keys interval and tags"},
		{"comment", "# nothing\n", "want a mapping with the keys interval and tags"},
		{"list", "- tags\n", "want a mapping with the keys interval and tags"},
		{"unknown key", "colour: red\ntags: {cpu.temp: 1}\n", `unknown key "colour", the keys are interval and tags`},
		{"no tags", "interval: 1s\n", "tags: want a mapping from metric name to tag, with at least one entry"},
		{"tags without value", "tags:\n", "tags: want a mapping from metric name to tag, with at least one entry"},
		{"empty tags", "tags: {}\n", "tags: want a mapping from metric name to tag, with at least one entry"},
		{"list tags", "tags: [1]\n", "tags: want a mapping from metric name to tag, with at least one entry"},
		{"not a metric", "tags: {cpu.tmp: 1}\n", `tags: "cpu.tmp" is not a metric name`},
		{"float tag", "tags: {cpu.temp: 1.0}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"string tag", "tags: {cpu.temp: \"1\"}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"word tag", "tags: {cpu.temp: x}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"mapping tag", "tags: {cpu.temp: {a: 1}}\n", "tags: cpu.temp: the tag must be a whole number from 0 to 255"},
		{"big tag", "tags: {cpu.temp: 256}\n", "tags: cpu.temp: the tag 256 is outside 0 to 255"},
		{"negative tag", "tags: {cpu.temp: -1}\n", "tags: cpu.temp: the tag -1 is outside 0 to 255"},
		{"same tag", "tags: {cpu.temp: 1, cpu.clock: 1}\n", "tags: cpu.clock and cpu.temp both have the tag 1"},
		{"same tag other order", "tags: {cpu.clock: 1, cpu.temp: 1}\n", "tags: cpu.clock and cpu.temp both have the tag 1"},
		{"interval number", "interval: 1\ntags: {cpu.temp: 1}\n", `interval: "1" is not a duration such as 1s or 500ms`},
		{"interval word", "interval: soon\ntags: {cpu.temp: 1}\n", `interval: "soon" is not a duration such as 1s or 500ms`},
		{"interval list", "interval: [1s]\ntags: {cpu.temp: 1}\n", `interval: "" is not a duration such as 1s or 500ms`},
		{"interval short", "interval: 99ms\ntags: {cpu.temp: 1}\n", "interval: 99ms is shorter than 100ms"},
		{"interval negative", "interval: -1s\ntags: {cpu.temp: 1}\n", "interval: -1s is shorter than 100ms"},
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
