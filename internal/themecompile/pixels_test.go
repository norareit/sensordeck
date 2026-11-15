package themecompile

import (
	"image"
	"image/color"
	"testing"

	"github.com/norareit/sensordeck/internal/themefile"
)

func TestRGB565(test *testing.T) {
	cases := []struct {
		red, green, blue uint8
		want             themefile.Colour
	}{
		{0, 0, 0, 0x0000},
		{255, 255, 255, 0xffff},
		{5, 0, 0, 1 << 11}, // 5*31+127 = 282, 282/255 = 1
		{4, 0, 0, 0},       // 4*31+127 = 251, 251/255 = 0
		{0, 3, 0, 1 << 5},  // 3*63+127 = 316, 316/255 = 1
		{0, 2, 0, 0},       // 2*63+127 = 253, 253/255 = 0
		{0, 0, 5, 1},       // as red
	}
	for _, each := range cases {
		if got := rgb565(each.red, each.green, each.blue); got != each.want {
			test.Errorf("%d,%d,%d: got %04x, want %04x", each.red, each.green, each.blue, got, each.want)
		}
	}
}

func TestBackgroundPlaneCompositesOverBlack(test *testing.T) {
	picture := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	picture.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	picture.SetNRGBA(1, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	plane := backgroundPlane(picture)
	first := rgb565(128, 0, 0)
	second := rgb565(10, 20, 30)
	want := []byte{byte(first), byte(first >> 8), byte(second), byte(second >> 8)}
	if string(plane) != string(want) {
		test.Errorf("got %x, want %x", plane, want)
	}
}

func TestImagePlanesKeepTheColourNonPremultiplied(test *testing.T) {
	picture := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	picture.SetNRGBA(0, 0, color.NRGBA{R: 200, A: 0x40})
	planes := imagePlanes(picture)
	colour := rgb565(200, 0, 0)
	want := []byte{0x40, byte(colour), byte(colour >> 8)}
	if string(planes) != string(want) {
		test.Errorf("got %x, want %x", planes, want)
	}
}

func Test16BitChannelsAreReducedTo8(test *testing.T) {
	picture := image.NewNRGBA64(image.Rect(0, 0, 2, 1))
	picture.SetNRGBA64(0, 0, color.NRGBA64{R: 0xff00, G: 0xff00, B: 0xff00, A: 0xffff})
	picture.SetNRGBA64(1, 0, color.NRGBA64{R: 0x00ff, G: 0x00ff, B: 0x00ff, A: 0xffff})
	plane := backgroundPlane(picture)
	if string(plane) != "\xff\xff\x00\x00" {
		test.Errorf("got %x", plane)
	}
}

func TestSolidPlaneIsLittleEndian(test *testing.T) {
	plane := solidPlane(0x1234)
	if len(plane) != 2*480*320 || plane[0] != 0x34 || plane[1] != 0x12 || plane[len(plane)-1] != 0x12 {
		test.Errorf("%d bytes, starts %x", len(plane), plane[:2])
	}
}
