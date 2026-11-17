# Plan: a number widget that changes colour past a threshold

## Approach

The work follows the path of an alarm number through the code. First come the types it shares: the threshold and the
alarm pair, in `config`, which both the compiler and `run` use. Then the key `tags` of the config file, which reads and
checks pairs. Then the compiler: the key `alarm`, the tags, the two records and the tags file. Then the reports of a
round in `smartmonitor`, then `run`, which sends the pairs, then the preview, the end-to-end tests and the documents.

What the tree on `feature-0028-number-alarm-colour` holds, and what the plan builds on:

* `internal/config/config.go` holds `Mapping{Name string; Tag uint8}`, `Config.Mappings` (ordered by tag), `parseTags`,
  which reads each value with `parseTag(name, node)` and refuses two equal tags, after sorting by tag and then name,
  with `tags: %s and %s both have the tag %d`, and `refuseRepeatedKeys`. The texts of `parseTag` are
  `tags: <name>: the tag must be a whole number from 0 to 255` and `tags: <name>: the tag <value> is outside 0 to 255`
* `internal/themecompile/document.go` holds `widgetSpec`, `widgetKeys`, `readWidgets`, which refuses a list of more than
  `maxWidgets` items before reading any of them, `readNumberKeys` and `readNumberOptions`, which sets the default of
  `align` to `center`. `nodes.go` holds `fields`, `required`, `readWholeNumber(node, place, key, low, high)`,
  `readColour`, `problem` and `join`
* `internal/themecompile/tags.go` holds `clockTag`, `assignTags(metrics []string) []config.Mapping` and
  `TagsFile(tags []config.Mapping) []byte`. `compile.go` holds `Result{File; Records; Tags []config.Mapping}`,
  `metricsOf`, `tagOf` and the loop that calls `buildWidget` once for each widget and adds one `built` to `records`
* `internal/themecompile/records.go` holds `widgetInput{spec; font; folder; tag byte; slot; offset}`, `buildWidget`,
  which gives one `built`, and `buildNumber`, which writes the twelve glyphs of `themefile.NumberGlyphOrder` with
  `glyphs.glyphOf`
* `internal/smartmonitor/values.go` holds `EncodeValues(values []Value) []Report`, at most 20 values in each report,
  which `run` and the command `values` use
* `internal/cli/run.go` holds the line `running: %d metrics every %s` counting `len(settings.Mappings)`,
  `feeder.probesMapped`, `feeder.reload`, which copies `Mappings` and `Metrics`, `feeder.read`, which reads the mapped
  metrics and gives the values in the order of `Mappings`, and `feeder.round`, which sends
  `smartmonitor.EncodeValues(feeding.read(...))`. `internal/cli/runconfig.go` holds `loadConfig`, which checks each
  name of `compiled.Tags` with `settings.IsMetric` and sets `settings.Mappings = compiled.Tags`. `internal/cli/theme.go`
  writes `themecompile.TagsFile(result.Tags)` and draws `themeimage.Preview(theme)` for `theme preview`
* `internal/themeimage/preview.go` holds `Preview(theme)`, `drawRecord` and `drawNumber`, which draws `SampleValue`, or
  `thousandsValue` in the thousands mode. `themedump` and the tests call `Preview(theme)`
* `e2e/custom_metrics_test.go` holds `compileTheme`, `writeFile` and `waitForReport` use with a custom metric read from a
  file; `e2e/theme_test.go` holds `writeThemeFolder` and the dump and compile tests

The design decisions the spec leaves to the plan:

* **The shared types live in `config`.** `config.Threshold{Above, Below int16; HasAbove, HasBelow bool}` with
  `Alarm(value int16) bool`, true when `HasAbove && value > Above || HasBelow && value < Below` (R2), and
  `config.AlarmPair{Name string; Threshold Threshold; Normal, Alarm uint8}`. `config.HidingValue int16 = -1` (D8) is
  the one place of the hiding value. `Config` gains `Alarms []AlarmPair`, ordered by `Normal` ascending, beside
  `Mappings`, which keeps the plain entries. The compiler's `Result` gains `Alarms []config.AlarmPair` in the same
  order, so that R18 compares two values of the same types. A method `Config.MetricNames() []string` gives the
  distinct names of `Mappings` and `Alarms`, sorted, and is the one place that `run` asks for the metrics it reads
