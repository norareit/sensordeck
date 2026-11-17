package themecompile

import (
	"encoding/binary"
	"image"
	"io/fs"

	"github.com/norareit/sensordeck/internal/themefile"
)

// built is a record with the resource it points to.
type built struct {
	record   themefile.Record
	resource []byte
	area     box // the box the record draws in, which a later text is compared with
}

// loadedFont is a font of the theme ready to draw, with its colour as the device holds it.
type loadedFont struct {
	name   string
	glyphs glyphFont
	colour themefile.Colour
}

// box is the rectangle of a record, in 64 bits so that no x or y of a theme can overflow it.
type box struct{ x, y, width, height int64 }

// overlaps tells whether two boxes share at least one pixel. A box that only touches another at its edge shares none,
// and a box of no width or no height shares none with any.
func (area box) overlaps(other box) bool {
	return area.x < other.x+other.width && other.x < area.x+area.width &&
		area.y < other.y+other.height && other.y < area.y+area.height &&
		area.width > 0 && area.height > 0 && other.width > 0 && other.height > 0
}

func (area box) head(recordType, index, tag byte) themefile.Head {
	return themefile.Head{Type: recordType, Index: index, Tag: tag,
		X: uint16(area.x), Y: uint16(area.y), Width: uint16(area.width), Height: uint16(area.height)}
}

func checkBox(place string, area box) error {
	// Each size is compared with the space left after its coordinate, never added to it, so that no coordinate can
	// overflow the sum.
	if area.x > screenWidth || area.width > screenWidth-area.x || area.y > screenHeight || area.height > screenHeight-area.y {
		return problem(place, "the box at %d,%d of %d x %d is not within the screen of %d x %d",
			area.x, area.y, area.width, area.height, screenWidth, screenHeight)
	}
	return nil
}

func checkGlyphs(place string, font loadedFont, characters string) error {
	if character, missing := font.glyphs.missing(characters); missing {
		return problem(place, "the font %s has no glyph for \"%c\"", font.name, character)
	}
	return nil
}

// widgetInput is what the record of a widget is built from.
type widgetInput struct {
	spec   widgetSpec
	font   loadedFont
	folder fs.FS
	tag    byte
	// alarmTag is the tag of the alarm widget of an alarm number, which has two records.
	alarmTag byte
	slot     int
	offset   uint32 // where its resource starts in the file
}

// buildWidget builds the records and the resources of one widget, which are one for each kind but an alarm number, which
// has two: the normal widget, then the alarm widget in the next slot. The glyphs are checked, then the box against the
// screen, and the pixels or masks are made only after, so that a box outside the screen costs nothing.
func buildWidget(input widgetInput) ([]built, error) {
	place := widgetPlace(input.spec.number, input.spec.kind)
	var widget built
	var err error
	switch input.spec.kind {
	case "text":
		widget, err = buildText(input, place)
	case "number":
		if input.spec.alarm != nil {
			return buildAlarmNumber(input, place)
		}
		widget, err = buildNumber(input, place)
	case "datetime":
		widget, err = buildDatetime(input, place)
	case "bar":
		widget, err = buildBar(input, place)
	default:
		widget, err = buildImage(input, place)
	}
	if err != nil {
		return nil, err
	}
	return []built{widget}, nil
}

func (input widgetInput) index() byte { return byte(input.slot - 1) }

func buildText(input widgetInput, place string) (built, error) {
	glyphs := input.font.glyphs
	if err := checkGlyphs(place, input.font, input.spec.text); err != nil {
		return built{}, err
	}
	width := glyphs.textWidth(input.spec.text)
	area := box{input.spec.x, input.spec.y, int64(width), int64(glyphs.height)}
	if err := checkBox(place, area); err != nil {
		return built{}, err
	}
	resource := glyphs.textMask(input.spec.text, width)
	record := themefile.Text{Head: area.head(themefile.TypeText, input.index(), 0), Offset: input.offset,
		Colour: input.font.colour, OpacityGuess: 0xff, Resource: resource}
	return built{record, resource, area}, nil
}

