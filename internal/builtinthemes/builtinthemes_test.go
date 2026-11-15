package builtinthemes

import (
	"bytes"
	"image/color"
	"image/png"
	"io/fs"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/builtinthemes/background"
	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/themefile"
)

const minimalDarkYAML = `name: Minimal Dark
author: sensordeck
license: MIT
background: background.png
fonts:
  label: {builtin: go-medium, size: 15, color: "#7f8ea3"}
  unit:  {builtin: go-medium, size: 15, color: "#4f5d70"}
  big:   {builtin: go-bold, size: 46, color: "#e6edf3"}
  small: {builtin: go-medium, size: 17, color: "#c9d4df"}
  clock: {builtin: go-bold, size: 24, color: "#e6edf3"}
  date:  {builtin: go-medium, size: 17, color: "#7f8ea3"}
widgets:
  - {type: text, font: label, text: CPU, x: 20, y: 18}
  - {type: text, font: unit, text: "°C", x: 136, y: 18}
  - {type: number, font: big, metric: cpu.temp, x: 20, y: 44}
  - {type: text, font: label, text: LOAD, x: 20, y: 102}
  - {type: number, font: small, metric: cpu.util, x: 70, y: 101}
  - {type: text, font: unit, text: "%", x: 140, y: 102}
  - {type: text, font: label, text: GPU, x: 176, y: 18}
  - {type: text, font: unit, text: "°C", x: 292, y: 18}
  - {type: number, font: big, metric: gpu.temp, x: 176, y: 44}
  - {type: text, font: label, text: LOAD, x: 176, y: 102}
  - {type: number, font: small, metric: gpu.util, x: 226, y: 101}
  - {type: text, font: unit, text: "%", x: 296, y: 102}
  - {type: text, font: label, text: RAM, x: 332, y: 18}
  - {type: text, font: unit, text: "%", x: 452, y: 18}
  - {type: number, font: big, metric: ram.util, x: 332, y: 44}
  - {type: text, font: label, text: USED, x: 332, y: 102}
  - {type: number, font: small, metric: ram.used, x: 382, y: 101}
  - {type: text, font: unit, text: GiB, x: 436, y: 102}
  - {type: text, font: label, text: VRAM, x: 20, y: 150}
  - {type: text, font: unit, text: MiB, x: 128, y: 150}
  - {type: number, font: big, metric: gpu.mem.used, digits: 5, x: 20, y: 176}
  - {type: text, font: label, text: TOTAL, x: 20, y: 234}
  - {type: number, font: small, metric: gpu.mem.total, digits: 5, x: 76, y: 233}
  - {type: text, font: label, text: DISK, x: 176, y: 150}
  - {type: text, font: unit, text: "°C", x: 292, y: 150}
  - {type: number, font: big, metric: nvme.1.temp, x: 176, y: 176}
  - {type: text, font: label, text: NETWORK, x: 332, y: 150}
  - {type: text, font: unit, text: KiB/s, x: 424, y: 150}
  - {type: text, font: label, text: UP, x: 332, y: 186}
  - {type: number, font: small, metric: net.up, digits: 5, x: 396, y: 185}
  - {type: text, font: label, text: DOWN, x: 332, y: 222}
  - {type: number, font: small, metric: net.down, digits: 5, x: 396, y: 221}
  - {type: datetime, font: clock, format: "HH:mm:ss", x: 20, y: 279}
  - {type: datetime, font: date, format: "YYYY-MM-DD", x: 364, y: 283}
`

const minimalDarkTags = `# Paste this block into the config file of sensordeck run.
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

func minimalDark(test *testing.T) fs.FS {
	test.Helper()
	folder, found := Folder("minimal-dark")
	if !found {
		test.Fatal("minimal-dark is not a built-in theme")
	}
	return folder
}

func TestTheBuiltInThemesAreMinimalDarkAlone(test *testing.T) {
	if got := Names(); !reflect.DeepEqual(got, []string{"minimal-dark"}) {
		test.Fatalf("names %q", got)
	}
	entries, err := fs.ReadDir(minimalDark(test), ".")
	if err != nil {
		test.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			test.Errorf("%s is not a regular file", entry.Name())
		}
		names = append(names, entry.Name())
	}
	if !reflect.DeepEqual(names, []string{"background.png", "theme.yaml"}) {
		test.Fatalf("files %q", names)
	}
}

func TestFolderIsFoundForADirectoryOfTheThemesAlone(test *testing.T) {
	for _, name := range []string{"missing", ".", "..", "", "minimal-dark/theme.yaml", "minimal-dark/", "themes"} {
		if _, found := Folder(name); found {
			test.Errorf("Folder(%q) is found", name)
		}
	}
}

func TestMinimalDarkHasTheThemeYamlOfItsSpec(test *testing.T) {
	got, err := fs.ReadFile(minimalDark(test), "theme.yaml")
	if err != nil {
		test.Fatal(err)
	}
	if string(got) != minimalDarkYAML {
		test.Fatal("theme.yaml is not the text of R2")
	}
}

func TestMinimalDarkBackgroundIsWhatTheGeneratorDraws(test *testing.T) {
	content, err := fs.ReadFile(minimalDark(test), "background.png")
	if err != nil {
		test.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(content))
	if err != nil {
		test.Fatal(err)
	}
	if decoded.Bounds().Dx() != 480 || decoded.Bounds().Dy() != 320 {
		test.Fatalf("bounds %v", decoded.Bounds())
	}
	drawn := background.MinimalDark()
	for row := 0; row < 320; row++ {
		for column := 0; column < 480; column++ {
			got := color.NRGBAModel.Convert(decoded.At(column, row)).(color.NRGBA)
			if want := drawn.NRGBAAt(column, row); got != want {
				test.Fatalf("pixel (%d, %d) is %v, the generator draws %v; run go generate ./internal/builtinthemes/", column, row, got, want)
			}
		}
	}
}

func TestMinimalDarkCompilesWithItsTags(test *testing.T) {
	result, err := themecompile.Compile(minimalDark(test))
	if err != nil {
		test.Fatal(err)
	}
	_, warnings, err := themefile.Parse(result.File)
	if err != nil || len(warnings) != 0 {
		test.Fatalf("read back: %v, warnings %q", err, warnings)
	}
	names := []string{"cpu.temp", "cpu.util", "gpu.mem.total", "gpu.mem.used", "gpu.temp", "gpu.util", "net.down", "net.up",
		"nvme.1.temp", "ram.used", "ram.util"}
	var want []config.Mapping
	for index, name := range names {
		want = append(want, config.Mapping{Name: name, Tag: uint8(index + 1)})
	}
	if !reflect.DeepEqual(result.Tags, want) {
		test.Fatalf("tags %v", result.Tags)
	}
	if got := string(themecompile.TagsFile(result.Tags)); got != minimalDarkTags {
		test.Fatalf("tags file %q", got)
	}
}
