# The options of a number widget, the bar widget and a preview that clips

## Context

Unit `0010-theme-file-reader` decoded the record of a number with three guesses, bytes 11, 14 and 15, and left the
record of a bar undecoded from byte 11 on. Unit `0011-theme-compiler` writes every number with byte 11 at 1, byte 14 at
0 and byte 15 at `0xff`, the values of the vendor samples, and compiles no bar. The preview of `theme dump` and
`theme preview` draws every number left-aligned from the left edge of its box, clips glyphs to the screen only, and
draws a bar as a magenta outline.

On 2026-11-17 the human tried these bytes on the screen, with test themes patched by hand and values sent with
`sensordeck values`. This unit makes what was seen usable in `theme.yaml`, makes the preview draw it as the screen does,
and corrects the documents. The test themes and their scripts are kept outside the repository, on the host, and the
sandbox the unit is built in does not have them, so this spec carries every fact the unit relies on.

"R*n* of unit 00*nn*" below names a requirement of the spec of that unit in `changes/`. "The format document" is
`docs/theme-format.md` and "the folder document" is `docs/theme-folder.md`.

### What was observed on the device

Of the number record (type `0x92`):

- **Byte 11 is the alignment of the value in its box**: 0 is left, 1 is centred, 2 is right. Three numbers of the same
  box and value side by side showed it: the one with 0 sat further left than the one with 1, and the one with 2 further
  right. The compiler writes 1 today, so every compiled theme shows its numbers centred on the screen
- **Byte 15 is the opacity**: 0 made the number invisible, and 255, which the compiler writes today, opaque. Values
  between were not tried
- **Byte 14, when it is not 0, is a thousands mode**: with 1, the value 1234 showed `1.2` and 12345 showed `12`; with 2,
  1234 showed `1.2` and 999 showed `0.9`. The values 1 and 2 looked the same in these tests. The exact rule is not
  established. All three results agree with the value divided by 1000 and cut, not rounded, to one decimal below 10000
  and to a whole number from 10000 on, but nothing else was tried: no negative value, no value from 10000 to 12344, no
  value below 999, and no other value of the byte
- **Glyphs are clipped to the box of the widget**: -32768 in a box of 5 digits showed `-3276` with the 8 cut at the
  right edge of the box, and 40000, which the screen reads as -25536 (unit 0026), in a right-aligned box showed
  `25536` with its minus sign cut at the left edge. The minus sign takes a place in the box like a digit. Only the
  clipping at the left and right edges was seen
- **A number that never received a value shows nothing**

Of the bar record (type `0x8B`):

- The common head is that of every record: type, index, tag, x, y, width, height. From byte 11 on, the bar of the
  vendor sample holds `01 00 00 a8 00 a8 00 76 b0 00 00 00 00 00 00 76 b0` followed by zeros. The test themes kept this
  and changed the bytes named below one at a time
- **The value is a share from 0 to 100**: 50 filled half the bar, 100 filled it, 0 showed it empty, and 150 showed it
  empty. Values from 101 to 149 and negative values were not tried. The values were sent to the tag of the bar, so the
  tag of a bar is the tag of a metric as for a number
- **With no value received**, the bar showed full, in the colour of bytes 14 to 17
- **Bytes 12-13** (`0000`, black) are the colour of the empty part
- **Bytes 14-15 and 16-17** (both `a800`, dark red) are the colours of the fill and the frame. Writing `07e0` into both
  made the fill and the frame green. Which of the two is the fill and which the frame was not separated
- **The frame** is drawn inside the box of the bar, seen as an outline one pixel wide
- **Bytes 18-19 and 26-27** (`76b0`): writing `001f` into both had no visible effect. Writing 50 into bytes 20-21, 22-23
  or 24-25 gave no threshold colour. Byte 11 was 1 in every test
- How the fill grows in a box taller than it is wide, and where exactly its edge falls in pixels, was not reported

Values are signed 16-bit numbers, as unit 0026 established.

