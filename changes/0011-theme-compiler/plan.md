# A compiler from a theme folder to the theme file

## Approach

One new package and two new commands. The compiler is pure and does no I/O of its own. The command line opens the
folder and writes the files:

* `internal/themecompile`: reads `theme.yaml` from an `fs.FS`, checks it, loads its fonts and PNG files through the
  same `fs.FS`, rasterises the glyphs, builds the records as values of `internal/themefile`, assigns the tags and lays
  out the bytes of the file (R1 to R18, R25, R26). Its errors are `<place>: <problem>`, with no path of the folder in
  them, so that nothing of the output depends on where the folder lies (R8, R25)
* `internal/cli`: `theme compile` and `theme preview`, beside `theme dump`. They open the folder as an `os.Root`,
  pass `root.FS()` to the compiler, put the path of `theme.yaml` before its error, and only then write the output
  (R19 to R24)

`os.Root` refuses any name that leaves the root, through `..` or through a symbolic link, with "path escapes from
parent", and `root.FS()` refuses a name that is not `fs.ValidPath`. The compiler cleans a path of the theme with
`path.Clean` first, so that `a/../b.png` names `b.png` and stays allowed, and refuses a cleaned path that is still not
`fs.ValidPath` (an absolute path, or one that starts with `..`) before opening it. Together they keep every read inside
the folder (R3, D15). This was checked in the sandbox with Go 1.27.4 before the plan was written.

The new dependency is `golang.org/x/image` v0.48.0, which needs Go 1.26 and brings `golang.org/x/text` v0.44.0 as an
indirect dependency and the `golang.org/x/sys` v0.50.0 the module already has (D1). These calls were checked in the
sandbox against v0.48.0, with the Go regular font at size 36:

* `opentype.Parse(data)` gives the font, and for a collection it fails with "invalid single font (data is a font
  collection)". The compiler does not depend on that text: it recognises a collection by its first four bytes `ttcf`
* `opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingNone})` gives a
  `font.Face`. Its `Metrics()` gives `Ascent` 34:01 and `Descent` 7:38, both positive, which `Ceil()` makes 35 and 8
* `face.GlyphAdvance(r)` gives `ok` false for a rune the font has no glyph for (`年` in the Go fonts), so it is the
  test of R9 for a missing glyph. The ten digits of the Go fonts have equal advances
* `font.MeasureString(face, text)` includes the kerning that `font.Drawer.DrawString` applies, and `DrawString` on an
  `*image.Alpha` with `Src: image.Opaque` leaves the rasteriser's alpha in the mask, clipped to the mask

The code past `theme.yaml` depends on `font.Face` alone, which is what lets a test put a face with a narrower digit in
front of a Go font to check the centring of R17. No Go font has one.

None of the items of "Beyond the request" is built, as the record decided.

### The colour of R2

The `#rrggbb` rule is already in `internal/config`, for `commander_color`, and R2 says the two are the same. A first
commit, a refactoring that changes no behaviour, extracts the decoding into `internal/config/config.go`:

```go
// DecodeColour reads a colour written as # and six hexadecimal digits, in either case. It gives false for any other
// text.
func DecodeColour(text string) (Colour, bool)
```

`parseCommanderColour` then calls it and keeps its own messages. The compiler calls it too and writes its own.

### The package

`internal/themecompile`, with no build constraint. Each file has one job:

| File | Job |
|---|---|
| `compile.go` | `Compile`, the order of the steps, `Result`, `FlashLimit`, the error with its place |
| `nodes.go` | reading YAML nodes: a mapping's keys, a string, a whole number in a range, a colour |
| `document.go` | `theme.yaml` to the typed `document`, with every check of R1, R2, R4 to R7 and the font references of R9 |
| `format.go` | the format of a datetime, R6 |
| `files.go` | opening a path of the folder (R3), the PNG header and pixels, the font files and the twelve Go fonts |
| `glyphs.go` | the glyphs and metrics of R17 |
| `pixels.go` | the colour conversion and the pixel planes of R15 |
| `records.go` | the record and resource of each widget and of the background, R13 to R16 |
| `layout.go` | the bytes of the file, R12 |
| `tags.go` | the tags of R18 and the tags file of R20 |

