package config

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// HidingValue is the value an alarm number is sent in the state in which it must show nothing. The screen draws every
// glyph of a negative value from the left, and the glyph of the minus of an alarm number is transparent and as wide as
// the box, so that the digits fall outside it.
const HidingValue int16 = -1

// Threshold says when a value is in the alarm state: greater than Above when HasAbove, or less than Below when
// HasBelow.
type Threshold struct {
	Above, Below       int16
	HasAbove, HasBelow bool
}

// Alarm tells whether the value, as it is sent to the screen, is in the alarm state.
func (threshold Threshold) Alarm(value int16) bool {
	return threshold.HasAbove && value > threshold.Above || threshold.HasBelow && value < threshold.Below
}

// AlarmPair sends the value of a metric to the Normal tag in the normal state and to the Alarm tag in the alarm state,
// and the hiding value to the other of the two.
type AlarmPair struct {
	Name      string
	Threshold Threshold
	Normal    uint8
	Alarm     uint8
}

// MetricNames gives the distinct names of the metrics of the plain entries and of the pairs, sorted.
func (config Config) MetricNames() []string {
	names := make([]string, 0, len(config.Mappings)+len(config.Alarms))
	for _, mapping := range config.Mappings {
		names = append(names, mapping.Name)
	}
	for _, pair := range config.Alarms {
		names = append(names, pair.Name)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// taggedMetric is a tag of the block with the metric it belongs to, to check that no two are the same.
type taggedMetric struct {
	name string
	tag  uint8
}

// parseMetricEntry reads the value of a metric of tags: a tag, or a mapping with tag and alarms. It adds what it reads
// to the plain entries and the pairs, and to every tag of the block.
func parseMetricEntry(name string, node *yaml.Node, block *tagsBlock) error {
	if node.Kind != yaml.MappingNode {
		tag, err := parseTag(name, node)
		if err != nil {
			return err
		}
		block.addPlain(name, tag)
		return nil
	}
	if err := refuseRepeatedKeys(node); err != nil {
		return fmt.Errorf("tags: %s: %w", name, err)
	}
	var tagNode, alarmsNode *yaml.Node
	for position := 0; position+1 < len(node.Content); position += 2 {
		switch key := node.Content[position].Value; key {
		case "tag":
			tagNode = node.Content[position+1]
		case "alarms":
			alarmsNode = node.Content[position+1]
		default:
			return fmt.Errorf("tags: %s: unknown key %q, the keys are alarms and tag", name, key)
		}
	}
	if tagNode == nil && alarmsNode == nil {
		return fmt.Errorf("tags: %s: want tag, alarms or both", name)
	}
	if tagNode != nil {
		tag, err := parseTag(name+": tag", tagNode)
		if err != nil {
			return err
		}
		block.addPlain(name, tag)
	}
	if alarmsNode == nil {
		return nil
	}
	return parseAlarmPairs(name, alarmsNode, block)
}

func parseAlarmPairs(name string, node *yaml.Node, block *tagsBlock) error {
	if node.Kind != yaml.SequenceNode || len(node.Content) == 0 {
		return fmt.Errorf("tags: %s: alarms: want a non-empty list of alarm pairs", name)
	}
	for position, item := range node.Content {
		place := fmt.Sprintf("%s: alarms: %d", name, position+1)
		pair, err := parseAlarmPair(name, place, item)
		if err != nil {
			return err
		}
		block.addPair(pair)
	}
	return nil
}

var pairKeys = []string{"above", "alarm", "below", "normal"}

func parseAlarmPair(name, place string, node *yaml.Node) (AlarmPair, error) {
	if node.Kind != yaml.MappingNode {
		return AlarmPair{}, fmt.Errorf("tags: %s: want a mapping with normal, alarm, above and below", place)
	}
	if err := refuseRepeatedKeys(node); err != nil {
		return AlarmPair{}, fmt.Errorf("tags: %s: %w", place, err)
	}
	values := make(map[string]*yaml.Node, len(pairKeys))
	for position := 0; position+1 < len(node.Content); position += 2 {
		key := node.Content[position].Value
		if !slices.Contains(pairKeys, key) {
			return AlarmPair{}, fmt.Errorf("tags: %s: unknown key %q, the keys are above, alarm, below and normal", place, key)
		}
		values[key] = node.Content[position+1]
	}
	pair := AlarmPair{Name: name}
	for _, tag := range []struct {
		key   string
		value *uint8
	}{{"normal", &pair.Normal}, {"alarm", &pair.Alarm}} {
		tagNode, found := values[tag.key]
		if !found {
			return AlarmPair{}, fmt.Errorf("tags: %s: the key %s is missing", place, tag.key)
		}
		var err error
		if *tag.value, err = parseTag(place+": "+tag.key, tagNode); err != nil {
			return AlarmPair{}, err
		}
	}
	var err error
	if pair.Threshold.Above, pair.Threshold.HasAbove, err = parseBound(place, "above", values["above"]); err != nil {
		return AlarmPair{}, err
	}
	if pair.Threshold.Below, pair.Threshold.HasBelow, err = parseBound(place, "below", values["below"]); err != nil {
		return AlarmPair{}, err
	}
	if !pair.Threshold.HasAbove && !pair.Threshold.HasBelow {
		return AlarmPair{}, fmt.Errorf("tags: %s: want above, below or both", place)
	}
	return pair, nil
}

// parseBound reads a threshold, and gives false when the key is not given.
func parseBound(place, key string, node *yaml.Node) (int16, bool, error) {
	if node == nil {
		return 0, false, nil
	}
	var number int64
	if node.Kind != yaml.ScalarNode || node.ShortTag() != "!!int" || node.Decode(&number) != nil ||
		number < math.MinInt16 || number > math.MaxInt16 {
		return 0, false, fmt.Errorf("tags: %s: %s: want a whole number from -32768 to 32767", place, key)
	}
	return int16(number), true, nil
}

// tagsBlock gathers what the key tags holds while it is read.
type tagsBlock struct {
	mappings []Mapping
	alarms   []AlarmPair
	tagged   []taggedMetric
}

func (block *tagsBlock) addPlain(name string, tag uint8) {
	block.mappings = append(block.mappings, Mapping{Name: name, Tag: tag})
	block.tagged = append(block.tagged, taggedMetric{name, tag})
}

func (block *tagsBlock) addPair(pair AlarmPair) {
	block.alarms = append(block.alarms, pair)
	block.tagged = append(block.tagged, taggedMetric{pair.Name, pair.Normal}, taggedMetric{pair.Name, pair.Alarm})
}

// finish orders what the block holds and refuses a tag that two entries have.
func (block *tagsBlock) finish() error {
	slices.SortFunc(block.tagged, func(first, second taggedMetric) int {
		if first.tag != second.tag {
			return int(first.tag) - int(second.tag)
		}
		return strings.Compare(first.name, second.name)
	})
	for position := 1; position < len(block.tagged); position++ {
		if block.tagged[position-1].tag == block.tagged[position].tag {
			return fmt.Errorf("tags: %s and %s both have the tag %d",
				block.tagged[position-1].name, block.tagged[position].name, block.tagged[position].tag)
		}
	}
	slices.SortFunc(block.mappings, func(first, second Mapping) int { return int(first.Tag) - int(second.Tag) })
	slices.SortFunc(block.alarms, func(first, second AlarmPair) int { return int(first.Normal) - int(second.Normal) })
	return nil
}
