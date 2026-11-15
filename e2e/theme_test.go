//go:build linux

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

const sampleWarning = "warning: slot 9: record type 0x77 is not known, kept as raw bytes\n"

var sampleFiles = []string{"preview.png", "slot01-background.png", "slot02-startup-frame1.png", "slot02-startup-frame2.png",
	"slot03-image-frame1.png", "slot03-image-frame2.png", "slot04-text.png", "slot05-number-glyphs.png",
	"slot06-datetime-glyphs.png", "theme.json"}

func writeThemeFile(test *testing.T, content []byte) string {
	test.Helper()
	path := filepath.Join(test.TempDir(), "img.dat")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		test.Fatal(err)
	}
	return path
}

func listDirectory(test *testing.T, directory string) []string {
	test.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		test.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func TestThemeDumpWritesTheFilesOfTheSampleWithNoScreenConnected(test *testing.T) {
	input := writeThemeFile(test, faketheme.Sample())
	output := filepath.Join(test.TempDir(), "out")
	got := run(test, fakescreen.New(test), "theme", "dump", input, output)
	if got.status != 0 || got.stdout != "wrote 10 files to "+output+"\n" || got.stderr != sampleWarning {
		test.Fatalf("status %d, stdout %q, stderr %q", got.status, got.stdout, got.stderr)
	}
	if names := listDirectory(test, output); !reflect.DeepEqual(names, sampleFiles) {
		test.Fatalf("files %q", names)
	}
	file, err := os.Open(filepath.Join(output, "preview.png"))
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	preview, err := png.Decode(file)
	if err != nil {
		test.Fatal(err)
	}
	if preview.Bounds().Dx() != 480 || preview.Bounds().Dy() != 320 {
		test.Errorf("preview bounds %v", preview.Bounds())
	}
	// The background of the sample is drawn first with its pixel 7 at 3,1, and the bar outlines its box in magenta.
	if got := color.NRGBAModel.Convert(preview.At(3, 1)).(color.NRGBA); got != (color.NRGBA{0, 227, 189, 255}) {
		test.Errorf("background pixel %v", got)
	}
	if got := color.NRGBAModel.Convert(preview.At(50, 50)).(color.NRGBA); got != (color.NRGBA{255, 0, 255, 255}) {
		test.Errorf("bar pixel %v", got)
	}
}

func TestThemeDumpJSONHoldsTheFieldsOfEachRecord(test *testing.T) {
	input := writeThemeFile(test, faketheme.Sample())
	output := filepath.Join(test.TempDir(), "out")
	if got := run(test, fakescreen.New(test), "theme", "dump", input, output); got.status != 0 {
		test.Fatalf("%+v", got)
	}
	content, err := os.ReadFile(filepath.Join(output, "theme.json"))
	if err != nil {
		test.Fatal(err)
	}
	var theme struct {
		FileSize int                          `json:"file_size"`
		Header   map[string]json.RawMessage   `json:"header"`
		Records  []map[string]json.RawMessage `json:"records"`
		Warnings []string                     `json:"warnings"`
	}
	if err := json.Unmarshal(content, &theme); err != nil {
		test.Fatal(err)
	}
	if theme.FileSize != len(faketheme.Sample()) || string(theme.Header["byte_0_unknown"]) != "150" || len(theme.Records) != 8 || len(theme.Warnings) != 1 {
		test.Fatalf("size %d, header %v, %d records, warnings %q", theme.FileSize, theme.Header, len(theme.Records), theme.Warnings)
	}
	wanted := []map[string]string{
		{"type_name": `"background"`, "offset": "4096", "fill_colour_guess": `"f79e"`, "files": `["slot01-background.png"]`},
		{"type_name": `"startup"`, "frame_count": "2", "duration_ms_guess": "1500", "files": `["slot02-startup-frame1.png","slot02-startup-frame2.png"]`},
		{"type_name": `"image"`, "frame_count": "2", "frame_period_ms_guess": "200"},
		{"type_name": `"text"`, "colour": `"ffff"`, "opacity_guess": "255", "files": `["slot04-text.png"]`},
		{"type_name": `"number"`, "tag": "1", "glyph_order": `"0123456789.-"`, "colour": `"07e0"`},
		{"type_name": `"datetime"`, "format": `"1-2-3 4:5:6"`, "other_glyph_order": `".-/ :年月日时分秒"`},
		{"type_name": `"bar"`, "files": "[]"},
		{"type_name": `"unknown"`, "type": "119", "files": "[]"},
	}
	for position, record := range theme.Records {
		for name, want := range wanted[position] {
			compact := new(bytes.Buffer)
			if err := json.Compact(compact, record[name]); err != nil {
				test.Fatalf("record %d, %s: %v", position, name, err)
			}
			if compact.String() != want {
				test.Errorf("record %d, %s is %s, want %s", position, name, compact.String(), want)
			}
		}
	}
	var widths []int
	if err := json.Unmarshal(theme.Records[4]["glyph_widths"], &widths); err != nil || len(widths) != 12 {
		test.Errorf("glyph_widths %s: %v", theme.Records[4]["glyph_widths"], err)
	}
	if string(theme.Records[6]["bytes_11_63_unknown"]) == "" {
		test.Error("the bar has no unknown bytes")
	}
}

func TestThemeDumpGivesIdenticalFilesOnASecondRunAndWithADeviceOption(test *testing.T) {
	input := writeThemeFile(test, faketheme.Sample())
	fake := fakescreen.New(test)
	first, second := filepath.Join(test.TempDir(), "first"), filepath.Join(test.TempDir(), "second")
	run(test, fake, "theme", "dump", input, first)
	got := run(test, fake, "--device", "/nonexistent", "theme", "dump", input, second)
	if got.status != 0 || got.stderr != sampleWarning {
		test.Fatalf("%+v", got)
	}
	for _, name := range sampleFiles {
		one, _ := os.ReadFile(filepath.Join(first, name))
		two, _ := os.ReadFile(filepath.Join(second, name))
		if len(one) == 0 || !bytes.Equal(one, two) {
			test.Errorf("%s differs between the runs", name)
		}
	}
}

func TestThemeDumpOfAShortFileFailsAndLeavesNoDirectory(test *testing.T) {
	input := writeThemeFile(test, make([]byte, 100))
	output := filepath.Join(test.TempDir(), "out")
	got := run(test, fakescreen.New(test), "theme", "dump", input, output)
	want := "sensordeck: cannot read the theme file " + input + ": the file is 100 bytes, shorter than the record table of 4096 bytes\n"
	if got.status != 1 || got.stdout != "" || got.stderr != want {
		test.Errorf("%+v", got)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		test.Error("the output directory exists")
	}
}

const folderTheme = `name: end to end
background: "#102030"
fonts:
  body: {file: body.ttf, size: 20, color: "#ffffff"}
  clock: {builtin: go-mono, size: 16, color: "#00c8ff"}
widgets:
  - {type: text, font: body, text: Hello, x: 4, y: 4}
  - {type: number, font: body, metric: gpu.temp, x: 4, y: 40}
  - {type: number, font: body, metric: cpu.temp, x: 100, y: 40}
  - {type: datetime, font: clock, format: "YYYY-MM-DD HH:mm:ss", x: 4, y: 100}
  - {type: image, file: dot.png, x: 200, y: 4}
`

func writeThemeFolder(test *testing.T, theme string) string {
	test.Helper()
	folder := filepath.Join(test.TempDir(), "folder")
	if err := os.Mkdir(folder, 0o755); err != nil {
		test.Fatal(err)
	}
	var dot bytes.Buffer
	if err := png.Encode(&dot, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		test.Fatal(err)
	}
	files := map[string][]byte{"theme.yaml": []byte(theme), "body.ttf": goregular.TTF, "dot.png": dot.Bytes()}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(folder, name), content, 0o644); err != nil {
			test.Fatal(err)
		}
	}
	return folder
}