```go
// FlashLimit is the size of the flash of the device, which a theme file must fit in.
const FlashLimit = 4 << 20

// Result is a compiled theme.
type Result struct {
	File    []byte             // the bytes of the theme file
	Records []themefile.Record // slots 1 and up in the order of the slots, each with its offset and resource
	Tags    []config.Mapping   // ascending by tag
}

// Compile compiles the theme folder. Its error is "<place>: <problem>", or "<problem>" for a fault of theme.yaml as a
// whole, and names no path of the folder other than the paths the theme gives.
func Compile(folder fs.FS) (Result, error)

// TagsFile gives the content of the tags file of R20.
func TagsFile(tags []config.Mapping) []byte
```

`Records` is given so that the round trip of R26 can compare what the reader reads back with what was built; the
commands use `File` and `Tags`.

#### The steps of Compile

Each step stops at its first fault, so that a folder with several faults gives the same one each time (R8). No step
iterates over a Go map:

1. read `theme.yaml` with `fs.ReadFile(folder, "theme.yaml")`; a failure is `cannot read the file: <cause>`, with the
   inner error of a `*fs.PathError`
2. `yaml.Unmarshal` into a `yaml.Node`; a syntax error is the parser's message as it is, such as `yaml: line 3: did
   not find expected key`, which holds the line as the place. The document must be one mapping
3. the document, as "The document" says, in the order of the keys of R1 for the top level and of the list for the
   widgets: every structural check, with no file opened
4. the background file, when it is a path, then each font in the order of `fonts` in the file
5. each widget in the order of the list: its glyphs, its box, its resource
6. the tags, then the layout, then the flash limit

#### Places and messages

A place is one of `name`, `author`, `license`, `background`, `fonts`, `font <name>`, `widgets`, `widget <n> (<type>)`,
or `widget <n>` while the type is missing or unknown. A fault of a key inside a place adds the key: `widget 2 (text):
x: want a whole number from 0`. The texts:

| Fault | Message |
|---|---|
| a key not listed | `<place>: unknown key "<key>", the keys are <the allowed keys in alphabetical order, joined with ", " and " and ">` |
| a key given twice | `<place>: the key "<key>" is given twice` |
| a required key missing | `<place>: the key <key> is missing` |
| not a mapping, not a list | `<place>: want a mapping` / `widgets: want a list of widgets` / `fonts: want a mapping from a font name to a font` |
| a string that is not one | `<place>: <key>: want a string` |
| an empty or controlled string | `<place>: <key>: want a non-empty string with no control character` (no "non-empty" for `author` and `license`) |
| a whole number out of range | `<place>: <key>: want a whole number from <low> to <high>`, or `from <low>` for x and y |
| a colour left out by `#` in YAML | `<place>: <key>: no colour given, write it in quotes such as "#1e1e2e", because # starts a comment in YAML` |
| a colour malformed | `<place>: <key>: "<text>" is not a colour such as "#1e1e2e"` |
| the type of a widget | `widget <n>: type: "<value>" is not a widget type, the types are datetime, image, number and text` |
| both or none of `file` and `builtin` | `font <name>: give exactly one of builtin and file` |
| an unknown Go font | `font <name>: builtin: "<value>" is not a Go font, the Go fonts are <the twelve in the order of R4>` |
| a font not in `fonts` | `widget <n> (<type>): font: "<name>" is not a font of fonts` |
| a format character | `widget <n> (datetime): format: the character "<c>" at position <p> is not a token or a separator` |
| a format too long | `widget <n> (datetime): format: compiles to <length> bytes, more than 18` |
| a metric name | `widget <n> (number): metric: "<name>" is not a metric name` |
| too many widgets | `widgets: <count> widgets, more than the 62 that fit beside the background` |
| a path outside the folder | `<place>: "<path>" is not a path inside the folder` |
| a file not read | `<place>: cannot read <path>: <cause>` |
| a file not decoded | `<place>: cannot decode <path>: <cause>` |
| a font collection | `font <name>: <path> holds a font collection, not one font` |
| a background PNG of another size | `background: <path> is <w> x <h>, want 480 x 320` |
| a missing glyph | `widget <n> (<type>): the font <font> has no glyph for "<c>"` |
| a box outside the screen | `widget <n> (<type>): the box at <x>,<y> of <w> x <h> is not within the screen of 480 x 320` |
| the flash limit | `widgets: the compiled file is <size> bytes, larger than the flash limit of 4194304 bytes` |

`<cause>` is the inner error of a `*fs.PathError`, so that the path is named once. A cause from `png` or `sfnt` holds
no Go type name. The command line puts `<path of theme.yaml>: ` before every message (R8, R23).

### The document

