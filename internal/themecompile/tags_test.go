package themecompile

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/config"
)

func numberWidgets(metrics ...string) []widgetSpec {
	widgets := make([]widgetSpec, len(metrics))
	for position, metric := range metrics {
		widgets[position] = widgetSpec{kind: "number", metric: metric}
	}
	return widgets
}

func alarmWidgetOf(metric string, threshold config.Threshold) widgetSpec {
	return widgetSpec{kind: "number", metric: metric, alarm: &alarmSpec{threshold: threshold}}
}

func above(value int16) config.Threshold { return config.Threshold{Above: value, HasAbove: true} }
func below(value int16) config.Threshold { return config.Threshold{Below: value, HasBelow: true} }

func TestTagsAreSortedAndShared(test *testing.T) {
	got, _ := assignTags(numberWidgets("gpu.temp", "cpu.temp", "cpu.temp"))
	want := []config.Mapping{{Name: "cpu.temp", Tag: 1}, {Name: "gpu.temp", Tag: 2}}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %+v", got)
	}
}

func TestTagsSkipTheClock(test *testing.T) {
	var names []string
	for number := 1; number <= 22; number++ {
		names = append(names, fmt.Sprintf("nvme.%d.temp", number))
	}
	got, _ := assignTags(numberWidgets(names...))
	// Sorted by bytes: 1, 10 to 19, 2, 20 to 22, 3 to 9.
	if len(got) != 22 {
		test.Fatalf("%d tags", len(got))
	}
	for position, mapping := range got {
		want := uint8(position + 1)
		if position >= 20 {
			want++
		}
		if mapping.Tag != want || mapping.Tag == 0 || mapping.Tag == 21 {
			test.Errorf("%s: tag %d, want %d", mapping.Name, mapping.Tag, want)
		}
	}
	if got[0].Name != "nvme.1.temp" || got[1].Name != "nvme.10.temp" {
		test.Errorf("not in byte order: %s, %s", got[0].Name, got[1].Name)
	}
}

func TestTagsFile(test *testing.T) {
	tags := []config.Mapping{{Name: "cpu.temp", Tag: 1}, {Name: "gpu.temp", Tag: 2}}
	text := TagsFile(tags, nil)
	want := "# Paste this block into the config file of sensordeck run.\ntags:\n  cpu.temp: 1\n  gpu.temp: 2\n"
	if string(text) != want {
		test.Errorf("got %q", text)
	}
	parsed, err := config.Parse(text)
	if err != nil || !reflect.DeepEqual(parsed.Mappings, tags) {
		test.Errorf("config read back %+v, %v", parsed.Mappings, err)
	}
	if got := string(TagsFile(nil, nil)); got != "# Paste this block into the config file of sensordeck run.\ntags: {}\n" {
		test.Errorf("no tags: %q", got)
	}
}

func TestABarGetsTheTagOfItsMetric(test *testing.T) {
	shared := compileFiles(test, solidTheme+"widgets:\n"+
		"  - {type: bar, metric: cpu.util, x: 0, y: 0, width: 5, height: 5, color: \"#00ff00\"}\n"+
		"  - {type: number, font: body, metric: cpu.util, x: 0, y: 10}\n", nil)
	if want := []config.Mapping{{Name: "cpu.util", Tag: 1}}; !reflect.DeepEqual(shared.Tags, want) {
		test.Errorf("tags %+v", shared.Tags)
	}
	if shared.File[2*64+2] != 1 || shared.File[3*64+2] != 1 {
		test.Errorf("tags of the records %d, %d", shared.File[2*64+2], shared.File[3*64+2])
	}
	alone := compileFiles(test, solidTheme+"widgets:\n  - {type: bar, metric: gpu.util, x: 0, y: 0, width: 5, height: 5, color: \"#00ff00\"}\n", nil)
	if want := []config.Mapping{{Name: "gpu.util", Tag: 1}}; !reflect.DeepEqual(alone.Tags, want) || alone.File[2*64+2] != 1 {
		test.Errorf("tags %+v, record tag %d", alone.Tags, alone.File[2*64+2])
	}
	if !strings.Contains(string(TagsFile(alone.Tags, alone.Alarms)), "  gpu.util: 1\n") {
		test.Errorf("tags file %q", TagsFile(alone.Tags, alone.Alarms))
	}
}

