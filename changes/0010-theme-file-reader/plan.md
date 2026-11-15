# A reader of the theme file and the document of its format

## Approach

Five parts, each with one job. The first four are pure and do no I/O, and only the command touches the file system:

* `internal/faketheme`: builds synthetic theme files in code from the layout of the spec, for the tests of every
  other part. It does not use the reader, so that a test of the reader does not check the reader against itself (R22)
* `internal/themefile`: the reader. It turns the bytes of a file into a `Theme` of typed records, warnings or an error,
  and holds the knowledge of the layout: the offsets of the fields, the sizes of the resources and how a resource is
  cut into frames and glyphs (R1 to R10)
* `internal/themeimage`: turns typed records into images: the images of each resource and the preview (R11, R12)
* `internal/themedump`: turns a theme into the named contents of the output files: `theme.json`, the PNG files of
  the records and `preview.png`, with no I/O (R13 to R15, R19)
* `internal/cli`: the command `theme dump`, which reads the file, asks `themedump` for the contents, and only then
  creates the directory and writes them (R16 to R18)

Then the end-to-end tests (R23) and the documents (R20, R21). No dependency is added (D10). None of the items of
"Beyond the request" is built, as the record decided.

The dependencies go one way: `themeimage` imports `themefile`, `themedump` imports both, `cli` imports `themedump`
and `themefile`. None of the four new packages imports a package of the device, the transport, the sensors or the
command line, or `os`, which a test of the layout checks (R1).

### The fake theme

`internal/faketheme/faketheme.go`, with no build constraint since it is pure:

```go
// Builder makes a theme file: the record table of 4096 bytes and the resources appended after it.
type Builder struct{ table [4096]byte; resources []byte }

func New() *Builder
func (builder *Builder) Header(header [64]byte)
func (builder *Builder) Slot(number int, slot [64]byte)
// Resource appends the bytes after what was appended before and gives their absolute offset.
func (builder *Builder) Resource(data []byte) uint32
func (builder *Builder) Bytes() []byte // a new slice: the table followed by the resources

// Head gives a slot with the common head of the spec written in bytes 0 to 10, big-endian, and zeros after it.
func Head(recordType, index, tag byte, x, y, width, height uint16) [64]byte
func Put16(slot *[64]byte, at int, value uint16) // big-endian
func Put32(slot *[64]byte, at int, value uint32) // big-endian

// Sample is the small valid file of R22, with one record of each of the seven types and one of an unknown type.
func Sample() []byte
```

`Sample` holds, with header byte 0 `0x96`:

| Slot | Record | Box (x, y, width x height) | Fields and resource |
|---|---|---|---|
| 1 | background `0x81` | 0, 0, 4 x 2 | byte 11 `1`, fill colour `f79e`, a resource of 8 RGB565 pixels, bytes 18 to 21 `01 00 00 01` |
| 2 | startup `0x94` | 0, 0, 2 x 2 | 2 frames, duration 1500, period 150 |
| 3 | image `0x84` | 10, 10, 3 x 2 | 2 frames, byte 16 `1`, period 200 |
| 4 | text `0x93` | 20, 20, 5 x 3 | colour `ffff`, opacity `ff` |
| 5 | number `0x92` | 30, 30, 20 x 3 | tag 1, alignment `1`, colour `07e0`, decimals 0, opacity `ff`, glyph height 3, widths 2 for the ten digits, 1 for `.` and 2 for `-` |
| 6 | datetime `0x8E` | 40, 40, 60 x 3 | tag 21, byte 11 `1`, colour `001f`, opacity `ff`, glyph height 3, digit width 2, the eleven other widths 1 to 11, format `1-2-3 4:5:6` |
| 7 | bar `0x8B` | 50, 50, 6 x 4 | bytes 11 to 28 as the bar bytes of the spec |
| 9 | unknown `0x77` | | bytes 1 to 63 the values 1 to 63 |

Slot 8 is empty, so that the sample also has a gap in the used slots. Every pixel and mask byte of the resources is a
distinct, documented value, given by a doc comment on `Sample`, so that a test can check a chosen pixel by its value.
The startup's frames and the image's frames are distinct resources; the sample has no shared resource, which the tests
of the reader build on their own.

### The reader

`internal/themefile/themefile.go`, `records.go` and `errors.go`:

