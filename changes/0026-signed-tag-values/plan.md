# Plan: sending the value of a tag as a signed 16-bit number

## Approach

The type of the value of a pair carries the rule. `smartmonitor.Value.Value` becomes an `int16`, and
`encodeValueReport` writes it as the 16-bit pattern of that number, `uint16(value.Value)`, high byte first, which is
the two's complement of a negative value and leaves the bytes of 0 to 32767 as they are (R2). `sensors.ToValue`
returns an `int16`, rounded with `math.Round` as today and clamped to -32768 to 32767 (R1). `run` already sends every
reading, built-in and custom, through `ToValue`, so with the type changed it sends the signed value with no change of
its own (R3).

`values` reads its value with a parser of its own, `parseValue`, beside the `parseNumber` that the tag and `--seconds`
keep: it takes at most one `-` off the front, reads the rest with `parseNumber`, refuses a magnitude above 32768 after
a `-` and above 65535 without one with the new error of the range, and gives the `int16` that R5 names: the negation
for a negative value, and `int16(uint16(magnitude))` for 0 to 65535 (R4, R5). `-0` is 0 and `-0x8000` is -32768. The
tag still goes through `parseNumber` alone, so `-1=5` stays refused, here and by the parser of the command line (D3).

What the tree on `bugfix-0026-signed-tag-values` holds, and what the plan builds on:

* `internal/sensors/convert.go` holds `ToValue(value float64) uint16`, which clamps to 0 to 65535, and
  `convert_test.go` its table of cases with `want uint16`
* `internal/smartmonitor/values.go` holds `Value{Tag uint8; Value uint16}` and `encodeValueReport`, which writes
  `byte(value.Value >> 8)` and `byte(value.Value)`. `values_test.go` builds pairs with `uint16(index * 3)`
* `internal/cli/arguments.go` holds `parseValues`, which reads the tag and the value with `parseNumber` and refuses a
  value above 65535 with `the value is outside 0 to 65535`, `parseNumber`, which gives the largest `uint64` for a
  number too large for 64 bits, and `allDigits`. `arguments_test.go` has `TestParseValues` and
  `TestParseValuesErrors`, four of whose cases give the old error of the range
* `internal/cli/run.go` line 477 appends `smartmonitor.Value{Tag: mapping.Tag, Value: sensors.ToValue(value)}`, and
  `internal/cli/run_test.go` line 512 builds a value with `uint16(round + 1)`
* `e2e/commands_test.go` has `TestValuesExample` and `TestWrongValuesArgumentWithNoScreen`. `e2e/run_test.go` has
  `valuePairs`, which decodes a report into `map[uint8]uint16` of the raw patterns, and `waitForReport`.
  `e2e/custom_metrics_test.go` has `TestRunSendsCustomMetricsToTheTagsOfTheTheme`, which compiles a theme folder,
  reads the tags of its metrics from the tags file with a local `tagOf`, uploads it and waits for a report
* `docs/protocol.md` says 16 bits unsigned in the table of the value update. `README.md` says at line 295 that values
  are sent as whole numbers, rounded, and at line 419 that the screen shows whole numbers from 0 to 65535 and that a
  negative value shows 0. The README says nothing yet about the range of `sensordeck values`

The frozen specs of the units 0002, 0003, 0004, 0007, 0011 and 0024 do not change; this unit supersedes 0004 and
names the others in its spec.

## Files

- `internal/sensors/convert.go`, `internal/sensors/convert_test.go`: `ToValue` gives an `int16` clamped to the signed
  range
- `internal/smartmonitor/values.go`, `internal/smartmonitor/values_test.go`: `Value.Value` is an `int16`, written as
  its 16-bit pattern
- `internal/cli/arguments.go`, `internal/cli/arguments_test.go`: `parseValue` and the new range of `values`
- `internal/custommetrics/number_test.go`: the table of `TestValuesAreSentAsTheScreenTakesThem` follows the new type
  and range of `ToValue`
- `internal/cli/run_test.go`: the conversion at line 512 follows the type
- `e2e/commands_test.go`: the example of R6 and the error on the range with no screen
- `e2e/custom_metrics_test.go`: a helper that compiles a theme folder and gives the tag of a metric, shared by the
  existing test and a new one for a custom metric of -5 and one of 40000
