// Package config reads and checks the config file of the command run. It is not limited to a platform.
package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/url"
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

	// DefaultCommanderInterval is the shortest time between two readings of the controller when the file does not say.
	DefaultCommanderInterval = time.Minute

	// DefaultLibreHardwareMonitorURL is where LibreHardwareMonitor serves its tree when its web server is left as it is.
	DefaultLibreHardwareMonitorURL = "http://localhost:8085/data.json"

	maxTag = 255
)

// DefaultMonitorURL gives the URL of DefaultLibreHardwareMonitorURL, as a value the caller may keep.
func DefaultMonitorURL() *url.URL {
	return &url.URL{Scheme: "http", Host: "localhost:8085", Path: "/data.json"}
}

// Mapping sends the value of a metric to a tag of the theme on the screen.
type Mapping struct {
	Name string
	Tag  uint8
}

// Colour is a colour of the LEDs, as the key commander_color gives it.
type Colour struct{ Red, Green, Blue uint8 }

// Config is what the file holds.
type Config struct {
	Interval          time.Duration
	CommanderInterval time.Duration // zero when the file does not give it
	CommanderColour   *Colour       // the key commander_color; nil when the file does not give it
	Theme             string        // the key theme as the file gives it; empty when the file does not give it
	// LibreHardwareMonitorURL is the key librehardwaremonitor_url, never nil after Parse. Only Windows reads it.
	LibreHardwareMonitorURL *url.URL
	Mappings                []Mapping // ordered by tag, ascending; empty with a theme, whose tags the compiler assigns
	Metrics                 []Metric  // the custom metrics, in the order of the file; empty without the key metrics
}

// IsMetric tells whether a name is a built-in metric or one defined under metrics.
func (config Config) IsMetric(name string) bool {
	return isMetric(name, config.Metrics)
}

func isMetric(name string, definitions []Metric) bool {
	return sensors.IsName(name) || slices.ContainsFunc(definitions, func(each Metric) bool { return each.Name == name })
}

// CommanderPeriod is the shortest time between two readings of the controller in run.
func (config Config) CommanderPeriod() time.Duration {
	if config.CommanderInterval != 0 {
		return config.CommanderInterval
	}
	return max(DefaultCommanderInterval, config.Interval)
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
	var themeNode, tagsNode, metricsNode, intervalNode, commanderIntervalNode, commanderColourNode, monitorURLNode *yaml.Node
	for position := 0; position+1 < len(root.Content); position += 2 {
		key, value := root.Content[position], root.Content[position+1]
		switch key.Value {
		case "theme":
			themeNode = value
		case "tags":
			tagsNode = value
		case "metrics":
			metricsNode = value
		case "interval":
			intervalNode = value
		case "commander_interval":
			commanderIntervalNode = value
		case "commander_color":
			commanderColourNode = value
		case "librehardwaremonitor_url":
			monitorURLNode = value
		default:
			return Config{}, fmt.Errorf("unknown key %q, the keys are commander_color, commander_interval, interval, librehardwaremonitor_url, metrics, tags and theme", key.Value)
		}
	}
	metrics, err := parseMetrics(metricsNode)
	if err != nil {
		return Config{}, err
	}
	theme, mappings, err := parseThemeOrTags(themeNode, tagsNode, metrics)
	if err != nil {
		return Config{}, err
	}
	interval, err := parseInterval(intervalNode)
	if err != nil {
		return Config{}, err
	}
	commanderInterval, err := parseCommanderInterval(commanderIntervalNode, interval)
	if err != nil {
		return Config{}, err
	}
	colour, err := parseCommanderColour(commanderColourNode)
	if err != nil {
		return Config{}, err
	}
	if colour != nil && commanderIntervalNode != nil {
		return Config{}, errors.New("commander_interval: cannot be used with commander_color, which reads the controller in every round")
	}
	monitorURL, err := parseMonitorURL(monitorURLNode)
	if err != nil {
		return Config{}, err
	}
	return Config{Interval: interval, CommanderInterval: commanderInterval, CommanderColour: colour, Theme: theme,
		LibreHardwareMonitorURL: monitorURL, Mappings: mappings, Metrics: metrics}, nil
}

var errNotAMapping = errors.New("want a mapping with the keys commander_color, commander_interval, interval, librehardwaremonitor_url, metrics, tags and theme")

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

// parseThemeOrTags takes the key theme or the key tags, which exclude each other. A key counts as given when it is
// present, whatever its value.
func parseThemeOrTags(themeNode, tagsNode *yaml.Node, metrics []Metric) (string, []Mapping, error) {
	switch {
	case themeNode != nil && tagsNode != nil:
		return "", nil, errors.New("theme and tags cannot both be given, the theme gives the tags")
	case themeNode == nil && tagsNode == nil:
		return "", nil, errors.New("one of theme and tags is needed")
	case themeNode != nil:
		theme, err := parseTheme(themeNode)
		return theme, nil, err
	}
	mappings, err := parseTags(tagsNode, metrics)
	return "", mappings, err
}

