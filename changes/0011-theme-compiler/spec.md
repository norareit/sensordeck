# A compiler from a theme folder to the theme file

## Context

Step 5 of the roadmap of the initial plan is the theme compiler: sensordeck's own declarative themes, written as a
folder, compiled into the theme file `img.dat` that the screen plays. Unit `0010-theme-file-reader` gives the reader
`internal/themefile`, whose typed records encode their own 64 bytes, the renderer `internal/themeimage`, which draws
the resources and the preview of a theme file, and `docs/theme-format.md`, the description of the format. The compiler
writes exactly the layout they read, builds its output from their types, and gets its preview from their rendering.

This unit adds:

- the format of a theme folder: `theme.yaml` and its assets
- the compiler, which turns a folder into the bytes of a theme file and the mapping of metric to tag that it chose
- `sensordeck theme compile <theme-dir> <out.dat>` and `sensordeck theme preview <theme-dir> <out.png>`
- the documentation of the folder format and a README section on writing a theme

"R*n* of unit 0010" below names a requirement of `changes/0010-theme-file-reader/spec.md`. "The format document" is
`docs/theme-format.md`.

### What the samples hold where the format is unknown

The compiler must write every byte of every slot, and many bytes have no known meaning. The values below are those of
the three vendor samples of unit 0010. Most are in the format document. The rest were read for this spec from the
dumps that `sensordeck theme dump` made of the samples on the human's machine, which are not committed. None of them
is confirmed on the device.

- The header is byte 0 `0x96` and 63 zeros in all three
- The index byte: in the vendor default theme the background has index 0 and the widgets after it are numbered 1, 2,
  3 and so on in the order of the slots. The orange sample numbers its records in the same way from 0, with one number
  skipped. The tool-made sample has the background at 0 and the others in another order
- Number: byte 11 is 1, byte 14 is 0, byte 15 is `0xff` and bytes 46 to 63 are zero in every number of the samples.
  The box is larger than the value it shows in most numbers (for example 82 x 41 for digits of 18 x 36), and smaller
  in height than the glyphs in one, so the box does not follow from the glyphs in a way the samples show
- Datetime: byte 11 is 1 and byte 14 is `0xff` in every datetime of the samples, and the format is followed by zero
  bytes. The six CJK glyphs are 28, 25 and 22 pixels wide in the samples, nearly twice the digit width
- Text: byte 17 is `0xff` and bytes 18 to 63 are zero in every text of the samples. The box is the size of the mask
- Image: byte 16 is 1 and bytes 19 to 63 are zero in every image, and bytes 17 to 18 are 0 in each image of one frame
- Background: the two samples whose background has pixel data hold byte 11 0, a colour `ffff` and `f79e`, and bytes
  18 to 63 `01 00 00 01` and `01`, each followed by zeros. The tool-made sample, whose background has offset 0, holds
  byte 11 1, colour `0000` and `01 00 00 64` followed by zeros
- The tags of the numbers in the samples are 0 and the tags 1 to 20 (not all of them). No tag above 20 occurs other
  than 21, the tag of every datetime

## Requirements

"The folder" is the theme folder named on the command line. "`theme.yaml`" is the file of that name in it, and its
path in a message is the folder path joined with `theme.yaml`. "The screen" is the rectangle from 0,0 of 480 x 320
pixels. "Within the screen" for a box at x, y of width w and height h means x + w <= 480 and y + h <= 320. "The flash
limit" is 4 MiB, 4194304 bytes.

### The folder format

- **R1** `theme.yaml` is a YAML mapping with these keys, and a key not listed here, at any level, is an error that names
  the key and where it is:
    - `name`, required: the name of the theme, a non-empty string with no control character
    - `author` and `license`, optional: strings with no control character
    - `background`, required: either a colour or the path of a PNG file in the folder. A value that starts with `#` is
      a colour
    - `fonts`, optional: a mapping from a font name, a non-empty string, to a font
    - `widgets`, optional: a list of widgets. Without the key, or with an empty list, the theme is its background alone
- **R2** A colour is a string of `#` and six hexadecimal digits, `#rrggbb`, in either case, written in quotes in YAML
  because `#` starts a comment, as `commander_color` of the `run` config file is
- **R3** A path is relative to the folder and uses `/` between its parts on every platform. It is opened inside the
  folder in such a way that no path, `..` and no symbolic link reach a file outside it