```go
const (SlotSize = 64; SlotCount = 64; TableSize = SlotSize * SlotCount)

const (
	TypeBackground byte = 0x81; TypeImage byte = 0x84; TypeBar byte = 0x8B; TypeDatetime byte = 0x8E
	TypeNumber byte = 0x92; TypeText byte = 0x93; TypeStartup byte = 0x94
)

// TypeName gives the name of the context of the spec, or "unknown".
func TypeName(recordType byte) string

type Colour uint16 // RGB565

type Head struct{ Type, Index, Tag byte; X, Y, Width, Height uint16 }

// Record is one of the eight record types below, and no other.
type Record interface {
	Encode() [SlotSize]byte // the 64 bytes of the slot, from the typed value alone (R3)
	sealed()
}

type Entry struct{ Slot int; Record Record }

type Theme struct {
	Header   [SlotSize]byte
	FileSize int
	Entries  []Entry // one for each used slot, in the order of the slots
}

func Parse(file []byte) (Theme, []string, error)
```

The eight record types hold their fields by name, each of the width the spec gives, in the order of the bytes. A
known field has a plain name, a guess ends in `Guess`, an unknown byte is `Byte<n>` and an unknown range
`Bytes<from>To<to>`, a fixed-size array, so that every byte of the slot is in exactly one field (R3):

* `Background{Head; Byte11 byte; FillColourGuess Colour; Offset uint32; Bytes18To63 [46]byte; Resource []byte}`
* `Startup{Head; Offset uint32; FrameCount byte; DurationMsGuess, FramePeriodMsGuess uint16; Bytes20To63 [44]byte; Resource []byte}`
* `Image{Head; Offset uint32; FrameCount byte; Byte16 byte; FramePeriodMsGuess uint16; Bytes19To63 [45]byte; Resource []byte}`
* `Text{Head; Offset uint32; Colour Colour; OpacityGuess byte; Bytes18To63 [46]byte; Resource []byte}`
* `Number{Head; AlignmentGuess byte; Colour Colour; DecimalsGuess, OpacityGuess byte; Offset uint32; GlyphHeight uint16; GlyphWidths [12]uint16; Bytes46To63 [18]byte; Resource []byte}`
* `Datetime{Head; Byte11 byte; Colour Colour; OpacityGuess byte; Offset uint32; GlyphHeight, DigitWidth uint16; OtherGlyphWidths [11]uint16; Format [19]byte; Resource []byte}`
* `Bar{Head; Bytes11To63 [53]byte}`
* `Unknown{Type byte; Bytes1To63 [63]byte}`

`Datetime.Format` keeps all 19 bytes, padding included, so that `Encode` gives them back. `FormatText() []byte` gives
the bytes from byte 45 up to the first zero or to byte 63 (R9).

`Resource` is a sub-slice of the input, `file[offset:end]`, so that two records with the same offset give the same
bytes and no buffer of the size of a resource is ever made (R4, R7, D11). It is nil for a background with offset 0.
`Encode` writes every field but `Resource`.

The layout of a resource is known in this package only, and the renderer asks for it:

* `resourceSize` of each type, in `uint64`, as the spec defines the size of a resource
* `(Startup) Frames() [][]byte`, each `2 x width x height` bytes
* `(Image) Frames() []ImageFrame` with `ImageFrame{Alpha, Colour []byte}`, the alpha plane and the colour plane
* `(Number) Glyphs() [][]byte`, the twelve masks in the order of the set, each `GlyphHeight x GlyphWidths[k]` bytes
* `(Datetime) GlyphWidths() [21]uint16`, ten times `DigitWidth` followed by `OtherGlyphWidths`, and
  `(Datetime) Glyphs() [][]byte`, the twenty-one masks in that order
* `NumberGlyphOrder = "0123456789.-"` and `DatetimeOtherGlyphOrder = ".-/ :年月日时分秒"`, for the renderer and the JSON

These are called only on a record that `Parse` gave, whose resource has the size its fields say.

`Parse`:

1. a file shorter than `TableSize` gives `fmt.Errorf("the file is %d bytes, shorter than the record table of 4096 bytes", len(file))` (R6)
2. the header is slot 0. For slots 1 to 63 in order, a slot of all zeros is skipped; otherwise its type byte chooses
   the decoder, and a type byte that is none of the seven gives an `Unknown` and the warning of R5,
   `fmt.Sprintf("slot %d: record type 0x%02x is not known, kept as raw bytes", slot, type)`
3. a decoder of a type with a resource computes the size in `uint64` and the end as `uint64(offset) + size`, and when
   the end is beyond `len(file)` returns a `*ResourceError` (R7), on which `Parse` stops, which makes the first
   malformed record in the order of the slots the error (R8). A background with offset 0 has no resource and no check
