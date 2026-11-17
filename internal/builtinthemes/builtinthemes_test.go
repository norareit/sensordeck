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

const minimalDarkProbesYAML = `name: Minimal Dark with probes
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
  - {type: text, font: label, text: DISK, x: 20, y: 234}
  - {type: number, font: small, metric: nvme.2.temp, x: 76, y: 233}
  - {type: text, font: unit, text: "°C", x: 136, y: 234}
  - {type: text, font: label, text: GPU CABLE, x: 176, y: 150}
  - {type: text, font: unit, text: "°C", x: 292, y: 150}
  - {type: number, font: big, metric: commander.1.temp, x: 176, y: 176}
  - {type: text, font: label, text: INTAKE, x: 176, y: 234}
  - {type: number, font: small, metric: commander.2.temp, x: 240, y: 233}
  - {type: text, font: unit, text: "°C", x: 292, y: 234}
  - {type: text, font: label, text: NETWORK, x: 332, y: 150}
  - {type: text, font: unit, text: KiB/s, x: 424, y: 150}
  - {type: text, font: label, text: UP, x: 332, y: 186}
  - {type: number, font: small, metric: net.up, digits: 5, x: 396, y: 185}
  - {type: text, font: label, text: DOWN, x: 332, y: 222}
  - {type: number, font: small, metric: net.down, digits: 5, x: 396, y: 221}
  - {type: datetime, font: clock, format: "HH:mm:ss", x: 20, y: 279}
  - {type: datetime, font: date, format: "YYYY-MM-DD", x: 364, y: 283}
`

const minimalDarkProbesTags = `# Paste this block into the config file of sensordeck run.
tags:
  commander.1.temp: 1
  commander.2.temp: 2
  cpu.temp: 3
  cpu.util: 4
  gpu.mem.used: 5
  gpu.temp: 6
  gpu.util: 7
  net.down: 8
  net.up: 9
  nvme.2.temp: 10
  ram.used: 11
  ram.util: 12
`

// builtInThemes is the table of the built-in themes, in the order of their names. A new built-in theme is a row.
var builtInThemes = []struct {
	name        string
	themeYAML   string
	metricNames []string
	tagsFile    string
}{
	{
		name:      "minimal-dark",
		themeYAML: minimalDarkYAML,
		metricNames: []string{"cpu.temp", "cpu.util", "gpu.mem.total", "gpu.mem.used", "gpu.temp", "gpu.util", "net.down",
			"net.up", "nvme.1.temp", "ram.used", "ram.util"},
		tagsFile: minimalDarkTags,
	},
	{
		name:      "minimal-dark-probes",
		themeYAML: minimalDarkProbesYAML,
		metricNames: []string{"commander.1.temp", "commander.2.temp", "cpu.temp", "cpu.util", "gpu.mem.used", "gpu.temp",
			"gpu.util", "net.down", "net.up", "nvme.2.temp", "ram.used", "ram.util"},
		tagsFile: minimalDarkProbesTags,
	},
}

func builtInFolder(test *testing.T, name string) fs.FS {
	test.Helper()
	folder, found := Folder(name)
	if !found {
		test.Fatalf("%s is not a built-in theme", name)
	}
	return folder
}

func TestTheBuiltInThemesAreTheRowsOfTheTable(test *testing.T) {
	var want []string
	for _, theme := range builtInThemes {
		want = append(want, theme.name)
	}
	if got := Names(); !reflect.DeepEqual(got, want) {
		test.Fatalf("names %q, want %q", got, want)
	}
}

func TestEachFolderHoldsTheRegularFilesOfATheme(test *testing.T) {
	for _, theme := range builtInThemes {
		test.Run(theme.name, func(test *testing.T) {
			entries, err := fs.ReadDir(builtInFolder(test, theme.name), ".")
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
		})
	}
}

func TestFolderIsFoundForADirectoryOfTheThemesAlone(test *testing.T) {
	for _, name := range []string{"missing", ".", "..", "", "minimal-dark/theme.yaml", "minimal-dark/", "themes"} {
		if _, found := Folder(name); found {
			test.Errorf("Folder(%q) is found", name)
		}
	}
}

func TestEachThemeHasItsThemeYaml(test *testing.T) {
	for _, theme := range builtInThemes {
		test.Run(theme.name, func(test *testing.T) {
			got, err := fs.ReadFile(builtInFolder(test, theme.name), "theme.yaml")
			if err != nil {
				test.Fatal(err)
			}
			if string(got) != theme.themeYAML {
				test.Fatal("theme.yaml is not the text of its specification")
			}
		})
	}
}

func TestEachBackgroundIsWhatTheGeneratorDraws(test *testing.T) {
	drawn := background.MinimalDark()
	for _, theme := range builtInThemes {
		test.Run(theme.name, func(test *testing.T) {
			content, err := fs.ReadFile(builtInFolder(test, theme.name), "background.png")
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
			for row := 0; row < 320; row++ {
				for column := 0; column < 480; column++ {
					got := color.NRGBAModel.Convert(decoded.At(column, row)).(color.NRGBA)
					if want := drawn.NRGBAAt(column, row); got != want {
						test.Fatalf("%s: pixel (%d, %d) is %v, the generator draws %v; run go generate ./internal/builtinthemes/",
							theme.name, column, row, got, want)
					}
				}
			}
		})
	}
}

func TestEachThemeCompilesWithItsTags(test *testing.T) {
	for _, theme := range builtInThemes {
		test.Run(theme.name, func(test *testing.T) {
			result, err := themecompile.Compile(builtInFolder(test, theme.name))
			if err != nil {
				test.Fatal(err)
			}
			_, warnings, err := themefile.Parse(result.File)
			if err != nil || len(warnings) != 0 {
				test.Fatalf("read back: %v, warnings %q", err, warnings)
			}
			var want []config.Mapping
			for index, name := range theme.metricNames {
				want = append(want, config.Mapping{Name: name, Tag: uint8(index + 1)})
			}
			if !reflect.DeepEqual(result.Tags, want) {
				test.Fatalf("tags %v", result.Tags)
			}
			if got := string(themecompile.TagsFile(result.Tags, result.Alarms)); got != theme.tagsFile {
				test.Fatalf("tags file %q", got)
			}
		})
	}
}
