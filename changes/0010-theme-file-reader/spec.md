# A reader of the theme file and the document of its format

## Context

The screen is a theme player: a theme file, `img.dat`, is uploaded once to its flash, and the screen draws the
layout, images and pre-rendered glyphs it holds, with the numbers the computer sends for its tags. Unit
`0003-linux-transport-and-command-line` uploads such a file as it is. sensordeck cannot yet read one, and so cannot
yet make one. Step 4 of the roadmap of the initial plan is the research of the format: a reader first, which dumps a
theme file to a JSON description and PNG images, and the document of the format. Step 5, the theme compiler, writes
theme files from the project's own declarative themes and tests its output by reading it back with this reader.

This unit adds:

- a reader that parses the bytes of a theme file into typed values, with no I/O, so that the compiler can reuse its
  types and test round trips against it
- the command `sensordeck theme dump <file.dat> <out-dir>`, which writes a JSON description of the file, a PNG of each
  resource and a preview of the whole screen, so that a theme can be checked by eye without the device
- `docs/theme-format.md`, the description of the format in the project's own words

### Provenance and licensing

The facts below were learned by the orchestrating session by inspecting the bytes of three vendor theme files on the
human's machine: one made by the vendor's theme tool (524070 bytes), an orange themed one (339037 bytes) and the
vendor's default theme (2127133 bytes, the only one with a startup animation). They are called the samples below. No
code or text of any other project was read for them. Every rule of the layout below was checked by composing a preview
of the whole screen of each sample from these rules alone, and all three previews showed the expected screen:
backgrounds, icons, text labels, numbers, the date and the time in place.

The samples are vendor files and are never committed, as `AGENTS.md` requires. The unit is built and reviewed in a
sandbox that does not have them. This spec therefore carries every fact the unit relies on, and the tests build their
own synthetic theme files in code from the layout below. No byte of a vendor file is part of the repository.

### What is known of the format

"Known" below means that the rule holds in all three samples and was confirmed by the previews. "Guess" means a
plausible reading that nothing has confirmed. "Unknown" means no reading at all.

#### Layout

- Every integer of more than one byte in the record table is big-endian, high byte first. Pixel data is little-endian
  (see the pixel formats below)
- Bytes `0x000` to `0xFFF` (4096 bytes) are the record table: 64 slots of 64 bytes each. Slot 0 is the file header,
  slots 1 to 63 hold the records of the widgets, so a theme has at most 63 widgets
- Resource data follows the table. In all three samples the first resource starts at `0x1000`
- A resource offset in a record is an absolute offset in the file, 32 bits big-endian
- In two samples the resources are packed back to back with no gap, and the last resource ends exactly at the end of
  the file. The third has one gap of 256 bytes between two resources
- A resource may be shared: in the default theme one image of 6 frames is used by three image records
- An unused slot is all zero. The used slots are not always contiguous, and the order of the slots is not always the
  order of their index bytes
- The device's flash is 4 MB, so a theme file must fit in 4 MB (from the initial plan, not tested here)

#### The header (slot 0)

Byte 0 is `0x96` (150) in all three samples and every other byte is 0. Its meaning is unknown.

#### The common head of a record (bytes 0 to 10 of a slot)

| Bytes | Meaning | Status |
|---|---|---|
| 0 | the record type | known |
| 1 | the index, an identifier of the record | value known, purpose unknown |
| 2 | the tag | see below |
| 3-4 | x | known |
| 5-6 | y | known |
| 7-8 | width | known |
| 9-10 | height | known |

The tag of a number record is the metric tag the computer sends values for: tags 1 (CPU temperature), 5 (GPU
temperature), 12 (RAM %), 13 (SSD temperature), 18 (upload KiB/s) and 19 (download KiB/s) of the tool-made sample were
confirmed on the device earlier. Number records with tag 0 occur in the orange sample. Date and time records have tag
21 (`0x15`). For progress bars the tag is a guess. Other records have tag 0.

#### The record types

| Type | Name used in this unit |
|---|---|
| `0x81` | background |
| `0x84` | image |
| `0x8B` | bar (progress bar) |
| `0x8E` | datetime |
| `0x92` | number |
| `0x93` | text (static text) |
| `0x94` | startup (startup animation) |

#### Pixel formats

- **RGB565** is 16 bits per pixel: red in the top 5 bits, green in the middle 6, blue in the low 5. In resource data a
  pixel is 2 bytes, little-endian, low byte first. A colour in a record is 2 bytes big-endian
