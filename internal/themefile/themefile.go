// Package themefile reads the bytes of a theme file, the img.dat that the screen plays, into typed values. It does no
// I/O and keeps no state, so that the theme compiler can reuse its types and test its output by reading it back. The
// format is described in docs/theme-format.md.
package themefile

import "fmt"

const (
	SlotSize  = 64
	SlotCount = 64
	TableSize = SlotSize * SlotCount
)

const (
	TypeBackground byte = 0x81
	TypeImage      byte = 0x84
	TypeBar        byte = 0x8B
	TypeDatetime   byte = 0x8E
	TypeNumber     byte = 0x92
	TypeText       byte = 0x93
	TypeStartup    byte = 0x94
)

const (
	// NumberGlyphOrder is the order of the twelve glyphs of a number.
	NumberGlyphOrder = "0123456789.-"
	// DatetimeOtherGlyphOrder is the order of the eleven glyphs of a datetime that follow its ten digits.
	DatetimeOtherGlyphOrder = ".-/ :年月日时分秒"
)

// TypeName gives the name of the type byte, or "unknown".
func TypeName(recordType byte) string {
	switch recordType {
	case TypeBackground:
		return "background"
	case TypeImage:
		return "image"
	case TypeBar:
		return "bar"
	case TypeDatetime:
		return "datetime"
	case TypeNumber:
		return "number"
	case TypeText:
		return "text"
	case TypeStartup:
		return "startup"
	}
	return "unknown"
}

// Entry is the record of one used slot.
type Entry struct {
	Slot   int
	Record Record
}

// Theme is a theme file read.
type Theme struct {
	Header   [SlotSize]byte
	FileSize int
	Entries  []Entry // one for each used slot, in the order of the slots
}

// Parse reads a whole theme file. It gives the theme and the warnings, or the error of the first malformed record in
// the order of the slots. It does not panic on any input. The resources of the records are sub-slices of file.
func Parse(file []byte) (Theme, []string, error) {
	if len(file) < TableSize {
		return Theme{}, nil, fmt.Errorf("the file is %d bytes, shorter than the record table of %d bytes", len(file), TableSize)
	}
	theme := Theme{FileSize: len(file)}
	copy(theme.Header[:], file[:SlotSize])
	var warnings []string
	for number := 1; number < SlotCount; number++ {
		slot := (*[SlotSize]byte)(file[number*SlotSize : (number+1)*SlotSize])
		if *slot == [SlotSize]byte{} {
			continue
		}
		record, recordWarnings, err := decode(file, number, slot)
		if err != nil {
			return Theme{}, nil, err
		}
		warnings = append(warnings, recordWarnings...)
		theme.Entries = append(theme.Entries, Entry{Slot: number, Record: record})
	}
	return theme, warnings, nil
}

func decode(file []byte, number int, slot *[SlotSize]byte) (Record, []string, error) {
	switch slot[0] {
	case TypeBackground:
		background := decodeBackground(slot)
		if background.Offset == 0 {
			return background, nil, nil
		}
		resource, err := resourceOf(file, number, TypeBackground, background.Offset, 2*area(background.Head))
		background.Resource = resource
		return background, nil, err
	case TypeStartup:
		startup := decodeStartup(slot)
		resource, err := resourceOf(file, number, TypeStartup, startup.Offset, uint64(startup.FrameCount)*2*area(startup.Head))
		startup.Resource = resource
		return startup, nil, err
	case TypeImage:
		image := decodeImage(slot)
		resource, err := resourceOf(file, number, TypeImage, image.Offset, uint64(image.FrameCount)*3*area(image.Head))
		image.Resource = resource
		return image, nil, err
	case TypeText:
		text := decodeText(slot)
		resource, err := resourceOf(file, number, TypeText, text.Offset, area(text.Head))
		text.Resource = resource
		return text, nil, err
	case TypeNumber:
		numberRecord := decodeNumber(slot)
		var widthSum uint64
		for _, width := range numberRecord.GlyphWidths {
			widthSum += uint64(width)
		}
		resource, err := resourceOf(file, number, TypeNumber, numberRecord.Offset, uint64(numberRecord.GlyphHeight)*widthSum)
		numberRecord.Resource = resource
		return numberRecord, nil, err
	case TypeDatetime:
		datetime := decodeDatetime(slot)
		widthSum := 10 * uint64(datetime.DigitWidth)
		for _, width := range datetime.OtherGlyphWidths {
			widthSum += uint64(width)
		}
		resource, err := resourceOf(file, number, TypeDatetime, datetime.Offset, uint64(datetime.GlyphHeight)*widthSum)
		datetime.Resource = resource
		return datetime, formatWarnings(number, datetime), err
	case TypeBar:
		return decodeBar(slot), nil, nil
	}
	warning := fmt.Sprintf("slot %d: record type 0x%02x is not known, kept as raw bytes", number, slot[0])
	return decodeUnknown(slot), []string{warning}, nil
}

