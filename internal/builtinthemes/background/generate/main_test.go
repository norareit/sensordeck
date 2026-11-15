package main

import (
	"bytes"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/builtinthemes/background"
)

func TestRunWritesTheBackgroundAsAPNG(test *testing.T) {
	path := filepath.Join(test.TempDir(), "background.png")
	var stderr bytes.Buffer
	if status := run([]string{path}, &stderr); status != 0 || stderr.Len() != 0 {
		test.Fatalf("status %d, stderr %q", status, stderr.String())
	}
	file, err := os.Open(path)
	if err != nil {
		test.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	decoded, err := png.Decode(file)
	if err != nil {
		test.Fatal(err)
	}
	drawn := background.MinimalDark()
	if decoded.Bounds() != drawn.Bounds() {
		test.Fatalf("bounds %v", decoded.Bounds())
	}
	for _, corner := range [][2]int{{0, 0}, {80, 70}, {80, 8}, {240, 290}} {
		got := color.NRGBAModel.Convert(decoded.At(corner[0], corner[1])).(color.NRGBA)
		if want := drawn.NRGBAAt(corner[0], corner[1]); got != want {
			test.Errorf("pixel %v is %v, want %v", corner, got, want)
		}
	}
}

func TestRunFailsOnAPathItCannotWrite(test *testing.T) {
	path := filepath.Join(test.TempDir(), "missing", "background.png")
	var stderr bytes.Buffer
	if status := run([]string{path}, &stderr); status != 1 || !strings.HasPrefix(stderr.String(), "generate: ") {
		test.Fatalf("status %d, stderr %q", status, stderr.String())
	}
}

func TestRunNeedsExactlyOneArgument(test *testing.T) {
	for _, arguments := range [][]string{nil, {"a", "b"}} {
		var stderr bytes.Buffer
		if status := run(arguments, &stderr); status != 1 || stderr.String() != "generate: the output path is the one argument\n" {
			test.Errorf("%q: status %d, stderr %q", arguments, status, stderr.String())
		}
	}
}
