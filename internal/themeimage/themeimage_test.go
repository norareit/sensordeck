package themeimage_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

func TestExpandRepeatsTheTopBitsOfEachChannel(test *testing.T) {
	cases := map[themefile.Colour]color.NRGBA{
		0x0000: {0, 0, 0, 255},
		0xffff: {255, 255, 255, 255},
		0xf800: {255, 0, 0, 255},
		0x07e0: {0, 255, 0, 255},
		0x001f: {0, 0, 255, 255},
		0x8410: {132, 130, 132, 255},
	}
	for colour, want := range cases {
		if got := themeimage.Expand(colour); got != want {
			test.Errorf("%04x gives %v, want %v", uint16(colour), got, want)
		}
	}
}

// sampleRecords reads the sample and gives its records by slot.
func sampleRecords(test *testing.T) map[int]themefile.Record {
	test.Helper()
	theme, _, err := themefile.Parse(faketheme.Sample())
	if err != nil {
		test.Fatal(err)
	}
	records := map[int]themefile.Record{}
	for _, entry := range theme.Entries {
		records[entry.Slot] = entry.Record
	}
	return records
}

func expectBounds(test *testing.T, name string, images []*image.NRGBA, sizes ...image.Point) {
	test.Helper()
	if len(images) != len(sizes) {
		test.Fatalf("%s: %d images, want %d", name, len(images), len(sizes))
	}
	for position, size := range sizes {
		if images[position].Bounds() != (image.Rectangle{Max: size}) {
			test.Errorf("%s: image %d has bounds %v", name, position, images[position].Bounds())
		}
	}
}

func expectPixel(test *testing.T, name string, picture *image.NRGBA, column, row int, want color.NRGBA) {
	test.Helper()
	if got := picture.NRGBAAt(column, row); got != want {
		test.Errorf("%s at %d,%d: %v, want %v", name, column, row, got, want)
	}
}

func TestTheBackgroundGivesOneOpaqueImageOfItsPixels(test *testing.T) {
	images := themeimage.Resources(sampleRecords(test)[1])
	expectBounds(test, "background", images, image.Pt(4, 2))
	// Pixel 1 is the bytes 0x11 and 0x01: little-endian 0x0111, red 0, green 8 and blue 17.
	expectPixel(test, "background", images[0], 1, 0, color.NRGBA{0, 32, 140, 255})
	// Pixel 7, at 3,1, is the bytes 0x17 and 0x07: 0x0717, red 0, green 56 and blue 23.
	expectPixel(test, "background", images[0], 3, 1, color.NRGBA{0, 227, 189, 255})
}

func TestAPixelIsLittleEndianAndAColourOfARecordIsBigEndian(test *testing.T) {
	builder := faketheme.New()
	slot := faketheme.Head(0x93, 1, 0, 0, 0, 1, 1)
	faketheme.Put32(&slot, 11, builder.Resource([]byte{0xff}))
	faketheme.Put16(&slot, 15, 0xf800)
	builder.Slot(1, slot)
	background := faketheme.Head(0x81, 2, 0, 0, 0, 1, 1)
	faketheme.Put32(&background, 14, builder.Resource([]byte{0x1f, 0xf8}))
	builder.Slot(2, background)
	theme, _, err := themefile.Parse(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	text := themeimage.Resources(theme.Entries[0].Record)[0]
	expectPixel(test, "text", text, 0, 0, color.NRGBA{255, 0, 0, 255})
	pixel := themeimage.Resources(theme.Entries[1].Record)[0]
	expectPixel(test, "background", pixel, 0, 0, color.NRGBA{255, 0, 255, 255})
}

func TestTheStartupGivesOneOpaqueImageForEachFrame(test *testing.T) {
	images := themeimage.Resources(sampleRecords(test)[2])
	expectBounds(test, "startup", images, image.Pt(2, 2), image.Pt(2, 2))
	// The first pixel of frame 2 is the bytes 0x28 and 0x29: 0x2928, red 5, green 9, blue 8.
	expectPixel(test, "startup", images[1], 0, 0, color.NRGBA{41, 36, 66, 255})
}

func TestTheImageGivesOneImageForEachFrameWithItsAlpha(test *testing.T) {
	images := themeimage.Resources(sampleRecords(test)[3])
	expectBounds(test, "image", images, image.Pt(3, 2), image.Pt(3, 2))
	// The first pixel of frame 1 has the alpha 0x31 and the colour bytes 0x40 and 0x41, 0x4140: red 8, green 10, blue 0.
	expectPixel(test, "image", images[0], 0, 0, color.NRGBA{66, 40, 0, 0x31})
	// The last pixel of frame 2 has the alpha 0x56.
	if got := images[1].NRGBAAt(2, 1).A; got != 0x56 {
		test.Errorf("alpha %#x", got)
	}
}

func TestTheTextGivesItsMaskInItsColour(test *testing.T) {
	images := themeimage.Resources(sampleRecords(test)[4])
	expectBounds(test, "text", images, image.Pt(5, 3))
	expectPixel(test, "text", images[0], 0, 0, color.NRGBA{255, 255, 255, 0x70})
	expectPixel(test, "text", images[0], 1, 1, color.NRGBA{255, 255, 255, 0x76})
}

func TestTheNumberGivesOneStripOfItsGlyphs(test *testing.T) {
	images := themeimage.Resources(sampleRecords(test)[5])
	expectBounds(test, "number", images, image.Pt(23, 3))
	// The glyph - is the twelfth, after 21 columns, and its mask starts at byte 63 of the set.
	expectPixel(test, "number", images[0], 21, 0, color.NRGBA{0, 255, 0, 0x80 + 63})
	expectPixel(test, "number", images[0], 21, 1, color.NRGBA{0, 255, 0, 0x80 + 65})
}

func TestTheDatetimeGivesOneStripOfItsGlyphs(test *testing.T) {
	images := themeimage.Resources(sampleRecords(test)[6])
	expectBounds(test, "datetime", images, image.Pt(86, 3))
	// The glyph for the second is the last of the 21: 20 + 55 columns before it, its mask at byte 225 of the set, and
	// 0x90 + 225 wraps to 0x71. Its width is 11, so the second row starts 11 bytes later.
	expectPixel(test, "datetime", images[0], 75, 0, color.NRGBA{0, 0, 255, 0x71})
	expectPixel(test, "datetime", images[0], 75, 1, color.NRGBA{0, 0, 255, 0x7c})
}

func TestARecordWithoutAResourceGivesNoImage(test *testing.T) {
	records := sampleRecords(test)
	if images := themeimage.Resources(records[7]); len(images) != 0 {
		test.Errorf("bar: %d images", len(images))
	}
	if images := themeimage.Resources(records[9]); len(images) != 0 {
		test.Errorf("unknown: %d images", len(images))
	}
	background := records[1].(themefile.Background)
	background.Offset, background.Resource = 0, nil
	if images := themeimage.Resources(background); len(images) != 0 {
		test.Errorf("background with offset 0: %d images", len(images))
	}
}