- **R4** A font is a mapping with:
    - exactly one of `file`, the path of a TTF or OTF file in the folder, and `builtin`, the name of one of the twelve
      Go fonts of `golang.org/x/image/font/gofont`: `go-regular`, `go-bold`, `go-italic`, `go-bold-italic`,
      `go-medium`, `go-medium-italic`, `go-mono`, `go-mono-bold`, `go-mono-italic`, `go-mono-bold-italic`,
      `go-smallcaps` and `go-smallcaps-italic`
    - `size`, required: the size in pixels of the font's em, a whole number from 1 to 320
    - `color`, required: a colour of R2
- **R5** A widget is a mapping with `type`, `x` and `y`, where x and y are whole numbers from 0, and the keys of its
  type:
    - `text`: `font`, the name of a font of `fonts`, and `text`, a non-empty single line with no control character
    - `number`: `font`, `metric`, a metric name, and `digits`, optional, a whole number from 1 to 5, 3 without the key:
      the number of digits the box of the value holds
    - `datetime`: `font` and `format`, of R6
    - `image`: `file`, the path of a PNG file in the folder
- **R6** A format is a non-empty string made of the tokens `YYYY` (the year in four digits), `MM` (the month), `DD`
  (the day), `HH` (the hour, 00 to 23), `mm` (the minute) and `ss` (the second), each of the last five in two digits,
  and the separators `.`, `-`, `/`, `:` and space. It is read from left to right, a token where one starts and a
  separator otherwise. Any other character is an error that names it and its position, counted from 1. The compiled
  format, in which each token is the device's digit (`1` to `6` in the order above) and each separator itself, is at
  most 18 bytes. So `YYYY-MM-DD` compiles to `1-2-3` and `HH:mm:ss` to `4:5:6`
- **R7** A metric name is well formed when `sensors.IsName` accepts it, which is the test that the `run` config file
  applies to the names of its `tags:`. The compiler does not check that the machine it runs on can read the metric

### The validation

- **R8** Every error of the folder names `theme.yaml` by its path and the place of the fault: `widget <n> (<type>)`
  with *n* counting the widgets from 1 in the order of the list, `font <name>`, `background`, or the top-level key. It
  has the form `<path of theme.yaml>: <place>: <problem>`. A file of the folder that cannot be read or decoded is named
  by its path in the folder, with the cause. No Go type name appears in an error. A folder with several faults gives
  one of them, and the same folder gives the same one each time
- **R9** These are errors, besides those of R1 to R7:
    - `theme.yaml` that cannot be read or is not valid YAML, or a key given twice in a mapping
    - a font file or a PNG file that does not exist, cannot be read or does not decode, and a font file that holds a
      font collection rather than one font
    - a background PNG that is not exactly 480 x 320
    - a widget that names a font that `fonts` does not have
    - a text widget with a character that its font has no glyph for, a number widget whose font lacks a glyph of `0`
      to `9`, `.` or `-`, and a datetime widget whose font lacks a glyph of `0` to `9`, `.`, `-`, `/`, `:` or space,
      each naming the character
    - a widget whose box (R16) is not within the screen
    - more than 62 widgets, so that with the background the theme has more than 63 records
    - a compiled file larger than the flash limit, with its size and the limit in the message
- **R10** The size of a PNG is read from its header before its pixels are decoded, and a PNG too large for the screen
  is an error before it is decoded. The box of a text is measured before its mask is made, and no mask is made for a
  box that is not within the screen

### What the compiler writes

- **R11** The compiler takes the folder and gives the bytes of a theme file and the mapping of metric to tag, or the
  error of R8. Its records are values of the types of `internal/themefile`, and the 64 bytes of each slot are those
  their `Encode` gives
- **R12** The file holds:
    - slot 0, the header: byte 0 `0x96` and 63 zeros
    - slot 1, the background record; slot 2 and up, one record for each widget in the order of the list, so that the
      order of the slots is the order of drawing in the preview; every slot after them all zero
    - from `0x1000`, the resource of each record in the order of the slots, back to back with no gap and none shared,
      and nothing after the last
- **R13** The common head of each record: the index 0 for the background and the slot number minus 1 for a widget;
  the tag of R18 for a number, 21 for a datetime and 0 for the others; the x, y, width and height of its box of R16
