package config

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"go.yaml.in/yaml/v3"

	"github.com/norareit/sensordeck/internal/sensors"
)

const (
	// DefaultEvery is the time between two runs of a command metric when the definition does not say.
	DefaultEvery = 10 * time.Second
	// MinimumEvery is the shortest time between two runs, so that a typo cannot start a process in every round.
	MinimumEvery = time.Second
)

// Metric is the definition of a custom metric. It is comparable, so that a reload can tell a definition that is
// unchanged. Exactly one of File, Hwmon and Command is not empty.
type Metric struct {
	Name    string
	File    string        // the file form
	Hwmon   string        // the hwmon form, with Input and Index
	Input   string        // the hwmon form only
	Index   int           // the hwmon form only, 1 without the key
	Command string        // the command form, with Every
	Every   time.Duration // the command form only, DefaultEvery without the key
	Scale   float64       // 1 without the key
	Offset  float64
	Unit    string
}

const metricsProblem = "metrics: want a mapping from metric name to definition, with at least one entry"

const metricKeys = "command, every, file, hwmon, index, input, offset, scale and unit"

// parseMetrics gives the definitions in the order of the file, and none when the key is not given.
func parseMetrics(node *yaml.Node) ([]Metric, error) {
	if node == nil {
		return nil, nil
	}
	if node.Kind != yaml.MappingNode || len(node.Content) == 0 {
		return nil, errors.New(metricsProblem)
	}
	if err := refuseRepeatedKeys(node); err != nil {
		return nil, err
	}
	definitions := make([]Metric, 0, len(node.Content)/2)
	for position := 0; position+1 < len(node.Content); position += 2 {
		name := node.Content[position].Value
		if !sensors.IsWellFormedName(name) {
			return nil, fmt.Errorf("metrics: %q is not a metric name: segments of a to z, 0 to 9, _ and - joined by dots, starting with a letter", name)
		}
		if sensors.IsName(name) {
			return nil, fmt.Errorf("metrics: %s: is a built-in metric and cannot be replaced", name)
		}
		definition, err := parseMetric(name, node.Content[position+1])
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func parseMetric(name string, node *yaml.Node) (Metric, error) {
	if node.Kind != yaml.MappingNode {
		return Metric{}, fmt.Errorf("metrics: %s: want a mapping with one of the keys file, hwmon and command", name)
	}
	if err := refuseRepeatedKeys(node); err != nil {
		return Metric{}, err
	}
	values := map[string]*yaml.Node{}
	for position := 0; position+1 < len(node.Content); position += 2 {
		key := node.Content[position].Value
		switch key {
		case "command", "every", "file", "hwmon", "index", "input", "offset", "scale", "unit":
			values[key] = node.Content[position+1]
		default:
			return Metric{}, fmt.Errorf("metrics: %s: unknown key %q, the keys are %s", name, key, metricKeys)
		}
	}
	return metricFromValues(name, values)
}

// metricFromValues checks the keys of one definition in a fixed order, so that a definition with several problems
// gives the first.
func metricFromValues(name string, values map[string]*yaml.Node) (Metric, error) {
	sources := 0
	for _, key := range []string{"file", "hwmon", "command"} {
		if values[key] != nil {
			sources++
		}
	}
	if sources != 1 {
		return Metric{}, fmt.Errorf("metrics: %s: want exactly one of the keys file, hwmon and command", name)
	}
	for _, key := range []string{"input", "index"} {
		if values[key] != nil && values["hwmon"] == nil {
			return Metric{}, fmt.Errorf("metrics: %s: %s: only with hwmon", name, key)
		}
	}
	if values["every"] != nil && values["command"] == nil {
		return Metric{}, fmt.Errorf("metrics: %s: every: only with command", name)
	}
	if values["hwmon"] != nil && values["input"] == nil {
		return Metric{}, fmt.Errorf("metrics: %s: input: needed with hwmon", name)
	}
	definition := Metric{Name: name, Scale: 1}
	for _, step := range []func(string, map[string]*yaml.Node, *Metric) error{
		readSource, readHwmonKeys, readNumbers, readUnit, readEvery,
	} {
		if err := step(name, values, &definition); err != nil {
			return Metric{}, err
		}
	}
	return definition, nil
}

func readSource(name string, values map[string]*yaml.Node, definition *Metric) error {
	if node := values["file"]; node != nil {
		if !isString(node) || !filepath.IsAbs(node.Value) {
			return fmt.Errorf("metrics: %s: file: want an absolute path", name)
		}
		definition.File = node.Value
	}
	if node := values["command"]; node != nil {
		if !isString(node) || node.Value == "" {
			return fmt.Errorf("metrics: %s: command: want a non-empty string", name)
		}
		definition.Command = node.Value
	}
	return nil
}

func readHwmonKeys(name string, values map[string]*yaml.Node, definition *Metric) error {
	node := values["hwmon"]
	if node == nil {
		return nil
	}
	if !isString(node) || node.Value == "" || strings.Contains(node.Value, "/") {
		return fmt.Errorf("metrics: %s: hwmon: want a chip name, a non-empty string without /", name)
	}
	definition.Hwmon = node.Value
	input := values["input"]
	if !isString(input) || input.Value == "" || input.Value == "." || input.Value == ".." || strings.Contains(input.Value, "/") {
		return fmt.Errorf("metrics: %s: input: want the name of a file of the chip, a non-empty string other than . and .. without /", name)
	}
	definition.Input = input.Value
	definition.Index = 1
	if index := values["index"]; index != nil {
		var number int64
		if index.Kind != yaml.ScalarNode || index.ShortTag() != "!!int" || index.Decode(&number) != nil || number < 1 {
			return fmt.Errorf("metrics: %s: index: want a whole number of 1 or more", name)
		}
		definition.Index = int(number)
	}
	return nil
}

func readNumbers(name string, values map[string]*yaml.Node, definition *Metric) error {
	for _, key := range []string{"scale", "offset"} {
		node := values[key]
		if node == nil {
			continue
		}
		number, ok := finiteNumber(node)
		if !ok {
			return fmt.Errorf("metrics: %s: %s: want a finite number", name, key)
		}
		if key == "scale" {
			definition.Scale = number
		} else {
			definition.Offset = number
		}
	}
	return nil
}

func readUnit(name string, values map[string]*yaml.Node, definition *Metric) error {
	node := values["unit"]
	if node == nil {
		return nil
	}
	if !isString(node) || strings.ContainsFunc(node.Value, unicode.IsControl) {
		return fmt.Errorf("metrics: %s: unit: want a string without tabs, line feeds or other control characters", name)
	}
	definition.Unit = node.Value
	return nil
}

func readEvery(name string, values map[string]*yaml.Node, definition *Metric) error {
	if values["command"] == nil {
		return nil
	}
	definition.Every = DefaultEvery
	node := values["every"]
	if node == nil {
		return nil
	}
	text := ""
	if node.Kind == yaml.ScalarNode {
		text = node.Value
	}
	every, err := time.ParseDuration(text)
	if err != nil {
		return fmt.Errorf("metrics: %s: every: %q is not a duration such as 30s or 5m", name, text)
	}
	if every < MinimumEvery {
		return fmt.Errorf("metrics: %s: every: %s is shorter than 1s", name, text)
	}
	definition.Every = every
	return nil
}

func isString(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.ShortTag() == "!!str"
}

func finiteNumber(node *yaml.Node) (float64, bool) {
	if node.Kind != yaml.ScalarNode || (node.ShortTag() != "!!int" && node.ShortTag() != "!!float") {
		return 0, false
	}
	var number float64
	if node.Decode(&number) != nil || math.IsInf(number, 0) || math.IsNaN(number) {
		return 0, false
	}
	return number, true
}
