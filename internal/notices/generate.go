package notices

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Generate gives the text of the notices file for the module in moduleRoot. It needs the module cache, and for a
// module that is not in it, go mod download.
func Generate(moduleRoot string) (string, error) {
	paths, versions, err := requiredModules(moduleRoot)
	if err != nil {
		return "", err
	}
	directories, err := moduleDirectories(moduleRoot, paths)
	if err != nil {
		return "", err
	}
	var modules []Module
	for _, path := range paths {
		if directories[path] == "" {
			return "", fmt.Errorf("%s %s is not in the module cache; run go mod download", path, versions[path])
		}
		files, err := readLicenceFiles(directories[path])
		if err != nil {
			return "", fmt.Errorf("%s %s: %w", path, versions[path], err)
		}
		modules = append(modules, Module{Path: path, Version: versions[path], Files: files})
	}
	goroot, err := command(moduleRoot, "go", "env", "GOROOT")
	if err != nil {
		return "", err
	}
	standardLibrary, err := os.ReadFile(filepath.Join(strings.TrimSpace(goroot), "LICENSE"))
	if err != nil {
		return "", fmt.Errorf("the licence of the Go standard library: %w", err)
	}
	return Render(string(standardLibrary), modules)
}

func command(directory, name string, arguments ...string) (string, error) {
	run := exec.Command(name, arguments...)
	run.Dir = directory
	var stdout, stderr bytes.Buffer
	run.Stdout, run.Stderr = &stdout, &stderr
	if err := run.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(arguments, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// requiredModules gives the paths that go.mod requires, directly or indirectly, and their versions.
func requiredModules(moduleRoot string) (paths []string, versions map[string]string, err error) {
	output, err := command(moduleRoot, "go", "mod", "edit", "-json")
	if err != nil {
		return nil, nil, err
	}
	var parsed struct {
		Require []struct{ Path, Version string }
	}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return nil, nil, fmt.Errorf("go mod edit -json: %w", err)
	}
	versions = map[string]string{}
	for _, required := range parsed.Require {
		paths = append(paths, required.Path)
		versions[required.Path] = required.Version
	}
	return paths, versions, nil
}

// moduleDirectories gives the directory of each module in the module cache, and none for a module that is not there.
func moduleDirectories(moduleRoot string, paths []string) (map[string]string, error) {
	// -e keeps go list from failing on a module that is not downloaded, which Generate reports by name.
	output, err := command(moduleRoot, "go", append([]string{"list", "-e", "-m", "-json"}, paths...)...)
	if err != nil {
		return nil, err
	}
	directories := map[string]string{}
	decoder := json.NewDecoder(strings.NewReader(output))
	for decoder.More() {
		var listed struct{ Path, Dir string }
		if err := decoder.Decode(&listed); err != nil {
			return nil, fmt.Errorf("go list -m -json: %w", err)
		}
		directories[listed.Path] = listed.Dir
	}
	return directories, nil
}

func readLicenceFiles(directory string) ([]File, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	var files []File
	for _, name := range LicenceFileNames(names) {
		text, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return nil, err
		}
		files = append(files, File{Name: name, Text: string(text)})
	}
	return files, nil
}
