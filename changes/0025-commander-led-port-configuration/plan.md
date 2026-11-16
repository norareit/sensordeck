# Plan: repairing the LED port configuration of the Commander Core XT

## Approach

The fix lives in `internal/commandercore`, which Linux and Windows share: the protocol gains the wanted configuration,
the rule of a known configuration and the configuration write, and the setup of `Session` gains steps 3 and 4 of R4.
`Lighting.Start` tells its caller what the setup did with the configuration, on success and on failure, so that `run`
can write the lines of R8. The keeper of `internal/cli` writes them, and the lines of R15 of unit 0009 stay where they
are. The fake controller learns the endpoint `0x1E` first, so that every later step is tested against it.

The order is: the fake controller, the protocol, the session, the keeper and its wiring, the end-to-end tests, the
documents. Each commit leaves the five commands of the suite passing.

What the tree on `bugfix-0025-commander-led-port-configuration` holds, and what the plan builds on:

* `internal/commandercore/protocol.go` holds `openCommand`, `readCommand`, `closeCommand` (which is
  `08 05 01 <channel> <endpoint>`), `ledCount`, which checks the data type `0F 00` and the room for the entries and sums
  the LED counts of the entries of mode 2, `colourCommand`, which builds 96 data bytes, `stepName`, `invalidity` and the
  constants `portsHeaderSize`, `portSize`, `connectedMode`, `channelZero`, `channelOne`, `portsEndpoint` and
  `ledEndpoint`
* `internal/commandercore/session.go` holds `Lighting.Start(stop) (*Session, error)`, `setUp`, which makes the enter,
  the read of `0x20` through `readEndpoint` with `ledCount` as its check, the open of `0x22` and the colour write,
  `readEndpoint`, `exchange`, `send`, which names the step by `stepName`, `closesChannelZero` and `teardown`.
  `stepName` falls back to `command % x`, so a command of exactly the three bytes `08 15 01` is named
  `command 08 15 01`
* `internal/commandercore/session_test.go` lists the writes of a setup with `setupWrites(count, through)`, which
  numbers the exchanges 1 to 7, and uses positions 1, 4, 6 and 7 in several tests and the total `7+12+2`.
  `hid_test.go` has `TestLightingStartLightsTheLEDsAndEndReturnsToHardwareMode`, whose fake answers one entry of mode 2
  with 8 LEDs, which R3 makes a configuration that is not known, so its writes do not change
* `internal/fakecommander/fakecommander.go` answers `08 06 00` only with `0x22` open on channel 0, gives no answer to a
  command it does not know, `08 15 01` included, answers no content for a read of `0x1E`, and keeps `lighting` and
  `colour`, which a return to hardware mode and the timeout reset. Its state is on `Controller`, not on `node`
* `internal/cli/sources.go` holds `sourceSet.startSession func(stop, colour) (heldSession, error)` and
  `commanderSources`, which maps `commandercore.ErrNoController` to `errNoController`. `run.go`'s `newKeeper` wraps it
  as `start func(stop) (heldSession, error)`. `keeper.go`'s `setUp` writes `controller setup failed: ` once until a
  setup succeeds, through `say`
* `internal/cli/keeper_test.go` has `fakeLighting.start` and `TestRunWritesTheLinesOfTheControllerOnceEach`;
  `commands_test.go`'s harness has the field `startSession` of the same signature
* `e2e/lighting_test.go` has `startRunWithColour(test, interval)`, which adds the controller and starts `run` at once,
  and `encoded(command...)`
* `docs/commander-core-xt.md` has "How it was learned", "### The endpoint `0x20`, the ports", "### The endpoint `0x22`,
  the LEDs, and the colour write", "## A session of run" with the paragraph of the commands OpenLinkHub wrote and
  sensordeck does not send, and "## Unknown". `README.md` has `## The Corsair iCUE Commander Core XT`, whose paragraph
  on other programs names iCUE, liquidctl, OpenLinkHub and `sensordeck sensors`

### The shape of what the setup tells

`commandercore` gains an exported type that `Start` gives beside the session and the error, whatever the error:

