# Plan: the options of a number widget, the bar widget and a preview that clips

## Approach

The work follows the path of a widget through the code: the reader's typed records and the dump's JSON first, because
the compiler and the preview build on their field names; then the compiler, which reads the new keys and writes the new
bytes; then the preview, which draws what the records hold; then the documents and the committed preview images.

What the tree on `feature-0027-number-options-and-bars` holds, and what the plan builds on:

* `internal/themefile/records.go` holds `Number{Head; AlignmentGuess byte; Colour; DecimalsGuess, OpacityGuess byte;
  Offset; GlyphHeight; GlyphWidths [12]uint16; Bytes46To63; Resource}` and `Bar{Head; Bytes11To63 [53]byte}`, each
  with its `Encode` and its `decode<Type>`. `records_test.go` has `TestTheNumberIsReadFromItsBytes` and
  `TestTheBarKeepsItsBytesAfterTheHead`, which fill a slot with distinct bytes from byte 11 with `distinctFrom`
* `internal/themedump/json.go` holds `numberJSON` with `alignment_guess`, `decimals_guess`, `opacity_guess`, and
  `barJSON` with `bytes_11_63_unknown`. `themedump_test.go` has the golden `theme.json` of `faketheme.Sample()` in
  `TestThemeJSONIsWrittenOutInFull`, whose number (slot 5) and bar (slot 7) hold the old names
* `internal/themecompile/document.go` holds `widgetSpec`, `widgetKeys`, `widgetKind` with the hard-coded text
  `the types are datetime, image, number and text`, `readWidget`, which reads a font for every type but `image`, and
  `readNumberKeys`, which reads and checks `metric` and `digits`. `nodes.go` holds `readString`, `readWholeNumber`,
  `readColour`, `required`, `problem` and `join`
* `internal/themecompile/records.go` holds `buildWidget`, a switch on the kind with `buildImage` as its default, and
  `buildNumber`, which writes `AlignmentGuess: 1` and `OpacityGuess: 0xff`. `compile.go` holds `metricsOf`, which takes
  the metric of the number widgets only, `tagOf` and `fontOf`, which gives no font for an image
* `internal/themeimage/preview.go` holds `SampleValue = "42"`, `barColour` (magenta), `drawRecord`, which draws a number
  with `drawGlyphs` from the box's x onto the whole screen and a bar with `drawOutline`, and `drawGlyphs`, `drawAt`,
  `fill` and `box`. `themeimage.go` holds `Expand`, `Resources`, `mask` and `glyphImages`. `preview_test.go` has
  `addNumber`, `glyphMasks` (each glyph of the one alpha 10 plus its index), `expectColour`, `expectRedWithin`,
  `TestANumberIsDrawnAs42FromTheLeftOfItsBoxWhateverItsAlignment`, `TestABarIsTheOutlineOfItsBoxInMagenta` and
  `TestWhatPassesTheEdgeOfTheScreenIsClipped`, whose bar is magenta
* `internal/cli/runconfig.go` refuses a compiled tag whose metric is neither built in nor defined, for every tag of
  `compiled.Tags`, so a bar's metric is refused there once the compiler gives it a tag. `runcustom_linux_test.go`
  holds `customTheme` and the test of the refusal for a number
* `e2e/theme_test.go` holds `folderTheme`, `writeThemeFolder`, the dump tests of the sample, whose bar at 50,50 of
  6 x 4 is checked as magenta and whose bar JSON is checked for `bytes_11_63_unknown`, and
  `TestThemePreviewIsThePreviewOfTheCompiledFile`
* `docs_test.go` compares `docs/minimal-dark.png` and `docs/minimal-dark-probes.png` with the preview of the built-in
  themes

The bar of the sample has the bytes `01 0000 a800 a800 76b0 ...` from byte 11. `Expand(0xa800)` is `#ad0000`, so its
preview after this unit is a frame of `#ad0000` at 50,50, one column of fill of `#ad0000` at x 51 (the inside is 4 wide
and 4 x 42 / 100 is 1), and black from x 52 to 54 on the rows 51 and 52.

The frozen specs and plans of the units 0010, 0011 and 0012 do not change; this unit supersedes 0010 and names the
requirements of the others in its spec.

### Choices of the plan