- **R14** The bytes whose meaning is unknown or a guess hold the values of the samples, as listed in the context:
    - background: byte 11 0, the colour of a solid background or `0000` for a PNG one in bytes 12-13, byte 18 1 and
      every other byte from 18 to 63 zero
    - image: frame count 1, byte 16 1, bytes 17-18 0, bytes 19-63 zero
    - text: byte 17 `0xff`, bytes 18-63 zero
    - number: byte 11 1, byte 14 0, byte 15 `0xff`, bytes 46-63 zero
    - datetime: byte 11 1, byte 14 `0xff`, and the compiled format of R6 in bytes 45 to 63, followed by zero bytes
- **R15** The resources:
    - a background is always 480 x 320 RGB565 pixels with an offset other than 0. A solid background is every pixel of
      its colour. A PNG background is its pixels composited over opaque black where they are not opaque
    - an image is one frame: its alpha plane, the alpha of the PNG, then its colour plane, the colour of the PNG with
      the alpha left out (the non-premultiplied colour)
    - a text is one alpha mask of its box, a number the glyph set of its twelve glyphs, and a datetime the glyph set
      of its twenty-one glyphs, as R17 makes them
    - a colour of 8 bits per channel is made RGB565 by rounding each channel to the nearest of its 5 or 6 bits, as
      `(value x 31 + 127) / 255` for red and blue and `(value x 63 + 127) / 255` for green in whole-number division.
      A PNG of 16 bits per channel is first reduced to 8
- **R16** The box of each record:
    - the background: 0, 0, 480 x 320
    - an image: at its x and y, of the size of its PNG
    - a text: at its x and y, of the width of its text as the font measures it, rounded up to a whole pixel, and of
      the glyph height
    - a number: at its x and y, of `digits` x the digit width and of the glyph height
    - a datetime: at its x and y, of the width of its format as drawn (each `YYYY` four digit widths, each other token
      two, each separator its width) and of the glyph height
- **R17** The glyphs of a font are rasterised from the font with `golang.org/x/image/font/opentype` at the size of the
  font in pixels (72 dots per inch) with no hinting, anti-aliased, as alpha masks in which the alpha of the rasteriser
  is the alpha of the mask:
    - the glyph height is the font's ascent rounded up plus its descent rounded up, and the baseline is the ascent
      rounded up below the top of each mask. What a glyph draws above or below the mask is cut off
    - the digit width is the largest advance of the ten digits, each rounded up. Each digit is drawn in a cell of the
      digit width, moved right by half of the cell width minus its rounded-up advance, rounded down, so that the ten
      digits have the same width and values do not move sideways as they change
    - each of `.`, `-`, `/`, `:` and space is a mask of its own advance rounded up, drawn from the left edge
    - the six CJK glyphs of a datetime are fully transparent masks of the digit width, whatever the font holds
    - a text is drawn from the left edge of its mask as `golang.org/x/image/font` draws a string, with the kerning
      that the face gives
- **R18** The tags: the distinct metric names of the number widgets, sorted by their bytes, get the tags 1, 2, 3 and up
  in that order, with 21 skipped. Two number widgets with the same metric have the same tag. No number gets 0 or 21,
  and every tag fits in the one byte of the tag and is below 64

### The commands

- **R19** `sensordeck theme compile <theme-dir> <out.dat>` compiles the folder and writes the theme file to
  `<out.dat>` and the tags file next to it, at the path of `<out.dat>` with its extension, as Go's `filepath.Ext`
  gives it, replaced by `.tags.yaml`, or with `.tags.yaml` added when there is none: `img.dat` gives
  `img.tags.yaml`. Both files are replaced when they exist. Neither the directory nor its parents are created
- **R20** The tags file is a comment line that says the block is to be pasted into the config file of `sensordeck run`,
  then a `tags:` mapping from each metric name to its tag, in ascending order of tag, two spaces indented, in the
  shape of the `tags:` section of the README. For a theme with no number widget it is `tags: {}`. For a theme with at
  least one, the file read by `internal/config` gives back the same mapping
- **R21** `sensordeck theme preview <theme-dir> <out.png>` compiles the folder in memory and writes to `<out.png>` the
  PNG of the preview of R12 of unit 0010 of the compiled file read back by `internal/themefile`. It writes no other
  file
- **R22** On success, `compile` writes `wrote <out.dat> (<size> bytes) and <tags path>` and `preview` writes
  `wrote <out.png>` to standard output, each on one line, and the exit status is 0
