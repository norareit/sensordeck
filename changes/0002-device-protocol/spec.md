# Device protocol

## Context

sensordeck is to drive a small USB screen, and nothing of it exists yet but the README, the licence and the agent
kit. Before any device can be opened, the bytes that go over the wire have to be right. This unit writes down the
wire protocol of the one device version 1 targets, and builds the part of the program that produces those bytes and
drives a theme upload, against a transport that is only an abstraction. It adds the Go module with it.

The unit is built in a sandbox that holds only what is committed. It has no USB device and reaches no network but
the Go module proxy. Everything known about the protocol therefore stands in this spec, and every requirement can be
checked by automated tests without hardware. The unit that follows adds the Linux transport and a first command
line, and is where the protocol is confirmed on the real device.

### What is known of the device

This is the whole of what is known. Nothing else may be assumed, except where a decision below says so.

**Identity.** USB ID `0483:0065`, manufacturer string `llhmi.com`, product string `llTechCo.,Ltd`. It is sold as
"3.5 inch PC CPU Data Monitor" and as SmartMonitor. The screen is 480 by 320 pixels.

**Nature.** The device is a theme player, not a framebuffer. A theme holds the layout, the images, the fonts and
widgets that are bound to numeric tags. It is uploaded once to the 4 MB flash of the device. Afterwards the computer
sends only small value updates and the clock. A theme upload takes about 10 seconds and reboots the device.

**Transport.** USB HID at full speed, one interface, two interrupt endpoints: `0x81` IN and `0x02` OUT. The computer
sends HID output reports of 64 bytes, and the device answers with input reports of 64 bytes. On Linux the device is
reached through `/dev/hidraw*`, where a report is written as the report ID byte `0x00` followed by the 64 bytes. The
report ID byte belongs to the transport and is no part of the 64 bytes this spec speaks of.

**Value update.** One report sets up to 20 values:

| Offset | Size | Content |
|---|---|---|
| 0 | 1 | `0x02` |
| 1 | 1 | the number of pairs in this report, 1 to 20 |
| 2 | 3 for each pair | the tag, one byte, then the value, 16 bits unsigned, high byte first |
| after the pairs | the rest | `0x00` up to 64 bytes |

**Clock.** One report sets the clock:

| Offset | Content |
|---|---|
| 0 | `0x03` |
| 1 | `0x01` |
| 2 | `0x15` |
| 3 | the year minus 2000 |
| 4 | the month, 1 to 12 |
| 5 | the day of the month |
| 6 | the hour, 0 to 23 |
| 7 | the minute |
| 8 | the second |
| 9 | the ISO weekday, 1 for Monday to 7 for Sunday |
| 10 | `0x64` |
| 11 to 63 | `0x00` |

What `0x01 0x15` at offsets 1 and 2 and `0x64` at offset 10 mean is not known.

**Theme upload.** The steps, in order:

1. The computer sends the six bytes `0x01 0x72 0x65 0x73 0x65 0x74`, which is `0x01` followed by the ASCII text
   `reset`. The device reboots and enumerates again, which takes about 2.5 seconds. The computer opens it again
2. The computer sends the six ASCII bytes `ymodem` and waits for the answer ACK, `0x06`
3. The computer sends a YMODEM header block: start byte SOH, block number 0, and 128 data bytes that hold the file
   name `img.dat`, a `0x00`, the size of the theme in decimal ASCII, and a `0x00`. Then it sends the theme in blocks
   of 1024 data bytes with the start byte STX, the last one filled up with `0x1A`. Every block carries a
   CRC-16/XMODEM, high byte first. The device answers every block with ACK, `0x06`, or with NAK, `0x15`, on which
   the block is sent again
4. The computer sends EOT, `0x04`, and waits for ACK. Then it sends an empty block with number 0, which ends the
   session
5. The device reboots again. Opening it again needs several attempts, and an I/O error while it enumerates is normal

**YMODEM block.** This is the public layout of the YMODEM protocol:

