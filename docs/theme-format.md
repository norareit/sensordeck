# The theme file format

A theme file, `img.dat`, is what the screen plays. The computer uploads it once to the flash of the device, as
[the protocol](protocol.md) describes, and the device draws the layout, the images and the pre-rendered glyphs it
holds, with the numbers the computer sends for the tags of the number records.

This document describes the format in the project's own words. `sensordeck theme dump` reads a file as this document
says, and the package `internal/themefile` is the reader. sensordeck's own themes are written as a folder and compiled
into this format, as [the theme folder document](theme-folder.md) describes.

## How the facts were established

The format was learned by inspecting the bytes of three vendor theme files: one made by the vendor's theme tool (524070
bytes), one themed in orange (339037 bytes) and the vendor's default theme (2127133 bytes, the only one with a startup
animation). They are called the samples below. No code or text of another project was read for this, and no file of a
vendor is part of this repository. Every rule of the layout was checked by composing a preview of the whole screen of
each sample from these rules alone. All three previews showed the expected screen: backgrounds, icons, text labels,
numbers, the date and the time in place.

Each rule below carries one of these marks:

* **known**: the rule holds in all three samples and the previews confirmed it
* **observed on the device**: seen on the screen on 2026-11-17, with test themes patched by hand and values sent with
  `sensordeck values`. The device was the screen of the project, and what was not tried is said to be unknown
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
  reader does not check it. The screen draws far less than that, as the next item says
* **Observed on the device** on 2026-11-17, with themes laid out as the compiler of sensordeck lays them out (the record
  table, the background of 307 200 bytes, then the data of the widgets in the order of their slots, back to back): the
  screen draws a widget only when its data ends within the first 503 692 to 504 015 bytes of the theme file. "The data
  of a widget" is its resource, from its offset to the offset of the next resource, or to the end of the file for the
  last one. The facts:
    * a theme of 508 370 bytes showed every widget but the date and the time, its last two widgets and last two
      resources. Every other part of the upload succeeded
    * with the time and the date moved to later slots, the time was drawn when its data ended at 503 369 and at 503 692
      bytes and not drawn when it ended at 504 015. The date, the last resource of each of these themes, ended past
      505 000 bytes and was never drawn
    * a number in slot 42 whose data ended at 496 566 bytes was drawn, so the number of slots is not what limits the
      screen
    * a copy of the theme without one alarm, 505 094 bytes, put the end of the data of the time at 501 062 bytes and that
      of the date at 505 094. The screen showed the time and not the date, as predicted
    * the same theme before the alarm colours of its five temperatures, 436 694 bytes, showed the date and the time

  The one multiple of 4096 in the range of 503 692 to 504 015 is 503 808, `0x7B000`, which the compiler takes as the limit
  and refuses a larger file for. **That choice is a guess within the range.** What sets the limit in the device is not
  known, and the vendor's tool-made sample, 524 070 bytes and laid out otherwise (its background has no data and a
  full-screen image of 460 800 bytes follows it), was not checked against it

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
(`0x15`). The tag of a bar is the tag of a metric: values sent to the tag of a bar were shown by it. Observed on the device on
2026-11-17. Other records have tag 0.

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
| 11 | 1 in every sample | the alignment of the value in its box: 0 left, 1 centred, 2 right. Observed on the device on 2026-11-17, with three numbers of the same box and value side by side. Values above 2 are unknown |
| 12-13 | RGB565 colour | known |
| 14 | 0 in every sample | the thousands mode when not 0. Observed on the device on 2026-11-17: with 1, 1234 showed `1.2` and 12345 showed `12`, and with 2, 1234 showed `1.2` and 999 showed `0.9`. The values 1 and 2 looked the same. Values above 2 and the difference between 1 and 2 are unknown |
| 15 | `0xff` in every sample | the opacity: 0 made the number invisible and 255 opaque. Observed on the device on 2026-11-17. Values from 1 to 254 were not tried and are unknown |
| 16-19 | offset of the glyph set | known |
| 20-21 | glyph height | known |
| 22-45 | twelve 16-bit glyph widths, for the glyphs `0 1 2 3 4 5 6 7 8 9 . -` in this order | known |
| 46-63 | | unknown |