* **The tags of the compiler are one sorted list of entries.** `assignTags` takes the widgets, makes an entry for each
  plain metric and for each distinct triple of an alarm number, sorts them as R13 says and gives the tags in that
  order, skipping 21 for each tag and so between the two of a pair. It gives `[]config.Mapping` and
  `[]config.AlarmPair`
* **An alarm number is built as one widget that gives two records.** `buildWidget` gives `[]built`, one for each kind
  but the alarm number, and the loop of `Compile` appends them all, each next slot and offset following the one before.
  `widgetInput` gains `alarmTag byte`. `Result` gains `AlarmSlots []int`, the slots of the alarm widgets in the
  numbering of `themefile.Entry.Slot`, so that `theme preview` knows them (R22, D12)
* **The count of records is checked before any widget is read, as today.** `readWidgets` counts the items of the list,
  an item counting as two when it is a mapping whose `type` is the string `number` and that has the key `alarm`, and
  refuses a count over 62 with the text of R11 before reading any widget, so that the order of the errors does not
  change for a list with no alarm number
* **A round is a list of units.** `smartmonitor.EncodeUnits(units [][]Value) []Report` fills the reports as R21a says,
  and `EncodeValues` becomes `EncodeUnits` of units of one value each, so that the command `values` and every round
  without a pair keep their reports. A pure function `roundUnits(settings config.Config, readings map[string]float64)
  [][]smartmonitor.Value` in `internal/cli/roundvalues.go` makes the units of a round (R19 to R21a), and
  `feeder.read` gives its result
* **The preview takes the values of some slots.** `themeimage.PreviewWithValues(theme, values map[int]int16)` draws the
  number of each slot in the map with that value, as text from `strconv.Itoa`, by the rules of R13 to R15 of unit 0027,
  and `Preview(theme)` is `PreviewWithValues(theme, nil)`. `theme preview` passes `config.HidingValue` for each slot of
  `Result.AlarmSlots`. `themeimage` so does not import `config`, and the dump keeps `Preview` (R23)

## Files

- `internal/config/alarms.go` (new): `Threshold`, `Alarm`, `AlarmPair`, `HidingValue`, `MetricNames`, and the reading of
  the mapping form of a value of `tags`
- `internal/config/alarms_test.go` (new): the threshold, the new forms of `tags` and their errors
- `internal/config/config.go`: `Config.Alarms`, `parseTags` taking either form and checking every tag distinct,
  `parseTag` taking a place instead of a name
- `internal/themecompile/document.go`: the key `alarm`, its errors, the left alignment, the refusal of `thousands`, the
  count of R11
- `internal/themecompile/document_test.go`: R1 to R5 and R11
- `internal/themecompile/tags.go`: `assignTags` for entries and pairs, `TagsFile` for the flow form
- `internal/themecompile/tags_test.go`: R13, R14, R17 and R18
- `internal/themecompile/compile.go`: `Result.Alarms`, `Result.AlarmSlots`, the loop over `[]built`, the tags of a widget
- `internal/themecompile/records.go`: `buildWidget` giving `[]built`, `buildAlarmNumber`, the transparent `-`
- `internal/themecompile/glyphs.go`: `hiddenMinus(width)`, a mask of zeros of the glyph height
- `internal/themecompile/records_test.go` and `compile_test.go`: R6 to R12
- `internal/smartmonitor/values.go` and `values_test.go`: `EncodeUnits` and R21a
- `internal/cli/roundvalues.go` and `roundvalues_test.go` (new): `roundUnits` and R2, R19 to R21
- `internal/cli/run.go`: the count of the line `running:`, `probesMapped`, `reload`, `read` and `round`
- `internal/cli/runconfig.go`: the pairs of the theme taken and checked
- `internal/cli/runtheme_test.go`: `run` with the key `theme` naming a folder with an alarm number
- `internal/cli/theme.go` and `theme_test.go`: `TagsFile` with the pairs, the preview of the normal state (R22)
- `internal/themeimage/preview.go` and `preview_test.go`: `PreviewWithValues`
- `e2e/alarm_test.go` (new): R28
- `docs/theme-folder.md`, `docs/theme-format.md` and `README.md`: R24 to R26