* The number of the sample is one constant: `SampleNumber = 42` in `themeimage`, with `SampleValue` made from it with
  `strconv.Itoa`, so that the number and the bar cannot drift apart (D11)
* The opacity is turned into its byte when `theme.yaml` is read, as the format of a datetime is compiled when it is
  read, so `widgetSpec` holds the bytes the record takes: `alignment`, `opacity`, and `thousands` as a bool
* The list of widget types in the error of `widgetKind` is made from the keys of `widgetKeys` with `listOf`, which
  sorts, so that `bar` and any later type are named without editing the text
* The checks of `metric`, shared by the number and the bar, move from `readNumberKeys` into one `readMetric`
* The alpha of the glyphs of a number is scaled in the preview only, by a copy of each glyph image, so that
  `Resources` and the glyph PNG of the dump stay as they are (R17). `(alpha x opacity + 127) / 255` in whole numbers is
  the rounding to the nearest of R15: 255 is odd, so the quotient is never a half
* The glyphs of a number are drawn into `screen.SubImage(box)`, which is the box intersected with the screen, so that
  `draw.Draw` clips to both (R14). Datetimes keep being drawn onto the whole screen (R17)
* The centre offset is `difference >> 1` on a signed `int`, which rounds towards minus infinity in Go where `/ 2`
  rounds towards zero (R13, D7)

## Files

- `internal/themefile/records.go`, `internal/themefile/records_test.go`: the new fields of a number and a bar
- `internal/themedump/json.go`, `internal/themedump/themedump_test.go`: the new JSON names
- `internal/themecompile/document.go`, `internal/themecompile/nodes.go`, `internal/themecompile/document_test.go`: the
  keys `align`, `opacity`, `thousands` and the widget `bar`
- `internal/themecompile/records.go`, `internal/themecompile/records_test.go`: the bytes of a number and the record of
  a bar
- `internal/themecompile/compile.go`, `internal/themecompile/compile_test.go`, `internal/themecompile/tags_test.go`:
  the tags of the bars, no font for a bar, the round trip
- `internal/cli/runcustom_linux_test.go`: the refusal of a bar whose metric is not defined
- `internal/themeimage/preview.go`, `internal/themeimage/preview_test.go`: the alignment, clipping and opacity of a
  number, `1.2`, the bar
- `e2e/theme_test.go`: the sample's bar and JSON names, and a theme with a bar and the new keys
- `docs/minimal-dark.png`, `docs/minimal-dark-probes.png`: drawn again
- `docs/theme-format.md`, `docs/theme-folder.md`, `README.md`: the documents

## Tasks

### The reader and the dump

- [x] In `internal/themefile/records_test.go`, change `TestTheNumberIsReadFromItsBytes` to check `Alignment`,
  `ThousandsMode` and `Opacity` against bytes 11, 14 and 15, and replace `TestTheBarKeepsItsBytesAfterTheHead` with
  `TestTheBarIsReadFromItsBytes`: a slot filled by `distinctFrom(&slot, 11)` gives `Byte11` as byte 11,
  `EmptyColour` as bytes 12-13, `FillOrFrameColour14` as bytes 14-15, `FillOrFrameColour16` as bytes 16-17 and
  `Bytes18To63` as bytes 18 to 63, and `Encode` of the record gives the slot back. Check that `parseOne` gives no
  warning for it (R10, R11, R12, R22)
- [x] In `internal/themefile/records.go`, rename the fields of `Number` to `Alignment`, `ThousandsMode` and `Opacity`,
  and make `Bar` `{Head; Byte11 byte; EmptyColour Colour; FillOrFrameColour14 Colour; FillOrFrameColour16 Colour;
  Bytes18To63 [46]byte}`, with its `Encode` and `decodeBar` reading and writing the fields at their bytes. A comment on
  `Bar` says that one of the two colours is the fill and the other the frame, which was not separated on the device.
  Change every user found by `go build ./...` (R10, R11)
