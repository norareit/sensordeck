# The theme folder

A theme of sensordeck's own is a folder. `sensordeck theme compile` turns it into a theme file, the `img.dat` that the
screen plays and that [the format document](theme-format.md) describes, and `sensordeck theme preview` draws what the
screen would show. This document describes the folder and what the compiler does with it.

## The folder

The folder holds `theme.yaml` and the files it names: PNG pictures and, if the theme wants a font of its own, TTF or
OTF files. Every path in `theme.yaml` is relative to the folder and uses `/` between its parts on every platform. A path
cannot leave the folder: `..` that climbs out of it, an absolute path and a symbolic link that points outside it are
errors. `a/../b.png` stays inside and names `b.png`.

## Naming a theme

The first argument of `theme compile` and `theme preview` is a folder path or a theme name. It is a folder path when it
is `.` or `..` or holds `/`, and on Windows also when it holds `\` or `:`. Any other argument is a name, and the empty
argument is an error. The rule looks at the text alone, so a folder in the working directory is named `./my-theme`.

A name is looked up in this order, and the first found is compiled:

1. an installed theme: the entry of that exact name in the themes directory, when it is a directory or a symbolic link
   to one. The themes directory is `sensordeck/themes` in the config directory of the user: `~/.config/sensordeck/themes`,
   or `$XDG_CONFIG_HOME/sensordeck/themes`, on Linux and `%AppData%\sensordeck\themes` on Windows. A themes directory
   that does not exist holds no theme
2. a built-in theme of that name. The built-in themes are `minimal-dark` and `minimal-dark-probes`

So a theme folder copied into the themes directory is found by its name, and one named like a built-in theme replaces it.

`sensordeck theme list` writes one line for each theme, sorted by name, with tabs between the fields: `<name>` and
`built-in`; `<name>`, `installed` and the folder; or `<name>`, `installed, overrides built-in` and the folder.

## theme.yaml

`theme.yaml` is a YAML mapping. A key that is not listed here, at any level, is an error that names the key and where it
is, and so is a key given twice in a mapping.

| Key | Type | Required | Meaning |
|---|---|---|---|
| `name` | string | yes | The name of the theme: not empty, no control character |
| `author` | string | no | No control character |
| `license` | string | no | No control character |
| `background` | colour or path | yes | A colour, or the path of a PNG of exactly 480 x 320 pixels. A value that starts with `#` is a colour |
| `fonts` | mapping | no | From a font name, a non-empty string, to a font |
| `widgets` | list | no | The widgets, in the order they are drawn. Without the key, or with an empty list, the theme is its background alone |

`name`, `author` and `license` are checked and not written to the theme file, which has no place for them.

### Colours

A colour is `#rrggbb`, six hexadecimal digits in either case, written in quotes in YAML because `#` starts a comment:
`"#1e1e2e"`. A colour that is left out by the comment gets an error that says so.

### Fonts

A font is a mapping with:

| Key | Type | Required | Meaning |
|---|---|---|---|
| `file` | path | one of `file` and `builtin` | A TTF or OTF file in the folder, holding one font and not a font collection |
| `builtin` | string | one of `file` and `builtin` | One of the twelve Go fonts, built into the executable |
| `size` | whole number, 1 to 320 | yes | The size of the font's em in pixels |
| `color` | colour | yes | The colour of everything drawn with this font |

The Go fonts are `go-regular`, `go-bold`, `go-italic`, `go-bold-italic`, `go-medium`, `go-medium-italic`, `go-mono`,
`go-mono-bold`, `go-mono-italic`, `go-mono-bold-italic`, `go-smallcaps` and `go-smallcaps-italic`. They are the Go fonts
of the `golang.org/x/image` module, by the Go authors, under a BSD licence. A theme that names one compiles on any
machine. A second font with the same file and another colour is how a widget gets another colour.

### Widgets

A widget is a mapping with `type`, `x` and `y`, where `x` and `y` are whole numbers from 0 and give the top left corner
of its box, and the keys of its type. The widgets are numbered from 1 in the order of the list, which is how an error
names them, and a theme has at most 62 of them, because the background takes one of the 63 records of a theme file.

