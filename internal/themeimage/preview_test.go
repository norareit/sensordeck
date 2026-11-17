package themeimage_test

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

func previewOf(test *testing.T, builder *faketheme.Builder) *image.RGBA {
	test.Helper()
	theme, _, err := themefile.Parse(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	return themeimage.Preview(theme)
}

func expectColour(test *testing.T, name string, screen *image.RGBA, column, row int, want color.RGBA) {
	test.Helper()
	if got := screen.RGBAAt(column, row); got != want {
		test.Errorf("%s at %d,%d: %v, want %v", name, column, row, got, want)
	}
}

// expectRedWithin checks the red channel of an opaque pixel, for the white glyphs over black that carry their alpha
// there.
func expectRedWithin(test *testing.T, name string, screen *image.RGBA, column, row int, want byte) {
	test.Helper()
	got := screen.RGBAAt(column, row)
	difference := int(got.R) - int(want)
	if difference < -1 || difference > 1 || got.A != 255 {
		test.Errorf("%s at %d,%d: %v, want red %d", name, column, row, got, want)
	}
}

func addBackgroundFill(builder *faketheme.Builder, number int, left, top, width, height uint16, colour uint16) {
	slot := faketheme.Head(0x81, byte(number), 0, left, top, width, height)
	faketheme.Put16(&slot, 12, colour)
	builder.Slot(number, slot)
}

func addText(builder *faketheme.Builder, number int, left, top, width, height uint16, colour uint16, alpha []byte) {
	slot := faketheme.Head(0x93, byte(number), 0, left, top, width, height)
	faketheme.Put32(&slot, 11, builder.Resource(alpha))
	faketheme.Put16(&slot, 15, colour)
	builder.Slot(number, slot)
}

func repeated(value byte, count int) []byte { return bytes.Repeat([]byte{value}, count) }

var black = color.RGBA{0, 0, 0, 255}

func TestAnEmptyThemeIsOpaqueBlack(test *testing.T) {
	screen := previewOf(test, faketheme.New())
	if screen.Bounds().Dx() != 480 || screen.Bounds().Dy() != 320 {
		test.Fatalf("bounds %v", screen.Bounds())
	}
	expectColour(test, "corner", screen, 0, 0, black)
	expectColour(test, "corner", screen, 479, 319, black)
}

func TestABackgroundFillsItsBoxOrShowsItsPixels(test *testing.T) {
	builder := faketheme.New()
	addBackgroundFill(builder, 1, 10, 20, 3, 2, 0xf800)
	slot := faketheme.Head(0x81, 2, 0, 100, 100, 2, 1)
	faketheme.Put32(&slot, 14, builder.Resource([]byte{0x1f, 0x00, 0xe0, 0x07}))
	builder.Slot(2, slot)
	screen := previewOf(test, builder)
	red := color.RGBA{255, 0, 0, 255}
	expectColour(test, "fill", screen, 10, 20, red)
	expectColour(test, "fill", screen, 12, 21, red)
	expectColour(test, "outside", screen, 13, 21, black)
	expectColour(test, "outside", screen, 9, 20, black)
	expectColour(test, "blue", screen, 100, 100, color.RGBA{0, 0, 255, 255})
	expectColour(test, "green", screen, 101, 100, color.RGBA{0, 255, 0, 255})
	expectColour(test, "past the pixels", screen, 102, 100, black)
}

func TestTheRecordsAreDrawnInTheOrderOfTheSlotsSourceOver(test *testing.T) {
	builder := faketheme.New()
	addText(builder, 5, 0, 0, 2, 1, 0x07e0, repeated(255, 2))
	addText(builder, 2, 1, 0, 2, 1, 0xf800, repeated(255, 2))
	screen := previewOf(test, builder)
	expectColour(test, "slot 2 only", screen, 2, 0, color.RGBA{255, 0, 0, 255})
	expectColour(test, "slot 5 on top", screen, 1, 0, color.RGBA{0, 255, 0, 255})
	expectColour(test, "slot 5 only", screen, 0, 0, color.RGBA{0, 255, 0, 255})
}

func TestAHalfOpaqueMaskIsBlendedOverWhatIsThere(test *testing.T) {
	builder := faketheme.New()
	addBackgroundFill(builder, 1, 0, 0, 1, 1, 0xffff)
	addText(builder, 2, 0, 0, 1, 1, 0x0000, []byte{128})
	got := previewOf(test, builder).RGBAAt(0, 0)
	if got.R < 126 || got.R > 128 || got.G != got.R || got.B != got.R || got.A != 255 {
		test.Errorf("blended %v", got)
	}
}

func TestAnImageIsDrawnAsItsFirstFrame(test *testing.T) {
	builder := faketheme.New()
	slot := faketheme.Head(0x84, 1, 0, 5, 6, 1, 1)
	// Frame 1: alpha 255 and the colour 0xf800; frame 2: alpha 255 and the colour 0x07e0.
	faketheme.Put32(&slot, 11, builder.Resource([]byte{255, 0x00, 0xf8, 255, 0xe0, 0x07}))
	slot[15] = 2
	builder.Slot(1, slot)
	expectColour(test, "first frame", previewOf(test, builder), 5, 6, color.RGBA{255, 0, 0, 255})
}

// glyphMasks gives the mask bytes of a glyph set in which each glyph is of one alpha, 10 plus its index.
func glyphMasks(base byte, height int, widths []uint16) []byte {
	var masks []byte
	for glyph, width := range widths {
		masks = append(masks, repeated(base+byte(glyph), height*int(width))...)
	}
	return masks
}

// numberAt describes a number of the tests. The glyphs are two wide but the point, which is one wide, and each is of
// the one alpha base plus its index. The zero value of a field is the value of an ordinary number: a box 10 wide at 30,40,
// glyphs one high, no thousands mode and the alpha 10.
type numberAt struct {
	left, top, width, boxHeight, glyphHeight uint16
	alignment, thousands, opacity, base      byte
}

func addNumber(builder *faketheme.Builder, number int, spec numberAt) {
	for field, value := range map[*uint16]uint16{&spec.left: 30, &spec.top: 40, &spec.width: 10} {
		if *field == 0 {
			*field = value
		}
	}
	if spec.boxHeight == 0 {
		spec.boxHeight = 1
	}
	if spec.glyphHeight == 0 {
		spec.glyphHeight = 1
	}
	if spec.base == 0 {
		spec.base = 10
	}
	widths := [12]uint16{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 2}
	slot := faketheme.Head(0x92, byte(number), 1, spec.left, spec.top, spec.width, spec.boxHeight)
	slot[11], slot[14], slot[15] = spec.alignment, spec.thousands, spec.opacity
	faketheme.Put16(&slot, 12, 0xffff)
	faketheme.Put32(&slot, 16, builder.Resource(glyphMasks(spec.base, int(spec.glyphHeight), widths[:])))
	faketheme.Put16(&slot, 20, spec.glyphHeight)
	for glyph, glyphWidth := range widths {
		faketheme.Put16(&slot, 22+2*glyph, glyphWidth)
	}
	builder.Slot(number, slot)
}

// numberPreview draws one number, opaque unless said otherwise.
func numberPreview(test *testing.T, spec numberAt) *image.RGBA {
	test.Helper()
	if spec.opacity == 0 {
		spec.opacity = 255
	}
	builder := faketheme.New()
	addNumber(builder, 1, spec)
	return previewOf(test, builder)
}

func TestANumberIsAlignedAsByte11Says(test *testing.T) {
	// The value 42 is 4 wide: the glyph 4 has the alpha 14 and the glyph 2 the alpha 12.
	left := numberPreview(test, numberAt{alignment: 0})
	expectRedWithin(test, "left, first column", left, 30, 40, 14)
	expectColour(test, "left, before", left, 29, 40, black)
	right := numberPreview(test, numberAt{alignment: 2})
	expectColour(test, "right, before", right, 35, 40, black)
	expectRedWithin(test, "right, first column", right, 36, 40, 14)
	expectRedWithin(test, "right, last column", right, 39, 40, 12)
	expectColour(test, "right, after", right, 40, 40, black)
	centre := numberPreview(test, numberAt{alignment: 1})
	expectColour(test, "centre, before", centre, 32, 40, black)
	expectRedWithin(test, "centre, first column", centre, 33, 40, 14)
	other := numberPreview(test, numberAt{alignment: 7})
	expectRedWithin(test, "another value, first column", other, 30, 40, 14)
	expectColour(test, "another value, before", other, 29, 40, black)
}

func TestACentredNumberWiderThanItsBoxRoundsDown(test *testing.T) {
	screen := numberPreview(test, numberAt{width: 3, alignment: 1})
	expectColour(test, "clipped", screen, 29, 40, black)
	expectRedWithin(test, "second column of the 4", screen, 30, 40, 14)
	expectRedWithin(test, "first column of the 2", screen, 31, 40, 12)
	expectRedWithin(test, "second column of the 2", screen, 32, 40, 12)
	expectColour(test, "after", screen, 33, 40, black)
}

func TestANumberIsCutAtTheEdgesOfItsBox(test *testing.T) {
	left := numberPreview(test, numberAt{width: 3, alignment: 0})
	expectRedWithin(test, "4", left, 30, 40, 14)
	expectRedWithin(test, "4", left, 31, 40, 14)
	expectRedWithin(test, "first column of the 2", left, 32, 40, 12)
	expectColour(test, "second column of the 2, outside", left, 33, 40, black)
	right := numberPreview(test, numberAt{width: 3, alignment: 2})
	expectColour(test, "first column of the 4, outside", right, 29, 40, black)
	expectRedWithin(test, "second column of the 4", right, 30, 40, 14)
	expectRedWithin(test, "2", right, 31, 40, 12)
	expectRedWithin(test, "2", right, 32, 40, 12)
	tall := numberPreview(test, numberAt{boxHeight: 1, glyphHeight: 2})
	expectRedWithin(test, "top row", tall, 30, 40, 14)
	expectColour(test, "row below the box", tall, 30, 41, black)
	expectColour(test, "row above the box", tall, 30, 39, black)
}

func TestANumberIsClippedToTheScreen(test *testing.T) {
	screen := numberPreview(test, numberAt{left: 478, width: 10})
	expectRedWithin(test, "first column", screen, 478, 40, 14)
	expectRedWithin(test, "last column of the screen", screen, 479, 40, 14)
}

func TestTheOpacityOfANumberScalesItsAlpha(test *testing.T) {
	// The glyph 4 has the alpha 104, so byte 15 at 128 gives (104 x 128 + 127) / 255, which is 52. The opacity 0 is
	// given to addNumber itself, since numberPreview takes 0 for opaque.
	builder := faketheme.New()
	addNumber(builder, 1, numberAt{base: 100, opacity: 0})
	expectColour(test, "byte 15 at 0", previewOf(test, builder), 30, 40, black)
	expectRedWithin(test, "byte 15 at 128", numberPreview(test, numberAt{base: 100, opacity: 128, alignment: 0}), 30, 40, 52)
	expectRedWithin(test, "byte 15 at 255", numberPreview(test, numberAt{base: 100, opacity: 255, alignment: 0}), 30, 40, 104)
}

func TestTheThousandsModeDraws1Point2(test *testing.T) {
	screen := numberPreview(test, numberAt{thousands: 1})
	expectRedWithin(test, "1", screen, 30, 40, 11)
	expectRedWithin(test, "1", screen, 31, 40, 11)
	expectRedWithin(test, "point", screen, 32, 40, 20)
	expectRedWithin(test, "2", screen, 33, 40, 12)
	expectRedWithin(test, "2", screen, 34, 40, 12)
	expectColour(test, "after", screen, 35, 40, black)
}

func TestADatetimeIsDrawnFromItsFormatWithTheSampleDateAndTime(test *testing.T) {
	builder := faketheme.New()
	slot := faketheme.Head(0x8e, 1, 21, 10, 5, 100, 1)
	faketheme.Put16(&slot, 12, 0xffff)
	faketheme.Put16(&slot, 19, 1)
	faketheme.Put16(&slot, 21, 2)
	var widths []uint16
	for range 10 {
		widths = append(widths, 2)
	}
	for glyph := range 11 {
		faketheme.Put16(&slot, 23+2*glyph, 1)
		widths = append(widths, 1)
	}
	faketheme.Put32(&slot, 15, builder.Resource(glyphMasks(10, 1, widths)))
	// The x has no glyph, so it is drawn as nothing and does not move what follows.
	copy(slot[45:], "1x-2-3 4:5:6")
	builder.Slot(1, slot)
	screen := previewOf(test, builder)
	column := 10
	for _, character := range "2026-11-15 09:41:07" {
		width, alpha := 2, byte(10+character-'0')
		switch character {
		case '-':
			width, alpha = 1, 10+11
		case ' ':
			width, alpha = 1, 10+13
		case ':':
			width, alpha = 1, 10+14
		}
		for offset := range width {
			expectRedWithin(test, string(character), screen, column+offset, 5, alpha)
		}
		column += width
	}
	expectColour(test, "after the time", screen, column, 5, black)
}

func addBar(builder *faketheme.Builder, number int, left, top, width, height uint16, empty, fill, frame uint16) {
	slot := faketheme.Head(0x8b, byte(number), 0, left, top, width, height)
	faketheme.Put16(&slot, 12, empty)
	faketheme.Put16(&slot, 14, fill)
	faketheme.Put16(&slot, 16, frame)
	builder.Slot(number, slot)
}

func TestABarIsDrawnFilledTo42(test *testing.T) {
	builder := faketheme.New()
	// The inside is 100 wide, so the fill is the 42 columns from 51 to 92.
	addBar(builder, 1, 50, 50, 102, 6, 0x001f, 0x07e0, 0xf800)
	screen := previewOf(test, builder)
	red, green, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}
	for _, point := range [][2]int{{50, 50}, {151, 50}, {50, 55}, {151, 55}, {50, 52}, {151, 52}, {100, 50}, {100, 55}} {
		expectColour(test, "frame", screen, point[0], point[1], red)
	}
	expectColour(test, "first column of the fill", screen, 51, 51, green)
	expectColour(test, "last column of the fill", screen, 92, 52, green)
	expectColour(test, "last row of the fill", screen, 92, 54, green)
	expectColour(test, "first column of the empty part", screen, 93, 52, blue)
	expectColour(test, "last column of the empty part", screen, 150, 54, blue)
	expectColour(test, "outside", screen, 152, 52, black)
	expectColour(test, "outside", screen, 49, 52, black)
}

