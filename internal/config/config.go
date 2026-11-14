// Package config reads and checks the config file of the command run. It is not limited to a platform.
package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/norareit/sensordeck/internal/sensors"
)

const (
	// DefaultInterval is the time between two rounds when the file does not say.
	DefaultInterval = time.Second
	// MinimumInterval is the shortest time between two rounds, so that a typo cannot flood the screen.
	MinimumInterval = 100 * time.Millisecond

	maxTag = 255
)

// Mapping sends the value of a metric to a tag of the theme on the screen.
type Mapping struct {
	Name string
	Tag  uint8
}

// Config is what the file holds.
type Config struct {
	Interval time.Duration
	Mappings []Mapping // ordered by tag, ascending
}

// Parse reads the content of a config file and checks it completely. Its errors are the texts that follow the prefix
// naming the file, which the caller adds. A file with several problems gives one of them.
func Parse(content []byte) (Config, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return Config{}, err
	}
	root, err := rootMapping(&document)
	if err != nil {
		return Config{}, err
	}
	var tagsNode, intervalNode *yaml.Node
	for position := 0; position+1 < len(root.Content); position += 2 {
		key, value := root.Content[position], root.Content[position+1]
		switch key.Value {
		case "tags":
			tagsNode = value
		case "interval":
			intervalNode = value
		default:
			return Config{}, fmt.Errorf("unknown key %q, the keys are interval and tags", key.Value)
		}
	}
	mappings, err := parseTags(tagsNode)
	if err != nil {
		return Config{}, err
	}
	interval, err := parseInterval(intervalNode)
	if err != nil {
		return Config{}, err
	}
	return Config{Interval: interval, Mappings: mappings}, nil
}

var errNotAMapping = errors.New("want a mapping with the keys interval and tags")

// rootMapping gives the mapping of the file, and refuses a key that is given twice. The parser reports that only when
// a mapping is decoded into a map.
func rootMapping(document *yaml.Node) (*yaml.Node, error) {
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return nil, errNotAMapping
	}
	root := document.Content[0]
	if err := refuseRepeatedKeys(root); err != nil {
		return nil, err
	}
	return root, nil
}

func refuseRepeatedKeys(mapping *yaml.Node) error {
	var ignored map[string]yaml.Node
	return mapping.Decode(&ignored)
}

const tagsProblem = "tags: want a mapping from metric name to tag, with at least one entry"

func parseTags(node *yaml.Node) ([]Mapping, error) {
	if node == nil || node.Kind != yaml.MappingNode || len(node.Content) == 0 {
		return nil, errors.New(tagsProblem)
	}
	if err := refuseRepeatedKeys(node); err != nil {
		return nil, err
	}
	mappings := make([]Mapping, 0, len(node.Content)/2)
	for position := 0; position+1 < len(node.Content); position += 2 {
		name := node.Content[position].Value
		if !sensors.IsName(name) {
			return nil, fmt.Errorf("tags: %q is not a metric name", name)
		}
		tag, err := parseTag(name, node.Content[position+1])
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, Mapping{Name: name, Tag: tag})
	}
	slices.SortFunc(mappings, func(first, second Mapping) int {
		if first.Tag != second.Tag {
			return int(first.Tag) - int(second.Tag)
		}
		return strings.Compare(first.Name, second.Name)
	})
	for position := 1; position < len(mappings); position++ {
		if mappings[position-1].Tag == mappings[position].Tag {
			return nil, fmt.Errorf("tags: %s and %s both have the tag %d",
				mappings[position-1].Name, mappings[position].Name, mappings[position].Tag)
		}
	}
	return mappings, nil
}

func parseTag(name string, node *yaml.Node) (uint8, error) {
	var tag int64
	if node.Kind != yaml.ScalarNode || node.ShortTag() != "!!int" || node.Decode(&tag) != nil {
		return 0, fmt.Errorf("tags: %s: the tag must be a whole number from 0 to 255", name)
	}
	if tag < 0 || tag > maxTag {
		return 0, fmt.Errorf("tags: %s: the tag %s is outside 0 to 255", name, node.Value)
	}
	return uint8(tag), nil
}

func parseInterval(node *yaml.Node) (time.Duration, error) {
	if node == nil {
		return DefaultInterval, nil
	}
	text := ""
	if node.Kind == yaml.ScalarNode {
		text = node.Value
	}
	interval, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("interval: %q is not a duration such as 1s or 500ms", text)
	}
	if interval < MinimumInterval {
		return 0, fmt.Errorf("interval: %s is shorter than 100ms", text)
	}
	return interval, nil
}