| Type | Keys | Meaning |
|---|---|---|
| `text` | `font`, `text` | A fixed text. `font` names a font of `fonts`. `text` is a non-empty single line with no control character |
| `number` | `font`, `metric`, `digits`, `align`, `opacity`, `thousands` | A value that the device fills in. `metric` is the name of a metric. `digits`, a whole number from 1 to 5 and 3 without the key, is how many digits the box of the value holds. `align` is `left`, `center` or `right`, and `center` without the key. `opacity` is a number from 0 to 1, whole or decimal, and 1 without the key. `thousands` is `true` or `false`, and `false` without the key |
| `datetime` | `font`, `format` | The date and the time, which the device fills in from the clock |
| `image` | `file` | A PNG in the folder, drawn at its own size |
| `bar` | `metric`, `width`, `height`, `color`, `background` | A bar that the device fills in from a metric. `metric` is as for a number. `width` and `height`, whole numbers from 1, are the size of its box. `color`, a colour, is the colour of the fill and of the frame. `background`, a colour and `"#000000"` without the key, is the colour of the empty part. A bar has no font |

The screen cuts a value that is wider than the box of its number. The minus sign of a negative value takes a place in the
box like a digit, so a number that shows negative values needs a `digits` one larger than its longest value. With
`thousands` the value is shown in thousands, `1.2` for 1234 and `12` for 12345, with the point taking its own glyph
width. The compiler writes any `opacity` between 0 and 1 as a byte, but only 0 and 1 were seen on the device.

