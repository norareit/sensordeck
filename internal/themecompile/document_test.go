package themecompile

import (
	"fmt"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/config"
)

const goodFonts = "fonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\n"

func TestParseDocumentErrors(test *testing.T) {
	cases := []struct{ name, theme, want string }{
		{"unknown key at the top", "name: a\nbackground: \"#000000\"\ncolour: 1\n",
			`unknown key "colour", the keys are author, background, fonts, license, name and widgets`},
		{"unknown key in a font", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#fff000\", bold: 1}\n",
			`font body: unknown key "bold", the keys are builtin, color, file and size`},
		{"key twice at the top", "name: a\nname: b\nbackground: \"#000000\"\n", `the key "name" is given twice`},
		{"key twice in a widget", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: text, font: body, text: a, x: 0, y: 0, x: 1}\n",
			`widget 1: the key "x" is given twice`},
		{"name missing", "background: \"#000000\"\n", "the key name is missing"},
		{"name empty", "name: \"\"\nbackground: \"#000000\"\n", "name: want a non-empty string with no control character"},
		{"name with a tab", "name: \"a\\tb\"\nbackground: \"#000000\"\n", "name: want a non-empty string with no control character"},
		{"author with a newline", "name: a\nauthor: \"a\\nb\"\nbackground: \"#000000\"\n", "author: want a string with no control character"},
		{"background missing", "name: a\n", "the key background is missing"},
		{"background unquoted", "name: a\nbackground: #123456\n", `background: no colour given, write it in quotes such as "#1e1e2e", because # starts a comment in YAML`},
		{"colour malformed", "name: a\nbackground: \"#12345\"\n", `background: "#12345" is not a colour such as "#1e1e2e"`},
		{"fonts as a list", "name: a\nbackground: \"#000000\"\nfonts: [a]\n", "fonts: want a mapping from a font name to a font"},
		{"font with both", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {builtin: go-regular, file: a.ttf, size: 20, color: \"#fff000\"}\n", "font body: give exactly one of builtin and file"},
		{"font with neither", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {size: 20, color: \"#fff000\"}\n", "font body: give exactly one of builtin and file"},
		{"unknown Go font", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {builtin: arial, size: 20, color: \"#fff000\"}\n",
			`font body: builtin: "arial" is not a Go font, the Go fonts are go-regular, go-bold, go-italic, go-bold-italic, go-medium, go-medium-italic, go-mono, go-mono-bold, go-mono-italic, go-mono-bold-italic, go-smallcaps, go-smallcaps-italic`},
		{"size 0", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {builtin: go-regular, size: 0, color: \"#fff000\"}\n", "font body: size: want a whole number from 1 to 320"},
		{"size 321", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {builtin: go-regular, size: 321, color: \"#fff000\"}\n", "font body: size: want a whole number from 1 to 320"},
		{"size 1.5", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {builtin: go-regular, size: 1.5, color: \"#fff000\"}\n", "font body: size: want a whole number from 1 to 320"},
		{"font without colour", "name: a\nbackground: \"#000000\"\nfonts:\n  body: {builtin: go-regular, size: 20}\n", "font body: the key color is missing"},
		{"widgets as a mapping", "name: a\nbackground: \"#000000\"\nwidgets: {a: b}\n", "widgets: want a list of widgets"},
		{"widget not a mapping", "name: a\nbackground: \"#000000\"\nwidgets: [a]\n", "widget 1: want a mapping"},
		{"widget without type", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {x: 0}\n", "widget 1: the key type is missing"},
		{"widget of type gauge", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: gauge}\n", `widget 1: type: "gauge" is not a widget type, the types are bar, datetime, image, number and text`},
		{"align middle", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, align: middle, x: 0, y: 0}\n", "widget 1 (number): align: want left, center or right"},
		{"align 1", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, align: 1, x: 0, y: 0}\n", "widget 1 (number): align: want left, center or right"},
		{"opacity: 1.5", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, opacity: 1.5, x: 0, y: 0}\n", "widget 1 (number): opacity: want a number from 0 to 1"},
		{"opacity: -0.1", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, opacity: -0.1, x: 0, y: 0}\n", "widget 1 (number): opacity: want a number from 0 to 1"},
		{"opacity as a string", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, opacity: \"0.5\", x: 0, y: 0}\n", "widget 1 (number): opacity: want a number from 0 to 1"},
		{"opacity: 50%", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, opacity: 50%, x: 0, y: 0}\n", "widget 1 (number): opacity: want a number from 0 to 1"},
		{"opacity: .nan", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, opacity: .nan, x: 0, y: 0}\n", "widget 1 (number): opacity: want a number from 0 to 1"},
		{"thousands: yes", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, thousands: yes, x: 0, y: 0}\n", "widget 1 (number): thousands: want true or false"},
		{"thousands: 1", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, thousands: 1, x: 0, y: 0}\n", "widget 1 (number): thousands: want true or false"},
		{"thousands as a string", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, thousands: \"true\", x: 0, y: 0}\n", "widget 1 (number): thousands: want true or false"},
		{"bar without metric", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, width: 10, height: 5, color: \"#00ff00\", x: 0, y: 0}\n", `widget 1 (bar): the key metric is missing`},
		{"bar without width", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu.util, height: 5, color: \"#00ff00\", x: 0, y: 0}\n", `widget 1 (bar): the key width is missing`},
		{"bar without height", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu.util, width: 10, color: \"#00ff00\", x: 0, y: 0}\n", `widget 1 (bar): the key height is missing`},
		{"bar without color", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu.util, width: 10, height: 5, x: 0, y: 0}\n", `widget 1 (bar): the key color is missing`},
		{"bar width 0", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu.util, width: 0, height: 5, color: \"#00ff00\", x: 0, y: 0}\n", `widget 1 (bar): width: want a whole number from 1`},
		{"bar height 0", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu.util, width: 10, height: 0, color: \"#00ff00\", x: 0, y: 0}\n", `widget 1 (bar): height: want a whole number from 1`},
		{"bar with a font", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu.util, width: 10, height: 5, color: \"#00ff00\", x: 0, y: 0, font: body}\n", `widget 1 (bar): unknown key "font", the keys are background, color, height, metric, type, width, x and y`},
		{"bar metric misspelt", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu..temp, width: 10, height: 5, color: \"#00ff00\", x: 0, y: 0}\n", `widget 1 (bar): metric: "cpu..temp" is not a metric name`},
		{"bar colour malformed", "name: a\nbackground: \"#000000\"\nwidgets:\n  - {type: bar, metric: cpu.util, width: 10, height: 5, color: \"#12345\", x: 0, y: 0}\n", `widget 1 (bar): color: "#12345" is not a colour such as "#1e1e2e"`},
		{"x negative", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: text, font: body, text: a, x: -1, y: 0}\n", "widget 1 (text): x: want a whole number from 0"},
		{"x a letter", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: text, font: body, text: a, x: a, y: 0}\n", "widget 1 (text): x: want a whole number from 0"},
		{"y missing", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: text, font: body, text: a, x: 0}\n", "widget 1 (text): the key y is missing"},
		{"text empty", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: text, font: body, text: \"\", x: 0, y: 0}\n", "widget 1 (text): text: want a non-empty string with no control character"},
		{"text with a newline", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: text, font: body, text: \"a\\nb\", x: 0, y: 0}\n", "widget 1 (text): text: want a non-empty string with no control character"},
		{"digits 0", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, digits: 0, x: 0, y: 0}\n", "widget 1 (number): digits: want a whole number from 1 to 5"},
		{"digits 6", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, digits: 6, x: 0, y: 0}\n", "widget 1 (number): digits: want a whole number from 1 to 5"},
		{"metric misspelt", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu..temp, x: 0, y: 0}\n", `widget 1 (number): metric: "cpu..temp" is not a metric name`},
		{"metric in capitals", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: CPU.temp, x: 0, y: 0}\n", `widget 1 (number): metric: "CPU.temp" is not a metric name`},
		{"format with a letter", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: datetime, font: body, format: \"HH:mmx\", x: 0, y: 0}\n", `widget 1 (datetime): format: the character "x" at position 6 is not a token or a separator`},
		{"format too long", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: datetime, font: body, format: \"" + strings.Repeat("ss:", 9) + ":\", x: 0, y: 0}\n", "widget 1 (datetime): format: compiles to 19 bytes, more than 18"},
		{"font not in fonts", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - {type: text, font: head, text: a, x: 0, y: 0}\n", `widget 1 (text): font: "head" is not a font of fonts`},
		{"63 widgets", "name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n" + strings.Repeat("  - {type: text, font: body, text: a, x: 0, y: 0}\n", 63), "widgets: 63 widgets, more than the 62 that fit beside the background"},
		{"not a mapping", "- a\n", "want a mapping"},
		{"empty", "", "want a mapping"},
	}
	for _, each := range cases {
		_, err := parseDocument([]byte(each.theme))
		if err == nil || !strings.HasPrefix(err.Error(), each.want) {
			test.Errorf("%s: got %v, want %q", each.name, err, each.want)
		}
	}
}

func TestParseDocumentAcceptsUpperCaseColoursAndDefaults(test *testing.T) {
	theme := "name: a\nbackground: \"#ABCDEF\"\n" + goodFonts + "widgets:\n  - {type: number, font: body, metric: cpu.temp, x: 0, y: 0}\n"
	parsed, err := parseDocument([]byte(theme))
	if err != nil {
		test.Fatal(err)
	}
	if parsed.background.colour.Red != 0xab || parsed.widgets[0].digits != 3 || parsed.widgets[0].number != 1 {
		test.Errorf("%+v", parsed)
	}
}

func TestParseDocumentGivesTheSameFaultEachTime(test *testing.T) {
	theme := "name: a\nbackground: \"#000000\"\n" + goodFonts +
		"widgets:\n  - {type: text, font: body, text: a, x: -1, y: 0}\n  - {type: gauge}\n"
	first := fmt.Sprint(parseError(theme))
	for range 20 {
		if again := fmt.Sprint(parseError(theme)); again != first {
			test.Fatalf("%q and %q", first, again)
		}
	}
	if first != "widget 1 (text): x: want a whole number from 0" {
		test.Errorf("got %q", first)
	}
}

func parseError(theme string) error {
	_, err := parseDocument([]byte(theme))
	return err
}

func parseWidget(test *testing.T, widget string) widgetSpec {
	test.Helper()
	parsed, err := parseDocument([]byte("name: a\nbackground: \"#000000\"\n" + goodFonts + "widgets:\n  - " + widget + "\n"))
	if err != nil {
		test.Fatal(err)
	}
	return parsed.widgets[0]
}

func TestParseDocumentReadsTheNumberOptions(test *testing.T) {
	cases := []struct {
		keys               string
		alignment, opacity byte
		thousands          bool
	}{
		{"", 1, 255, false},
		{", align: left", 0, 255, false},
		{", align: center", 1, 255, false},
		{", align: right", 2, 255, false},
		{", opacity: 0", 1, 0, false},
		{", opacity: 0.5", 1, 128, false},
		{", opacity: 1", 1, 255, false},
		{", opacity: 0.0", 1, 0, false},
		{", thousands: true", 1, 255, true},
		{", thousands: false", 1, 255, false},
	}
	for _, each := range cases {
		widget := parseWidget(test, "{type: number, font: body, metric: cpu.temp, x: 0, y: 0"+each.keys+"}")
		if widget.alignment != each.alignment || widget.opacity != each.opacity || widget.thousands != each.thousands {
			test.Errorf("%q: got %d, %d, %v", each.keys, widget.alignment, widget.opacity, widget.thousands)
		}
	}
}

func TestParseDocumentReadsABar(test *testing.T) {
	widget := parseWidget(test, `{type: bar, metric: cpu.util, x: 3, y: 4, width: 100, height: 10, color: "#00ff00", background: "#ff0000"}`)
	if widget.kind != "bar" || widget.metric != "cpu.util" || widget.x != 3 || widget.y != 4 || widget.width != 100 || widget.height != 10 ||
		widget.colour.Green != 0xff || widget.background.Red != 0xff {
		test.Errorf("%+v", widget)
	}
	plain := parseWidget(test, `{type: bar, metric: cpu.util, x: 3, y: 4, width: 100, height: 10, color: "#00ff00"}`)
	if plain.background != (config.Colour{}) {
		test.Errorf("background %+v", plain.background)
	}
}