### The requirements this unit replaces

- Unit `0010-theme-file-reader`: in R3, that a bar holds bytes 11 to 63 as one unknown range; in R12, that a number is
  drawn as `42` left-aligned whatever byte 11 holds, that a bar is drawn as a magenta outline, and that glyphs are
  clipped to the screen only; in R13, the JSON names `alignment_guess`, `decimals_guess` and `opacity_guess` of a
  number and `bytes_11_63_unknown` of a bar; in R20 and R21, the statements and open questions on the alignment, byte 14,
  the opacity of a number, the bars and the clipping; its D6 and D8 on the left alignment and the outline of a bar
- Unit `0011-theme-compiler`: in R5, the keys of a number widget and the list of widget types; in R14, the bytes 11, 14
  and 15 of a number; in R18, that only number widgets get tags; its non-goals on progress bars and the alignment of a
  number; in R27 and R28, the parts of the documents that this unit changes
- Unit `0012-built-in-theme-and-theme-names`: in R19, the statement of the README that what a tile shows before it
  was ever sent a value has not been checked on the device

## Requirements

"The preview" is the image of R12 of unit 0010, which `theme dump` writes as `preview.png` and `theme preview` writes
for a theme. "The value drawn" of a number is the text of characters the preview draws for it, and its width is the sum
of the widths of the glyphs of its characters.

### The number widget in `theme.yaml`

- **R1** A number widget takes three more optional keys:
    - `align`: `left`, `center` or `right`, and `center` without the key
    - `opacity`: a number from 0 to 1, integer or decimal, and 1 without the key
    - `thousands`: `true` or `false`, and `false` without the key
- **R2** A value of these keys that is not as R1 says is an error of the form of R8 of unit 0011 that names the widget
  and the key: `align: want left, center or right`, `opacity: want a number from 0 to 1` and
  `thousands: want true or false`. A string such as `"0.5"` or `50%` is not a number, and `yes` or `1` is not a boolean
- **R3** The compiler writes into the record of a number byte 11 as 0 for `left`, 1 for `center` and 2 for `right`;
  byte 14 as 1 with `thousands: true` and 0 otherwise; and byte 15 as the opacity times 255 rounded to the nearest whole
  number, halves up, so that 1 gives 255, 0 gives 0 and 0.5 gives 128. Every other byte of the record is as unit 0011
  writes it, so a number widget without the three keys gives the same 64 bytes as today
- **R4** The box of a number does not change: `digits` digit widths wide and the glyph height high, at its x and y

### The bar widget in `theme.yaml`

- **R5** A widget of `type: bar` has the keys `x` and `y` of every widget and:
    - `metric`, required: a metric name, checked as the metric of a number is (R7 of unit 0011)
    - `width` and `height`, required: whole numbers from 1, the size of the box of the bar
    - `color`, required: a colour of R2 of unit 0011, the colour of the fill and of the frame
    - `background`, optional: a colour, the colour of the empty part, `"#000000"` without the key

  Any other key, such as `font`, is an error that names it, as R1 of unit 0011 says of every key not listed. The error
  for an unknown widget type names `bar` among the types
- **R6** The box of a bar is at its x and y, of its width and height, and must be within the screen, as R9 of unit 0011
  says of every box
- **R7** The record of a bar is a record of type `0x8B` in the slot of its widget, with the index and the place in the
  order of the slots of every widget (R12 and R13 of unit 0011), the tag of its metric, its box, and from byte 11 on:
  byte 11 1; bytes 12-13 the RGB565 colour of `background`; bytes 14-15 and 16-17 both the RGB565 colour of `color`;
  bytes 18-19 `76b0`; bytes 20 to 25 zero; bytes 26-27 `76b0`; bytes 28 to 63 zero. Colours are made RGB565 as R15 of
  unit 0011 says. A bar has no resource, so it adds no byte after the record table and moves no offset of the records
  after it
