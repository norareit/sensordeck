// Package background draws the background of the built-in theme minimal-dark. It does no I/O.
package background

import (
	"image"
	"image/color"
)

const (
	width       = 480
	height      = 320
	panelRadius = 10
	panelWidth  = 151
	panelHeight = 123
)

var (
	groundColour  = color.NRGBA{R: 0x0b, G: 0x0f, B: 0x14, A: 255}
	fillColour    = color.NRGBA{R: 0x14, G: 0x1b, B: 0x24, A: 255}
	outlineColour = color.NRGBA{R: 0x1f, G: 0x2a, B: 0x36, A: 255}
)

// roundedRectangle is the rule of R4 of unit 0012: the corners are inclusive and a pixel is taken at its whole
// coordinates, so that the limit radius * (radius + 1) is (radius + 0.5) squared and no floating point is needed.
type roundedRectangle struct {
	left, top, right, bottom, radius int
}

func (rectangle roundedRectangle) holds(column, row int) bool {
	if column < rectangle.left || column > rectangle.right || row < rectangle.top || row > rectangle.bottom {
		return false
	}
	centreX := clamp(column, rectangle.left+rectangle.radius, rectangle.right-rectangle.radius)
	centreY := clamp(row, rectangle.top+rectangle.radius, rectangle.bottom-rectangle.radius)
	distanceX, distanceY := column-centreX, row-centreY
	return distanceX*distanceX+distanceY*distanceY <= rectangle.radius*(rectangle.radius+1)
}

// shrunk is the rectangle 1 pixel inside on every side, with a radius 1 smaller, so that the outline stays 1 pixel wide
// at the corners.
func (rectangle roundedRectangle) shrunk() roundedRectangle {
	return roundedRectangle{rectangle.left + 1, rectangle.top + 1, rectangle.right - 1, rectangle.bottom - 1, rectangle.radius - 1}
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}

func panelAt(left, top int) roundedRectangle {
	return roundedRectangle{left, top, left + panelWidth, top + panelHeight, panelRadius}
}

var panels = []roundedRectangle{
	panelAt(8, 8), panelAt(164, 8), panelAt(320, 8),
	panelAt(8, 140), panelAt(164, 140), panelAt(320, 140),
	{8, 272, 471, 311, panelRadius},
}

// MinimalDark draws the background of minimal-dark: 480 x 320 opaque pixels, by the rule of R4 of unit 0012.
func MinimalDark() *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			picture.SetNRGBA(column, row, colourAt(column, row))
		}
	}
	return picture
}

func colourAt(column, row int) color.NRGBA {
	for _, panel := range panels {
		if panel.shrunk().holds(column, row) {
			return fillColour
		}
		if panel.holds(column, row) {
			return outlineColour
		}
	}
	return groundColour
}
