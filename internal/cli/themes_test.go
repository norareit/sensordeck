package cli

import (
	"errors"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/builtinthemes"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

// themesHarness gives a harness whose config directory is a temporary one, and the themes directory in it, which is
// not created.
func themesHarness(test *testing.T) (*harness, string) {
	test.Helper()
	harness := newHarness()
	harness.configDir = test.TempDir()
	return harness, filepath.Join(harness.configDir, "sensordeck", "themes")
}

// installTheme copies a theme folder written by writeThemeFolder into the themes directory under the name.
func installTheme(test *testing.T, themes, name, theme string) string {
	test.Helper()
	source := writeThemeFolder(test, theme)
	target := filepath.Join(themes, name)
	if err := os.CopyFS(target, os.DirFS(source)); err != nil {
		test.Fatal(err)
	}
	return target
}

func builtinResult(test *testing.T) themecompile.Result {
	test.Helper()
	folder, _ := builtinthemes.Folder("minimal-dark")
	result, err := themecompile.Compile(folder)
	if err != nil {
		test.Fatal(err)
	}
	return result
}

func readFile(test *testing.T, path string) []byte {
	test.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	return content
}

func TestIsFolderPathTellsAPathFromAName(test *testing.T) {
	for argument, want := range map[string]bool{".": true, "..": true, "./x": true, "a/b": true, "/abs": true,
		"x": false, "x.y": false, "minimal-dark": false} {
		if got := isFolderPath(argument); got != want {
			test.Errorf("isFolderPath(%q) = %v", argument, got)
		}
	}
}

func TestThemeCommandsRefuseTheEmptyArgument(test *testing.T) {
	for _, command := range []string{"compile", "preview"} {
		harness, _ := themesHarness(test)
		out := filepath.Join(test.TempDir(), "out")
		harness.expectFailure(test, "a theme name or folder is needed", "theme", command, "", out)
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			test.Errorf("%s: a file was written", command)
		}
	}
}

func TestAnInstalledThemeIsCompiledByItsName(test *testing.T) {
	harness, themes := themesHarness(test)
	path := installTheme(test, themes, "night", compiledTheme)
	want := compileExpected(test, path)
	out := filepath.Join(test.TempDir(), "img.dat")
	if status := harness.run("theme", "compile", "night", out); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := readFile(test, out); !reflect.DeepEqual(got, want.File) {
		test.Error("the theme file differs from the one of the folder")
	}
	if got := readFile(test, tagsPath(out)); !reflect.DeepEqual(got, themecompile.TagsFile(want.Tags)) {
		test.Error("the tags file differs from the one of the folder")
	}
}

func TestAFaultOfAnInstalledThemeNamesItsThemeYaml(test *testing.T) {
	harness, themes := themesHarness(test)
	path := installTheme(test, themes, "broken", "name: [")
	out := filepath.Join(test.TempDir(), "img.dat")
	if status := harness.run("theme", "compile", "broken", out); status != 1 {
		test.Fatalf("status %d", status)
	}
	if got := harness.stderr.String(); !strings.HasPrefix(got, "sensordeck: "+filepath.Join(path, "theme.yaml")+": ") {
		test.Errorf("stderr %q", got)
	}
}

func TestABuiltInThemeIsCompiledByItsNameWithNoThemesDirectory(test *testing.T) {
	harness, _ := themesHarness(test)
	want := builtinResult(test)
	out := filepath.Join(test.TempDir(), "img.dat")
	if status := harness.run("theme", "compile", "minimal-dark", out); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if !reflect.DeepEqual(readFile(test, out), want.File) || !reflect.DeepEqual(readFile(test, tagsPath(out)), themecompile.TagsFile(want.Tags)) {
		test.Error("the files differ from the compilation of the built-in folder")
	}
}

func TestTheBuiltInThemeCompilesLikeItsFilesInAFolder(test *testing.T) {
	builtin, _ := builtinthemes.Folder("minimal-dark")
	folder := test.TempDir()
	for _, name := range []string{"theme.yaml", "background.png"} {
		content, err := fs.ReadFile(builtin, name)
		if err != nil {
			test.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, name), content, 0o644); err != nil {
			test.Fatal(err)
		}
	}
	fromFolder, fromName := filepath.Join(test.TempDir(), "folder.dat"), filepath.Join(test.TempDir(), "name.dat")
	harness, _ := themesHarness(test)
	if harness.run("theme", "compile", folder, fromFolder) != 0 || harness.run("theme", "compile", "minimal-dark", fromName) != 0 {
		test.Fatalf("stderr %q", harness.stderr.String())
	}
	for _, pair := range [][2]string{{fromFolder, fromName}, {tagsPath(fromFolder), tagsPath(fromName)}} {
		if !reflect.DeepEqual(readFile(test, pair[0]), readFile(test, pair[1])) {
			test.Errorf("%s and %s differ", pair[0], pair[1])
		}
	}
}