- **R8** The metrics of the bars take part in the tags as those of the numbers do (R18 of unit 0011): the tags are given
  to the distinct metric names of the number and bar widgets together, and a bar and a number of the same metric have
  the same tag. The tags file of R20 of unit 0011, and the tags that `run` takes from a compiled theme, hold the metrics
  of the bars, so that `run` sends their values, and refuses a theme whose bar names a metric that is neither built in
  nor defined under `metrics`, as it does for a number
- **R9** The value sent to a bar is the value of its metric, as for a number. Nothing scales it

### The reader and the dump

- **R10** The typed value of a number names its bytes 11, 14 and 15 as the alignment, the thousands mode and the
  opacity, and in `theme.json` they are `alignment`, `thousands_mode` and `opacity`, each with the value of its byte as
  a number, in place of `alignment_guess`, `decimals_guess` and `opacity_guess`. The other fields of a number do not
  change
- **R11** The typed value of a bar holds byte 11 as an unknown byte, bytes 12-13 as the colour of the empty part, bytes
  14-15 and 16-17 as two colours of which one is the fill and the other the frame, and bytes 18 to 63 as one unknown
  range, so that every one of its 64 bytes is in exactly one field (R3 of unit 0010). In `theme.json` they are
  `byte_11_unknown`, `empty_colour`, `fill_or_frame_colour_14`, `fill_or_frame_colour_16` and `bytes_18_63_unknown`,
  the colours in the form of R13 of unit 0010, in place of `bytes_11_63_unknown`
- **R12** The reader gives no new error and no new warning: every value of the bytes of R10 and R11 is read

### The preview

- **R13** A number is drawn as its value drawn, which is `42`, or `1.2` when byte 14 is not 0. Its glyphs are placed
  side by side, each the next at the right edge of the one before, with their tops at the box's y. The left edge of the
  first is at:
    - the box's x when byte 11 is 0
    - the box's x plus the box's width minus the width of the value drawn when byte 11 is 2
    - the box's x plus half of the difference of the box's width and the width of the value drawn, rounded down,
      towards minus infinity when the value is wider than the box, when byte 11 is 1
    - the box's x, as for 0, when byte 11 is any other value
- **R14** The glyphs of a number are clipped to its box, on all four sides, and to the screen. A value wider than its
  box is cut at the edge of the box, so that no pixel of a number is drawn outside its box
- **R15** The glyphs of a number are drawn with the alpha of each pixel of the mask multiplied by byte 15 and divided by
  255, rounded to the nearest whole number. So byte 15 at 0 draws nothing and at 255 draws the mask as it is
- **R16** A bar is drawn with the sample value 42, inside its box:
    - its frame, the one-pixel inside edge of the box, in the colour of bytes 16-17
    - inside the frame, the leftmost columns, as many as the width inside the frame times 42 divided by 100 rounded
      down, in the colour of bytes 14-15, and the other columns in the colour of bytes 12-13
    - each colour opaque, expanded as R11 of unit 0010 says. A box of width or height 2 or less has no inside, and
      shows its frame alone
- **R17** Every other record is drawn as R12 of unit 0010 says, a datetime included, whose glyphs stay clipped to the
  screen only. The PNG files of the resources (R11 and R14 of unit 0010) do not change: a glyph strip shows the masks
  in the colour, with no opacity applied

### The documents

