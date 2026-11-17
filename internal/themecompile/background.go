package themecompile

import (
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

// drawTexts gives the plane of a background with the texts drawn over it in order. The preview does the drawing, so that
// a text in the background looks as the same text as a record looks in the preview. The result is rounded to RGB565
// once, and the plane given is left as it is.
func drawTexts(background themefile.Background, plane []byte, texts []themefile.Text) []byte {
	background.Resource = plane
	entries := []themefile.Entry{{Slot: 1, Record: background}}
	for position, text := range texts {
		entries = append(entries, themefile.Entry{Slot: position + 2, Record: text})
	}
	screen := themeimage.Preview(themefile.Theme{Entries: entries})
	drawn := make([]byte, 0, len(plane))
	for y := range screenHeight {
		for x := range screenWidth {
			pixel := screen.RGBAAt(x, y)
			drawn = appendPixel(drawn, rgb565(pixel.R, pixel.G, pixel.B))
		}
	}
	return drawn
}
