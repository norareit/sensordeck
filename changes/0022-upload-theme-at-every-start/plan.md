# The theme uploaded at every start of `run`: plan

## Approach

The decision whether to upload moves from the state file into the theme holder of `run`, which keeps the compiled file
of its last completed upload in memory. The holder is due to upload at its creation whenever the config has a theme,
so the first round of every start uploads. A reload compares the bytes of the reloaded theme with the last upload. A
screen that comes back makes nothing due, and a restart that recovered a hung screen clears the wait of an upload that
is due. Everything that served only the state file is then removed: `stateFile`, the package `uploadstate`, the hash of
`configTheme`, the look-up of the cache directory by `run` and `upload`, and `Serial` of every open transport. The
README is rewritten last.

### The theme holder after this unit

`internal/cli/themeholder.go` keeps its role and its seams to the feeder (`ensure`, `onScreen`, `setTheme`), with
these fields and rules:

* Fields: `theme *configTheme`, `last []byte` (the last upload of R3, nil when there is none), `due bool`,
  `next time.Time`, `wait time.Duration`, `pause`, `now`, `say`, `stderr`. The fields `state`, `checkDue` and
  `starting` and the method `check` go
* `newThemeHolder(theme, pause, now, say, stderr)` sets `due` to `theme != nil` and `wait` to `firstRetry`
* `ensure(stop, opened)`: with no theme, or nothing due, it gives ready. Once stop is done, or before `next`, it gives
  not ready. Otherwise it uploads, as today
* `upload(opened)` sets `last` to nil first (R3, D5), writes `uploading theme <value>`, and calls
  `smartmonitor.Upload(opened, holder.theme.file, holder.pause, func(smartmonitor.Stage, int, int) {})`. On a failure it
  writes `theme upload failed: <error>` and sets `next` and `wait` as today. On success it writes
  `theme uploaded: <n> bytes`, sets `last` to the uploaded file, clears `due` and `next`, and sets `wait` to
  `firstRetry`
* `setTheme(theme)`: nil sets `theme`, `last` to nil and `due` to false (R3, R5). Otherwise it sets `theme` and sets
  `due` to `!bytes.Equal(theme.file, holder.last)`, which is true when there is no last upload (R4). It leaves `next`
  and `wait` as they are, as the holder of unit 0013 does
* `restartRecovered()` replaces `screenBack()`: when an upload is due, it sets `next` to the zero time, so that the
  round that opened the screen makes it, and it leaves `wait` as it is, so that a failure of that upload waits as long
  as the doubling of R9 of unit 0013 had reached (R21)
* `onScreen()` is `theme == nil || !due`

### The feeder

In `internal/cli/run.go`, `reopen` no longer calls anything on the holder for a screen that came back (R6).
`recoverHung` gives two results, `(open, recovered bool)`: `recovered` is true when the node was opened in the
`awaitingOpen` branch, and when `restartScreen` ended with `restartedOpen`, and false for a reopening within
`restartInterval`, for `notRestarted` and for `restartNotPossible`. `restartScreen` gives the same pair. `reopen` calls
`feeding.holder.restartRecovered()` when `recovered` is true. Nothing else of the hang changes.

## Files

* `internal/cli/themeholder.go`, `internal/cli/themeholder_test.go`
* `internal/cli/run.go`, `internal/cli/runconfig.go`, `internal/cli/commands.go`, `internal/cli/main.go`,
  `internal/cli/trace.go`
* `internal/cli/statefile.go` and `internal/cli/statefile_test.go` (deleted)
* `internal/uploadstate/uploadstate.go` and `internal/uploadstate/uploadstate_test.go` (deleted, with the directory)
* `internal/cli/runtheme_test.go`, `internal/cli/runreload_test.go`, `internal/cli/runhang_linux_test.go`,
  `internal/cli/runconfig_test.go`, `internal/cli/commands_test.go`, `internal/cli/fake_test.go`,
  `internal/cli/trace_test.go`
* `internal/cli/device_linux.go`
* `internal/hidraw/transport.go`, `internal/hidraw/transport_test.go`
* `internal/winhid/transport.go`, `internal/winhid/transport_test.go`, `internal/winhid/device.go`,
  `internal/winhid/device_test.go`
* `e2e/theme_upload_test.go`
* `README.md`