| Offset | Size | Content |
|---|---|---|
| 0 | 1 | the start byte: SOH, `0x01`, for 128 data bytes, or STX, `0x02`, for 1024 data bytes |
| 1 | 1 | the block number |
| 2 | 1 | the one's complement of the block number, which is 255 minus the block number |
| 3 | 128 or 1024 | the data |
| after the data | 1 | the high byte of the CRC of the data |
| last | 1 | the low byte of the CRC of the data |

A block is therefore 133 or 1029 bytes long. The CRC is computed over the data bytes alone.

**CRC-16/XMODEM.** Polynomial `0x1021`, initial value `0x0000`, no reflection of input or output, no final XOR. The
CRC of the nine ASCII bytes `123456789` is `0x31C3`.

**Unknown.** How the brightness is set. How device information or the firmware version is read back. What the device
sends on the IN endpoint without being asked. Whether the touch layer is exposed.

## Requirements

"The package" is the Go package of D2. "A report" is a sequence of exactly 64 bytes, without a report ID byte.
Numbers in a `0x` form are bytes or 16-bit values in hexadecimal.

### Documents and module

- **R1** `docs/protocol.md` describes the wire protocol in the project's own words. It holds every fact of "What is
  known of the device" above: identity, nature, transport, the value update, the clock, the five steps of the theme
  upload, the YMODEM block layout and the CRC parameters with the check value, and every byte value named there
- **R2** `docs/protocol.md` has one section that lists everything under "Unknown" above together with the two
  unknown meanings in the clock report, and one section that lists every assumption of D5 to D10, each stated as not
  yet confirmed on the device. It says that the protocol was learned from the project the README credits, and it
  holds no code and no text taken from that project
- **R3** `AGENTS.md` documents, under "Specific to this project", how the project is tested: `go test ./...` passes,
  `go vet ./...` reports nothing, and `gofmt -l .` prints no file name. It says that these three are the whole
  suite and that there is no end-to-end suite yet, because there is no executable
- **R4** The root of the repository holds a `go.mod` with the module path `github.com/norareit/sensordeck` and the
  directive `go 1.27`. It has no `require` directive, and there is no `go.sum`
- **R5** The package and its tests import nothing but the Go standard library, and none of `os`, `os/exec`,
  `os/signal`, `net`, `syscall` and `unsafe`. The three commands of R3 succeed in a checkout that has no network and
  no device
- **R6** The tests of the package do not wait in real time: no test sleeps, and no test depends on a time limit
  passing

### Reports

- **R7** The package has a type for a report whose size is fixed at 64 bytes by the type, and every function of the
  package that produces or accepts a report uses it
- **R8** Encoding a list of 1 to 20 pairs of a tag, 8 bits unsigned, and a value, 16 bits unsigned, gives one report
  laid out as the value update above. As an example, the pairs (tag `0x01`, value `0x1234`) and (tag `0xFF`, value
  `0x0001`) give `02 02 01 12 34 FF 00 01` followed by 56 bytes of `0x00`
- **R9** Encoding a list of more than 20 pairs gives several reports, in the order of the list: every report but the
  last holds 20 pairs, the last holds the rest, and no report holds none. 41 pairs give three reports with the counts
  20, 20 and 1, and 40 pairs give two with 20 and 20. An empty list gives no report. The list passed in is not
  changed, and a tag that appears twice is encoded twice
- **R10** Encoding the clock from a Go `time.Time` gives the clock report above, from the fields of the value as it
  is, in the location it carries. 2026-11-14 15:04:05, a Saturday, gives `03 01 15 1A 0B 0E 0F 04 05 06 64` followed
  by 53 bytes of `0x00`. A Sunday gives `0x07` at offset 9 and a Monday `0x01`
- **R11** Encoding the clock from a time whose year is before 2000 or after 2255 returns an error that names the
  year, and no report
- **R12** The reset command is the report `01 72 65 73 65 74` followed by 58 bytes of `0x00`, and the ymodem command
  is the report `79 6D 6F 64 65 6D` followed by 58 bytes of `0x00`

### YMODEM framing