`nodes.go` reads a mapping node with one helper, which walks its pairs in document order and gives the first key that
is given twice or not allowed, then the value of each allowed key, so that a check never decodes into a Go type and
no message holds a Go type name (R8):

```go
// fields gives the values of the keys of a mapping, or the error of its first key that is given twice or not in
// allowed, in the order of the document.
func fields(node *yaml.Node, place string, allowed []string) (map[string]*yaml.Node, error)
```

The returned map is only ever looked up by key, never ranged over. A string is a scalar node whose tag is not `!!null`,
taken as `node.Value`, so that `text: 42` is the text `42`. A whole number is a scalar node of tag `!!int` decoded into
an `int64`; a value that does not decode, as one beyond 64 bits, is out of range. A colour is read as config's
`commander_color` is: a null or empty scalar has the message of the left-out colour, any other text goes to
`config.DecodeColour`.

`document.go` gives:

```go
type document struct {
	background backgroundSpec // colour or path, never both
	fonts      []fontSpec     // in the order of the file
	widgets    []widgetSpec   // in the order of the list
}
type backgroundSpec struct {
	colour config.Colour
	path   string // empty for a colour
}
type fontSpec struct {
	name, file, builtin string
	size                int
	colour              config.Colour
}
type widgetSpec struct {
	number            int    // counted from 1
	kind              string // "text", "number", "datetime" or "image"
	x, y              int64
	font              int    // the position of its font in document.fonts
	text, metric      string
	digits            int
	format            []byte // compiled, R6
	file              string
}
```

`name`, `author` and `license` are checked and dropped, since nothing of them is written to the file. `background` is a
colour when its text starts with `#` and a path otherwise (R1). A font name is the key's `Value` and must be non-empty
(R1). The widgets list is checked for its length before any widget, so that 63 widgets give the error of R9 at once.

`format.go`:

```go
// compileFormat compiles a format of R6: YYYY to 1, MM to 2, DD to 3, HH to 4, mm to 5, ss to 6, and each of the
// separators ". - / : space" to itself. Positions count runes from 1.
func compileFormat(text string) ([]byte, error)
```

It walks the runes from left to right, takes a token where the next runes are one of the six, a separator where the
rune is one, and fails at any other rune. An empty format and one compiled to more than 18 bytes fail.

### The files and the fonts

`files.go`:

* `readInside(folder fs.FS, place, given string) ([]byte, error)` checks the path as "Approach" says and reads it with
  `fs.ReadFile`
* `pngSize(content []byte) (width, height int, error)` is `png.DecodeConfig`, which reads the header only (R10). The
  pixels are decoded with `png.Decode` only after the size and the box are checked
* `loadFont(folder, spec) (font.Face, error)`: the TTF bytes of the Go font of `builtin` or the file read, a `ttcf`
  start refused as a collection, `opentype.Parse`, then `opentype.NewFace` with the options of "Approach" (R17)
* the twelve Go fonts are a slice of name and TTF bytes in the order of R4, from the packages `goregular`, `gobold`,
  `goitalic`, `gobolditalic`, `gomedium`, `gomediumitalic`, `gomono`, `gomonobold`, `gomonoitalic`,
  `gomonobolditalic`, `gosmallcaps` and `gosmallcapsitalic` of `golang.org/x/image/font/gofont`. The message of an
  unknown name lists them in that order

### The glyphs

`glyphs.go` (R17, D6, D7):

```go
// glyphFont is a font of the theme made ready to draw glyphs.
type glyphFont struct {
	face       font.Face
	ascent     int // Metrics().Ascent rounded up: the baseline below the top of every mask
	height     int // ascent plus Metrics().Descent rounded up
	digitWidth int // the largest of the ten digits' advances, each rounded up
}

func newGlyphFont(face font.Face) glyphFont
// missing gives the first rune of characters that the face has no glyph for, by GlyphAdvance.
func (glyphs glyphFont) missing(characters string) (rune, bool)
// digit gives the mask of a digit: digitWidth wide, the digit drawn at (digitWidth - its advance rounded up) / 2.
func (glyphs glyphFont) digit(character rune) []byte
// separator gives the mask of one of ". - / : space", as wide as its advance rounded up, drawn from the left edge.
func (glyphs glyphFont) separator(character rune) ([]byte, int)
// textWidth is font.MeasureString rounded up; textMask draws the text from the left edge of a mask of that width.
func (glyphs glyphFont) textWidth(text string) int
func (glyphs glyphFont) textMask(text string, width int) []byte
```

