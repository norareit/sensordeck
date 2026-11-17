# The screen draws no widget whose data ends past 503 808 bytes

## Context

Unit `0030-probes-theme-alarm-colours` gave the five temperatures of the built-in theme `minimal-dark-probes` an alarm
colour. Each alarm number is two number records with a glyph set each (unit `0028-number-alarm-colour`), so the theme
grew from 436 694 bytes, compiled from its `theme.yaml` before unit 0030, to 508 370 bytes. Uploaded to the screen, it
shows no date and no time. The compiler accepted it, because it checks only that a theme has at most 62 widgets and
that the file fits in the 4 MiB of the flash of the device, and neither is what the screen limits.

"R*n* of unit 00*nn*" below names a requirement of the spec of that unit in `changes/`. "The format document" is
`docs/theme-format.md`, "the folder document" is `docs/theme-folder.md` and "the protocol document" is
`docs/protocol.md`. "The limit" is 503 808 bytes, `0x7B000`, 492 KiB. "The data of a widget" is its resource in the
theme file: the bytes from its offset to the offset of the next resource, or to the end of the file for the last one.

### What was observed on the device

The human watched the screen of the project on 2026-11-17 from 18:50 to 19:20 UTC while themes were uploaded to it.
The themes were made on the host from `minimal-dark-probes` with the widgets moved or removed, and are not in the
repository, so this spec carries every fact the unit relies on.

- **The theme of unit 0030**: 508 370 bytes. The screen showed every widget but the date and the time, which are its
  last two widgets and its last two resources. Every other part of the upload succeeded
- **The time and the date moved** to the slots 40 and 41, 41 and 42, 42 and 43, and 43 and 44, which moved the end of
  their data. The time was drawn when its data ended at 503 369 and at 503 692 bytes, and was not drawn when it ended
  at 504 015. The date, the last resource of each of these themes, ended past 505 000 bytes and was never drawn
- **A number in slot 42** whose data ended at 496 566 bytes was drawn, so the number of slots is not what limits the
  screen
- **A prediction confirmed**: a copy of the theme without the alarm of INTAKE, 505 094 bytes, put the end of the data
  of the time at 501 062 bytes and that of the date at 505 094. The screen showed the time and not the date, as
  predicted
- **Before unit 0030** the theme showed the date and the time

So the screen draws a widget only when its data ends within the first 503 692 to 504 015 bytes of the theme file. The
one multiple of 4096 in that range is 503 808, `0x7B000`, which this unit takes as the limit. The exact limit within
the range, and what in the device sets it, are not known. Everything above was seen with themes laid out as the
compiler of sensordeck lays them out: the record table, the background of 307 200 bytes, then the data of the widgets
in the order of their slots. The vendor's tool-made sample, 524 070 bytes and laid out otherwise (its background has no
data and a full-screen image of 460 800 bytes follows it), was not checked against this.

### The bytes of the two built-in themes today

Measured on the host with the installed `sensordeck` from the head of the default branch, and read back with
`sensordeck theme dump`:

| Theme | File | Table | Background | Numbers | Texts | Datetimes | Records beside the background |
|---|---|---|---|---|---|---|---|
| `minimal-dark-probes` | 508 370 | 4 096 | 307 200 | 17, 169 728 | 24, 15 542 | 2, 11 804 | 43 |
| `minimal-dark` | 431 788 | 4 096 | 307 200 | 11, 95 616 | 21, 13 072 | 2, 11 804 | 34 |

A number of the font `big` (`go-bold` at 46) takes 16 200 bytes, and 18 954 as one of the two records of an alarm
number, whose minus is as wide as the box; one of the font `small` (`go-medium` at 17) 2 436, and 2 856 in an alarm
number. The time takes 7 772 bytes and the date 4 032. No text overlaps a widget before it in either theme.

### The requirements this unit replaces

- Unit `0011-theme-compiler`: in R9, the error for more than 62 widgets and the flash limit as the size check of a
  theme; in R12 and R13, one record for each widget; in R15 and R16, the resource and the box of a text record, for
  the texts this unit draws into the background
- Unit `0028-number-alarm-colour`: R11, the count of 62 widgets with an alarm number counting as two

## Observed behavior

A theme whose compiled file is larger than the limit compiles with no error, uploads with no error, and the screen
shows it without each widget whose data ends past the limit. The built-in theme `minimal-dark-probes` is such a theme
since unit 0030, and the screen shows it with no date and no time.

