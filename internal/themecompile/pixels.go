package themecompile

import (
	"image"
	"image/color"

	"github.com/norareit/sensordeck/internal/themefile"
)

const (
	screenWidth  = 480
	screenHeight = 320
)

// rgb565 rounds each channel to the nearest of its bits: (value*31+127)/255 for red and blue, (value*63+127)/255 for
// green.
func rgb565(red, green, blue uint8) themefile.Colour {
	return themefile.Colour((uint16(red)*31+127)/255<<11 | (uint16(green)*63+127)/255<<5 | (uint16(blue)*31+127)/255)
}

// nonPremultiplied gives the colour of a pixel of a decoded PNG with its alpha, 8 bits each. The pixels with alpha of
// the PNG decoder are already non-premultiplied, so they are not passed through the premultiplied form, which would
// lose colour in the pixels that are not opaque.
func nonPremultiplied(picture image.Image, x, y int) color.NRGBA {
	switch pixel := picture.At(x, y).(type) {
	case color.NRGBA:
		return pixel
	case color.NRGBA64:
		return color.NRGBA{R: uint8(pixel.R >> 8), G: uint8(pixel.G >> 8), B: uint8(pixel.B >> 8), A: uint8(pixel.A >> 8)}
	default:
		return color.NRGBAModel.Convert(pixel).(color.NRGBA)
	}
}

func appendPixel(plane []byte, colour themefile.Colour) []byte {
	return append(plane, byte(colour), byte(colour>>8))
}

// solidPlane gives 480 x 320 pixels of one colour, each two bytes with the low byte first.
func solidPlane(colour themefile.Colour) []byte {
	plane := make([]byte, 0, 2*screenWidth*screenHeight)
	for range screenWidth * screenHeight {
		plane = appendPixel(plane, colour)
	}
	return plane
}

// backgroundPlane gives the pixels of a picture composited over opaque black.
func backgroundPlane(picture image.Image) []byte {
	bounds := picture.Bounds()
	plane := make([]byte, 0, 2*bounds.Dx()*bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := nonPremultiplied(picture, x, y)
			over := func(channel uint8) uint8 { return uint8((uint16(channel)*uint16(pixel.A) + 127) / 255) }
			plane = appendPixel(plane, rgb565(over(pixel.R), over(pixel.G), over(pixel.B)))
		}
	}
	return plane
}

// imagePlanes gives the alpha plane of a picture followed by its colour plane, whose pixels leave the alpha out.
func imagePlanes(picture image.Image) []byte {
	bounds := picture.Bounds()
	area := bounds.Dx() * bounds.Dy()
	planes := make([]byte, 0, 3*area)
	colours := make([]byte, 0, 2*area)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := nonPremultiplied(picture, x, y)
			planes = append(planes, pixel.A)
			colours = appendPixel(colours, rgb565(pixel.R, pixel.G, pixel.B))
		}
	}
	return append(planes, colours...)
}
