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

func addBackgroundFill(builder *faketheme.Builder, number int, x, y, width, height uint16, colour uint16) {
	slot := faketheme.Head(0x81, byte(number), 0, x, y, width, height)
	faketheme.Put16(&slot, 12, colour)
	builder.Slot(number, slot)
}

func addText(builder *faketheme.Builder, number int, x, y, width, height uint16, colour uint16, alpha []byte) {
	slot := faketheme.Head(0x93, byte(number), 0, x, y, width, height)
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
func glyphMasks(height int, widths []uint16) []byte {
	var masks []byte
	for glyph, width := range widths {
		masks = append(masks, repeated(byte(10+glyph), height*int(width))...)
	}
	return masks
}

func addNumber(builder *faketheme.Builder, number int, x, y, width uint16, alignment byte, widths [12]uint16) {
	slot := faketheme.Head(0x92, byte(number), 1, x, y, width, 1)
	slot[11] = alignment
	faketheme.Put16(&slot, 12, 0xffff)
	faketheme.Put32(&slot, 16, builder.Resource(glyphMasks(1, widths[:])))
	faketheme.Put16(&slot, 20, 1)
	for glyph, glyphWidth := range widths {
		faketheme.Put16(&slot, 22+2*glyph, glyphWidth)
	}
	builder.Slot(number, slot)
}

func TestANumberIsDrawnAs42FromTheLeftOfItsBoxWhateverItsAlignment(test *testing.T) {
	builder := faketheme.New()
	// The glyph 4 is 3 wide, which is wider than the box, and the glyph 2 is 2 wide.
	addNumber(builder, 1, 30, 40, 2, 2, [12]uint16{2, 2, 2, 2, 3, 2, 2, 2, 2, 2, 1, 2})
	screen := previewOf(test, builder)
	expectRedWithin(test, "4", screen, 30, 40, 14)
	expectRedWithin(test, "4 beyond the box", screen, 32, 40, 14)
	expectRedWithin(test, "2", screen, 33, 40, 12)
	expectRedWithin(test, "2", screen, 34, 40, 12)
	expectColour(test, "after", screen, 35, 40, black)
	expectColour(test, "before", screen, 29, 40, black)
	expectColour(test, "below", screen, 30, 41, black)
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
	faketheme.Put32(&slot, 15, builder.Resource(glyphMasks(1, widths)))
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

func TestABarIsTheOutlineOfItsBoxInMagenta(test *testing.T) {
	builder := faketheme.New()
	builder.Slot(1, faketheme.Head(0x8b, 1, 0, 50, 50, 6, 4))
	builder.Slot(2, faketheme.Head(0x8b, 2, 0, 100, 100, 1, 1))
	screen := previewOf(test, builder)
	magenta := color.RGBA{255, 0, 255, 255}
	for _, point := range [][2]int{{50, 50}, {55, 50}, {50, 53}, {55, 53}, {52, 50}, {52, 53}, {50, 51}, {55, 52}, {100, 100}} {
		expectColour(test, "edge", screen, point[0], point[1], magenta)
	}
	for _, point := range [][2]int{{51, 51}, {54, 52}, {56, 50}, {50, 54}, {101, 100}} {
		expectColour(test, "inside or outside", screen, point[0], point[1], black)
	}
}

func TestWhatPassesTheEdgeOfTheScreenIsClipped(test *testing.T) {
	builder := faketheme.New()
	addText(builder, 1, 478, 0, 5, 1, 0xffff, repeated(255, 5))
	addBackgroundFill(builder, 2, 0, 318, 2, 10, 0x07e0)
	builder.Slot(3, faketheme.Head(0x8b, 3, 0, 478, 318, 5, 5))
	screen := previewOf(test, builder)
	expectColour(test, "text", screen, 479, 0, color.RGBA{255, 255, 255, 255})
	expectColour(test, "background", screen, 1, 319, color.RGBA{0, 255, 0, 255})
	expectColour(test, "bar edge", screen, 478, 319, color.RGBA{255, 0, 255, 255})
	expectColour(test, "bar inside", screen, 479, 319, black)
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