- **Alpha** is 1 byte per pixel, 0 transparent to 255 opaque
- Pixel planes are stored row by row, top row first, left to right, with no padding at the end of a row

#### `0x81` background

| Bytes | Meaning | Status |
|---|---|---|
| 11 | 0 in two samples, 1 in the tool-made one | unknown |
| 12-13 | an RGB565 colour: `ffff`, `f79e` and `0000` in the samples | guess: the fill colour when there is no image |
| 14-17 | offset of the pixel data, or 0 for none | known |
| 18-63 | `01 00 00 01`, `01` and `01 00 00 64` followed by zeros | unknown |

The pixel data is width x height RGB565 pixels, 2 bytes each, with no alpha. The tool-made sample's background has
offset 0, colour `0000`, and a full-screen image record follows it.

#### `0x94` startup

Only the default theme has one: x 0, y 0, 480 x 320.

| Bytes | Meaning | Status |
|---|---|---|
| 11-14 | offset of the frames (`0x1000` in the sample) | known |
| 15 | frame count (5 in the sample) | known |
| 16-17 | 1500 in the sample | guess: the duration in milliseconds |
| 18-19 | 150 in the sample | guess: the period of a frame in milliseconds |
| 20-63 | | unknown |

The frames are width x height RGB565 pixels each, 2 bytes per pixel, back to back with no alpha. In the sample
`0x1000` plus 5 x 307200 is exactly the offset of the next resource.

#### `0x84` image, optionally animated

| Bytes | Meaning | Status |
|---|---|---|
| 11-14 | offset of the frames | known |
| 15 | frame count (1, 3 or 6 in the samples) | known |
| 16 | 1 in every sample | unknown |
| 17-18 | 0 for single frames; 30, 100, 150, 200 and 250 for animations | guess: the period of a frame in milliseconds. In the default theme the same resource is used with 150, 200 and 250 |
| 19-63 | | unknown |

Each frame is an alpha plane of width x height bytes followed by a colour plane of width x height RGB565 pixels, so
3 x width x height bytes. The frames follow one another with no gap.

#### `0x93` text

| Bytes | Meaning | Status |
|---|---|---|
| 11-14 | offset of the mask | known |
| 15-16 | RGB565 colour | known |
| 17 | `0xff` in every sample | guess: the opacity |
| 18-63 | | unknown |

The text is pre-rendered: one alpha mask of width x height bytes, drawn in the colour.

#### `0x92` number

| Bytes | Meaning | Status |
|---|---|---|
| 11 | 1 in every sample | guess: the alignment of the value in its box. Left alignment looked plausible in the previews, centring is as likely |
| 12-13 | RGB565 colour | known |
| 14 | 0 in every sample | guess: the number of decimals, since the device shows whole numbers |
| 15 | `0xff` in every sample | guess, by analogy with byte 17 of text: the opacity |
| 16-19 | offset of the glyph set | known |
| 20-21 | glyph height | known |
| 22-45 | twelve 16-bit glyph widths, for the glyphs `0 1 2 3 4 5 6 7 8 9 . -` in this order | known |
| 46-63 | | unknown |

The glyph set is the twelve glyphs back to back in the order of the widths, each an alpha mask of glyph height x its
own width bytes, row by row, drawn in the colour. Its size is the glyph height times the sum of the twelve widths. In
the samples the ten digit widths are equal.

#### `0x8E` datetime

| Bytes | Meaning | Status |
|---|---|---|
| 11 | 1 in every sample | unknown |
| 12-13 | RGB565 colour | known |
| 14 | `0xff` in every sample | guess, by analogy with byte 17 of text: the opacity |
| 15-18 | offset of the glyph set | known |
| 19-20 | glyph height | known |
| 21-22 | the width of each of the ten digits | known |
| 23-44 | eleven 16-bit widths, for `.` `-` `/` space `:` and the six CJK unit characters 年 月 日 时 分 秒 (year, month, day, hour, minute, second), in this order | known |
| 45-63 | the format, ASCII, padded with zeros | known |

The glyph set is the ten digits `0` to `9`, each of the digit width, followed by the eleven other glyphs in the order
above, each an alpha mask of glyph height x its width, back to back. Its size is the glyph height times the sum of ten
digit widths and the eleven other widths.