## Expected behavior

The compiler refuses a theme the screen cannot draw in full, with a message that says by how much it is too large and
what costs bytes. The texts of a theme, which never change, cost no bytes of their own and no slot, because the
compiler draws them into the background, which takes 307 200 bytes whatever it shows. The built-in theme
`minimal-dark-probes` with its five alarms fits, and the screen shows its date and its time.

## Steps to reproduce

1. On the head of the default branch, `sensordeck theme compile minimal-dark-probes img.dat` writes a file of
   508 370 bytes and no error
2. Upload it to the screen, with `sensordeck upload img.dat` or with `run` and the key `theme: minimal-dark-probes`
3. The screen shows every widget but the date and the time

## Root cause

The screen draws only the widgets whose data ends within the first 503 808 bytes of the theme file, as observed above,
and the compiler does not know it. It checks the size of the file only against the 4 MiB of the flash (R9 of unit
0011), and the number of widgets against the 62 slots of the record table, neither of which the screen reaches with
the themes of the project. It also writes each text as a record with an alpha mask of its own after the background,
so every text costs bytes and a slot although the background, which is a full picture in every compiled theme, could
hold it at no cost. Unit 0030 added five glyph sets and widened five others, which moved the data of the date and the
time, the last two widgets of the theme, past the limit. Why the device has the limit is not known.

## Fix

- The compiler draws each text widget into the background picture, as the preview draws a text record, instead of
  writing it as a text record, unless the text overlaps a widget before it that is still a record, so that the order
  of drawing and so the picture do not change
- The compiler refuses a theme whose file is larger than the limit, which with its layout is a theme with a widget
  whose data ends past the limit, in every command that compiles
- The check of 62 widgets becomes a check of the records written, since a text drawn into the background takes no
  slot
- The documents state the limit, the evidence and what costs bytes, and the preview images of the built-in themes are
  written again

## Requirements

### Texts drawn into the background

- **R1** A text widget is "drawn into the background" when its box (R16 of unit 0011) overlaps the box of no widget
  before it in the list that is written as a record. Two boxes overlap when they share at least one pixel. The widgets
  are decided in the order of the list, so a text kept as a record (R3) is a record for the texts after it. An image,
  a number, both records of an alarm number, a datetime and a bar are always records. A text drawn into the background
  is checked as today, its glyphs (R9 of unit 0011) and its box against the screen, with the same errors, which name
  the widget by its place in the list as today
- **R2** The background resource of a theme with texts drawn into it is made so:
    - the picture of today, a solid colour or the PNG composited over opaque black, in the 8-bit channels that
      `themeimage.Expand` gives each pixel of the RGB565 plane of today
    - each text drawn into the background drawn over it in the order of the list, as `themeimage` draws a text record:
      its mask, made as today, in the colour of its font as `Expand` gives it, at the x and y of its box, with
      Porter-Duff "over"
    - each pixel then made RGB565 by the rounding of R15 of unit 0011

  The background record itself is as today, its bytes 11 to 63 included: for a solid background the colour of bytes
  12-13 is still the solid colour. A theme with no text drawn into the background has a background resource byte for
  byte as today
- **R3** A text that overlaps a widget before it that is a record is written as a text record, exactly as today, in
  the slot that its place among the records gives it
- **R4** The records are written in the order of the list, the texts drawn into the background left out: slot 1 the
  background, then slot 2 and up the records, with their resources in the order of the slots, back to back, as R12 of
  unit 0011 says. The index byte of each record is its slot number minus 1 (R13 of unit 0011). An alarm number keeps
  its two records in consecutive slots, its alarm widget after its normal widget (R7 of unit 0028), and
  `Result.AlarmSlots` names the slots the records take now
- **R5** The tags (R13 of unit 0028) and the tags file are as today for every theme, since texts have no tag
- **R6** The preview that `sensordeck theme preview` draws of a theme differs from the preview of the same folder before
  this unit only at the pixels where the mask of a text drawn into the background has an alpha from 1 to 254, and
  there by at most 4 in red and in blue and 2 in green. Every other pixel is the same. (The preview before this unit
  blended the text over the background in 8 bits; the background now holds that blend rounded to RGB565, which
  `Expand` gives back within those bounds.) For a theme with no text drawn into the background the preview is the same
  at every pixel