## Tasks

### The shared types and the key `tags` (R2, R15 to R18)

- [ ] Write `internal/config/alarms_test.go` with tests, failing, of: `Threshold.Alarm` for `above: 80` at 80 and 81,
  `below: 1000` at 999 and 1000, `above: 80, below: 20` at 19, 20, 80 and 81 (R2); `Parse` of a block with
  `cpu.temp: 1` and `gpu.temp: {tag: 2, alarms: [{above: 80, normal: 3, alarm: 4}]}` giving `Mappings`
  `[{cpu.temp 1} {gpu.temp 2}]` and `Alarms` with the one pair; a metric with only `alarms`; a metric with only `tag`
  in the mapping form, which gives the same `Mappings` as the scalar form; two pairs of one metric ordered by their
  normal tag; each block valid before this unit giving the same `Config` as before (R15); `MetricNames` giving each
  name once. And the errors of R16, each with its full text: two equal tags of two metrics, as today; a plain tag and a
  pair tag of the same metric equal (`tags: gpu.temp and gpu.temp both have the tag 3`); `normal` equal to `alarm`; a
  mapping with neither `tag` nor `alarms` (`tags: gpu.temp: want tag, alarms or both`); `alarms` that is not a
  non-empty list (`tags: gpu.temp: alarms: want a non-empty list of alarm pairs`); an item that is not a mapping
  (`tags: gpu.temp: alarms: 1: want a mapping with normal, alarm, above and below`); a pair with neither `above` nor
  `below` (`tags: gpu.temp: alarms: 1: want above, below or both`); a missing `normal` or `alarm`
  (`tags: gpu.temp: alarms: 1: the key normal is missing`); `normal: 300`
  (`tags: gpu.temp: alarms: 1: normal: the tag 300 is outside 0 to 255`); `normal: x`
  (`tags: gpu.temp: alarms: 2: normal: the tag must be a whole number from 0 to 255`, with the pair second in the list);
  `above: 80.5`, `above: "80"` and `above: 40000`
  (`tags: gpu.temp: alarms: 1: above: want a whole number from -32768 to 32767`), and the same of `below`; an unknown key
  in the mapping of a metric (`tags: gpu.temp: unknown key "x", the keys are alarms and tag`) and in a pair
  (`tags: gpu.temp: alarms: 1: unknown key "x", the keys are above, alarm, below and normal`); a key given twice in
  either mapping, refused with `refuseRepeatedKeys` as the root is (R16)
- [ ] Add `internal/config/alarms.go` with `Threshold`, `Alarm`, `AlarmPair`, `HidingValue` and `MetricNames`, and the
  reading of a mapping value of `tags`: `tag` read with `parseTag`, `alarms` read item by item, `normal` and `alarm` with
  `parseTag`, `above` and `below` as whole numbers of the `!!int` tag from `math.MinInt16` to `math.MaxInt16`. Change
  `parseTag` to take the place, the metric name for a plain tag and `<name>: alarms: <n>: normal` for a tag of a pair,
  so that its texts stay those of today for a plain tag. Add `Alarms` to `Config`, ordered by `Normal`. Change the check
  of distinct tags in `parseTags` to gather every tag of the block, plain and of pairs, each with its metric name, sort
  them by tag and then name, and refuse two equal neighbours with today's text. Make the tests pass (R2, R15, R16)

### The key `alarm` (R1, R3 to R6, R11)

