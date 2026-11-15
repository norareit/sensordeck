package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/builtinthemes"
	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/uploadstate"
)

func newSession() *session {
	harness := newHarness()
	harness.configDir = "/nowhere"
	return &session{env: environment{userConfigDir: func() (string, error) { return harness.configDir, nil }}}
}

func TestLoadConfigWithTagsGivesItsMappingsAndNoTheme(test *testing.T) {
	loaded, err := newSession().loadConfig("/c/config.yaml", []byte(twoTags))
	if err != nil || loaded.theme != nil || len(loaded.settings.Mappings) != 2 {
		test.Errorf("got %+v, %v", loaded, err)
	}
}

func TestLoadConfigWithABuiltInThemeGivesItsTagsAndTheHashOfItsFile(test *testing.T) {
	folder, _ := builtinthemes.Folder("minimal-dark")
	want, err := themecompile.Compile(folder)
	if err != nil {
		test.Fatal(err)
	}
	loaded, err := newSession().loadConfig("/c/config.yaml", []byte("theme: minimal-dark\n"))
	if err != nil {
		test.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.settings.Mappings, want.Tags) || loaded.theme.value != "minimal-dark" ||
		!bytes.Equal(loaded.theme.file, want.File) || loaded.theme.hash != uploadstate.Hash(want.File) {
		test.Errorf("got %+v", loaded.theme)
	}
}

func TestLoadConfigResolvesARelativeFolderAgainstTheConfigFileNotTheWorkingDirectory(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	want := compileExpected(test, folder)
	for _, value := range []string{"./folder", "folder/", "../" + filepath.Base(filepath.Dir(folder)) + "/folder"} {
		directory := filepath.Dir(folder)
		path := filepath.Join(directory, "config.yaml")
		loaded, err := newSession().loadConfig(path, []byte("theme: "+value+"\n"))
		if err != nil || !bytes.Equal(loaded.theme.file, want.File) {
			test.Errorf("%s: %v", value, err)
		}
	}
	inside := filepath.Join(folder, "config.yaml")
	loaded, err := newSession().loadConfig(inside, []byte("theme: .\n"))
	if err != nil || !bytes.Equal(loaded.theme.file, want.File) {
		test.Errorf("theme .: %v", err)
	}
	loaded, err = newSession().loadConfig("/elsewhere/config.yaml", []byte("theme: '"+folder+"'\n"))
	if err != nil || !bytes.Equal(loaded.theme.file, want.File) {
		test.Errorf("an absolute folder: %v", err)
	}
}

func TestLoadConfigWithAThemeWithNoNumberWidgetGivesNoMapping(test *testing.T) {
	folder := writeThemeFolder(test, "name: none\nbackground: \"#000000\"\nfonts:\n  body: {file: body.ttf, size: 20, color: \"#ffffff\"}\nwidgets:\n  - {type: text, font: body, text: Hello, x: 4, y: 4}\n")
	loaded, err := newSession().loadConfig("/c/config.yaml", []byte("theme: '"+folder+"'\n"))
	if err != nil || len(loaded.settings.Mappings) != 0 || loaded.theme == nil {
		test.Errorf("got %+v, %v", loaded, err)
	}
}

func TestLoadConfigNamesAThemeThatFailsToCompile(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "nothing")
	_, err := newSession().loadConfig("/c/config.yaml", []byte("theme: '"+missing+"'\n"))
	if want := "theme: " + filepath.Join(missing, "theme.yaml") + ": cannot read the file: no such file or directory"; err == nil || err.Error() != want {
		test.Errorf("error %v, want %s", err, want)
	}
}

func TestRestartKeys(test *testing.T) {
	base := config.Config{Interval: time.Second}
	colour := &config.Colour{Red: 1}
	cases := []struct {
		name            string
		running, loaded config.Config
		want            []string
	}{
		{"none", base, base, nil},
		{"interval", base, config.Config{Interval: 2 * time.Second}, []string{"interval"}},
		{"commander interval", base, config.Config{Interval: time.Second, CommanderInterval: time.Minute}, []string{"commander_interval"}},
		{"colour given", base, config.Config{Interval: time.Second, CommanderColour: colour}, []string{"commander_color"}},
		{"colour removed", config.Config{Interval: time.Second, CommanderColour: colour}, base, []string{"commander_color"}},
		{"colour the same", config.Config{Interval: time.Second, CommanderColour: colour}, config.Config{Interval: time.Second, CommanderColour: &config.Colour{Red: 1}}, nil},
		{"all", base, config.Config{Interval: 3 * time.Second, CommanderInterval: time.Minute, CommanderColour: colour},
			[]string{"interval", "commander_interval", "commander_color"}},
	}
	for _, each := range cases {
		if got := restartKeys(each.running, each.loaded); !reflect.DeepEqual(got, each.want) {
			test.Errorf("%s: got %v, want %v", each.name, got, each.want)
		}
	}
}