- **R18** The format document holds every fact of "What was observed on the device" above, each marked
  **observed on the device** with the date 2026-11-17, beside the marks known, guess and unknown it uses today:
    - the table of the number names bytes 11, 14 and 15 with their observed meaning and values, and says that the
      values of byte 15 from 1 to 254, the values of byte 11 above 2 and of byte 14 above 2, and the difference between
      1 and 2 of byte 14 are unknown. It gives the rule of the thousands mode that agrees with the three results as a
      guess, and says that nothing beyond these three results is established
    - the section of the bar gives the common head, a table of bytes 11 to 63 with each field and its status, the range
      of the value with the results for 0, 50, 100 and 150 and the bar with no value, the frame inside the box, the
      bytes whose change had no visible effect, and that the tag of a bar is the tag of a metric
    - it says that the device clips the glyphs of a number to its box, that the minus sign takes a place in the box,
      and that a number that never received a value shows nothing
    - its open questions list what stays unknown: the values of the number bytes named above; byte 11 of a bar, which
      of bytes 14-15 and 16-17 is the fill, bytes 18 to 63, the values from 101 to 149 and below 0, how a bar fills a
      box taller than wide, and the exact pixels of the edge of the fill; whether the device clips a number at the top
      and the bottom of its box, and whether it clips the glyphs of a datetime. It no longer lists as open what this
      unit established
    - the section on the dump says how the preview draws numbers and bars, as R13 to R17 say, which of its choices
      are its own (the rounding of the centre and of the fill, the fill from the left, the colour of bytes 14-15 as the
      fill, the clipping at the top and bottom), and the new JSON names of R10 and R11
- **R19** The folder document:
    - describes `align`, `opacity` and `thousands` in the table of the widgets, with their values and defaults, and
      the bar widget with its keys, defaults and box
    - says that the screen cuts a value that is wider than the box of its number, that the minus sign of a negative
      value takes a place in the box, so that a number that shows negative values needs a `digits` one larger than
      its longest value, and that with `thousands` the value is shown in thousands, `1.2` for 1234 and `12` for 12345,
      with the point taking its own glyph width
    - says that a bar shows the value of its metric as a share from 0 (empty) to 100 (full), that a value above 100
      was seen to show the bar empty, that a bar that never received a value shows full, and that the tags include the
      metrics of the bars
    - in the section on the bytes the compiler writes, gives bytes 11, 14 and 15 of a number as R3 writes them and the
      bytes of a bar as R7 writes them, and says which of them were observed on the device and which are copied from the
      vendor sample
    - no longer says that the alignment of a number has not been checked on the device
- **R20** The README:
    - in "Writing a theme", the example `theme.yaml` holds a bar and a number with `align`, and a paragraph says what
      `align`, `opacity`, `thousands` and the bar do, that a value wider than its box is cut, that the minus sign takes
      a place in the box so that `digits` must count it, and links to the folder document for the rest
    - says that a bar shows 0 to 100: that `cpu.util`, `gpu.util` and `ram.util` fit it as they are, and that a metric
      of another range is shown through a custom metric whose `scale` makes it 0 to 100, with an example of a
      temperature from a chip of hwmon in which 90 °C fills the bar
    - in "Quick start", says that a tile that was never sent a value shows nothing, in place of the statement that this
      has not been checked
    - in "Reading a theme file", says that the preview draws numbers as their record aligns them, clipped to their
      box, and bars filled to 42, in place of the left alignment and the magenta outlines
- **R21** The committed preview images of the built-in themes, `docs/minimal-dark.png` and
  `docs/minimal-dark-probes.png`, are those that `sensordeck theme preview` draws after this unit, so that the test of
  `docs_test.go` passes. Their numbers are now centred in their boxes, as the screen shows them

### Tests

- **R22** The unit tests cover at least:
    - R1 to R3: a number with each value of `align`, with `opacity` 0, 0.5 and 1, and with `thousands` true and false,
      with every byte of its slot checked; a number without the keys giving the same slot as before; each error of R2
    - R5 to R8: a bar compiled with every byte of its slot checked, with and without `background`; each missing
      required key, a width of 0, a box outside the screen, an unknown key and the error naming `bar` among the types;
      a bar and a number of the same metric sharing a tag, a bar alone giving its metric a tag, and the tags file
    - the round trip of R26 of unit 0011, extended with a bar and a number with each of the three keys, found again
      field by field
    - R10 and R11: a number and a bar read from slots whose bytes are all distinct where the layout allows, and the
      JSON names of the dump
    - R13 to R16 at chosen pixels: the first column of the value of each of the three alignments and of a byte 11 of
      another value; a value wider than its box cut at the left and at the right edge, with the pixel just outside the
      box left as it was; a box that crosses the edge of the screen; byte 15 at 0, 128 and 255; `1.2` drawn for a
      nonzero byte 14; a bar's frame, its last column of fill, its first column of the empty part, and a box too small
      to have an inside
