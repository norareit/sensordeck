package themefile

import "encoding/binary"

// Colour is an RGB565 colour: red in the top 5 bits, green in the middle 6, blue in the low 5.
type Colour uint16

// Head is the common head of a record, bytes 0 to 10 of its slot.
type Head struct {
	Type, Index, Tag    byte
	X, Y, Width, Height uint16
}

// Record is one of the eight record types below, and no other.
type Record interface {
	// Encode gives the 64 bytes of the slot from the typed value alone.
	Encode() [SlotSize]byte
	sealed()
}

// Each type holds the fields of its table by name, in the order of the bytes. A guess ends in Guess, an unknown byte is
// Byte<n> and an unknown range is Bytes<from>To<to>, so that every byte of the slot is in exactly one field. Resource is
// a sub-slice of the file and is no part of the slot.

type Background struct {
	Head
	Byte11          byte
	FillColourGuess Colour
	Offset          uint32
	Bytes18To63     [46]byte
	Resource        []byte
}

type Startup struct {
	Head
	Offset             uint32
	FrameCount         byte
	DurationMsGuess    uint16
	FramePeriodMsGuess uint16
	Bytes20To63        [44]byte
	Resource           []byte
}

type Image struct {
	Head
	Offset             uint32
	FrameCount         byte
	Byte16             byte
	FramePeriodMsGuess uint16
	Bytes19To63        [45]byte
	Resource           []byte
}

type Text struct {
	Head
	Offset       uint32
	Colour       Colour
	OpacityGuess byte
	Bytes18To63  [46]byte
	Resource     []byte
}

type Number struct {
	Head
	AlignmentGuess byte
	Colour         Colour
	DecimalsGuess  byte
	OpacityGuess   byte
	Offset         uint32
	GlyphHeight    uint16
	GlyphWidths    [12]uint16
	Bytes46To63    [18]byte
	Resource       []byte
}

type Datetime struct {
	Head
	Byte11           byte
	Colour           Colour
	OpacityGuess     byte
	Offset           uint32
	GlyphHeight      uint16
	DigitWidth       uint16
	OtherGlyphWidths [11]uint16
	// Format keeps all 19 bytes, padding included, so that Encode gives them back.
	Format   [19]byte
	Resource []byte
}

type Bar struct {
	Head
	Bytes11To63 [53]byte
}

type Unknown struct {
	Type       byte
	Bytes1To63 [63]byte
}

func (Background) sealed() {}
func (Startup) sealed()    {}
func (Image) sealed()      {}
func (Text) sealed()       {}
func (Number) sealed()     {}
func (Datetime) sealed()   {}
func (Bar) sealed()        {}
func (Unknown) sealed()    {}

// FormatText gives the format of a datetime: the bytes from byte 45 up to the first zero byte or to byte 63.
func (datetime Datetime) FormatText() []byte {
	for position, value := range datetime.Format {
		if value == 0 {
			return datetime.Format[:position]
		}
	}
	return datetime.Format[:]
}

// slotReader reads the fields of one slot. It reads only inside the array, so no input makes an index run out of range.
type slotReader struct{ slot *[SlotSize]byte }

func (reader slotReader) u16(at int) uint16 { return binary.BigEndian.Uint16(reader.slot[at:]) }
func (reader slotReader) u32(at int) uint32 { return binary.BigEndian.Uint32(reader.slot[at:]) }

func (reader slotReader) head() Head {
	return Head{Type: reader.slot[0], Index: reader.slot[1], Tag: reader.slot[2],
		X: reader.u16(3), Y: reader.u16(5), Width: reader.u16(7), Height: reader.u16(9)}
}

// slotWriter is the inverse of slotReader.
type slotWriter struct{ slot [SlotSize]byte }

func (writer *slotWriter) u16(at int, value uint16) {
	binary.BigEndian.PutUint16(writer.slot[at:], value)
}
func (writer *slotWriter) u32(at int, value uint32) {
	binary.BigEndian.PutUint32(writer.slot[at:], value)
}

func (writer *slotWriter) head(head Head) {
	writer.slot[0], writer.slot[1], writer.slot[2] = head.Type, head.Index, head.Tag
	writer.u16(3, head.X)
	writer.u16(5, head.Y)
	writer.u16(7, head.Width)
	writer.u16(9, head.Height)
}

func (background Background) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.head(background.Head)
	writer.slot[11] = background.Byte11
	writer.u16(12, uint16(background.FillColourGuess))
	writer.u32(14, background.Offset)
	copy(writer.slot[18:], background.Bytes18To63[:])
	return writer.slot
}

func (startup Startup) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.head(startup.Head)
	writer.u32(11, startup.Offset)
	writer.slot[15] = startup.FrameCount
	writer.u16(16, startup.DurationMsGuess)
	writer.u16(18, startup.FramePeriodMsGuess)
	copy(writer.slot[20:], startup.Bytes20To63[:])
	return writer.slot
}