```go
// PortConfiguration is what a setup did with the LED port configuration of the endpoint 0x1E.
type PortConfiguration struct {
	Written bool   // the configuration write succeeded, whatever the rest of the setup gave
	Unknown string // the name of the first entry with LEDs whose count is not 4, empty when the configuration is known
	LEDs    int    // the LED count of that entry
}
```

`Start` becomes `Start(stop) (*Session, PortConfiguration, error)`. A setup that ends before the answer of `0x20` is
checked gives the zero value. `Written` is set once the answer of the configuration write is valid, before
`08 05 01 01 1E` and `08 15 01`, so that a failure of those still reports the write (D12).

`internal/cli` keeps the keeper free of `commandercore`: it gains

```go
// setupNotice is what a setup tells of the LED port configuration, for the lines the keeper writes.
type setupNotice struct {
	configured  bool   // the stored configuration was rewritten
	unknownPort string // the entry that made the configuration not known, empty when it is known
	unknownLEDs int
}
```

and `startSession`, the `start` of `newKeeper` and `newSessionKeeper`'s `start` all give
`(heldSession, setupNotice, error)`. `commanderSources` maps `PortConfiguration` to `setupNotice`. The keeper gains a
field `unknownReported bool`, which is set once the line of an unknown configuration is written and never reset in a
run.

## Files

- `internal/fakecommander/fakecommander.go`: the stored configuration, the lighting configuration, the read and write
  of `0x1E` on channel 1, `08 15 01`, and the colour write lit or dark by the lighting configuration
- `internal/fakecommander/fakecommander_test.go`: tests of each of these
- `internal/commandercore/protocol.go`: the wanted configuration, the rule of R3, the configuration write, the names of
  the new exchanges
- `internal/commandercore/protocol_test.go`: the tests of R13 and of the names of R7
- `internal/commandercore/session.go`: `PortConfiguration`, steps 3 and 4 of the setup, the new signature of `Start`
- `internal/commandercore/session_test.go`: `setupWrites` with the read of `0x1E`, the shifted positions, the tests of
  R14
- `internal/commandercore/hid_test.go`: the new signature of `Start`
- `internal/cli/sources.go`, `internal/cli/run.go`, `internal/cli/keeper.go`: `setupNotice`, the signatures, the lines
- `internal/cli/keeper_test.go`, `internal/cli/commands_test.go`: the stand-in with notices and the tests of R15
- `e2e/lighting_test.go`: the tests of R16
- `docs/commander-core-xt.md`, `README.md`: R10 and R11
- `changes/0025-commander-led-port-configuration/record.md`: the log entries

## Tasks

### 1. The fake controller

- [x] In `fakecommander_test.go`, write the tests first, each failing against the tree:
    - the read of `0x1E` on channel 1 in software mode answers status `00` with
      `0D 00 07 00 00 00 01 03 01 03 01 03 01 03` from byte 3 by default, and the bytes set with
      `SetPortConfiguration` after it is called (R12)
    - with `0x1E` open on channel 1, a write `08 06 01 0E 00 00 00 0D 00 07 06 06 06 06 06 06 06 00 00 01 03` answers
      status `00`, and `PortConfiguration()` gives the 14 bytes from byte 7; the same write with `0x1E` not open on
      channel 1 answers status `03` and stores nothing (R12)
    - `08 15 01` answers status `00` in software mode, and `08 15 01`, `08 06 01` and the read of `0x1E` answer status
      `03` outside software mode (R12)
    - the stored configuration survives a return to hardware mode, the software-mode timeout, and a close and reopen of
      the node (a second harness session that opens the same node path) (R12)
    - with the default lighting configuration and the iCUE configuration
      `0D 00 07 06 06 06 06 06 06 06 00 00 01 03 01 03 01 03 01 03` stored, a colour write answers status `00`, is kept
      in `ColourWrites`, and `Lighting()` gives `Dark`; after the wanted configuration is written over `0x1E` and the
      colour written again, `Lighting()` gives `SentColour` with the colour; with `SetLightingConfiguration(nil)` a
      colour write lights whatever is stored (R12, D13)
    - a new `Failure` value `UnavailableStatus` writes `03` in byte 2, as a case of
      `TestEachFailureChangesTheAnswerOfItsStep`, so that the session tests of R14 can fail an exchange with the
      status `03` that the spec names (R14)
    - `TestLightingFollowsTheModeAndTheColourWrites` and the other tests of the fake keep what they expect (R12, R17)