4. a datetime gives the warning of R9, `fmt.Sprintf("slot %d (datetime): format character %q is not known", slot, character)`
   with `character` a `byte`, for each byte of `FormatText()` that is not `1` to `6`, `.`, `-`, `/`, space or `:`
5. nothing else is checked (R10)

```go
type ResourceError struct {
	Slot     int
	TypeName string
	Field    string // the JSON name of the field of the offset, which is "offset" for every type
	Offset   uint32
	Size     uint64
	FileSize int
}
// Error: "slot <n> (<type name>): <field> 0x<offset> with <size> bytes runs past the end of the file of <file size> bytes"
```

The offset is written as `%x`, lowercase with no padding. All reads of the slot are of the 64-byte array of the slot,
and the slices of the file are made only after the check, so that no input can make an index out of range (R8).

### The images

`internal/themeimage/themeimage.go` and `preview.go`:

```go
// Expand gives the 8-bit channels of an RGB565 colour as the spec defines them, opaque.
func Expand(colour themefile.Colour) color.NRGBA

// Resources gives the images of R11 of a record: none for a bar, an unknown record and a background with offset 0.
func Resources(record themefile.Record) []*image.NRGBA

// Preview gives the 480 x 320 image of R12.
func Preview(theme themefile.Theme) *image.RGBA

// SampleTime is the date and time of the preview, 2026-11-15 09:41:07, and SampleValue the value of every number.
var SampleTime = time.Date(2026, 11, 15, 9, 41, 7, 0, time.UTC)
const SampleValue = "42"
```

* an RGB565 pixel of a resource is `uint16(low) | uint16(high)<<8`, read as two bytes, low byte first
* `opaque(pixels, width, height)` gives the image of a background and a startup frame, and `masked(alpha, pixels,
  width, height)` that of an image frame; `mask(alpha, width, height, colour)` gives a text and a glyph, each pixel the
  expanded colour with the alpha of the mask, no opacity guess applied
* a strip of R11 is the glyphs of `Glyphs()` side by side, as wide as the sum of their widths and as high as the glyph
  height
* the preview starts as `image.RGBA` filled with opaque black, and each entry is drawn with `draw.Draw(..., draw.Over)`,
  which clips to the bounds of the preview and nothing else, so that glyphs are clipped to the screen and not to their
  box. A background with offset 0 is `draw.Draw` of a `image.Uniform` of its expanded colour over its box. A bar is
  the four one-pixel lines of the inside edge of its box in `#ff00ff`, so a box of width or height 0 draws nothing and
  one of width or height 1 or 2 is filled. A startup and an unknown record are not drawn
* a number draws the glyphs of `SampleValue` and a datetime the glyphs of its `FormatText()` with `SampleTime`, both
  through one function that places a list of glyph masks with their widths from the box's x, each the next at the
  right edge of the one before, tops at the box's y. In a format, `1` is the year as `%04d`, `2` to `6` the month,
  day, hour, minute and second as `%02d`, each digit drawn as its digit glyph, a glyph character as its glyph of the
  other glyphs, and any other byte as nothing with no advance

### The dump

`internal/themedump/themedump.go` and `json.go`:

```go
type File struct{ Name string; Content []byte }

// Dump gives the files of the dump in this order: theme.json, the PNG files of the records in the order of the
// entries and of their images, and preview.png; and the warnings of the reader. An error is that of the reader.
func Dump(file []byte) ([]File, []string, error)
```

* the PNG files of an entry are named as R14 says, from `themeimage.Resources`, with `slot%02d-<type name>.png`,
  `slot%02d-<type name>-frame%d.png` counting from 1 and `slot%02d-<type name>-glyphs.png`. An image whose bounds are
  empty is left out and gets no name
* each image is encoded with `png.Encode` into a `bytes.Buffer`, which is deterministic (R19)
* the JSON is written with an `encoding/json` `Encoder` with `SetIndent("", "  ")` and `SetEscapeHTML(false)`, whose
  `Encode` ends it with a newline (R13). Its records are per-type structs whose field order is the order of R13:

