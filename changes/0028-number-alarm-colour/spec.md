# A number widget that changes colour past a threshold

## Context

A theme author wants a number to change colour when its value passes a threshold, such as a GPU temperature that turns
red above 80 °C. The screen has no such feature of its own: no byte of a number or of a bar that was tried on the
device gave a threshold colour. No value makes a number invisible once it showed a value either: 65535 shows `-1` and
32768 shows `-32768`, since the screen reads every value as a signed 16-bit number (unit 0026).

On 2026-11-17 the human showed on the screen a way to do it with two number widgets, in a theme made by hand. The
theme and its scripts are kept on the host, outside the repository, and the sandbox the unit is built in does not have
them, so this spec carries every fact the unit relies on.

"R*n* of unit 00*nn*" below names a requirement of the spec of that unit in `changes/`. "The format document" is
`docs/theme-format.md` and "the folder document" is `docs/theme-folder.md`.

### What was observed on the device

- **The theme**: two number widgets on the same spot with the same box, the normal one in white and the alarm one in
  red, each with a tag of its own, both left-aligned (byte 11 at 0). The alarm widget was later in the order of the
  slots than the normal one
- **The glyph sets**: in both widgets the glyph of `-` was fully transparent, every alpha 0, and as wide as the box of
  the widget. The other glyphs, `0` to `9` and `.`, were as the compiler of sensordeck makes them
- **Why that hides a widget**: a negative value draws the transparent minus first, which fills the box, and pushes the
  digits past the right edge of the box, where the screen clips them (unit 0027). So a widget sent a negative value
  shows nothing
- **The normal state**: the normal widget was sent the value and the alarm widget -1. The screen showed a white 47 and
  nothing red
- **The alarm state**: the normal widget was sent -1 and the alarm widget 95. The screen showed a red 95 and nothing
  white
- **Back to normal**: a white 52 and nothing red
- **The layers**: the screen keeps its widgets as layers in the order of their slots
- **Covers fail**: an earlier design put an opaque cover over the widget to hide. A cover hid every layer below it and
  not only the one widget, so covers are not part of this unit

It follows that such widgets cannot show a negative value of their own: its digits are clipped as those of -1 are.

### The requirements this unit replaces

- Unit `0011-theme-compiler`: in R5, the keys of a number widget; in R18, the rule that gives each distinct metric
  name one tag; in R20, the content of the tags file; the limit of 62 widgets, which counted each widget of the list as
  one record
- Unit `0027-number-options-and-bars`: in R1, the default of `align`, which an alarm number does not have (R4); in R8,
  the tags of the numbers and the bars as one tag for each distinct metric name; in R13, that every number is drawn as
  `42`, for `theme preview` (R20)
- Unit `0004-linux-sensors-and-run` and unit `0013-theme-key-and-upload-on-change`: that the key `tags` maps a metric
  to one tag, and that `run` sends each mapped metric to one tag

## Requirements

"An alarm number" is a number widget with the key `alarm`. Its "normal widget" and its "alarm widget" are the two
number records the compiler writes for it. "The value" of a metric in a round is its reading as `sensors.ToValue`
gives it (R1 of unit 0026): rounded, halves away from zero, and clamped to -32768 to 32767. "The hiding value" is -1.

### The key `alarm` in `theme.yaml`

- **R1** A number widget takes the optional key `alarm`, a mapping with:
    - `above`, optional: a whole number from -32768 to 32767
    - `below`, optional: a whole number from -32768 to 32767
    - `color`, required: a colour as R2 of unit 0011 has it, the colour of the value in the alarm state

  At least one of `above` and `below` is given, and both may be. Any other key in it is an error that names `alarm` and
  the key, as R1 of unit 0011 says of every key not listed
- **R2** The alarm state holds for a value when it is greater than `above`, or less than `below`, of those given. So
  with `above: 80` the value 80 is normal and 81 is the alarm, and a reading of 80.4 is normal and 80.5 the alarm. With
  `below: 1000` the value 999 is the alarm and 1000 is normal. With `above: 80` and `below: 20`, 19 and 81 are the
  alarm and 20 to 80 normal
- **R3** These are errors of the form of R8 of unit 0011, each naming the widget:
    - `alarm` that is not a mapping: `alarm: want a mapping with above, below and color`
    - neither `above` nor `below`: `alarm: want above, below or both`
    - `above` or `below` that is not a whole number from -32768 to 32767, a decimal and a string such as `"80"`
      included: `alarm: above: want a whole number from -32768 to 32767`, and the same with `below`
    - a missing or invalid `color`: an error that names `alarm` and `color`, as a colour of a bar is refused