- **R13** The CRC function of the package gives `0x31C3` for the ASCII bytes `123456789`, `0x0000` for no bytes, and
  `0x0000` for 128 bytes of `0x00`
- **R14** The header block for a theme of a given size is 133 bytes: `0x01`, `0x00`, `0xFF`, then 128 data bytes, then
  the CRC of the data, high byte first. The data is `img.dat`, a `0x00`, the size in decimal ASCII without leading
  zeros, a `0x00`, and `0x00` up to 128 bytes. For a size of 2500 the data starts with
  `69 6D 67 2E 64 61 74 00 32 35 30 30 00` and the CRC bytes are `82 BF`
- **R15** The theme is cut into data blocks of 1029 bytes each: `0x02`, the block number, its one's complement, 1024
  data bytes, and the CRC of the data, high byte first. The first data block has the sequence number 1, and the block
  number byte is the low 8 bits of the sequence number, so that data block 255 has `FF 00`, data block 256 has
  `00 FF` and data block 257 has `01 FE` at offsets 1 and 2. A last block with fewer than 1024 theme bytes is filled
  up with `0x1A`, and a theme whose size is a multiple of 1024 gets no extra block
- **R16** For a theme of 2500 bytes in which byte *i* is *i* modulo 256, there are three data blocks. The first and
  the second have the CRC bytes `C2 E0`. The third holds 452 theme bytes and 572 bytes of `0x1A`, and has the CRC
  bytes `FC 8B`
- **R17** The end of transmission is the single byte `0x04`. The closing block is 133 bytes: `0x01`, `0x00`, `0xFF`,
  128 bytes of `0x00`, and the CRC bytes `00 00`
- **R18** One function of the package turns a sequence of bytes of any length from 1 up into reports: consecutive
  pieces of 64 bytes in order, the last piece filled up with `0x00`. 133 bytes give 3 reports, of which the last
  holds 5 bytes and 59 of `0x00`. 1029 bytes give 17 reports, 64 bytes give 1, and 1 byte gives 1. It is the only
  place in the package that decides how a block, the end of transmission and the closing block are carried in
  reports, and the upload of R20 to R26 sends them through it and in no other way

### Theme upload

- **R19** The upload works on a transport abstraction that has exactly three operations, and on nothing else of the
  outside world:
    - write one report
    - read one report, with a time limit after which it returns an error
    - reopen the device: let go of the device as it was opened and try once to open it again, returning an error when
      that fails
- **R20** Given a transport that answers ACK to everything and reopens at once, and a theme, the upload performs
  these calls in this order and no others, and returns no error:
    1. writes the reset report
    2. reopens the device as R24 says
    3. writes the ymodem report, and reads one report
    4. writes the reports of the header block, and reads one report
    5. for every data block in order: writes its reports, and reads one report
    6. writes the report of the end of transmission, and reads one report
    7. writes the reports of the closing block, and reads none
    8. reopens the device as R24 says
- **R21** The first byte of a report that is read is the answer. `0x06` is ACK, and the upload goes on. `0x15` is NAK,
  and the upload writes again, in full, what it wrote before that read, and reads again. This holds alike for the
  ymodem command, the header block, every data block and the end of transmission
- **R22** One item is written at most 10 times. When the answer to the tenth is NAK, the upload ends with an error,
  and writes nothing more
- **R23** The upload ends with an error and writes nothing more when a write returns an error, when a read returns
  an error, which includes the time limit passing, and when the answer is neither ACK nor NAK. In the last case the
  error states the byte that was received. A read that fails is not followed by writing the item again
- **R24** Reopening is done as follows: the upload pauses for 1 second, then calls the reopen operation. When that
  returns an error of any kind, it pauses for 500 milliseconds and calls it again, up to 20 calls in all. When the
  twentieth fails, the upload ends with an error that wraps the error of the last call
- **R25** The upload never waits by itself. Every pause of R24 goes through a function that its caller supplies and
  that is given the length of the pause, and the time limit of every read is 10 seconds, passed to the read
  operation