- [ ] Add tests, failing, to `internal/themecompile/document_test.go`: `parseWidget` of an alarm number with `above`,
  with `below` and with both, giving the threshold and the colour; without `align` its alignment is 0, and with
  `align: left` too (R4); `thousands: false` accepted (R5). And each error with its full text, for widget 1:
  `alarm: 3` (`widget 1 (number): alarm: want a mapping with above, below and color`); `alarm: {color: "#ff0000"}`
  (`... alarm: want above, below or both`); `above` of `80.5`, of `"80"`, of `40000` and of `-40000`
  (`... alarm: above: want a whole number from -32768 to 32767`), and the same of `below`; a missing `color`
  (`... alarm: the key color is missing`, the text `required` gives) and `color: red` (the text `readColour` gives
  under `alarm: color`); an unknown key (`... alarm: unknown key "x", the keys are above, below and color`);
  `align: center` and `align: right` (`... align: an alarm number is left-aligned, want left or no align`);
  `thousands: true` (`... thousands: an alarm number cannot show thousands`) (R1, R3 to R5)
- [ ] Add tests, failing, of R11 to `internal/themecompile/compile_test.go`: 60 texts and one alarm number compile; 61
  texts and one alarm number are refused with
  `widgets: 63 widgets, more than the 62 that fit beside the background, an alarm number counting as two`; 63 texts are
  refused with `widgets: 63 widgets, more than the 62 that fit beside the background`, as today (R11)
- [ ] In `internal/themecompile/document.go`, add `alarm` to the keys of `number` and an `alarm *alarmSpec` to
  `widgetSpec`, with `threshold config.Threshold` and `colour config.Colour`. Read it in `readNumberKeys` after
  `readNumberOptions`, with `fields` of the keys `above`, `below` and `color`, `readWholeNumber` from -32768 to 32767
  under the place `join(place, "alarm")`, `required` and `readColour` for `color`. For an alarm number, set the
  alignment to `left` when the key `align` is absent, and refuse `align` other than `left` and `thousands: true` with
  the texts of R4 and R5. In `readWidgets`, count the records of the list as the approach says before reading any
  widget, and refuse a count over 62 with the text of R11, adding the ending only when an alarm number is in the list.
  Make the tests pass (R1, R3 to R6, R11)

### The tags of the compiler and the tags file (R13, R14, R17, R18)

- [ ] Add tests, failing, to `internal/themecompile/tags_test.go`: a plain number, a bar and two alarm numbers of
  `gpu.temp` with `above: 80` and `above: 90`, and a number of `cpu.temp`, giving `cpu.temp` 1, `gpu.temp` 2 and the
  pairs 3 and 4, then 5 and 6; a pair with no `above` before one with `above: -5`, and within one `above` the
  `below` ordered the same way; two alarm numbers of one metric and thresholds with different colours, boxes and fonts
  sharing one pair; a theme whose entries put a pair on tag 20, which gets the alarm tag 22; the same widgets in another
  order giving the same tags; the tags of a theme with no alarm number as `assignTags` gave them before (R13, R14).
  And `TagsFile` of the example of R17 byte for byte, of a metric with only pairs (no `tag, `), of a pair with both
  `above` and `below`, of a pair with only `below`, of tags with no pair as today, and of a compiled theme with a plain
  entry and two pairs of one metric parsed back with `config.Parse` into `Mappings` and `Alarms` equal to the
  compiler's `Result.Tags` and `Result.Alarms` (R17, R18)
- [ ] Change `assignTags` in `internal/themecompile/tags.go` to take the widgets and give `[]config.Mapping` and
  `[]config.AlarmPair`, by the entries and the sort of R13, with `clockTag` skipped for each tag given. Change
  `TagsFile(tags []config.Mapping, alarms []config.AlarmPair)` to write the metrics in the order of their lowest tag,
  a metric with no pair as today and one with pairs in the flow form of R17. Remove `metricsOf`, whose job
  `assignTags` takes. Make the tests pass (R13, R14, R17, R18)

### The records of an alarm number (R6 to R10, R12)

- [ ] Add tests, failing, to `internal/themecompile/records_test.go`: a theme of a text, an alarm number with
  `digits: 4` and a number after it, compiled and read back with `themefile.Parse`. Check the alarm number's two records
  in slots 3 and 4 with the indexes 2 and 3 and the number after them in slot 5 with index 4; both records byte for byte
  against a slot built from the glyphs of the font as `numberSlot` does, with byte 11 0, byte 14 0, byte 15 as the
  opacity, the colour of the font on the first and of `alarm.color` in RGB565 on the second, the normal and the alarm
  tag; the offsets, each record's glyph set after the resource before it; the two glyph sets equal; the glyph of `-`
  with the width of four digit widths and a mask of zeros; the other eleven glyphs and widths equal to those of a number
  without `alarm` of the same font (R7 to R10). A number without `alarm` compiled to the bytes it had before, by the
  existing `TestNumberRecord` and a check that `Result.AlarmSlots` is empty (R6). An alarm number in a font without
  `-` refused with the error of today for a missing glyph (R12)