```go
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
	Files []string `json:"files"` // never nil, so an empty list is []
}
type numberJSON struct {
	head
	AlignmentGuess byte      `json:"alignment_guess"`
	Colour         colourHex `json:"colour"`
	DecimalsGuess  byte      `json:"decimals_guess"`
	OpacityGuess   byte      `json:"opacity_guess"`
	Offset         uint32    `json:"offset"`
	GlyphHeight    uint16    `json:"glyph_height"`
	GlyphWidths    []uint16  `json:"glyph_widths"`
	GlyphOrder     string    `json:"glyph_order"`
	Bytes46To63    hexBytes  `json:"bytes_46_63_unknown"`
	tail
}
```

  `encoding/json` puts the promoted fields of an embedded struct at its place, so `head` comes first and `tail` last.
  The other types follow the same shape with the fields of their table in the order of the bytes:
  background `byte_11_unknown`, `fill_colour_guess`, `offset`, `bytes_18_63_unknown`; startup `offset`,
  `frame_count`, `duration_ms_guess`, `frame_period_ms_guess`, `bytes_20_63_unknown`; image `offset`, `frame_count`,
  `byte_16_unknown`, `frame_period_ms_guess`, `bytes_19_63_unknown`; text `offset`, `colour`, `opacity_guess`,
  `bytes_18_63_unknown`; datetime `byte_11_unknown`, `colour`, `opacity_guess`, `offset`, `glyph_height`,
  `digit_width`, `other_glyph_widths`, `other_glyph_order`, `format`; bar `bytes_11_63_unknown`. An unknown record has
  only `slot`, `type`, `type_name` `"unknown"`, `raw` and `files`, as R13 gives the common head for a known type only
* `format` is a `*string` with `omitempty`, nil when a byte of `FormatText()` is outside `0x20` to `0x7e`, so that an
  empty format is written as `""` and a non-printable one is left out
* `hexBytes []byte` and `colourHex themefile.Colour` have `MarshalJSON`, giving lowercase hexadecimal with no
  separators and the four digits `%04x`
* `raw` is `record.Encode()`, which the tests of the reader prove equal to the slot's bytes
* the top level is `{file_size, header{byte_0_unknown, raw}, records, warnings}`, with `records` and `warnings` never
  nil

### The command

`internal/cli/theme.go`: `session.themeCommand()` gives `theme`, with the one subcommand `dump <file.dat> <out-dir>`
and `cobra.ExactArgs(2)`, added to the root after `run`:

1. `readTheme(path)`, which gives `cannot read the theme file <path>: <cause>` as it does for `upload` (R17)
2. `themedump.Dump`; its error is wrapped as `fmt.Errorf("cannot read the theme file %s: %w", path, err)` (R17)
3. only then `os.MkdirAll(outDir, 0o755)`, its failure as `cannot create the output directory <dir>: <cause>`, and
   for each file `os.WriteFile(filepath.Join(outDir, name), content, 0o644)`, its failure as `cannot write the file
   <path>: <cause>`, each with the inner error of a `*fs.PathError` so that the path is named once, as
   `readNamedFile` does. `os.WriteFile` replaces a file of the same name and leaves other files (R13)
4. each warning as `warning: <warning>` on standard error, then `wrote <count> files to <out-dir>` on standard output,
   with `<out-dir>` as given (R16)

The command never calls `session.open`, `listScreens` or `newSources`, so it opens no device and reads no sensor, with
`--device` or without it (R18). The cobra error path of `run` in `main.go` already writes `sensordeck: ` and exits 1.

## Files

* `internal/faketheme/faketheme.go`, `internal/faketheme/faketheme_test.go`: new
* `internal/themefile/themefile.go`, `records.go`, `errors.go`, `themefile_test.go`, `records_test.go`: new
* `internal/themeimage/themeimage.go`, `preview.go`, `themeimage_test.go`, `preview_test.go`: new
* `internal/themedump/themedump.go`, `json.go`, `themedump_test.go`: new
* `internal/cli/theme.go`, `internal/cli/theme_test.go`: new
* `internal/cli/main.go`: the `theme` command added to the root
* `internal/cli/commands_test.go`: `theme` in the help test and the argument cases
* `layout_test.go`: the test of the imports of the theme packages
* `e2e/theme_test.go`: new, with `//go:build linux`
* `docs/theme-format.md`: new
* `README.md`: the status line and the section on `theme dump`

## Tasks

### The images

`internal/themeimage/themeimage.go` and `preview.go`:

```go
// Expand gives the 8-bit channels of an RGB565 colour as the spec defines them, opaque.
func Expand(colour themefile.Colour) color.NRGBA

// Resources gives the images of R11 of a record: none for a bar, an unknown record and a background with offset 0.
func Resources(record themefile.Record) []*image.NRGBA

// Preview gives the 480 x 320 image of R12.
func Preview(theme themefile.Theme) *image.RGBA

// SampleTime is the date and time of the preview, 2026-11-15 09:41:07, and SampleValue the value of every number.
var SampleTime = time.Date(2026, 11, 15, 9, 41, 7, 0, time.UTC)
const SampleValue = "42"
```