- **R26** Every error the upload returns can be inspected, with `errors.As`, for the stage at which it failed and
  for the sequence number of the data block, which is 0 at every stage but the data blocks. The stages are told
  apart as: the reset, the reopening after the reset, the ymodem command, the header block, a data block, the end of
  transmission, the closing block, and the reopening after the upload. The text of the error names the stage, and
  for a data block its sequence number. An error returned by the transport stays reachable with `errors.Is`
- **R27** The upload does not change the theme bytes it is given

### Tests

- **R28** The package has unit tests for each of R8 to R18 that check the examples those requirements give
- **R29** The upload is tested with a fake transport that is scripted with the answers and the reopen results it
  gives, and that records every call. The tests cover at the least:
    - a complete upload of a theme of more than one data block whose size is not a multiple of 1024, checking the
      full order of calls of R20 and the content of every report written
    - NAK and then ACK on a data block: the reports of that block are written twice, identical, and the upload
      completes
    - NAK ten times on one item: the error of R22, with the stage and the block of R26
    - a missing ACK, which is a read that returns an error for the passed time limit: the error carries the stage and
      the block, wraps the error of the read, and nothing is written after it
    - an answer that is neither ACK nor NAK
    - a write that returns an error
    - reopening that fails a few times and then succeeds, after the reset and after the upload: the upload completes,
      and the pauses asked for are those of R24
    - reopening that fails 20 times: the error of R24, with the stage of R26

## Decisions

Those the human made before this spec:

- **D1** sensordeck is written in Go 1.27 and ships as a single static binary, in pure Go without cgo. The module
  path is `github.com/norareit/sensordeck`. The suite of the project is `go test ./...`, `go vet ./...` and a clean
  `gofmt -l`. There is no end-to-end suite yet, because there is no executable to run: the rule of the standards to
  run the end-to-end tests after every change has nothing to run until a unit adds one, and that unit documents it
- **D2** The project is under the MIT licence. The protocol was learned from a project under the GPL-3.0, which the
  README credits. All code and documents of this unit are written from the description in this spec alone. Nobody
  reads, translates or copies the code of that project, and nobody looks for a checkout of it. The description of
  the wire format is the only thing carried over

Those this spec makes:

- **D3** The package is `internal/smartmonitor`, with the import path
  `github.com/norareit/sensordeck/internal/smartmonitor`. It is under `internal` because sensordeck is a program and
  not a library: nothing outside the module may come to depend on it, which keeps every later change to it free. It is
  named after the device family and not `protocol`, because the README promises further screens and the protocol
  belongs to this one; another screen gets a package of its own beside it. The framing of YMODEM and the CRC live in
  the same package: they have one user, and a separate package for them would be an abstraction with one
  implementation. How the package is divided into files is for the plan. The layout of the repository is otherwise
  `go.mod` in the root and `docs/` for documents, and no `cmd/` directory is added before there is a command
- **D4** The encoders return reports and do not send them. Only the upload talks to the transport, because it alone
  is a procedure with answers and failure. Sending a value update or the clock is one write per report, which the
  caller does, so a function for it would add nothing
- **D5** A YMODEM block is carried as consecutive reports of 64 bytes, the last filled up with `0x00`: 3 reports for
  133 bytes and 17 for 1029. What is known says no more than that the transport is reports of 64 bytes, so this is
  an assumption, to be confirmed on the device in the next unit. The alternatives are a length byte or a command
  byte in front of each piece, which nothing known points to. R18 keeps the choice in one function so that it is
  cheap to change
- **D6** The reset command, the ymodem command and the end of transmission are filled up with `0x00` to 64 bytes.
  The fill byte is known for the value update and the clock only, and is assumed for these
- **D7** The 128 data bytes of the header block are filled up with `0x00` after the size, as YMODEM does it. What is
  known gives only what the data holds
- **D8** The answer of the device is the first byte of an input report. What is known says only that the device
  answers with ACK or NAK
- **D9** The upload does not read an answer to the closing block. What is known does not say that the device sends
  one, and the device reboots at that point