- `docs/protocol.md`: the signed value, its example and the observations of 2026-11-17
- `README.md`: the clamp, the negative values, the note on custom metrics, and what `values` takes

## Tasks

### The value of a reading and its encoding

- [x] In `internal/sensors/convert_test.go`, change the table of `TestToValue` to `want int16` and to exactly the
  examples of R1: 51.5 gives 52, 51.49 gives 51, 0.5 gives 1, -0.4 gives 0, -0.5 gives -1, -5 gives -5, -51.5 gives
  -52, 32767.4 gives 32767, 32767.5 gives 32767, 40000 gives 32767, 70000 gives 32767, -32768.4 gives -32768,
  -32768.5 gives -32768, -40000 gives -32768, `math.Inf(1)` gives 32767 and `math.Inf(-1)` gives -32768, and keep 0
  giving 0 (R1, R11)
- [x] In `internal/smartmonitor/values_test.go`, add `TestEncodeValuesSigned`: one report of the pairs (1, -1),
  (2, -5), (3, -32768), (4, 32767) and (5, 0) gives `02 05 01 FF FF 02 FF FB 03 80 00 04 7F FF 05 00 00` followed by
  `0x00`. The reading of -0.4 is the value 0 by R1, which this list encodes as `00 00`. Change `pairs` to
  `int16(index * 3)` (R2, R11)
- [x] In `internal/smartmonitor/values.go`, make `Value.Value` an `int16`, with a comment that the screen reads it as
  a two's complement number (`docs/protocol.md`), and write `bits := uint16(value.Value)` high byte first in
  `encodeValueReport` (R2)
- [x] In `internal/sensors/convert.go`, make `ToValue` return an `int16`: `math.Round`, then `math.MinInt16` when the
  rounded value is at or below it, `math.MaxInt16` when at or above it, and `int16(rounded)` otherwise. The comment
  says rounded with halves away from zero and clamped to -32768 to 32767. A rounded -0 converts to 0 (R1, D5)
- [x] In `internal/cli/run_test.go` line 512, write `int16(round + 1)`. Check with `go build ./...` and
  `GOOS=windows GOARCH=amd64 go build ./...` that `run.go` and every other user of `Value` and `ToValue` compile
  unchanged, which is what R3 needs of `run` (R3)

- [x] In `internal/custommetrics/number_test.go`, change `TestValuesAreSentAsTheScreenTakesThem` to `int16` with -5 giving
  -5 and 70000 giving 32767, since it calls `ToValue` and did not compile with the new type (R1, R3)

### The `values` command

- [ ] In `internal/cli/arguments_test.go`, add to `TestParseValues` the cases of R12 with the `int16` that R5 gives:
  `1=-1` gives -1, `1=-32768`, `1=-0x8000`, `1=32768` and `1=0x8000` give -32768, `1=65535` and `1=0xFFFF` give -1,
  `1=-0` gives 0, `1=32767` gives 32767; and the existing `0x1F=0XFFFF` now wants `Value: -1`. Add a test that
  `parseValues([]string{"1=-1", "2=32767"})` encoded with `smartmonitor.EncodeValues` gives
  `02 02 01 FF FF 02 7F FF` followed by `0x00`, and that `1=65535` gives the same bytes as `1=-1` (R5, R12)
- [ ] In `TestParseValuesErrors`, give the four existing cases of the range the text
  `the value is outside -32768 to 65535`, and add: `1=-32769`, `1=-0x8001`, `1=65536` and `1=-` followed by thirty
  `9`s, and `1=-18446744073709551616`, both negative numbers too large for 64 bits, refused with that error; `1=-`, `1=--1`, `1=+1`, `1=- 1` and `1=-0x` refused
  with `want <tag>=<value>`. Keep `-1=1` refused with `want <tag>=<value>` (R4, R5, R12, D3)
- [ ] In `internal/cli/arguments.go`, add `parseValue(text string) (value int16, isNumber, inRange bool)`, with a
  comment, that cuts one leading `-`, reads the rest with `parseNumber`, gives not a number when the rest is not one,
  and refuses a magnitude above 32768 with a `-` or above 65535 without one. `parseValues` uses it for the value and
  `parseNumber` for the tag, and the error is
  `invalid argument "<argument>": the value is outside -32768 to 65535`. The order of the checks stays: not a number,
  then the tag, then the value (R4, R5)

