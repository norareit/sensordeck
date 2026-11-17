package themecompile

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/norareit/sensordeck/internal/faketheme"
)

const (
	solidTheme   = "name: t\nbackground: \"#ffffff\"\n" + goodFonts
	widgetOffset = 0x1000 + 2*480*320
)

func compileFiles(test *testing.T, theme string, files map[string][]byte) Result {
	test.Helper()
	folder := fstest.MapFS{"theme.yaml": {Data: []byte(theme)}}
	for name, content := range files {
		folder[name] = &fstest.MapFile{Data: content}
	}
	result, err := Compile(folder)
	if err != nil {
		test.Fatal(err)
	}
	return result
}

// solidBackgroundSlot is slot 1 of a theme with the background "#ffffff", written by hand from the bytes of R14.
func solidBackgroundSlot() [64]byte {
	slot := faketheme.Head(0x81, 0, 0, 0, 0, 480, 320)
	faketheme.Put16(&slot, 12, 0xffff)
	faketheme.Put32(&slot, 14, 0x1000)
	slot[18] = 1
	return slot
}

// checkTable compares the whole record table: slot 0, slot 1 and the widget slots, and zero in every other.
func checkTable(test *testing.T, file []byte, widgets ...[64]byte) {
	test.Helper()
	var want [4096]byte
	want[0] = 0x96
	slot := solidBackgroundSlot()
	copy(want[64:], slot[:])
	for position, widget := range widgets {
		copy(want[(position+2)*64:], widget[:])
	}
	for number := range 64 {
		if !bytes.Equal(file[number*64:(number+1)*64], want[number*64:(number+1)*64]) {
			test.Errorf("slot %d: got %x\nwant %x", number, file[number*64:(number+1)*64], want[number*64:(number+1)*64])
		}
	}
}

func TestBackgroundOnly(test *testing.T) {
	result := compileFiles(test, solidTheme, nil)
	checkTable(test, result.File)
	if len(result.File) != 0x1000+2*480*320 {
		test.Errorf("%d bytes", len(result.File))
	}
	for _, pixel := range [][]byte{result.File[0x1000:0x1002], result.File[len(result.File)-2:]} {
		if pixel[0] != 0xff || pixel[1] != 0xff {
			test.Errorf("pixel %x", pixel)
		}
	}
}

func TestTextRecord(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	width := glyphs.textWidth("Hi")
	result := compileFiles(test, solidTheme+"widgets:\n  - {type: text, font: body, text: Hi, x: 10, y: 20}\n", nil)
	slot := faketheme.Head(0x93, 1, 0, 10, 20, uint16(width), uint16(glyphs.height))
	faketheme.Put32(&slot, 11, widgetOffset)
	faketheme.Put16(&slot, 15, 0xffff)
	slot[17] = 0xff
	checkTable(test, result.File, slot)
	if len(result.File) != widgetOffset+width*glyphs.height {
		test.Errorf("%d bytes", len(result.File))
	}
}

// numberSlot is the slot of the number of numberTheme with `keys` after its metric, and byte 11, 14 and 15 of today.
func numberSlot(glyphs glyphFont) [64]byte {
	slot := faketheme.Head(0x92, 1, 1, 1, 2, uint16(4*glyphs.digitWidth), uint16(glyphs.height))
	slot[11], slot[15] = 1, 0xff
	faketheme.Put16(&slot, 12, 0xffff)
	faketheme.Put32(&slot, 16, widgetOffset)
	faketheme.Put16(&slot, 20, uint16(glyphs.height))
	for glyph := range 10 {
		faketheme.Put16(&slot, 22+2*glyph, uint16(glyphs.digitWidth))
	}
	faketheme.Put16(&slot, 42, uint16(glyphs.advance('.')))
	faketheme.Put16(&slot, 44, uint16(glyphs.advance('-')))
	return slot
}

func numberTheme(options string) string {
	return solidTheme + "widgets:\n  - {type: number, font: body, metric: cpu.temp, digits: 4, x: 1, y: 2" + options + "}\n"
}

func TestNumberRecord(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	result := compileFiles(test, numberTheme(""), nil)
	checkTable(test, result.File, numberSlot(glyphs))
	if want := widgetOffset + glyphs.height*(10*glyphs.digitWidth+glyphs.advance('.')+glyphs.advance('-')); len(result.File) != want {
		test.Errorf("%d bytes, want %d", len(result.File), want)
	}
}

func TestNumberRecordOptions(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	cases := []struct {
		options  string
		position int
		value    byte
	}{
		{", align: left", 11, 0}, {", align: center", 11, 1}, {", align: right", 11, 2},
		{", opacity: 0", 15, 0}, {", opacity: 0.5", 15, 128}, {", opacity: 1", 15, 255},
		{", thousands: true", 14, 1}, {", thousands: false", 14, 0},
	}
	for _, each := range cases {
		slot := numberSlot(glyphs)
		slot[each.position] = each.value
		checkTable(test, compileFiles(test, numberTheme(each.options), nil).File, slot)
	}
}

func barTheme(options string) string {
	return solidTheme + "widgets:\n  - {type: bar, metric: cpu.util, x: 3, y: 4, width: 100, height: 10, color: \"#00ff00\"" + options + "}\n"
}

