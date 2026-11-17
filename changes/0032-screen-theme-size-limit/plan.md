# The screen draws no widget whose data ends past 503 808 bytes

## Approach

The work is in the compiler, `internal/themecompile`, with the tests, the built-in previews and the documents after it.
Nothing changes in `themefile`, `themeimage`, `themedump` or `internal/cli`: every command that compiles reaches the
compiler through `themecompile.Compile`, and gives its error prefixed as it does today.

`Compile` keeps its single pass over the list, and decides each text as it reaches it:

- every built record carries its box, in a new field `area box` of `built`, set by each builder
- the boxes of the widget records written so far, the background left out, are kept in a slice. A text is built as
  today, its glyphs and its box checked first with the errors of today. When its box shares a pixel with none of the
  boxes in the slice, its `themefile.Text` record is appended to a list of texts for the background and nothing else
  advances: no slot, no offset, no box in the slice. Otherwise it is appended as a record exactly as today
- two boxes share a pixel when both have a width and a height above 0 and they overlap on both axes, compared with
  strict inequalities on `x < other.x + other.width` and the three others, so that a box that touches another at its
  edge shares no pixel. A box with a width or height of 0 shares no pixel with any box
- after the loop, the background is built with the texts drawn into it. Its resource is always 307 200 bytes, so the
  offsets of the widget records, computed during the loop from `backgroundOffset + 2*480*320`, stay true

The background with texts is made by the preview itself, so that R2 and R6 hold by construction: a
`themefile.Theme` whose entries are the background record of today (slot 1, its offset and its plane) and each text
record for the background in the order of the list is passed to `themeimage.Preview`, and each pixel of the
`*image.RGBA` it gives is made RGB565 with `rgb565`. The picture is opaque, so its premultiplied channels are its
colours. This is the composition of R2: `Preview` draws opaque black, then the background through `Expand`, then each
text mask in its font colour with `draw.Over`, exactly as it draws a text record. `themecompile` may import
`themeimage`, which imports only the standard library and `themefile`; `TestTheThemeCompilerDoesNoIO` in
`layout_test.go` forbids no such import. A theme with no text for the background skips this step, so its background is
byte for byte as today. Since `rgb565(Expand(c)) == c` for every RGB565 colour, the plane changes only where a mask
has an alpha other than 0, and the record table of the background is unchanged in both cases.

The limits:

- `checkWidgetCount` and `isAlarmNumber` in `document.go` go, with the constant `maxWidgets` moved to `compile.go` as
  the limit of records. After the loop, before `layout`, more than 62 records beside the background gives the error of
  R9. It comes before the size check, which gives R10, and before `layout`, which could not place a 64th slot in the
  table
- `FlashLimit` is replaced by `DrawLimit = 503808` in `compile.go`, the one place the number is written, with a
  comment that names the observation of 2026-11-17 and the format document. A function `checkSize(size int) error`
  gives the error of R8 for a size above it, and `Compile` calls it on the length of the file that `layout` gives. The
  307 200 of the message is written as `2*screenWidth*screenHeight`
- a widget fault anywhere in the list now comes before the count of records, since the count is known only at the end
  of the list (D7). The order is fixed, so a folder with several faults gives the same error each time

## Files

- `internal/themecompile/records.go`: the field `area` of `built`, set by every builder; the overlap test of two boxes;
  `buildBackground` takes the texts to draw
- `internal/themecompile/background.go`, new: the drawing of the texts into the background plane through
  `themeimage.Preview` and back to RGB565
- `internal/themecompile/compile.go`: the decision of each text, the list of texts for the background, the check of
  records and the size check; `DrawLimit` in place of `FlashLimit`; `maxWidgets` moved here from `document.go`
- `internal/themecompile/document.go`: `checkWidgetCount`, `isAlarmNumber` and their call removed
- `internal/themecompile/records_test.go`, `compile_test.go`, `document_test.go`, `background_test.go` (new): the tests
  of R1 to R10 and the existing tests brought to what the unit writes
- `internal/builtinthemes/builtinthemes_test.go`: the sizes and record counts of R12
- `docs/minimal-dark.png`, `docs/minimal-dark-probes.png`: written again (R13)
- `e2e/theme_test.go`, `e2e/theme_upload_test.go`: the tests of R22
- `docs/theme-format.md`, `docs/protocol.md`, `docs/theme-folder.md`, `README.md`: R15 to R18
- `changes/0032-screen-theme-size-limit/record.md`: the log of the work