- **R4** An alarm number is left-aligned. Without the key `align` it is `left`, and `align: left` is accepted.
  `align: center` and `align: right` are the error `align: an alarm number is left-aligned, want left or no align`
- **R5** `thousands: true` on an alarm number is the error `thousands: an alarm number cannot show thousands`.
  `thousands: false` is accepted
- **R6** `digits`, `font`, `metric`, `opacity`, `x` and `y` of an alarm number are as for every number, and apply to
  both of its widgets. A number widget without `alarm` compiles to the same bytes as before this unit

### The records of an alarm number

- **R7** An alarm number is written as two number records in consecutive slots: its normal widget, then its alarm
  widget in the next slot. The widgets after it in the list take the slots after both. The index byte of each record is
  its slot number minus 1, as for every widget (R13 of unit 0011)
- **R8** Both records have the same box, the box of a number of its `digits` and font (R4 of unit 0027), at its x and
  y; byte 11 0; byte 14 0; byte 15 the opacity as R3 of unit 0027 writes it; the glyph height of the font; and bytes 46
  to 63 zero
- **R9** The normal widget has the colour of the font and the tag of R13. The alarm widget has the colour of
  `alarm.color`, made RGB565 as R15 of unit 0011 says, and the alarm tag of R13
- **R10** Each of the two records points to a glyph set of its own, written after the records before it as every
  resource is. The two glyph sets are byte for byte the same. In each, the glyphs of `0` to `9` and `.` and their widths
  are those that a number without `alarm` of the same font has, and the glyph of `-` has the width of the box, `digits`
  digit widths, and a mask of the glyph height times that width whose every byte is 0
- **R11** A theme holds at most 62 number, text, datetime, image and bar records beside the background, an alarm number
  counting as two. A list over the limit is the error
  `widgets: <n> widgets, more than the 62 that fit beside the background`, where `<n>` counts an alarm number as two,
  and the text ends in `, an alarm number counting as two` when the list holds an alarm number. A list with no alarm
  number gives the error of today
- **R12** The fonts of an alarm number need the same glyphs as any number (the limits of the folder document), `-`
  included, and the error for a missing one does not change

### The tags

- **R13** The compiler gives tags to these entries:
    - a plain entry for each distinct metric name of the numbers without `alarm` and of the bars, which takes one tag
    - an alarm pair for each distinct triple of a metric name, an `above` or none and a `below` or none among the alarm
      numbers, which takes two tags: the normal tag, then the alarm tag. Two alarm numbers with the same metric and the
      same `above` and `below` share a pair, whatever their colours, boxes and fonts

  The entries are sorted by their metric name, by its bytes, and within one metric name the plain entry comes first,
  then the pairs, ordered by `above`, none before any number and the numbers ascending, then by `below` in the same way.
  The tags are given in that order as 1, 2, 3 and up, with 21 skipped as R18 of unit 0011 says, a pair taking two of
  them, its normal tag first. The normal and the alarm tag of a pair may so be 20 and 22. No entry gets 0, and the tags
  do not depend on the order of the widgets. A theme with no alarm number gets the tags it got before this unit
- **R14** A number without `alarm` and a bar keep the tag of the plain entry of their metric. An alarm number never
  shares a tag with a number without `alarm` or with a bar of the same metric

### The key `tags` of the config file and the tags file

- **R15** In the key `tags` of the config file of `run`, the value of a metric is either:
    - a whole number from 0 to 255, its tag, as today
    - or a mapping with the optional keys `tag`, a whole number from 0 to 255 that the metric is sent to as a plain
      entry, and `alarms`, a non-empty list of alarm pairs. At least one of the two is given. An alarm pair is a mapping
      with `normal` and `alarm`, both required and each a whole number from 0 to 255, and `above` and `below` as in R1,
      at least one of them

  Every block that is valid today stays valid and means the same
- **R16** Every tag of the block, plain or of a pair, is distinct from every other. Two that are the same are the error
  that names both metrics and the tag, as today, a metric that is named twice when both tags are its own. An invalid
  part of the new form is an error that names the metric and the key, and for an item of `alarms` its position counted
  from 1, such as `tags: gpu.temp: alarms: 1: normal: the tag must be a whole number from 0 to 255`. Any other key in
  the mapping of a metric or of a pair is an error that names it