Left as they are: `internal/hidraw/discover.go`, `internal/winhid/collections.go`, `Seam.Serial` and
`internal/fakehid/fakehid.go`, which give the serial of discovery and of `devices`; `entryOfNode` in
`internal/hidraw/transport.go`, which `internal/hidraw/usb.go` uses for the restart of unit 0021;
`environment.userCacheDir`, which `background.go` and `startuptask.go` use for the log on Windows; and
`e2e/main_test.go`, which keeps giving each test an `XDG_CACHE_HOME` of its own.

## Tasks

### 1. `run` uploads at every start and decides reloads in memory

A test that needs a state file that an earlier release wrote writes it itself, at
`filepath.Join(cache, "sensordeck", "uploaded-themes.json")`, with the JSON object that maps the serial to the
lower-case hexadecimal SHA-256 of the compiled file, from `crypto/sha256` and `encoding/hex` in the test, so that no
test depends on `uploadstate`, which task 2 deletes.

- [ ] In `internal/cli/themeholder_test.go`, drop the state file from `holderFixture` and `aTheme` (no `hash`), and
  replace the tests of the check by tests of the rules of "The theme holder after this unit": the first `ensure`
  uploads and says the two lines; a second holder made with the same theme uploads again; `setTheme` with other bytes
  makes the next `ensure` upload, and with the same bytes uploads nothing; `setTheme(nil)` then the same theme again
  uploads; a failed upload of theme `B` after a completed upload of `A`, followed by `setTheme(A)`, uploads `A`; the
  line `theme <value> is on the screen` is never said. Keep the tests of the retry, of no theme, of stop and of the
  error of a failed upload, without the state file (R1, R2, R3, R4, R5, R18)
- [ ] In `internal/cli/runtheme_test.go`, rewrite `runningTheme` with no state file and no serial, and change
  `TestRunWithAThemeUploadsItBeforeTheClockAndTheValuesAndRecordsTheHash` to check the upload before the clock and the
  values with no state file, and again with a state file that names the hash of the compiled theme for the serial,
  which `run` uploads anyway and leaves byte for byte as it was (R1, R9, R18)
- [ ] In `runtheme_test.go`, change `TestTwoStartsWithTheSameThemeAndScreenMakeOneUpload` to two starts that each
  upload, with no `theme <value> is on the screen` in standard output (R1, R2, R18)
- [ ] In `runtheme_test.go`, change `TestRunChecksTheThemeAgainAfterTheScreenCameBack` into one test of a screen lost
  by a failed write and reopened, whose writes are `UCVCV` (R6, R18), and add one in which the first upload fails with
  `errFake`, the screen is reopened in the next round, and the upload is made once the wait of 10 seconds has passed,
  before any clock or value (R6, R18)
- [ ] In `runtheme_test.go`, remove the case `no cache directory` of
  `TestRunRefusesWhatItCannotStartWithAThemeBeforeOpeningTheScreen`, and add a test in which `run` with a theme and a
  `userCacheDir` that fails uploads and sends the values, and `harness.cacheDirCalls` is 0 (R11, R18). Rename
  `TestAStopDuringAnUploadEndsRunAfterTheUploadWithTheHashRecorded` to
  `TestAStopDuringAnUploadEndsRunAfterTheUpload` and drop its check of the state file
- [ ] In `internal/cli/runreload_test.go`, replace `TestAReloadToAThemeWithNoCacheDirectoryIsRefusedWithItsLine` by a
  test in which that reload uploads the theme with nothing written to standard error, and add a test of `theme` to
  `tags` and back to the same theme, which uploads twice in all. Keep the tests of `tags` to `theme`, of `theme` to
  `tags` and of the rewrite with the same bytes and the edited folder, without the state file (R4, R5, R11, R18)
- [ ] In `internal/cli/runhang_linux_test.go`, change `TestRunChecksTheThemeAfterTheRestartOfAHungScreen` to
  `TestRunUploadsNothingAfterTheRestartOfAHungScreen`: the theme is uploaded at the start, a write of the clock or of a
  value times out, the screen is restarted with `openedAt33`, and the writes after the restart hold no upload. Drop the
  state file from `TestRunWritesThatAnUploadFailedBeforeThatTheScreenHung` (R6, R7, R18)
