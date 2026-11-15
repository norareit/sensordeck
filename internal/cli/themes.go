package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/norareit/sensordeck/internal/builtinthemes"
	"github.com/norareit/sensordeck/internal/themecompile"
)

// isFolderPath tells a folder path from a theme name by the text alone: . and .. and anything with a separator of a
// path on this platform is a path.
func isFolderPath(argument string) bool {
	if argument == "." || argument == ".." || strings.Contains(argument, "/") {
		return true
	}
	return runtime.GOOS == "windows" && strings.ContainsAny(argument, `\:`)
}

// themesDirectory gives the directory sensordeck/themes in the config directory of the user.
func (current *session) themesDirectory() (string, error) {
	directory, err := current.ownConfigDirectory()
	if err != nil {
		return "", fmt.Errorf("cannot find the config directory: %w", err)
	}
	return filepath.Join(directory, "themes"), nil
}

// installedTheme gives the path of the entry of the name in the themes directory and whether it is an installed theme:
// a directory, or a symbolic link that leads to one. An entry that does not exist, or a link that leads nowhere, is no
// theme and no error.
func installedTheme(directory, name string) (string, bool, error) {
	path := filepath.Join(directory, name)
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("cannot examine the installed theme %s: %w", path, innerError(err))
	}
	return path, info.IsDir(), nil
}

// compileTheme compiles the theme the first argument of theme compile and theme preview names: a folder path, or a
// name looked up among the installed themes and then the built-in ones.
func (current *session) compileTheme(argument string) (themecompile.Result, error) {
	if argument == "" {
		return themecompile.Result{}, errors.New("a theme name or folder is needed")
	}
	if isFolderPath(argument) {
		return compileFolder(argument)
	}
	return current.compileNamed(argument)
}

// compileNamed compiles a theme by its name, looked up among the installed themes and then the built-in ones.
func (current *session) compileNamed(argument string) (themecompile.Result, error) {
	directory, err := current.themesDirectory()
	if err != nil {
		return themecompile.Result{}, err
	}
	path, installed, err := installedTheme(directory, argument)
	if err != nil {
		return themecompile.Result{}, err
	}
	if installed {
		return compileFolder(path)
	}
	folder, found := builtinthemes.Folder(argument)
	if !found {
		return themecompile.Result{}, fmt.Errorf("no installed or built-in theme is named %s; a folder is named by a path, such as ./%s", argument, argument)
	}
	result, err := themecompile.Compile(folder)
	if err != nil {
		return themecompile.Result{}, fmt.Errorf("built-in theme %s: %w", argument, err)
	}
	return result, nil
}

// installedThemes gives the installed themes by name with their folder: the entries of the themes directory that
// installedTheme accepts, so that the list and the lookup agree. A themes directory that does not exist holds none.
func (current *session) installedThemes() (map[string]string, error) {
	directory, err := current.themesDirectory()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read the themes directory %s: %w", directory, innerError(err))
	}
	installed := map[string]string{}
	for _, entry := range entries {
		path, found, err := installedTheme(directory, entry.Name())
		if err != nil {
			return nil, err
		}
		if found {
			installed[entry.Name()] = path
		}
	}
	return installed, nil
}

// themeListLines gives the lines of theme list, sorted by name by their bytes: the built-in themes by name, and the
// installed themes by name with their folder.
func themeListLines(builtin []string, installed map[string]string) []string {
	lines := map[string]string{}
	for _, name := range builtin {
		lines[name] = name + "\tbuilt-in"
	}
	for name, folder := range installed {
		kind := "installed"
		if _, replaces := lines[name]; replaces {
			kind = "installed, overrides built-in"
		}
		lines[name] = name + "\t" + kind + "\t" + folder
	}
	names := make([]string, 0, len(lines))
	for name := range lines {
		names = append(names, name)
	}
	sort.Strings(names)
	sorted := make([]string, 0, len(names))
	for _, name := range names {
		sorted = append(sorted, lines[name])
	}
	return sorted
}