- **R17** The tags file writes a metric that has only a plain entry as today, `  <metric>: <tag>`, so that the tags file
  of a theme with no alarm number is byte for byte what it was. A metric with an alarm pair is one line in the flow
  style of YAML: `  <metric>: {tag: <t>, alarms: [<pair>, <pair>]}`, with `tag, ` and its value left out when the
  metric has no plain entry, the pairs in the order of R13 and separated by `, `, and each pair written
  `{above: <a>, below: <b>, normal: <n>, alarm: <m>}` with `above: <a>, ` or `below: <b>, ` left out when not given. The
  metrics are in the order of the first tag of each, ascending. For example:

  ```yaml
  # Paste this block into the config file of sensordeck run.
  tags:
    cpu.temp: 1
    gpu.temp: {tag: 2, alarms: [{above: 80, normal: 3, alarm: 4}]}
  ```
- **R18** The tags file of a compiled theme, parsed as the key `tags` of a config file, gives exactly the plain entries
  and the alarm pairs that `run` takes from that theme with the key `theme`

### What `run` sends

- **R19** In each round, for each metric that has a value, `run` sends:
    - the value to the tag of its plain entry, when it has one, as today
    - for each of its alarm pairs: in the normal state the value to the normal tag and the hiding value to the alarm
      tag, and in the alarm state the hiding value to the normal tag and the value to the alarm tag

  This holds with the key `theme`, from the tags the compiler gave, and with the key `tags`, from the block
- **R20** A metric that has no value in a round is not sent, to its plain tag or to either tag of a pair, so the screen
  keeps what it showed, as for any metric today
- **R21** The pairs of a round are sent in the order of their tags, ascending, in the reports of the value update as
  today. `run` refuses a theme or a block in which a metric of a pair is neither built in nor defined under `metrics`,
  with the error of today. The line `running: <n> metrics every <interval>` counts each metric once, whatever entries it
  has

### The preview

- **R22** `sensordeck theme preview` draws the normal state: the normal widget of an alarm number with the sample value
  of R13 of unit 0027, and its alarm widget with the value -1, drawn by the rules of R13 to R15 of unit 0027 with the
  glyphs of `-` and `1` of its glyph set, so that the alarm widget draws no pixel. Every other record is drawn as today
- **R23** `sensordeck theme dump` draws every number of a theme file as today, since the file does not say which
  records form a pair. The preview of the dump of a compiled theme with an alarm number so shows its alarm widget's `42`
  in the alarm colour, over its normal widget

### The documents

- **R24** The folder document:
    - describes the key `alarm` in the table of the widgets and in a section of its own: its keys, the comparison of R2
      on the value as sent, the left alignment of R4 and why, the refusal of `thousands` of R5
    - says how the compiler writes an alarm number, as R7 to R10 say, and that the method rests on the observation of
      2026-11-17, with a link to the format document
    - says that an alarm number cannot show a negative value: in the normal state as in the alarm state, a negative
      value shows nothing, so `below: 0` hides the value rather than showing it in the alarm colour
    - in the limits, says that an alarm number counts as two of the 62 widgets
    - in the section on the tags, gives the rule of R13 and R14 and the form of the tags file of R17
    - says that `theme preview` draws the normal state and that the preview of `theme dump` draws the alarm widgets
      over the normal ones
- **R25** The format document:
    - holds every fact of "What was observed on the device" above, marked **observed on the device** with the date
      2026-11-17: that no byte that was tried gave a threshold colour, that no value hides a number that showed one,
      the theme of the two widgets, the transparent `-` as wide as the box, the three states and what the screen
      showed, that the screen keeps its widgets as layers in the order of their slots, and that an opaque cover hid
      every layer below it
    - its open question on the order in which the device draws the records names what this observation established
      and what stays open
- **R26** The README:
    - in "Writing a theme", gives an example of an alarm number, such as
      `{type: number, font: value, metric: gpu.temp, x: 176, y: 44, alarm: {above: 80, color: "#ff3030"}}`, and a
      paragraph that says what `above` and `below` do, that the value is compared as it is sent, whole, that the number
      is left-aligned and cannot show negative values or thousands, that it counts as two widgets, that `theme preview`
      shows its normal colour, and links to the folder document
    - shows the form of a metric with an alarm pair in the tags file, and in the section of the key `tags` says that a
      metric may map to a mapping with `tag` and `alarms`, as the tags file writes it

### Tests