func TestABarTooSmallForAnInsideShowsItsFrame(test *testing.T) {
	builder := faketheme.New()
	addBar(builder, 1, 10, 10, 2, 2, 0x001f, 0x07e0, 0xf800)
	addBar(builder, 2, 20, 10, 3, 6, 0x001f, 0x07e0, 0xf800)
	addBar(builder, 3, 30, 10, 1, 1, 0x001f, 0x07e0, 0xf800)
	screen := previewOf(test, builder)
	red := color.RGBA{255, 0, 0, 255}
	for _, point := range [][2]int{{10, 10}, {11, 10}, {10, 11}, {11, 11}, {30, 10}} {
		expectColour(test, "frame alone", screen, point[0], point[1], red)
	}
	// The inside of the second is one column, and 1 x 42 / 100 is 0 columns of fill.
	expectColour(test, "one inside column", screen, 21, 12, color.RGBA{0, 0, 255, 255})
	expectColour(test, "frame", screen, 20, 12, red)
	expectColour(test, "frame", screen, 22, 12, red)
}

func TestWhatPassesTheEdgeOfTheScreenIsClipped(test *testing.T) {
	builder := faketheme.New()
	addText(builder, 1, 478, 0, 5, 1, 0xffff, repeated(255, 5))
	addBackgroundFill(builder, 2, 0, 318, 2, 10, 0x07e0)
	addBar(builder, 3, 478, 318, 5, 5, 0x001f, 0x07e0, 0xf800)
	screen := previewOf(test, builder)
	expectColour(test, "text", screen, 479, 0, color.RGBA{255, 255, 255, 255})
	expectColour(test, "background", screen, 1, 319, color.RGBA{0, 255, 0, 255})
	expectColour(test, "bar frame", screen, 478, 319, color.RGBA{255, 0, 0, 255})
	// The inside is 3 wide, and 3 x 42 / 100 is 1 column of fill.
	expectColour(test, "bar inside", screen, 479, 319, color.RGBA{0, 255, 0, 255})
}

