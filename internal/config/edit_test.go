package config

import (
	"errors"
	"testing"
)

func TestSetTheme(test *testing.T) {
	cases := []struct{ name, content, value, want string }{
		{"empty", "", "minimal-dark", "theme: minimal-dark\n"},
		{"comments only", "# my config\n", "minimal-dark", "# my config\ntheme: minimal-dark\n"},
		{"replaced", "# c\ninterval: 2s\ntheme: old # note\n# end\n", "new", "# c\ninterval: 2s\ntheme: new\n# end\n"},
		{"added with other key", "interval: 2s\n# tail\n", "new", "interval: 2s\n# tail\ntheme: new\n"},
		{"added with no final line feed", "interval: 2s", "new", "interval: 2s\ntheme: new\n"},
		{"carriage return kept", "interval: 2s\r\ntheme: old\r\ncommander_interval: 1m\r\n", "new", "interval: 2s\r\ntheme: new\r\ncommander_interval: 1m\r\n"},
		{"indented key", "  interval: 2s\n  theme: old\n  commander_interval: 1m\n", "new", "  interval: 2s\n  theme: new\n  commander_interval: 1m\n"},
		{"quoted old value", "theme: \"old\"\n", "new", "theme: new\n"},
		{"windows path", "interval: 1s\n", `C:\Users\ada\my theme`, "interval: 1s\ntheme: C:\\Users\\ada\\my theme\n"},
		{"quoted apostrophe", "", "it's #x", "theme: 'it''s #x'\n"},
		{"hash", "", "a #b", "theme: 'a #b'\n"},
		{"apostrophe", "", "it's", "theme: it's\n"},
		{"number", "", "2024", "theme: '2024'\n"},
		{"bool", "", "true", "theme: 'true'\n"},
		{"dash", "", "- x", "theme: '- x'\n"},
		{"tab", "", "a\tb", "theme: \"a\\x09b\"\n"},
	}
	for _, each := range cases {
		got, err := SetTheme([]byte(each.content), each.value)
		if err != nil || string(got) != each.want {
			test.Errorf("%s: got %q, %v, want %q", each.name, got, err, each.want)
			continue
		}
		parsed, err := Parse(got)
		if err != nil || parsed.Theme != each.value {
			test.Errorf("%s: the result reads back as %q, %v, want %q", each.name, parsed.Theme, err, each.value)
		}
	}
}

func TestSetThemeRefusesWhatItCannotEdit(test *testing.T) {
	for name, content := range map[string]string{
		"flow root":       "{theme: old}\n",
		"list root":       "- theme\n",
		"block scalar":    "theme: |\n  old\n",
		"value next line": "theme:\n  old\n",
		"continued plain": "theme: old\n  more\n",
		"alias":           "x: &a one\ntheme: *a\n",
	} {
		got, err := SetTheme([]byte(content), "new")
		if !errors.Is(err, ErrNotEditable) {
			test.Errorf("%s: got %q, %v, want ErrNotEditable", name, got, err)
		}
	}
}

func TestSetThemeGivesTheErrorOfTheParser(test *testing.T) {
	_, err := SetTheme([]byte("a: [1\n  - x: {"), "new")
	if err == nil || errors.Is(err, ErrNotEditable) {
		test.Errorf("error %v, want that of the parser", err)
	}
}