- **R27** The unit tests cover at least:
    - R1 to R6: an alarm number with `above`, with `below` and with both; each error of R3 to R5; `align: left`
      accepted; a number without `alarm` giving the same bytes as before
    - R7 to R10: a theme with a widget before and after an alarm number, compiled and read back with the reader of
      the theme file, with both records checked byte for byte, their slots, indexes, offsets and tags, both glyph sets
      equal, the `-` glyph's width and its mask all zero, and the other eleven glyphs equal to those of a number without
      `alarm` of the same font
    - R11: 62 records accepted with an alarm number among them, 63 refused with each form of the error
    - R13 and R14: a plain entry and two pairs of one metric, two alarm numbers sharing a pair, a pair that straddles
      tag 21, the order independent of the order of the widgets, and the tags of a theme without alarm unchanged
    - R15 to R18: the tags file of R17 byte for byte, parsed back to the tags of the compiler; each new form of the key
      `tags` accepted and each error of R16
    - R2 and R19 to R21: the pair sent across the threshold upward and back downward with `above`, and with `below`;
      a value equal to the threshold; a reading of 80.4 and 80.5 with `above: 80`; a metric with no value sending
      nothing to either tag; a plain entry and a pair of the same metric in one round
    - R22: the preview of an alarm number drawn with the normal colour at a pixel of its value and no pixel of the
      alarm colour
- **R28** The end-to-end tests:
    - compile a theme with an alarm number and check the tags file and the two records read back with `theme dump`
    - run `run` against the fake screen with the key `theme` naming a folder with an alarm number of a custom metric
      read from a file, and check the values the screen receives as the file goes from a normal value to one in the
      alarm state and back: the normal tag the value and the alarm tag `FF FF`, then the normal tag `FF FF` and the
      alarm tag the value, then as at first
- **R29** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 Two widgets and a transparent minus.** It is the one method seen to work on the device. The screen has no
  threshold colour, no value hides a number, and a cover hides every layer below it. Sending -1 to the widget that is
  not shown hides it, because its transparent minus fills the box and pushes the digits out of it. Rejected: covers,
  which failed on the device; an opacity of 0, which is a byte of the record and cannot change with the value
- **D2 `above` and `below`, alone or together.** `above` is the common case, a temperature too high; `below` is a pump
  or fan speed too low. Both together cost one more comparison and no more widgets, and they make a band, such as a
  voltage that is too high or too low. One colour serves both, because there is one alarm widget. Rejected: a list of
  thresholds with a colour each, which needs a widget for each colour and is a guess about a need nobody stated
- **D3 Strict comparison on the value as sent.** The value compared is the whole number the screen shows, so a red
  number never shows a value that the threshold calls normal, as a red 80 with `above: 80` would when 80.4 were compared
  before rounding. `above` and `below` read as strictly greater and strictly less, and since the thresholds are whole
  numbers, "80 or more" is written `above: 79`. Decimal thresholds are refused, because the value compared is whole,
  so a decimal threshold says nothing that a whole one does not. Rejected: `>=`, which reads against the word;
  comparing the reading before rounding, which can disagree with what the screen shows
- **D4 The alarm changes the colour and not the font.** The request is a colour, and a colour is all the observation
  covers. The two widgets share the font, so they have the same box and the same glyphs, and the alarm widget differs
  in its colour and its tag alone. A font of its own is item 2 of "Beyond the request"
- **D5 `align` is refused and not emulated.** The method needs the left alignment: a right-aligned negative value puts
  its digits at the right edge of the box, where they show, and a centred one shows part of them. Emulating `right` or
  `center` would need the screen to place a value of varying width in a left-aligned box, which nothing the compiler
  writes can do, since the glyphs of the digits have one width and the value's length varies. So an alarm number is
  left-aligned, and without the key it is `left` rather than the `center` of other numbers, so that the common case
  needs no key. Rejected: requiring `align: left`, which makes every alarm number repeat what is the only possible
  value
- **D6 `thousands` is refused.** The thousands mode was never tried with a negative value, so whether -1 shows a minus,
  and so whether the widget is hidden, is unknown. Refusing it costs nothing that was asked for
- **D7 Negative values are documented, not refused.** The compiler cannot know whether a metric can be negative: a
  custom metric is defined in the config file, which the compiler does not read, and the built-in metrics are not
  negative in practice. `run` cannot know it before a value arrives. So the documents say that an alarm number shows
  nothing for a negative value, and the author of a metric that can be negative uses a number without `alarm`, or a
  custom metric with an `offset`
- **D8 The hiding value is -1.** It was the value seen to hide a widget, and any negative value works the same way
  under the method. Fixing one value makes the bytes sent testable