- [x] In `fakecommander.go`, add the constant `portConfigurationEndpoint = 0x1E`, a package variable
  `defaultPortConfiguration` holding `0D 00 07 00 00 00 01 03 01 03 01 03 01 03`, and on `Controller` the fields
  `portConfiguration []byte` and `lightingConfiguration []byte`, both set to copies of the default in `newController`
  (R12)
- [x] Add `SetPortConfiguration(configuration []byte)`, `PortConfiguration() []byte` (a copy) and
  `SetLightingConfiguration(configuration []byte)`, where nil means none, each under the mutex, with doc comments that
  say the defaults (R12)
- [x] In `answerOf`, answer `portConfigurationEndpoint` with a copy of the stored configuration (R12)
- [x] In `handle`, after the software-mode check, add the case `08 06 01`: status `03` unless `open[1]` is
  `portConfigurationEndpoint`; otherwise read the length from bytes 3 and 4, low byte first, store a copy of
  `command[7:7+length]`, bounded by the command, and answer `00`. Add the case `08 15 01`, answering `00` (R12)
- [x] In the case `08 06 00`, keep the write in `writtenColours` and the colour as today, and set `lighting` to
  `SentColour` when `lightingConfiguration` is nil or equals `portConfiguration`, and to `Dark` otherwise (R12, D13)
- [x] Add `UnavailableStatus` after the last value of `Failure`, and its case in `applyFailure`, writing `0x03` in
  byte 2 (R14)
- [x] Update the package comment and the comment of `Controller` to name the LED port configuration (R12)
- [x] Commit: `bugfix-0025 Keep an LED port configuration in the fake controller`

### 2. The protocol

- [x] In `protocol_test.go`, write the tests first (R13):
    - `wantedConfiguration` of the answer of the host, built with `portsAnswer`, gives
      `0D 00 07 00 00 00 01 03 01 03 01 03 01 03`, and `configurationCommand` of it gives 96 bytes that start with
      `08 06 01 0E 00 00 00 0D 00 07 00 00 00 01 03 01 03 01 03 01 03` and are zero after (R1, R2)
    - entries of mode 2 with 4 LEDs on EXT and the ports 1 and 2 and mode 0 on the ports 3 to 6 give
      `0D 00 07 01 03 01 03 01 03 00 00 00 00`, with the length `0D 00` in its write (R1, R2)
    - an entry of mode 2 with 0 LEDs and an entry of mode 3 with 4 LEDs are each `00` (R1, D6)
    - entries with a port of mode 2 with 8 LEDs, `{0,0}, {2,4}, {2,8}`, make `portTypeProblem` give `port 2` and 8, and
      the host's entries give an empty name (R3)
    - `stepName` gives `write of the LED port configuration` for a configuration write, `command 08 15 01` for
      `08 15 01`, and `read of endpoint 0x1e on channel 1`, `open of endpoint 0x1e on channel 1` and
      `close of endpoint 0x1e on channel 1` for the commands of `0x1E` (R7)
- [x] In `protocol.go`, add the constants `portConfigurationEndpoint byte = 0x1E` and `knownLEDCount = 4`, and the
  variables `portConfigurationDataType = []byte{0x0D, 0x00}`, `ledPortType = []byte{0x01, 0x03}` and
  `afterPortConfiguration = []byte{0x08, 0x15, 0x01}`, with a comment that its meaning is unknown and that it follows
  the write because every setup that lit the LEDs sent it (D7)
- [x] Add `entryName(position int) string`, `EXT` for 0 and `port <position>` otherwise, and `hasLEDs(entry []byte)`,
  mode 2 and an LED count other than 0. Use `hasLEDs` where `ledCount` sums, so that both decide "connected" in one
  place (R1, D6)