The glyph set is the twelve glyphs back to back in the order of the widths, each an alpha mask of glyph height x its own
width bytes, row by row, drawn in the colour. Its size is the glyph height times the sum of the twelve widths. In the
samples the ten digit widths are equal.

The thousands mode: all three results agree with the value divided by 1000 and cut, not rounded, to one decimal below
10000 and to a whole number from 10000 on. That is a guess, and nothing beyond the three results is established: no
negative value, no value from 10000 to 12344, no value below 999 and no other value of the byte was tried.

On the device, observed on 2026-11-17:

* The glyphs of a number are clipped to the box of the record: -32768 in a box of 5 digits showed `-3276` with the 8
  cut at the right edge of the box, and 40000, which the screen reads as -25536, in a right-aligned box showed `25536`
  with its minus sign cut at the left edge. Only the left and the right edge were seen
* The minus sign takes a place in the box, like a digit
* A number that never received a value shows nothing

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

The common head is that of every record: type, index, tag, x, y, width and height. Only the orange sample has bars, four
of them, and all four have the same bytes from byte 11 on: `01 00 00 a8 00 a8 00 76 b0 00 00 00 00 00 00 76 b0`,
followed by zeros. There is no resource offset. The test themes kept these bytes and changed the ones named below one at
a time.

| Bytes | Meaning | Status |
|---|---|---|
| 11 | 1 in every sample and every test | unknown |
| 12-13 | `0000` in the samples: the colour of the empty part. Observed on the device on 2026-11-17 | observed on the device |
| 14-15 | `a800` in the samples (dark red): the colour of the fill or of the frame. Writing `07e0` into both 14-15 and 16-17 made the fill and the frame green. Which of the two is the fill was not separated | observed on the device, for the pair |
| 16-17 | `a800` in the samples: the colour of the fill or of the frame, as 14-15 | observed on the device, for the pair |
| 18-19 | `76b0` in the samples. Writing `001f` into this and 26-27 had no visible effect | unknown |
| 20-25 | zero in the samples. Writing 50 into 20-21, 22-23 or 24-25 gave no threshold colour | unknown |
| 26-27 | `76b0` in the samples. No visible effect when changed, as 18-19 | unknown |
| 28-63 | zero | unknown |

What was observed on the device on 2026-11-17:

* The value is a share from 0 to 100: 50 filled half the bar, 100 filled it, 0 showed it empty and 150 showed it empty.
  Values from 101 to 149 and negative values were not tried
* A bar that never received a value shows full, in the colour of bytes 14 to 17
* The fill grows from the left edge: at 50 each bar was filled from the left edge to about the middle, and at 100
  entirely. Where the edge of the fill falls in pixels was not measured, and no bar taller than wide was tried
* The frame is drawn inside the box of the bar, an outline one pixel wide
* The tag of a bar is the tag of a metric, and the values were sent to it as to a number

## A threshold colour with two widgets

All of this is **observed on the device** on 2026-11-17.

* No byte of a number or of a bar that was tried gave a threshold colour
* No value makes a number invisible once it showed a value: 65535 shows `-1` and 32768 shows `-32768`, since the screen
  reads every value as a signed 16-bit number
* The test theme was made by hand and had two number widgets on the same spot with the same box, the normal one in white
  and the alarm one in red, each with a tag of its own, both left-aligned (byte 11 at 0). The alarm widget was later in the
  order of the slots than the normal one
* In both widgets the glyph of `-` was fully transparent, every alpha 0, and as wide as the box of the widget. The other
  glyphs, `0` to `9` and `.`, were as the compiler of sensordeck makes them
* A negative value draws the transparent minus first, which fills the box, and pushes the digits past the right edge of the
  box, where the screen clips them. So a widget sent a negative value shows nothing, and such a widget cannot show a
  negative value of its own, since its digits are clipped as those of -1 are
