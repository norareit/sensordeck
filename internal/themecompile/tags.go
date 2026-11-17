package themecompile

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/norareit/sensordeck/internal/config"
)

// clockTag is the tag that the device fills with the clock, so that no metric gets it.
const clockTag = 21

// tagEntry is what takes tags: the plain entry of a metric (one tag), or one alarm pair of a metric (two tags).
type tagEntry struct {
	name      string
	pair      bool
	threshold config.Threshold // of a pair
}

// compareBound orders an optional bound: none before any number, and the numbers ascending.
func compareBound(hasFirst bool, first int16, hasSecond bool, second int16) int {
	if hasFirst != hasSecond {
		if hasFirst {
			return 1
		}
		return -1
	}
	return cmp.Compare(first, second)
}

func compareEntries(first, second tagEntry) int {
	if byName := strings.Compare(first.name, second.name); byName != 0 {
		return byName
	}
	if first.pair != second.pair {
		if first.pair {
			return 1
		}
		return -1
	}
	if byAbove := compareBound(first.threshold.HasAbove, first.threshold.Above, second.threshold.HasAbove, second.threshold.Above); byAbove != 0 {
		return byAbove
	}
	return compareBound(first.threshold.HasBelow, first.threshold.Below, second.threshold.HasBelow, second.threshold.Below)
}

// entriesOf gives the distinct entries of the widgets that take tags: a plain entry for each metric of a number
// without alarm and of a bar, and a pair for each distinct metric and threshold of the alarm numbers.
func entriesOf(widgets []widgetSpec) []tagEntry {
	var entries []tagEntry
	for _, spec := range widgets {
		switch {
		case spec.kind == "number" && spec.alarm != nil:
			entries = append(entries, tagEntry{name: spec.metric, pair: true, threshold: spec.alarm.threshold})
		case spec.kind == "number" || spec.kind == "bar":
			entries = append(entries, tagEntry{name: spec.metric})
		}
	}
	slices.SortFunc(entries, compareEntries)
	return slices.CompactFunc(entries, func(first, second tagEntry) bool { return compareEntries(first, second) == 0 })
}

// assignTags gives the entries of the widgets, sorted by their metric name's bytes, the plain entry first and the pairs
// by their thresholds, the tags 1, 2, 3 and up, skipping the tag of the clock. Sorting makes the tags independent of
// the order of the widgets.
func assignTags(widgets []widgetSpec) ([]config.Mapping, []config.AlarmPair) {
	var mappings []config.Mapping
	var alarms []config.AlarmPair
	next := uint8(1)
	take := func() uint8 {
		if next == clockTag {
			next++
		}
		next++
		return next - 1
	}
	for _, entry := range entriesOf(widgets) {
		if !entry.pair {
			mappings = append(mappings, config.Mapping{Name: entry.name, Tag: take()})
			continue
		}
		normal := take()
		alarms = append(alarms, config.AlarmPair{Name: entry.name, Threshold: entry.threshold, Normal: normal, Alarm: take()})
	}
	return mappings, alarms
}

// plainTagOf gives the tag of the plain entry of a metric, and 0 for a widget with no metric.
func plainTagOf(mappings []config.Mapping, metric string) byte {
	for _, mapping := range mappings {
		if mapping.Name == metric {
			return mapping.Tag
		}
	}
	return 0
}

// pairOf gives the pair of an alarm number.
func pairOf(alarms []config.AlarmPair, metric string, threshold config.Threshold) config.AlarmPair {
	for _, pair := range alarms {
		if pair.Name == metric && pair.Threshold == threshold {
			return pair
		}
	}
	return config.AlarmPair{}
}

// TagsFile gives the content of the tags file: a block in the shape of the tags section of the config file of run. The
// metrics are in the order of their lowest tag.
func TagsFile(tags []config.Mapping, alarms []config.AlarmPair) []byte {
	var text strings.Builder
	text.WriteString("# Paste this block into the config file of sensordeck run.\n")
	if len(tags) == 0 && len(alarms) == 0 {
		text.WriteString("tags: {}\n")
		return []byte(text.String())
	}
	text.WriteString("tags:\n")
	for _, name := range metricsByLowestTag(tags, alarms) {
		text.WriteString("  " + name + ": " + metricValue(name, tags, alarms) + "\n")
	}
	return []byte(text.String())
}

func metricsByLowestTag(tags []config.Mapping, alarms []config.AlarmPair) []string {
	lowest := make(map[string]uint8)
	note := func(name string, tag uint8) {
		if known, found := lowest[name]; !found || tag < known {
			lowest[name] = tag
		}
	}
	for _, mapping := range tags {
		note(mapping.Name, mapping.Tag)
	}
	for _, pair := range alarms {
		note(pair.Name, min(pair.Normal, pair.Alarm))
	}
	names := make([]string, 0, len(lowest))
	for name := range lowest {
		names = append(names, name)
	}
	slices.SortFunc(names, func(first, second string) int { return cmp.Compare(lowest[first], lowest[second]) })
	return names
}

// metricValue is the value of a metric in the block: its tag, when it has no pair, or a mapping in the flow style.
func metricValue(name string, tags []config.Mapping, alarms []config.AlarmPair) string {
	var parts []string
	var pairs []string
	for _, mapping := range tags {
		if mapping.Name == name {
			parts = append(parts, "tag: "+strconv.Itoa(int(mapping.Tag)))
		}
	}
	for _, pair := range alarms {
		if pair.Name == name {
			pairs = append(pairs, pairText(pair))
		}
	}
	if len(pairs) == 0 {
		return strings.TrimPrefix(parts[0], "tag: ")
	}
	parts = append(parts, "alarms: ["+strings.Join(pairs, ", ")+"]")
	return "{" + strings.Join(parts, ", ") + "}"
}

func pairText(pair config.AlarmPair) string {
	var parts []string
	if pair.Threshold.HasAbove {
		parts = append(parts, "above: "+strconv.Itoa(int(pair.Threshold.Above)))
	}
	if pair.Threshold.HasBelow {
		parts = append(parts, "below: "+strconv.Itoa(int(pair.Threshold.Below)))
	}
	parts = append(parts, "normal: "+strconv.Itoa(int(pair.Normal)), "alarm: "+strconv.Itoa(int(pair.Alarm)))
	return "{" + strings.Join(parts, ", ") + "}"
}
