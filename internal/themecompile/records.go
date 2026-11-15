package themecompile

import (
	"image"
	"io/fs"

	"github.com/norareit/sensordeck/internal/themefile"
)

// built is a record with the resource it points to.
type built struct {
	record   themefile.Record
	resource []byte
}

// loadedFont is a font of the theme ready to draw, with its colour as the device holds it.
type loadedFont struct {
	name   string
	glyphs glyphFont
	colour themefile.Colour
}

// box is the rectangle of a record, in 64 bits so that no x or y of a theme can overflow it.
type box struct{ x, y, width, height int64 }

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
	slot   int
	offset uint32 // where its resource starts in the file
}

// buildWidget builds the record and the resource of one widget. The glyphs are checked, then the box against the
// screen, and the pixels or masks are made only after, so that a box outside the screen costs nothing.
func buildWidget(input widgetInput) (built, error) {
	place := widgetPlace(input.spec.number, input.spec.kind)
	switch input.spec.kind {
	case "text":
		return buildText(input, place)
	case "number":
		return buildNumber(input, place)
	case "datetime":
		return buildDatetime(input, place)
	}
	return buildImage(input, place)
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
	return built{record, resource}, nil
}

func buildNumber(input widgetInput, place string) (built, error) {
	glyphs := input.font.glyphs
	if err := checkGlyphs(place, input.font, themefile.NumberGlyphOrder); err != nil {
		return built{}, err
	}
	area := box{input.spec.x, input.spec.y, int64(input.spec.digits * glyphs.digitWidth), int64(glyphs.height)}
	if err := checkBox(place, area); err != nil {
		return built{}, err
	}
	record := themefile.Number{Head: area.head(themefile.TypeNumber, input.index(), input.tag), AlignmentGuess: 1,
		Colour: input.font.colour, OpacityGuess: 0xff, Offset: input.offset, GlyphHeight: uint16(glyphs.height)}
	var resource []byte
	for position, character := range themefile.NumberGlyphOrder {
		mask, width := glyphs.glyphOf(character)
		record.GlyphWidths[position] = uint16(width)
		resource = append(resource, mask...)
	}
	record.Resource = resource
	return built{record, resource}, nil
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
	return built{record, resource}, nil
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
	return built{record, resource}, nil
}

// backgroundOffset is where the first resource starts: right after the record table.
const backgroundOffset = themefile.TableSize

// buildBackground builds the background record. A solid background is a full picture of its colour, and a PNG one is
// composited over black, so that the record needs no alpha.
func buildBackground(spec backgroundSpec, picture image.Image) built {
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
	record.Resource = resource
	return built{record, resource}
}
