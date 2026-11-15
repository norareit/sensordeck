package themecompile

import (
	"bytes"
	"image"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestGlyphHeightIsAscentPlusDescentRoundedUp(test *testing.T) {
	face := regularFace(test, 36)
	glyphs := newGlyphFont(face)
	metrics := face.Metrics()
	if glyphs.height != metrics.Ascent.Ceil()+metrics.Descent.Ceil() || glyphs.height != 43 {
		test.Errorf("height %d, ascent %d", glyphs.height, glyphs.ascent)
	}
}

func TestDigitMasksHaveEqualWidthAndHeight(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 36))
	for digit := '0'; digit <= '9'; digit++ {
		if got := len(glyphs.digit(digit)); got != glyphs.digitWidth*glyphs.height {
			test.Errorf("digit %c: %d bytes, want %d", digit, got, glyphs.digitWidth*glyphs.height)
		}
	}
	if bytes.Equal(glyphs.digit('8'), glyphs.blank()) {
		test.Error("the mask of 8 is blank")
	}
}

func leftmostColumn(mask []byte, width int) int {
	for column := range width {
		for row := 0; row*width < len(mask); row++ {
			if mask[row*width+column] != 0 {
				return column
			}
		}
	}
	return -1
}

func TestNarrowDigitIsCentred(test *testing.T) {
	wide := newGlyphFont(regularFace(test, 36))
	narrow := newGlyphFont(narrowOne{regularFace(test, 36)})
	if narrow.digitWidth != wide.digitWidth {
		test.Fatalf("the digit width changed: %d and %d", narrow.digitWidth, wide.digitWidth)
	}
	shift := (narrow.digitWidth - narrow.advance('1')) / 2
	if shift == 0 {
		test.Fatal("the narrow one is not narrower")
	}
	alpha := image.NewAlpha(image.Rect(0, 0, narrow.digitWidth, narrow.height))
	drawer := font.Drawer{Dst: alpha, Src: image.Opaque, Face: narrow.face, Dot: fixed.P(shift, narrow.ascent)}
	drawer.DrawString("1")
	got := narrow.digit('1')
	if !bytes.Equal(got, alpha.Pix) {
		test.Error("the mask of 1 is not the glyph drawn at the centring shift")
	}
	if leftmostColumn(got, narrow.digitWidth) <= leftmostColumn(wide.digit('1'), wide.digitWidth) {
		test.Error("the narrow 1 is not to the right of the unmoved one")
	}
}

func TestSeparatorIsAsWideAsItsAdvance(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 36))
	for _, character := range ".-/: " {
		mask, width := glyphs.separator(character)
		if width != glyphs.advance(character) || len(mask) != width*glyphs.height {
			test.Errorf("%q: width %d, %d bytes", character, width, len(mask))
		}
	}
}

func TestBlankGlyphIsTransparent(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 36))
	blank := glyphs.blank()
	if len(blank) != glyphs.digitWidth*glyphs.height || !bytes.Equal(blank, make([]byte, len(blank))) {
		test.Error("the blank glyph is not transparent or not of the digit width")
	}
}

func TestTextMaskFillsItsMeasuredWidth(test *testing.T) {
	face := regularFace(test, 36)
	glyphs := newGlyphFont(face)
	width := glyphs.textWidth("AV")
	if want := font.MeasureString(face, "AV").Ceil(); width != want {
		test.Errorf("width %d, want %d", width, want)
	}
	mask := glyphs.textMask("AV", width)
	if len(mask) != width*glyphs.height || bytes.Equal(mask, make([]byte, len(mask))) {
		test.Errorf("%d bytes, or blank", len(mask))
	}
}

func TestMissingGivesTheFirstCharacterWithNoGlyph(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 36))
	if got, found := glyphs.missing("12年"); !found || got != '年' {
		test.Errorf("got %q, %v", got, found)
	}
	if _, found := glyphs.missing("0123456789.-"); found {
		test.Error("a number glyph is missing")
	}
}