- [x] In `internal/themedump/themedump_test.go`, change the golden JSON of `TestThemeJSONIsWrittenOutInFull`: the
  number of slot 5 holds `"alignment": 1`, `"thousands_mode": 0` and `"opacity": 255` in the places of the three old
  names, and the bar of slot 7 holds `"byte_11_unknown": 1`, `"empty_colour": "0000"`,
  `"fill_or_frame_colour_14": "a800"`, `"fill_or_frame_colour_16": "a800"` and `"bytes_18_63_unknown"` as
  `76b000000000000076b0` followed by 72 zeros, which is 46 bytes, in place of `bytes_11_63_unknown` (R10, R11, R22)
- [x] In `internal/themedump/json.go`, rename the fields and tags of `numberJSON` to `alignment`, `thousands_mode` and
  `opacity`, and give `barJSON` the five fields of R11 with the tags above, the colours as `colourHex`. Fill them in
  `describe` (R10, R11)
- [x] Run `go test ./internal/themefile/... ./internal/themedump/...` and `go vet ./...`

Commit point 1.

### The compiler

- [x] In `internal/themecompile/document_test.go`, add to `TestParseDocumentErrors` the cases, each with the full text
  it must start with:
    - `align: middle` and `align: 1` give `widget 1 (number): align: want left, center or right`
    - `opacity: 1.5`, `opacity: -0.1`, `opacity: "0.5"`, `opacity: 50%` and `opacity: .nan` give
      `widget 1 (number): opacity: want a number from 0 to 1`
    - `thousands: yes`, `thousands: 1` and `thousands: "true"` give `widget 1 (number): thousands: want true or false`
    - a bar without `metric`, without `width`, without `height` and without `color` gives
      `widget 1 (bar): the key <key> is missing`
    - a bar with `width: 0` gives `widget 1 (bar): width: want a whole number from 1`, and one with `height: 0` the same
      for `height`
    - a bar with `font: body` gives `widget 1 (bar): unknown key "font", the keys are background, color, height,
      metric, type, width, x and y`
    - a bar with `metric: cpu..temp` gives `widget 1 (bar): metric: "cpu..temp" is not a metric name`
    - a bar with `color: "#12345"` gives `widget 1 (bar): color: "#12345" is not a colour such as "#1e1e2e"`
    - change the case `widget of type gauge` to want `the types are bar, datetime, image, number and text`

  (R1, R2, R5, R22)
- [x] In `internal/themecompile/document_test.go`, add `TestParseDocumentReadsTheNumberOptions`, which reads the
  widget spec of numbers with `align` `left`, `center` and `right` (alignment 0, 1 and 2), `opacity` 0, 0.5, 1 and the
  integer 0 and 1 (opacity 0, 128, 255, 0 and 255), and `thousands` `true` and `false`, and of a number with none of
  the keys (alignment 1, opacity 255, thousands false); and `TestParseDocumentReadsABar`, which reads a bar with and
  without `background` (black without it) (R1, R3, R5)
- [x] In `internal/themecompile/nodes.go`, add `readFraction(node, place, key string) (float64, error)`, which accepts
  a scalar of the tag `!!int` or `!!float` decoded into a `float64` with `!(value >= 0 && value <= 1)` refused, so that
  NaN is refused, with the text `want a number from 0 to 1`; and `readBoolean(node, place, key string) (bool, error)`,
  which accepts a scalar of the tag `!!bool` only, with the text `want true or false` (R1, R2)
- [x] In `internal/themecompile/document.go`:
    - add `alignment byte`, `opacity byte`, `thousands bool`, `width, height int64`, `colour, background
      config.Colour` to `widgetSpec`
    - add `align`, `opacity` and `thousands` to the keys of `number`, and the entry `"bar": {"background", "color",
      "height", "metric", "type", "width", "x", "y"}` to `widgetKeys`
    - build the list of types of the error of `widgetKind` with `listOf` over the keys of `widgetKeys`
    - move the reading and the check of `metric` out of `readNumberKeys` into `readMetric(widget, values, place)`
    - in `readNumberKeys`, read `align` through a table `{"left": 0, "center": 1, "right": 2}` with the default 1 and
      the error `want left, center or right` for any other value or a value that is no string, `opacity` with
      `readFraction` and its byte as `byte(math.Floor(opacity*255 + 0.5))` with the default 255, and `thousands` with
      `readBoolean` with the default false
    - in `readWidget`, read a `bar` with `readBarKeys` and no font, as an `image` is read with no font, and add
      `readBarKeys`: `readMetric`, `width` and `height` required with `readWholeNumber(node, place, key, 1, -1)`,
      `color` required with `readColour`, and `background` optional with `readColour`, black without it

  (R1, R2, R5)