* The normal state: the normal widget was sent the value and the alarm widget -1. The screen showed a white 47 and
  nothing red
* The alarm state: the normal widget was sent -1 and the alarm widget 95. The screen showed a red 95 and nothing white
* Back to normal: a white 52 and nothing red
* The screen keeps its widgets as layers in the order of their slots
* An earlier design put an opaque cover over the widget to hide it. The cover hid every layer below it and not only the
  one widget, so covers do not do

[The theme folder document](theme-folder.md#a-number-with-an-alarm-colour) says how the compiler uses this for the key
`alarm`.

## Open questions

Each of these needs an experiment on the device, or more samples:

* the values of the number bytes that were not tried: byte 15 from 1 to 254, byte 11 above 2, byte 14 above 2 and the
  difference between 1 and 2 of byte 14, and the rule of the thousands mode, whose guess rests on three results
* the bars: byte 11, which of bytes 14-15 and 16-17 is the fill, bytes 18 to 63, the values from 101 to 149 and below 0,
  how a bar fills a box taller than wide, and the exact pixels of the edge of the fill
* the startup timings: bytes 16 to 19 of a startup, whether they are the duration and the period of a frame
* the opacity bytes: byte 17 of a text and byte 14 of a datetime
* the header: what byte 0 means and whether the other bytes matter
* the index byte of a record, whose value is known and whose purpose is not
* the exact size limit within 503 692 to 504 015 bytes, and what the device limits: the end of the data in the file, or a
  budget of another shape that the layout of the compiler cannot tell apart
* whether the vendor samples, one of which is larger than the limit, are drawn in full
* how a datetime refers to its six CJK glyphs, since its format is ASCII
* the order in which the device draws the records: the dump draws them in the order of the slots. It was observed on the
  device on 2026-11-17 that the screen keeps its widgets as layers in the order of their slots, a later slot over an
  earlier one, for two numbers on the same spot, and that an opaque cover over a widget hid every layer below it. It is
  still open whether a later slot is always drawn over an earlier one, and in what order records of other types are
  drawn
* whether the device clips a number at the top and the bottom of its box, and whether it clips the glyphs of a datetime

## What the dump of `sensordeck theme dump` shows

The preview of the dump is composed from the records alone: it starts black, and the records are drawn in the order of
the slots, each over what is there. A startup and a record of an unknown type are not drawn.

* A number is drawn as the sample value `42`, or as `1.2` when byte 14 is not 0, with its glyphs side by side. Byte 11
  places the value in the box: 0 at the left edge, 2 with its right edge at the right edge of the box, and 1 centred,
  and any other value as 0. A number is clipped to its box on all four sides and to the screen, and the alpha of each
  pixel of its glyphs is multiplied by byte 15 over 255
* A datetime is drawn as 2026-11-15 09:41:07, clipped to the screen only
* A bar is drawn at the sample value 42: its frame, one pixel inside the edge of the box, in the colour of bytes 16-17,
  then inside the frame as many columns from the left as the width inside the frame times 42 over 100, rounded down, in
  the colour of bytes 14-15, and the other columns in the colour of bytes 12-13. A box 2 high or wide or less has no
  inside and shows its frame alone

Some of these are choices of the preview and not facts of the device: the centre rounded down when the margin is odd, the
fill rounded down, the fill from the left in a box taller than wide, bytes 14-15 as the fill, and the clipping of a
number at the top and the bottom of its box.

The names of the fields in `theme.json` carry the marks of this document: a name that ends in `_guess` is a guess, and a
name of the form `byte_<n>_unknown` or `bytes_<from>_<to>_unknown` is a byte or a range whose meaning is unknown. When a
guess is confirmed, the field is renamed. A number has `alignment`, `thousands_mode` and `opacity`, each with the value
of its byte. A bar has `byte_11_unknown`, `empty_colour`, `fill_or_frame_colour_14`, `fill_or_frame_colour_16` and
`bytes_18_63_unknown`.
