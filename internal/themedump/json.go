package themedump

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/norareit/sensordeck/internal/themefile"
)

// The JSON names exist for the reader of the dump only and are no contract for another program. A name that ends in
// _guess is a guess of the meaning, and one of the form byte_<n>_unknown or bytes_<from>_<to>_unknown is a byte or a
// range whose meaning is unknown.

type hexBytes []byte

func (data hexBytes) MarshalJSON() ([]byte, error) { return json.Marshal(hex.EncodeToString(data)) }

type colourHex themefile.Colour

func (colour colourHex) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%04x", uint16(colour)))
}

// recordJSON is what a record of the dump is described as: one of the eight types below, and no other.
type recordJSON interface{ describesRecord() }

func (head) describesRecord()        {}
func (unknownJSON) describesRecord() {}

type head struct {
	Slot     int    `json:"slot"`
	Type     byte   `json:"type"`
	TypeName string `json:"type_name"`
	Index    byte   `json:"index"`
	Tag      byte   `json:"tag"`
	X        uint16 `json:"x"`
	Y        uint16 `json:"y"`
	Width    uint16 `json:"width"`
	Height   uint16 `json:"height"`
}

type tail struct {
	Raw   hexBytes `json:"raw"`
	Files []string `json:"files"`
}

type backgroundJSON struct {
	head
	Byte11          byte      `json:"byte_11_unknown"`
	FillColourGuess colourHex `json:"fill_colour_guess"`
	Offset          uint32    `json:"offset"`
	Bytes18To63     hexBytes  `json:"bytes_18_63_unknown"`
	tail
}

type startupJSON struct {
	head
	Offset             uint32   `json:"offset"`
	FrameCount         byte     `json:"frame_count"`
	DurationMsGuess    uint16   `json:"duration_ms_guess"`
	FramePeriodMsGuess uint16   `json:"frame_period_ms_guess"`
	Bytes20To63        hexBytes `json:"bytes_20_63_unknown"`
	tail
}

type imageJSON struct {
	head
	Offset             uint32   `json:"offset"`
	FrameCount         byte     `json:"frame_count"`
	Byte16             byte     `json:"byte_16_unknown"`
	FramePeriodMsGuess uint16   `json:"frame_period_ms_guess"`
	Bytes19To63        hexBytes `json:"bytes_19_63_unknown"`
	tail
}

type textJSON struct {
	head
	Offset       uint32    `json:"offset"`
	Colour       colourHex `json:"colour"`
	OpacityGuess byte      `json:"opacity_guess"`
	Bytes18To63  hexBytes  `json:"bytes_18_63_unknown"`
	tail
}

type numberJSON struct {
	head
	Alignment     byte      `json:"alignment"`
	Colour        colourHex `json:"colour"`
	ThousandsMode byte      `json:"thousands_mode"`
	Opacity       byte      `json:"opacity"`
	Offset        uint32    `json:"offset"`
	GlyphHeight   uint16    `json:"glyph_height"`
	GlyphWidths   []uint16  `json:"glyph_widths"`
	GlyphOrder    string    `json:"glyph_order"`
	Bytes46To63   hexBytes  `json:"bytes_46_63_unknown"`
	tail
}

type datetimeJSON struct {
	head
	Byte11           byte      `json:"byte_11_unknown"`
	Colour           colourHex `json:"colour"`
	OpacityGuess     byte      `json:"opacity_guess"`
	Offset           uint32    `json:"offset"`
	GlyphHeight      uint16    `json:"glyph_height"`
	DigitWidth       uint16    `json:"digit_width"`
	OtherGlyphWidths []uint16  `json:"other_glyph_widths"`
	OtherGlyphOrder  string    `json:"other_glyph_order"`
	// Format is left out when a byte of the format is not printable ASCII, and is "" for an empty format.
	Format *string `json:"format,omitempty"`
	tail
}

type barJSON struct {
	head
	Byte11              byte      `json:"byte_11_unknown"`
	EmptyColour         colourHex `json:"empty_colour"`
	FillOrFrameColour14 colourHex `json:"fill_or_frame_colour_14"`
	FillOrFrameColour16 colourHex `json:"fill_or_frame_colour_16"`
	Bytes18To63         hexBytes  `json:"bytes_18_63_unknown"`
	tail
}