- [x] Add `wantedConfiguration(answer []byte) []byte` for an answer of `0x20` that `ledCount` accepted: the data type,
  the count *c*, and for each entry `01 03` or `00` (R1)
- [x] Add `portTypeProblem(answer []byte) (name string, leds int)` for an answer that `ledCount` accepted: the name and
  count of the first entry that has LEDs and a count other than 4, or the empty name (R3)
- [x] Add `configurationCommand(configuration []byte) []byte`, 96 bytes: `08 06 01`, the length low byte first, `00 00`,
  the configuration from byte 7 and zeros (R2)
- [x] In `stepName`, add before the case of `08 06 00` the case of the prefix `08 06 01`, named
  `write of the LED port configuration`. `08 15 01` keeps the fallback (R7)
- [x] Commit: `bugfix-0025 Build the LED port configuration from the answer of the ports`

### 3. The setup

- [x] In `session_test.go`, change `setupWrites` so that its list is enter (1), the read of `0x20` (2 to 5), the read of
  `0x1E` (6 to 9), the open of `0x22` (10) and the colour write (11), and shift every position past 5 that the
  existing tests use by 4: 6 becomes 10, 7 becomes 11, and the total `7+12+2` becomes `11+12+2`. The positions 1 and 4,
  and `setupWrites(0, 4)`, stay. Change every call of `Start` to take three results. No other expectation changes
  (R4, R17)
- [x] Add a helper `configurationWrites` for the five exchanges of step 4 with the wanted configuration of the host:
  `08 05 01 01 1E`, `08 0D 01 1E`, the configuration write, `08 05 01 01 1E` and `08 15 01` (R4)
- [x] Write the tests of R14 first, each failing against the tree:
    - with the default configuration stored, the writes of a setup are `setupWrites(16, 11)`, which hold the read of
      `0x1E`, no configuration write and no `08 15 01`, and `PortConfiguration` is the zero value (R4, R14)
    - with the iCUE configuration stored, the writes are enter, the read of `0x20`, the read of `0x1E`,
      `configurationWrites`, the open of `0x22` and the colour write, in that order; the fake's `PortConfiguration()`
      equals the wanted configuration; `Lighting()` gives `SentColour` with `FF 00 00`; `Written` is true (R4, R14)
    - a stored configuration equal to the wanted one but for its last byte, `02` for `03`, is rewritten, with the same
      writes (R4, R14, D4)
    - with the ports `{0,0}, {3,0}, {3,0}, {2,8}, {2,4}, {2,4}, {2,4}`: no write starts with `08 0D 01 1E`,
      `08 05 01 01 1E`, `08 06 01` or `08 15 01`, the setup succeeds with the colour write of 20 triples, and
      `PortConfiguration` is `{Unknown: "port 3", LEDs: 8}` (R3, R14)
    - with the iCUE configuration stored, a table over the read of `0x1E` (prefix `08 08 01 1E`), the configuration
      write (prefix `08 06 01`) and `08 15 01`, each failed with `FailCommand` and `UnavailableStatus` of task 1. Each
      case stops the setup at that exchange, is followed by `teardownWithClose`, writes nothing after it, and gives the
      error
      `read of endpoint 0x1e on channel 1: status 03`, `write of the LED port configuration: status 03` or
      `command 08 15 01: status 03`. The case of `08 15 01` gives `Written` true (R4, R7, R14, D12)
    - the same three with `NoAnswer`: the teardown is `teardownWithoutClose`, and the error ends in
      `: no answer within 50ms` (R4, R7, R14)
    - with the iCUE configuration stored, a session started, read, ended and started again: both setups read `0x1E`,
      the first writes the configuration and the second does not, and the writes of the reading are
      `readingWrites()`, with no command to `0x1E` (R5, R14)
- [x] In `session.go`, add `PortConfiguration` as in the approach, and change `Start` to give it from `setUp` on every
  path (R4, R8)
