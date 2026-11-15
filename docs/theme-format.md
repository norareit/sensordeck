# The theme file format

A theme file, `img.dat`, is what the screen plays. The computer uploads it once to the flash of the device, as
[the protocol](protocol.md) describes, and the device draws the layout, the images and the pre-rendered glyphs it
holds, with the numbers the computer sends for the tags of the number records.

This document describes the format in the project's own words. `sensordeck theme dump` reads a file as this document
says, and the package `internal/themefile` is the reader.

## How the facts were established

The format was learned by inspecting the bytes of three vendor theme files: one made by the vendor's theme tool (524070
bytes), one themed in orange (339037 bytes) and the vendor's default theme (2127133 bytes, the only one with a startup
animation). They are called the samples below. No code or text of another project was read for this, and no file of a
vendor is part of this repository. Every rule of the layout was checked by composing a preview of the whole screen of
each sample from these rules alone. All three previews showed the expected screen: backgrounds, icons, text labels,
numbers, the date and the time in place.

Each rule below carries one of three marks:

* **known**: the rule holds in all three samples and the previews confirmed it
* **guess**: a plausible reading that nothing has confirmed
* **unknown**: no reading at all

## Layout

* Every integer of more than one byte in the record table is big-endian, high byte first. Pixel data is little-endian
  (see the pixel formats). Known
* Bytes `0x000` to `0xFFF` (4096 bytes) are the record table: 64 slots of 64 bytes each. Slot 0 is the file header,
  slots 1 to 63 hold the records of the widgets, so a theme has at most 63 widgets. Known
* Resource data follows the table. In all three samples the first resource starts at `0x1000`. Known
* A resource offset in a record is an absolute offset in the file, 32 bits big-endian. Known
* In two samples the resources are packed back to back with no gap, and the last resource ends exactly at the end of the
  file. The third has one gap of 256 bytes between two resources. Known
* A resource may be shared: in the default theme one image of 6 frames is used by three image records. Known
* An unused slot is all zero. The used slots are not always contiguous, and the order of the slots is not always the
  order of their index bytes. Known
* The flash of the device is 4 MB, so a theme file must fit in 4 MB. This comes from the plan of the project and the
  reader does not check it

## The header (slot 0)

Byte 0 is `0x96` (150) in all three samples and every other byte is 0. Its meaning is unknown.

## The common head of a record

Bytes 0 to 10 of a slot:

| Bytes | Meaning | Status |
|---|---|---|
| 0 | the record type | known |
| 1 | the index, an identifier of the record | value known, purpose unknown |
| 2 | the tag | see below |
| 3-4 | x | known |
| 5-6 | y | known |
| 7-8 | width | known |
| 9-10 | height | known |

### The tags seen

The tag of a number record is the metric tag the computer sends values for. Tags 1 (CPU temperature), 5 (GPU
temperature), 12 (RAM %), 13 (SSD temperature), 18 (upload KiB/s) and 19 (download KiB/s) of the tool-made sample were
confirmed on the device earlier. Number records with tag 0 occur in the orange sample. Date and time records have tag 21
(`0x15`). For progress bars the tag is a guess. Other records have tag 0.

## The record types

| Type | Name |
|---|---|
| `0x81` | background |
| `0x84` | image |
| `0x8B` | bar (progress bar) |
| `0x8E` | datetime |
| `0x92` | number |
| `0x93` | text (static text) |
| `0x94` | startup (startup animation) |

A slot whose type byte is none of these is read as a record of an unknown type, with its other 63 bytes kept as they
are.

## Pixel formats

* **RGB565** is 16 bits per pixel: red in the top 5 bits, green in the middle 6, blue in the low 5. In resource data a
  pixel is 2 bytes, little-endian, low byte first. A colour in a record is 2 bytes big-endian. Known
* **Alpha** is 1 byte per pixel, 0 transparent to 255 opaque. Known
* Pixel planes are stored row by row, top row first, left to right, with no padding at the end of a row. Known

The dump expands an RGB565 colour to 8 bits for each channel by repeating the top bits of the channel in its low bits,
so that 0 gives 0 and the largest value gives 255.

## `0x81` background

| Bytes | Meaning | Status |
|---|---|---|
| 11 | 0 in two samples, 1 in the tool-made one | unknown |
| 12-13 | an RGB565 colour: `ffff`, `f79e` and `0000` in the samples | guess: the fill colour when there is no image |
| 14-17 | offset of the pixel data, or 0 for none | known |
| 18-63 | `01 00 00 01`, `01` and `01 00 00 64` followed by zeros | unknown |