- [ ] In `internal/cli/runconfig_test.go`, drop the check of `loaded.theme.hash` and the import of `uploadstate`
- [ ] Rewrite `internal/cli/themeholder.go` as "The theme holder after this unit" says, without `restartRecovered`,
  which task 4 adds. Update its comments: the holder uploads the theme of the config at the start and again when a
  reload gives other bytes than the last upload (R1, R2, R3, R4, R5, R8)
- [ ] In `internal/cli/runconfig.go`, remove `hash` from `configTheme` and the import of `uploadstate` (R10)
- [ ] In `internal/cli/run.go`, remove `state` and `stateErr` from `preparedRun` and the call of `current.stateFile()`
  from `prepareRun`, make the `load` of the watcher `current.loadConfig(prepared.path, content)` alone, make the holder
  with `newThemeHolder(prepared.started.theme, current.env.sleep, current.env.now, current.say, current.env.stderr)`,
  and remove the call of `screenBack` from `reopen` (R6, R9, R11)
- [ ] In `e2e/theme_upload_test.go`, drop `statePath` and `stateHolds` from `themeSetup`, keep its own
  `XDG_CACHE_HOME`, and rename `TestRunUploadsTheThemeOnceAndAStartAfterItUploadsNothing` to
  `TestRunUploadsTheThemeAtEveryStart`: the first `run` uploads `minimal-dark` and then sends the clock and the values,
  and the second, on a screen added again with `AcceptUpload` armed, uploads it again, with no
  `theme minimal-dark is on the screen` in standard output (R19)
- [ ] In `e2e/theme_upload_test.go`, add a test in which a state file in the `XDG_CACHE_HOME` of the setup names the
  hash of `minimal-dark` for `screenSerial`, `run` uploads `minimal-dark` all the same, and the file is byte for byte
  as it was after `run` stopped (R19)
- [ ] In `e2e/theme_upload_test.go`, drop the check of the state file from
  `TestThemeUseWhileRunRunsPutsTheNewThemeOnTheScreenWithItsTags` (R19)
- [ ] In `e2e/theme_upload_test.go`, add a test in which `run` with `minimal-dark` uploads it and sends values, the
  screen is moved to hidraw 7 with `MoveAfter` once the upload is done, and `run` then sends the clock and the values on
  the new node with no reset report after the move and no second `uploading theme` in standard output (R6, R19)
- [ ] In `e2e/theme_upload_test.go`, replace `TestUploadKeepsTheStateFileTrueForRun` by
  `TestRunPutsItsThemeBackAfterAnUpload`: `upload` of another file, then `run`, which uploads `minimal-dark` and sends
  values. The second half, in which `run` uploads nothing after an `upload` of the same file, goes, since it is the
  behavior this unit removes (R1, R19)
- [ ] Run the five commands of the suite (R20)
- [ ] Commit: `bugfix-0022 Upload the theme at every start of run and decide reloads in memory`

### 2. `upload` without the state file, and the state file removed

- [ ] In `internal/cli/commands_test.go`, replace `TestUploadWritesNoReportWithoutACacheDirectory` by a test in which
  `upload` with a `userCacheDir` that fails completes as `TestUploadComplete` does, and `harness.cacheDirCalls` is 0.
  Remove `TestUploadWritesNoReportWhenTheRemovalCannotBeWritten` and the checks of the state file in the other tests of
  `upload`, and remove `useCacheDirectory`, whose callers set `cacheDir` themselves where they still need one (R11,
  R18)
- [ ] In `e2e/theme_upload_test.go`, add `TestRunAndUploadCreateNothingInTheCacheDirectory`: after `theme use`, a `run`
  that uploads and is stopped, and an `upload` of another file, the `XDG_CACHE_HOME` of the setup holds no entry named
  `sensordeck` (R9, R19)
- [ ] In `internal/cli/commands.go`, have `upload` call `smartmonitor.Upload(opened, theme, current.env.sleep,
  progress)` with no look-up of the state file, and keep its lines, its errors and `restartAfterHang` as they are (R9,
  R11)
- [ ] Delete `internal/cli/statefile.go`, `internal/cli/statefile_test.go` and the directory `internal/uploadstate/`,
  and remove every remaining import of `uploadstate` (R12)
- [ ] Run `grep -rn "uploadstate\|stateFile\|uploaded-themes" --include=*.go .` and leave only the literal path of the
  tests of task 1 that write a file of an earlier release (R9, R12)
