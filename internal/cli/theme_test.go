package cli

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/themedump"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
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

const compiledTheme = `name: cli
background: "#102030"
fonts:
  body: {file: body.ttf, size: 20, color: "#ffffff"}
widgets:
  - {type: text, font: body, text: Hello, x: 4, y: 4}
  - {type: number, font: body, metric: cpu.temp, x: 4, y: 40}
  - {type: image, file: dot.png, x: 100, y: 4}
`

func pngBytes(test *testing.T, width, height int) []byte {
	test.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	picture.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		test.Fatal(err)
	}
	return buffer.Bytes()
}

// writeThemeFolder writes a theme folder with the text as its theme.yaml, and gives its path.
func writeThemeFolder(test *testing.T, theme string) string {
	test.Helper()
	folder := filepath.Join(test.TempDir(), "folder")
	if err := os.Mkdir(folder, 0o755); err != nil {
		test.Fatal(err)
	}
	files := map[string][]byte{"theme.yaml": []byte(theme), "body.ttf": goregular.TTF, "dot.png": pngBytes(test, 4, 4)}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(folder, name), content, 0o644); err != nil {
			test.Fatal(err)
		}
	}
	return folder
}

func compileExpected(test *testing.T, folder string) themecompile.Result {
	test.Helper()
	result, err := compileFolder(folder)
	if err != nil {
		test.Fatal(err)
	}
	return result
}

func TestThemeCompileWritesTheThemeFileAndTheTagsFile(test *testing.T) {
	for _, extra := range [][]string{nil, {"--device", "/nowhere"}} {
		folder := writeThemeFolder(test, compiledTheme)
		want := compileExpected(test, folder)
		for out, tags := range map[string]string{"img.dat": "img.tags.yaml", "theme": "theme.tags.yaml"} {
			directory := test.TempDir()
			outPath, tagsPath := filepath.Join(directory, out), filepath.Join(directory, tags)
			for _, stale := range []string{outPath, tagsPath} {
				if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
					test.Fatal(err)
				}
			}
			harness := newHarness()
			harness.sourcesErr = errors.New("sources must not be asked for")
			args := append(append([]string{}, extra...), "theme", "compile", folder, outPath)
			if status := harness.run(args...); status != 0 {
				test.Fatalf("%v: status %d, stderr %q", args, status, harness.stderr.String())
			}
			if line := fmt.Sprintf("wrote %s (%d bytes) and %s\n", outPath, len(want.File), tagsPath); harness.stdout.String() != line {
				test.Errorf("stdout %q, want %q", harness.stdout.String(), line)
			}
			gotFile, _ := os.ReadFile(outPath)
			gotTags, _ := os.ReadFile(tagsPath)
			if !bytes.Equal(gotFile, want.File) || !bytes.Equal(gotTags, themecompile.TagsFile(want.Tags)) {
				test.Errorf("%s: the files are not what the compiler gives", out)
			}
			if len(harness.opened) != 0 || harness.stderr.Len() != 0 {
				test.Errorf("opened %v, stderr %q", harness.opened, harness.stderr.String())
			}
		}
	}
}

func TestTagsPath(test *testing.T) {
	cases := map[string]string{"img.dat": "img.tags.yaml", "theme": "theme.tags.yaml", "a.b/theme": "a.b/theme.tags.yaml", "x.tar.dat": "x.tar.tags.yaml"}
	for out, want := range cases {
		if got := tagsPath(filepath.FromSlash(out)); got != filepath.FromSlash(want) {
			test.Errorf("%s: got %s, want %s", out, got, want)
		}
	}
}

