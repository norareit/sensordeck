package themecompile

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// goFace parses a font as loadFont does.
func goFace(test *testing.T, ttf []byte, size float64) font.Face {
	test.Helper()
	parsed, err := opentype.Parse(ttf)
	if err != nil {
		test.Fatal(err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		test.Fatal(err)
	}
	return face
}

func regularFace(test *testing.T, size float64) font.Face {
	test.Helper()
	return goFace(test, goregular.TTF, size)
}

// narrowOne is a face whose 1 has half the advance of the face it embeds.
type narrowOne struct{ font.Face }

func (face narrowOne) GlyphAdvance(character rune) (advance fixed.Int26_6, found bool) {
	advance, found = face.Face.GlyphAdvance(character)
	if character == '1' {
		advance /= 2
	}
	return advance, found
}

func encodePNG(test *testing.T, picture image.Image) []byte {
	test.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		test.Fatal(err)
	}
	return buffer.Bytes()
}