- **R23** The end-to-end tests compile and preview a theme with a bar and a number with each of the new keys, and
  check the theme file read back and the preview against the preview of the compiled file, and check the new JSON
  names of a number and a bar in the output of `theme dump`
- **R24** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 The alignment keeps centre as its default.** Every theme compiled so far writes 1, and the screen shows its
  numbers centred, so a theme without the key looks on the screen as it did. Only its preview changes, and it now
  matches the screen. Rejected: `left` as the default, which matches the old preview but moves every number of every
  existing theme on the screen, the built-in ones included
- **D2 `center`, spelled as in `color`.** The keys of `theme.yaml` follow the spelling of CSS, as `color` does, and a
  theme author writes `center` from habit. The prose of the documents keeps the British spelling. Rejected: `centre`,
  which would mix spellings within one file. Accepting both is item 3 of "Beyond the request"
- **D3 Opacity as a fraction from 0 to 1.** A fraction is how CSS and drawing tools write opacity, and it does not
  expose the byte. 0 and 1 give the two values that were seen, 0 and 255. The values between were not tried on the
  device; the compiler writes them and the preview blends linearly, and the documents say that this is a guess.
  Rejected: a whole number from 0 to 255, the byte itself, which asks the author for the format's unit; a percentage,
  which YAML reads as a string
- **D4 `thousands: true` and not `scale: thousands`.** A custom metric already has a key `scale` that is a number the
  reading is multiplied by, and the same word with another meaning in the theme would mislead. Only one mode is known:
  the values 1 and 2 of the byte looked the same, so a boolean says all that is known, and the compiler writes 1, the
  first value tried. Rejected: an enumeration of modes, a guess about a second mode that was not seen
- **D5 The preview draws `1.2` for the thousands mode.** The preview shows one sample value for every number, and 42
  in thousands would need a rule for values below 999, which was not seen. 1234 showing `1.2` was seen on the device,
  so drawing `1.2` for any nonzero byte 14 asks no guess of the preview. The guessed rule stays in the format document,
  marked as a guess. Rejected: applying the guessed rule to 42, which draws `0.0`, a picture of a guess that also shows
  nothing useful to the author
- **D6 Clipping to the whole box.** The screen was seen to cut glyphs at the left and right edges of the box. The
  preview also clips at the top and the bottom, because a screen that clips to a box most likely clips to all of it, and
  a compiled number's box is exactly the glyph height, so no compiled theme is affected. A vendor number whose box is
  less high than its glyphs is cut in the preview, and the format document says that this side was not observed.
  Rejected: clipping at the left and right only, a rule shaped by what the experiment happened to test rather than by
  what the screen most likely does. Datetimes stay clipped to the screen only, because their clipping was not seen, and
  a compiled datetime's box is exactly its format, so the question matters only for vendor themes
- **D7 The centre rounds down.** The pixel at which the screen places a centred value whose margin is odd was not
  measured. Rounding down is a choice of the preview, at most one pixel off, and the format document says so
- **D8 Bars show their metric's value as it is, from 0 to 100.** The bar takes a share, and `cpu.util`, `gpu.util` and
  `ram.util` are shares already. Another range is made a share with a custom metric, whose `scale` exists today: a
  temperature read from its chip in millidegrees with `scale` 0.001 times 100 / 90 fills the bar at 90 °C. Rejected for
  this unit: a key `max:` on the bar with `run` scaling the value, which puts the scaling into the mapping of tags. A
  bar of `cpu.temp` with `max: 90` and a number of `cpu.temp` would need two tags for one metric with two values, so the
  tags would map a metric and a scale rather than a metric, which changes the compiler's tags, the tags file and `run`.
  It is item 1 of "Beyond the request"