func TestAStartupAndAnUnknownRecordAreNotDrawn(test *testing.T) {
	builder := faketheme.New()
	slot := faketheme.Head(0x94, 1, 0, 0, 0, 1, 1)
	faketheme.Put32(&slot, 11, builder.Resource([]byte{0xff, 0xff}))
	slot[15] = 1
	builder.Slot(1, slot)
	var unknown [64]byte
	unknown[0], unknown[3], unknown[4] = 0x55, 0xff, 0xff
	builder.Slot(2, unknown)
	expectColour(test, "startup", previewOf(test, builder), 0, 0, black)
}

// addAlarmNumber adds a left-aligned number of a box 10 wide whose glyph of the minus is transparent and as wide as the box.
func addAlarmNumber(builder *faketheme.Builder, number int) {
	widths := [12]uint16{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 10}
	masks := glyphMasks(10, 1, widths[:])
	copy(masks[len(masks)-10:], make([]byte, 10))
	slot := faketheme.Head(0x92, byte(number), 1, 30, 40, 10, 1)
	slot[15] = 255
	faketheme.Put16(&slot, 12, 0xffff)
	faketheme.Put32(&slot, 16, builder.Resource(masks))
	faketheme.Put16(&slot, 20, 1)
	for glyph, glyphWidth := range widths {
		faketheme.Put16(&slot, 22+2*glyph, glyphWidth)
	}
	builder.Slot(number, slot)
}

func TestAHidingValueDrawsNoPixelOfANumberWithAWideTransparentMinus(test *testing.T) {
	builder := faketheme.New()
	addAlarmNumber(builder, 1)
	theme, _, err := themefile.Parse(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	hidden := themeimage.PreviewWithValues(theme, map[int]int16{1: -1})
	for column := 25; column < 50; column++ {
		expectColour(test, "hidden", hidden, column, 40, black)
	}
	shown := themeimage.PreviewWithValues(theme, map[int]int16{2: -1})
	expectRedWithin(test, "a slot not in the map is drawn as 42", shown, 30, 40, 14)
	if !bytes.Equal(shown.Pix, themeimage.Preview(theme).Pix) {
		test.Error("Preview differs from PreviewWithValues with no value for the slot")
	}
}