* an RGB565 pixel of a resource is `uint16(low) | uint16(high)<<8`, read as two bytes, low byte first
* `opaque(pixels, width, height)` gives the image of a background and a startup frame, and `masked(alpha, pixels,
  width, height)` that of an image frame; `mask(alpha, width, height, colour)` gives a text and a glyph, each pixel the
  expanded colour with the alpha of the mask, no opacity guess applied
* a strip of R11 is the glyphs of `Glyphs()` side by side, as wide as the sum of their widths and as high as the glyph
  height
* the preview starts as `image.RGBA` filled with opaque black, and each entry is drawn with `draw.Draw(..., draw.Over)`,
  which clips to the bounds of the preview and nothing else, so that glyphs are clipped to the screen and not to their
  box. A background with offset 0 is `draw.Draw` of a `image.Uniform` of its expanded colour over its box. A bar is
  the four one-pixel lines of the inside edge of its box in `#ff00ff`, so a box of width or height 0 draws nothing and
  one of width or height 1 or 2 is filled. A startup and an unknown record are not drawn
* a number draws the glyphs of `SampleValue` and a datetime the glyphs of its `FormatText()` with `SampleTime`, both
  through one function that places a list of glyph masks with their widths from the box's x, each the next at the
  right edge of the one before, tops at the box's y. In a format, `1` is the year as `%04d`, `2` to `6` the month,
  day, hour, minute and second as `%02d`, each digit drawn as its digit glyph, a glyph character as its glyph of the
  other glyphs, and any other byte as nothing with no advance

### The dump

`internal/themedump/themedump.go` and `json.go`:

```go
type File struct{ Name string; Content []byte }

// Dump gives the files of the dump in this order: theme.json, the PNG files of the records in the order of the
// entries and of their images, and preview.png; and the warnings of the reader. An error is that of the reader.
func Dump(file []byte) ([]File, []string, error)
```

* the PNG files of an entry are named as R14 says, from `themeimage.Resources`, with `slot%02d-<type name>.png`,
  `slot%02d-<type name>-frame%d.png` counting from 1 and `slot%02d-<type name>-glyphs.png`. An image whose bounds are
  empty is left out and gets no name
* each image is encoded with `png.Encode` into a `bytes.Buffer`, which is deterministic (R19)
* the JSON is written with an `encoding/json` `Encoder` with `SetIndent("", "  ")` and `SetEscapeHTML(false)`, whose
  `Encode` ends it with a newline (R13). Its records are per-type structs whose field order is the order of R13:

```go
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
	Files []string `json:"files"` // never nil, so an empty list is []
}
type numberJSON struct {
	head
	AlignmentGuess byte      `json:"alignment_guess"`
	Colour         colourHex `json:"colour"`
	DecimalsGuess  byte      `json:"decimals_guess"`
	OpacityGuess   byte      `json:"opacity_guess"`
	Offset         uint32    `json:"offset"`
	GlyphHeight    uint16    `json:"glyph_height"`
	GlyphWidths    []uint16  `json:"glyph_widths"`
	GlyphOrder     string    `json:"glyph_order"`
	Bytes46To63    hexBytes  `json:"bytes_46_63_unknown"`
	tail
}
```

  `encoding/json` puts the promoted fields of an embedded struct at its place, so `head` comes first and `tail` last.
  The other types follow the same shape with the fields of their table in the order of the bytes:
  background `byte_11_unknown`, `fill_colour_guess`, `offset`, `bytes_18_63_unknown`; startup `offset`,
  `frame_count`, `duration_ms_guess`, `frame_period_ms_guess`, `bytes_20_63_unknown`; image `offset`, `frame_count`,
  `byte_16_unknown`, `frame_period_ms_guess`, `bytes_19_63_unknown`; text `offset`, `colour`, `opacity_guess`,
  `bytes_18_63_unknown`; datetime `byte_11_unknown`, `colour`, `opacity_guess`, `offset`, `glyph_height`,
  `digit_width`, `other_glyph_widths`, `other_glyph_order`, `format`; bar `bytes_11_63_unknown`. An unknown record has
  only `slot`, `type`, `type_name` `"unknown"`, `raw` and `files`, as R13 gives the common head for a known type only
* `format` is a `*string` with `omitempty`, nil when a byte of `FormatText()` is outside `0x20` to `0x7e`, so that an
  empty format is written as `""` and a non-printable one is left out
* `hexBytes []byte` and `colourHex themefile.Colour` have `MarshalJSON`, giving lowercase hexadecimal with no
  separators and the four digits `%04x`
* `raw` is `record.Encode()`, which the tests of the reader prove equal to the slot's bytes
* the top level is `{file_size, header{byte_0_unknown, raw}, records, warnings}`, with `records` and `warnings` never
  nil

