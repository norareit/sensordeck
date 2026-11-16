// Package builtinthemes holds the themes embedded in the executable. They are read-only, and a theme is one folder of
// themes/ in the format of the theme compiler, so that a new built-in theme is a new folder and no code.
package builtinthemes

import (
	"embed"
	"io/fs"
	"sort"
)

//go:generate go run ./background/generate themes/minimal-dark/background.png
//go:generate go run ./background/generate themes/minimal-dark-probes/background.png

//go:embed themes
var themes embed.FS

// Names gives the names of the built-in themes, sorted.
func Names() []string {
	entries, err := fs.ReadDir(themes, "themes")
	if err != nil {
		panic("the embedded themes directory is unreadable: " + err.Error())
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

// Folder gives the folder of the built-in theme of that exact name, and false when there is none.
func Folder(name string) (fs.FS, bool) {
	for _, each := range Names() {
		if each == name {
			folder, err := fs.Sub(themes, "themes/"+name)
			return folder, err == nil
		}
	}
	return nil, false
}
