package themefile_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themefile"
)

// distinctFrom fills the bytes from the position to the end of the slot with values that no other byte of the slot
// holds, which a test then overwrites where the layout fixes a value.
func distinctFrom(slot *[64]byte, from int) {
	for position := from; position < 64; position++ {
		slot[position] = byte(0xa0 + position)
	}
}

func u16(slot [64]byte, at int) uint16 { return binary.BigEndian.Uint16(slot[at:]) }
func u32(slot [64]byte, at int) uint32 { return binary.BigEndian.Uint32(slot[at:]) }

func colourAt(slot [64]byte, at int) themefile.Colour { return themefile.Colour(u16(slot, at)) }

// parseOne builds a file with the slot as slot 1 and the resource after the table, and gives the record read.
func parseOne(test *testing.T, slot [64]byte, resource []byte) (themefile.Record, []byte) {
	test.Helper()
	builder := faketheme.New()
	builder.Slot(1, slot)
	builder.Resource(resource)
	file := builder.Bytes()
	theme, _, err := themefile.Parse(file)
	if err != nil {
		test.Fatal(err)
	}
	if len(theme.Entries) != 1 || theme.Entries[0].Slot != 1 {
		test.Fatalf("entries %v", theme.Entries)
	}
	record := theme.Entries[0].Record
	if encoded := record.Encode(); encoded != slot {
		test.Errorf("Encode gave % x, want % x", encoded, slot)
	}
	return record, file
}

func resourceOf(count int, first byte) []byte {
	data := make([]byte, count)
	for position := range data {
		data[position] = first + byte(position)
	}
	return data
}

func expectHead(test *testing.T, got themefile.Head, slot [64]byte) {
	test.Helper()
	want := themefile.Head{Type: slot[0], Index: slot[1], Tag: slot[2], X: u16(slot, 3), Y: u16(slot, 5), Width: u16(slot, 7), Height: u16(slot, 9)}
	if got != want {
		test.Errorf("head %+v, want %+v", got, want)
	}
}

func TestTheBackgroundIsReadFromItsBytes(test *testing.T) {
	slot := faketheme.Head(0x81, 0x31, 0x32, 0x0102, 0x0304, 2, 2)
	distinctFrom(&slot, 11)
	faketheme.Put32(&slot, 14, 0x1000)
	resource := resourceOf(8, 1)
	record, _ := parseOne(test, slot, resource)
	got := record.(themefile.Background)
	expectHead(test, got.Head, slot)
	if got.Byte11 != slot[11] || got.FillColourGuess != colourAt(slot, 12) || got.Offset != 0x1000 || !bytes.Equal(got.Bytes18To63[:], slot[18:]) {
		test.Errorf("fields %+v", got)
	}
	if !bytes.Equal(got.Resource, resource) {
		test.Errorf("resource % x", got.Resource)
	}
}

func TestABackgroundWithOffsetZeroHasNoResource(test *testing.T) {
	slot := faketheme.Head(0x81, 1, 0, 0, 0, 480, 320)
	record, _ := parseOne(test, slot, nil)
	if got := record.(themefile.Background); got.Resource != nil || got.Offset != 0 {
		test.Errorf("resource %v offset %d", got.Resource, got.Offset)
	}
}

func TestTheStartupIsReadFromItsBytes(test *testing.T) {
	slot := faketheme.Head(0x94, 0x31, 0x32, 0x0102, 0x0304, 2, 3)
	distinctFrom(&slot, 11)
	faketheme.Put32(&slot, 11, 0x1000)
	slot[15] = 3
	resource := resourceOf(3*2*2*3, 1)
	record, _ := parseOne(test, slot, resource)
	got := record.(themefile.Startup)
	expectHead(test, got.Head, slot)
	if got.Offset != 0x1000 || got.FrameCount != 3 || got.DurationMsGuess != u16(slot, 16) || got.FramePeriodMsGuess != u16(slot, 18) || !bytes.Equal(got.Bytes20To63[:], slot[20:]) {
		test.Errorf("fields %+v", got)
	}
	if !bytes.Equal(got.Resource, resource) {
		test.Errorf("resource % x", got.Resource)
	}
}

func TestTheImageIsReadFromItsBytes(test *testing.T) {
	slot := faketheme.Head(0x84, 0x31, 0x32, 0x0102, 0x0304, 3, 2)
	distinctFrom(&slot, 11)
	faketheme.Put32(&slot, 11, 0x1000)
	slot[15] = 2
	resource := resourceOf(2*3*3*2, 1)
	record, _ := parseOne(test, slot, resource)
	got := record.(themefile.Image)
	expectHead(test, got.Head, slot)
	if got.Offset != 0x1000 || got.FrameCount != 2 || got.Byte16 != slot[16] || got.FramePeriodMsGuess != u16(slot, 17) || !bytes.Equal(got.Bytes19To63[:], slot[19:]) {
		test.Errorf("fields %+v", got)
	}
	if !bytes.Equal(got.Resource, resource) {
		test.Errorf("resource % x", got.Resource)
	}
}