- **D9 A pair of tags per metric and threshold.** The normal widget of an alarm number receives -1 in the alarm state,
  so it cannot share the tag of a number without `alarm` of the same metric, which must show the value always. Two
  alarm numbers with the same metric and thresholds receive the same values in every round, so they share a pair, and
  their colours do not matter to the values. The sort of R13 keeps the tags of every theme without an alarm number as
  they are, so that no tags block that a user pasted goes out of date with this unit
- **D10 The pair is carried in the value of a metric in `tags`.** Every block valid today keeps its meaning, a metric
  stays one key, and a pasted block can say everything the compiler assigns: plain tags, pairs, and both for one
  metric. The flow style keeps one line per metric, as today. Rejected: a key `alarms` of the config beside `tags`,
  which splits the mapping of one theme in two places; the threshold in the metric's key, such as `gpu.temp>80`, which
  is not a metric name and needs a grammar of its own; a mapping from tag to metric, which would break every block
  that exists
- **D11 Two copies of the glyph set.** Two records sharing one resource is known for images in the vendor's default
  theme and was not tried for numbers. Each record gets its own copy, as in a theme without alarm, which costs the size
  of one glyph set, a few tens of kilobytes for a large font, within the 4 MiB. Sharing is item 4 of "Beyond the
  request"
- **D12 The preview shows the normal state.** It is the state the screen shows most of the time, and a single picture
  of 42 cannot be both. `theme preview` knows which records form a pair from the compiler, so it draws the alarm widget
  with -1 by the rules the preview already has, which draws nothing for it, as the screen does. The dump reads a file
  that does not say which records form a pair, so it draws every number as 42, and the documents say what that shows.
  A way to draw the alarm state is item 1 of "Beyond the request": the alarm state differs from the normal one in the
  colour alone, which the author wrote
- **D13 A pair may cross two reports.** A value update carries 20 pairs, and a pair whose two tags fall in two
  reports is shown with both widgets or with neither for the time between two writes of one round. That is not worth a
  change of the encoding for this unit, and it is item 6 of "Beyond the request"
- **D14 The threat model is unit 0011's.** A theme folder may come from anyone, and D15 of unit 0011 holds. The new
  keys are scalars checked as R3 says, and the boxes are checked against the screen as every box is. The config file is
  the user's own, as it is today
- **D15 No device check is waited for.** That a theme compiled by sensordeck with an alarm number switches colour on
  the screen when the value crosses the threshold is the human's to check, and the unit is complete without it. A
  finding of that check is a new unit

## Non-goals

- A threshold colour of a bar, a text or a datetime, and anything with covers
- Showing a negative value in an alarm number (D7)
- `align: center` or `right` and `thousands` on an alarm number (D5, D6)
- More than one alarm colour on one number, and a font of its own for the alarm
- Hysteresis, a delay, or blinking: the state is decided anew from each round's value
- A change of the built-in themes, whose bytes and previews stay as they are
- Recognising the pairs of a theme file in `theme dump`
- The check on the device, which is the human's

## Beyond the request

1. A flag `--alarm` of `theme preview` that draws the alarm state: the normal widgets with -1 and the alarm widgets
   with the sample value. Cost: a flag, its test and a line in the README and the folder document. Without it the
   preview never shows the alarm colour, which the author wrote and sees on the screen when the threshold is crossed,
   or at once with a custom metric read from a file
2. A key `font` in `alarm`, so that the alarm shows in another font, such as a bold one. Cost: two boxes of different
   widths on the same spot, the glyph checks of a second font, the documents and tests. Without it only the colour
   changes
3. Hysteresis, such as `above: 80` with a `clear: 75` below which the alarm ends. Cost: `run` keeps the state of each
   pair from round to round and across reloads, a key, its errors and tests. Without it a value that wanders across
   the threshold changes colour from round to round, which the user sees and can move the threshold for
4. One glyph set shared by the two records of an alarm number. Cost: a change of how resources are laid out, and a
   check on the device that numbers may share a resource. Without it each alarm number takes one more glyph set of the
   flash, a few tens of kilobytes
5. Refusing `below` greater than or equal to `above`, with which the alarm holds for every value but at most one.
   Cost: an error and a test. Without it the number shows in the alarm colour always, which the author sees and fixes
6. Keeping both tags of a pair in one report of the value update. Cost: a change of how `run` groups the pairs into
   reports and a test. Without it a pair split over two reports shows both widgets or neither for the time between two
   writes of a round, and only in a round that sends more than 20 values
