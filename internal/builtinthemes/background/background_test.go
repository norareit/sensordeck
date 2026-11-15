package background

import (
	"image/color"
	"testing"
)

var (
	ground  = color.NRGBA{R: 0x0b, G: 0x0f, B: 0x14, A: 255}
	fill    = color.NRGBA{R: 0x14, G: 0x1b, B: 0x24, A: 255}
	outline = color.NRGBA{R: 0x1f, G: 0x2a, B: 0x36, A: 255}
)

func TestMinimalDarkIsOpaqueAndHasTheSizeOfTheScreen(test *testing.T) {
	picture := MinimalDark()
	if picture.Bounds().Dx() != 480 || picture.Bounds().Dy() != 320 {
		test.Fatalf("bounds %v", picture.Bounds())
	}
	for row := 0; row < 320; row++ {
		for column := 0; column < 480; column++ {
			if picture.NRGBAAt(column, row).A != 255 {
				test.Fatalf("pixel (%d, %d) is not opaque", column, row)
			}
		}
	}
}

func TestMinimalDarkFollowsTheRuleAtChosenPixels(test *testing.T) {
	cases := []struct {
		name string
		column, row int
		want color.NRGBA
	}{
		{"a corner of the background", 0, 0, ground},
		{"inside the first panel", 80, 70, fill},
		{"the top edge of the first panel", 80, 8, outline},
		{"the left edge of the first panel", 8, 70, outline},
		{"the rounded outline of the first panel", 11, 11, outline},
		{"just outside the rounded corner", 8, 8, ground},
		{"just outside the rounded corner, diagonally", 9, 9, ground},
		{"inside the bar", 240, 290, fill},
		{"between the first two panels", 160, 70, ground},
	}
	picture := MinimalDark()
	for _, each := range cases {
		if got := picture.NRGBAAt(each.column, each.row); got != each.want {
			test.Errorf("%s, (%d, %d): got %v, want %v", each.name, each.column, each.row, got, each.want)
		}
	}
}