A bar shows the value of its metric as a share from 0 (empty) to 100 (full), as it is: nothing scales it. A value above
100 was seen to show the bar empty, and a bar that never received a value shows full. A metric of another range is made
a share with a custom metric, as the [README](../README.md#writing-a-theme) shows.

A metric name is a built-in one, from the table in the
[README](../README.md#showing-sensor-values-on-the-screen), or a custom one that the user defines under `metrics` in the
config file of `sensordeck run`, as [Custom metrics](../README.md#custom-metrics) says. The compiler checks only that a
name has the form of a metric name, one or more segments joined by single dots, each of the characters `a` to `z`, `0` to
`9`, `_` and `-`, beginning with a letter. It does not know the config file, and does not check that the machine it runs
on can read the metric, so that a theme compiled on one machine can be shared. `run` refuses a theme whose metric is
neither built in nor defined under `metrics`, and names the metric.

### Formats of a datetime

A format is a non-empty string made of tokens and separators, read from left to right:

| Token | Meaning | Compiles to |
|---|---|---|
| `YYYY` | the year, four digits | `1` |
| `MM` | the month, two digits | `2` |
| `DD` | the day, two digits | `3` |
| `HH` | the hour from 00 to 23, two digits | `4` |
| `mm` | the minute, two digits | `5` |
| `ss` | the second, two digits | `6` |

The separators are `.`, `-`, `/`, `:` and space, and each compiles to itself. Any other character is an error that names
it and its position. The compiled format is at most 18 bytes, which leaves one of the 19 bytes of the field as the
terminator. So `YYYY-MM-DD` compiles to `1-2-3` and `HH:mm:ss` to `4:5:6`. There is no 12-hour form, no name of a day or
month and no unit character.

## The tags

The device takes a value for a tag, a number, and not for a metric. The compiler gives the number and bar widgets their tags:
the distinct metric names of the theme, sorted by their bytes, get 1, 2, 3 and up, with 21 skipped, because 21 is the tag
of the clock. Two widgets with the same metric, a bar and a number included, have the same tag, and no widget gets 0. The tags do not depend on the
order of the widgets.

`sensordeck theme compile <name-or-folder> <out.dat>` writes the tags beside the theme file, in a file named like it with its
extension replaced by `.tags.yaml`, or with `.tags.yaml` added when there is none, so `img.dat` gives `img.tags.yaml`.
It is a block in the shape of the `tags:` section of the config file of `sensordeck run`, to be pasted into it. For a
theme with neither a number nor a bar it is `tags: {}`, and such a theme needs no values from `run`.

The tags file is for a config file with `tags`, which is the way to use a vendor theme. A config file with the key
`theme`, which names this theme by its name or folder, needs no tags block, because `run` compiles the theme and takes
its tags itself.

## How glyphs and colours are made

The device draws numbers, text and dates from pre-rendered glyphs, so the compiler draws them from the fonts, with
`golang.org/x/image/font/opentype`, at the size of the font in pixels (72 dots per inch), with no hinting and with
anti-aliasing. The alpha of the rasteriser is the alpha of the mask.

* The height of every glyph of a font is its ascent rounded up plus its descent rounded up, and the baseline is the
  ascent rounded up below the top. What a glyph draws above or below that is cut off
* The digit width is the largest advance of the ten digits, each rounded up. Each digit is drawn in a cell of that
  width, moved right by half of the cell width minus its own rounded-up advance, rounded down, so that all digits have
  the same width and a value does not move sideways as it changes
* `.`, `-`, `/`, `:` and space are masks of their own advance rounded up, drawn from the left edge
* A datetime has six more glyphs, for characters the formats of the compiler never use. They are fully transparent
  masks of the digit width, whatever the font holds
* A text is drawn from the left edge of its mask with the kerning that the font gives

A colour of 8 bits per channel is made RGB565 by rounding each channel to the nearest of its 5 or 6 bits, as
`(value x 31 + 127) / 255` for red and blue and `(value x 63 + 127) / 255` for green, in whole-number division. A PNG
of 16 bits per channel is first reduced to 8. An image is one frame: its alpha, then its colour without the alpha, so
that a half-transparent pixel keeps its colour. A PNG background is composited over opaque black where it is not opaque,
because the background record has no alpha. A solid background is a full picture of its colour.

## The box of a widget

The box of each widget is exactly what it draws, from its x and y: an image has the size of its PNG, a text the width of
its text as the font measures it, rounded up, a number `digits` digit widths, a bar its `width` and `height`, and a datetime the width of its format as
drawn, each `YYYY` four digit widths and each other token two. All have the glyph height, except an image and a bar. The preview
draws from the top left of the box, so the box and the preview agree.

## The limits

* a box must be within the screen of 480 x 320 pixels
* a background PNG must be exactly 480 x 320
* at most 62 widgets
* the compiled file must fit in the 4 MiB (4194304 bytes) of the flash of the device
* the size of a PNG is read from its header before its pixels, so a PNG too large for the screen is refused before it is
  decoded
* every character of a text must have a glyph in its font, a number needs the glyphs of `0` to `9`, `.` and `-`, and a
  datetime those of `0` to `9`, `.`, `-`, `/`, `:` and space

Every error names `theme.yaml` and the place of the fault, such as `theme.yaml: widget 2 (text): x: want a whole number
from 0`, and a folder with several faults gives one of them, the same each time. The same folder gives byte-identical
files, run after run of the same executable. Byte identity across processor architectures is not promised, because the
rasteriser computes in floating point.

## The bytes the compiler writes where the meaning is unknown

Many bytes of a theme file have no known meaning, as [the format document](theme-format.md) says. The compiler writes
the values that the three vendor samples agree on. **They come from the samples and are not confirmed on the device.**

* The header is byte 0 `0x96` and 63 zeros. Slot 1 is the background and slot 2 and up are the widgets in the order of
  the list. The index byte is 0 for the background and the slot number minus 1 for a widget
* Background: byte 11 is 0, the colour guess is the colour of a solid background and `0000` for a PNG one, byte 18 is 1
  and the rest of 18 to 63 is zero. Its offset is never 0
* Image: one frame, byte 16 is 1, bytes 17 to 18 are 0, bytes 19 to 63 are zero
* Text: the tag is 0, byte 17 is `0xff`, bytes 18 to 63 are zero
* Number: byte 11 is 0 for `left`, 1 for `center` and 2 for `right`, byte 14 is 1 with `thousands: true` and 0 otherwise,
  byte 15 is the opacity times 255 rounded to the nearest whole number, halves up, and bytes 46 to 63 are zero. Bytes
  11, 14 and 15 were observed on the device on 2026-11-17, with 0 and 255 for byte 15 only
* Bar: the tag of its metric, byte 11 is 1, bytes 12-13 are the colour of `background`, bytes 14-15 and 16-17 are both
  the colour of `color`, bytes 18-19 and 26-27 are `76b0` and the rest is zero. Bytes 12 to 17 were observed on the
  device on 2026-11-17. Byte 11, bytes 18-19, bytes 20 to 25 and bytes 26-27 are copied from the vendor sample, and
  changing them showed nothing. A bar adds no resource
* Datetime: the tag is 21, byte 11 is 1, byte 14 is `0xff`, and the compiled format fills bytes 45 on, followed by zero
  bytes

Whether the device draws a compiled theme as the preview shows it, with these bytes, the tags above 20, the order of
drawing and the bytes of a bar that are copied from the vendor sample, has not been checked on the device.
