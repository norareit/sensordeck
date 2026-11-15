package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inFixture makes a module with one licensed dependency, or with an unlicensed one, the working directory.
func inFixture(test *testing.T, licensed bool) {
	test.Helper()
	test.Setenv("GOPROXY", "off")
	root := test.TempDir()
	dependency := filepath.Join(root, "dependency")
	if err := os.Mkdir(dependency, 0o755); err != nil {
		test.Fatal(err)
	}
	files := map[string]string{"go.mod": "module example.com/dependency\n"}
	if licensed {
		files["LICENSE"] = "licence text\n"
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dependency, name), []byte(text), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	goMod := "module fixture\n\ngo 1.21\n\nrequire example.com/dependency v1.0.0\n\nreplace example.com/dependency => ./dependency\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		test.Fatal(err)
	}
	test.Chdir(root)
}

func TestRunWritesTheNoticesToTheOutputPath(test *testing.T) {
	inFixture(test, true)
	var stderr bytes.Buffer
	if status := run([]string{"out.txt"}, &stderr); status != 0 || stderr.Len() != 0 {
		test.Fatalf("status %d, stderr %q", status, stderr.String())
	}
	written, err := os.ReadFile("out.txt")
	if err != nil || !strings.Contains(string(written), "example.com/dependency v1.0.0") {
		test.Errorf("%q, %v", written, err)
	}
}

func TestRunNeedsExactlyOneArgument(test *testing.T) {
	for _, arguments := range [][]string{nil, {"one", "two"}} {
		var stderr bytes.Buffer
		if status := run(arguments, &stderr); status != 1 || !strings.Contains(stderr.String(), "one argument") {
			test.Errorf("%v: status %d, stderr %q", arguments, status, stderr.String())
		}
	}
}

func TestRunFailsForAnOutputPathThatCannotBeWritten(test *testing.T) {
	inFixture(test, true)
	var stderr bytes.Buffer
	if status := run([]string{filepath.Join("missing-directory", "out.txt")}, &stderr); status != 1 || stderr.Len() == 0 {
		test.Errorf("status %d, stderr %q", status, stderr.String())
	}
}

func TestRunLeavesAnExistingFileAsItWasWhenGenerationFails(test *testing.T) {
	inFixture(test, false)
	if err := os.WriteFile("out.txt", []byte("earlier"), 0o644); err != nil {
		test.Fatal(err)
	}
	var stderr bytes.Buffer
	status := run([]string{"out.txt"}, &stderr)
	kept, err := os.ReadFile("out.txt")
	if status != 1 || !strings.Contains(stderr.String(), "example.com/dependency") || string(kept) != "earlier" || err != nil {
		test.Errorf("status %d, stderr %q, file %q, %v", status, stderr.String(), kept, err)
	}
}
