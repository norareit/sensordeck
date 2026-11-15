package themecompile

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io/fs"
	"path"
	"slices"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomediumitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gosmallcaps"
	"golang.org/x/image/font/gofont/gosmallcapsitalic"
	"golang.org/x/image/font/opentype"
)

// goFonts are the twelve fonts built into the executable, in the order the messages list them.
type goFont struct {
	name string
	ttf  []byte
}

var goFonts = []goFont{
	{"go-regular", goregular.TTF}, {"go-bold", gobold.TTF}, {"go-italic", goitalic.TTF},
	{"go-bold-italic", gobolditalic.TTF}, {"go-medium", gomedium.TTF}, {"go-medium-italic", gomediumitalic.TTF},
	{"go-mono", gomono.TTF}, {"go-mono-bold", gomonobold.TTF}, {"go-mono-italic", gomonoitalic.TTF},
	{"go-mono-bold-italic", gomonobolditalic.TTF}, {"go-smallcaps", gosmallcaps.TTF},
	{"go-smallcaps-italic", gosmallcapsitalic.TTF},
}

func findGoFont(name string) (goFont, bool) {
	position := slices.IndexFunc(goFonts, func(candidate goFont) bool { return candidate.name == name })
	if position < 0 {
		return goFont{}, false
	}
	return goFonts[position], true
}

func goFontNames() string {
	names := make([]string, len(goFonts))
	for position, candidate := range goFonts {
		names[position] = candidate.name
	}
	return strings.Join(names, ", ")
}

// cause gives the error inside a *fs.PathError, so that the path is named once.
func cause(err error) error {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return pathError.Err
	}
	return err
}

// readInside reads a file named by the theme. The path is cleaned first, so that a/../b.png names b.png, and is then
// refused unless it is inside the folder. The folder, an os.Root in the command, refuses any name that leaves it by a
// symbolic link.
func readInside(folder fs.FS, place, given string) ([]byte, error) {
	cleaned := path.Clean(given)
	if !fs.ValidPath(cleaned) {
		return nil, problem(place, "%q is not a path inside the folder", given)
	}
	content, err := fs.ReadFile(folder, cleaned)
	if err != nil {
		return nil, problem(place, "cannot read %s: %s", given, cause(err))
	}
	return content, nil
}

// pngSize reads the size of a PNG from its header alone, so that a large one is refused before its pixels are decoded.
func pngSize(place, given string, content []byte) (width, height int, err error) {
	header, err := png.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return 0, 0, problem(place, "cannot decode %s: %s", given, err)
	}
	return header.Width, header.Height, nil
}

func decodePNG(place, given string, content []byte) (image.Image, error) {
	picture, err := png.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, problem(place, "cannot decode %s: %s", given, err)
	}
	return picture, nil
}

// loadFont gives the face of a font of the theme: a Go font, or a TTF or OTF file of the folder.
func loadFont(folder fs.FS, spec fontSpec) (font.Face, error) {
	place := "font " + spec.name
	var content []byte
	given := spec.file
	if spec.builtin != "" {
		given = spec.builtin
		builtin, _ := findGoFont(spec.builtin)
		content = builtin.ttf
	} else {
		var err error
		if content, err = readInside(folder, place, spec.file); err != nil {
			return nil, err
		}
	}
	if bytes.HasPrefix(content, []byte("ttcf")) {
		return nil, problem(place, "%s holds a font collection, not one font", given)
	}
	parsed, err := opentype.Parse(content)
	if err != nil {
		return nil, problem(place, "cannot decode %s: %s", given, err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(spec.size), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, problem(place, "cannot decode %s: %s", given, err)
	}
	return face, nil
}

func isGoFont(name string) bool {
	_, found := findGoFont(name)
	return found
}