- **R7** `sensordeck theme dump` of a compiled theme shows the texts drawn into the background in the PNG of the
  background and in the preview, and lists no record for them. Nothing changes in how `theme dump` reads a theme file:
  a text record of a vendor theme, or one kept by R3, is read, described and drawn as today

### The limits

- **R8** A theme whose compiled file is larger than 503 808 bytes is refused with the error
  `widgets: the compiled theme is <size> bytes, <over> more than the 503808 bytes that the screen draws widgets from; the background takes 307200 of them, each number a glyph set of its font and an alarm number two, each datetime a glyph set and each image 3 bytes a pixel, and a text none`,
  where `<size>` is the size of the file and `<over>` the size minus 503 808, both in decimal digits with no
  separator. A file of exactly 503 808 bytes is accepted. This error replaces the error of the flash limit, which a
  file of at most 503 808 bytes cannot reach. As every error of the compiler, it is given by `theme compile`, `theme
  preview` and `theme use`, prefixed as they prefix it today, and `run` refuses such a theme at its start and at a
  reload as it refuses any theme that does not compile. No file is written, no config file changed and nothing
  uploaded for it
- **R9** A theme with more than 62 records beside the background is refused with the error
  `widgets: <n> records, more than the 62 that fit beside the background; an alarm number is two records and a text drawn into the background none`,
  where `<n>` is the number of records the theme would have beside the background. The list itself may be of any
  length. The check of R11 of unit 0028, made before any widget is read, and its two forms of the error are removed
- **R10** When a theme has faults of R9 and R8, the error of R9 is given. A folder with several faults gives the same
  error each time, as today

### The built-in themes

- **R11** The theme folders of `minimal-dark` and `minimal-dark-probes` do not change: their `theme.yaml` and their
  `background.png` are byte for byte as today
- **R12** Every text of the two built-in themes is drawn into the background. `minimal-dark-probes` compiles to
  508 370 - 15 542 = 492 828 bytes, 10 980 under the limit, with 19 records beside the background, and `minimal-dark`
  to 431 788 - 13 072 = 418 716 bytes, with 13. Their tags files are byte for byte as today
- **R13** `docs/minimal-dark.png` and `docs/minimal-dark-probes.png` are the previews that `sensordeck theme preview`
  draws after this unit, and each differs from the committed image of today only as R6 allows
- **R14** On the screen of the human, `minimal-dark-probes` uploaded after this unit shows the date, the time, every
  label and the five temperatures with their alarms, as its preview shows it. This is the human's to check, as
  `AGENTS.md` says of every check that needs the device

### The documents

- **R15** The format document, in its section on the layout, records the limit as **observed on the device** on
  2026-11-17: every fact of "What was observed on the device" above, the range of 503 692 to 504 015 bytes, why 503 808
  is taken as the limit and that this choice is a guess within the range, that what sets the limit is unknown, and that
  the vendor's tool-made sample was not checked against it. The sentence on the 4 MB of the flash stays, with the limit
  beside it. Its open questions gain the exact limit, what the device limits (the end of the data in the file, or a
  budget of another shape that the layout of the compiler cannot tell apart), and whether the vendor samples are drawn
  in full
- **R16** The protocol document, in its section on the theme upload, says that an upload of a larger file succeeds and
  the screen then draws no widget whose data ends past the limit, with a link to the format document
- **R17** The folder document:
    - in the limits, replaces the flash limit of 4 MiB with the limit of R8 and its error, says that with the record
      table and the background a theme has 192 512 bytes for the data of its widgets, and replaces the 62 widgets with
      the 62 records of R9
    - says what costs bytes: a number its glyph set, the font's glyph height times ten digit widths and the widths of
      `.` and `-`; an alarm number two such sets, each with a minus as wide as the box; a datetime its glyph set; an
      image 3 bytes a pixel; a bar nothing; a text drawn into the background nothing. It gives the sizes of a number of
      each font of `minimal-dark-probes`, of its time and of its date, as an example, and says that `theme dump`
      shows the size of each resource
    - says that a text is drawn into the background, the rule of R1 and why, and that a text kept as a record is
      written as the section on the bytes of the compiler says today
    - in the section on the widgets, replaces "at most 62 of them" with the records of R9
- **R18** The README:
    - in "Writing a theme", says that the screen draws a theme of at most 503 808 bytes in full, that the compiler
      refuses a larger one and why, that the background takes 307 200 bytes of it, what each widget costs as R17 says
      in one sentence, and that texts cost nothing, with a link to the folder document
    - in the paragraph on the key `alarm`, replaces "it counts as two of the 62 widgets" with a sentence that says it
      is two records and two glyph sets