func TestThemeCompileWritesTheThemeFileAndTheTagsFileWithNoScreen(test *testing.T) {
	folder := writeThemeFolder(test, folderTheme)
	directory := test.TempDir()
	out := filepath.Join(directory, "img.dat")
	got := run(test, fakescreen.New(test), "theme", "compile", folder, out)
	tagsFile := filepath.Join(directory, "img.tags.yaml")
	content, err := os.ReadFile(out)
	if err != nil {
		test.Fatal(err)
	}
	if want := fmt.Sprintf("wrote %s (%d bytes) and %s\n", out, len(content), tagsFile); got.status != 0 || got.stdout != want || got.stderr != "" {
		test.Fatalf("status %d, stdout %q, stderr %q", got.status, got.stdout, got.stderr)
	}
	theme, warnings, err := themefile.Parse(content)
	if err != nil || len(warnings) != 0 || len(theme.Entries) != 6 {
		test.Fatalf("%v, %v, %d entries", err, warnings, len(theme.Entries))
	}
	tags, err := os.ReadFile(tagsFile)
	if err != nil {
		test.Fatal(err)
	}
	parsed, err := config.Parse(tags)
	if err != nil || !reflect.DeepEqual(parsed.Mappings, []config.Mapping{{Name: "cpu.temp", Tag: 1}, {Name: "gpu.temp", Tag: 2}}) {
		test.Errorf("tags %+v, %v", parsed.Mappings, err)
	}
}