func (image Image) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.head(image.Head)
	writer.u32(11, image.Offset)
	writer.slot[15] = image.FrameCount
	writer.slot[16] = image.Byte16
	writer.u16(17, image.FramePeriodMsGuess)
	copy(writer.slot[19:], image.Bytes19To63[:])
	return writer.slot
}

func (text Text) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.head(text.Head)
	writer.u32(11, text.Offset)
	writer.u16(15, uint16(text.Colour))
	writer.slot[17] = text.OpacityGuess
	copy(writer.slot[18:], text.Bytes18To63[:])
	return writer.slot
}

func (number Number) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.head(number.Head)
	writer.slot[11] = number.AlignmentGuess
	writer.u16(12, uint16(number.Colour))
	writer.slot[14] = number.DecimalsGuess
	writer.slot[15] = number.OpacityGuess
	writer.u32(16, number.Offset)
	writer.u16(20, number.GlyphHeight)
	for glyph, width := range number.GlyphWidths {
		writer.u16(22+2*glyph, width)
	}
	copy(writer.slot[46:], number.Bytes46To63[:])
	return writer.slot
}

func (datetime Datetime) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.head(datetime.Head)
	writer.slot[11] = datetime.Byte11
	writer.u16(12, uint16(datetime.Colour))
	writer.slot[14] = datetime.OpacityGuess
	writer.u32(15, datetime.Offset)
	writer.u16(19, datetime.GlyphHeight)
	writer.u16(21, datetime.DigitWidth)
	for glyph, width := range datetime.OtherGlyphWidths {
		writer.u16(23+2*glyph, width)
	}
	copy(writer.slot[45:], datetime.Format[:])
	return writer.slot
}

func (bar Bar) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.head(bar.Head)
	copy(writer.slot[11:], bar.Bytes11To63[:])
	return writer.slot
}

func (unknown Unknown) Encode() [SlotSize]byte {
	var writer slotWriter
	writer.slot[0] = unknown.Type
	copy(writer.slot[1:], unknown.Bytes1To63[:])
	return writer.slot
}

func decodeBackground(slot *[SlotSize]byte) Background {
	reader := slotReader{slot}
	background := Background{Head: reader.head(), Byte11: slot[11], FillColourGuess: Colour(reader.u16(12)), Offset: reader.u32(14)}
	copy(background.Bytes18To63[:], slot[18:])
	return background
}

func decodeStartup(slot *[SlotSize]byte) Startup {
	reader := slotReader{slot}
	startup := Startup{Head: reader.head(), Offset: reader.u32(11), FrameCount: slot[15], DurationMsGuess: reader.u16(16), FramePeriodMsGuess: reader.u16(18)}
	copy(startup.Bytes20To63[:], slot[20:])
	return startup
}

func decodeImage(slot *[SlotSize]byte) Image {
	reader := slotReader{slot}
	image := Image{Head: reader.head(), Offset: reader.u32(11), FrameCount: slot[15], Byte16: slot[16], FramePeriodMsGuess: reader.u16(17)}
	copy(image.Bytes19To63[:], slot[19:])
	return image
}

func decodeText(slot *[SlotSize]byte) Text {
	reader := slotReader{slot}
	text := Text{Head: reader.head(), Offset: reader.u32(11), Colour: Colour(reader.u16(15)), OpacityGuess: slot[17]}
	copy(text.Bytes18To63[:], slot[18:])
	return text
}

func decodeNumber(slot *[SlotSize]byte) Number {
	reader := slotReader{slot}
	number := Number{Head: reader.head(), AlignmentGuess: slot[11], Colour: Colour(reader.u16(12)), DecimalsGuess: slot[14],
		OpacityGuess: slot[15], Offset: reader.u32(16), GlyphHeight: reader.u16(20)}
	for glyph := range number.GlyphWidths {
		number.GlyphWidths[glyph] = reader.u16(22 + 2*glyph)
	}
	copy(number.Bytes46To63[:], slot[46:])
	return number
}

func decodeDatetime(slot *[SlotSize]byte) Datetime {
	reader := slotReader{slot}
	datetime := Datetime{Head: reader.head(), Byte11: slot[11], Colour: Colour(reader.u16(12)), OpacityGuess: slot[14],
		Offset: reader.u32(15), GlyphHeight: reader.u16(19), DigitWidth: reader.u16(21)}
	for glyph := range datetime.OtherGlyphWidths {
		datetime.OtherGlyphWidths[glyph] = reader.u16(23 + 2*glyph)
	}
	copy(datetime.Format[:], slot[45:])
	return datetime
}

func decodeBar(slot *[SlotSize]byte) Bar {
	bar := Bar{Head: slotReader{slot}.head()}
	copy(bar.Bytes11To63[:], slot[11:])
	return bar
}

func decodeUnknown(slot *[SlotSize]byte) Unknown {
	unknown := Unknown{Type: slot[0]}
	copy(unknown.Bytes1To63[:], slot[1:])
	return unknown
}
