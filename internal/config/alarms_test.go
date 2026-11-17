package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestThresholdAlarm(t *testing.T) {
	above := Threshold{Above: 80, HasAbove: true}
	below := Threshold{Below: 1000, HasBelow: true}
	band := Threshold{Above: 80, Below: 20, HasAbove: true, HasBelow: true}
	for _, test := range []struct {
		name      string
		threshold Threshold
		value     int16
		want      bool
	}{
		{"above at the threshold", above, 80, false},
		{"above past the threshold", above, 81, true},
		{"below under the threshold", below, 999, true},
		{"below at the threshold", below, 1000, false},
		{"band under", band, 19, true},
		{"band at the lower edge", band, 20, false},
		{"band at the upper edge", band, 80, false},
		{"band over", band, 81, true},
	} {
		if got := test.threshold.Alarm(test.value); got != test.want {
			t.Errorf("%s: Alarm(%d) = %v, want %v", test.name, test.value, got, test.want)
		}
	}
}

func TestParseTagsWithPairs(t *testing.T) {
	got, err := Parse([]byte("tags:\n  cpu.temp: 1\n  gpu.temp: {tag: 2, alarms: [{above: 80, normal: 3, alarm: 4}]}\n"))
	if err != nil {
		t.Fatal(err)
	}
	wantMappings := []Mapping{{"cpu.temp", 1}, {"gpu.temp", 2}}
	wantAlarms := []AlarmPair{{"gpu.temp", Threshold{Above: 80, HasAbove: true}, 3, 4}}
	if !reflect.DeepEqual(got.Mappings, wantMappings) || !reflect.DeepEqual(got.Alarms, wantAlarms) {
		t.Errorf("got %v and %v, want %v and %v", got.Mappings, got.Alarms, wantMappings, wantAlarms)
	}
	if names := got.MetricNames(); !reflect.DeepEqual(names, []string{"cpu.temp", "gpu.temp"}) {
		t.Errorf("MetricNames = %v", names)
	}
}

func TestParseTagsOnlyPairsAndOnlyTag(t *testing.T) {
	got, err := Parse([]byte("tags:\n  gpu.temp: {alarms: [{below: -5, normal: 9, alarm: 7}, {above: 1, below: 0, normal: 3, alarm: 4}]}\n  cpu.temp: {tag: 2}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Mappings, []Mapping{{"cpu.temp", 2}}) {
		t.Errorf("Mappings = %v", got.Mappings)
	}
	wantAlarms := []AlarmPair{
		{"gpu.temp", Threshold{Above: 1, Below: 0, HasAbove: true, HasBelow: true}, 3, 4},
		{"gpu.temp", Threshold{Below: -5, HasBelow: true}, 9, 7},
	}
	if !reflect.DeepEqual(got.Alarms, wantAlarms) {
		t.Errorf("Alarms = %v, want %v", got.Alarms, wantAlarms)
	}
	if names := got.MetricNames(); !reflect.DeepEqual(names, []string{"cpu.temp", "gpu.temp"}) {
		t.Errorf("MetricNames = %v", names)
	}
}

func TestParseTagsPairErrors(t *testing.T) {
	for _, test := range []struct{ name, tags, want string }{
		{"two metrics", "{cpu.temp: 3, gpu.temp: {alarms: [{above: 1, normal: 3, alarm: 4}]}}", "tags: cpu.temp and gpu.temp both have the tag 3"},
		{"plain and pair", "{gpu.temp: {tag: 3, alarms: [{above: 1, normal: 3, alarm: 4}]}}", "tags: gpu.temp and gpu.temp both have the tag 3"},
		{"normal is alarm", "{gpu.temp: {alarms: [{above: 1, normal: 3, alarm: 3}]}}", "tags: gpu.temp and gpu.temp both have the tag 3"},
		{"empty mapping", "{gpu.temp: {}}", "tags: gpu.temp: want tag, alarms or both"},
		{"empty alarms", "{gpu.temp: {alarms: []}}", "tags: gpu.temp: alarms: want a non-empty list of alarm pairs"},
		{"alarms not a list", "{gpu.temp: {alarms: 3}}", "tags: gpu.temp: alarms: want a non-empty list of alarm pairs"},
		{"item not a mapping", "{gpu.temp: {alarms: [3]}}", "tags: gpu.temp: alarms: 1: want a mapping with normal, alarm, above and below"},
		{"no threshold", "{gpu.temp: {alarms: [{normal: 3, alarm: 4}]}}", "tags: gpu.temp: alarms: 1: want above, below or both"},
		{"no normal", "{gpu.temp: {alarms: [{above: 1, alarm: 4}]}}", "tags: gpu.temp: alarms: 1: the key normal is missing"},
		{"no alarm", "{gpu.temp: {alarms: [{above: 1, normal: 4}]}}", "tags: gpu.temp: alarms: 1: the key alarm is missing"},
		{"normal outside", "{gpu.temp: {alarms: [{above: 1, normal: 300, alarm: 4}]}}", "tags: gpu.temp: alarms: 1: normal: the tag 300 is outside 0 to 255"},
		{"normal word", "{gpu.temp: {alarms: [{above: 1, normal: 3, alarm: 4}, {above: 2, normal: x, alarm: 5}]}}", "tags: gpu.temp: alarms: 2: normal: the tag must be a whole number from 0 to 255"},
		{"plain tag word", "{gpu.temp: {tag: x}}", "tags: gpu.temp: tag: the tag must be a whole number from 0 to 255"},
		{"decimal above", "{gpu.temp: {alarms: [{above: 80.5, normal: 3, alarm: 4}]}}", "tags: gpu.temp: alarms: 1: above: want a whole number from -32768 to 32767"},
		{"string above", "{gpu.temp: {alarms: [{above: \"80\", normal: 3, alarm: 4}]}}", "tags: gpu.temp: alarms: 1: above: want a whole number from -32768 to 32767"},
		{"big above", "{gpu.temp: {alarms: [{above: 40000, normal: 3, alarm: 4}]}}", "tags: gpu.temp: alarms: 1: above: want a whole number from -32768 to 32767"},
		{"decimal below", "{gpu.temp: {alarms: [{below: 0.5, normal: 3, alarm: 4}]}}", "tags: gpu.temp: alarms: 1: below: want a whole number from -32768 to 32767"},
		{"unknown key", "{gpu.temp: {x: 1}}", `tags: gpu.temp: unknown key "x", the keys are alarms and tag`},
		{"unknown pair key", "{gpu.temp: {alarms: [{above: 1, normal: 3, alarm: 4, x: 1}]}}", `tags: gpu.temp: alarms: 1: unknown key "x", the keys are above, alarm, below and normal`},
		{"repeated key", "{gpu.temp: {tag: 1, tag: 2}}", "mapping key \"tag\" already defined"},
		{"repeated pair key", "{gpu.temp: {alarms: [{above: 1, above: 2, normal: 3, alarm: 4}]}}", "mapping key \"above\" already defined"},
	} {
		_, err := Parse([]byte("tags: " + test.tags + "\n"))
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: error %v, want %q", test.name, err, test.want)
		}
	}
}
