package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themedump"
)

func writeSample(test *testing.T) string {
	test.Helper()
	path := filepath.Join(test.TempDir(), "img.dat")
	if err := os.WriteFile(path, faketheme.Sample(), 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

func TestThemeDumpWritesTheFilesOfTheDump(test *testing.T) {
	for _, extra := range [][]string{nil, {"--device", "/nowhere"}} {
		input := writeSample(test)
		output := filepath.Join(test.TempDir(), "a", "b")
		harness := newHarness()
		args := append(append([]string{}, extra...), "theme", "dump", input, output)
		if status := harness.run(args...); status != 0 {
			test.Fatalf("%v: status %d, stderr %q", args, status, harness.stderr.String())
		}
		if want := fmt.Sprintf("wrote 10 files to %s\n", output); harness.stdout.String() != want {
			test.Errorf("stdout %q, want %q", harness.stdout.String(), want)
		}
		if want := "warning: slot 9: record type 0x77 is not known, kept as raw bytes\n"; harness.stderr.String() != want {
			test.Errorf("stderr %q", harness.stderr.String())
		}
		if len(harness.opened) != 0 {
			test.Errorf("%v: opened the transport", args)
		}
		files, _, err := themedump.Dump(faketheme.Sample())
		if err != nil {
			test.Fatal(err)
		}
		entries, err := os.ReadDir(output)
		if err != nil || len(entries) != len(files) {
			test.Fatalf("%d files in the output, %v", len(entries), err)
		}
		for _, file := range files {
			content, err := os.ReadFile(filepath.Join(output, file.Name))
			if err != nil || !bytes.Equal(content, file.Content) {
				test.Errorf("%s is not as the dump gives it: %v", file.Name, err)
			}
		}
	}
}

func TestThemeDumpReplacesItsFilesAndLeavesOthers(test *testing.T) {
	input := writeSample(test)
	output := test.TempDir()
	for name, content := range map[string]string{"theme.json": "old", "keep.txt": "kept"} {
		if err := os.WriteFile(filepath.Join(output, name), []byte(content), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	harness := newHarness()
	if status := harness.run("theme", "dump", input, output); status != 0 {
		test.Fatalf("status %d: %s", status, harness.stderr.String())
	}
	replaced, _ := os.ReadFile(filepath.Join(output, "theme.json"))
	kept, _ := os.ReadFile(filepath.Join(output, "keep.txt"))
	if string(replaced) == "old" || !strings.HasPrefix(string(replaced), "{") || string(kept) != "kept" {
		test.Errorf("theme.json %q, keep.txt %q", replaced, kept)
	}
}

func TestThemeDumpFailsWithoutWritingAnythingOnAFileItCannotRead(test *testing.T) {
	directory := test.TempDir()
	missing := filepath.Join(directory, "missing.dat")
	short := filepath.Join(directory, "short.dat")
	if err := os.WriteFile(short, make([]byte, 100), 0o644); err != nil {
		test.Fatal(err)
	}
	cases := map[string]string{
		missing: "cannot read the theme file " + missing + ": no such file or directory",
		short:   "cannot read the theme file " + short + ": the file is 100 bytes, shorter than the record table of 4096 bytes",
	}
	for input, want := range cases {
		output := filepath.Join(directory, "out")
		harness := newHarness()
		harness.expectFailure(test, want, "theme", "dump", input, output)
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			test.Errorf("%s: the output directory was created", input)
		}
	}
}

func TestThemeDumpNamesThePathItCannotWrite(test *testing.T) {
	input := writeSample(test)
	directory := test.TempDir()
	regular := filepath.Join(directory, "regular")
	if err := os.WriteFile(regular, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	below := filepath.Join(regular, "out")
	newHarness().expectFailure(test, "cannot create the output directory "+below+": not a directory", "theme", "dump", input, below)

	blocked := filepath.Join(directory, "blocked")
	if err := os.MkdirAll(filepath.Join(blocked, "theme.json"), 0o755); err != nil {
		test.Fatal(err)
	}
	newHarness().expectFailure(test, "cannot write the file "+filepath.Join(blocked, "theme.json")+": is a directory", "theme", "dump", input, blocked)
}