- [ ] Run the five commands of the suite (R20)
- [ ] Commit: `bugfix-0022 Upload a theme file without the state file and remove the state file`

### 3. The serial of the open transport removed

- [ ] Remove `Serial()` from the interface `transport` in `internal/cli/main.go`, from `tracingTransport` in
  `internal/cli/trace.go` with `TestTraceGivesTheSerialOfTheInnerTransportAndWritesNoLine` in `trace_test.go`, and the
  field `serial` and its method from `fakeTransport` in `internal/cli/fake_test.go` with every test that sets it (R12)
- [ ] In `internal/hidraw/transport.go`, remove the field `serial` of `Transport` and of `opened`, and `Serial()`.
  `OpenPath` takes the path alone, as `OpenPath(path string)`, and opens it with no call of `entryOfNode`; update its
  comment and the call in `internal/cli/device_linux.go`. Keep `entryOfNode` for `usb.go` (R12)
- [ ] In `internal/hidraw/transport_test.go`, remove `TestSerialOfADiscoveredScreen`,
  `TestSerialOfAScreenWithNone`, `TestSerialOfAPathIsThatOfTheEntryOfTheNode`, `TestSerialOfAPathWithNoEntryIsEmpty`
  and `TestSerialAfterAReopenIsThatOfTheNodeOpenedThen`, and call `OpenPath` with the path alone in the others. The
  restart tests of `usb_test.go` and `restart_test.go` keep covering `entryOfNode` (R12)
- [ ] In `internal/winhid/transport.go`, remove the field `serial` and `Serial()`. In `internal/winhid/device.go`,
  remove the field `serial`, `Serial()` and the call of `seam.Serial` from `Open`, and update the comment of `Open` to
  say it reads the caps (R12)
- [ ] In `internal/winhid/device_test.go`, rename `TestOpenOpensOnlyThePathForReadingAndWritingAndReadsCapsAndSerial`
  to `TestOpenOpensOnlyThePathForReadingAndWritingAndReadsCaps` and drop its check of the serial. In
  `internal/winhid/transport_test.go`, drop the checks of `Serial()` and remove
  `TestTheSerialIsEmptyWhenTheCollectionHasNone`. Keep the tests of the serial in `collections_test.go`, which is that of
  discovery (R12)
- [ ] Run `grep -rn "Serial()" --include=*.go .` and leave none on a transport or a device; `Seam.Serial` and the
  `Serial` fields of `Entry`, `Screen`, `Collection` and `fakehid.Collection` stay (R12)
- [ ] Run the five commands of the suite, which build and vet for Windows (R20)
- [ ] Commit: `bugfix-0022 Remove the serial of the open transport`

### 4. The upload made at once after a restart that recovered the screen

Each test below runs with `newHungRun` and a theme config, with `ackWhenEmpty` set. The first upload of the start
fails by a timed-out write (an entry `true` in `failWrites`, `failErr` being `timeoutError`), and the fake clock gives
the moment of each upload attempt, as `uploadStarts` in `runtheme_test.go` does for the reset report.

- [ ] In `internal/cli/themeholder_test.go`, test `restartRecovered`: with an upload due and a wait not passed, the next
  `ensure` uploads; with no upload due, it changes nothing; and after it, a failure of the upload made at once sets the
  next attempt the doubled wait later, 20 seconds after the first failure of 10 (R21, R22)
- [ ] In `internal/cli/runhang_linux_test.go`, add a test with `openedAt33` in which the upload is made again in the
  round of the restart, before the 10 seconds after the failed attempt have passed, followed by the clock and the
  values, and a test with `notOpenedAt33` in which it is made in the later round that opens the screen (R7, R21, R22)
- [ ] In `runhang_linux_test.go`, add one test with three cases, `didNotRestart`, `notPossibleNow`, and a first upload
  that fails with `errFake` rather than a timeout so that no hang begins, in each of which the second upload attempt
  begins no earlier than 10 seconds after the first (R21, R22)
- [ ] In `runhang_linux_test.go`, add a test in which the upload made at once after `openedAt33` fails too, by a write
  that times out, and the attempt after it begins no earlier than 20 seconds after that failure (R21, R22)