- [x] In `internal/themecompile/records_test.go`, add:
    - `TestNumberRecordOptions`: a table of numbers with `align: left`, `align: right`, `opacity: 0`, `opacity: 0.5`
      and `thousands: true`, each compared slot for slot with `checkTable` against the slot of `TestNumberRecord` with
      byte 11, 15 or 14 changed to 0, 2, 0, 128 or 1. Move the building of that slot into a helper shared by both
      tests, so that `TestNumberRecord` still checks that a number without the keys gives the slot of today
    - `TestBarRecord`: `{type: bar, metric: cpu.util, x: 3, y: 4, width: 100, height: 10, color: "#00ff00",
      background: "#ff0000"}` gives in slot 2 the head `0x8b`, index 1, tag 1, 3, 4, 100, 10, then byte 11 1, bytes
      12-13 `f800`, bytes 14-15 and 16-17 `07e0`, bytes 18-19 `76b0`, bytes 26-27 `76b0` and zero elsewhere, checked
      with `checkTable`; the same bar without `background` has bytes 12-13 `0000`; and the length of the file is that of
      `TestBackgroundOnly`, since a bar adds no resource
    - `TestABarBeforeANumberMovesNoOffset`: a bar then a number give the number the offset `widgetOffset`
    - `TestABarOutsideTheScreenIsRefused`: a bar at x 400 of width 81 gives the error of `checkBox` naming
      `widget 1 (bar)`

  (R3, R4, R6, R7, R22)
- [x] In `internal/themecompile/records.go`, write `Alignment`, `Opacity` and `ThousandsMode` (1 when `thousands`)
  from the spec in `buildNumber`, and add `buildBar` and its case in `buildWidget`: the box of the spec's x, y, width
  and height checked with `checkBox`, then a `themefile.Bar` with the head of `TypeBar`, the index and the tag, `Byte11`
  1, `EmptyColour` the RGB565 of `background`, both colours the RGB565 of `colour`, and bytes 18-19 and 26-27 of
  `Bytes18To63` set to `76b0` through one named constant whose comment says these are the bytes of every bar of the
  vendor sample (D10). It returns no resource (R3, R4, R6, R7)
- [x] In `internal/themecompile/tags_test.go` or `compile_test.go`, add `TestABarGetsTheTagOfItsMetric`: a theme with a
  bar of `cpu.util` and a number of `cpu.util` gives both tag 1 and `Result.Tags` `[{cpu.util 1}]`, a theme with a
  bar of `gpu.util` alone gives `[{gpu.util 1}]` and its record tag 1, and `TagsFile` of the second holds
  `  gpu.util: 1` (R8, R22)
- [x] In `internal/themecompile/compile.go`, take the metrics of the `bar` widgets as well in `metricsOf`, give a bar
  no font in `fontOf`, and change the comment of `tagOf` to a widget with no metric (R8, R9)
- [x] In `internal/themecompile/compile_test.go`, extend `roundTripFolder` with a number of `cpu.util` with `align:
  right`, `opacity: 0.5` and `thousands: true`, and a bar of `cpu.util` with `background`, and extend `checkWidgets`
  to find them again field by field: the alignment 2, the opacity 128 and the thousands mode 1 of the number, and the
  box, the tag, `Byte11`, the three colours and `Bytes18To63` of the bar. Adjust the counts and the tags of the
  existing checks to the new metric (`cpu.temp` 1, `cpu.util` 2, `gpu.temp` 3) (R22)
- [x] In `internal/cli/runcustom_linux_test.go`, add `TestLoadConfigRefusesABarMetricThatIsNeitherBuiltInNorDefined`,
  a theme folder with a bar of `water.temp` and the error of the number case (R8)
- [x] Run `go test ./internal/themecompile/... ./internal/cli/...`

Commit point 2.

### The preview

