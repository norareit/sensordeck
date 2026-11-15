// Package notices makes the file of the licence notices of the code linked into the executable.
package notices

import (
	"fmt"
	"sort"
	"strings"
)

// File is a licence or notice file of a module.
type File struct {
	Name string
	Text string
}

// Module is a module of go.mod with the licence files at its root.
type Module struct {
	Path    string
	Version string
	Files   []File
}

var licencePrefixes = []string{"LICENSE", "LICENCE", "COPYING", "NOTICE"}

// LicenceFileNames keeps the names that start with LICENSE, LICENCE, COPYING or NOTICE, in byte order.
func LicenceFileNames(names []string) []string {
	var kept []string
	for _, name := range names {
		for _, prefix := range licencePrefixes {
			if strings.HasPrefix(name, prefix) {
				kept = append(kept, name)
				break
			}
		}
	}
	sort.Strings(kept)
	return kept
}

const partSeparator = "\n" + "================================================================================" + "\n\n"

// Render gives the text of the notices file: the part of the Go standard library, then one part for each module in the
// order of their paths. It fails, naming the module, when a module has no file.
func Render(standardLibrary string, modules []Module) (string, error) {
	ordered := append([]Module(nil), modules...)
	sort.Slice(ordered, func(first, second int) bool { return ordered[first].Path < ordered[second].Path })
	parts := []string{part("Go standard library", []File{{Name: "LICENSE", Text: standardLibrary}})}
	for _, module := range ordered {
		if len(module.Files) == 0 {
			return "", fmt.Errorf("%s %s has no licence file at its root", module.Path, module.Version)
		}
		parts = append(parts, part(module.Path+" "+module.Version, module.Files))
	}
	return strings.Join(parts, partSeparator), nil
}

func part(header string, files []File) string {
	var text strings.Builder
	text.WriteString(header + "\n")
	for _, file := range files {
		text.WriteString("\n" + file.Name + "\n" + strings.Repeat("-", len(file.Name)) + "\n\n" + file.Text)
		if !strings.HasSuffix(file.Text, "\n") {
			text.WriteString("\n")
		}
	}
	return text.String()
}