- [ ] In `internal/cli/themeholder.go`, add `restartRecovered()` as "The theme holder after this unit" says (R21)
- [ ] In `internal/cli/run.go`, give `recoverHung` and `restartScreen` the results `(open, recovered bool)` as "The
  feeder" says, and have `reopen` call `feeding.holder.restartRecovered()` when `recovered` is true. Update the comments
  of `reopen`, `recoverHung` and `restartScreen` (R21)
- [ ] Run the five commands of the suite (R20)
- [ ] Commit: `bugfix-0022 Upload at once after a restart that recovered the screen`

### 5. The README

- [ ] In the opening paragraph, say that `theme use` sets the theme of the config file, which `run` uploads at its
  start, in place of "which `run` uploads when it is not on the screen". In the list of what sensordeck does, replace
  "It uploads again only when the theme changes." by a sentence that `run` uploads the theme at each start and when a
  reload changes it (R13)
- [ ] In step 3 of the quick start, say that `run` uploads the theme to the screen at each start, which takes about 10
  seconds and reboots the screen, and sends the values after that, and remove the sentence on a later start (R14)
- [ ] In "Showing sensor values on the screen", replace "uploads the theme to the screen when it is not there" by
  "uploads the theme to the screen at its start", keeping the link to "Writing a theme" (R13)
- [ ] In "Running as a service", in the Linux section beside the paragraph on `theme use` and in the Windows section
  beside the same sentence, say that each start of the service, at the login or logon, at boot with lingering on Linux,
  and at each restart after an error, uploads the theme of the config, so that the values appear about 10 seconds after
  the start and the screen reboots once, and that a new theme set by `theme use` or an edit is uploaded when it reaches
  the running service within a round (R15)
- [ ] In "Writing a theme", replace the paragraph that begins "`run` uploads the theme only when it is not on the
  screen" and its three bullets by text that says every item of R16, in its order: the upload at each start with its 10
  seconds, its reboot and no value or clock before it ends, and the compilation in memory with no compiled file kept;
  the upload within one `run` only for a reload that compiles to another file, with `theme use` again after an edit of
  the folder; no upload at a replug or a restart after a hang, because the screen keeps its theme in its flash; the
  retry after 10 seconds and then longer waits up to 5 minutes; the theme of other software, of `sensordeck upload`, of
  the other operating system of a dual-boot machine or of another computer, replaced at the next start; the screen given
  another theme while unplugged from a running `run`, which keeps it until `run` starts again; and
  `uploaded-themes.json` of earlier releases, at `$XDG_CACHE_HOME/sensordeck/uploaded-themes.json` or
  `~/.cache/sensordeck/uploaded-themes.json` on Linux and `%LocalAppData%\sensordeck\uploaded-themes.json` on Windows,
  no longer used and safe to delete (R16)
- [ ] Search `README.md` and `docs/` for `state file`, `uploaded-themes`, `is on the screen`, `not on the screen`,
  `only when`, `uploads nothing`, `SHA-256` and `hash`, and leave none that says `run` keeps a state file, uploads on a
  change only, or checks whether the theme is on the screen, except the sentence of R16 on the file of earlier releases
  (R13, R17)
- [ ] Run the five commands of the suite (R20)
- [ ] Commit: `bugfix-0022 Document the upload of the theme at every start`

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [ ] `npm run check-changes` passes on the committed branch

## Commit Points

- [ ] `bugfix-0022 Upload the theme at every start of run and decide reloads in memory` (task 1)
- [ ] `bugfix-0022 Upload a theme file without the state file and remove the state file` (task 2)
- [ ] `bugfix-0022 Remove the serial of the open transport` (task 3)
- [ ] `bugfix-0022 Upload at once after a restart that recovered the screen` (task 4)
- [ ] `bugfix-0022 Document the upload of the theme at every start` (task 5)

## Trace

| Requirement | Tasks |
|---|---|
| R1, R2, R3, R4, R5 | 1 |
| R6 | 1 |
| R7 | 1, 4 |
| R8 | 1 |
| R9 | 1, 2 |
| R10 | 1 |
| R11 | 1, 2 |
| R12 | 2, 3 |
| R13, R14, R15, R16, R17 | 5 |
| R18 | 1, 2 |
| R19 | 1, 2 |
| R20 | 1, 2, 3, 4, 5 |
| R21, R22 | 4 |