func area(head Head) uint64 { return uint64(head.Width) * uint64(head.Height) }

// resourceOf gives the bytes of a resource, or the error that it runs past the end of the file. The end is computed in
// 64 bits, and the file is sliced only after the check, so that no field makes it wrap or makes a buffer.
func resourceOf(file []byte, slot int, recordType byte, offset uint32, size uint64) ([]byte, error) {
	end := uint64(offset) + size
	if end > uint64(len(file)) {
		return nil, &ResourceError{Slot: slot, TypeName: TypeName(recordType), Field: "offset", Offset: offset, Size: size, FileSize: len(file)}
	}
	return file[offset:end:end], nil
}

func formatWarnings(slot int, datetime Datetime) []string {
	var warnings []string
	for _, character := range datetime.FormatText() {
		if (character >= '1' && character <= '6') || character == '.' || character == '-' || character == '/' || character == ' ' || character == ':' {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("slot %d (datetime): format character %q is not known", slot, character))
	}
	return warnings
}

// Frames cuts the resource of a startup into its frames, each 2 x width x height bytes of RGB565 pixels.
func (startup Startup) Frames() [][]byte {
	return chunks(startup.Resource, int(startup.FrameCount), 2*int(area(startup.Head)))
}

// ImageFrame is one frame of an image: an alpha plane and a colour plane of RGB565 pixels.
type ImageFrame struct{ Alpha, Colour []byte }

// Frames cuts the resource of an image into its frames.
func (image Image) Frames() []ImageFrame {
	plane := int(area(image.Head))
	var frames []ImageFrame
	for _, frame := range chunks(image.Resource, int(image.FrameCount), 3*plane) {
		frames = append(frames, ImageFrame{Alpha: frame[:plane], Colour: frame[plane:]})
	}
	return frames
}

// Glyphs cuts the resource of a number into the twelve masks of its set, each GlyphHeight x its width bytes.
func (number Number) Glyphs() [][]byte {
	return glyphMasks(number.Resource, int(number.GlyphHeight), number.GlyphWidths[:])
}

// GlyphWidths gives the widths of the twenty-one glyphs of a datetime: ten digits, then the eleven others.
func (datetime Datetime) GlyphWidths() [21]uint16 {
	var widths [21]uint16
	for digit := range 10 {
		widths[digit] = datetime.DigitWidth
	}
	copy(widths[10:], datetime.OtherGlyphWidths[:])
	return widths
}

// Glyphs cuts the resource of a datetime into the twenty-one masks of its set.
func (datetime Datetime) Glyphs() [][]byte {
	widths := datetime.GlyphWidths()
	return glyphMasks(datetime.Resource, int(datetime.GlyphHeight), widths[:])
}

func chunks(data []byte, count, size int) [][]byte {
	parts := make([][]byte, count)
	for position := range parts {
		parts[position] = data[position*size : (position+1)*size]
	}
	return parts
}

func glyphMasks(data []byte, height int, widths []uint16) [][]byte {
	masks := make([][]byte, len(widths))
	start := 0
	for glyph, width := range widths {
		end := start + height*int(width)
		masks[glyph] = data[start:end]
		start = end
	}
	return masks
}