Every mask is an `image.NewAlpha` of `width x height`, whose `Pix` is the plane row by row since its stride is its
width, drawn with a `font.Drawer` of `Src: image.Opaque` whose `Dot` is `fixed.P(left, ascent)`. A CJK glyph is
`make([]byte, digitWidth*height)`, all zero.

### The pixels

`pixels.go` (R15):

```go
// rgb565 rounds each channel to the nearest of its bits: (value*31+127)/255 for red and blue, (value*63+127)/255 for
// green.
func rgb565(red, green, blue uint8) themefile.Colour
// nonPremultiplied gives the colour of a pixel of a decoded PNG with its alpha, 8 bits each.
func nonPremultiplied(picture image.Image, x, y int) color.NRGBA
func solidPlane(colour themefile.Colour) []byte                 // 480 x 320 pixels, little-endian
func backgroundPlane(picture image.Image) []byte                // composited over black
func imagePlanes(picture image.Image) []byte                    // the alpha plane, then the colour plane
```

`nonPremultiplied` gives a `color.NRGBA` pixel as it is, a `color.NRGBA64` pixel with each channel shifted right by 8,
and any other pixel through `color.NRGBAModel`, so that the colour of a half-transparent pixel is not passed through
the premultiplied form and loses nothing. The PNG decoder gives `color.NRGBA` or `color.NRGBA64` for every PNG with
alpha, so the model is reached only for opaque pixels, for which it is exact. Compositing over black is
`(channel*alpha+127)/255` for each channel, then `rgb565`. A pixel of a plane is two bytes, low byte first.

### The records

`records.go` builds each record from its spec, its font, the offset its resource starts at and its slot, with the
fields of R13 and R14, and its `Resource` set (R11). Index is 0 for the background and slot minus 1 for a widget:

| Record | Head | Other fields | Resource |
|---|---|---|---|
| background | type `0x81`, index 0, tag 0, 0, 0, 480 x 320 | `Byte11` 0, `FillColourGuess` the colour or `0000`, `Bytes18To63[0]` 1 | `solidPlane` or `backgroundPlane` |
| image | type `0x84`, tag 0, at x, y, the PNG's size | `FrameCount` 1, `Byte16` 1, `FramePeriodMsGuess` 0 | `imagePlanes` |
| text | type `0x93`, tag 0, at x, y, `textWidth` x `height` | `Colour`, `OpacityGuess` `0xff` | `textMask` |
| number | type `0x92`, the tag of R18, at x, y, `digits` x `digitWidth` by `height` | `AlignmentGuess` 1, `Colour`, `DecimalsGuess` 0, `OpacityGuess` `0xff`, `GlyphHeight` `height`, `GlyphWidths` ten `digitWidth` then `.` and `-` | the ten digits, `.` and `-` |
| datetime | type `0x8E`, tag 21, at x, y, the width of its format by `height` | `Byte11` 1, `Colour`, `OpacityGuess` `0xff`, `GlyphHeight`, `DigitWidth`, `OtherGlyphWidths` the five separators in the order `. - / space :` of `themefile.DatetimeOtherGlyphOrder` then six `digitWidth`, `Format` the compiled format then zeros | the ten digits, the five separators, six blank CJK glyphs |

Every field not named is zero. The colour of a text, number or datetime is its font's colour through `rgb565`. The
width of a datetime's format is four digit widths for `1`, two for `2` to `6`, and its own width for each separator
(R16). For a widget the order is: `missing` on the characters of R9 for its type, then the box against the screen in
`int64`, so that no x or y can overflow, then the resource; for a text the width is measured, and for an image the PNG
header read, before the box is checked, and the mask or the pixels are made only after (R10).

### The layout

`layout.go` (R12): a table of 4096 bytes whose slot 0 is `0x96` and 63 zeros, slot 1 the background and slot 2 and up
the widgets, each `record.Encode()`, then the resources back to back in the order of the slots. The offsets are given
while the records are built: the background's is `0x1000`, and each next one is the offset before it plus the length
of the resource before it. The flash limit is checked on the length of the result.

### The tags

`tags.go` (R18, R20, D9, D11): the distinct metric names of the number widgets, sorted with `slices.Sort` by their
bytes, get 1, 2, 3 and up, with 21 skipped. With at most 62 widgets the largest tag is 63. `TagsFile` gives:

```
# Paste this block into the config file of sensordeck run.
tags:
  cpu.temp: 1
  gpu.temp: 2
```

