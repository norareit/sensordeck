package config

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ErrNotEditable is the config file whose key theme cannot be set by replacing or adding one line.
var ErrNotEditable = errors.New("cannot be edited, set the key theme in it by hand")

// SetTheme gives the content with the key theme set to the value, as R25 and R26 of unit 0013 say, every other byte
// kept. Its error is that of the YAML parser for content that is not YAML, or ErrNotEditable.
func SetTheme(content []byte, value string) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, err
	}
	line := "theme: " + scalarOf(value)
	edited, err := withThemeLine(content, &document, line)
	if err != nil {
		return nil, err
	}
	// A replaced line can leave a continuation of the old value behind, which only the new content shows.
	if !readsBackAs(edited, value) {
		return nil, ErrNotEditable
	}
	return edited, nil
}

// withThemeLine replaces the line of the key theme, or adds the line at the end when the file has no such key.
func withThemeLine(content []byte, document *yaml.Node, line string) ([]byte, error) {
	if document.Kind == 0 {
		return appendLine(content, line), nil
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return nil, ErrNotEditable
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode || root.Style&yaml.FlowStyle != 0 {
		return nil, ErrNotEditable
	}
	for position := 0; position+1 < len(root.Content); position += 2 {
		key, value := root.Content[position], root.Content[position+1]
		if key.Value != "theme" {
			continue
		}
		onTheLine := value.Kind == yaml.ScalarNode && value.Style&(yaml.LiteralStyle|yaml.FoldedStyle) == 0 && value.Line == key.Line
		if !onTheLine {
			return nil, ErrNotEditable
		}
		return replaceLine(content, key.Line, strings.Repeat(" ", key.Column-1)+line), nil
	}
	// The new key has the indentation of the keys the file has, or the mapping is not the one of the root.
	return appendLine(content, strings.Repeat(" ", max(root.Column-1, 0))+line), nil
}

func appendLine(content []byte, line string) []byte {
	edited := string(content)
	if edited != "" && !strings.HasSuffix(edited, "\n") {
		edited += "\n"
	}
	return []byte(edited + line + "\n")
}

// replaceLine replaces the text of the line, counted from 1, and keeps its line ending.
func replaceLine(content []byte, number int, replacement string) []byte {
	lines := strings.Split(string(content), "\n")
	old := lines[number-1]
	if strings.HasSuffix(old, "\r") {
		replacement += "\r"
	}
	lines[number-1] = replacement
	return []byte(strings.Join(lines, "\n"))
}

// readsBackAs tells whether the root key theme of the content is exactly the value.
func readsBackAs(content []byte, value string) bool {
	var document yaml.Node
	if yaml.Unmarshal(content, &document) != nil || document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return false
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return false
	}
	for position := 0; position+1 < len(root.Content); position += 2 {
		if root.Content[position].Value == "theme" {
			read := root.Content[position+1]
			return read.Kind == yaml.ScalarNode && read.ShortTag() == "!!str" && read.Value == value
		}
	}
	return false
}

// scalarOf gives the value as it is when the line reads back as a plain string of exactly the value, and quoted
// otherwise: in single quotes, and in double quotes when the value holds a control character.
func scalarOf(value string) string {
	var document yaml.Node
	if !strings.ContainsFunc(value, isControl) && yaml.Unmarshal([]byte("theme: "+value+"\n"), &document) == nil && plainStringOf(&document, value) {
		return value
	}
	if !strings.ContainsFunc(value, isControl) {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	var quoted strings.Builder
	quoted.WriteByte('"')
	for _, character := range value {
		switch {
		case character == '\\' || character == '"':
			quoted.WriteByte('\\')
			quoted.WriteRune(character)
		case isControl(character):
			fmt.Fprintf(&quoted, `\x%02X`, character)
		default:
			quoted.WriteRune(character)
		}
	}
	quoted.WriteByte('"')
	return quoted.String()
}

func isControl(character rune) bool { return character < 0x20 || character == 0x7f }

func plainStringOf(document *yaml.Node, value string) bool {
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return false
	}
	root := document.Content[0]
	if len(root.Content) != 2 {
		return false
	}
	read := root.Content[1]
	return read.Kind == yaml.ScalarNode && read.Style == 0 && read.ShortTag() == "!!str" && read.Value == value
}