### The command

`internal/cli/theme.go`: `session.themeCommand()` gives `theme`, with the one subcommand `dump <file.dat> <out-dir>`
and `cobra.ExactArgs(2)`, added to the root after `run`:

1. `readTheme(path)`, which gives `cannot read the theme file <path>: <cause>` as it does for `upload` (R17)
2. `themedump.Dump`; its error is wrapped as `fmt.Errorf("cannot read the theme file %s: %w", path, err)` (R17)
3. only then `os.MkdirAll(outDir, 0o755)`, its failure as `cannot create the output directory <dir>: <cause>`, and
   for each file `os.WriteFile(filepath.Join(outDir, name), content, 0o644)`, its failure as `cannot write the file
   <path>: <cause>`, each with the inner error of a `*fs.PathError` so that the path is named once, as
   `readNamedFile` does. `os.WriteFile` replaces a file of the same name and leaves other files (R13)
4. each warning as `warning: <warning>` on standard error, then `wrote <count> files to <out-dir>` on standard output,
   with `<out-dir>` as given (R16)

The command never calls `session.open`, `listScreens` or `newSources`, so it opens no device and reads no sensor, with
`--device` or without it (R18). The cobra error path of `run` in `main.go` already writes `sensordeck: ` and exits 1.

## Files

* `internal/faketheme/faketheme.go`, `internal/faketheme/faketheme_test.go`: new
* `internal/themefile/themefile.go`, `records.go`, `errors.go`, `themefile_test.go`, `records_test.go`: new
* `internal/themeimage/themeimage.go`, `preview.go`, `themeimage_test.go`, `preview_test.go`: new
* `internal/themedump/themedump.go`, `json.go`, `themedump_test.go`: new
* `internal/cli/theme.go`, `internal/cli/theme_test.go`: new
* `internal/cli/main.go`: the `theme` command added to the root
* `internal/cli/commands_test.go`: `theme` in the help test and the argument cases
* `layout_test.go`: the test of the imports of the theme packages
* `e2e/theme_test.go`: new, with `//go:build linux`
* `docs/theme-format.md`: new
* `README.md`: the status line and the section on `theme dump`

## Tasks

### The fake theme and the reader

- [ ] Add `internal/faketheme/faketheme_test.go` with tests that `New().Bytes()` is 4096 zero bytes, that `Header`
  and `Slot` write their 64 bytes at `0` and `64 x n`, that two `Resource` calls give `0x1000` and `0x1000` plus the
  length of the first and append the bytes there, that `Head`, `Put16` and `Put32` write big-endian at the given
  positions, and that `Sample` has the slots, boxes and header of "The fake theme" and ends where its last resource
  ends (R22)
- [ ] Add `internal/faketheme/faketheme.go` as "The fake theme" says, until `go test ./internal/faketheme` passes (R22)
- [ ] Add `internal/themefile/records_test.go`, with one test for each of the seven types (R2, R3, R4, R22): a slot
  whose bytes 0 to 63 are all distinct where the layout allows (the type byte fixed, a frame count, the sizes and the
  offset chosen so that the resource fits in the file, every other byte distinct from all others), read with `Parse`
  from a file of `faketheme`, each field compared with the value read from its bytes by hand in the test, `Resource`
  compared with the bytes appended for it, and `Encode()` compared with the 64 bytes of the slot. Add a test that two
  image records with the same offset give the same `Resource` bytes, and that the background with offset 0 has a nil
  `Resource`
- [ ] Add `internal/themefile/themefile_test.go` (R1, R2, R5 to R10, R22):
    - `Parse(faketheme.Sample())` gives the header, `FileSize`, the slots 1 to 7 and 9 in order with their types, and
      the one warning of slot 9
    - a file of the table only, all zeros, gives no entries and no warning; slots used in the order 5, 2 with index
      bytes 9, 1 give the entries in the order 2, 5
    - a slot of type byte `0x00` with a byte other than zero, and one of type `0xaa`, give an `Unknown` with the 63
      bytes and the warnings `slot 3: record type 0x00 is not known, kept as raw bytes` and `... 0xaa ...`
    - a datetime with the format `1x2` followed by byte `0x7f` and `0x00` gives the warnings with `'x'` and
      `'\x7f'` in that order, and a format of 19 bytes with no zero is read to byte 63
    - the errors of R6 for lengths 0 and 4095, with their texts
    - for each of the six types with a resource, an offset one byte too far gives a `*ResourceError` (`errors.As`)
      with the slot, the type name, `offset`, the offset, the size and the file size, and the text of R7; an image
      with offset `0xfffffff0`, frame count 255 and a box of 65535 x 65535 gives the error with the size in 64 bits
      and the offset `0xfffffff0`, with no wrap
    - two malformed records in slots 4 and 2 give the error of slot 2
    - a header and tags, indexes and boxes off the screen that differ from the samples, a resource inside the table and
      a gap between resources give no error and no warning
    - for every length from 0 to `len(faketheme.Sample())`, `Parse` of the prefix gives either a theme and a nil error,
      or an error, and does not panic; the full length gives a theme