and for no tag the comment line followed by `tags: {}`. A metric name is made only of the characters that
`sensors.IsName` allows, which need no quoting in YAML.

### The commands

`internal/cli/theme.go`, two subcommands of `theme` beside `dump`, each with `cobra.ExactArgs(2)`:

* `compile <theme-dir> <out.dat>`: `compileFolder`, then `os.WriteFile(out, result.File, 0o644)`, then the tags file
  at `tagsPath(out)` with `themecompile.TagsFile(result.Tags)`, then `wrote <out.dat> (<size> bytes) and <tags path>`
* `preview <theme-dir> <out.png>`: `compileFolder`, `themefile.Parse(result.File)`, `themeimage.Preview`, `png.Encode`
  into a buffer, then `os.WriteFile(out, ..., 0o644)`, then `wrote <out.png>`

```go
// compileFolder opens the folder as a root and compiles it. Its error starts with the path of theme.yaml.
func compileFolder(directory string) (themecompile.Result, error)
// tagsPath gives the path of the tags file of R19: the extension of filepath.Ext replaced, or .tags.yaml added.
func tagsPath(out string) string
```

`compileFolder` takes `filepath.Join(directory, "theme.yaml")` as the path of its messages. A folder that
`os.OpenRoot` cannot open gives `<path of theme.yaml>: cannot read the file: <cause>`, as a `theme.yaml` that cannot be
read does. A write that fails is `cannot write the file <path>: <cause>` with `innerError`, as `theme dump` gives it.
Nothing is written before the whole compilation and, for `preview`, the PNG encoding has succeeded (R23).
`os.WriteFile` replaces a file that exists and creates no directory (R19). Neither command calls `session.open`,
`listScreens` or `newSources` (R24). The cobra error path already writes `sensordeck: ` and exits 1.

## Files

* `internal/config/config.go`, `internal/config/config_test.go`: `DecodeColour` extracted
* `go.mod`, `go.sum`: `golang.org/x/image` v0.48.0 and `golang.org/x/text` v0.44.0
* `internal/themecompile/compile.go`, `nodes.go`, `document.go`, `format.go`, `files.go`, `glyphs.go`, `pixels.go`,
  `records.go`, `layout.go`, `tags.go`: new
* `internal/themecompile/` tests: `helpers_test.go`, `format_test.go`, `glyphs_test.go`, `pixels_test.go`,
  `tags_test.go`, `document_test.go`, `records_test.go`, `compile_test.go`: new
* `internal/cli/theme.go`, `internal/cli/theme_test.go`: the two subcommands and their tests
* `internal/cli/commands_test.go`: the argument cases of the two subcommands
* `layout_test.go`: the test that the compiler does no I/O
* `e2e/theme_test.go`: the end-to-end tests of the two subcommands
* `docs/theme-folder.md`: new
* `docs/theme-format.md`: the link to `docs/theme-folder.md`
* `README.md`: the status line, "Planned usage", the section on writing a theme and "Credits"

## Tasks

### The colour of R2, a refactoring

- [x] In `internal/config/config_test.go`, add a test of `DecodeColour`: `#00c8ff` and `#00C8FF` give 0, 200, 255 and
  true; `00c8ff`, `#00c8f`, `#00c8fff`, `#00c8fg` and the empty text give false (R2)
- [x] Extract `DecodeColour` as "The colour of R2" says and call it from `parseCommanderColour`, with its messages
  unchanged, until `go test ./internal/config` passes and the tests of `commander_color` pass unchanged (R2)
- [x] Run the five commands of the suite and commit: `feature-0011 Extract the decoding of a colour from the config`

### The glyphs, the pixels, the format and the tags

- [x] Add `golang.org/x/image` v0.48.0 with `go get golang.org/x/image@v0.48.0` once the first file below imports it,
  and `go mod tidy`, so that `go.mod` and `go.sum` hold it and `golang.org/x/text` (R17, D1)
- [x] Add `internal/themecompile/helpers_test.go`: a `goFace(test, gofontTTF, size)` that parses a Go font as
  `loadFont` does, a `narrowOne` type that embeds a `font.Face` and gives half the advance of `1` from `GlyphAdvance`,
  and PNG builders that encode an `image.NRGBA` or an `image.NRGBA64` made in code (R29)