In the format, the digits `1` to `6` stand for the year, month, day, hour, minute and second. The other characters
are drawn as the glyph of the same character in the set. The samples hold the formats `1-2-3` and ` 4:5:6`, with a
leading space in the second, and the previews drew them as `2026-11-15` and `09:41:07`, so the year has four digits
and the other fields two, with a leading zero. How a format refers to the six CJK glyphs is unknown, since the format
is ASCII.

#### `0x8B` bar

Only the orange sample has bars, four of them, and all four have the same bytes from byte 11 on:
`01 00 00 a8 00 a8 00 76 b0 00 00 00 00 00 00 76 b0`, followed by zeros. Read as 16-bit values from byte 12 on, they are
`0000`, `a800`, `a800`, `76b0`, `0000`, `0000`, `0000`, `76b0`. There is no resource offset. `a800` (dark red) and `76b0`
(light green) look like RGB565 colours. The fields are unknown and are not decoded in this unit; decoding them needs
experiments on the device.

## Requirements

"The reader" is the code that turns the bytes of a theme file into typed values. "A used slot" is a slot of 1 to 63
that holds a byte other than zero. "The size of a resource" is the number of bytes the record's fields say it takes:
for a background with an offset other than 0, 2 x width x height; for a startup, frame count x 2 x width x height; for
an image, frame count x 3 x width x height; for a text, width x height; for a number and a datetime, the size of its
glyph set given above. A background with offset 0 has no resource. "R*n* of unit 0003" names a requirement of
`changes/0003-linux-transport-and-command-line/spec.md`.

### The reader

- **R1** The reader takes the bytes of a whole file and gives either a theme, with a list of warnings, or an error. It
  reads and writes no file, opens no device and keeps no state between calls. It is part of the Linux and the Windows
  build and imports no package of the device, the transport, the sensors or the command line
- **R2** The theme holds the 64 bytes of the header, the size of the file, and one typed value for each used slot, in
  the order of the slots, with its slot number. A record of each of the seven types of the context has its own type,
  which holds the common head and the fields of its table by name, with the meanings and widths given there
- **R3** In the typed value of a known record, every one of the 64 bytes of its slot belongs to exactly one field: a
  decoded field of its table or a kept range of unknown bytes. So the 64 bytes of the slot are determined by the typed
  value alone. A record of type bar holds its common head and bytes 11 to 63 as one unknown range
- **R4** The typed value of a record with a resource gives the bytes of that resource, a range of the file of the
  size of the resource that starts at its offset. Two records with the same offset give the same bytes
- **R5** A slot that holds a byte other than zero and whose type byte is none of the seven, including a type byte of
  0, is a record of an unknown type. Its typed value holds the type byte and the other 63 bytes as they are, and the
  reader gives the warning `slot <n>: record type 0x<tt> is not known, kept as raw bytes`, with the slot number in
  decimal and the type as two lowercase hexadecimal digits
- **R6** A file of fewer than 4096 bytes is an error: `the file is <size> bytes, shorter than the record table of 4096
  bytes`
- **R7** A record whose resource, from its offset over its size, runs past the end of the file is an error that names
  the slot, the type name, the field of the offset by its JSON name of R13, the offset in hexadecimal, the size and the
  size of the file, in the form `slot <n> (<type name>): <field> 0x<offset> with <size> bytes runs past the end of the
  file of <file size> bytes`. The size and the end are computed in 64 bits, so that no value of the fields can make them
  wrap. No buffer of the size of a resource is made before this check has passed
- **R8** When several records are malformed, the error is that of the first in the order of the slots. The reader
  does not panic on any input, whatever the bytes and the length
- **R9** A datetime whose format holds a character other than the digits `1` to `6` and the five glyph characters `.`
  `-` `/` space `:` gives the warning `slot <n> (datetime): format character <c> is not known`, once for each such
  character in the order of the format, with the character written as Go's `%q` writes a byte. The format is the
  bytes from byte 45 up to the first zero byte or to byte 63
- **R10** The reader does not check the record table against the screen, the flash size, the header bytes, the tags,
  the index bytes, or the gaps between resources. Anything R5 to R9 do not name is neither an error nor a warning

### The images

The rendering of the resources and of the preview takes typed values and gives images, with no I/O, as the reader
does. "Expanding" an RGB565 colour gives 8 bits for each channel by repeating the top bits of the channel in the low
bits: red and blue as `(value << 3) | (value >> 2)` of their 5 bits, green as `(value << 2) | (value >> 4)` of its 6,
so that 0 gives 0 and the largest value gives 255. "Drawing a mask in a colour" gives each pixel the expanded colour
with the alpha of the mask, and the opacity guesses of the records are not applied.

