package themecompile

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// glyphFont is a font of the theme made ready to draw glyphs.
type glyphFont struct {
	face       font.Face
	ascent     int // the ascent rounded up: the baseline below the top of every mask
	height     int // the ascent plus the descent, each rounded up
	digitWidth int // the largest of the ten digits' advances, each rounded up
}

func newGlyphFont(face font.Face) glyphFont {
	metrics := face.Metrics()
	glyphs := glyphFont{face: face, ascent: metrics.Ascent.Ceil(), height: metrics.Ascent.Ceil() + metrics.Descent.Ceil()}
	for digit := '0'; digit <= '9'; digit++ {
		if width := glyphs.advance(digit); width > glyphs.digitWidth {
			glyphs.digitWidth = width
		}
	}
	return glyphs
}

func (glyphs glyphFont) advance(character rune) int {
	advance, _ := glyphs.face.GlyphAdvance(character)
	return advance.Ceil()
}

// missing gives the first rune of characters that the face has no glyph for.
func (glyphs glyphFont) missing(characters string) (rune, bool) {
	for _, character := range characters {
		if _, found := glyphs.face.GlyphAdvance(character); !found {
			return character, true
		}
	}
	return 0, false
}

// mask draws the text from the left edge of a mask of the width, with the kerning that the face gives. What the text
// draws outside the mask is cut off.
func (glyphs glyphFont) mask(text string, left, width int) []byte {
	alpha := image.NewAlpha(image.Rect(0, 0, width, glyphs.height))
	drawer := font.Drawer{Dst: alpha, Src: image.Opaque, Face: glyphs.face, Dot: fixed.P(left, glyphs.ascent)}
	drawer.DrawString(text)
	return alpha.Pix
}

// digit gives the mask of a digit: digitWidth wide, the digit moved right by half of what its advance leaves free,
// rounded down, so that the ten digits have the same width and a value does not move sideways as it changes.
func (glyphs glyphFont) digit(character rune) []byte {
	return glyphs.mask(string(character), (glyphs.digitWidth-glyphs.advance(character))/2, glyphs.digitWidth)
}

// separator gives the mask of one of ". - / : space", as wide as its advance rounded up, and that width.
func (glyphs glyphFont) separator(character rune) ([]byte, int) {
	width := glyphs.advance(character)
	return glyphs.mask(string(character), 0, width), width
}

// textWidth is the width of the text as the face measures it, rounded up.
func (glyphs glyphFont) textWidth(text string) int {
	return font.MeasureString(glyphs.face, text).Ceil()
}

func (glyphs glyphFont) textMask(text string, width int) []byte {
	return glyphs.mask(text, 0, width)
}

// blank gives a fully transparent mask of the digit width, for the glyphs of a datetime that no format refers to.
func (glyphs glyphFont) blank() []byte {
	return make([]byte, glyphs.digitWidth*glyphs.height)
}

// glyphOf gives the mask and width of a glyph of a number or of a datetime: a digit or a separator.
func (glyphs glyphFont) glyphOf(character rune) ([]byte, int) {
	if character >= '0' && character <= '9' {
		return glyphs.digit(character), glyphs.digitWidth
	}
	return glyphs.separator(character)
}