- **R23** On an error of the folder, both commands fail as the other commands do, with `sensordeck: ` before the
  message of R8 and exit status 1, and write no file. A failure to write a file names its path
- **R24** Neither command opens a device or reads a sensor, with `--device` or without it. Both work with no screen
  connected, on Linux and on Windows
- **R25** The same folder gives byte-identical output files, run after run of the same executable, and wherever the
  folder lies: the output holds no time, no path and nothing of the order of a Go map

### The round trip

- **R26** The bytes the compiler gives, read by `internal/themefile`, give no error and no warning, a header of byte 0
  `0x96`, and one record in each of the slots 1 to the number of widgets plus 1, each equal, field by field and with
  its resource, to the record the compiler built. In it each widget is found again: its type, x and y, the colour of
  its font, the tag of its metric, its compiled format, and for an image its pixels as R15 converts them

### The documents

- **R27** A new document, `docs/theme-folder.md`, describes the folder format in the project's own words: every key of
  R1 to R6 with its type, whether it is required and its default, the Go font names, the tokens and separators of a
  format and what they compile to, the metric names by a link to the README, how the tags are assigned and where the
  tags file is written, how glyphs and colours are made (R15 to R17), the limits of R9, and the values the compiler
  writes for the bytes of unknown meaning (R13, R14) with the statement that they come from the vendor samples and are
  not confirmed on the device. The format document links to it
- **R28** `README.md` has a section on writing a theme: what a theme folder is, a complete example `theme.yaml` with a
  background, two fonts, one of them a Go font, and a widget of each of the four types, the two commands, the tags file
  and how its block goes into the config file, and how a compiled theme is put on the screen today with
  `sensordeck upload` and `sensordeck run`. It links to `docs/theme-folder.md`. The status line names `theme compile`
  and `theme preview` among the commands that work, and the line of `theme preview` under "Planned usage" shows its
  arguments. The section "Credits" names `golang.org/x/image` and the Go fonts with their BSD licence

### Tests

- **R29** The tests build their themes in code or in a temporary folder, with the Go fonts as their fonts, written to
  the folder as TTF files where a test needs a `file:` font, and PNGs made in code. They need no file of the host, no
  vendor file and no device. They cover at least:
    - each record type compiled, with every byte of its slot checked against R13 and R14
    - the round trip of R26 for a theme with a widget of each type and two numbers of the same metric
    - the glyph metrics of R17: equal digit widths, the centring of a narrower digit, the glyph height, the transparent
      CJK glyphs, and a text mask against its measured width
    - the colour conversion of R15 at 0, 255 and a value that rounds up, and the compositing of a PNG background
    - the tags of R18: the order, the shared tag, the skip of 21 with more than 20 metrics, and the tags file of R20
      read back by `internal/config`
    - each error of R1 to R9, with the place R8 says it names, including a path that leads outside the folder by
      `..` and by a symbolic link, a PNG too large for the screen, 63 widgets, and a theme over the flash limit
    - R25 with two compilations of the same folder and of a copy of it elsewhere
- **R30** The end-to-end tests run the built executable's `theme compile` and `theme preview` on a theme folder in a
  temporary directory, and check the output of R22, the theme file read back by `internal/themefile`, the tags file,
  the preview against the preview of the compiled file, a failure of R23 that leaves no file, and R24 with no fake
  screen
- **R31** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 A new dependency, `golang.org/x/image`.** The device draws numbers and text from pre-rendered glyphs, so the
  compiler must rasterise fonts itself. `golang.org/x/image/font/opentype` parses TTF and OTF files and rasterises
  their outlines in pure Go, with no cgo, so the single executable and the cross-compilation to Windows are kept. Its
  module also holds the Go fonts, so a theme can be written and every test run with no font of the host. It is
  BSD-licensed by the Go authors, which is compatible with MIT, and the initial plan names it. Rejected: writing a TTF
  rasteriser, which is a project of its own; a cgo binding to FreeType, which breaks the pure-Go build; a fixed bitmap
  font, which gives up any TTF of the theme, the point of the theme format. The twelve Go fonts are built into the
  executable, which grows it by about 2 MB, because a theme that names one must compile on any machine