- **R11** Each resource gives these images, of the record's width and height unless said otherwise:
    - a background with a resource: one opaque image of its pixels
    - a startup: one opaque image for each frame, in order
    - an image: one image for each frame, in order, each pixel of the colour plane with the alpha of the alpha plane
    - a text: its mask drawn in its colour
    - a number: one strip of the glyph height, as wide as the sum of its twelve widths, with the glyphs left to right
      in the order of the set, each its mask drawn in the colour
    - a datetime: one strip in the same way, with its twenty-one glyphs in the order of its set
- **R12** The preview is a 480 x 320 image composed from the records alone:
    - it starts black and opaque, and the records are drawn on it in the order of the slots, each over what is
      already there, with the alpha of the record (source over), clipped to the screen
    - a background with a resource is drawn as its pixels at its x and y; one with offset 0 fills its box with its
      colour
    - an image is drawn as its first frame at its x and y
    - a text is drawn as its mask in its colour at its x and y
    - a number is drawn as the sample value `42`: the glyphs `4` and `2` side by side, the first with its left edge at
      the box's x, each the next at the right edge of the one before, the tops at the box's y, each its mask in the
      colour. The value is drawn left-aligned whatever byte 11 holds
    - a datetime is drawn as its format with the sample date and time 2026-11-15 09:41:07, from the box's x and y in
      the same way as a number: `1` as the year in four digits, `2` to `6` as the month, day, hour, minute and second in
      two digits with a leading zero, each of the five glyph characters as its glyph, and a character of R9 as nothing,
      with no advance
    - a bar is drawn as the outline of its box, one pixel wide on the inside edge of the box, in `#ff00ff`
    - a startup and a record of an unknown type are not drawn

  The glyphs are not clipped to the box, only to the screen

### The command

- **R13** `sensordeck theme dump <file.dat> <out-dir>` reads the file, creates the output directory with its parents
  when it does not exist, and writes into it `theme.json` and the PNG files of R14 and R15. Files of the same names in
  it are replaced, and other files are left. The JSON is indented with two spaces, ends with a newline, and holds:
    - `file_size`, the size of the file in bytes
    - `header`, with `byte_0_unknown`, the value of byte 0, and `raw`, its 64 bytes as hexadecimal
    - `records`, one object for each used slot in the order of the slots, with `slot`, `type` (the type byte as a
      number), `type_name` (the name of the context, or `unknown`), and for a known type the common head as `index`,
      `tag`, `x`, `y`, `width` and `height`, then the fields of its table, then `raw`, the 64 bytes of the slot as
      hexadecimal, and `files`, the names of the PNG files written for the record, in order, or an empty list
    - `warnings`, the warnings of the reader in order, or an empty list

  The fields of the tables are named in this way:
    - a field whose meaning is known has a plain name: `offset` for the offset of a background, startup, image, text,
      number or datetime, `frame_count`, `colour`, `glyph_height`, `glyph_widths` (the list of the twelve of a number),
      `digit_width`, `other_glyph_widths` (the list of the eleven of a datetime) and `format` (the text of R9 when all
      its bytes are printable ASCII, from `0x20` to `0x7e`, and left out otherwise)
    - a field whose meaning is a guess ends in `_guess`: `fill_colour_guess` of a background, `duration_ms_guess` and
      `frame_period_ms_guess` of a startup, `frame_period_ms_guess` of an image, `opacity_guess` of a text, number and
      datetime, `alignment_guess` and `decimals_guess` of a number
    - a byte of unknown meaning is `byte_<n>_unknown` with its value as a number, and a range of unknown bytes is
      `bytes_<from>_<to>_unknown` with its bytes as hexadecimal, both ends included
    - a number also has `glyph_order` `"0123456789.-"`, and a datetime `other_glyph_order` `".-/ :年月日时分秒"`

  A colour is written as its four lowercase hexadecimal digits of RGB565, such as `"f79e"`. Hexadecimal bytes are
  lowercase with no separators. The JSON names exist for the reader of the dump only, and are no contract for another
  program