func TestAlarmPairsTakeTwoTagsAfterThePlainEntry(test *testing.T) {
	widgets := []widgetSpec{
		alarmWidgetOf("gpu.temp", above(90)), {kind: "bar", metric: "gpu.temp"},
		alarmWidgetOf("gpu.temp", above(80)), {kind: "number", metric: "cpu.temp"},
	}
	mappings, alarms := assignTags(widgets)
	wantMappings := []config.Mapping{{Name: "cpu.temp", Tag: 1}, {Name: "gpu.temp", Tag: 2}}
	wantAlarms := []config.AlarmPair{{Name: "gpu.temp", Threshold: above(80), Normal: 3, Alarm: 4},
		{Name: "gpu.temp", Threshold: above(90), Normal: 5, Alarm: 6}}
	if !reflect.DeepEqual(mappings, wantMappings) || !reflect.DeepEqual(alarms, wantAlarms) {
		test.Errorf("got %+v and %+v", mappings, alarms)
	}
}

func TestPairsAreOrderedByAboveThenBelowNoneFirst(test *testing.T) {
	both := config.Threshold{Above: 5, Below: 1, HasAbove: true, HasBelow: true}
	otherBoth := config.Threshold{Above: 5, Below: -3, HasAbove: true, HasBelow: true}
	_, alarms := assignTags([]widgetSpec{
		alarmWidgetOf("a.b", above(-5)), alarmWidgetOf("a.b", both), alarmWidgetOf("a.b", below(7)),
		alarmWidgetOf("a.b", otherBoth), alarmWidgetOf("a.b", above(5)),
	})
	want := []config.Threshold{below(7), above(-5), above(5), otherBoth, both}
	for position, pair := range alarms {
		if pair.Threshold != want[position] || pair.Normal != uint8(1+2*position) {
			test.Errorf("pair %d: %+v, want threshold %+v", position, pair, want[position])
		}
	}
	if len(alarms) != len(want) {
		test.Errorf("%d pairs", len(alarms))
	}
}

func TestAlarmNumbersOfOneMetricAndThresholdShareAPair(test *testing.T) {
	result := compileFiles(test, solidTheme+"widgets:\n"+
		"  - {type: number, font: body, metric: gpu.temp, x: 0, y: 0, alarm: {above: 80, color: \"#ff0000\"}}\n"+
		"  - {type: number, font: body, metric: gpu.temp, x: 0, y: 40, digits: 5, alarm: {above: 80, color: \"#00ff00\"}}\n", nil)
	if len(result.Alarms) != 1 || len(result.Tags) != 0 {
		test.Fatalf("tags %+v, pairs %+v", result.Tags, result.Alarms)
	}
	for _, slot := range []int{2, 3, 4, 5} {
		wantTag := result.Alarms[0].Normal
		if slot%2 == 1 {
			wantTag = result.Alarms[0].Alarm
		}
		if got := result.File[slot*64+2]; got != wantTag {
			test.Errorf("slot %d has the tag %d, want %d", slot, got, wantTag)
		}
	}
}

func TestAPairStraddlesTheClockTag(test *testing.T) {
	var widgets []widgetSpec
	for number := 1; number <= 19; number++ {
		widgets = append(widgets, widgetSpec{kind: "number", metric: fmt.Sprintf("a.m%02d", number)})
	}
	widgets = append(widgets, alarmWidgetOf("b.x", above(1)))
	_, alarms := assignTags(widgets)
	if len(alarms) != 1 || alarms[0].Normal != 20 || alarms[0].Alarm != 22 {
		test.Errorf("pairs %+v, want the tags 20 and 22", alarms)
	}
}

