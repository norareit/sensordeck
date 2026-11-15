package themeimage

import (
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/norareit/sensordeck/internal/themefile"
)

const (
	ScreenWidth  = 480
	ScreenHeight = 320
)

// SampleTime is the date and time that the preview shows in a datetime, and SampleValue the value of every number.
var SampleTime = time.Date(2026, 11, 15, 9, 41, 7, 0, time.UTC)

const SampleValue = "42"

// barColour is a colour that no theme of the samples uses, so that a bar, whose fields are not decoded, shows at a
// glance that it is a placeholder.
var barColour = color.NRGBA{R: 255, G: 0, B: 255, A: 255}

// Preview composes the 480 x 320 image of the screen from the records alone: opaque black, then each record over what
// is there in the order of the slots, clipped to the screen and not to its box.
func Preview(theme themefile.Theme) *image.RGBA {
	screen := image.NewRGBA(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	draw.Draw(screen, screen.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	for _, entry := range theme.Entries {
		drawRecord(screen, entry.Record)
	}
	return screen
}

func drawRecord(screen *image.RGBA, record themefile.Record) {
	switch typed := record.(type) {
	case themefile.Background:
		if typed.Offset == 0 {
			fill(screen, box(typed.Head), Expand(typed.FillColourGuess))
			return
		}
		drawAt(screen, int(typed.X), int(typed.Y), Resources(typed)[0])
	case themefile.Image:
		if frames := Resources(typed); len(frames) > 0 {
			drawAt(screen, int(typed.X), int(typed.Y), frames[0])
		}
	case themefile.Text:
		drawAt(screen, int(typed.X), int(typed.Y), Resources(typed)[0])
	case themefile.Number:
		glyphs := glyphImages(typed.Glyphs(), typed.GlyphWidths[:], typed.GlyphHeight, typed.Colour)
		var indexes []int
		for _, character := range SampleValue {
			indexes = append(indexes, indexOf(themefile.NumberGlyphOrder, character))
		}
		drawGlyphs(screen, int(typed.X), int(typed.Y), glyphs, indexes)
	case themefile.Datetime:
		widths := typed.GlyphWidths()
		glyphs := glyphImages(typed.Glyphs(), widths[:], typed.GlyphHeight, typed.Colour)
		drawGlyphs(screen, int(typed.X), int(typed.Y), glyphs, datetimeGlyphs(typed.FormatText(), SampleTime))
	case themefile.Bar:
		drawOutline(screen, box(typed.Head))
	}
}

func box(head themefile.Head) image.Rectangle {
	return image.Rect(int(head.X), int(head.Y), int(head.X)+int(head.Width), int(head.Y)+int(head.Height))
}

func fill(screen *image.RGBA, area image.Rectangle, colour color.NRGBA) {
	draw.Draw(screen, area, image.NewUniform(colour), image.Point{}, draw.Over)
}

func drawAt(screen *image.RGBA, x, y int, picture *image.NRGBA) {
	target := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x, y).Add(picture.Bounds().Size())}
	draw.Draw(screen, target, picture, image.Point{}, draw.Over)
}

// drawOutline draws the inside edge of the box, one pixel wide. A box with no area draws nothing.
func drawOutline(screen *image.RGBA, area image.Rectangle) {
	if area.Empty() {
		return
	}
	fill(screen, image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+1), barColour)
	fill(screen, image.Rect(area.Min.X, area.Max.Y-1, area.Max.X, area.Max.Y), barColour)
	fill(screen, image.Rect(area.Min.X, area.Min.Y, area.Min.X+1, area.Max.Y), barColour)
	fill(screen, image.Rect(area.Max.X-1, area.Min.Y, area.Max.X, area.Max.Y), barColour)
}

// drawGlyphs places the glyphs of the indexes side by side from x, each the next at the right edge of the one before,
// with their tops at y.
func drawGlyphs(screen *image.RGBA, x, y int, glyphs []*image.NRGBA, indexes []int) {
	for _, index := range indexes {
		drawAt(screen, x, y, glyphs[index])
		x += glyphs[index].Bounds().Dx()
	}
}

func indexOf(order string, character rune) int {
	for index, candidate := range []rune(order) {
		if candidate == character {
			return index
		}
	}
	return -1
}

// datetimeGlyphs gives the indexes, in the glyph set of a datetime, of the glyphs that its format draws for the time.
// In a format 1 is the year in four digits and 2 to 6 are the month, day, hour, minute and second in two. The five
// glyph characters stand for their glyphs, and any other byte for nothing.
func datetimeGlyphs(format []byte, moment time.Time) []int {
	fields := map[byte]string{
		'1': moment.Format("2006"), '2': moment.Format("01"), '3': moment.Format("02"),
		'4': moment.Format("15"), '5': moment.Format("04"), '6': moment.Format("05"),
	}
	var indexes []int
	for _, character := range format {
		if digits, isField := fields[character]; isField {
			for _, digit := range digits {
				indexes = append(indexes, int(digit-'0'))
			}
			continue
		}
		if other := indexOf(themefile.DatetimeOtherGlyphOrder[:5], rune(character)); other >= 0 {
			indexes = append(indexes, 10+other)
		}
	}
	return indexes
}