- **R14** The PNG files of a record are named after its slot, as two decimal digits: `slot<nn>-background.png`,
  `slot<nn>-startup-frame<k>.png`, `slot<nn>-image-frame<k>.png` with *k* counting the frames from 1,
  `slot<nn>-text.png`, `slot<nn>-number-glyphs.png` and `slot<nn>-datetime-glyphs.png`. A resource shared by several
  records is written once for each. An image with no pixels, of width 0, height 0 or frame count 0, is not written
- **R15** `preview.png` is the preview of R12
- **R16** On success the command writes `wrote <count> files to <out-dir>` to standard output, with the count of the
  files it wrote, `theme.json` included, and each warning as `warning: <warning>` on its own line to standard error,
  and its exit status is 0
- **R17** When the file cannot be read or the reader gives an error, the command fails as the other commands do, with
  `sensordeck: ` before the message and exit status 1. The error of the reader is given as `cannot read the theme
  file <path>: <error of R6 or R7>`. In this case the command writes nothing and does not create the output directory.
  A failure to create the directory or to write a file names that path
- **R18** The command opens no device and reads no sensor, with `--device` or without it. It works with no screen
  connected, on Linux and on Windows
- **R19** The same input gives byte-identical output files

### The documents

- **R20** `docs/theme-format.md` describes the format in the project's own words: everything of "What is known of the
  format" above, the layout, the header, the common head, every record type with each of its fields, the pixel formats,
  the glyph orders, the format of a datetime, the tags seen and how the facts were established. Each field and each
  rule is marked as known, a guess or unknown, as the context marks it, and the document has a section that lists the
  open questions: byte 11 and byte 14 of a number, the bars, the startup timings, the opacity bytes, the header, the
  index, how a datetime refers to its CJK glyphs, the order in which the device draws the records, and whether the
  device clips glyphs to their box. It says that the preview of `theme dump` draws numbers left-aligned because the
  device's own alignment is not known yet. It holds no bytes of a vendor file other than the field values the context
  quotes, and names no vendor theme file
- **R21** `README.md` has a section that says what `sensordeck theme dump` does, what it writes, that the preview
  uses the sample value 42 and the sample date and time, draws numbers left-aligned and bars as outlines, and that it
  touches no device, and links to `docs/theme-format.md`. Its status line names the command among those that work

### Tests

- **R22** The tests build their own theme files in code from the layout of this spec, with no file of the repository
  and no vendor file as input. They cover at least:
    - each of the seven types, read into its typed value with each field, from a slot whose bytes are all distinct
      where the layout allows, so that a field read from the wrong byte fails
    - the pixels of each image of R11 at chosen points, the expansion of RGB565, and the little-endian order of the
      pixels against the big-endian order of a record's colour
    - the preview of R12 at chosen points: the order of the slots, source over, the fill of a background with offset 0,
      the left-aligned number, a datetime with each of the six fields and a literal, a bar's outline and its inside, and
      clipping at the edge of the screen
    - each error of R6 and R7, its text, and the slot and field it names, including an offset and a size whose sum does
      not fit in 32 bits
    - every truncation of a small valid file with one record of each type, at every length from 0 to its full length,
      gives a theme or an error and no panic
    - each warning of R5 and R9
- **R23** The end-to-end tests run the built executable on a synthetic file and check the files of R13 to R15, the
  JSON fields of a record of each type, the output of R16, the failure of R17 with no output directory left behind, and
  R18 with no fake screen
- **R24** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 What is an error and what is a warning.** An error is a fact that makes a known record undecodable: the file
  is shorter than the record table, or a resource runs past the end of the file. Going on would mean inventing the
  missing bytes, so the reader stops. An unknown record type loses nothing, since its 63 bytes are kept as they are,
  and the format is only partly known: a vendor theme with a type not seen in the samples must still be dumpable, so
  it is a warning. An unknown character in a datetime format is a warning for the same reason, and the preview draws
  nothing for it. Rejected: making the unknown type an error, which would make the reader useless on the next vendor
  theme; and making a short resource a warning with a partial image, which would show the user a picture the file does
  not hold
- **D2 The first error only, and nothing written on an error.** One error in the order of the slots is deterministic
  and enough to locate the fault, and a dump written partly could be taken for a complete one. Rejected: collecting
  every error, which costs a list type for a case of malformed files that are not the use of the tool
- **D3 Only the end of the file is checked.** A resource that overlaps the record table, gaps, unused bytes, a file
  over 4 MB, a box outside the screen and a header unlike the samples are left unchecked, because none of them stops
  the reader from decoding the file, and the samples give no rule for them. The flash limit belongs to the compiler and
  the upload, which make or send a file for the device. These checks are items of "Beyond the request"