- [x] In `internal/themeimage/preview_test.go`, replace
  `TestANumberIsDrawnAs42FromTheLeftOfItsBoxWhateverItsAlignment` with tests at chosen pixels, with the glyph 4 and
  the glyph 2 two wide and a box of 10 at x 30 unless said otherwise (R13, R14, R15, R22):
    - `TestANumberIsAlignedAsByte11Says`: byte 11 0 gives the first column of `4` at 30 and black at 29; 2 gives it
      at 36 and the last column of `2` at 39 with black at 40; 1 gives it at 33; 7 gives it at 30
    - `TestACentredNumberWiderThanItsBoxRoundsDown`: a box of 3 at x 30 with byte 11 1, the value 4 wide, starts at
      30 + (3 - 4) >> 1 = 29, so the pixel at 29 is black, clipped, and 30 is the second column of `4`
    - `TestANumberIsCutAtTheEdgesOfItsBox`: a box of 3 with byte 11 0 shows `4` at 30 and 31 and the first column of
      `2` at 32, and the pixel at 33 is black; with byte 11 2 the second column of `4` is at 30, the pixel at 29 is
      black, and the `2` is at 31 and 32; a number box of height 1 whose glyphs are 2 high leaves the row below the box
      black
    - `TestANumberIsClippedToTheScreen`: a box at x 478 of 10 with byte 11 0 draws at 478 and 479 and the preview has
      no other bounds, and nothing panics
    - `TestTheOpacityOfANumberScalesItsAlpha`: glyph masks of alpha 255 in white over black, byte 15 0 gives black,
      128 gives red 128 within 1, 255 gives 255
    - `TestTheThousandsModeDraws1Point2`: byte 14 1 with byte 11 0 draws `1`, `.` and `2` from x 30, each glyph found
      by its alpha 10 plus its index, `.` being index 10
  Give `addNumber` a parameter for each of the bytes 14 and 15 and a mask alpha, or a slot it fills further, as the
  tests need
- [x] In `internal/themeimage/preview_test.go`, replace `TestABarIsTheOutlineOfItsBoxInMagenta` with
  `TestABarIsDrawnFilledTo42` and `TestABarTooSmallForAnInsideShowsItsFrame`, with a helper `addBar` that writes the
  head and the three colours: a box at 50,50 of 102 x 6 with bytes 12-13 `001f`, 14-15 `07e0` and 16-17 `f800` shows
  red at the four corners and at 50,52 and 151,52; the inside is 100 wide, so the fill is the columns 51 to 92 in
  green and the empty part from 93 to 150 in blue, checked at 92,52 and 93,52; and the pixel at 152,52 is black. A box
  of 2 x 2 is red in its four pixels, a box of 3 x 6 has one inside column, which is blue since 1 x 42 / 100 is 0, and
  a box of 1 x 1 is red. In `TestWhatPassesTheEdgeOfTheScreenIsClipped`, give the bar its colours and check its frame
  at 478,319 in the colour of bytes 16-17 and its inside at 479,319 in that of bytes 14-15, since the inside is 3 wide
  and 3 x 42 / 100 is 1 (R16, R22)
- [x] In `internal/themeimage/preview.go`:
    - add `SampleNumber = 42` and make `SampleValue` `strconv.Itoa(SampleNumber)`, with the comment that the bar takes
      the same value as a share (D11)
    - remove `barColour` and `drawOutline`'s magenta, and add `drawBar(screen, bar)`: the box, nothing for an empty
      box, the frame in `Expand(FillOrFrameColour16)` on the one-pixel inside edge, and when `area.Inset(1)` is not
      empty, `inside.Dx() * SampleNumber / 100` columns from its left in `Expand(FillOrFrameColour14)` and the rest in
      `Expand(EmptyColour)`. Its comment names as the preview's own the fill from 14-15, the rounding down and the fill
      from the left of any box (D9, D12)
    - add `drawNumber(screen, number)`: the characters `SampleValue`, or `1.2` when `ThousandsMode` is not 0; the glyph
      images of the record with each alpha scaled by a copy as `(alpha x Opacity + 127) / 255`; the width of the value
      drawn; the left edge of R13 from `Alignment` and `Width`, with `difference >> 1` for the centre; and
      `drawGlyphs` into `screen.SubImage(box(number.Head)).(*image.RGBA)` (D5, D6, D7)
    - change the comment of `Preview` to say that numbers are clipped to their box and the screen and everything else
      to the screen
  (R13, R14, R15, R16, R17)
