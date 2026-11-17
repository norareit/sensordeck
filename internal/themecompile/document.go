package themecompile

import (
	"fmt"
	"math"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
)

const (
	maxWidgets   = 62
	maxFontSize  = 320
	maxDigits    = 5
	defaultDigit = 3
)

type document struct {
	background backgroundSpec
	fonts      []fontSpec   // in the order of the file
	widgets    []widgetSpec // in the order of the list
}

type backgroundSpec struct {
	colour config.Colour
	path   string // empty for a colour
}

type fontSpec struct {
	name, file, builtin string
	size                int
	colour              config.Colour
}

type widgetSpec struct {
	number        int    // counted from 1
	kind          string // "text", "number", "datetime", "image" or "bar"
	x, y          int64
	font          int // the position of its font in document.fonts
	text, metric  string
	digits        int
	alignment     byte          // of a number, as byte 11 of its record holds it
	opacity       byte          // of a number, as byte 15 of its record holds it
	thousands     bool          // of a number
	width, height int64         // of a bar
	colour        config.Colour // of a bar, the fill and the frame
	background    config.Colour // of a bar, the empty part
	format        []byte        // compiled
	file          string
	alarm         *alarmSpec // of a number; nil without the key alarm
}

// alarmSpec is the key alarm of a number: when it is in the alarm state, and the colour it has then.
type alarmSpec struct {
	threshold config.Threshold
	colour    config.Colour
}

var (
	topKeys    = []string{"author", "background", "fonts", "license", "name", "widgets"}
	fontKeys   = []string{"builtin", "color", "file", "size"}
	widgetKeys = map[string][]string{
		"text":     {"font", "text", "type", "x", "y"},
		"number":   {"align", "alarm", "digits", "font", "metric", "opacity", "thousands", "type", "x", "y"},
		"datetime": {"font", "format", "type", "x", "y"},
		"image":    {"file", "type", "x", "y"},
		"bar":      {"background", "color", "height", "metric", "type", "width", "x", "y"},
	}
)

// parseDocument reads the content of theme.yaml and checks everything about it that needs no other file.
func parseDocument(content []byte) (document, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return document{}, err
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 || resolve(root.Content[0]).Kind != yaml.MappingNode {
		return document{}, problem("", "want a mapping")
	}
	values, err := fields(resolve(root.Content[0]), "", topKeys)
	if err != nil {
		return document{}, err
	}
	var parsed document
	if err := checkIdentity(values); err != nil {
		return document{}, err
	}
	if parsed.background, err = readBackground(values); err != nil {
		return document{}, err
	}
	if parsed.fonts, err = readFonts(values["fonts"]); err != nil {
		return document{}, err
	}
	parsed.widgets, err = readWidgets(values["widgets"], parsed.fonts)
	return parsed, err
}