- [x] In `setUp`, keep the answer of `0x20` from the check of `readEndpoint`. After the read, take `portTypeProblem`;
  when it names an entry, set `Unknown` and `LEDs` and go to the open of `0x22`. Otherwise make
  `readEndpoint(stop, channelOne, portConfigurationEndpoint, check)` with a check that keeps the answer and gives no
  error, and when bytes 3 to 3 + *L* − 1 of it differ from the wanted configuration, make `writeConfiguration`. Then the
  open of `0x22` and the colour write as today. Keep `setUp` at one level of abstraction by putting steps 3 and 4 in a
  method `repairConfiguration(stop, answer) (PortConfiguration, error)` (R3, R4, D3, D4)
- [x] Add `writeConfiguration(stop, configuration) (written bool, err error)`: the close and open of `0x1E` on
  channel 1, `configurationCommand`, which sets `written` once it succeeds, the close, and `afterPortConfiguration`,
  each through `exchange` so that a stop, a failure and no answer end it as every exchange does (R4, D8, D12)
- [x] Check that nothing in `Read` and `End` changed, and that no command of R6's forbidden list is built anywhere in
  the package (`grep -n "0x14\|0x18\|0x02, 0x13" internal/commandercore/*.go`) (R5, R6)
- [x] In `hid_test.go`, take the three results of `Start`. Its writes do not change, since its one entry has 8 LEDs (R9,
  R17)
- [x] Commit: `bugfix-0025 Repair the LED port configuration at each setup`

The sandbox runs the suite here, and the Windows build and vet of `AGENTS.md` cover `hid.go` and the shared session
(R9).

### 4. The lines of run

- [x] In `keeper_test.go`, give `fakeLighting` a queue `notices []setupNotice`, taken one per setup like
  `setupErrors`, and change `start` to give `(heldSession, setupNotice, error)`. Change the field `startSession` of
  the harness in `commands_test.go` to the new signature (R15)
- [x] Write the tests of R15 first:
    - two setups that fail with notices `{configured: true}` and then a setup that succeeds with
      `{configured: true}`: standard error is the line of the configuration write, then `controller setup failed: fake
      failure`, then the line of the configuration write twice more, in that order, so that the line comes after each
      setup that made one and before the failure line of the same setup (R8, R15, D10, D12)
    - a run whose setups all give the zero notice writes nothing to standard error (R8, R15)
    - setups with `{unknownPort: "port 3", unknownLEDs: 8}`, one failing and the next succeeding and a later one after a
      lost session: standard error holds
      `controller LED ports left as stored: port 3 has 8 LEDs, and only fans with 4 LEDs are known` exactly once
      (R8, R15, D10)
    - the existing tests of the keeper keep what they expect (R17)
- [x] In `keeper.go`, add `setupNotice` as in the approach, change the `start` field and `newSessionKeeper` to the new
  signature, and in `setUp` write, before anything else that the setup writes, the line
  `controller LED ports configured: the stored configuration did not enable the ports with LEDs, iCUE may have changed it`
  when `configured`, and the line of an unknown configuration when `unknownPort` is set and `unknownReported` is
  false, setting it (R8)
- [x] In `sources.go`, change `startSession` and its comment, and map `PortConfiguration` to `setupNotice` in
  `commanderSources`, giving the notice with the error too. In `run.go`, change `newKeeper`'s `start` (R8, R9)
- [x] Commit: `bugfix-0025 Write the lines of the LED port configuration in run`

### 5. End to end

- [x] In `e2e/lighting_test.go`, add `startRunWithColourAfter(test, interval, prepare func(*fakecommander.Controller))`,
  which calls `prepare` on the controller before `run` starts, and make `startRunWithColour` call it with a `prepare`
  that does nothing (R16)
- [x] Add a helper that counts the writes of `controller.Writes()` that start with `encoded(0x08, 0x06, 0x01)[:4]` and
  those equal to `encoded(0x08, 0x15, 0x01)` (R16)
- [x] Add `TestRunWithAColourRepairsTheLEDPortConfigurationThatICUELeft`: the iCUE configuration stored and the lighting
  configuration left at its default; wait for the first colour write; `Lighting()` gives `SentColour` with `FF 00 00`
  while `run` runs; `PortConfiguration()` equals the wanted configuration; after SIGTERM, standard error holds the line
  of the configuration write and the counts are one configuration write and one `08 15 01` (R16)