- [x] Run `go test ./internal/themeimage/... ./internal/themedump/...`, then draw the committed images again with
  `go run . theme preview minimal-dark docs/minimal-dark.png` and `go run . theme preview minimal-dark-probes
  docs/minimal-dark-probes.png`, look at them to see the numbers centred in their boxes, and run `go test .` (R21)

Commit point 3.

### End to end

- [x] In `e2e/theme_test.go`, change `TestThemeDumpWritesTheFilesOfTheSampleWithNoScreenConnected` to check the bar
  of the sample at 50,50 as `#ad0000` (frame), at 51,51 as `#ad0000` (fill) and at 52,51 as black (empty), with its
  comment; and in `TestThemeDumpJSONHoldsTheFieldsOfEachRecord` check the number for `"alignment": 1`,
  `"thousands_mode": 0` and `"opacity": 255` and the absence of `alignment_guess`, and the bar for
  `"byte_11_unknown": 1`, `"empty_colour": "0000"`, `"fill_or_frame_colour_14": "a800"`,
  `"fill_or_frame_colour_16": "a800"` and `bytes_18_63_unknown`, and the absence of `bytes_11_63_unknown` (R23)
- [x] In `e2e/theme_test.go`, add `TestThemeWithABarAndNumberOptionsCompilesPreviewsAndDumps` with a folder theme
  holding a number with `align: right`, `opacity: 0.5` and `thousands: true`, a number with `align: left`, and a bar
  of `cpu.util` with `background`: `theme compile` succeeds, the file read back with `themefile.Parse` gives the
  number's bytes 11, 14 and 15 as 2, 1 and 128, the second number's byte 11 as 0 and the bar's record with its box,
  tag and colours, the tags file holds the bar's metric; `theme preview` writes the preview of the compiled file,
  compared as in `TestThemePreviewIsThePreviewOfTheCompiledFile`; and `theme dump` of the compiled file writes a
  `theme.json` whose number holds `alignment` 2, `thousands_mode` 1 and `opacity` 128 and whose bar holds
  `empty_colour` and both `fill_or_frame_colour` names with the colours of the theme (R23)
- [x] Run `go test ./e2e/...`

Commit point 4.

### The documents

- [x] In `docs/theme-format.md` (R18):
    - add the mark **observed on the device** to the list of marks, for a fact seen on the screen on 2026-11-17
    - in "The tags seen", say that the tag of a bar is the tag of a metric, observed on the device, in place of the
      guess
    - in the table of `0x92`, give byte 11 as the alignment (0 left, 1 centred, 2 right), byte 14 as the thousands mode
      with the three results, its values 1 and 2 that looked the same, and the rule of the value divided by 1000 and
      cut to one decimal below 10000 and to a whole number from 10000 on as a guess that agrees with the three results
      and nothing more, and byte 15 as the opacity with 0 invisible and 255 opaque, all observed on the device on
      2026-11-17, and the values of byte 15 from 1 to 254, of byte 11 above 2 and of byte 14 above 2 and the
      difference of 1 and 2 as unknown. Below the table, say that the device clips the glyphs of a number to its box
      at the left and right, as the two results showed, that the minus sign takes a place in the box, and that a
      number that never received a value shows nothing
    - rewrite the section of `0x8B` with the common head, a table of bytes 11 to 63 (11 unknown, 1 in every sample and
      test; 12-13 the colour of the empty part; 14-15 and 16-17 the fill and the frame, which of the two not separated;
      18-19 and 26-27 `76b0`, no visible effect when changed; 20-25 no threshold colour when written; 28-63 unknown),
      the value as a share with the results for 0, 50, 100 and 150, the full bar in the colour of 14-17 with no value,
      the fill from the left edge, and the frame of one pixel inside the box, each marked observed on the device on
      2026-11-17
    - replace the open questions on byte 11 and byte 14 of a number, the bars, the opacity of a number and the clipping
      with those of R18: the unknown values of the number bytes, byte 11 of a bar, which of 14-15 and 16-17 is the
      fill, bytes 18 to 63, the values from 101 to 149 and below 0, a box taller than wide, the exact pixels of the
      edge of the fill, the clipping of a number at the top and bottom of its box, and the clipping of a datetime. Keep
      the opacity of a text and of a datetime open
    - in "What the dump of `sensordeck theme dump` shows", describe the preview of R13 to R17: `42`, or `1.2` for a
      nonzero byte 14, placed by byte 11, the centre rounded down, clipped to the box on four sides and to the screen,
      its alpha scaled by byte 15; a bar with its frame in 16-17, the fill of 42 in 14-15 from the left rounded down,
      and the empty part in 12-13; and name as the preview's own the rounding of the centre and of the fill, the fill
      from the left of a tall box, 14-15 as the fill and the clipping at the top and bottom. Name the JSON fields of
      R10 and R11