### End-to-end

- [ ] In `e2e/commands_test.go`, add `TestValuesWithANegativeValue`: `values 1=-1 2=32767` against a fake screen gives
  exit status 0, `values sent: 2, reports written: 1` and the 65 bytes `00 02 02 01 FF FF 02 7F FF` followed by 56
  bytes of `0x00`; and `TestValuesOutsideTheRangeWithNoScreen`: `values 1=65536` with no screen gives exit status 1,
  no standard output, and
  `sensordeck: invalid argument "1=65536": the value is outside -32768 to 65535` on standard error (R6, R13)
- [ ] In `e2e/custom_metrics_test.go`, move the compiling of the theme folder, the reading of the tags file with
  `tagOf` and the reading of the compiled file out of `TestRunSendsCustomMetricsToTheTagsOfTheTheme` into a helper,
  such as `compileTheme(test, setup, folder) (compiled []byte, tagOf func(string) uint8)`, with no change of what the
  test checks, and run the suite (R13)
- [ ] Add `TestRunSendsNegativeAndClampedCustomMetrics`: a setup whose config has two file metrics, `below` reading a
  file that holds `-5` and `above` reading one that holds `40000`, and a theme folder with a number widget for each;
  after the upload, a value report holds the tag of `below` with the pattern `0xFFFB` and the tag of `above` with
  `0x7FFF`, as `valuePairs` decodes them (R3, R13)

### The documents

- [ ] In `docs/protocol.md`, the table of the value update says the value is 16 bits signed, two's complement, -32768
  to 32767, high byte first; add the example of tag 1 with the value -1, `02 01 01 FF FF`, beside the example of
  `0x1234` and `0x0001`, which stays; add the observations of 2026-11-17 as the table of the spec gives them, the five
  values sent with their bytes and what the screen showed, that the screen draws the minus sign with the `-` glyph of
  the number record, and that the box of a widget can clip it (R7)
- [ ] In `README.md` at the paragraph of line 295, say that values are sent as whole numbers, rounded and clamped to
  -32768 to 32767, that a negative value shows with its minus sign, and that a reading above 32767, such as a network
  rate of 40000 KiB/s, shows 32767 while `sensordeck sensors` shows the reading (R8)
- [ ] In `README.md` at line 419, say that the screen shows whole numbers from -32768 to 32767, as for every metric,
  so that `run` rounds the value, a negative value shows with its minus sign, and tenths are shown by a `scale` ten
  times larger, and remove that a negative value shows 0 (R9)
- [ ] In `README.md`, after the block of commands of `## Commands`, add a paragraph on what `sensordeck values` takes:
  `<tag>=<value>` with a tag of 0 to 255, a value in decimal or in hexadecimal with `0x`, optionally after a `-`, of
  -32768 to 32767 for the number the screen shows, and of 32768 to 65535 as the raw 16-bit pattern, which the screen
  shows as that value minus 65536, so `-1` and `65535` send the same bytes (R10, D2)
- [ ] Run `grep -rn "unsigned\|65535" README.md docs/` and confirm that no remaining text says that a value is
  unsigned or that its range is 0 to 65535, apart from the observation of 65535 in `docs/protocol.md` and the raw
  range of `values`; the tables of metrics stay as they are (R10)

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [ ] `npm run check-changes` passes after the last commit

## Commit Points

- [ ] `bugfix-0026 Send the value of a reading as a signed 16-bit number`: the tasks of "The value of a reading and
  its encoding"
- [ ] `bugfix-0026 Take negative values in values`: the tasks of "The `values` command" and the first task of
  "End-to-end"
- [ ] `bugfix-0026 Share the compiling of a theme in the tests of custom metrics`: the move to the helper alone, with
  no change of behavior
- [ ] `bugfix-0026 Test a negative and a clamped custom metric end to end`: the last task of "End-to-end"
- [ ] `bugfix-0026 Say in the documents that the value is signed`: the tasks of "The documents"