- [x] Add `TestRunWithAColourLeavesAStoredConfigurationAlone`: the default configuration; after SIGTERM no line holding
  `controller LED ports`, no configuration write and no `08 15 01` (R16)
- [x] Run `go test ./e2e/...` and the rest of the suite; the existing end-to-end tests keep what they expect (R17)
- [x] Commit: `bugfix-0025 Test the repair of the LED port configuration end to end`

### 6. The documents

- [x] In `docs/commander-core-xt.md`, add after "### The endpoint `0x22`, the LEDs, and the colour write" a section
  "### The endpoint `0x1E`, the LED port configuration" that says, in the project's own words: the read on channel 1
  and its layout, data type `0D 00`, the count and one entry per connection; the write `08 06 01` and its layout, with
  the bytes of the host `08 06 01 0E 00 00 00 0D 00 07 00 00 00 01 03 01 03 01 03 01 03`; the configuration of
  OpenLinkHub and the one left after iCUE had run, `0D 00 07` and seven `06`, with the bytes after them taken to be
  stale; that it is stored in the controller and survives a power cut; that in software mode the colour lights only
  the ports it enables, as the device tests of unit 0025 showed; the rule of R1 and R3, with `01 03` written only when
  every port with LEDs has 4; and `08 15 01` after the write (R10)
- [x] In "How it was learned", add the device tests of unit 0025 of 2026-11-16 to the source of tests made by hand,
  with what they showed: the configuration of `0x1E` decides which ports light (R10)
- [x] In "## A session of run", give the setup of R4, with the read of `0x1E` and, when it differs, the five exchanges
  of the write. In the paragraph of the commands that sensordeck does not send, remove `0x1E` and `08 15 01` and keep
  `08 02 13`, `08 14 0N 01`, the write to `0x18`, the zero triples and the other opens on channel 0 (R10)
- [x] In "## Unknown", add: the meaning of the type `03` and what a port with another LED count needs; the meaning of
  `06`; whether `08 15 01` is needed; whether the bytes after the seven `06` are stale; whether a write of `0x1E` wears
  the memory of the controller; and what iCUE does with the configuration that sensordeck writes (R10)
- [x] Check with `git diff` that no line of the document was removed but those the requirement names (R10)
- [x] In `README.md`, section `## The Corsair iCUE Commander Core XT`, add a paragraph after the one on losing the
  controller that says: the controller stores which of its ports have LEDs; iCUE rewrites it, and afterwards the LEDs
  stay dark in software mode, on Linux and on Windows; `run` checks it at the start of each session and repairs it when
  it does not match the fans the controller reports, writing `controller LED ports configured: ...` with the rest of
  the line of R8, and the repair stays stored until iCUE changes it again; the repair is made only when every port with
  LEDs has 4, the one kind of fan it was tested with, and otherwise `run` writes `controller LED ports left as stored`
  and the LEDs may stay dark. In the paragraph on other programs, say that running iCUE and sensordeck at the same time
  is not supported, keeping the programs it names (R11)
- [x] Commit: `bugfix-0025 Describe the LED port configuration in the documents`

## Verification

- [x] `go test ./...` passes
- [x] `go vet ./...` reports nothing
- [x] `gofmt -l .` prints no file name
- [x] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [x] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [x] `npm run check-changes` passes after the last commit

## Commit Points

- [x] After task 1: the fake controller
- [x] After task 2: the protocol
- [x] After task 3: the setup
- [x] After task 4: the lines of run
- [x] After task 5: the end-to-end tests
- [x] After task 6: the documents

## Trace

| Requirement | Tasks |
|-------------|-------|
| R1 | 2, 3 |
| R2 | 2 |
| R3 | 2, 3 |
| R4 | 3 |
| R5 | 3 |
| R6 | 3 |
| R7 | 2, 3 |
| R8 | 3, 4 |
| R9 | 3, 4, Verification |
| R10 | 6 |
| R11 | 6 |
| R12 | 1 |
| R13 | 2 |
| R14 | 1, 3 |
| R15 | 4 |
| R16 | 5 |
| R17 | 1, 3, 4, 5 |