func TestAnInstalledThemeReplacesABuiltInOne(test *testing.T) {
	harness, themes := themesHarness(test)
	path := installTheme(test, themes, "minimal-dark", compiledTheme)
	want := compileExpected(test, path)
	out := filepath.Join(test.TempDir(), "img.dat")
	if status := harness.run("theme", "compile", "minimal-dark", out); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := readFile(test, out); !reflect.DeepEqual(got, want.File) {
		test.Error("the installed theme was not the one compiled")
	}
}

func TestASymbolicLinkToAFolderIsAnInstalledTheme(test *testing.T) {
	harness, themes := themesHarness(test)
	if err := os.MkdirAll(themes, 0o755); err != nil {
		test.Fatal(err)
	}
	folder := writeThemeFolder(test, compiledTheme)
	if err := os.Symlink(folder, filepath.Join(themes, "linked")); err != nil {
		test.Skip("symbolic links cannot be made here")
	}
	out := filepath.Join(test.TempDir(), "img.dat")
	if status := harness.run("theme", "compile", "linked", out); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := readFile(test, out); !reflect.DeepEqual(got, compileExpected(test, folder).File) {
		test.Error("the linked theme was not compiled")
	}
}

func TestAFileInTheThemesDirectoryIsNoTheme(test *testing.T) {
	harness, themes := themesHarness(test)
	if err := os.MkdirAll(themes, 0o755); err != nil {
		test.Fatal(err)
	}
	for _, name := range []string{"minimal-dark", "other"} {
		if err := os.WriteFile(filepath.Join(themes, name), nil, 0o644); err != nil {
			test.Fatal(err)
		}
	}
	out := filepath.Join(test.TempDir(), "img.dat")
	if status := harness.run("theme", "compile", "minimal-dark", out); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if got := readFile(test, out); !reflect.DeepEqual(got, builtinResult(test).File) {
		test.Error("the built-in theme was not compiled")
	}
	other := newHarness()
	other.configDir = harness.configDir
	other.expectFailure(test, "no installed or built-in theme is named other; a folder is named by a path, such as ./other",
		"theme", "compile", "other", filepath.Join(test.TempDir(), "x.dat"))
}

func TestANameFoundNowhereIsAnErrorAndWritesNothing(test *testing.T) {
	for _, command := range []string{"compile", "preview"} {
		harness, _ := themesHarness(test)
		out := filepath.Join(test.TempDir(), "out")
		harness.expectFailure(test, "no installed or built-in theme is named nowhere; a folder is named by a path, such as ./nowhere",
			"theme", command, "nowhere", out)
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			test.Errorf("%s: a file was written", command)
		}
	}
}

func TestAMissingConfigDirectoryFailsANameButNotAFolder(test *testing.T) {
	harness := newHarness()
	harness.configDirErr = errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
	out := filepath.Join(test.TempDir(), "img.dat")
	harness.expectFailure(test, "cannot find the config directory: neither $XDG_CONFIG_HOME nor $HOME are defined",
		"theme", "compile", "minimal-dark", out)
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		test.Error("a file was written")
	}
	folder := writeThemeFolder(test, compiledTheme)
	if status := harness.run("theme", "compile", folder, out); status != 0 {
		test.Errorf("a folder: status %d, stderr %q", status, harness.stderr.String())
	}
}