// checkIdentity checks name, author and license, whose values nothing of the file holds.
func checkIdentity(values map[string]*yaml.Node) error {
	name, err := required(values, "", "name")
	if err != nil {
		return err
	}
	if _, err := readString(name, "", "name", true); err != nil {
		return err
	}
	for _, key := range []string{"author", "license"} {
		if value, found := values[key]; found {
			if _, err := readString(value, "", key, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func readBackground(values map[string]*yaml.Node) (backgroundSpec, error) {
	node, err := required(values, "", "background")
	if err != nil {
		return backgroundSpec{}, err
	}
	if !isString(node) {
		if node.Kind == yaml.ScalarNode {
			colour, err := readColour(node, "", "background")
			return backgroundSpec{colour: colour}, err
		}
		return backgroundSpec{}, problem("background", "want a string")
	}
	if strings.HasPrefix(node.Value, "#") {
		colour, err := readColour(node, "", "background")
		return backgroundSpec{colour: colour}, err
	}
	path, err := readString(node, "", "background", true)
	return backgroundSpec{path: path}, err
}

func readFonts(node *yaml.Node) ([]fontSpec, error) {
	if node == nil {
		return nil, nil
	}
	if node.Kind != yaml.MappingNode {
		return nil, problem("fonts", "want a mapping from a font name to a font")
	}
	found, err := entries(node, "fonts", nil)
	if err != nil {
		return nil, err
	}
	var fonts []fontSpec
	for _, each := range found {
		if each.key == "" {
			return nil, problem("fonts", "a font name must not be empty")
		}
		font, err := readFont(each.key, each.value)
		if err != nil {
			return nil, err
		}
		fonts = append(fonts, font)
	}
	return fonts, nil
}

func readFont(name string, node *yaml.Node) (fontSpec, error) {
	place := "font " + name
	if node.Kind != yaml.MappingNode {
		return fontSpec{}, problem(place, "want a mapping")
	}
	values, err := fields(node, place, fontKeys)
	if err != nil {
		return fontSpec{}, err
	}
	font := fontSpec{name: name}
	fileNode, hasFile := values["file"]
	builtinNode, hasBuiltin := values["builtin"]
	if hasFile == hasBuiltin {
		return fontSpec{}, problem(place, "give exactly one of builtin and file")
	}
	if hasFile {
		font.file, err = readString(fileNode, place, "file", true)
	} else {
		font.builtin, err = readString(builtinNode, place, "builtin", true)
		if err == nil && !isGoFont(font.builtin) {
			err = problem(join(place, "builtin"), "%q is not a Go font, the Go fonts are %s", font.builtin, goFontNames())
		}
	}
	if err != nil {
		return fontSpec{}, err
	}
	size, err := required(values, place, "size")
	if err != nil {
		return fontSpec{}, err
	}
	number, err := readWholeNumber(size, place, "size", 1, maxFontSize)
	if err != nil {
		return fontSpec{}, err
	}
	font.size = int(number)
	colour, err := required(values, place, "color")
	if err != nil {
		return fontSpec{}, err
	}
	font.colour, err = readColour(colour, place, "color")
	return font, err
}

func readWidgets(node *yaml.Node, fonts []fontSpec) ([]widgetSpec, error) {
	if node == nil {
		return nil, nil
	}
	if node.Kind != yaml.SequenceNode {
		return nil, problem("widgets", "want a list of widgets")
	}
	if err := checkWidgetCount(node); err != nil {
		return nil, err
	}
	var widgets []widgetSpec
	for position, item := range node.Content {
		widget, err := readWidget(position+1, resolve(item), fonts)
		if err != nil {
			return nil, err
		}
		widgets = append(widgets, widget)
	}
	return widgets, nil
}

// checkWidgetCount refuses a list with more records than fit, before any widget is read. An alarm number is two records.
func checkWidgetCount(list *yaml.Node) error {
	records, alarmNumbers := 0, 0
	for _, item := range list.Content {
		records++
		if isAlarmNumber(resolve(item)) {
			records++
			alarmNumbers++
		}
	}
	if records <= maxWidgets {
		return nil
	}
	ending := ""
	if alarmNumbers > 0 {
		ending = ", an alarm number counting as two"
	}
	return problem("widgets", "%d widgets, more than the %d that fit beside the background%s", records, maxWidgets, ending)
}

// isAlarmNumber tells whether a widget of the list is a number with the key alarm, before it is read in full.
func isAlarmNumber(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	isNumber, hasAlarm := false, false
	for position := 0; position+1 < len(node.Content); position += 2 {
		switch resolve(node.Content[position]).Value {
		case "type":
			value := resolve(node.Content[position+1])
			isNumber = isString(value) && value.Value == "number"
		case "alarm":
			hasAlarm = true
		}
	}
	return isNumber && hasAlarm
}

func widgetPlace(number int, kind string) string {
	if kind == "" {
		return fmt.Sprintf("widget %d", number)
	}
	return fmt.Sprintf("widget %d (%s)", number, kind)
}

func widgetTypes() []string {
	types := make([]string, 0, len(widgetKeys))
	for kind := range widgetKeys {
		types = append(types, kind)
	}
	return types
}

// widgetKind reads the type of a widget, which decides the keys that the rest of it may have.
func widgetKind(number int, node *yaml.Node) (string, error) {
	place := widgetPlace(number, "")
	found, err := entries(node, place, nil)
	if err != nil {
		return "", err
	}
	for _, each := range found {
		if each.key != "type" {
			continue
		}
		kind, err := readString(each.value, place, "type", true)
		if err != nil {
			return "", err
		}
		if _, known := widgetKeys[kind]; !known {
			return "", problem(place, "type: %q is not a widget type, the types are %s", kind, listOf(widgetTypes()))
		}
		return kind, nil
	}
	return "", problem(place, "the key type is missing")
}

func readWidget(number int, node *yaml.Node, fonts []fontSpec) (widgetSpec, error) {
	if node.Kind != yaml.MappingNode {
		return widgetSpec{}, problem(widgetPlace(number, ""), "want a mapping")
	}
	kind, err := widgetKind(number, node)
	if err != nil {
		return widgetSpec{}, err
	}
	place := widgetPlace(number, kind)
	values, err := fields(node, place, widgetKeys[kind])
	if err != nil {
		return widgetSpec{}, err
	}
	widget := widgetSpec{number: number, kind: kind, digits: defaultDigit}
	if widget.x, widget.y, err = readPosition(values, place); err != nil {
		return widgetSpec{}, err
	}
	switch kind {
	case "image":
		err = readImageKeys(&widget, values, place)
	case "bar":
		err = readBarKeys(&widget, values, place)
	default:
		err = readFontReference(&widget, values, place, fonts)
		if err == nil {
			err = readTypeKeys(&widget, values, place)
		}
	}
	return widget, err
}

func readPosition(values map[string]*yaml.Node, place string) (x, y int64, err error) {
	for _, axis := range []struct {
		key   string
		value *int64
	}{{"x", &x}, {"y", &y}} {
		node, err := required(values, place, axis.key)
		if err != nil {
			return 0, 0, err
		}
		if *axis.value, err = readWholeNumber(node, place, axis.key, 0, -1); err != nil {
			return 0, 0, err
		}
	}
	return x, y, nil
}

func readImageKeys(widget *widgetSpec, values map[string]*yaml.Node, place string) error {
	node, err := required(values, place, "file")
	if err != nil {
		return err
	}
	widget.file, err = readString(node, place, "file", true)
	return err
}

func readFontReference(widget *widgetSpec, values map[string]*yaml.Node, place string, fonts []fontSpec) error {
	node, err := required(values, place, "font")
	if err != nil {
		return err
	}
	name, err := readString(node, place, "font", true)
	if err != nil {
		return err
	}
	for position, font := range fonts {
		if font.name == name {
			widget.font = position
			return nil
		}
	}
	return problem(place, "font: %q is not a font of fonts", name)
}

func readTypeKeys(widget *widgetSpec, values map[string]*yaml.Node, place string) error {
	switch widget.kind {
	case "text":
		node, err := required(values, place, "text")
		if err != nil {
			return err
		}
		widget.text, err = readString(node, place, "text", true)
		return err
	case "number":
		return readNumberKeys(widget, values, place)
	}
	node, err := required(values, place, "format")
	if err != nil {
		return err
	}
	text, err := readString(node, place, "format", true)
	if err != nil {
		return err
	}
	if widget.format, err = compileFormat(text); err != nil {
		return problem(join(place, "format"), "%s", err)
	}
	return nil
}

func readMetric(widget *widgetSpec, values map[string]*yaml.Node, place string) error {
	node, err := required(values, place, "metric")
	if err != nil {
		return err
	}
	if widget.metric, err = readString(node, place, "metric", true); err != nil {
		return err
	}
	if !sensors.IsWellFormedName(widget.metric) {
		return problem(join(place, "metric"), "%q is not a metric name", widget.metric)
	}
	return nil
}

var alignments = map[string]byte{"left": 0, "center": 1, "right": 2}

func readNumberKeys(widget *widgetSpec, values map[string]*yaml.Node, place string) error {
	if err := readMetric(widget, values, place); err != nil {
		return err
	}
	if err := readNumberOptions(widget, values, place); err != nil {
		return err
	}
	if digits, found := values["digits"]; found {
		number, err := readWholeNumber(digits, place, "digits", 1, maxDigits)
		if err != nil {
			return err
		}
		widget.digits = int(number)
	}
	if node, found := values["alarm"]; found {
		return readAlarm(widget, values, node, place)
	}
	return nil
}

var alarmKeys = []string{"above", "below", "color"}

// readAlarm reads the key alarm of a number, and makes the number left-aligned, which its method needs: a value that is
// hidden must push its digits out of the box on the right.
func readAlarm(widget *widgetSpec, values map[string]*yaml.Node, node *yaml.Node, place string) error {
	where := join(place, "alarm")
	if node.Kind != yaml.MappingNode {
		return problem(where, "want a mapping with above, below and color")
	}
	fieldValues, err := fields(node, where, alarmKeys)
	if err != nil {
		return err
	}
	var alarm alarmSpec
	for _, bound := range []struct {
		key string
		set func(value int16)
	}{
		{"above", func(value int16) { alarm.threshold.Above, alarm.threshold.HasAbove = value, true }},
		{"below", func(value int16) { alarm.threshold.Below, alarm.threshold.HasBelow = value, true }},
	} {
		boundNode, found := fieldValues[bound.key]
		if !found {
			continue
		}
		number, err := readWholeNumber(boundNode, where, bound.key, math.MinInt16, math.MaxInt16)
		if err != nil {
			return err
		}
		bound.set(int16(number))
	}
	if !alarm.threshold.HasAbove && !alarm.threshold.HasBelow {
		return problem(where, "want above, below or both")
	}
	colour, err := required(fieldValues, where, "color")
	if err != nil {
		return err
	}
	if alarm.colour, err = readColour(colour, where, "color"); err != nil {
		return err
	}
	if _, found := values["align"]; found && widget.alignment != alignments["left"] {
		return problem(join(place, "align"), "an alarm number is left-aligned, want left or no align")
	}
	if widget.thousands {
		return problem(join(place, "thousands"), "an alarm number cannot show thousands")
	}
	widget.alignment = alignments["left"]
	widget.alarm = &alarm
	return nil
}

// readNumberOptions reads align, opacity and thousands, whose defaults are the values every theme compiled before them
// holds: centred, opaque and not in thousands.
func readNumberOptions(widget *widgetSpec, values map[string]*yaml.Node, place string) error {
	widget.alignment, widget.opacity = alignments["center"], 0xff
	if node, found := values["align"]; found {
		alignment, known := alignments[node.Value]
		if !isString(node) || !known {
			return problem(join(place, "align"), "want left, center or right")
		}
		widget.alignment = alignment
	}
	if node, found := values["opacity"]; found {
		opacity, err := readFraction(node, place, "opacity")
		if err != nil {
			return err
		}
		widget.opacity = byte(math.Floor(opacity*255 + 0.5))
	}
	if node, found := values["thousands"]; found {
		thousands, err := readBoolean(node, place, "thousands")
		if err != nil {
			return err
		}
		widget.thousands = thousands
	}
	return nil
}

func readBarKeys(widget *widgetSpec, values map[string]*yaml.Node, place string) error {
	if err := readMetric(widget, values, place); err != nil {
		return err
	}
	for _, size := range []struct {
		key   string
		value *int64
	}{{"width", &widget.width}, {"height", &widget.height}} {
		node, err := required(values, place, size.key)
		if err != nil {
			return err
		}
		if *size.value, err = readWholeNumber(node, place, size.key, 1, -1); err != nil {
			return err
		}
	}
	node, err := required(values, place, "color")
	if err != nil {
		return err
	}
	if widget.colour, err = readColour(node, place, "color"); err != nil {
		return err
	}
	if node, found := values["background"]; found {
		widget.background, err = readColour(node, place, "background")
	}
	return err
}