- **R19** No other document changes its meaning. The paragraphs that name the 4 MiB of the flash as a fact of the
  device stay

### Tests

- **R20** The unit tests cover at least:
    - R1 and R3: a text drawn into the background; a text that overlaps an earlier number, an earlier image and an
      earlier text kept as a record, each kept as a record in the right slot; a text that overlaps only an earlier text
      drawn into the background, drawn into it; a text that touches the box of an earlier record at its edge without
      sharing a pixel, drawn into the background; the errors of a text's glyphs and box unchanged
    - R2: the background resource of a solid and of a PNG background with a text, pixel for pixel against the rule;
      two overlapping texts drawn in the order of the list; a theme with no text with the background of today
    - R4: the slots, indexes and offsets of a theme with texts between an alarm number and other widgets, and its
      `AlarmSlots`
    - R6: the preview of a theme with texts drawn into the background against the preview of the same theme with every
      text written as a record, within the bounds of R6 and equal everywhere else
    - R8: a theme over the limit with the error text exact; the decision at exactly 503 808 bytes and at 503 809
    - R9 and R10: 62 records accepted; 63 refused with the error; a list of more than 62 widgets that compiles because
      texts are drawn into the background; a theme over both limits giving the error of R9
- **R21** The tests of the built-in themes check R11 and R12: the folders unchanged, the size of each compiled theme,
  its records and its tags file. The test of the preview images in `docs_test.go` passes with the images of R13
- **R22** The end-to-end tests:
    - `theme compile` of a folder over the limit fails with the error of R8, prefixed with the path of its
      `theme.yaml`, and writes neither the theme file nor the tags file
    - `theme use` of that folder fails with that error and leaves the config file as it was
    - `theme compile` and `theme dump` of a folder with a text show the text in the background and no text record
    - the end-to-end tests that compile a theme with a text today pass with what this unit writes
- **R23** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 Draw the texts into the background.** The human chose this fix. The background is a full picture of 307 200
  bytes in every compiled theme (D4 of unit 0011), so a text drawn into it costs no byte, while as a record it costs its
  mask and a slot: 15 542 bytes and 24 slots in `minimal-dark-probes`, enough to bring it under the limit with its five
  alarms. A text never changes, so nothing is lost by making it part of the picture. Rejected: sharing one glyph set
  between the two records of an alarm number, or between numbers of one font, which needs a device check that numbers
  may share a resource (D11 of unit 0028) and was not chosen; removing widgets or alarms from the built-in theme, which
  undoes unit 0030; a background with no data (offset 0), whose meaning to the device is unknown
- **D2 A text over an earlier record stays a record.** The screen keeps its widgets as layers in the order of their
  slots, and the background is under them all. A text drawn into the background moves under every widget, which shows
  the same only when no widget before it shares a pixel with it. Keeping such a text as a record keeps every theme
  folder of today showing what it shows, at the cost of its mask, and no built-in theme has one. Comparing boxes and not
  the pixels drawn is the simpler rule and errs only toward a record. Rejected: drawing every text into the background,
  which changes the picture of a theme with a text over an image or a number; refusing such a theme, which breaks a
  folder that is valid today
- **D3 The blend of the preview.** The texts are drawn over the background as the preview draws a text record, then
  rounded to RGB565 once, so the preview of the compiled theme is that of today within the rounding of R6. How the
  device blends a text record is not known, so the screen may differ from today by a rounding at the anti-aliased
  edges of the letters, which the human's check of R14 looks at. Rejected: blending in RGB565 pixel by pixel, which
  rounds twice where texts overlap and moves the preview further from today
- **D4 The limit is the size of the file.** The compiler lays the resources out in the order of the slots with nothing
  after the last, so the data of every widget ends within the limit exactly when the file is no larger than the limit.
  503 808 is the one multiple of 4096 between the last end seen drawn, 503 692, and the first seen not drawn, 504 015.
  Taking 503 692 instead would refuse themes that may well work for a margin of 116 bytes, and nothing measured says the
  limit is lower than 503 808. The number is a constant of the compiler in one place, so a better measurement changes
  one line and the documents. The check replaces the flash limit, which a file within 503 808 bytes cannot reach