func TestAThemesDirectoryThatIsAFileFailsAName(test *testing.T) {
	harness, themes := themesHarness(test)
	if err := os.MkdirAll(filepath.Dir(themes), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(themes, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	harness.expectFailure(test, "cannot examine the installed theme "+filepath.Join(themes, "night")+": not a directory",
		"theme", "compile", "night", filepath.Join(test.TempDir(), "img.dat"))
}

func TestThemePreviewDrawsABuiltInTheme(test *testing.T) {
	harness, _ := themesHarness(test)
	out := filepath.Join(test.TempDir(), "preview.png")
	if status := harness.run("theme", "preview", "minimal-dark", out); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	theme, _, err := themefile.Parse(builtinResult(test).File)
	if err != nil {
		test.Fatal(err)
	}
	var want strings.Builder
	if err := png.Encode(&want, themeimage.Preview(theme)); err != nil {
		test.Fatal(err)
	}
	if string(readFile(test, out)) != want.String() {
		test.Error("the preview differs")
	}
}

func TestThemeCommandsByNameOpenNoDevice(test *testing.T) {
	for _, extra := range [][]string{nil, {"--device", "/nowhere"}} {
		for _, command := range [][]string{{"list"}, {"compile", "minimal-dark", "img.dat"}, {"preview", "minimal-dark", "img.png"}} {
			harness, _ := themesHarness(test)
			harness.sourcesErr = errors.New("sensors were read")
			harness.openErr = errors.New("a device was opened")
			args := append(append([]string{"theme"}, command...), extra...)
			if len(command) == 3 {
				args[3] = filepath.Join(test.TempDir(), command[2])
			}
			if status := harness.run(args...); status != 0 {
				test.Errorf("%v: status %d, stderr %q", args, status, harness.stderr.String())
			}
			if len(harness.opened) != 0 {
				test.Errorf("%v: opened %q", args, harness.opened)
			}
		}
	}
}

func TestThemeListLinesAreSortedByName(test *testing.T) {
	builtin := []string{"minimal-dark"}
	cases := []struct {
		name      string
		installed map[string]string
		want      []string
	}{
		{"none installed", nil, []string{"minimal-dark\tbuilt-in"}},
		{"one sorting first", map[string]string{"night": "/t/night"},
			[]string{"minimal-dark\tbuilt-in", "night\tinstalled\t/t/night"}},
		{"one sorting before", map[string]string{"alpha": "/t/alpha"},
			[]string{"alpha\tinstalled\t/t/alpha", "minimal-dark\tbuilt-in"}},
		{"one overriding", map[string]string{"minimal-dark": "/t/minimal-dark"},
			[]string{"minimal-dark\tinstalled, overrides built-in\t/t/minimal-dark"}},
	}
	for _, each := range cases {
		if got := themeListLines(builtin, each.installed); !reflect.DeepEqual(got, each.want) {
			test.Errorf("%s: %q", each.name, got)
		}
	}
}

func TestThemeListShowsWhatTheLookupFinds(test *testing.T) {
	harness, themes := themesHarness(test)
	if status := harness.run("theme", "list"); status != 0 || harness.stdout.String() != "minimal-dark\tbuilt-in\n" {
		test.Fatalf("no themes directory: status %d, stdout %q", status, harness.stdout.String())
	}

	night := installTheme(test, themes, "night", compiledTheme)
	if err := os.WriteFile(filepath.Join(themes, "notes"), nil, 0o644); err != nil {
		test.Fatal(err)
	}
	linked := filepath.Join(themes, "linked")
	if err := os.Symlink(night, linked); err != nil {
		linked = ""
	}
	override := installTheme(test, themes, "minimal-dark", compiledTheme)
	want := "linked\tinstalled\t" + linked + "\nminimal-dark\tinstalled, overrides built-in\t" + override + "\nnight\tinstalled\t" + night + "\n"
	if linked == "" {
		want = strings.SplitN(want, "\n", 2)[1]
	}
	harness = newHarness()
	harness.configDir = filepath.Dir(filepath.Dir(themes))
	if status := harness.run("theme", "list"); status != 0 || harness.stdout.String() != want {
		test.Fatalf("status %d, stdout %q, want %q", status, harness.stdout.String(), want)
	}
}

func TestThemeListFailsWhereTheLookupFails(test *testing.T) {
	harness := newHarness()
	harness.configDirErr = errors.New("no home")
	harness.expectFailure(test, "cannot find the config directory: no home", "theme", "list")

	harness, themes := themesHarness(test)
	if err := os.MkdirAll(filepath.Dir(themes), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(themes, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	harness.expectFailure(test, "cannot read the themes directory "+themes+": not a directory", "theme", "list")

	harness, _ = themesHarness(test)
	if status := harness.run("theme", "list", "extra"); status != 1 {
		test.Errorf("an argument: status %d", status)
	}
}
