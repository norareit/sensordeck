package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useHarness gives a harness whose config directory is a temporary one, and the path of its config file.
func useHarness(test *testing.T) (*harness, string) {
	test.Helper()
	harness := newHarness()
	harness.configDir = test.TempDir()
	harness.sourcesErr = os.ErrInvalid // theme use reads no sensor
	return harness, filepath.Join(harness.configDir, "sensordeck", "config.yaml")
}

func readText(test *testing.T, path string) string {
	test.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	return string(content)
}

func TestThemeUseCreatesTheConfigFileAndItsDirectories(test *testing.T) {
	harness, path := useHarness(test)
	if status := harness.run("theme", "use", "minimal-dark"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := readText(test, path); got != "theme: minimal-dark\n" {
		test.Errorf("the file holds %q", got)
	}
	if want := "theme minimal-dark set in " + path + "\n"; harness.stdout.String() != want {
		test.Errorf("stdout %q", harness.stdout.String())
	}
	if len(harness.opened) != 0 || harness.stderr.Len() != 0 {
		test.Errorf("opened %v, stderr %q", harness.opened, harness.stderr.String())
	}
}

func TestThemeUseTakesTheFileOfTheFlagAndFailsWithoutAConfigDirectory(test *testing.T) {
	harness := newHarness()
	path := filepath.Join(test.TempDir(), "a", "b.yaml")
	if status := harness.run("theme", "use", "--config", path, "minimal-dark"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := readText(test, path); got != "theme: minimal-dark\n" {
		test.Errorf("the file holds %q", got)
	}
	broken := newHarness()
	broken.configDirErr = os.ErrNotExist
	broken.expectFailure(test, "cannot find the config directory, name the file with --config: file does not exist", "theme", "use", "minimal-dark")
}

func TestThemeUseEditsOneLineAndKeepsEveryOtherByte(test *testing.T) {
	cases := []struct{ name, before, after string }{
		{"replaced", "# mine\ninterval: 2s\ntheme: old   # why\n# end\n", "# mine\ninterval: 2s\ntheme: minimal-dark\n# end\n"},
		{"added", "# mine\ninterval: 2s\n", "# mine\ninterval: 2s\ntheme: minimal-dark\n"},
		{"added without final line feed", "interval: 2s", "interval: 2s\ntheme: minimal-dark\n"},
		{"comments only", "# nothing yet\n", "# nothing yet\ntheme: minimal-dark\n"},
	}
	for _, each := range cases {
		harness, path := useHarness(test)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			test.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(each.before), 0o600); err != nil {
			test.Fatal(err)
		}
		if status := harness.run("theme", "use", "minimal-dark"); status != 0 {
			test.Fatalf("%s: status %d, stderr %q", each.name, status, harness.stderr.String())
		}
		if got := readText(test, path); got != each.after {
			test.Errorf("%s: the file holds %q", each.name, got)
		}
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			test.Errorf("%s: the permissions are %v", each.name, info.Mode().Perm())
		}
	}
}

func TestThemeUseLeavesTheFileAloneWhenItCannotOrWillNotEditIt(test *testing.T) {
	cases := []struct{ name, content, want string }{
		{"flow root", "{interval: 2s}\n", "config file %s: cannot be edited, set the key theme in it by hand"},
		{"tags", twoTags, "config file %s: theme and tags cannot both be given, the theme gives the tags"},
		{"not yaml", "a: [1\n  - x: {", "config file %s: yaml: "},
	}
	for _, each := range cases {
		harness, path := useHarness(test)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			test.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(each.content), 0o644); err != nil {
			test.Fatal(err)
		}
		if status := harness.run("theme", "use", "minimal-dark"); status != 1 {
			test.Fatalf("%s: status %d", each.name, status)
		}
		want := "sensordeck: " + strings.Replace(each.want, "%s", path, 1)
		if got := harness.stderr.String(); !strings.HasPrefix(got, want) {
			test.Errorf("%s: stderr %q, want %q", each.name, got, want)
		}
		if got := readText(test, path); got != each.content {
			test.Errorf("%s: the file was changed to %q", each.name, got)
		}
	}
}

func TestThemeUseRefusesAThemeThatDoesNotCompileAndWritesNothing(test *testing.T) {
	harness, path := useHarness(test)
	missing := filepath.Join(test.TempDir(), "nothing")
	harness.expectFailure(test, filepath.Join(missing, "theme.yaml")+": cannot read the file: no such file or directory", "theme", "use", missing)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		test.Errorf("the config file was created: %v", err)
	}
	other, otherPath := useHarness(test)
	other.expectFailure(test, "no installed or built-in theme is named nothing; a folder is named by a path, such as ./nothing", "theme", "use", "nothing")
	if _, err := os.Stat(otherPath); !os.IsNotExist(err) {
		test.Errorf("the config file was created: %v", err)
	}
}

func TestThemeUseWritesAFolderAsItsAbsoluteCleanedPathThatLoadConfigReadsBack(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	want := compileExpected(test, folder)
	working, _ := os.Getwd()
	relative, err := filepath.Rel(working, folder)
	if err != nil {
		test.Fatal(err)
	}
	for _, argument := range []string{relative, folder + "/../folder/"} {
		harness, path := useHarness(test)
		if status := harness.run("theme", "use", argument); status != 0 {
			test.Fatalf("%s: status %d, stderr %q", argument, status, harness.stderr.String())
		}
		loaded, err := newSession().loadConfig(path, []byte(readText(test, path)))
		if err != nil || string(loaded.theme.file) != string(want.File) {
			test.Errorf("%s: the file holds %q, %v", argument, readText(test, path), err)
		}
	}
	harness, path := useHarness(test)
	harness.run("theme", "use", relative)
	if !strings.Contains(readText(test, path), folder) {
		test.Errorf("the file holds %q, want the absolute path %s", readText(test, path), folder)
	}
}

func TestThemeUseQuotesAFolderPathThatNeedsIt(test *testing.T) {
	base := filepath.Join(test.TempDir(), "my themes #1")
	folder := filepath.Join(base, "it's")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		test.Fatal(err)
	}
	source := writeThemeFolder(test, compiledTheme)
	entries, _ := os.ReadDir(source)
	for _, entry := range entries {
		content, _ := os.ReadFile(filepath.Join(source, entry.Name()))
		if err := os.WriteFile(filepath.Join(folder, entry.Name()), content, 0o644); err != nil {
			test.Fatal(err)
		}
	}
	harness, path := useHarness(test)
	if status := harness.run("theme", "use", folder); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	loaded, err := newSession().loadConfig(path, []byte(readText(test, path)))
	if err != nil || loaded.theme.value != folder {
		test.Errorf("the file holds %q, %v", readText(test, path), err)
	}
}

func TestThemeUseWritesThroughASymbolicLink(test *testing.T) {
	harness, path := useHarness(test)
	target := filepath.Join(test.TempDir(), "dotfiles.yaml")
	if err := os.WriteFile(target, []byte("interval: 2s\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		test.Fatal(err)
	}
	if status := harness.run("theme", "use", "minimal-dark"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
		test.Errorf("the link was replaced: %v", err)
	}
	if got := readText(test, target); got != "interval: 2s\ntheme: minimal-dark\n" {
		test.Errorf("the target holds %q", got)
	}
}

func TestThemeUseAddsTheKeyToAnIndentedMapping(test *testing.T) {
	harness, path := useHarness(test)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("  interval: 2s\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	if status := harness.run("theme", "use", "minimal-dark"); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := readText(test, path); got != "  interval: 2s\n  theme: minimal-dark\n" {
		test.Errorf("the file holds %q", got)
	}
}