func buildNumber(input widgetInput, place string) (built, error) {
	area, err := numberArea(input, place)
	if err != nil {
		return built{}, err
	}
	return numberRecord(input, area, input.tag, input.font.colour, false), nil
}

// numberArea checks the glyphs and the box of a number, and gives the box.
func numberArea(input widgetInput, place string) (box, error) {
	glyphs := input.font.glyphs
	if err := checkGlyphs(place, input.font, themefile.NumberGlyphOrder); err != nil {
		return box{}, err
	}
	area := box{input.spec.x, input.spec.y, int64(input.spec.digits * glyphs.digitWidth), int64(glyphs.height)}
	return area, checkBox(place, area)
}

// buildAlarmNumber builds the two records of an alarm number. Both have the box, the glyphs and the options of the
// number, and a transparent minus as wide as the box: a negative value then draws that minus first, which pushes its
// digits out of the box, where the screen clips them, and so the widget shows nothing. They differ in their colours and
// their tags, and each has a glyph set of its own, so that no two records share a resource.
func buildAlarmNumber(input widgetInput, place string) ([]built, error) {
	area, err := numberArea(input, place)
	if err != nil {
		return nil, err
	}
	normal := numberRecord(input, area, input.tag, input.font.colour, true)
	alarmColour := input.spec.alarm.colour
	following := input
	following.slot++
	following.offset += uint32(len(normal.resource))
	alarm := numberRecord(following, area, input.alarmTag, rgb565(alarmColour.Red, alarmColour.Green, alarmColour.Blue), true)
	return []built{normal, alarm}, nil
}

// numberRecord writes the record of a number and its glyph set. With hidden set, the glyph of the minus is transparent
// and as wide as the box.
func numberRecord(input widgetInput, area box, tag byte, colour themefile.Colour, hidden bool) built {
	glyphs := input.font.glyphs
	record := themefile.Number{Head: area.head(themefile.TypeNumber, input.index(), tag), Alignment: input.spec.alignment,
		Colour: colour, Opacity: input.spec.opacity, Offset: input.offset, GlyphHeight: uint16(glyphs.height)}
	if input.spec.thousands {
		record.ThousandsMode = 1
	}
	var resource []byte
	for position, character := range themefile.NumberGlyphOrder {
		mask, width := glyphs.glyphOf(character)
		if hidden && character == '-' {
			mask, width = glyphs.hiddenMinus(int(area.width)), int(area.width)
		}
		record.GlyphWidths[position] = uint16(width)
		resource = append(resource, mask...)
	}
	record.Resource = resource
	return built{record, resource, area}
}

// barSampleBytes are bytes 18-19 and 26-27 of every bar of the vendor sample. The device draws them, and changing them
// had no visible effect.
const barSampleBytes = 0x76b0

// buildBar writes one colour into both of the colours of the fill and the frame, since the device check did not
// separate them, and the bytes the device check did not decode as the vendor sample holds them.
func buildBar(input widgetInput, place string) (built, error) {
	spec := input.spec
	area := box{spec.x, spec.y, spec.width, spec.height}
	if err := checkBox(place, area); err != nil {
		return built{}, err
	}
	colour := rgb565(spec.colour.Red, spec.colour.Green, spec.colour.Blue)
	record := themefile.Bar{Head: area.head(themefile.TypeBar, input.index(), input.tag), Byte11: 1,
		EmptyColour:         rgb565(spec.background.Red, spec.background.Green, spec.background.Blue),
		FillOrFrameColour14: colour, FillOrFrameColour16: colour}
	binary.BigEndian.PutUint16(record.Bytes18To63[0:], barSampleBytes)
	binary.BigEndian.PutUint16(record.Bytes18To63[8:], barSampleBytes)
	return built{record: record, area: area}, nil
}

// datetimeOthers are the five glyphs after the digits that a format can refer to, in the order of the file; the six
// that follow them in the file are never referred to.
const datetimeOthers = ".-/ :"