func TestFileStampsAreTheSameWhenBothFailedOrTimeAndSizeAre(test *testing.T) {
	moment := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		first, second fileStamp
		want          bool
	}{
		{fileStamp{failed: true}, fileStamp{failed: true}, true},
		{fileStamp{failed: true}, fileStamp{modTime: moment}, false},
		{fileStamp{modTime: moment, size: 3}, fileStamp{modTime: moment, size: 3}, true},
		{fileStamp{modTime: moment, size: 3}, fileStamp{modTime: moment.Add(time.Second), size: 3}, false},
		{fileStamp{modTime: moment, size: 3}, fileStamp{modTime: moment, size: 4}, false},
	}
	for position, each := range cases {
		if got := each.first.same(each.second); got != each.want {
			test.Errorf("case %d: got %v", position, got)
		}
	}
}

// newWatcher gives a watcher of a config file that holds twoTags, whose loader counts its calls and fails on a file
// that holds the text broken.
func newWatcher(test *testing.T) (*configWatcher, string, *bytes.Buffer, *int) {
	test.Helper()
	path := writeConfig(test, twoTags)
	var stderr bytes.Buffer
	loads := 0
	watcher := &configWatcher{path: path, last: stampOf(path), stderr: &stderr}
	watcher.load = func(content []byte) (runConfig, error) {
		loads++
		if strings.Contains(string(content), "broken") {
			return runConfig{}, os.ErrInvalid
		}
		settings, err := config.Parse(content)
		return runConfig{settings: settings}, err
	}
	return watcher, path, &stderr, &loads
}

func TestPollGivesNothingWhileTheFileIsUnchanged(test *testing.T) {
	watcher, _, stderr, loads := newWatcher(test)
	if _, changed := watcher.poll(); changed || *loads != 0 || stderr.Len() != 0 {
		test.Errorf("changed %v, loads %d, stderr %q", changed, *loads, stderr.String())
	}
}

func TestPollSeesANewTimeWithTheSameContentAndANewSizeWithTheSameTime(test *testing.T) {
	watcher, path, _, loads := newWatcher(test)
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(path, later, later); err != nil {
		test.Fatal(err)
	}
	if _, changed := watcher.poll(); !changed || *loads != 1 {
		test.Errorf("a new time: changed %v, loads %d", changed, *loads)
	}
	if _, changed := watcher.poll(); changed {
		test.Error("the same stamp was read twice")
	}
	if err := os.WriteFile(path, []byte(twoTags+"# more\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	if err := os.Chtimes(path, later, later); err != nil {
		test.Fatal(err)
	}
	if reloaded, changed := watcher.poll(); !changed || len(reloaded.settings.Mappings) != 2 || *loads != 2 {
		test.Errorf("a new size: changed %v, loads %d", changed, *loads)
	}
}

func TestPollReportsABrokenFileOnceAndReadsTheNextChange(test *testing.T) {
	watcher, path, stderr, loads := newWatcher(test)
	if err := os.WriteFile(path, []byte("broken"), 0o644); err != nil {
		test.Fatal(err)
	}
	for range 3 {
		if _, changed := watcher.poll(); changed {
			test.Fatal("a broken file was applied")
		}
	}
	want := "config file " + path + ": invalid argument, keeping the running config\n"
	if stderr.String() != want || *loads != 1 {
		test.Errorf("stderr %q, loads %d", stderr.String(), *loads)
	}
	if err := os.WriteFile(path, []byte("broken more"), 0o644); err != nil {
		test.Fatal(err)
	}
	watcher.poll()
	if strings.Count(stderr.String(), "keeping the running config") != 2 {
		test.Errorf("stderr %q", stderr.String())
	}
}

func TestPollTakesADeletedFileAsAChangeThatIsNotApplied(test *testing.T) {
	watcher, path, stderr, _ := newWatcher(test)
	if err := os.Remove(path); err != nil {
		test.Fatal(err)
	}
	for range 2 {
		if _, changed := watcher.poll(); changed {
			test.Fatal("a deleted file was applied")
		}
	}
	want := "config file " + path + ": cannot read it: no such file or directory, keeping the running config\n"
	if stderr.String() != want {
		test.Errorf("stderr %q", stderr.String())
	}
}