## Tasks

### Texts drawn into the background

- [ ] In `records_test.go`, write the failing tests of the decision of a text, each compiling a solid theme with the
  fonts of `goodFonts` and reading the result with `themefile.Parse`: a lone text gives no record beside the
  background; a text whose box overlaps an earlier number, one that overlaps an earlier image, and one that overlaps an
  earlier text that was itself kept as a record (a text over a number, then a text over that text) are each a
  `themefile.Text` record in the slot of their place among the records, with their index byte the slot minus 1 and their
  offset after the resource of the record before; a text that overlaps only an earlier text drawn into the background is
  drawn into it too; a text whose box starts at the right edge (x equal to the number's x plus its width) of an earlier
  number, and one that starts at its bottom edge, are drawn into the background (R1, R3, R20)
- [ ] Keep the errors of a text unchanged: `TestFileFaults` (a glyph the font lacks) and
  `TestLargeHeaderIsRefusedBeforeDecoding` (a box past the edge) pass as they are, and add one case of each for a text
  that comes after a number it would overlap, so that both paths of the decision give the same error (R1, R20)
- [ ] In `background_test.go`, write the failing tests of the background resource, each against an expectation computed
  in the test from the rule of R2, pixel by pixel over the whole plane: a solid background with one text; a PNG
  background with varied pixels and partial alpha (made with `encodePNG`) with one text; two texts that overlap each
  other, in two colours, drawn in the order of the list; and a theme with numbers and no text, whose background resource
  equals `solidPlane` of its colour and whose background slot equals `solidBackgroundSlot()`. The expectation is
  written independently of `themeimage.Preview`: `themeimage.Expand` of each plane pixel, each mask pixel blended over it
  with `draw.Over` on an `*image.RGBA` and `rgb565` of the result. Check that the background slot of a solid
  background with a text still holds the solid colour in bytes 12-13 (R2, R20)
- [ ] In `records.go`, add `area box` to `built` and set it in `buildText`, `numberRecord` (both records of an alarm
  number), `buildBar`, `buildDatetime`, `buildImage` and `buildBackground`, and add `func (area box) overlaps(other
  box) bool` with the rule of the approach (R1)
- [ ] Add `internal/themecompile/background.go` with `drawTexts(plane []byte, texts []themefile.Text) []byte`, which
  gives a new plane and leaves its argument as it is, and change `buildBackground` to take the texts and use it only
  when there is at least one (R2)
- [ ] In `compile.go`, decide each text as the approach says, collect the texts for the background, build the background
  after the loop and put it in slot 1, and keep `AlarmSlots` computed from the slot the alarm number gets now. Write the
  doc comment of `Compile` and of `Result.Records` so that they say a text drawn into the background has no record
  (R1, R3, R4)
- [ ] Write the failing test of R4 first, then make it pass with the change above: a theme with a text drawn into the
  background, an alarm number, a text over the alarm number, a text drawn into the background and a bar gives the
  background in slot 1, the two records of the alarm number in slots 2 and 3 with indexes 1 and 2, the text record in
  slot 4 with index 3 and its offset right after the second glyph set, the bar in slot 5 with index 4, `AlarmSlots`
  `[3]`, and a file that ends at the end of the text's mask (R4, R20)
- [ ] Bring the existing tests that compile a text to what the unit writes, without weakening what they check of the
  other widgets: `TestRoundTrip` and `checkWidgets` (8 entries, the text checked in the background plane instead of a
  record), `TestTextRecord` (the text placed over an earlier number so that it is still a record, its slot and offset
  checked as today), `TestSecondWidgetHasIndex2` (two widgets that are records, such as two numbers), and
  `TestAlarmNumberIsTwoRecords`, `TestTheGlyphSetsOfAnAlarmNumberHideTheMinus` and `TestANumberWithoutAlarmHasNoAlarmSlots`
  through `alarmFolderTheme`, whose first text now goes into the background, so that their slots, indexes, offsets and
  `AlarmSlots` move down by one (R3, R4)
- [ ] Check that the tags are unchanged: the tests of `tags_test.go` pass as they are, and the test of R4 checks
  `Result.Tags` and `Result.Alarms` against the values written by hand (R5)
- [ ] In `background_test.go`, write the test of R6: a theme with a PNG background, a number, two overlapping texts
  drawn into the background, a text kept as a record over the number, and an image after the texts that covers part of
  one of them. Its "before" theme is put together in the test from the compiler's own builders: the background of
  today with `buildBackground` and no texts, and every text built as a record with `buildText` in the order of the list,
  with the other records as compiled. Both themes are drawn with `themeimage.Preview`; every pixel where no mask of a
  text drawn into the background has an alpha from 1 to 254 is equal, and every other one is within 4 in red and blue
  and 2 in green. A second case, a theme with no text, gives previews equal at every pixel (R6, R20)
- [ ] Commit: `bugfix-0032 Draw the texts of a theme into its background`

### The limits

- [ ] In `compile_test.go`, replace `TestWidgetCountAndFlashLimit` and `TestAnAlarmNumberCountsAsTwoWidgets` with failing
  tests: `checkSize(503808)` is nil and `checkSize(503809)` gives the error of R8 with 503809 and 1; a solid theme
  with one image of 480 x 320 (311 296 + 460 800 = 772 096 bytes) gives exactly
  `widgets: the compiled theme is 772096 bytes, 268288 more than the 503808 bytes that the screen draws widgets from; the background takes 307200 of them, each number a glyph set of its font and an alarm number two, each datetime a glyph set and each image 3 bytes a pixel, and a text none`;
  62 bars compile and 63 bars give exactly
  `widgets: 63 records, more than the 62 that fit beside the background; an alarm number is two records and a text drawn into the background none`;
  61 bars and one alarm number give the error with 63 records, in either order; 70 texts at distinct places that
  overlap no record compile with one record; 62 bars and one image of 480 x 320 give the error of the records, not of
  the size. Remove the case "63 widgets" of `TestParseDocumentErrors` in `document_test.go` (R8, R9, R10, R20)
- [ ] In `compile.go`, replace `FlashLimit` with `DrawLimit` and its comment, add `checkSize` and the check of records
  as the approach says, and remove `checkWidgetCount`, `isAlarmNumber` and their call from `document.go` (R8, R9, R10)
- [ ] Run `go test ./...` and check that the tests of `internal/cli` and `e2e` that refuse a theme through
  `theme compile`, `theme preview`, `theme use` and `run` still pass, since they go through the same `Compile` (R8)
- [ ] Commit: `bugfix-0032 Refuse a theme the screen cannot draw in full`

### The built-in themes

- [ ] In `builtinthemes_test.go`, add to the table `builtInThemes` the size of the compiled file and the number of
  records beside the background of each theme, 492 828 and 19 for `minimal-dark-probes` and 418 716 and 13 for
  `minimal-dark`, and check both in `TestEachThemeCompilesWithItsTags`, with no `themefile.Text` among the records. The
  tests of the folders, `TestEachThemeHasItsThemeYaml` and `TestEachBackgroundIsWhatTheGeneratorDraws`, and the check
  of the tags file stay as they are (R11, R12, R21)
- [ ] Before writing the images again, compare the preview each built-in theme gives now with its committed image:
  in a throwaway test or program outside the repository's committed files, every pixel equal or within 4 in red and
  blue and 2 in green. Log in `record.md` the number of pixels that differ and the largest difference of each channel
  for each theme (R13)
- [ ] Write the images again with `XDG_CONFIG_HOME` set to a new empty directory, so that no installed theme replaces a
  built-in one: `go run . theme preview minimal-dark docs/minimal-dark.png` and `go run . theme preview
  minimal-dark-probes docs/minimal-dark-probes.png`. `TestThePreviewImagesAreWhatTheCodeDrawsNow` in `docs_test.go`
  passes with them (R13, R21)
- [ ] Commit: `bugfix-0032 Write the previews of the built-in themes again`

### The end-to-end tests

- [ ] In `e2e/theme_test.go`, add a test in which `theme compile` of a folder over the limit (the folder of
  `writeThemeFolder` with a 480 x 320 PNG added and a theme with that image as a widget) exits with status 1, gives on
  standard error `sensordeck: <folder>/theme.yaml: ` followed by the error of R8 with its sizes, and leaves the output
  directory empty, so no theme file and no tags file (R22)
- [ ] In `e2e/theme_upload_test.go`, add a test in which `theme use` of that folder exits with status 1 with the same
  error and leaves the config file byte for byte as it was before, where it existed, and creates none where it did not
  (R22)
- [ ] In `e2e/theme_test.go`, add a test in which `theme compile` of a folder with a lone text and a number, then `theme
  dump` of the file, give a `theme.json` with no record of the type name `text`, and a `slot01-background.png` in
  which a pixel inside the text's mask differs from the solid colour of the background (R7, R22)
- [ ] Bring `TestThemeCompileWritesTheThemeFileAndTheTagsFileWithNoScreen` to 5 entries, since the text of
  `folderTheme` is drawn into the background, and run the whole of `e2e` (R22)
- [ ] Commit: `bugfix-0032 Cover the theme limit and the texts in the end-to-end tests`

### The documents

- [ ] `docs/theme-format.md`: in "Layout", beside the sentence on the 4 MB of the flash, which stays, the limit as
  observed on the device on 2026-11-17 with every fact of "What was observed on the device" of the spec, the range of
  503 692 to 504 015 bytes, why 503 808 is taken and that it is a guess within the range, that what sets the limit is
  unknown and that the vendor's tool-made sample was not checked against it. In "Open questions", the exact limit, what
  the device limits (the end of the data in the file, or a budget of another shape that the layout of the compiler
  cannot tell apart), and whether the vendor samples are drawn in full (R15, R19)
- [ ] `docs/protocol.md`: in "Theme upload", that an upload of a larger file succeeds and the screen then draws no widget
  whose data ends past the limit, with a link to the layout section of the format document. "Nature of the device"
  keeps its 4 MB (R16, R19)
- [ ] `docs/theme-folder.md`: in "The limits", the limit of R8 and its error in place of the flash limit, the 192 512
  bytes left for the data of the widgets, and the 62 records of R9 in place of the 62 widgets; a new section on what
  costs bytes, with the cost of each kind as R17 gives it, the sizes from the spec's table of a `big` number (16 200,
  18 954 in an alarm number), a `small` number (2 436, 2 856), the time (7 772) and the date (4 032) of
  `minimal-dark-probes` as the example, and that `theme dump` shows the size of each resource; a section on texts drawn
  into the background with the rule of R1 and why (D1, D2), and that a text kept as a record is written as "The bytes
  the compiler writes" says; in "Widgets", the records of R9 in place of "at most 62 of them"; in "The bytes the
  compiler writes", "slot 2 and up are the widgets" made "the records of the widgets", since a text drawn into the
  background has none (R17, R19)
