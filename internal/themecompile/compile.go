// Package themecompile compiles a theme folder into the bytes of a theme file. It reads the folder through an fs.FS and
// does no other I/O, so that the command line decides what the folder is and where the output goes. The folder format
// is described in docs/theme-folder.md.
package themecompile

import (
	"fmt"
	"image"
	"io/fs"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/themefile"
)

// FlashLimit is the size of the flash of the device, which a theme file must fit in.
const FlashLimit = 4 << 20

// Result is a compiled theme.
type Result struct {
	File    []byte             // the bytes of the theme file
	Records []themefile.Record // slots 1 and up in the order of the slots, each with its offset and resource
	Tags    []config.Mapping   // the plain entries, ascending by tag
	Alarms  []config.AlarmPair // ascending by normal tag
	// AlarmSlots are the slots of the alarm widgets of the alarm numbers, in the numbering of themefile.Entry.Slot.
	AlarmSlots []int
}

// Compile compiles the theme folder. Its error is "<place>: <problem>", or "<problem>" for a fault of theme.yaml as a
// whole, and names no path of the folder other than the paths the theme gives. Each step stops at its first fault, so
// that a folder with several faults gives the same one each time.
func Compile(folder fs.FS) (Result, error) {
	content, err := fs.ReadFile(folder, "theme.yaml")
	if err != nil {
		return Result{}, fmt.Errorf("cannot read the file: %s", cause(err))
	}
	parsed, err := parseDocument(content)
	if err != nil {
		return Result{}, err
	}
	picture, err := loadBackground(folder, parsed.background)
	if err != nil {
		return Result{}, err
	}
	fonts, err := loadFonts(folder, parsed.fonts)
	if err != nil {
		return Result{}, err
	}
	tags, alarms := assignTags(parsed.widgets)
	records := []built{buildBackground(parsed.background, picture)}
	offset := backgroundOffset + len(records[0].resource)
	var alarmSlots []int
	for _, spec := range parsed.widgets {
		input := widgetInput{spec: spec, font: fontOf(fonts, spec), folder: folder, tag: plainTagOf(tags, spec.metric),
			slot: len(records) + 1, offset: uint32(offset)}
		if spec.alarm != nil {
			pair := pairOf(alarms, spec.metric, spec.alarm.threshold)
			input.tag, input.alarmTag = pair.Normal, pair.Alarm
			alarmSlots = append(alarmSlots, input.slot+1)
		}
		widgets, err := buildWidget(input)
		if err != nil {
			return Result{}, err
		}
		for _, widget := range widgets {
			records = append(records, widget)
			offset += len(widget.resource)
		}
	}
	file := layout(records)
	if len(file) > FlashLimit {
		return Result{}, problem("widgets", "the compiled file is %d bytes, larger than the flash limit of %d bytes", len(file), FlashLimit)
	}
	result := Result{File: file, Tags: tags, Alarms: alarms, AlarmSlots: alarmSlots}
	for _, each := range records {
		result.Records = append(result.Records, each.record)
	}
	return result, nil
}

// loadBackground gives the picture of a PNG background, and nil for a solid one.
func loadBackground(folder fs.FS, spec backgroundSpec) (image.Image, error) {
	if spec.path == "" {
		return nil, nil
	}
	content, err := readInside(folder, "background", spec.path)
	if err != nil {
		return nil, err
	}
	width, height, err := pngSize("background", spec.path, content)
	if err != nil {
		return nil, err
	}
	if width != screenWidth || height != screenHeight {
		return nil, problem("background", "%s is %d x %d, want %d x %d", spec.path, width, height, screenWidth, screenHeight)
	}
	return decodePNG("background", spec.path, content)
}

func loadFonts(folder fs.FS, specs []fontSpec) ([]loadedFont, error) {
	fonts := make([]loadedFont, 0, len(specs))
	for _, spec := range specs {
		face, err := loadFont(folder, spec)
		if err != nil {
			return nil, err
		}
		fonts = append(fonts, loadedFont{name: spec.name, glyphs: newGlyphFont(face),
			colour: rgb565(spec.colour.Red, spec.colour.Green, spec.colour.Blue)})
	}
	return fonts, nil
}

// fontOf gives the font of a widget, and none for an image or a bar, which have no font.
func fontOf(fonts []loadedFont, spec widgetSpec) loadedFont {
	if spec.kind == "image" || spec.kind == "bar" {
		return loadedFont{}
	}
	return fonts[spec.font]
}