The pixel data is width x height RGB565 pixels, 2 bytes each, with no alpha. The tool-made sample's background has offset
0, colour `0000`, and a full-screen image record follows it.

## `0x94` startup

Only the default theme has one: x 0, y 0, 480 x 320.

| Bytes | Meaning | Status |
|---|---|---|
| 11-14 | offset of the frames (`0x1000` in the sample) | known |
| 15 | frame count (5 in the sample) | known |
| 16-17 | 1500 in the sample | guess: the duration in milliseconds |
| 18-19 | 150 in the sample | guess: the period of a frame in milliseconds |
| 20-63 | | unknown |

The frames are width x height RGB565 pixels each, 2 bytes per pixel, back to back with no alpha. In the sample `0x1000`
plus 5 x 307200 is exactly the offset of the next resource.

## `0x84` image, optionally animated

| Bytes | Meaning | Status |
|---|---|---|
| 11-14 | offset of the frames | known |
| 15 | frame count (1, 3 or 6 in the samples) | known |
| 16 | 1 in every sample | unknown |
| 17-18 | 0 for single frames; 30, 100, 150, 200 and 250 for animations | guess: the period of a frame in milliseconds. In the default theme the same resource is used with 150, 200 and 250 |
| 19-63 | | unknown |

Each frame is an alpha plane of width x height bytes followed by a colour plane of width x height RGB565 pixels, so 3 x
width x height bytes. The frames follow one another with no gap.

## `0x93` text

| Bytes | Meaning | Status |
|---|---|---|
| 11-14 | offset of the mask | known |
| 15-16 | RGB565 colour | known |
| 17 | `0xff` in every sample | guess: the opacity |
| 18-63 | | unknown |

The text is pre-rendered: one alpha mask of width x height bytes, drawn in the colour.

## `0x92` number

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

The glyph set is the twelve glyphs back to back in the order of the widths, each an alpha mask of glyph height x its own
width bytes, row by row, drawn in the colour. Its size is the glyph height times the sum of the twelve widths. In the
samples the ten digit widths are equal.

## `0x8E` datetime

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

### The format of a datetime

In the format, the digits `1` to `6` stand for the year, month, day, hour, minute and second. The other characters are
drawn as the glyph of the same character in the set. The samples hold the formats `1-2-3` and ` 4:5:6`, with a leading
space in the second, and the previews drew them as `2026-11-15` and `09:41:07`, so the year has four digits and the
other fields two, with a leading zero. How a format refers to the six CJK glyphs is unknown, since the format is ASCII.

## `0x8B` bar

Only the orange sample has bars, four of them, and all four have the same bytes from byte 11 on: `01 00 00 a8 00 a8 00 76
b0 00 00 00 00 00 00 76 b0`, followed by zeros. Read as 16-bit values from byte 12 on, they are `0000`, `a800`, `a800`,
`76b0`, `0000`, `0000`, `0000`, `76b0`. There is no resource offset. `a800` (dark red) and `76b0` (light green) look like
RGB565 colours. The fields are unknown and are not decoded: decoding them needs experiments on the device. The dump
keeps bytes 11 to 63 as one unknown range.

## Open questions

Each of these needs an experiment on the device, or more samples:

* byte 11 of a number: whether it is the alignment of the value in its box, and what its values mean
* byte 14 of a number: whether it is the number of decimals
* the bars: every field from byte 11 on
* the startup timings: bytes 16 to 19 of a startup, whether they are the duration and the period of a frame
* the opacity bytes: byte 17 of a text, byte 15 of a number and byte 14 of a datetime
* the header: what byte 0 means and whether the other bytes matter
* the index byte of a record, whose value is known and whose purpose is not
* how a datetime refers to its six CJK glyphs, since its format is ASCII
* the order in which the device draws the records: the dump draws them in the order of the slots
* whether the device clips glyphs to the box of their record: the dump clips them to the screen only

## What the dump of `sensordeck theme dump` shows

The preview of the dump is composed from the records alone: it starts black, and the records are drawn in the order of
the slots, each over what is there. Every number is drawn as the sample value `42`, and a datetime as 2026-11-15
09:41:07. Numbers are drawn left-aligned in their box, because the alignment of the device is not known yet. A bar is
drawn as the outline of its box in `#ff00ff`, a placeholder because its fields are not decoded. A startup and a record
of an unknown type are not drawn.

The names of the fields in `theme.json` carry the marks of this document: a name that ends in `_guess` is a guess, and a
name of the form `byte_<n>_unknown` or `bytes_<from>_<to>_unknown` is a byte or a range whose meaning is unknown. When a
guess is confirmed, the field is renamed.