- [ ] `README.md`: in "Writing a theme", a paragraph that the screen draws a theme of at most 503 808 bytes in full,
  that the compiler refuses a larger one and why, that the background takes 307 200 of them, what each widget costs in
  one sentence, and that texts cost nothing, with a link to the folder document; in the paragraph on `alarm`, "so it
  counts as two of the 62 widgets" replaced by a sentence that it is two records and two glyph sets (R18, R19)
- [ ] Read each changed paragraph against the documents of today, so that no paragraph that names the 4 MiB of the
  flash as a fact of the device is lost and nothing else changes its meaning (R19)
- [ ] Commit: `bugfix-0032 Document the limit of the screen and the texts in the background`

### Handing over

- [ ] Log in `record.md` that the work is done, with the sizes the built-in themes compile to, and that R14 is the
  human's check on the device, which the unit waits for before it is set to `implemented` (D10). Commit it alone:
  `bugfix-0032 Record the implementation` (R14)

## Verification

- [ ] `go test ./...` passes (R23)
- [ ] `go vet ./...` reports nothing (R23)
- [ ] `gofmt -l .` prints no file name (R23)
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing (R23)
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing (R23)
- [ ] `npm run check-changes` passes on the committed branch
- [ ] On the screen of the human, `minimal-dark-probes` shows the date, the time, every label and the five temperatures
  with their alarms, as its preview shows it. This is the human's check (R14)

## Commit Points

- [ ] `bugfix-0032 Write the plan`: `plan.md` and the log entry in `record.md`
- [ ] `bugfix-0032 Draw the texts of a theme into its background`
- [ ] `bugfix-0032 Refuse a theme the screen cannot draw in full`
- [ ] `bugfix-0032 Write the previews of the built-in themes again`
- [ ] `bugfix-0032 Cover the theme limit and the texts in the end-to-end tests`
- [ ] `bugfix-0032 Document the limit of the screen and the texts in the background`
- [ ] `bugfix-0032 Record the implementation`
