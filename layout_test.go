package main

import (
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	hidrawImport       = "github.com/norareit/sensordeck/internal/hidraw"
	linuxsensorsImport = "github.com/norareit/sensordeck/internal/linuxsensors"
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
	return excludedFrom(test, path, "windows")
}

func excludedFrom(test *testing.T, path, system string) bool {
	test.Helper()
	context := build.Default
	context.GOOS = system
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
	linuxOnly := []string{"internal/hidraw", "internal/fakescreen", "internal/linuxsensors", "internal/fakesensors",
		"internal/fakecommander", "internal/fakepty", "e2e"}
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

func TestEveryFileThatImportsTheLinuxCodeIsLimitedToLinux(test *testing.T) {
	for _, path := range goFiles(test) {
		if excludedFromWindows(test, path) {
			continue
		}
		for _, imported := range importsOf(test, path) {
			if imported == hidrawImport || imported == linuxsensorsImport {
				test.Errorf("%s imports %s and is part of a Windows build", path, imported)
			}
		}
	}
}

func TestOnlyTheSeamOnWindowsImportsTheWindowsPackage(test *testing.T) {
	const windowsImport = "golang.org/x/sys/windows"
	allowed := []string{"internal/winhid/seam_windows.go", "internal/winhid/seam_windows_test.go"}
	for _, path := range goFiles(test) {
		for _, imported := range importsOf(test, path) {
			if imported != windowsImport {
				continue
			}
			if !slices.Contains(allowed, filepath.ToSlash(path)) {
				test.Errorf("%s imports %s", path, imported)
			} else if !excludedFrom(test, path, "linux") {
				test.Errorf("%s imports %s and is part of a Linux build", path, imported)
			}
		}
	}
}

func TestTheOnlyLibraryLoadedByNameIsHidDLLFromTheSystemDirectory(test *testing.T) {
	for _, path := range goFiles(test) {
		if strings.HasSuffix(path, "layout_test.go") {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			test.Fatal(err)
		}
		text := string(content)
		for _, forbidden := range []string{"NewLazyDLL", "LoadLibrary", "LoadDLL", "MustLoadDLL"} {
			if strings.Contains(text, forbidden) {
				test.Errorf("%s uses %s, which does not load from the system directory only", path, forbidden)
			}
		}
		for _, call := range strings.Split(text, "NewLazySystemDLL(")[1:] {
			if !strings.HasPrefix(call, `"hid.dll")`) {
				test.Errorf("%s loads a library other than hid.dll", path)
			}
		}
	}
}

// retiredYAMLImport is written in two pieces so that a search of the module for the name finds no use of the package.
const retiredYAMLImport = "gopkg.in/" + "yaml.v3"

func TestNoFileImportsTheArchivedYAMLPackage(test *testing.T) {
	for _, path := range goFiles(test) {
		for _, imported := range importsOf(test, path) {
			if imported == retiredYAMLImport {
				test.Errorf("%s imports %s, which is archived", path, imported)
			}
		}
	}
}

func TestThePackagesOfThemesDoNoIO(test *testing.T) {
	const themePrefix = "github.com/norareit/sensordeck/internal/"
	themePackages := []string{"themefile", "themeimage", "themedump", "faketheme"}
	forbidden := map[string]bool{"os": true, "os/exec": true, "os/signal": true, "net": true, "syscall": true, "unsafe": true}
	for _, path := range goFiles(test) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		for _, name := range themePackages {
			if !inDirectory(path, "internal/"+name) {
				continue
			}
			for _, imported := range importsOf(test, path) {
				own := false
				for _, other := range themePackages {
					own = own || imported == themePrefix+other
				}
				standard := !strings.Contains(strings.Split(imported, "/")[0], ".")
				if forbidden[imported] || !(standard || own) {
					test.Errorf("%s imports %s", path, imported)
				}
			}
		}
	}
}

func TestTheThemeCompilerDoesNoIO(test *testing.T) {
	const prefix = "github.com/norareit/sensordeck/internal/"
	forbidden := []string{"os", "os/exec", "os/signal", "net", "syscall", "unsafe",
		prefix + "hidraw", prefix + "linuxsensors", prefix + "commandercore", prefix + "smartmonitor"}
	for _, path := range goFiles(test) {
		if strings.HasSuffix(path, "_test.go") || !inDirectory(path, "internal/themecompile") {
			continue
		}
		for _, imported := range importsOf(test, path) {
			if slices.Contains(forbidden, imported) {
				test.Errorf("%s imports %s", path, imported)
			}
		}
	}
}
