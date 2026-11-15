// Package themedump turns a theme file into the contents of the files of a dump: theme.json, a PNG of each resource and
// preview.png. It does no I/O, so that the command writes the files only after the whole dump has succeeded.
package themedump

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

// File is a file of the dump.
type File struct {
	Name    string
	Content []byte
}

// Dump gives the files of the dump in this order: theme.json, the PNG files of the records in the order of the entries
// and of their images, and preview.png; and the warnings of the reader. An error is that of the reader.
func Dump(file []byte) ([]File, []string, error) {
	theme, warnings, err := themefile.Parse(file)
	if err != nil {
		return nil, nil, err
	}
	var pictures []File
	records := []recordJSON{}
	for _, entry := range theme.Entries {
		names := []string{}
		for number, picture := range nonEmpty(themeimage.Resources(entry.Record)) {
			encoded, err := encodePNG(picture)
			if err != nil {
				return nil, nil, err
			}
			name := pictureName(entry, number+1)
			names = append(names, name)
			pictures = append(pictures, File{Name: name, Content: encoded})
		}
		records = append(records, describe(entry, names))
	}
	if warnings == nil {
		warnings = []string{}
	}
	description := themeJSON{
		FileSize: theme.FileSize,
		Header:   headerJSON{Byte0Unknown: theme.Header[0], Raw: theme.Header[:]},
		Records:  records,
		Warnings: warnings,
	}
	content, err := encode(description)
	if err != nil {
		return nil, nil, err
	}
	preview, err := encodePNG(themeimage.Preview(theme))
	if err != nil {
		return nil, nil, err
	}
	files := append([]File{{Name: "theme.json", Content: content}}, pictures...)
	return append(files, File{Name: "preview.png", Content: preview}), warnings, nil
}

// nonEmpty leaves out the images with no pixels, of width 0 or height 0, which a PNG cannot hold.
func nonEmpty(pictures []*image.NRGBA) []*image.NRGBA {
	var kept []*image.NRGBA
	for _, picture := range pictures {
		if !picture.Bounds().Empty() {
			kept = append(kept, picture)
		}
	}
	return kept
}

// pictureName names the number-th image (from 1) of the record after its slot.
func pictureName(entry themefile.Entry, number int) string {
	name := themefile.TypeName(entry.Record.Encode()[0])
	switch entry.Record.(type) {
	case themefile.Startup, themefile.Image:
		return fmt.Sprintf("slot%02d-%s-frame%d.png", entry.Slot, name, number)
	case themefile.Number, themefile.Datetime:
		return fmt.Sprintf("slot%02d-%s-glyphs.png", entry.Slot, name)
	}
	return fmt.Sprintf("slot%02d-%s.png", entry.Slot, name)
}

func encodePNG(picture image.Image) ([]byte, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		return nil, fmt.Errorf("cannot encode a PNG image: %w", err)
	}
	return buffer.Bytes(), nil
}
