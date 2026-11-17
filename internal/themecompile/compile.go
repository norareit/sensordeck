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

const (
	// DrawLimit is the size of a theme file up to which the screen draws every widget. The screen left out each widget
	// whose data ends between 503 692 and 504 015 bytes of the file or later, as observed on 2026-11-17, and 503 808 is
	// the one multiple of 4096 in that range. The guess and the evidence are in docs/theme-format.md, and a better
	// measurement changes this line and that document.
	DrawLimit = 0x7B000
	// maxRecords is the number of records that fit in the record table beside the header and the background.
	maxRecords = 62
)

// Result is a compiled theme.
type Result struct {
	File    []byte             // the bytes of the theme file
	Records []themefile.Record // slots 1 and up in the order of the slots, each with its offset and resource; no text drawn into the background is among them
	Tags    []config.Mapping   // the plain entries, ascending by tag
	Alarms  []config.AlarmPair // ascending by normal tag
	// AlarmSlots are the slots of the alarm widgets of the alarm numbers, in the numbering of themefile.Entry.Slot.
	AlarmSlots []int
}

// Compile compiles the theme folder. A text whose box overlaps no record before it is drawn into the background and has
// no record. Its error is "<place>: <problem>", or "<problem>" for a fault of theme.yaml as a
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
	records, texts, alarmSlots, err := buildRecords(parsed, fonts, folder, tags, alarms)
	if err != nil {
		return Result{}, err
	}
	records[0] = buildBackground(parsed.background, picture, texts)
	if len(records)-1 > maxRecords {
		return Result{}, problem("widgets", "%d records, more than the %d that fit beside the background; an alarm number is two records and a text drawn into the background none", len(records)-1, maxRecords)
	}
	file := layout(records)
	if err := checkSize(len(file)); err != nil {
		return Result{}, err
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

// buildRecords builds the records of the widgets in the order of the list, after a placeholder for the background, which
// the caller builds once the texts for it are known. A text whose box overlaps no record before it is not a record but
// goes to the texts for the background, since the background is under every record and the picture stays the same.
func buildRecords(parsed document, fonts []loadedFont, folder fs.FS, tags []config.Mapping, alarms []config.AlarmPair) ([]built, []themefile.Text, []int, error) {
	records := []built{{}}
	var texts []themefile.Text
	var alarmSlots []int
	offset := backgroundEnd
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
			return nil, nil, nil, err
		}
		if text, isText := widgets[0].record.(themefile.Text); isText && !overlapsAny(widgets[0].area, records[1:]) {
			texts = append(texts, text)
			continue
		}
		for _, widget := range widgets {
			records = append(records, widget)
			offset += len(widget.resource)
		}
	}
	return records, texts, alarmSlots, nil
}

func overlapsAny(area box, records []built) bool {
	for _, each := range records {
		if area.overlaps(each.area) {
			return true
		}
	}
	return false
}

// checkSize refuses a file that the screen would not draw in full.
func checkSize(size int) error {
	if size <= DrawLimit {
		return nil
	}
	return problem("widgets", "the compiled theme is %d bytes, %d more than the %d bytes that the screen draws widgets from; the background takes %d of them, each number a glyph set of its font and an alarm number two, each datetime a glyph set and each image 3 bytes a pixel, and a text none",
		size, size-DrawLimit, DrawLimit, 2*screenWidth*screenHeight)
}