var errEmptyTheme = errors.New("theme: the theme name or folder path is empty")

func parseTheme(node *yaml.Node) (string, error) {
	if node.Kind == yaml.ScalarNode && node.ShortTag() == "!!null" {
		return "", errEmptyTheme
	}
	if node.Kind != yaml.ScalarNode || node.ShortTag() != "!!str" {
		return "", errors.New("theme: want a theme name or folder path as a string")
	}
	if node.Value == "" {
		return "", errEmptyTheme
	}
	return node.Value, nil
}

const tagsProblem = "tags: want a mapping from metric name to tag, with at least one entry"

func parseTags(node *yaml.Node, metrics []Metric) ([]Mapping, error) {
	if node == nil || node.Kind != yaml.MappingNode || len(node.Content) == 0 {
		return nil, errors.New(tagsProblem)
	}
	if err := refuseRepeatedKeys(node); err != nil {
		return nil, err
	}
	mappings := make([]Mapping, 0, len(node.Content)/2)
	for position := 0; position+1 < len(node.Content); position += 2 {
		name := node.Content[position].Value
		if !isMetric(name, metrics) {
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
	notWhole := fmt.Errorf("tags: %s: the tag must be a whole number from 0 to 255", name)
	outside := fmt.Errorf("tags: %s: the tag %s is outside 0 to 255", name, node.Value)
	if node.Kind != yaml.ScalarNode {
		return 0, notWhole
	}
	switch node.ShortTag() {
	case "!!int":
		// The resolved tag decides whether it is whole, whatever the spelling, so a value that does not fit int64
		// is out of range and not malformed.
		var tag int64
		if node.Decode(&tag) != nil || tag < 0 || tag > maxTag {
			return 0, outside
		}
		return uint8(tag), nil
	case "!!float":
		// The parser reads a decimal integer too large even for uint64 as a float, so for that case alone the text
		// decides.
		if _, whole := new(big.Int).SetString(strings.ReplaceAll(node.Value, "_", ""), 10); whole {
			return 0, outside
		}
	}
	return 0, notWhole
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

// parseCommanderInterval gives zero when the file does not give the key. It is checked after the interval, which it
// must not be shorter than.
func parseCommanderInterval(node *yaml.Node, interval time.Duration) (time.Duration, error) {
	if node == nil {
		return 0, nil
	}
	text := ""
	if node.Kind == yaml.ScalarNode {
		text = node.Value
	}
	commanderInterval, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("commander_interval: %q is not a duration such as 1m or 30s", text)
	}
	if commanderInterval < interval {
		return 0, fmt.Errorf("commander_interval: %s is shorter than the interval %s", text, interval)
	}
	return commanderInterval, nil
}

// parseMonitorURL gives the default when the file does not give the key. The host is the host without the port, so
// that a URL with only a port is refused.
func parseMonitorURL(node *yaml.Node) (*url.URL, error) {
	if node == nil {
		return DefaultMonitorURL(), nil
	}
	notAURL := fmt.Errorf("librehardwaremonitor_url: %q is not an http URL such as %s", node.Value, DefaultLibreHardwareMonitorURL)
	if node.Kind != yaml.ScalarNode || node.ShortTag() != "!!str" {
		return nil, notAURL
	}
	address, err := url.Parse(node.Value)
	if err != nil || !address.IsAbs() || (address.Scheme != "http" && address.Scheme != "https") || address.Hostname() == "" {
		return nil, notAURL
	}
	return address, nil
}

// parseCommanderColour gives nil when the file does not give the key. An unquoted # starts a comment in YAML, so the
// likeliest mistake arrives as an empty value and gets a text of its own.
func parseCommanderColour(node *yaml.Node) (*Colour, error) {
	if node == nil {
		return nil, nil
	}
	text := ""
	if node.Kind == yaml.ScalarNode {
		text = node.Value
	}
	if node.Kind == yaml.ScalarNode && (node.ShortTag() == "!!null" || text == "") {
		return nil, errors.New(`commander_color: no colour given, write it in quotes such as "#00c8ff", because # starts a comment in YAML`)
	}
	colour, ok := DecodeColour(text)
	if !ok {
		return nil, fmt.Errorf(`commander_color: %q is not a colour such as "#00c8ff"`, text)
	}
	return &colour, nil
}

// DecodeColour reads a colour written as # and six hexadecimal digits, in either case. It gives false for any other
// text.
func DecodeColour(text string) (Colour, bool) {
	digits, found := strings.CutPrefix(text, "#")
	if !found || len(digits) != 6 {
		return Colour{}, false
	}
	components, err := hex.DecodeString(digits)
	if err != nil {
		return Colour{}, false
	}
	return Colour{Red: components[0], Green: components[1], Blue: components[2]}, true
}