- **D4 Raw bytes beside the decoded fields.** The JSON keeps every slot's 64 bytes in `raw` and names every unknown
  byte, so that nothing of the table is lost and a later unit can find a pattern in bytes not understood today. R3
  makes the typed values just as complete, so that the compiler can write every byte back and test the round trip
  field by field
- **D5 The guesses are in the names.** `_guess` and `_unknown` in the JSON names keep a reader of a dump from taking
  a guess for a fact, which is what the human asked for. When a guess is confirmed later, the field is renamed in the
  unit that confirms it
- **D6 The preview draws in slot order, left-aligned, over black.** The order of the slots is the only order the file
  gives, and the tool-made sample needs it, with its background followed by a full-screen image. Black is what an
  unlit panel shows where nothing is drawn. Left alignment is how the previews of the samples were checked, and
  centring is as likely; the document says so. The startup is not drawn because the device shows it at power on, not
  as part of the screen
- **D7 One sample value for every number.** `42` fits every box made for a temperature or a percentage, and the glyph
  strip of each number shows all twelve glyphs, so a glyph the value does not use is still checked by eye. Rejected:
  a value per tag, which would encode a guess about the tags of a vendor theme. The sample date and time is the one
  the previews of the samples were checked with
- **D8 A bar as a magenta outline.** Its fields are not decoded, and drawing a guess of its colours would look like a
  fact. A colour no theme of the samples uses shows at a glance that it is a placeholder
- **D9 Images per record, not per resource.** Naming the files after the slot lets the JSON of a record point to them
  and keeps the names stable. Writing a shared resource more than once costs only disk space for a dump
- **D10 No new dependency.** The glyphs are pre-rendered in the file, so no font library is needed. The standard
  library's `image`, `image/png` and `encoding/json` do the rest
- **D11 The threat model.** The input is a file the user names, which may be a vendor theme downloaded from anywhere
  and may be malformed or hostile. The reader defends against panics, out-of-bounds reads and buffers sized by a field
  rather than by the file: no input makes it panic, and no resource buffer is larger than the file. It does not defend
  against a file that is large in itself, which the user chose to read. The output is written only under file names the
  command makes, inside the directory the user names. It does not defend against a directory that another user can
  change while the command runs
- **D12 No device check.** The unit touches no device, so nothing in it needs the physical screen. Whether the guesses
  hold on the device is the subject of later units and of the human's checks, and the unit does not wait for them. The
  orchestrating session may compare dumps of its local samples by eye, which is no condition of the unit

## Non-goals

- The theme compiler, the declarative theme format, `theme preview` of a declarative theme and the other `theme`
  commands of the README. They are step 5 and later of the roadmap
- Decoding the fields of a bar, the alignment and decimals of a number, the opacity bytes, the startup timings and how
  a datetime refers to its CJK glyphs. They need experiments on the device
- Playing animations, or writing them as animated GIF or APNG
- Writing a theme file or changing one
- Checking a theme against the device's limits, such as the flash size, the screen or the number of tags
- Reading the vendor's theme project files, of which only the compiled `img.dat` is in scope

## Beyond the request

1. A Go fuzz target for the reader, with the small file of R22 as its seed. The seeds run in `go test`, and a longer
   fuzz run is made by hand. Cost: one test function. Without it, the truncation test of R22 covers short files, and a
   panic on another malformed file would be found when it happens and fixed then
2. A warning when a resource starts inside the record table, below `0x1000`. Cost: one check and one test. Without it,
   such a file is dumped with table bytes shown as pixels, which the user sees in the PNG
3. A list in the JSON of the ranges of the file that no resource covers, such as the 256-byte gap of the tool-made
   sample and bytes after the last resource. Cost: a sort of the resources and a JSON field. Without it, data hidden in
   gaps is not seen, which matters only if the device reads something there
4. A warning when the file is larger than the 4 MB of the flash. Cost: one check. Without it, the user learns it from
   the upload, or from the compiler, which must check it anyway
5. A warning when the header differs from byte 0 `0x96` followed by zeros. Cost: one check. Without it, the difference
   is visible in `header.raw` of the JSON
6. A link from `docs/protocol.md`, where the upload names `img.dat`, to `docs/theme-format.md`. Cost: one sentence.
   Without it, the README links the two documents
