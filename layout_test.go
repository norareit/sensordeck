package main

import (
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	hidrawImport       = "github.com/norareit/sensordeck/internal/hidraw"
	linuxsensorsImport = "github.com/norareit/sensordeck/internal/linuxsensors"
	retiredYAMLImport  = "gopkg.in/yaml.v3"
)

// goFiles walks the module and gives the path of every Go file, leaving out what is no part of the module.
func goFiles(test *testing.T) []string {
	test.Helper()
	var files []string
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		skipped := entry.Name() == "node_modules" || entry.Name() == "local-themes" ||
			(entry.Name() != "." && strings.HasPrefix(entry.Name(), "."))
		if entry.IsDir() && skipped {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		test.Fatal(err)
	}
	return files
}

func importsOf(test *testing.T, path string) []string {
	test.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		test.Fatal(err)
	}
	var imports []string
	for _, spec := range parsed.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			test.Fatal(err)
		}
		imports = append(imports, imported)
	}
	return imports
}

// excludedFromWindows reads both the build line and the suffix of the file name, as the go tool does.
func excludedFromWindows(test *testing.T, path string) bool {
	test.Helper()
	context := build.Default
	context.GOOS = "windows"
	matches, err := context.MatchFile(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		test.Fatal(err)
	}
	return !matches
}

func inDirectory(path, directory string) bool {
	return strings.HasPrefix(filepath.ToSlash(path), directory+"/")
}

func TestNoFileImportsC(test *testing.T) {
	for _, path := range goFiles(test) {
		for _, imported := range importsOf(test, path) {
			if imported == "C" {
				test.Errorf("%s imports C", path)
			}
		}
	}
}

func TestLinuxCodeIsLimitedToLinux(test *testing.T) {
	linuxOnly := []string{"internal/hidraw", "internal/fakescreen", "internal/linuxsensors", "internal/fakesensors", "e2e"}
	for _, path := range goFiles(test) {
		for _, directory := range linuxOnly {
			if inDirectory(path, directory) && !excludedFromWindows(test, path) {
				test.Errorf("%s is part of a Windows build", path)
			}
		}
	}
}

func TestOnlyLinuxFilesOfTheCommandsNameTheLinuxCode(test *testing.T) {
	for _, path := range goFiles(test) {
		if !inDirectory(path, "internal/cli") || excludedFromWindows(test, path) {
			continue
		}
		for _, imported := range importsOf(test, path) {
			if imported == hidrawImport || imported == linuxsensorsImport {
				test.Errorf("%s imports %s and is part of a Windows build", path, imported)
			}
		}
	}
}

func TestNoFileImportsTheArchivedYAMLPackage(test *testing.T) {
	for _, path := range goFiles(test) {
		for _, imported := range importsOf(test, path) {
			if imported == retiredYAMLImport {
				test.Errorf("%s imports %s, which is archived", path, imported)
			}
		}
	}
}