- [ ] Add `internal/themefile/themefile.go`, `records.go` and `errors.go` as "The reader" says, until
  `go test ./internal/themefile` passes (R1 to R10)
- [ ] Add to `layout_test.go` the test `TestThePackagesOfThemesDoNoIO`: for each non-test Go file of
  `internal/themefile`, `internal/themeimage`, `internal/themedump` and `internal/faketheme`, every import is of the
  standard library or one of these four packages, and none is `os`, `os/exec`, `os/signal`, `net`, `syscall` or
  `unsafe`. A directory that does not exist yet is skipped by the walk, so the test passes now and holds as the
  packages are added (R1)
- [ ] Run the five commands of the suite and commit: `feature-0010 Read the record table of a theme file`

### The images

- [ ] Add `internal/themeimage/themeimage_test.go` (R11, R22): `Expand` of `0000`, `ffff`, `f800`, `07e0`, `001f`
  and `8410` against the formula of the spec; for a record of each type with a resource, built with `faketheme` and
  read with `Parse`, the count and bounds of `Resources` and the pixel values at chosen points, including an RGB565
  pixel stored as bytes `0x1f 0xf8` that must read as `f81f` and not `1ff8`, the alpha of the alpha plane of an image
  frame at a point where it is neither 0 nor 255, a text pixel of mask 128 in its colour, and the first column of the
  `-` glyph in a number strip and of the `秒` glyph in a datetime strip; and no image for a bar, an unknown record and
  a background with offset 0
- [ ] Add `internal/themeimage/themeimage.go` until the test passes (R11)
- [ ] Add `internal/themeimage/preview_test.go` (R12, R22), each at chosen points of the 480 x 320 preview:
    - an empty theme is opaque black everywhere
    - a background with offset 0 fills its box with its colour and leaves the outside black; one with a resource shows
      its pixels at its x and y
    - two text records overlapping, in slots 2 and 5, show slot 5 on top; a mask of 128 over a white background gives
      a channel within 1 of the source over value
    - an image is drawn as its first frame
    - a number shows the glyph `4` from the box's x and the glyph `2` from x plus the width of `4`, with byte 11 set to
      2 to show that alignment is ignored, and a glyph wider than the box drawn beyond the box
    - a datetime with the format `1-2-3 4:5:6x` and distinct digit masks shows the glyphs of `2026-11-15 09:41:07` at
      their advances, with `x` drawn as nothing and no advance
    - a bar of 6 x 4 has `#ff00ff` on each pixel of its inside edge and black inside; one of 1 x 1 is one pixel
    - a text at x 478 of width 5 and a background with a box past the bottom edge are clipped, with no panic
    - a startup and an unknown record change nothing
- [ ] Add `internal/themeimage/preview.go` until `go test ./internal/themeimage` passes (R12)
- [ ] Run the five commands of the suite and commit: `feature-0010 Render the resources and the preview of a theme`

### The dump

- [ ] Add `internal/themedump/themedump_test.go` (R13 to R15, R19):
    - `Dump(faketheme.Sample())` gives the file names `theme.json`, `slot01-background.png`,
      `slot02-startup-frame1.png`, `slot02-startup-frame2.png`, `slot03-image-frame1.png`, `slot03-image-frame2.png`,
      `slot04-text.png`, `slot05-number-glyphs.png`, `slot06-datetime-glyphs.png` and `preview.png` in that order, and
      the warning of slot 9
    - its `theme.json` equals an expected text written out in the test in full, which fixes every name, the order of
      the keys, the indent of two spaces, the final newline, `[]` for the files of the bar and the unknown record and
      `年月日时分秒` written as UTF-8
    - each PNG decodes to the bounds and a chosen pixel of the image `themeimage` gives for it
    - a datetime whose format holds byte `0x80` has no `format` key, and one whose format is empty has `"format": ""`
    - an image record with width 0, and one with frame count 0, give no PNG file and `"files": []`
    - a resource shared by two image records is written once for each slot
    - two calls on the same input give byte-identical files
    - a short file gives the error of the reader and no files