func TestTwoImagesWithTheSameOffsetShareTheirBytes(test *testing.T) {
	builder := faketheme.New()
	resource := resourceOf(3*2*2, 1)
	offset := builder.Resource(resource)
	for _, number := range []int{1, 2} {
		slot := faketheme.Head(0x84, byte(number), 0, 0, 0, 2, 2)
		faketheme.Put32(&slot, 11, offset)
		slot[15] = 1
		builder.Slot(number, slot)
	}
	theme, _, err := themefile.Parse(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	first, second := theme.Entries[0].Record.(themefile.Image), theme.Entries[1].Record.(themefile.Image)
	if !bytes.Equal(first.Resource, resource) || !bytes.Equal(second.Resource, resource) {
		test.Error("a shared resource is not read for both records")
	}
}

func TestTheTextIsReadFromItsBytes(test *testing.T) {
	slot := faketheme.Head(0x93, 0x31, 0x32, 0x0102, 0x0304, 5, 3)
	distinctFrom(&slot, 11)
	faketheme.Put32(&slot, 11, 0x1000)
	resource := resourceOf(15, 1)
	record, _ := parseOne(test, slot, resource)
	got := record.(themefile.Text)
	expectHead(test, got.Head, slot)
	if got.Offset != 0x1000 || got.Colour != colourAt(slot, 15) || got.OpacityGuess != slot[17] || !bytes.Equal(got.Bytes18To63[:], slot[18:]) {
		test.Errorf("fields %+v", got)
	}
	if !bytes.Equal(got.Resource, resource) {
		test.Errorf("resource % x", got.Resource)
	}
}

func TestTheNumberIsReadFromItsBytes(test *testing.T) {
	slot := faketheme.Head(0x92, 0x31, 0x32, 0x0102, 0x0304, 20, 3)
	distinctFrom(&slot, 11)
	faketheme.Put32(&slot, 16, 0x1000)
	faketheme.Put16(&slot, 20, 3)
	widthSum := 0
	for glyph := range 12 {
		faketheme.Put16(&slot, 22+2*glyph, uint16(glyph+1))
		widthSum += glyph + 1
	}
	resource := resourceOf(3*widthSum, 1)
	record, _ := parseOne(test, slot, resource)
	got := record.(themefile.Number)
	expectHead(test, got.Head, slot)
	if got.Alignment != slot[11] || got.Colour != colourAt(slot, 12) || got.ThousandsMode != slot[14] || got.Opacity != slot[15] ||
		got.Offset != 0x1000 || got.GlyphHeight != 3 || !bytes.Equal(got.Bytes46To63[:], slot[46:]) {
		test.Errorf("fields %+v", got)
	}
	for glyph, width := range got.GlyphWidths {
		if width != u16(slot, 22+2*glyph) {
			test.Errorf("glyph %d has width %d", glyph, width)
		}
	}
	if !bytes.Equal(got.Resource, resource) {
		test.Errorf("resource % x", got.Resource)
	}
}

func TestTheDatetimeIsReadFromItsBytes(test *testing.T) {
	slot := faketheme.Head(0x8e, 0x31, 0x32, 0x0102, 0x0304, 60, 3)
	distinctFrom(&slot, 11)
	faketheme.Put32(&slot, 15, 0x1000)
	faketheme.Put16(&slot, 19, 3)
	faketheme.Put16(&slot, 21, 2)
	widthSum := 20
	for glyph := range 11 {
		faketheme.Put16(&slot, 23+2*glyph, uint16(glyph+1))
		widthSum += glyph + 1
	}
	resource := resourceOf(3*widthSum, 1)
	record, _ := parseOne(test, slot, resource)
	got := record.(themefile.Datetime)
	expectHead(test, got.Head, slot)
	if got.Byte11 != slot[11] || got.Colour != colourAt(slot, 12) || got.OpacityGuess != slot[14] || got.Offset != 0x1000 ||
		got.GlyphHeight != 3 || got.DigitWidth != 2 || !bytes.Equal(got.Format[:], slot[45:]) {
		test.Errorf("fields %+v", got)
	}
	for glyph, width := range got.OtherGlyphWidths {
		if width != u16(slot, 23+2*glyph) {
			test.Errorf("other glyph %d has width %d", glyph, width)
		}
	}
	if !bytes.Equal(got.Resource, resource) {
		test.Errorf("resource % x", got.Resource)
	}
}

func TestTheBarIsReadFromItsBytes(test *testing.T) {
	slot := faketheme.Head(0x8b, 0x31, 0x32, 0x0102, 0x0304, 5, 6)
	distinctFrom(&slot, 11)
	record, _ := parseOne(test, slot, nil)
	got := record.(themefile.Bar)
	expectHead(test, got.Head, slot)
	if got.Byte11 != slot[11] || got.EmptyColour != colourAt(slot, 12) || got.FillOrFrameColour14 != colourAt(slot, 14) ||
		got.FillOrFrameColour16 != colourAt(slot, 16) || !bytes.Equal(got.Bytes18To63[:], slot[18:]) {
		test.Errorf("fields %+v", got)
	}
	if encoded := got.Encode(); encoded != slot {
		test.Errorf("encoded % x", encoded)
	}
}

func TestAnUnknownRecordKeepsItsBytes(test *testing.T) {
	var slot [64]byte
	distinctFrom(&slot, 0)
	slot[0] = 0x77
	record, _ := parseOne(test, slot, nil)
	got := record.(themefile.Unknown)
	if got.Type != 0x77 || !bytes.Equal(got.Bytes1To63[:], slot[1:]) {
		test.Errorf("fields %+v", got)
	}
}