// unknownJSON has no common head, which is given for a known type only.
type unknownJSON struct {
	Slot     int      `json:"slot"`
	Type     byte     `json:"type"`
	TypeName string   `json:"type_name"`
	Raw      hexBytes `json:"raw"`
	Files    []string `json:"files"`
}

type headerJSON struct {
	Byte0Unknown byte     `json:"byte_0_unknown"`
	Raw          hexBytes `json:"raw"`
}

type themeJSON struct {
	FileSize int          `json:"file_size"`
	Header   headerJSON   `json:"header"`
	Records  []recordJSON `json:"records"`
	Warnings []string     `json:"warnings"`
}

func describeHead(slot int, source themefile.Head) head {
	return head{Slot: slot, Type: source.Type, TypeName: themefile.TypeName(source.Type), Index: source.Index, Tag: source.Tag,
		X: source.X, Y: source.Y, Width: source.Width, Height: source.Height}
}

// describe gives the JSON value of a record. The files are the names of the PNG files written for it.
func describe(entry themefile.Entry, files []string) recordJSON {
	raw := entry.Record.Encode()
	closing := tail{Raw: raw[:], Files: files}
	switch typed := entry.Record.(type) {
	case themefile.Background:
		return backgroundJSON{head: describeHead(entry.Slot, typed.Head), Byte11: typed.Byte11, FillColourGuess: colourHex(typed.FillColourGuess),
			Offset: typed.Offset, Bytes18To63: typed.Bytes18To63[:], tail: closing}
	case themefile.Startup:
		return startupJSON{head: describeHead(entry.Slot, typed.Head), Offset: typed.Offset, FrameCount: typed.FrameCount,
			DurationMsGuess: typed.DurationMsGuess, FramePeriodMsGuess: typed.FramePeriodMsGuess, Bytes20To63: typed.Bytes20To63[:], tail: closing}
	case themefile.Image:
		return imageJSON{head: describeHead(entry.Slot, typed.Head), Offset: typed.Offset, FrameCount: typed.FrameCount, Byte16: typed.Byte16,
			FramePeriodMsGuess: typed.FramePeriodMsGuess, Bytes19To63: typed.Bytes19To63[:], tail: closing}
	case themefile.Text:
		return textJSON{head: describeHead(entry.Slot, typed.Head), Offset: typed.Offset, Colour: colourHex(typed.Colour),
			OpacityGuess: typed.OpacityGuess, Bytes18To63: typed.Bytes18To63[:], tail: closing}
	case themefile.Number:
		return numberJSON{head: describeHead(entry.Slot, typed.Head), Alignment: typed.Alignment, Colour: colourHex(typed.Colour),
			ThousandsMode: typed.ThousandsMode, Opacity: typed.Opacity, Offset: typed.Offset, GlyphHeight: typed.GlyphHeight,
			GlyphWidths: typed.GlyphWidths[:], GlyphOrder: themefile.NumberGlyphOrder, Bytes46To63: typed.Bytes46To63[:], tail: closing}
	case themefile.Datetime:
		return datetimeJSON{head: describeHead(entry.Slot, typed.Head), Byte11: typed.Byte11, Colour: colourHex(typed.Colour),
			OpacityGuess: typed.OpacityGuess, Offset: typed.Offset, GlyphHeight: typed.GlyphHeight, DigitWidth: typed.DigitWidth,
			OtherGlyphWidths: typed.OtherGlyphWidths[:], OtherGlyphOrder: themefile.DatetimeOtherGlyphOrder,
			Format: printableFormat(typed.FormatText()), tail: closing}
	case themefile.Bar:
		return barJSON{head: describeHead(entry.Slot, typed.Head), Byte11: typed.Byte11, EmptyColour: colourHex(typed.EmptyColour),
			FillOrFrameColour14: colourHex(typed.FillOrFrameColour14), FillOrFrameColour16: colourHex(typed.FillOrFrameColour16),
			Bytes18To63: typed.Bytes18To63[:], tail: closing}
	}
	return unknownJSON{Slot: entry.Slot, Type: raw[0], TypeName: themefile.TypeName(raw[0]), Raw: raw[:], Files: files}
}

func printableFormat(format []byte) *string {
	for _, character := range format {
		if character < 0x20 || character > 0x7e {
			return nil
		}
	}
	text := string(format)
	return &text
}

// encode writes the description with an indent of two spaces and a final newline.
func encode(description themeJSON) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(description); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
