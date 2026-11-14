# Device protocol

## Approach

The unit adds the Go module and one package, `internal/smartmonitor`, which encodes reports, frames YMODEM and drives
a theme upload over a transport interface. Nothing in it does I/O of its own: the transport and the pause function
come from the caller, and the tests supply fakes for both. The protocol document and the suite description in
`AGENTS.md` come first, so that the code is written against a document of the project and not against the spec
alone.

Each source file has one job, and each is driven by its tests, written first:

* `report.go` holds the type `Report`, a `[64]byte`, and the one function that cuts a byte sequence into reports
  (R7, R18). It is the only place that decides how a block is carried, as D5 asks
* `values.go` encodes value updates, `clock.go` the clock, and `commands.go` holds the reset and ymodem command
  reports (R8 to R12)
* `ymodem.go` holds the CRC and the header, data, end of transmission and closing blocks as byte slices (R13 to R17)
* `upload.go` holds the `Transport` interface, the named constants of D10, the error type of R26 and the upload
  itself (R19 to R27)

The exported names below are what the next unit's transport and command line will call. They are fixed here so that
the implementer does not choose them:

```go
type Report [64]byte

type Value struct {
	Tag   uint8
	Value uint16
}

func EncodeValues(values []Value) []Report
func EncodeClock(moment time.Time) (Report, error)
func ResetCommand() Report
func YmodemCommand() Report

type Transport interface {
	Write(report Report) error
	Read(timeout time.Duration) (Report, error)
	Reopen() error
}

type Stage int // with one constant per stage of R26, and a String method naming it
type UploadError struct {
	Stage Stage
	Block int   // the sequence number of the data block, 0 at every other stage
	Err   error // the cause; Unwrap returns it
}

func Upload(transport Transport, theme []byte, pause func(time.Duration)) error
```

The framing functions stay unexported, since only the upload uses them: `crc16`, `headerBlock(size int) []byte`,
`dataBlocks(theme []byte) [][]byte`, `endOfTransmission() []byte`, `closingBlock() []byte` and
`toReports(data []byte) []Report`. Tests in the same package reach them.

The stages are, in this order, named by `String` as given: `StageReset` "reset", `StageReopenAfterReset` "reopening
after the reset", `StageYmodemCommand` "ymodem command", `StageHeader` "header block", `StageData` "data block",
`StageEndOfTransmission` "end of transmission", `StageClosing` "closing block", `StageReopenAfterUpload`
"reopening after the upload". `UploadError.Error` gives `upload failed at the <stage>: <cause>`, and for a data block
`upload failed at data block <n>: <cause>`.

The upload is built from three steps that each take the stage and block number and return an `*UploadError` or nil:

* `sendItem(transport, reports, stage, block)`: writes the reports in full, reads one answer with the time limit of
  10 seconds, and on NAK writes them again, up to 10 writes in all (R21, R22, R23). An error of a write or a read is
  wrapped as the cause and ends the step; an answer that is neither ACK nor NAK ends it with a cause that states the
  byte in hexadecimal, as `unexpected answer 0x43`; the tenth NAK ends it with a cause that says the device refused
  the item 10 times
* `writeOnly(transport, reports, stage)`: writes the reports of the closing block and reads nothing (D9)
* `reopen(transport, pause, stage)`: pauses 1 second, calls `Reopen`, and on any error pauses 500 milliseconds and
  calls it again, up to 20 calls; after the twentieth failure the cause wraps the error of the last call (R24, D13)

The named constants are `reopenFirstPause = time.Second`, `reopenRetryPause = 500 * time.Millisecond`,
`reopenAttempts = 20`, `answerTimeout = 10 * time.Second` and `maxWrites = 10`. ACK is `0x06` and NAK `0x15`.

`Upload` orchestrates the steps in the order of R20 and returns the first error. It reads the theme only through
slicing and `copy`, so it never changes it (R27). All the errors returned are `*UploadError`, so `errors.As` finds the
stage and block, and since the transport's error is the cause, `errors.Is` reaches it (R26).

The fake transport of the tests, in `upload_test.go`, holds a script of answers (each a report or an error) and a
script of reopen results, and records every call as an entry of kind write, read or reopen with the report written
or the time limit passed. The pause function of the tests records each duration and returns at once (R6). A read
past the end of the script fails the test, so that an unexpected read is caught.

`go.mod` is written by hand, because `go mod init` of Go 1.27 does not write the directive `go 1.27` that R4 asks
for. It has no `toolchain` line.

## Files