func TestThemePreviewWritesThePreviewOfTheCompiledFile(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	directory := test.TempDir()
	out := filepath.Join(directory, "preview.png")
	harness := newHarness()
	if status := harness.run("theme", "preview", folder, out); status != 0 {
		test.Fatalf("status %d, stderr %q", status, harness.stderr.String())
	}
	if harness.stdout.String() != "wrote "+out+"\n" || len(harness.opened) != 0 {
		test.Errorf("stdout %q", harness.stdout.String())
	}
	theme, _, err := themefile.Parse(compileExpected(test, folder).File)
	if err != nil {
		test.Fatal(err)
	}
	var want bytes.Buffer
	if err := png.Encode(&want, themeimage.Preview(theme)); err != nil {
		test.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if !bytes.Equal(got, want.Bytes()) {
		test.Error("the preview is not the preview of the compiled file")
	}
	if entries, _ := os.ReadDir(directory); len(entries) != 1 {
		test.Errorf("%d files written", len(entries))
	}
}

func TestThemeCompileAndPreviewFailOnAFaultOfTheFolderAndWriteNothing(test *testing.T) {
	folder := writeThemeFolder(test, "name: cli\nbackground: \"#102030\"\nwidgets:\n  - {type: text, font: body, text: a, x: 0, y: 0}\n")
	want := filepath.Join(folder, "theme.yaml") + `: widget 1 (text): font: "body" is not a font of fonts`
	for _, command := range []string{"compile", "preview"} {
		directory := test.TempDir()
		out := filepath.Join(directory, "out")
		newHarness().expectFailure(test, want, "theme", command, folder, out)
		if entries, _ := os.ReadDir(directory); len(entries) != 0 {
			test.Errorf("%s: wrote %d files", command, len(entries))
		}
	}
}

func TestThemeCompileNamesWhatItCannotRead(test *testing.T) {
	missing := filepath.Join(test.TempDir(), "nowhere")
	newHarness().expectFailure(test, filepath.Join(missing, "theme.yaml")+": cannot read the file: no such file or directory",
		"theme", "compile", missing, filepath.Join(test.TempDir(), "out.dat"))
}

func TestThemeCompileNamesThePathItCannotWrite(test *testing.T) {
	folder := writeThemeFolder(test, compiledTheme)
	below := filepath.Join(test.TempDir(), "missing", "img.dat")
	newHarness().expectFailure(test, "cannot write the file "+below+": no such file or directory", "theme", "compile", folder, below)
	if _, err := os.Stat(filepath.Dir(below)); !os.IsNotExist(err) {
		test.Error("the directory was created")
	}
}

func TestThemeFolderRefusesALinkOutOfTheFolder(test *testing.T) {
	outside := filepath.Join(test.TempDir(), "outside.png")
	if err := os.WriteFile(outside, pngBytes(test, 480, 320), 0o644); err != nil {
		test.Fatal(err)
	}
	folder := writeThemeFolder(test, "name: cli\nbackground: link.png\n")
	if err := os.Symlink(outside, filepath.Join(folder, "link.png")); err != nil {
		test.Skip("no symbolic links here: ", err)
	}
	newHarness().expectFailure(test, filepath.Join(folder, "theme.yaml")+": background: cannot read link.png: path escapes from parent",
		"theme", "compile", folder, filepath.Join(test.TempDir(), "out.dat"))

	inside := writeThemeFolder(test, "name: cli\nbackground: link.png\n")
	if err := os.WriteFile(filepath.Join(inside, "real.png"), pngBytes(test, 480, 320), 0o644); err != nil {
		test.Fatal(err)
	}
	if err := os.Symlink("real.png", filepath.Join(inside, "link.png")); err != nil {
		test.Fatal(err)
	}
	harness := newHarness()
	if status := harness.run("theme", "compile", inside, filepath.Join(test.TempDir(), "out.dat")); status != 0 {
		test.Errorf("a link inside the folder: status %d, %s", status, harness.stderr.String())
	}
}

func TestThemeCompileGivesTheSameBytesForACopyOfTheFolder(test *testing.T) {
	first, second := writeThemeFolder(test, compiledTheme), writeThemeFolder(test, compiledTheme)
	var files [2][2][]byte
	for position, folder := range []string{first, second} {
		out := filepath.Join(test.TempDir(), "img.dat")
		if status := newHarness().run("theme", "compile", folder, out); status != 0 {
			test.Fatalf("status %d", status)
		}
		files[position][0], _ = os.ReadFile(out)
		files[position][1], _ = os.ReadFile(tagsPath(out))
	}
	if !bytes.Equal(files[0][0], files[1][0]) || !bytes.Equal(files[0][1], files[1][1]) {
		test.Error("the copies compile differently")
	}
}
