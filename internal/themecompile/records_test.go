package themecompile

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themefile"
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

// TestTextRecord is a text over an earlier number, which stays a record in the slot after the number.
func TestTextRecord(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	width := glyphs.textWidth("Hi")
	theme := solidTheme + "widgets:\n  - {type: number, font: body, metric: cpu.temp, digits: 4, x: 10, y: 20}\n" +
		"  - {type: text, font: body, text: Hi, x: 10, y: 20}\n"
	result := compileFiles(test, theme, nil)
	numberSet := glyphs.height * (10*glyphs.digitWidth + glyphs.advance('.') + glyphs.advance('-'))
	number := numberSlot(glyphs)
	copy(number[3:], []byte{0, 10, 0, 20})
	slot := faketheme.Head(0x93, 2, 0, 10, 20, uint16(width), uint16(glyphs.height))
	faketheme.Put32(&slot, 11, uint32(widgetOffset+numberSet))
	faketheme.Put16(&slot, 15, 0xffff)
	slot[17] = 0xff
	checkTable(test, result.File, number, slot)
	if len(result.File) != widgetOffset+numberSet+width*glyphs.height {
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
	result := compileFiles(test, solidTheme+"widgets:\n  - {type: number, font: body, metric: cpu.temp, x: 0, y: 0}\n  - {type: number, font: body, metric: gpu.temp, x: 0, y: 30}\n", nil)
	if result.File[3*64+1] != 2 || result.File[2*64+1] != 1 {
		test.Errorf("indexes %d, %d", result.File[2*64+1], result.File[3*64+1])
	}
}

const alarmFolderTheme = solidTheme + "widgets:\n" +
	"  - {type: text, font: body, text: hi, x: 0, y: 0}\n" +
	"  - {type: number, font: body, metric: gpu.temp, digits: 4, x: 1, y: 2, opacity: 0.5, alarm: {above: 80, color: \"#ff3030\"}}\n" +
	"  - {type: number, font: body, metric: cpu.temp, digits: 4, x: 1, y: 60}\n"

// alarmSlot is a slot of an alarm number written by hand: left-aligned, the glyph of the minus as wide as the box.
func alarmSlot(glyphs glyphFont, index, tag byte, colour, offset uint32) [64]byte {
	slot := faketheme.Head(0x92, index, tag, 1, 2, uint16(4*glyphs.digitWidth), uint16(glyphs.height))
	slot[15] = 128
	faketheme.Put16(&slot, 12, uint16(colour))
	faketheme.Put32(&slot, 16, offset)
	faketheme.Put16(&slot, 20, uint16(glyphs.height))
	for glyph := range 10 {
		faketheme.Put16(&slot, 22+2*glyph, uint16(glyphs.digitWidth))
	}
	faketheme.Put16(&slot, 42, uint16(glyphs.advance('.')))
	faketheme.Put16(&slot, 44, uint16(4*glyphs.digitWidth))
	return slot
}

func TestAlarmNumberIsTwoRecords(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	result := compileFiles(test, alarmFolderTheme, nil)
	theme, _, err := themefile.Parse(result.File)
	if err != nil || len(theme.Entries) != 4 {
		test.Fatalf("%v, %d entries", err, len(theme.Entries))
	}
	setSize := glyphs.height * (10*glyphs.digitWidth + glyphs.advance('.') + 4*glyphs.digitWidth)
	normalOffset := uint32(widgetOffset)
	alarmOffset := normalOffset + uint32(setSize)
	red := rgb565(0xff, 0x30, 0x30)
	normal := alarmSlot(glyphs, 1, 2, uint32(rgb565(0xff, 0xff, 0xff)), normalOffset)
	alarm := alarmSlot(glyphs, 2, 3, uint32(red), alarmOffset)
	for slot, want := range map[int][64]byte{2: normal, 3: alarm} {
		if got := [64]byte(result.File[slot*64 : slot*64+64]); got != want {
			test.Errorf("slot %d: got %x\nwant %x", slot, got, want)
		}
	}
	after := result.Records[3].(themefile.Number)
	if after.Index != 3 || after.Tag != 1 || after.Offset != alarmOffset+uint32(setSize) || after.Alignment != 1 {
		test.Errorf("the number after: %+v", after.Head)
	}
	if !reflect.DeepEqual(result.AlarmSlots, []int{3}) || len(result.Alarms) != 1 || result.Alarms[0].Normal != 2 || result.Alarms[0].Alarm != 3 {
		test.Errorf("alarm slots %v, pairs %+v", result.AlarmSlots, result.Alarms)
	}
}

func TestTheGlyphSetsOfAnAlarmNumberHideTheMinus(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	result := compileFiles(test, alarmFolderTheme, nil)
	normal, alarm := result.Records[1].(themefile.Number), result.Records[2].(themefile.Number)
	if !bytes.Equal(normal.Resource, alarm.Resource) || normal.GlyphWidths != alarm.GlyphWidths {
		test.Error("the two glyph sets differ")
	}
	plain := compileFiles(test, numberTheme(""), nil).Records[1].(themefile.Number)
	masks, plainMasks := normal.Glyphs(), plain.Glyphs()
	for position := range 12 {
		if position == 11 {
			continue
		}
		if !bytes.Equal(masks[position], plainMasks[position]) || normal.GlyphWidths[position] != plain.GlyphWidths[position] {
			test.Errorf("glyph %d differs from that of a number without alarm", position)
		}
	}
	if normal.GlyphWidths[11] != uint16(4*glyphs.digitWidth) || len(masks[11]) != 4*glyphs.digitWidth*glyphs.height ||
		!bytes.Equal(masks[11], make([]byte, len(masks[11]))) {
		test.Errorf("minus: width %d, %d bytes", normal.GlyphWidths[11], len(masks[11]))
	}
}

func TestANumberWithoutAlarmHasNoAlarmSlots(test *testing.T) {
	if result := compileFiles(test, numberTheme(""), nil); result.AlarmSlots != nil || result.Alarms != nil {
		test.Errorf("%v, %v", result.AlarmSlots, result.Alarms)
	}
}

// noMinus is a face without the glyph of the minus.
type noMinus struct{ font.Face }

func (face noMinus) GlyphAdvance(character rune) (fixed.Int26_6, bool) {
	if character == '-' {
		return 0, false
	}
	return face.Face.GlyphAdvance(character)
}

func TestAnAlarmNumberNeedsTheMinusGlyph(test *testing.T) {
	spec := widgetSpec{number: 1, kind: "number", digits: 3, metric: "cpu.temp", alarm: &alarmSpec{}}
	input := widgetInput{spec: spec, font: loadedFont{name: "body", glyphs: newGlyphFont(noMinus{regularFace(test, 20)})}, slot: 2}
	_, err := buildWidget(input)
	want := `widget 1 (number): the font body has no glyph for "-"`
	if err == nil || err.Error() != want {
		test.Errorf("got %v, want %q", err, want)
	}
}