- [ ] Add `hiddenMinus(width int) []byte` to `internal/themecompile/glyphs.go`. In `internal/themecompile/records.go`,
  make `buildWidget` give `[]built`, add `alarmTag` to `widgetInput` and a `buildAlarmNumber` that builds the normal
  record as `buildNumber` does with the glyph of `-` replaced by `hiddenMinus` of the box width, then the alarm record
  in the next slot with its own copy of the resource, the colour of `alarm.color`, the alarm tag, the next index and the
  offset after the first resource. In `internal/themecompile/compile.go`, give each widget its tags from the result of
  `assignTags`, append every record `buildWidget` gives with the slot and offset after the one before, and fill
  `Result.Alarms` and `Result.AlarmSlots`. Make the tests pass (R6 to R10, R12)
- [ ] Run the five commands of the suite. Commit: `feature-0028 Compile a number with an alarm colour into two records
  and a pair of tags`

### What `run` sends (R19 to R21a)

- [ ] Add tests, failing, to `internal/smartmonitor/values_test.go` of `EncodeUnits`: 19 units of one and a unit of two
  giving two reports, the first of 19 values and the second of the pair; 18 units of one and a pair giving one report
  of 20; 21 units of one giving the two reports `EncodeValues` gave before; no unit giving no report (R21a)
- [ ] Add `EncodeUnits` to `internal/smartmonitor/values.go` and make `EncodeValues` call it with units of one value.
  Make the tests pass, and the existing tests of `EncodeValues` with them (R21a)
- [ ] Add `internal/cli/roundvalues_test.go` with tests, failing, of `roundUnits`: a pair with `above: 80` sent with
  the readings 79, 80, 81 and back to 79, giving the value to the normal tag and -1 to the alarm tag in the normal
  state and the reverse in the alarm state; the same with `below: 1000` and 1001, 1000, 999, 1000; readings of 80.4 and
  80.5 with `above: 80`; a metric with no reading giving no unit; a plain entry and a pair of one metric in one round,
  the plain value a unit of one and the pair a unit of two; units ordered by their lowest tag; a block whose pair has
  the tags 2 and 5 with a plain tag 3 between them, giving the units 2 and 5, then 3 (R2, R19 to R21a)
- [ ] Add `roundUnits` in `internal/cli/roundvalues.go`, pure, taking the settings and the readings by name. In
  `internal/cli/run.go`, make `feeder.read` give the units of `roundUnits` and read the names of `MetricNames`, make
  `round` send `smartmonitor.EncodeUnits` of them, make `probesMapped` and the count of the line `running:` use
  `MetricNames`, and make `reload` copy `Alarms` beside `Mappings`. In `internal/cli/runconfig.go`, set
  `settings.Alarms = compiled.Alarms` and check every name of `MetricNames` with `IsMetric`, with the text of today.
  Make the tests pass (R19 to R21a)
- [ ] Add tests to `internal/cli/runtheme_test.go`: `run` with the key `theme` naming a folder with an alarm number
  of a metric of the fake source writes the value report of the normal state; the line `running: 1 metrics every ...`
  for a theme with a plain number and an alarm number of the same metric; a theme whose alarm number names a metric that
  is neither built in nor defined, refused with the error of today (R19, R21). And to `internal/cli/runreload_test.go`
  or `runconfig_test.go`: a config with the key `tags` holding a pair sends both tags of the pair (R19)
- [ ] Run the five commands of the suite. Commit: `feature-0028 Send the values of an alarm pair by its threshold, in
  one report`

### The preview (R22, R23)