func buildDatetime(input widgetInput, place string) (built, error) {
	glyphs := input.font.glyphs
	if err := checkGlyphs(place, input.font, "0123456789"+datetimeOthers); err != nil {
		return built{}, err
	}
	area := box{input.spec.x, input.spec.y, int64(formatWidth(input.spec.format, glyphs)), int64(glyphs.height)}
	if err := checkBox(place, area); err != nil {
		return built{}, err
	}
	record := themefile.Datetime{Head: area.head(themefile.TypeDatetime, input.index(), 21), Byte11: 1,
		Colour: input.font.colour, OpacityGuess: 0xff, Offset: input.offset,
		GlyphHeight: uint16(glyphs.height), DigitWidth: uint16(glyphs.digitWidth)}
	copy(record.Format[:], input.spec.format)
	var resource []byte
	for _, character := range "0123456789" {
		resource = append(resource, glyphs.digit(character)...)
	}
	for position, character := range datetimeOthers {
		mask, width := glyphs.separator(character)
		record.OtherGlyphWidths[position] = uint16(width)
		resource = append(resource, mask...)
	}
	for position := len(datetimeOthers); position < len(record.OtherGlyphWidths); position++ {
		record.OtherGlyphWidths[position] = uint16(glyphs.digitWidth)
		resource = append(resource, glyphs.blank()...)
	}
	record.Resource = resource
	return built{record, resource, area}, nil
}

// formatWidth is the width of a compiled format as drawn: each year four digit widths, each other token two, each
// separator its own width.
func formatWidth(format []byte, glyphs glyphFont) int {
	width := 0
	for _, code := range format {
		switch {
		case code == '1':
			width += 4 * glyphs.digitWidth
		case code >= '2' && code <= '6':
			width += 2 * glyphs.digitWidth
		default:
			width += glyphs.advance(rune(code))
		}
	}
	return width
}

func buildImage(input widgetInput, place string) (built, error) {
	content, err := readInside(input.folder, place, input.spec.file)
	if err != nil {
		return built{}, err
	}
	width, height, err := pngSize(place, input.spec.file, content)
	if err != nil {
		return built{}, err
	}
	area := box{input.spec.x, input.spec.y, int64(width), int64(height)}
	if err := checkBox(place, area); err != nil {
		return built{}, err
	}
	picture, err := decodePNG(place, input.spec.file, content)
	if err != nil {
		return built{}, err
	}
	resource := imagePlanes(picture)
	record := themefile.Image{Head: area.head(themefile.TypeImage, input.index(), 0), Offset: input.offset,
		FrameCount: 1, Byte16: 1, Resource: resource}
	return built{record, resource, area}, nil
}

// backgroundOffset is where the first resource starts: right after the record table.
const backgroundOffset = themefile.TableSize

// backgroundEnd is where the resource after the background starts: the background is a full picture in every theme.
const backgroundEnd = backgroundOffset + 2*screenWidth*screenHeight

// buildBackground builds the background record. A solid background is a full picture of its colour, and a PNG one is
// composited over black, so that the record needs no alpha. The texts are drawn into the picture, which keeps its size.
func buildBackground(spec backgroundSpec, picture image.Image, texts []themefile.Text) built {
	record := themefile.Background{Head: box{0, 0, screenWidth, screenHeight}.head(themefile.TypeBackground, 0, 0),
		Offset: backgroundOffset}
	record.Bytes18To63[0] = 1
	var resource []byte
	if picture == nil {
		colour := rgb565(spec.colour.Red, spec.colour.Green, spec.colour.Blue)
		record.FillColourGuess = colour
		resource = solidPlane(colour)
	} else {
		resource = backgroundPlane(picture)
	}
	if len(texts) > 0 {
		resource = drawTexts(record, resource, texts)
	}
	record.Resource = resource
	return built{record, resource, box{0, 0, screenWidth, screenHeight}}
}