- `docs/protocol.md`: new, the wire protocol, the unknowns and the assumptions
- `AGENTS.md`: a subsection under "Specific to this project" on how the project is tested
- `go.mod`: new, module `github.com/norareit/sensordeck`, `go 1.27`
- `internal/smartmonitor/doc.go`: new, the package comment
- `internal/smartmonitor/report.go`, `report_test.go`: the report type and `toReports`
- `internal/smartmonitor/values.go`, `values_test.go`: the value update
- `internal/smartmonitor/clock.go`, `clock_test.go`: the clock
- `internal/smartmonitor/commands.go`, `commands_test.go`: the reset and ymodem commands
- `internal/smartmonitor/ymodem.go`, `ymodem_test.go`: the CRC and the blocks
- `internal/smartmonitor/upload.go`, `upload_test.go`: the transport, the errors and the upload
- `internal/smartmonitor/imports_test.go`: new, the check of R5 on the imports of the package

## Tasks

### Documents and module

- [ ] Write `docs/protocol.md` in the project's own words, with a section each for the identity, the nature of the
  device, the transport, the value update, the clock, the five steps of the theme upload, the YMODEM block and the
  CRC with its check value `0x31C3`, keeping every byte value of "What is known of the device" in the spec (R1)
- [ ] Add to `docs/protocol.md` a section "Unknown" that lists the brightness, reading back device information and
  the firmware version, what the device sends unasked, the touch layer, and the meaning of `0x01 0x15` at offsets 1
  and 2 and of `0x64` at offset 10 of the clock report; a section "Assumptions" that states each of D5 to D10 as not
  yet confirmed on the device, with the numbers of D10; and a sentence that the protocol was learned from the
  project the README credits and that this document holds no code or text of it (R2)
- [ ] Add a subsection "Testing" under "Specific to this project" in `AGENTS.md` that says `go test ./...` passes,
  `go vet ./...` reports nothing and `gofmt -l .` prints no file name, that these three are the whole suite, and
  that there is no end-to-end suite yet because there is no executable (R3)
- [ ] Write `go.mod` holding only `module github.com/norareit/sensordeck` and `go 1.27`, and add no `go.sum` (R4)
- [ ] Write `internal/smartmonitor/doc.go` with the package comment: the package speaks the wire protocol of the
  SmartMonitor screen `0483:0065`, described in `docs/protocol.md`, and does no I/O of its own (D3)
- [ ] Commit: `feature-0002 Document the protocol and add the Go module`

### Reports and encoders

- [ ] In `report_test.go`, test `toReports`: 133 bytes give 3 reports, the last holding bytes 128 to 132 and 59 of
  `0x00`; 1029 bytes give 17; 64 give 1; 1 gives 1 with 63 of `0x00`; the pieces are in order (R18, R28)
- [ ] In `report.go`, define `Report` as `[64]byte` and write `toReports` so the tests pass (R7, R18)
- [ ] In `values_test.go`, test `EncodeValues`: the example of R8 byte for byte; 41 pairs give counts 20, 20 and 1
  with the pairs in list order; 40 pairs give 20 and 20; an empty list and nil give no report; a copy of the input
  taken before the call equals the input after it; a tag given twice is encoded twice (R8, R9, R28)
- [ ] In `values.go`, define `Value` and write `EncodeValues` so the tests pass (R8, R9)
- [ ] In `clock_test.go`, test `EncodeClock`: 2026-11-14 15:04:05 in `time.UTC` gives the bytes of R10; a time in a
  fixed zone made with `time.FixedZone` is encoded with its own fields and not converted; a Sunday gives `0x07` and a
  Monday `0x01` at offset 9; the years 1999 and 2256 give an error whose text holds the year and a zero `Report`;
  the years 2000 and 2255 give no error (R10, R11, R28)
- [ ] In `clock.go`, write `EncodeClock` so the tests pass, mapping `time.Sunday` to 7 (R10, R11, D16)
- [ ] In `commands_test.go`, test that `ResetCommand` and `YmodemCommand` give the reports of R12 (R12, R28)
- [ ] In `commands.go`, write `ResetCommand` and `YmodemCommand` so the tests pass (R12, D6)
- [ ] In `imports_test.go`, list every `.go` file of the package directory, tests included, with
  `filepath.Glob("*.go")`, parse each with `parser.ParseFile` in `parser.ImportsOnly` mode, and fail on any import
  path whose first element holds a dot, which is not the standard library, and on any of `os`, `os/exec`,
  `os/signal`, `net`, `syscall` and `unsafe`. Neither call needs the test to import `os` (R5)
- [ ] Commit: `feature-0002 Encode reports, value updates, the clock and the commands`

### YMODEM framing

- [ ] In `ymodem_test.go`, test `crc16`: `0x31C3` for `123456789`, `0x0000` for no bytes and for 128 bytes of `0x00`
  (R13, R28)
- [ ] In `ymodem_test.go`, test `headerBlock(2500)`: 133 bytes, `01 00 FF`, the data starting with
  `69 6D 67 2E 64 61 74 00 32 35 30 30 00` and `0x00` after it, and the last two bytes `82 BF` (R14, R28)
