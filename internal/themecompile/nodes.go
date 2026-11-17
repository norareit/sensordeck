package themecompile

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"

	"github.com/norareit/sensordeck/internal/config"
)

// problem makes an error from a place and what is wrong there. The place is empty for a fault of the file as a whole.
func problem(place, format string, arguments ...any) error {
	text := fmt.Sprintf(format, arguments...)
	if place == "" {
		return fmt.Errorf("%s", text)
	}
	return fmt.Errorf("%s: %s", place, text)
}

// join puts a key after the place it is in.
func join(place, key string) string {
	if place == "" {
		return key
	}
	return place + ": " + key
}

func resolve(node *yaml.Node) *yaml.Node {
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	return node
}

type entry struct {
	key   string
	value *yaml.Node
}

// entries gives the pairs of a mapping in the order of the document, or the error of the first key that is given twice
// or, when allowed is not nil, not in allowed.
func entries(node *yaml.Node, place string, allowed []string) ([]entry, error) {
	var found []entry
	for position := 0; position+1 < len(node.Content); position += 2 {
		key := resolve(node.Content[position]).Value
		if allowed != nil && !slices.Contains(allowed, key) {
			return nil, problem(place, "unknown key %q, the keys are %s", key, listOf(allowed))
		}
		for _, earlier := range found {
			if earlier.key == key {
				return nil, problem(place, "the key %q is given twice", key)
			}
		}
		found = append(found, entry{key: key, value: resolve(node.Content[position+1])})
	}
	return found, nil
}

// listOf joins the names in alphabetical order with ", " and " and ".
func listOf(names []string) string {
	sorted := slices.Sorted(slices.Values(names))
	if len(sorted) < 2 {
		return strings.Join(sorted, "")
	}
	return strings.Join(sorted[:len(sorted)-1], ", ") + " and " + sorted[len(sorted)-1]
}

// fields gives the values of the keys of a mapping, which the caller only looks up and never ranges over, so that no
// output depends on the order of a Go map.
func fields(node *yaml.Node, place string, allowed []string) (map[string]*yaml.Node, error) {
	found, err := entries(node, place, allowed)
	if err != nil {
		return nil, err
	}
	values := make(map[string]*yaml.Node, len(found))
	for _, each := range found {
		values[each.key] = each.value
	}
	return values, nil
}

// required gives the value of a key, or the error that it is missing.
func required(values map[string]*yaml.Node, place, key string) (*yaml.Node, error) {
	value, found := values[key]
	if !found {
		return nil, problem(place, "the key %s is missing", key)
	}
	return value, nil
}

func hasControl(text string) bool {
	return strings.ContainsFunc(text, unicode.IsControl)
}

func isString(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.ShortTag() != "!!null"
}

// readString reads a string with no control character, and a non-empty one when nonEmpty is set.
func readString(node *yaml.Node, place, key string, nonEmpty bool) (string, error) {
	where := join(place, key)
	if !isString(node) {
		return "", problem(where, "want a string")
	}
	if hasControl(node.Value) || (nonEmpty && node.Value == "") {
		if nonEmpty {
			return "", problem(where, "want a non-empty string with no control character")
		}
		return "", problem(where, "want a string with no control character")
	}
	return node.Value, nil
}

// readWholeNumber reads a whole number from low, up to high when high is not negative.
func readWholeNumber(node *yaml.Node, place, key string, low, high int64) (int64, error) {
	var number int64
	valid := node.Kind == yaml.ScalarNode && node.ShortTag() == "!!int" && node.Decode(&number) == nil
	if valid && number >= low && (high < 0 || number <= high) {
		return number, nil
	}
	if high < 0 {
		return 0, problem(join(place, key), "want a whole number from %d", low)
	}
	return 0, problem(join(place, key), "want a whole number from %d to %d", low, high)
}

// readFraction reads a number from 0 to 1, whole or decimal. The comparison is written so that NaN fails it.
func readFraction(node *yaml.Node, place, key string) (float64, error) {
	var number float64
	tag := node.ShortTag()
	valid := node.Kind == yaml.ScalarNode && (tag == "!!int" || tag == "!!float") && node.Decode(&number) == nil
	if !valid || !(number >= 0 && number <= 1) {
		return 0, problem(join(place, key), "want a number from 0 to 1")
	}
	return number, nil
}

// readBoolean reads true or false, and nothing else: yes and 1 are no booleans.
func readBoolean(node *yaml.Node, place, key string) (bool, error) {
	var value bool
	if node.Kind != yaml.ScalarNode || node.ShortTag() != "!!bool" || node.Decode(&value) != nil {
		return false, problem(join(place, key), "want true or false")
	}
	return value, nil
}

// readColour reads a colour written as #rrggbb. An unquoted # starts a comment in YAML, so the likeliest mistake
// arrives as an empty value and gets a text of its own.
func readColour(node *yaml.Node, place, key string) (config.Colour, error) {
	where := join(place, key)
	if node.Kind == yaml.ScalarNode && (node.ShortTag() == "!!null" || node.Value == "") {
		return config.Colour{}, problem(where, `no colour given, write it in quotes such as "#1e1e2e", because # starts a comment in YAML`)
	}
	colour, ok := config.DecodeColour(node.Value)
	if node.Kind != yaml.ScalarNode || !ok {
		return config.Colour{}, problem(where, `%q is not a colour such as "#1e1e2e"`, node.Value)
	}
	return colour, nil
}