- [ ] Add `internal/themedump/themedump.go` and `json.go` as "The dump" says, until `go test ./internal/themedump`
  passes (R13 to R15, R19)
- [ ] Run the five commands of the suite and commit: `feature-0010 Describe a theme as JSON and PNG files`

### The command

- [ ] In `internal/cli/commands_test.go`, add `theme` to the names of `TestHelpNamesTheCommandsAndTheOptions`, and to
  `TestWrongArgumentsFailBeforeTheTransportIsOpened` the cases `theme dump` with no argument and with one, giving
  `accepts 2 arg(s), received 0` and `... received 1`
- [ ] Add `internal/cli/theme_test.go`, with the harness of `commands_test.go` (R13, R16 to R18):
    - `theme dump` of `faketheme.Sample()` written to a temporary file, into `<temp>/a/b`, gives status 0, the stdout
      `wrote 10 files to <temp>/a/b`, the stderr of the one warning, and the ten files with the contents of
      `themedump.Dump`; the harness's `opened` is empty, and so it is with `--device /nowhere`
    - an out-dir holding `theme.json` with other content and a file `keep.txt` gets `theme.json` replaced and
      `keep.txt` left
    - a missing file gives `sensordeck: cannot read the theme file <path>: no such file or directory`, and a file of
      100 bytes gives `sensordeck: cannot read the theme file <path>: the file is 100 bytes, shorter than the record
      table of 4096 bytes`, both with status 1, no stdout and no out-dir created
    - an out-dir below a regular file gives `cannot create the output directory <dir>: not a directory`, and an out-dir
      that holds a directory named `theme.json` gives `cannot write the file <dir>/theme.json: is a directory`, both
      with status 1. These two hold for root as well
- [ ] Add `internal/cli/theme.go` as "The command" says and add the command to the root in `internal/cli/main.go`,
  until `go test ./internal/cli` passes (R13, R16 to R18)
- [ ] Run the five commands of the suite and commit: `feature-0010 Add the command theme dump`

### The end-to-end tests

- [ ] Add `e2e/theme_test.go`, with `//go:build linux`, running the built executable with `run` and a
  `fakescreen.New(test)` to which no screen is added (R23):
    - `theme dump` of `faketheme.Sample()` gives status 0, the stdout of R16 and the stderr
      `warning: slot 9: record type 0x77 is not known, kept as raw bytes\n`; the out-dir holds exactly the ten files of
      R13 to R15; `preview.png` decodes to 480 x 320 with a chosen pixel of the background and one of the bar
    - `theme.json` decoded with `encoding/json` into a struct whose records are `[]map[string]json.RawMessage` has
      `file_size`, the header, and for each of the eight records chosen fields of its type, such as `offset`,
      `frame_count`, `colour`, `glyph_widths`, `other_glyph_order`, `format` and `bytes_11_63_unknown`, and its `files`
    - a second run into another directory gives byte-identical files (R19)
    - `--device /nonexistent theme dump ...` succeeds in the same way (R18)
    - a file of 100 bytes gives status 1, the stderr of R17, no stdout and no out-dir
- [ ] Run the five commands of the suite and commit: `feature-0010 Test theme dump end to end`

### The documents

- [ ] Add `docs/theme-format.md`, in the project's own words, with everything of "What is known of the format" of the
  spec: how the facts were established (by inspecting the bytes of three vendor theme files and composing previews
  from these rules alone, with no code or text of another project, and none of the files committed), the layout, the
  header, the common head and the tags seen, each record type with each of its fields in a table marked known, guess or
  unknown as the spec marks it, the pixel formats, the glyph orders, the format of a datetime, a section of open
  questions listing each item R20 names, and that the preview of `theme dump` draws numbers left-aligned because the
  device's alignment is not known. It quotes no bytes of a vendor file other than the field values the spec quotes and
  names no vendor theme file (R20)
- [ ] In `README.md`, name `sensordeck theme dump` in the status line among the commands that work, and add the
  section `## Reading a theme file` after "The Corsair iCUE Commander Core XT": what the command does, the
  `theme.json`, the PNG of each resource and `preview.png` it writes, that the preview shows every number as 42 and the
  date and time as 2026-11-15 09:41:07, draws numbers left-aligned and bars as magenta outlines, that it touches no
  device and works with no screen connected, and a link to `docs/theme-format.md` (R21)
- [ ] Commit: `feature-0010 Document the theme format and theme dump`

### Before handing over

- [ ] Run the five commands of the suite and `npm run check-changes`, and check every requirement R1 to R24 against
  the tree, ticking the tasks in this plan (R24)