- [ ] In `ymodem_test.go`, test `dataBlocks`: for the theme of R16 three blocks of 1029 bytes starting with `02 01 FE`,
  `02 02 FD` and `02 03 FC`, the CRC bytes `C2 E0`, `C2 E0` and `FC 8B`, the third holding 452 theme bytes and 572 of
  `0x1A`; for a theme of 2048 bytes two blocks and no third; for a theme of 257 times 1024 bytes the bytes at offsets
  1 and 2 of blocks 255, 256 and 257 are `FF 00`, `00 FF` and `01 FE`; the theme is unchanged after the call (R15,
  R16, R27, R28)
- [ ] In `ymodem_test.go`, test that `endOfTransmission` is the single byte `0x04` and that `closingBlock` is 133
  bytes, `01 00 FF`, 128 of `0x00` and `00 00` (R17, R28)
- [ ] In `ymodem.go`, write `crc16`, `headerBlock`, `dataBlocks`, `endOfTransmission` and `closingBlock` so the tests
  pass, with a shared unexported helper that frames a start byte, a block number and data into a block (R13 to R17,
  D7, D14)
- [ ] Commit: `feature-0002 Frame YMODEM blocks with their CRC`

### Theme upload

- [ ] In `upload_test.go`, write the fake transport and the recording pause described under Approach, neither of
  which waits (R6, R29)
- [ ] Test a complete upload of the theme of R16 against a transport that answers ACK to everything and reopens at
  once: the recorded calls equal, in order, the write of the reset report, a reopen, the ymodem report and a read,
  the 3 reports of the header block and a read, for each of the 3 data blocks its 17 reports and a read, the one
  report of the end of transmission and a read, the 3 reports of the closing block, and a reopen; every report
  written equals what `toReports` gives for that item; every read is passed 10 seconds; the pauses are two of
  1 second; the error is nil; the theme is unchanged (R19, R20, R25, R27, R29)
- [ ] Test NAK and then ACK on data block 2: its 17 reports are written twice, identical, with a read after each, and
  the upload completes (R21, R29)
- [ ] Test NAK and then ACK on the ymodem command, on the header block and on the end of transmission: the item is
  written again and the upload completes (R21)
- [ ] Test ten NAKs on data block 2: the item is written 10 times, nothing is written after the tenth read, and
  `errors.As` gives an `*UploadError` with `StageData` and block 2, whose text names the data block and the number 2
  (R22, R26, R29)
- [ ] Test a read that returns an error standing for the passed time limit on the header block: `errors.As` gives
  `StageHeader` and block 0, `errors.Is` reaches the read's error, and nothing is written after the read (R23, R26,
  R29, D11)
- [ ] Test an answer of `0x43` on data block 1: the error has `StageData` and block 1, its text holds `0x43`, and
  nothing is written after the read (R23, R29, D12)
- [ ] Test a write that returns an error, once on the reset and once on the closing block: the error has the stage
  of the item, `errors.Is` reaches the write's error, and no call follows the failing write (R23, R26, R29)
- [ ] Test reopening that fails three times and then succeeds, after the reset and after the upload: the upload
  completes, and the pauses after each reset are 1 second followed by three of 500 milliseconds (R24, R29)
- [ ] Test reopening that fails 20 times after the reset: 20 reopen calls, the pauses 1 second and 19 of
  500 milliseconds, nothing written after, and an error with `StageReopenAfterReset` that reaches the twentieth
  reopen error with `errors.Is`. Test the same after the upload with `StageReopenAfterUpload` (R24, R26, R29)
- [ ] Test that `Stage.String` gives the name of each of the eight stages and that the text of an error of every
  stage but the data block names its stage (R26)
- [ ] In `upload.go`, write `Transport`, `Stage`, `UploadError` with `Error` and `Unwrap`, the constants, `sendItem`,
  `writeOnly`, `reopen` and `Upload` as described under Approach so the tests pass (R19 to R27, D9 to D13, D15, D17)
- [ ] Commit: `feature-0002 Upload a theme over a transport`

## Verification

- [ ] `go test ./...` passes (R5, R28, R29)
- [ ] `go vet ./...` reports nothing (R3)
- [ ] `gofmt -l .` prints no file name (R3)
- [ ] `go list -deps ./internal/smartmonitor` lists only standard library packages, and there is no `go.sum` (R4, R5)
- [ ] `grep -rn 'time.Sleep\|time.After\|time.NewTimer\|time.Tick' internal/` finds nothing (R6, R25)
- [ ] `npm run check-changes` passes on the committed work

## Commit Points

- [ ] `feature-0002 Document the protocol and add the Go module`
- [ ] `feature-0002 Encode reports, value updates, the clock and the commands`
- [ ] `feature-0002 Frame YMODEM blocks with their CRC`
- [ ] `feature-0002 Upload a theme over a transport`