- [x] Add `internal/themecompile/glyphs_test.go` (R17, R29), with the Go regular font at 36: `height` is 43, the ascent
  rounded up plus the descent rounded up; each of the twelve number glyphs and the twenty-one datetime glyphs has
  `height` rows; the ten digit masks are `digitWidth` wide; with `narrowOne` the mask of `1` is the mask of `1` drawn
  by a `font.Drawer` at `(digitWidth - ceil(advance)) / 2`, and its leftmost non-zero column is to the right of that
  of the unmoved glyph; a separator is as wide as its advance rounded up; a CJK glyph is all zero and `digitWidth`
  wide; `textMask("AV")` has `textWidth("AV")` columns, which is `font.MeasureString` rounded up, and is not all zero;
  `missing` gives `年` for `"12年"` and false for `"0123456789.-"`
- [x] Add `internal/themecompile/glyphs.go` as "The glyphs" says, until the test passes (R17)
- [x] Add `internal/themecompile/pixels_test.go` (R15, R29): `rgb565` of 0, 0, 0 is `0000` and of 255, 255, 255 is
  `ffff`; red 5 gives a red field of 1 and red 4 of 0, green 3 gives 1 and green 2 gives 0; `backgroundPlane` of a PNG
  pixel of red 255 at alpha 128 is red 128 before `rgb565`, and of an opaque pixel is its colour; `imagePlanes` of a
  pixel of red 200 at alpha 0x40 gives alpha `0x40` and the RGB565 of red 200, not premultiplied; a 16-bit PNG pixel
  of `0xff00` per channel gives 255 and one of `0x00ff` gives 0; pixels are little-endian
- [x] Add `internal/themecompile/pixels.go` as "The pixels" says, until the test passes (R15)
- [x] Add `internal/themecompile/format_test.go` (R6, R29): `YYYY-MM-DD` gives `1-2-3`, `HH:mm:ss` gives `4:5:6`,
  `YYYY-MM-DD HH:mm:ss` gives `1-2-3 4:5:6`, `DD/MM/YYYY` gives `3/2/1`, `YYYY.MM` gives `1.2`; `YYY` fails at
  position 1 with `Y`, `HH:mmx` at position 6 with `x`, `HH h` at position 4 with `h`, `hh` at position 1; the empty
  format fails; `YYYY YYYY YYYY YYYY YYYY YYYY` compiles to 11 bytes and passes; nine `ss` each followed by `:`
  compile to 18 bytes and pass, and the same with one more `:` (19 bytes) fails
- [x] Add `internal/themecompile/format.go` until the test passes (R6)
- [x] Add `internal/themecompile/tags_test.go` (R18, R20, R29): `gpu.temp`, `cpu.temp`, `cpu.temp` give `cpu.temp` 1
  and `gpu.temp` 2; 22 distinct metric names (`nvme.1.temp` to `nvme.22.temp`) give the tags 1 to 20, 22 and 23 in
  byte order of the names, none 0 or 21; `TagsFile` of two mappings is exactly the text of "The tags", and
  `config.Parse` of it gives the same mappings; `TagsFile` of none ends with `tags: {}\n`
- [x] Add `internal/themecompile/tags.go` until the test passes (R18, R20)
- [x] Run the five commands of the suite and commit: `feature-0011 Draw the glyphs and convert the pixels of a theme`

### The document and the compiler

- [x] Add `internal/themecompile/document_test.go`, table-driven over `fstest.MapFS` folders, each case a
  `theme.yaml` text and the exact message wanted, for each fault of "Places and messages" that is checked before a
  file is opened (R1, R2, R4 to R9, R29): an unknown key at the top level, in a font and in a widget; a key given twice
  at the top level and in a widget; `name` missing, empty and with a tab; `author` with a newline; `background`
  missing and as an unquoted `#123456`; a malformed colour and an upper-case one that passes; `fonts` as a list; a
  font with both `file` and `builtin`, with neither, with an unknown Go font, with `size` 0, 321 and `1.5`, and with no
  `color`; `widgets` as a mapping; a widget that is not a mapping; a widget with no type and with type `gauge`; x as
  -1 and as `a`; y missing; a text with an empty text and with a newline; a number with `digits` 0 and 6, and with
  `cpu.tmp`; a datetime with a format of R6's errors; a widget that names a font `fonts` does not have; 63 widgets;
  and a folder with two faults, compiled twice, gives the first in the order of "The steps of Compile" both times
- [x] Add `internal/themecompile/nodes.go`, `document.go` and the steps 1 to 3 of `compile.go`, until the test passes
  (R1, R2, R4 to R9)