func TestTagsDoNotDependOnTheOrderOfTheWidgets(test *testing.T) {
	widgets := []widgetSpec{alarmWidgetOf("gpu.temp", above(80)), {kind: "bar", metric: "cpu.util"},
		alarmWidgetOf("gpu.temp", below(5)), {kind: "number", metric: "gpu.temp"}}
	mappings, alarms := assignTags(widgets)
	reversed := []widgetSpec{widgets[3], widgets[2], widgets[1], widgets[0]}
	reversedMappings, reversedAlarms := assignTags(reversed)
	if !reflect.DeepEqual(mappings, reversedMappings) || !reflect.DeepEqual(alarms, reversedAlarms) {
		test.Errorf("%+v %+v and %+v %+v", mappings, alarms, reversedMappings, reversedAlarms)
	}
}

func TestTagsFileWithPairs(test *testing.T) {
	both := config.Threshold{Above: 80, Below: 20, HasAbove: true, HasBelow: true}
	cases := []struct {
		name   string
		tags   []config.Mapping
		alarms []config.AlarmPair
		want   string
	}{
		{"the example", []config.Mapping{{Name: "cpu.temp", Tag: 1}, {Name: "gpu.temp", Tag: 2}},
			[]config.AlarmPair{{Name: "gpu.temp", Threshold: above(80), Normal: 3, Alarm: 4}},
			"  cpu.temp: 1\n  gpu.temp: {tag: 2, alarms: [{above: 80, normal: 3, alarm: 4}]}\n"},
		{"only pairs", nil, []config.AlarmPair{{Name: "gpu.temp", Threshold: above(80), Normal: 1, Alarm: 2},
			{Name: "gpu.temp", Threshold: both, Normal: 3, Alarm: 4}},
			"  gpu.temp: {alarms: [{above: 80, normal: 1, alarm: 2}, {above: 80, below: 20, normal: 3, alarm: 4}]}\n"},
		{"only below", nil, []config.AlarmPair{{Name: "gpu.util", Threshold: below(1000), Normal: 1, Alarm: 2}},
			"  gpu.util: {alarms: [{below: 1000, normal: 1, alarm: 2}]}\n"},
		{"no pairs", []config.Mapping{{Name: "cpu.temp", Tag: 1}}, nil, "  cpu.temp: 1\n"},
		{"by the lowest tag", []config.Mapping{{Name: "gpu.temp", Tag: 1}}, []config.AlarmPair{{Name: "cpu.temp", Threshold: below(1), Normal: 2, Alarm: 3}},
			"  gpu.temp: 1\n  cpu.temp: {alarms: [{below: 1, normal: 2, alarm: 3}]}\n"},
	}
	for _, each := range cases {
		want := "# Paste this block into the config file of sensordeck run.\ntags:\n" + each.want
		got := TagsFile(each.tags, each.alarms)
		if string(got) != want {
			test.Errorf("%s: got %q", each.name, got)
		}
		parsed, err := config.Parse(got)
		if err != nil || !reflect.DeepEqual(parsed.Mappings, each.tags) || !reflect.DeepEqual(parsed.Alarms, each.alarms) {
			test.Errorf("%s: read back %+v and %+v, %v", each.name, parsed.Mappings, parsed.Alarms, err)
		}
	}
}

func TestTheTagsFileOfACompiledThemeReadsBackToItsTags(test *testing.T) {
	result := compileFiles(test, solidTheme+"widgets:\n"+
		"  - {type: number, font: body, metric: gpu.temp, x: 0, y: 0}\n"+
		"  - {type: number, font: body, metric: gpu.temp, x: 0, y: 40, alarm: {above: 80, color: \"#ff0000\"}}\n"+
		"  - {type: number, font: body, metric: gpu.temp, x: 0, y: 80, alarm: {below: 20, color: \"#ff0000\"}}\n", nil)
	parsed, err := config.Parse(TagsFile(result.Tags, result.Alarms))
	if err != nil || !reflect.DeepEqual(parsed.Mappings, result.Tags) || !reflect.DeepEqual(parsed.Alarms, result.Alarms) || len(result.Alarms) != 2 {
		test.Errorf("read back %+v and %+v of %+v and %+v, %v", parsed.Mappings, parsed.Alarms, result.Tags, result.Alarms, err)
	}
}