- **D10** The numbers of the upload are assumptions to be tuned on the device: the pause of 1 second before the first
  attempt to reopen, 500 milliseconds between attempts, 20 attempts, 10 seconds for an answer, and 10 writes of one
  item. The first pause is there because the device may still be present for a moment after the reset, and an
  attempt made at once could open the device that is about to disappear. 20 attempts cover about 10 seconds against
  a reboot of about 2.5 seconds. 10 seconds for an answer leaves room for the device to erase its flash after the
  header block, with the whole upload taking about 10 seconds. 10 writes is the customary limit of YMODEM. They are
  named constants and not parameters: no caller needs other values yet, and changing a constant in the next unit is
  one line
- **D11** A read that fails, including by its time limit, ends the upload and is not answered by writing the item
  again. YMODEM as published sends again after a silence, but it is not known whether this device accepts a block it
  already has, and writing a block twice into a flash upload is worse than a clear error. A NAK is different: it is
  the device asking for the block
- **D12** An answer that is neither ACK nor NAK ends the upload with an error that states the byte. What the device
  sends without being asked is unknown, and YMODEM as published has the receiver send `C`, `0x43`, at points where
  this spec expects only ACK. Skipping what is not understood would hide exactly what the next unit has to learn,
  while the error names the byte and the stage. Item 1 of "Beyond the request" is the alternative
- **D13** Every error of the reopen operation is tried again, without telling kinds of error apart, because an I/O
  error while the device enumerates is normal and the kinds of error belong to a transport that does not exist yet
- **D14** A data block always has 1024 data bytes and the start byte STX, also the last one. YMODEM as published
  allows a short last block with SOH, and what is known of the device speaks only of blocks of 1024 bytes
- **D15** The block number in an error is the sequence number counted from 1, not the byte on the wire, which starts
  over after 255. A theme that fills the flash has about 4096 blocks
- **D16** The clock is encoded from the fields of the time as given, and the caller chooses the time zone. A year
  the byte cannot hold is an error and is not wrapped around, because a wrong date on the screen would not be
  noticed as a fault of the program
- **D17** The pause is a function the caller supplies and not a call to the clock of the system. This keeps the
  package free of side effects but those of the transport, and lets the tests run without waiting
- **D18** `AGENTS.md` documents the suite, because the standards send the reader to the documentation of the project
  for how tests are run, and the project has none yet

## Non-goals

- Any real device I/O: the Linux hidraw transport and the Windows HID transport, and the report ID byte they add
- Finding or listing connected devices
- A command line, a daemon or any executable
- Reading sensors
- The format of a theme file, and compiling a theme. The upload takes the theme as bytes and does not look into them
- The brightness and every other command that is not known
- A driver interface over several kinds of device
- Confirming anything on the real device. The assumptions of D5 to D10 stay assumptions until the next unit

## Beyond the request

1. Skip input reports that are neither ACK nor NAK while waiting for an answer, up to a limit, where D12 fails on
   the first. It costs a limit to choose and two or three more tests. Without it, an upload on the real device fails
   with an error that states the byte and the stage if the device sends anything else, and the next unit then decides
   with that knowledge
2. Reject a theme of no bytes, and one larger than the flash, before anything is written. It costs a few lines and
   two tests, and the usable size of the 4 MB flash is not known. Without it, an empty theme is sent as a header
   block with the size 0 and no data blocks, and a theme that is too large is sent in full, and what the device does
   in either case is unknown. No caller that could pass such a theme exists yet
3. A way to stop a running upload, with a `context.Context`. It costs a parameter on the upload and on the three
   operations of the transport, and tests. Without it an upload runs to its end or to its first error, at most about
   10 seconds for an answer and about 10 seconds for a reopening
4. A callback that reports the progress of an upload by block. It costs a parameter and a test. Without it a caller
   knows only that the upload ended, which matters once a command line shows it
5. A link to `docs/protocol.md` from the README. It costs one line. Without it the document is found only by looking
   into `docs/`
6. A check in continuous integration that runs the three commands on every push. It costs a workflow file and the
   choice of a service. Without it the suite runs where the units are built and where a developer runs it