- [ ] Add tests, failing, to `internal/themeimage/preview_test.go`: `PreviewWithValues` of a number drawn with the
  value -1 from a glyph set whose `-` is transparent and as wide as the box draws no pixel; a number not in the map is
  drawn as 42 as today. And to `internal/cli/theme_test.go`: `theme preview` of an alarm number in a white font with a
  red `alarm.color` has a white pixel in the value and no pixel of the red; the preview of the dump of the same compiled
  theme has the red (R22, R23)
- [ ] Add `PreviewWithValues` to `internal/themeimage/preview.go` and make `Preview` call it with no values. In
  `internal/cli/theme.go`, draw `theme preview` with `config.HidingValue` for each slot of `Result.AlarmSlots`, and
  write the tags file with `TagsFile(result.Tags, result.Alarms)`. Make the tests pass (R22, R23)

### The end-to-end tests (R28)

- [ ] Add `e2e/alarm_test.go`: compile a folder with an alarm number of `gpu.temp` with `above: 80`, check the tags file
  byte for byte and read the theme file back with `theme dump`, checking in its `theme.json` the two number records,
  their slots, tags and colours. Then `run` with the key `theme` naming a folder with an alarm number of a custom metric
  read from a file with `above: 80`, as `TestRunSendsNegativeAndClampedCustomMetrics` does: the file at 47, a report with
  the normal tag 47 and the alarm tag `FF FF`; then at 95, a report with the normal tag `FF FF` and the alarm tag 95;
  then at 52, a report with the normal tag 52 and the alarm tag `FF FF`. The tags are read from the tags file of the
  compiled theme (R28)
- [ ] Run the five commands of the suite. Commit: `feature-0028 Preview the normal state of an alarm number and test it
  end to end`

### The documents (R24 to R26)

- [ ] In `docs/theme-folder.md`: add `alarm` to the keys of a number in the table of "Widgets" and a section "A number
  with an alarm colour" after "Formats of a datetime", which gives the keys and their errors, the comparison of R2 on
  the value as sent, the left alignment and why (D5), the refusal of `thousands` (D6), how the compiler writes it (R7 to
  R10) on the observation of 2026-11-17 with a link to the format document, that an alarm number cannot show a negative
  value and that `below: 0` hides the value, that `theme preview` draws the normal state and the preview of
  `theme dump` the alarm widgets over the normal ones. In "The tags", the rule of R13 and R14 and the flow form of R17
  with its example. In "The limits", that an alarm number counts as two of the 62 widgets (R24)
- [ ] In `docs/theme-format.md`: a section of the observation of 2026-11-17, marked **observed on the device**, with
  every fact of "What was observed on the device" of the spec; and in "Open questions", the item on the order in which
  the device draws the records says that the screen was seen to keep its widgets as layers in the order of their slots,
  a later slot over an earlier one, and what stays open: whether a later slot is always drawn over an earlier one, and
  the order of records of other types (R25)
- [ ] In `README.md`, "Writing a theme": the example of an alarm number of R26 and its paragraph, with a link to the
  folder document; the tags file example extended with a metric with an alarm pair. In the section of the key `tags`
  under "Showing sensor values on the screen", that a metric may map to a mapping with `tag` and `alarms`, as the tags
  file writes it (R26)
- [ ] Commit: `feature-0028 Document the number with an alarm colour`

## Verification

- [ ] `go test ./...` passes, which runs the end-to-end tests (R27, R28, R29)
- [ ] `go vet ./...` reports nothing (R29)
- [ ] `gofmt -l .` prints no file name (R29)
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing (R29)
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing (R29)
- [ ] `npm run check-changes` succeeds
- [ ] The tags files of the two built-in themes are unchanged, by `e2e/builtin_theme_test.go`, and so are the committed
  preview images, by `docs_test.go` (R13, R23)

## Commit Points

- [x] `feature-0028 Write the plan of the number with an alarm colour`
- [ ] `feature-0028 Compile a number with an alarm colour into two records and a pair of tags`
- [ ] `feature-0028 Send the values of an alarm pair by its threshold, in one report`
- [ ] `feature-0028 Preview the normal state of an alarm number and test it end to end`
- [ ] `feature-0028 Document the number with an alarm colour`
