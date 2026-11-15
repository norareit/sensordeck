//go:build linux

package e2e

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

const minimalDarkTagsFile = `# Paste this block into the config file of sensordeck run.
tags:
  cpu.temp: 1
  cpu.util: 2
  gpu.mem.total: 3
  gpu.mem.used: 4
  gpu.temp: 5
  gpu.util: 6
  net.down: 7
  net.up: 8
  nvme.1.temp: 9
  ram.used: 10
  ram.util: 11
`

// configHome gives a temporary config directory, the variable that makes it the one of the executable, and the themes
// directory in it, which is not created.
func configHome(test *testing.T) (variables []string, themes string) {
	test.Helper()
	home := test.TempDir()
	return []string{"XDG_CONFIG_HOME=" + home}, filepath.Join(home, "sensordeck", "themes")
}

// installFolder writes a theme folder that holds a one-colour theme under the path.
func installFolder(test *testing.T, path string) {
	test.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		test.Fatal(err)
	}
	theme := "name: night\nbackground: \"#102030\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\nwidgets:\n  - {type: text, font: body, text: Hi, x: 4, y: 4}\n"
	if err := os.WriteFile(filepath.Join(path, "theme.yaml"), []byte(theme), 0o644); err != nil {
		test.Fatal(err)
	}
}

func TestThemeListShowsTheBuiltInAndTheInstalledThemes(test *testing.T) {
	variables, themes := configHome(test)
	listed := func() string {
		got := runWith(test, fakescreen.New(test), variables, "theme", "list")
		if got.status != 0 || got.stderr != "" {
			test.Fatalf("status %d, stderr %q", got.status, got.stderr)
		}
		return got.stdout
	}
	if got := listed(); got != "minimal-dark\tbuilt-in\n" {
		test.Errorf("no themes directory: %q", got)
	}
	installFolder(test, filepath.Join(themes, "night"))
	want := "minimal-dark\tbuilt-in\nnight\tinstalled\t" + filepath.Join(themes, "night") + "\n"
	if got := listed(); got != want {
		test.Errorf("an installed theme: %q", got)
	}
	installFolder(test, filepath.Join(themes, "minimal-dark"))
	want = "minimal-dark\tinstalled, overrides built-in\t" + filepath.Join(themes, "minimal-dark") + "\nnight\tinstalled\t" + filepath.Join(themes, "night") + "\n"
	if got := listed(); got != want {
		test.Errorf("an installed theme that replaces the built-in one: %q", got)
	}
}

func TestThemeCompileAndPreviewTheBuiltInTheme(test *testing.T) {
	variables, _ := configHome(test)
	directory := test.TempDir()
	out := filepath.Join(directory, "img.dat")
	got := runWith(test, fakescreen.New(test), variables, "theme", "compile", "minimal-dark", out)
	content, err := os.ReadFile(out)
	if err != nil {
		test.Fatal(err)
	}
	tagsFile := filepath.Join(directory, "img.tags.yaml")
	if want := fmt.Sprintf("wrote %s (%d bytes) and %s\n", out, len(content), tagsFile); got.status != 0 || got.stdout != want || got.stderr != "" {
		test.Fatalf("status %d, stdout %q, stderr %q", got.status, got.stdout, got.stderr)
	}
	theme, warnings, err := themefile.Parse(content)
	if err != nil || len(warnings) != 0 {
		test.Fatalf("%v, %v", err, warnings)
	}
	if tags, err := os.ReadFile(tagsFile); err != nil || string(tags) != minimalDarkTagsFile {
		test.Errorf("tags %q, %v", tags, err)
	}

	preview := filepath.Join(directory, "preview.png")
	got = runWith(test, fakescreen.New(test), variables, "theme", "preview", "minimal-dark", preview)
	if got.status != 0 || got.stdout != "wrote "+preview+"\n" || got.stderr != "" {
		test.Fatalf("preview: status %d, stdout %q, stderr %q", got.status, got.stdout, got.stderr)
	}
	var want bytes.Buffer
	if err := png.Encode(&want, themeimage.Preview(theme)); err != nil {
		test.Fatal(err)
	}
	if written, err := os.ReadFile(preview); err != nil || !bytes.Equal(written, want.Bytes()) {
		test.Errorf("the preview is not that of the compiled file: %v", err)
	}
}

func TestAnInstalledThemeIsCompiledByItsNameAndAFolderByItsPath(test *testing.T) {
	variables, themes := configHome(test)
	installFolder(test, filepath.Join(themes, "night"))
	byName := filepath.Join(test.TempDir(), "name.dat")
	if got := runWith(test, fakescreen.New(test), variables, "theme", "compile", "night", byName); got.status != 0 {
		test.Fatalf("by name: %+v", got)
	}
	byPath := filepath.Join(test.TempDir(), "path.dat")
	if got := runIn(test, fakescreen.New(test), themes, variables, "theme", "compile", "./night", byPath); got.status != 0 {
		test.Fatalf("by path: %+v", got)
	}
	for _, pair := range [][2]string{{byName, byPath}, {tagsFilePath(byName), tagsFilePath(byPath)}} {
		first, err := os.ReadFile(pair[0])
		if err != nil {
			test.Fatal(err)
		}
		second, err := os.ReadFile(pair[1])
		if err != nil {
			test.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			test.Errorf("%s and %s differ", pair[0], pair[1])
		}
	}
}

func tagsFilePath(out string) string {
	return out[:len(out)-len(filepath.Ext(out))] + ".tags.yaml"
}

func TestANameFoundNowhereIsAnErrorAndWritesNothing(test *testing.T) {
	variables, _ := configHome(test)
	directory := test.TempDir()
	got := runWith(test, fakescreen.New(test), variables, "theme", "compile", "nowhere", filepath.Join(directory, "img.dat"))
	want := "sensordeck: no installed or built-in theme is named nowhere; a folder is named by a path, such as ./nowhere\n"
	if got.status != 1 || got.stdout != "" || got.stderr != want {
		test.Fatalf("status %d, stdout %q, stderr %q", got.status, got.stdout, got.stderr)
	}
	if names := listDirectory(test, directory); len(names) != 0 {
		test.Errorf("files %q", names)
	}
}

func TestThemeCommandsByNameNeedNoScreen(test *testing.T) {
	variables, _ := configHome(test)
	directory := test.TempDir()
	for _, command := range [][]string{{"list"}, {"compile", "minimal-dark", filepath.Join(directory, "img.dat")},
		{"preview", "minimal-dark", filepath.Join(directory, "preview.png")}} {
		args := append([]string{"theme"}, command...)
		args = append(args, "--device", "/nonexistent")
		if got := runWith(test, fakescreen.New(test), variables, args...); got.status != 0 || got.stderr != "" {
			test.Errorf("%v: status %d, stderr %q", args, got.status, got.stderr)
		}
	}
}
