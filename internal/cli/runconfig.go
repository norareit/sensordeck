package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/uploadstate"
)

// configTheme is the theme of the config, compiled.
type configTheme struct {
	value string // the value of the key theme as the file gives it
	file  []byte
	hash  string
}

// runConfig is a config of run as it is applied: its settings, with the mappings of the theme when it has one.
type runConfig struct {
	settings config.Config
	theme    *configTheme // nil with tags
}

// loadConfig parses the content of the config file at the path and, with theme, compiles the theme and takes its
// tags as the mappings. Its error is the text after the prefix naming the file.
func (current *session) loadConfig(path string, content []byte) (runConfig, error) {
	settings, err := config.Parse(content)
	if err != nil {
		return runConfig{}, err
	}
	if settings.Theme == "" {
		return runConfig{settings: settings}, nil
	}
	compiled, err := current.compileConfigTheme(path, settings.Theme)
	if err != nil {
		return runConfig{}, fmt.Errorf("theme: %w", err)
	}
	settings.Mappings = compiled.Tags
	return runConfig{
		settings: settings,
		theme:    &configTheme{value: settings.Theme, file: compiled.File, hash: uploadstate.Hash(compiled.File)},
	}, nil
}

// compileConfigTheme compiles the theme a config file names. A folder path that is not absolute is relative to the
// directory of the config file, so that it means the same wherever run was started. The joined path is compiled as a
// folder and not looked up again, because a join can lose the separator that makes a text a folder path.
func (current *session) compileConfigTheme(configPath, value string) (themecompile.Result, error) {
	if !isFolderPath(value) {
		return current.compileNamed(value)
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(filepath.Dir(configPath), value)
	}
	return compileFolder(value)
}

// restartKeys gives the keys that changed and that a reload does not apply, in the order interval,
// commander_interval, commander_color.
func restartKeys(running, reloaded config.Config) []string {
	var keys []string
	if running.Interval != reloaded.Interval {
		keys = append(keys, "interval")
	}
	if running.CommanderInterval != reloaded.CommanderInterval {
		keys = append(keys, "commander_interval")
	}
	if !reflect.DeepEqual(running.CommanderColour, reloaded.CommanderColour) {
		keys = append(keys, "commander_color")
	}
	return keys
}

// fileStamp is what os.Stat gives of the config file, or that it gave an error.
type fileStamp struct {
	failed  bool
	modTime time.Time
	size    int64
}

func stampOf(path string) fileStamp {
	info, err := os.Stat(path)
	if err != nil {
		return fileStamp{failed: true}
	}
	return fileStamp{modTime: info.ModTime(), size: info.Size()}
}

// same tells whether two stamps are the same: both failed, or neither failed and the times and sizes are.
func (stamp fileStamp) same(other fileStamp) bool {
	if stamp.failed || other.failed {
		return stamp.failed && other.failed
	}
	return stamp.modTime.Equal(other.modTime) && stamp.size == other.size
}

// configWatcher gives a reloaded config when the stamp of the file changed.
type configWatcher struct {
	path   string
	last   fileStamp
	load   func(content []byte) (runConfig, error)
	stderr io.Writer
}

// poll takes the stamp and, when it differs from the last one, reads and loads the file. It gives the config when it
// loaded, and writes the line of R19 when it did not. The stamp becomes the last one either way, so that a broken file
// is reported once.
func (watcher *configWatcher) poll() (runConfig, bool) {
	stamp := stampOf(watcher.path)
	if stamp.same(watcher.last) {
		return runConfig{}, false
	}
	watcher.last = stamp
	content, err := os.ReadFile(watcher.path)
	if err != nil {
		watcher.refuse(fmt.Errorf("cannot read it: %w", innerError(err)))
		return runConfig{}, false
	}
	reloaded, err := watcher.load(content)
	if err != nil {
		watcher.refuse(err)
		return runConfig{}, false
	}
	return reloaded, true
}

func (watcher *configWatcher) refuse(err error) {
	_, _ = fmt.Fprintf(watcher.stderr, "config file %s: %v, keeping the running config\n", watcher.path, err)
}