- [x] Add `internal/themecompile/records_test.go` (R12 to R16, R29), one test for each record type in a theme of the
  background and that one widget, with the Go regular font: every byte of its slot is compared with a slot written by
  `faketheme.Head`, `faketheme.Put16` and `faketheme.Put32` from the values of "The records", independent of
  `Encode`; slot 0 is `0x96` and 63 zeros, slots 3 to 63 are zero, the background's offset is `0x1000` and the
  widget's is `0x1000 + 307200`, and the file ends where the widget's resource ends. The solid background's resource
  is 307200 bytes of its colour; a PNG background's pixel is composited over black; the text's box is `textWidth` by
  `height`; a number of `digits` 4 is `4 x digitWidth` wide; the datetime of `HH:mm` is `4 x digitWidth` plus the width
  of `:`; the slot of the second of two widgets has index 2
- [x] Add `internal/themecompile/files.go`, `records.go`, `layout.go` and the steps 4 to 6 of `compile.go`, until the
  test passes (R3, R10 to R16)
- [x] Add `internal/themecompile/compile_test.go` (R3, R9, R10, R25, R26, R29):
    - the round trip: a theme with a PNG background, a `file:` font written into the `MapFS` from `gomono.TTF`, a
      builtin font, and a text, an image, two numbers of the same metric, a third number and a datetime; `themefile.Parse`
      of `File` gives no error and no warning, header byte 0 `0x96`, an entry in each of the slots 1 to 7 and none
      after, each entry's record `reflect.DeepEqual` to the one of `Records`, and each widget found again by its type,
      x, y, colour, tag, compiled format and image pixels; the two numbers of one metric share a tag
    - the faults of the files: a font file and a PNG that do not exist, a PNG that does not decode, a font file that is
      not a font, a font file of a `ttcf` start; a background PNG of 480 x 321; a path `../x.png`, `/x.png` and
      `a/../../x.png` refused as outside the folder while `a/../x.png` opens `x.png`; a text with `é` in a font that
      has the glyph and one with `年`, which no Go font has; a box one pixel past the right edge and one exactly at the
      edge, which passes; each with its exact message
    - R10: a PNG whose IHDR claims 10000 x 10000, written by patching the width and height of a 1 x 1 PNG and its
      CRC with `hash/crc32`, gives the size error and not a decode error, for an image widget and for the background;
      a text too wide for the screen gives the box error
    - 62 widgets compile; 9 image widgets of 480 x 320 give the flash limit error with the size 4458496
    - R25: the same `MapFS` compiled twice gives equal `File` bytes and equal tags
- [x] Run the five commands of the suite until all tests pass and commit: `feature-0011 Compile a theme folder into a
  theme file`
- [x] Add to `layout_test.go` a test that no non-test file of `internal/themecompile` imports `os`, `os/exec`,
  `os/signal`, `net`, `syscall`, `unsafe` or a package of the device, the transport or the sensors readers
  (`internal/hidraw`, `internal/linuxsensors`, `internal/commandercore`, `internal/smartmonitor`), and commit with the
  suite passing: `feature-0011 Test that the theme compiler does no I/O` (R24)

### The commands

- [ ] In `internal/cli/commands_test.go`, add the argument cases `theme compile` and `theme preview` with zero and one
  argument, each `accepts 2 arg(s), received <n>` (R19, R21)
- [ ] Add to `internal/cli/theme_test.go`, with the harness of `commands_test.go` and a theme folder written to a
  temporary directory with a `file:` font from `goregular.TTF` and a PNG made in code (R19 to R25, R29):
    - `compile` writes the theme file equal to `themecompile.Compile` of the folder and the tags file equal to
      `TagsFile`, at `img.tags.yaml` for `img.dat` and at `theme.tags.yaml` for `theme`, prints the line of R22 and
      exits 0, with and without `--device /nowhere`, with no transport opened and with `sourcesErr` set so that a
      call of `newSources` would fail; existing files at both paths are replaced
    - `preview` writes a PNG equal to `png.Encode` of `themeimage.Preview` of the compiled file read back, prints
      `wrote <out.png>`, exits 0 and writes no other file in the directory
    - a fault of the folder gives `sensordeck: <dir>/theme.yaml: widget 1 (text): ...` and status 1 for both commands,
      and neither output file exists after it; a folder that does not exist gives `sensordeck: <dir>/theme.yaml:
      cannot read the file: no such file or directory`; an output directory that does not exist gives `cannot write
      the file <path>: no such file or directory` and is not created
    - a symbolic link inside the folder to a PNG outside it, as a background and as an image, gives `cannot read
      <link>: path escapes from parent`, and a link to a PNG inside the folder compiles
    - R25: the folder and a copy of it in another temporary directory give byte-identical theme files and tags files
