package themecompile

import (
	"strings"
	"testing"
)

func TestCompileFormat(test *testing.T) {
	cases := map[string]string{
		"YYYY-MM-DD":          "1-2-3",
		"HH:mm:ss":            "4:5:6",
		"YYYY-MM-DD HH:mm:ss": "1-2-3 4:5:6",
		"DD/MM/YYYY":          "3/2/1",
		"YYYY.MM":             "1.2",
		strings.TrimSpace(strings.Repeat("YYYY ", 6)): "1 1 1 1 1 1",
		strings.Repeat("ss:", 9):                      strings.Repeat("6:", 9),
	}
	for format, want := range cases {
		got, err := compileFormat(format)
		if err != nil || string(got) != want {
			test.Errorf("%q: got %q, %v, want %q", format, got, err, want)
		}
	}
}

func TestCompileFormatErrors(test *testing.T) {
	cases := map[string]string{
		"YYY":                          `the character 'Y' at position 1 is not a token or a separator`,
		"HH:mmx":                       `the character 'x' at position 6 is not a token or a separator`,
		"HH h":                         `the character 'h' at position 4 is not a token or a separator`,
		"hh":                           `the character 'h' at position 1 is not a token or a separator`,
		"":                             "want a non-empty format",
		strings.Repeat("ss:", 9) + ":": "compiles to 19 bytes, more than 18",
	}
	for format, want := range cases {
		if _, err := compileFormat(format); err == nil || err.Error() != want {
			test.Errorf("%q: got %v, want %q", format, err, want)
		}
	}
}