func barSlot(empty uint16) [64]byte {
	slot := faketheme.Head(0x8b, 1, 1, 3, 4, 100, 10)
	slot[11] = 1
	faketheme.Put16(&slot, 12, empty)
	faketheme.Put16(&slot, 14, 0x07e0)
	faketheme.Put16(&slot, 16, 0x07e0)
	faketheme.Put16(&slot, 18, 0x76b0)
	faketheme.Put16(&slot, 26, 0x76b0)
	return slot
}

func TestBarRecord(test *testing.T) {
	with := compileFiles(test, barTheme(", background: \"#ff0000\""), nil)
	checkTable(test, with.File, barSlot(0xf800))
	without := compileFiles(test, barTheme(""), nil)
	checkTable(test, without.File, barSlot(0))
	if len(without.File) != 0x1000+2*480*320 {
		test.Errorf("%d bytes", len(without.File))
	}
}

func TestABarBeforeANumberMovesNoOffset(test *testing.T) {
	theme := solidTheme + "widgets:\n  - {type: bar, metric: cpu.util, x: 0, y: 0, width: 5, height: 5, color: \"#00ff00\"}\n" +
		"  - {type: number, font: body, metric: cpu.util, x: 0, y: 10}\n"
	result := compileFiles(test, theme, nil)
	if offset := binary.BigEndian.Uint32(result.File[3*64+16:]); offset != widgetOffset {
		test.Errorf("the number starts at %#x, want %#x", offset, widgetOffset)
	}
}

func TestABarOutsideTheScreenIsRefused(test *testing.T) {
	theme := solidTheme + "widgets:\n  - {type: bar, metric: cpu.util, x: 400, y: 0, width: 81, height: 5, color: \"#00ff00\"}\n"
	if message := compileError(test, theme, nil); !strings.HasPrefix(message, "widget 1 (bar): the box at 400,0 of 81 x 5 is not within the screen") {
		test.Errorf("got %q", message)
	}
}

func TestDatetimeRecord(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	result := compileFiles(test, solidTheme+"widgets:\n  - {type: datetime, font: body, format: \"HH:mm\", x: 0, y: 0}\n", nil)
	slot := faketheme.Head(0x8e, 1, 21, 0, 0, uint16(4*glyphs.digitWidth+glyphs.advance(':')), uint16(glyphs.height))
	slot[11], slot[14] = 1, 0xff
	faketheme.Put16(&slot, 12, 0xffff)
	faketheme.Put32(&slot, 15, widgetOffset)
	faketheme.Put16(&slot, 19, uint16(glyphs.height))
	faketheme.Put16(&slot, 21, uint16(glyphs.digitWidth))
	for position, character := range ".-/ :" {
		faketheme.Put16(&slot, 23+2*position, uint16(glyphs.advance(character)))
	}
	for position := 5; position < 11; position++ {
		faketheme.Put16(&slot, 23+2*position, uint16(glyphs.digitWidth))
	}
	copy(slot[45:], "4:5")
	checkTable(test, result.File, slot)
	others := glyphs.advance('.') + glyphs.advance('-') + glyphs.advance('/') + glyphs.advance(' ') + glyphs.advance(':')
	if want := widgetOffset + glyphs.height*(16*glyphs.digitWidth+others); len(result.File) != want {
		test.Errorf("%d bytes, want %d", len(result.File), want)
	}
}

func TestImageRecord(test *testing.T) {
	picture := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	picture.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 7})
	result := compileFiles(test, solidTheme+"widgets:\n  - {type: image, file: i.png, x: 5, y: 6}\n", map[string][]byte{"i.png": encodePNG(test, picture)})
	slot := faketheme.Head(0x84, 1, 0, 5, 6, 3, 2)
	faketheme.Put32(&slot, 11, widgetOffset)
	slot[15], slot[16] = 1, 1
	checkTable(test, result.File, slot)
	resource := result.File[widgetOffset:]
	if len(resource) != 3*6 || resource[0] != 7 || resource[6] != 0x00 || resource[7] != 0xf8 {
		test.Errorf("resource %x", resource)
	}
}

func TestPNGBackgroundIsCompositedOverBlack(test *testing.T) {
	picture := image.NewNRGBA(image.Rect(0, 0, 480, 320))
	picture.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	result := compileFiles(test, "name: t\nbackground: b.png\n", map[string][]byte{"b.png": encodePNG(test, picture)})
	first := rgb565(128, 0, 0)
	if result.File[0x1000] != byte(first) || result.File[0x1001] != byte(first>>8) {
		test.Errorf("pixel %x", result.File[0x1000:0x1002])
	}
	if result.File[12] != 0 || result.File[64+12] != 0 || result.File[64+13] != 0 {
		test.Errorf("fill colour %x", result.File[64+12:64+14])
	}
}

func TestSecondWidgetHasIndex2(test *testing.T) {
	result := compileFiles(test, solidTheme+"widgets:\n  - {type: text, font: body, text: a, x: 0, y: 0}\n  - {type: text, font: body, text: b, x: 0, y: 30}\n", nil)
	if result.File[3*64+1] != 2 || result.File[2*64+1] != 1 {
		test.Errorf("indexes %d, %d", result.File[2*64+1], result.File[3*64+1])
	}
}
