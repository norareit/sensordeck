// Package themeimage turns the typed records of a theme file into images: the images of each resource and the preview
// of the whole screen. It does no I/O.
package themeimage

import (
	"image"
	"image/color"

	"github.com/norareit/sensordeck/internal/themefile"
)

// Expand gives the 8-bit channels of an RGB565 colour by repeating the top bits of each channel in its low bits, so
// that 0 gives 0 and the largest value gives 255. The colour is opaque.
func Expand(colour themefile.Colour) color.NRGBA {
	red := uint8(colour>>11) & 0x1f
	green := uint8(colour>>5) & 0x3f
	blue := uint8(colour) & 0x1f
	return color.NRGBA{R: red<<3 | red>>2, G: green<<2 | green>>4, B: blue<<3 | blue>>2, A: 255}
}

// Resources gives the images of a record's resource, of the record's width and height unless said otherwise: none for
// a bar, a record of an unknown type and a background with offset 0.
func Resources(record themefile.Record) []*image.NRGBA {
	switch typed := record.(type) {
	case themefile.Background:
		if typed.Offset == 0 {
			return nil
		}
		return []*image.NRGBA{opaque(typed.Resource, int(typed.Width), int(typed.Height))}
	case themefile.Startup:
		var frames []*image.NRGBA
		for _, frame := range typed.Frames() {
			frames = append(frames, opaque(frame, int(typed.Width), int(typed.Height)))
		}
		return frames
	case themefile.Image:
		var frames []*image.NRGBA
		for _, frame := range typed.Frames() {
			frames = append(frames, masked(frame.Alpha, frame.Colour, int(typed.Width), int(typed.Height)))
		}
		return frames
	case themefile.Text:
		return []*image.NRGBA{mask(typed.Resource, int(typed.Width), int(typed.Height), typed.Colour)}
	case themefile.Number:
		return []*image.NRGBA{strip(glyphImages(typed.Glyphs(), typed.GlyphWidths[:], typed.GlyphHeight, typed.Colour))}
	case themefile.Datetime:
		widths := typed.GlyphWidths()
		return []*image.NRGBA{strip(glyphImages(typed.Glyphs(), widths[:], typed.GlyphHeight, typed.Colour))}
	}
	return nil
}

// pixelAt reads the RGB565 pixel at the index from a plane, little-endian: the low byte comes first.
func pixelAt(pixels []byte, index int) themefile.Colour {
	return themefile.Colour(uint16(pixels[2*index]) | uint16(pixels[2*index+1])<<8)
}

func opaque(pixels []byte, width, height int) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for index := range width * height {
		picture.SetNRGBA(index%width, index/width, Expand(pixelAt(pixels, index)))
	}
	return picture
}

func masked(alpha, pixels []byte, width, height int) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for index := range width * height {
		pixel := Expand(pixelAt(pixels, index))
		pixel.A = alpha[index]
		picture.SetNRGBA(index%width, index/width, pixel)
	}
	return picture
}

// mask draws an alpha mask in a colour. The opacity guesses of the records are not applied.
func mask(alpha []byte, width, height int, colour themefile.Colour) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	pixel := Expand(colour)
	for index := range width * height {
		pixel.A = alpha[index]
		picture.SetNRGBA(index%width, index/width, pixel)
	}
	return picture
}

func glyphImages(masks [][]byte, widths []uint16, height uint16, colour themefile.Colour) []*image.NRGBA {
	glyphs := make([]*image.NRGBA, len(masks))
	for glyph, glyphMask := range masks {
		glyphs[glyph] = mask(glyphMask, int(widths[glyph]), int(height), colour)
	}
	return glyphs
}

// strip puts the glyphs side by side in one image as wide as their widths added and as high as the first.
func strip(glyphs []*image.NRGBA) *image.NRGBA {
	width := 0
	for _, glyph := range glyphs {
		width += glyph.Bounds().Dx()
	}
	picture := image.NewNRGBA(image.Rect(0, 0, width, glyphs[0].Bounds().Dy()))
	left := 0
	for _, glyph := range glyphs {
		for row := range glyph.Bounds().Dy() {
			for column := range glyph.Bounds().Dx() {
				picture.SetNRGBA(left+column, row, glyph.NRGBAAt(column, row))
			}
		}
		left += glyph.Bounds().Dx()
	}
	return picture
}
