package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/norareit/sensordeck/internal/builtinthemes"
	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

// previewImages is the table of the committed preview images, each the preview of the built-in theme of its row.
var previewImages = []struct {
	themeName string
	path      string
}{
	{"minimal-dark", "docs/minimal-dark.png"},
	{"minimal-dark-probes", "docs/minimal-dark-probes.png"},
}

// TestThePreviewImagesAreWhatTheCodeDrawsNow compares decoded pixels rather than bytes, so that another PNG encoder
// that compresses the same pixels differently does not fail it.
func TestThePreviewImagesAreWhatTheCodeDrawsNow(test *testing.T) {
	for _, preview := range previewImages {
		test.Run(preview.themeName, func(test *testing.T) {
			folder, found := builtinthemes.Folder(preview.themeName)
			if !found {
				test.Fatalf("the built-in theme %s is missing", preview.themeName)
			}
			compiled, err := themecompile.Compile(folder)
			if err != nil {
				test.Fatal(err)
			}
			theme, _, err := themefile.Parse(compiled.File)
			if err != nil {
				test.Fatal(err)
			}
			// The preview draws the normal state, so the widget of the alarm state of each alarm number is hidden.
			hidden := make(map[int]int16, len(compiled.AlarmSlots))
			for _, slot := range compiled.AlarmSlots {
				hidden[slot] = config.HidingValue
			}
			want := themeimage.PreviewWithValues(theme, hidden)

			if mismatch := compareWithImage(preview.path, want); mismatch != "" {
				test.Errorf("%s %s: write the image again with %q", preview.path, mismatch,
					"sensordeck theme preview "+preview.themeName+" "+preview.path)
			}
		})
	}
}

// compareWithImage gives what differs between the committed image at path and want, and nothing when they are the same.
func compareWithImage(path string, want image.Image) string {
	file, err := os.Open(path)
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