func TestThemePreviewIsThePreviewOfTheCompiledFile(test *testing.T) {
	folder := writeThemeFolder(test, folderTheme)
	directory := test.TempDir()
	out, preview := filepath.Join(directory, "img.dat"), filepath.Join(directory, "preview.png")
	if got := run(test, fakescreen.New(test), "theme", "compile", folder, out); got.status != 0 {
		test.Fatalf("compile: %+v", got)
	}
	got := run(test, fakescreen.New(test), "theme", "preview", folder, preview)
	if got.status != 0 || got.stdout != "wrote "+preview+"\n" || got.stderr != "" {
		test.Fatalf("status %d, stdout %q, stderr %q", got.status, got.stdout, got.stderr)
	}
	content, err := os.ReadFile(out)
	if err != nil {
		test.Fatal(err)
	}
	theme, _, err := themefile.Parse(content)
	if err != nil {
		test.Fatal(err)
	}
	var want bytes.Buffer
	if err := png.Encode(&want, themeimage.Preview(theme)); err != nil {
		test.Fatal(err)
	}
	if written, _ := os.ReadFile(preview); !bytes.Equal(written, want.Bytes()) {
		test.Error("the preview differs from the preview of the compiled file")
	}
}

func TestThemeCompileAndPreviewFailOnAFaultAndLeaveNoFile(test *testing.T) {
	folder := writeThemeFolder(test, "name: x\nbackground: \"#000000\"\ncolour: 1\n")
	for _, command := range []string{"compile", "preview"} {
		directory := test.TempDir()
		got := run(test, fakescreen.New(test), "theme", command, folder, filepath.Join(directory, "out"))
		want := "sensordeck: " + filepath.Join(folder, "theme.yaml") + ": unknown key \"colour\", the keys are author, background, fonts, license, name and widgets\n"
		if got.status != 1 || got.stdout != "" || got.stderr != want {
			test.Errorf("%s: status %d, stdout %q, stderr %q", command, got.status, got.stdout, got.stderr)
		}
		if names := listDirectory(test, directory); len(names) != 0 {
			test.Errorf("%s: left %q", command, names)
		}
	}
}

func TestThemeCompileAndPreviewIgnoreTheDeviceOption(test *testing.T) {
	folder := writeThemeFolder(test, folderTheme)
	for _, command := range []string{"compile", "preview"} {
		out := filepath.Join(test.TempDir(), "out")
		got := run(test, fakescreen.New(test), "--device", "/nonexistent", "theme", command, folder, out)
		if got.status != 0 || got.stderr != "" {
			test.Errorf("%s: status %d, stderr %q", command, got.status, got.stderr)
		}
	}
}