- **D5 Every command that compiles refuses.** The check is in the compiler, so `theme compile`, `theme preview`, `theme
  use` and `run` refuse the theme as they refuse any fault of a theme. A preview of a theme that the screen does not
  draw in full would mislead, and `run` would upload a theme that shows less than its preview. `upload` sends a file as
  it is and compiles nothing, so it is outside the check (item 2 of "Beyond the request")
- **D6 The message names the cost and not the widgets.** It gives the size, the limit and the excess, and what each kind
  of widget costs, which is what an author needs to choose what to remove. `theme dump` shows each resource and its
  size for a closer look. Rejected: listing the largest widgets in the message, item 3 of "Beyond the request"
- **D7 The 62 records stay, counted as written.** The record table has 64 slots, the header and the background take
  two, and that is a fact of the format whatever the screen draws. The observation showed that the slot count was not
  what failed: widgets in slots 42 to 44 were drawn when their data ended within the limit. So the limit of slots stays
  as a limit of the format, counted on the records the compiler writes, and the limit of bytes is the one a theme of
  the project meets first: 192 512 bytes of widget data leave room for 62 records only with small fonts and bars. A
  list may now be longer than 62 widgets, since its texts may take no slot. The early check before any widget is read
  goes, because the count is known only once the boxes of the texts are known
- **D8 The method of the alarm numbers is unaffected.** Unit 0028 hides one record of a pair with a transparent minus
  and needs no cover. The two records keep consecutive slots, the alarm widget after the normal one, and what shows
  through the hidden widget is the background, with the texts drawn into it, as it showed through before
- **D9 The theme folders do not change.** A text keeps its key and its meaning in `theme.yaml`, so no folder needs an
  edit, the built-in ones included. Where a text is written is the compiler's choice, as the layout of the resources
  already is
- **D10 The device check is the human's and the unit waits for it.** The bug is what the screen shows, and the sandbox
  cannot see the screen. R14 is checked by the human after the implementation and before the unit is set to
  `implemented`, as `AGENTS.md` says of every check that needs the device
- **D11 This unit supersedes unit 0011.** Its R9 makes the flash limit the size check of a theme and its R12 writes
  every text as a record, and read alone after this unit both would mislead. Unit 0028's R11 is replaced as well and is
  named above. Unit 0030 is not superseded: its theme, its tags and its record stay true, its record claims no result on
  the device, and this unit makes its theme fit without changing it
- **D12 The threat model is unit 0011's.** A theme folder may come from anyone, and D15 of unit 0011 holds. The new
  steps draw masks that are already made and checked against the screen, and compare boxes already checked, so they
  add no input of their own

## Non-goals

- Drawing images, numbers or datetimes into the background
- Finding the exact limit within 503 692 to 504 015 bytes, what in the device sets it, or whether the vendor samples
  are drawn in full
- A check of a file sent with `sensordeck upload`, or of a vendor theme read with `theme dump`
- Sharing glyph sets between records
- Any change of the keys of `theme.yaml`, of the theme folders, of the layout of the built-in themes, of `run`'s values
  or of the tags
- A change of how `theme dump` reads or draws a file
- Windows, where nothing of this unit differs

## Beyond the request

1. Drawing an image into the background by the rule of R1, as a text is. Cost: the same overlap rule for images, the
   alpha of an image blended over the background, tests and documents. Without it an image costs 3 bytes a pixel of
   the 192 512, which an author sees in the error of R8 and can answer by putting the image into `background.png`
2. A warning of `sensordeck upload` for a file larger than 503 808 bytes. Cost: a check of the size, a line on standard
   error, a test and a sentence of the README. Without it a vendor or hand-made file that large uploads with no word,
   and the screen leaves out its last widgets, as today
3. The largest widgets in the error of R8, with the bytes of each. Cost: a longer message, its tests and documents.
   Without it the author reads the costs from the hint and the sizes of the resources from `theme dump`
4. The room left in the line of `theme compile`, such as `wrote img.dat (492828 bytes, 10980 under the limit) and
   img.tags.yaml`. Cost: a change of the line, of the end-to-end tests that read it and of the README. Without it the
   author sees the size and compares it with the limit in the documents
5. One glyph set shared by the two records of an alarm number, or by every number of one font. Cost: a device check
   that numbers may share a resource, a change of the layout, tests and documents. Without it each number pays its own
   glyph set, which is what fills the limit in practice