- **D2 One widget, one record, in the order of the list.** The order of the slots is the only drawing order the format
  gives, and the preview draws in it, so the order of the list is the order in which a user layers the widgets. The
  background takes slot 1, which leaves 62 slots for widgets. Resources are not shared even when two records could
  share one, because sharing is an optimisation the flash limit does not need for the themes of this unit, and it is
  an item of "Beyond the request"
- **D3 The unknown bytes are copied from the samples.** A byte whose meaning is unknown is safest at a value the device
  is known to draw correctly, which is a value of the samples. Where the samples agree, that value is written. Where
  they differ, the compiler writes the values of a sample of the same kind: for the background, the values of the two
  samples whose background has pixel data, with byte 21 at 0 as in the vendor default theme. The fill colour guess is
  the colour of a solid background and black for a PNG one, because if it is the colour drawn where there is no image,
  black is what the screen shows where nothing is drawn. The index follows the vendor default theme. None of this is
  confirmed on the device, and the documents say so
- **D4 A solid background is a full image.** The meaning of a background with offset 0 is unknown: the tool-made
  sample has one, followed by a full-screen image that covers it. A full image of the colour relies only on what the
  previews confirmed, at the cost of 307200 bytes of the 4 MiB. A PNG background is composited over black, because the
  background record has no alpha and black is what the preview starts from
- **D5 The boxes of the compiler.** The samples' boxes of numbers and datetimes do not follow from their glyphs, and
  the meaning of the box to the device is unknown. A box that is exactly what the record draws is the one that agrees
  with the preview, which draws from the top left of the box. For a number the compiler cannot know the longest value,
  so the theme says it with `digits`, 3 by default because the temperatures and percentages that most metrics are fit
  in 3, and at most 5 because a value is 16 bits unsigned (65535). Rejected: a box of 5 digits always, which keeps a
  number with a large font from the right of the screen; and a width and height of the theme's choosing, which asks
  the author to guess a meaning the format document does not know
- **D6 Glyph height and widths from the font's metrics.** The ascent and descent give every glyph of a font the same
  height and baseline, so digits, separators and letters line up, as the format needs one height per glyph set.
  Rounding up never cuts a glyph's advance. The digit width is the widest digit, centred, so that a proportional font
  still gives the equal digit widths of the samples and a value does not jitter. No hinting is used, because hinting is
  not applied the same way by every font file and the advances are rounded up anyway
- **D7 Blank CJK glyphs.** The formats the compiler writes never refer to the six CJK glyphs, so the device never draws
  them. They must still be present for the layout of the glyph set. A transparent mask of the digit width keeps the
  shape the samples have, a glyph of non-zero width, and does not depend on whether the font has them, which the Go
  fonts do not. Rejected: rasterising them when the font has them, a second path for glyphs nobody sees; zero widths,
  which no sample holds and which may matter to the device
- **D8 The compiled format is at most 18 bytes.** The field has 19 bytes and every sample ends its format with a zero
  byte. Keeping one zero leaves the device a terminator whether or not it needs one. The longest readable format of
  all six fields with separators, `YYYY-MM-DD HH:mm:ss`, compiles to 11 bytes. Only `HH` is offered for the hour,
  because the device's fields have two digits with a leading zero and no 12-hour form is known
- **D9 Tags in the order of the metric names, never 0 or 21.** Tag 21 is the clock, which the device fills itself from
  the clock command. Tag 0 is held by numbers of the orange sample that show no metric, so it may mean "unbound".
  Sorting by name makes the tags independent of the order of the widgets, so that a user who changes the layering does
  not have to paste the tags again. The tags start at 1 so that a theme with up to 20 metrics uses only tags the
  samples use; tags above 20 are not seen in any sample and are not confirmed. Rejected: the vendor's own numbering
  (1 for the CPU temperature, 5 for the GPU and so on), which ties the compiler to a guess about meanings the device
  does not need, since `run` sends whatever the mapping says
- **D10 "Known" metrics are well formed ones.** The metrics of a machine are only known when `run` reads them, and the
  numbered ones, such as `nvme.3.temp` or `commander.2.temp`, depend on the hardware present. A theme is also meant to
  be compiled on one machine and shared: the built-in themes of the next step are compiled on every user's machine. So
  the compiler checks the name against the table of metrics, which catches a typo, and not against the host. A metric
  the host cannot read is left out by `run`, as the README says, and the number shows nothing new. Checking the host
  is an item of "Beyond the request"