- [ ] Add `compile` and `preview` to `internal/cli/theme.go` as "The commands" says, until `go test ./internal/cli`
  passes (R19 to R24)
- [ ] Run the five commands of the suite and commit: `feature-0011 Add the commands theme compile and theme preview`

### End to end

- [ ] Add to `e2e/theme_test.go`, with a theme folder of each widget type in a temporary directory and
  `fakescreen.New(test)` with no screen added (R30):
    - `theme compile` exits 0 with the line of R22 and empty standard error; the theme file read by
      `themefile.Parse` gives no error and one entry for each widget plus the background; the tags file read by
      `config.Parse` gives the metrics of the number widgets
    - `theme preview` exits 0 with `wrote <out.png>`, and its PNG is byte-identical to `png.Encode` of
      `themeimage.Preview` of the theme file that `theme compile` wrote
    - a theme with an unknown key exits 1 with `sensordeck: <dir>/theme.yaml: unknown key ...` and leaves neither
      output file
    - both commands with `--device /nonexistent` succeed with no screen in the fake roots
- [ ] Run the five commands of the suite and commit: `feature-0011 Test theme compile and theme preview end to end`

### The documents

- [ ] Add `docs/theme-folder.md`, in the project's own words (R27): every key of R1 to R6 with its type, whether it is
  required and its default (`digits` 3, `widgets` none); the twelve Go font names; the tokens and separators of a
  format, what each compiles to and the limit of 18 bytes; the metric names by a link to the metrics table of the
  README; how tags are assigned (sorted names, from 1, 21 skipped) and where the tags file is written; how glyphs are
  made (size in pixels at 72 dpi, no hinting, glyph height, digit width and centring, separators, blank CJK glyphs,
  text with kerning) and colours (the RGB565 rounding, 16-bit PNGs, compositing of a PNG background over black,
  non-premultiplied image colour); the limits of R9 (the screen, 62 widgets, exact background size, the flash limit
  of 4 MiB); the values written for the bytes of unknown meaning of R13 and R14, stating that they come from the
  vendor samples and are not confirmed on the device; that paths are relative with `/` and cannot leave the folder
- [ ] In `docs/theme-format.md`, add a link to `docs/theme-folder.md` near the top, saying that sensordeck's own
  themes are written as a folder and compiled into this format (R27)
- [ ] In `README.md` (R28):
    - name `theme compile` and `theme preview` in the status line among the commands that work, and name the
      built-in themes, Windows and the service as still to come, in place of making themes, Windows and the service
    - change the line of `theme preview` under "Planned usage" to `sensordeck theme preview <theme-dir> <out.png>`
    - add `## Writing a theme` before "Reading a theme file": what a theme folder is; a complete `theme.yaml` with a
      PNG background, a `file:` font and a `builtin:` font, and one widget of each of the four types; the two commands;
      the tags file and how its block goes into the config file; how the compiled file is put on the screen today with
      `sensordeck upload img.dat` and then `sensordeck run`; a link to `docs/theme-folder.md`
    - under "Credits", name `golang.org/x/image` and the Go fonts, by the Go authors, under the BSD licence
- [ ] Compile the example `theme.yaml` of the README once in a scratch directory outside the repository, with a font
  and a PNG made for it, and correct the example until it compiles. Nothing of the check is committed (R28)
- [ ] Commit: `feature-0011 Document the theme folder and theme compile`

### Before handing over

- [ ] Run the five commands of the suite and `npm run check-changes`, and check every requirement R1 to R31 against
  the tree, ticking the tasks in this plan (R31)

## Verification

- [ ] `go test ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=windows GOARCH=amd64 go vet ./...` succeed with no output of a fault
- [ ] `npm run check-changes` passes

## Commit Points

- [x] `feature-0011 Extract the decoding of a colour from the config`
- [ ] `feature-0011 Draw the glyphs and convert the pixels of a theme`
- [ ] `feature-0011 Compile a theme folder into a theme file`
- [ ] `feature-0011 Test that the theme compiler does no I/O`
- [ ] `feature-0011 Add the commands theme compile and theme preview`
- [ ] `feature-0011 Test theme compile and theme preview end to end`
- [ ] `feature-0011 Document the theme folder and theme compile`