- **D9 One colour for the fill and the frame.** Which of bytes 14-15 and 16-17 is the fill was not separated, so the
  theme gives one colour and the compiler writes it into both, which shows that colour whichever is which. A key for
  the frame alone waits for the experiment that separates them. The preview draws bytes 14-15 as the fill and 16-17 as
  the frame, a choice that changes nothing for a compiled bar and that the format document names as a guess
- **D10 The other bytes of a bar are the vendor sample's.** Byte 11, bytes 18-19 and bytes 26-27 are the values of
  every bar of the vendor sample, which the device draws, and were kept in every test on the device. Bytes 20 to 25 are
  zero, as in the sample and in the tests in which no threshold colour appeared. Threshold colours are the next unit
- **D11 The sample value of a bar is 42.** It is the sample value of every number, so a preview with a number and a bar
  of the same metric shows them agree, and 42 is neither empty nor full, so the fill, the empty part and their border
  are all visible
- **D12 The fill grows from the left.** The direction was not reported. A bar from the left is what a share of a
  horizontal box looks like, and the format document says that the direction in a tall box is unknown
- **D13 The names of the dump follow what is known.** Bytes 11, 14 and 15 of a number get plain names, as D5 of unit
  0010 says a confirmed guess does. Byte 14 is `thousands_mode` with the byte's value, not a boolean, so that a dump
  shows the 2 of a vendor theme as it is. The two colours of a bar are named by their first byte because their roles are
  known only as a pair
- **D14 The threat model is unit 0011's.** A theme folder may come from anyone, and D15 of unit 0011 holds. The new keys
  are scalars checked as R2 says, a bar has no file and no resource, and its box is checked against the screen as every
  box is
- **D15 No device check is waited for.** That the new options look on the screen as the preview draws them is the
  human's to check, and the unit is complete without it. A finding of that check is a new unit

## Non-goals

- Threshold colours of a bar and anything else of bytes 18 to 63 of a bar: the next unit
- A colour for the frame of a bar apart from its fill, and a vertical bar or a bar that fills from another side
- A key `max:` or any scaling of the value of a bar in `run` (D8)
- The opacity of text and datetime widgets, and the opacity guesses of their records, which the preview still does not
  apply
- Changing the layout of the built-in themes: their numbers were already centred on the screen, and only their preview
  images change (R21)
- Raising the limit of 5 for `digits`
- Clipping datetimes to their box
- The other values of byte 14 of a number, and a key for them
- The check on the device, which is the human's

## Beyond the request

1. A key `max:` on a bar, with which `run` sends the value of its metric as a share of `max` (for example a temperature
   in a bar full at 90 °C), so that no custom metric is needed. Cost: the tags become pairs of a metric and a scale,
   with a tag of their own for each distinct pair, and that changes the compiler's tags, the tags file and its use in
   the config file, `run` and their tests. Without it, a bar of another range needs a custom metric with a `scale`, as
   the README shows, which works for the metrics read from a file, a chip or a command
2. A warning of the compiler when a bar names a built-in metric whose unit is not %, such as `cpu.temp`. Cost: a table
   of the units of the built-in metrics in the compiler, a warning channel it does not have today, and a test. Without
   it, such a bar shows full or empty on the screen and in the preview at 42, which the author sees
3. Accepting `centre` beside `center`. Cost: one more value and a test. Without it, `centre` gives the clear error of
   R2, which names the three values
4. Raising the limit of `digits` from 5 to 6, so that -32768 fits in a box with its minus sign, as item 2 of "Beyond
   the request" of unit 0026 proposed. Cost: a change of the compiler, its error text, a test and the folder document.
   Without it, a negative value of five digits has its minus sign or its last digit cut, which only a custom metric
   below -9999 reaches