- **D11 The tags file beside the theme file, in the config's shape.** The live link from a theme to `run` is a later
  unit, so the user carries the mapping by hand today, and a block in the exact shape of the config's `tags:` is the
  least work. The path is derived from the theme file's, so that one argument names both and they cannot be confused.
  A theme with no number gives `tags: {}`, which the config file rejects as it rejects any empty mapping: such a theme
  needs no values from `run`
- **D12 Errors named by place.** The author fixes `theme.yaml` by hand, so the error names the file and the widget by
  its number in the list, as the editor shows it. One error at a time is what the config file of `run` gives, and the
  same fault each time keeps the tests and the user's runs stable
- **D13 Determinism within one executable.** Nothing of the output depends on the clock, on the path of the folder or
  on map order. The rasteriser computes in floating point, and the Go compiler may fuse multiplications and additions
  differently on another architecture, so byte identity across architectures is not promised. The later cache keys a
  compiled theme by the content of its folder and compiles on the machine that uploads, which needs identity on one
  machine only
- **D14 The preview goes through the file.** `theme preview` reads back the bytes the compiler gives and draws them
  with the renderer of unit 0010, so the preview shows the file the device would get, and its numbers show `42` and
  its datetimes 2026-11-15 09:41:07, left-aligned, as R12 of unit 0010 draws them. Rejected: drawing the theme from
  `theme.yaml` directly, a second renderer that could disagree with the file
- **D15 The threat model.** A theme folder may come from anyone, since themes are meant to be shared. The compiler
  defends against a theme that reads files outside its folder, by paths, `..` or symbolic links (R3), against images
  that are large in their header (R10), and against a panic on malformed YAML or PNG. It relies on
  `golang.org/x/image` for malformed font files. It does not defend against a file in the folder that is large in
  itself, such as a huge `theme.yaml` or font, or against a folder that changes while it is compiled. The output is
  written only to the paths the user names and the path derived from it
- **D16 No device check in this unit.** Whether the device draws a compiled theme as the preview shows it, and with
  it the unknown bytes, the tags above 20, the order of drawing and the alignment of a number in its box, is to be
  checked by the human on the device later. The unit is complete without that check, and does not wait for it. A
  finding of the check is a new unit

## Non-goals

- Progress bars: the fields of a bar record from byte 11 on are not decoded, so a bar cannot be compiled
- Animations: images of more than one frame, the startup animation, and the frame period
- A unit or label inside a number widget, such as `°C`. A text widget beside the number shows it
- Decimals, negative values and alignment of a number: the device's values are whole 16-bit unsigned numbers, and the
  meaning of the alignment guess is unknown
- The 12-hour clock, names of days and months, and the CJK unit characters in a format
- A colour per widget other than its font's: a second font of the same file gives another colour
- The built-in themes, the next step of the roadmap
- The live link between a theme and `run`: `theme use`, `theme install`, `theme list`, `theme new`, the compile cache,
  the active theme in the config file and the upload on change
- A `device` key in `theme.yaml`: there is one device
- Reading vendor themes into the folder format
- JPEG or other image formats than PNG

## Beyond the request

1. Sharing identical resources, such as the glyph sets of two numbers of the same font. Cost: a map from resource
   bytes to offset in the writer and a test. Without it, a theme with many numbers of one font is larger by a few KB
   for each, and a theme that reaches the flash limit gets the clear error of R9
2. A warning for each number whose metric the machine cannot read now, from the sources `sensors` uses. Cost: the
   sources in the compile command, which then reads the sensors, and a test with fake sensors. Without it, a number
   with a metric the machine lacks shows nothing new under `run`, which the user sees on the screen
3. Writing both output files through a temporary file and a rename. Cost: a helper and a test. Without it, a write that
   fails half way leaves a partial file, which the next run of the command replaces
4. Reporting every fault of `theme.yaml` at once rather than the first. Cost: a list of errors through the validation.
   Without it, the author fixes one fault per run
5. A colour per widget that overrides its font's colour. Cost: one optional key and its check. Without it, a second
   font entry with the same file gives the colour
6. A test that compiles the example `theme.yaml` of the README, with its files made in the test. Cost: a test that
   reads the README's code block, or a copy of the example kept as test data beside it. Without it, the example can go
   out of date with the format, which a user who copies it sees as a clear error of R8
