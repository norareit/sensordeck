package themeimage

import (
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"time"

	"github.com/norareit/sensordeck/internal/themefile"
)

const (
	ScreenWidth  = 480
	ScreenHeight = 320
)

// SampleTime is the date and time that the preview shows in a datetime, and SampleValue the value of every number.
var SampleTime = time.Date(2026, 11, 15, 9, 41, 7, 0, time.UTC)

// SampleNumber is the value of every number, and of every bar as a share, so that a number and a bar of the same
// metric show the same value.
const SampleNumber = 42

var SampleValue = strconv.Itoa(SampleNumber)

// thousandsValue is what the preview draws for a number in the thousands mode: 1234 was seen to show it on the device.
const thousandsValue = "1.2"

// The values of byte 11 of a number.
const (
	alignLeft   = 0
	alignRight  = 2
	alignCentre = 1
)

// Preview composes the 480 x 320 image of the screen from the records alone: opaque black, then each record over what
// is there in the order of the slots. A number is clipped to its box and to the screen, and everything else to the screen
// and not to its box.
func Preview(theme themefile.Theme) *image.RGBA {
	return PreviewWithValues(theme, nil)
}

// PreviewWithValues is Preview with the number of each slot in values drawn with that value, as the rules of drawNumber
// place it, and every other number drawn as the sample.
func PreviewWithValues(theme themefile.Theme, values map[int]int16) *image.RGBA {
	screen := image.NewRGBA(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	draw.Draw(screen, screen.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	for _, entry := range theme.Entries {
		drawRecord(screen, entry, values)
	}
	return screen
}

func drawRecord(screen *image.RGBA, entry themefile.Entry, values map[int]int16) {
	switch typed := entry.Record.(type) {
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
		drawNumber(screen, typed, numberValue(typed, values, entry.Slot))
	case themefile.Datetime:
		widths := typed.GlyphWidths()
		glyphs := glyphImages(typed.Glyphs(), widths[:], typed.GlyphHeight, typed.Colour)
		drawGlyphs(screen, int(typed.X), int(typed.Y), glyphs, datetimeGlyphs(typed.FormatText(), SampleTime))
	case themefile.Bar:
		drawBar(screen, typed)
	}
}

func box(head themefile.Head) image.Rectangle {
	return image.Rect(int(head.X), int(head.Y), int(head.X)+int(head.Width), int(head.Y)+int(head.Height))
}

func fill(screen *image.RGBA, area image.Rectangle, colour color.NRGBA) {
	draw.Draw(screen, area, image.NewUniform(colour), image.Point{}, draw.Over)
}

func drawAt(screen draw.Image, left, top int, picture *image.NRGBA) {
	target := image.Rectangle{Min: image.Pt(left, top), Max: image.Pt(left, top).Add(picture.Bounds().Size())}
	draw.Draw(screen, target, picture, image.Point{}, draw.Over)
}

// drawBar draws the frame, the fill and the empty part of a bar at the sample value. That bytes 14-15 are the fill and
// bytes 16-17 the frame, the rounding down of the fill and the fill from the left of a box of any shape are the
// preview's own choices, which the device check did not settle.
func drawBar(screen *image.RGBA, bar themefile.Bar) {
	area := box(bar.Head)
	if area.Empty() {
		return
	}
	fill(screen, area, Expand(bar.FillOrFrameColour16))
	inside := area.Inset(1)
	if inside.Empty() {
		return
	}
	filled := inside.Dx() * SampleNumber / 100
	fill(screen, image.Rect(inside.Min.X, inside.Min.Y, inside.Min.X+filled, inside.Max.Y), Expand(bar.FillOrFrameColour14))
	fill(screen, image.Rect(inside.Min.X+filled, inside.Min.Y, inside.Max.X, inside.Max.Y), Expand(bar.EmptyColour))
}

// numberValue is the text a number is drawn with: the value of its slot when values holds one, and else the sample, or
// the sample of the thousands mode.
func numberValue(number themefile.Number, values map[int]int16, slot int) string {
	if value, found := values[slot]; found {
		return strconv.Itoa(int(value))
	}
	if number.ThousandsMode != 0 {
		return thousandsValue
	}
	return SampleValue
}

// drawNumber draws the value of a number placed in its box by its alignment and clipped to the box and the screen.
func drawNumber(screen *image.RGBA, number themefile.Number, value string) {
	glyphs := glyphImages(number.Glyphs(), number.GlyphWidths[:], number.GlyphHeight, number.Colour)
	for _, glyph := range glyphs {
		fade(glyph, number.Opacity)
	}
	var indexes []int
	width := 0
	for _, character := range value {
		index := indexOf(themefile.NumberGlyphOrder, character)
		indexes = append(indexes, index)
		width += glyphs[index].Bounds().Dx()
	}
	area := box(number.Head)
	left := area.Min.X
	switch number.Alignment {
	case alignRight:
		left = area.Max.X - width
	case alignCentre:
		// An arithmetic shift rounds towards minus infinity, as a value wider than its box needs.
		left = area.Min.X + (area.Dx()-width)>>1
	}
	clip := screen.SubImage(area).(*image.RGBA)
	drawGlyphs(clip, left, area.Min.Y, glyphs, indexes)
}

// fade multiplies the alpha of every pixel of the picture by opacity over 255, rounded to the nearest whole number.
// 255 is odd, so the quotient is never a half.
func fade(picture *image.NRGBA, opacity byte) {
	for position := 3; position < len(picture.Pix); position += 4 {
		picture.Pix[position] = byte((int(picture.Pix[position])*int(opacity) + 127) / 255)
	}
}

// drawGlyphs places the glyphs of the indexes side by side from left, each the next at the right edge of the one before,
// with their tops at top.
func drawGlyphs(screen draw.Image, left, top int, glyphs []*image.NRGBA, indexes []int) {
	for _, index := range indexes {
		drawAt(screen, left, top, glyphs[index])
		left += glyphs[index].Bounds().Dx()
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
