package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/norareit/sensordeck/internal/builtinthemes"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

const previewImagePath = "docs/minimal-dark.png"

const previewCommand = "sensordeck theme preview minimal-dark docs/minimal-dark.png"

// TestThePreviewImageIsWhatTheCodeDrawsNow compares decoded pixels rather than bytes, so that another PNG encoder
// that compresses the same pixels differently does not fail it.
func TestThePreviewImageIsWhatTheCodeDrawsNow(test *testing.T) {
	folder, found := builtinthemes.Folder("minimal-dark")
	if !found {
		test.Fatal("the built-in theme minimal-dark is missing")
	}
	compiled, err := themecompile.Compile(folder)
	if err != nil {
		test.Fatal(err)
	}
	theme, _, err := themefile.Parse(compiled.File)
	if err != nil {
		test.Fatal(err)
	}
	want := themeimage.Preview(theme)

	if mismatch := compareWithImage(want); mismatch != "" {
		test.Errorf("%s %s: write the image again with %q", previewImagePath, mismatch, previewCommand)
	}
}

// compareWithImage gives what differs between the committed image and want, and nothing when they are the same.
func compareWithImage(want image.Image) string {
	file, err := os.Open(previewImagePath)
	if err != nil {
		return err.Error()
	}
	defer func() { _ = file.Close() }()
	got, err := png.Decode(file)
	if err != nil {
		return err.Error()
	}
	if got.Bounds() != want.Bounds() {
		return fmt.Sprintf("is %v and the preview is %v", got.Bounds(), want.Bounds())
	}
	bounds := want.Bounds()
	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for column := bounds.Min.X; column < bounds.Max.X; column++ {
			if color.RGBAModel.Convert(got.At(column, row)) != color.RGBAModel.Convert(want.At(column, row)) {
				return fmt.Sprintf("differs from the preview at pixel (%d, %d)", column, row)
			}
		}
	}
	return ""
}