- [x] In `docs/theme-folder.md` (R19):
    - in the table of the widgets, add `align`, `opacity` and `thousands` to the number with their values and defaults,
      and the row of `bar` with `metric`, `width`, `height`, `color` and `background`, its defaults and its box
    - after the table, say that the screen cuts a value wider than the box, that the minus sign takes a place so that a
      number that shows negative values needs a `digits` one larger than its longest value, that `thousands` shows
      `1.2` for 1234 and `12` for 12345 with the point taking its own glyph width, and that `opacity` between 0 and 1 is
      written by the compiler but only 0 and 1 were seen on the device
    - say that a bar shows its metric's value as a share from 0 (empty) to 100 (full), that a value above 100 was seen
      to show it empty and one never sent to show it full
    - in "The tags", say that the number and bar widgets get the tags, and that a theme with neither has `tags: {}`
    - in "The box of a widget", add the box of a bar, its `width` and `height`
    - in the section on the bytes the compiler writes, give the number's bytes 11, 14 and 15 as R3 writes them, observed
      on the device, and the bar's bytes as R7 writes them, with 12-17 observed and 11, 18-19, 20-25 and 26-27 copied
      from the vendor sample, and remove the alignment from the closing sentence of what was not checked
- [x] In `README.md` (R20):
    - in "Writing a theme", give the number of the example `align: center` or `align: right` and add a bar such as
      `{type: bar, metric: cpu.util, x: 20, y: 130, width: 200, height: 12, color: "#00c8ff"}`, and add a paragraph on
      `align`, `opacity`, `thousands` and the bar, the value cut at the edge of its box and the minus sign that
      `digits` must count, linking to the folder document
    - add that a bar shows 0 to 100, that `cpu.util`, `gpu.util` and `ram.util` fit it as they are, and an example of a
      custom metric of a hwmon chip temperature with `scale: 0.0011111` (0.001 times 100 / 90), so that 90 °C fills
      the bar
    - say that the compiler gives each metric of the number and bar widgets a tag
    - in "Quick start", replace the sentence that what a tile never sent a value shows has not been checked with the
      fact that it shows nothing
    - in "Reading a theme file", replace the left alignment and the magenta outlines with the numbers aligned as their
      record says and clipped to their box, and the bars filled to 42
- [x] Read each changed section against R18 to R20 item by item, and check that no other part of the three documents
  still says that the alignment, byte 14, the opacity of a number, the bar or the clipping is unknown or a guess where
  this unit established it, with `grep -n "magenta\|left-aligned\|_guess\|not been checked\|bytes_11_63" README.md
  docs/*.md`

Commit point 5.

## Verification

- [x] `go test ./...` passes
- [x] `go vet ./...` reports nothing
- [x] `gofmt -l .` prints no file name
- [x] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [x] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [x] `npm run check-changes` passes on the committed branch
- [x] Every requirement has a task: R1 to R3 and R5 in the compiler's document tasks, R4, R6 and R7 in its record
  tasks, R8 and R9 in the tags tasks and the test of `run`, R10 to R12 in the reader and dump tasks, R13 to R17 in the
  preview tasks, R18 to R20 in the documents, R21 in the drawing of the images, R22 across the unit tests, R23 in the
  end-to-end tasks and R24 here

## Commit Points

- [x] 1: `feature-0027 Name the bytes of a number and of a bar in the reader and the dump`
- [x] 2: `feature-0027 Compile the number options and the bar widget`
- [x] 3: `feature-0027 Align, clip and fade numbers and draw bars in the preview`
- [x] 4: `feature-0027 Cover the bar and the number options end to end`
- [x] 5: `feature-0027 Document the number options, the bar and the observations of the device`
