# The clock of run sent every second

## Approach

The feeder of `run` in `internal/cli/run.go` gets a second kind of wake-up besides its rounds: a tick at each whole
second of the time, which writes only the clock. The schedule stays in `feed`, which keeps the grid of the rounds as
it is today and adds the next whole second as a second candidate for the next wake-up:

* `feed` keeps `nextRound`, the place of the next round on the grid of the interval, computed exactly as today with
  `index = max(index+1, int(end.Sub(first)/interval))` after each round. It starts at `first`
* Each pass of the loop takes `start := feeding.now()`. When `start` is not before `nextRound` it does a round, which
  then updates `nextRound`; otherwise it does a tick. A round and a tick at the same moment are therefore one round
* After the round or the tick, the next wake-up is the earlier of `nextRound` and
  `start.Truncate(time.Second).Add(time.Second)`, the first whole second after the start of what just ended. When it
  is after `now()` the feeder waits for it, and otherwise it goes on at once, so passed seconds and passed rounds are
  not made up
* The test whether the clock is to be written is one helper used by both the round and the tick, so that the rule of
  R2 is written once: the clock is due, or the whole second of `start` differs from the whole second of `lastClock`.
  Both are compared with `Truncate(time.Second)`, which compares the seconds of the time of the system and makes a
  time set back count as a different second, as D4 decides. The helper writes the clock with `writeClock` and clears
  `clockDue` once it was written. `clockPeriod` and its comment are removed, and the comment on the helper says that
  the screen shows the time of the last clock report and does not advance it, so the clock is written once a second
* The round passes its `start` on from `feed` instead of taking its own, and otherwise keeps all it does today: the
  reload, the reopening, `holder.ensure`, the clock and the values. Only the condition of its clock changes
* The tick does nothing but the clock: it returns at once when the screen is lost or the theme holder says the theme
  of the config is not known to be on the screen, and otherwise calls the helper. A failed write goes through
  `feeding.write` and so marks the screen lost with the line `screen lost: <error>`, and the next round reopens it.
  An error of `EncodeClock` ends `feed` as it does in a round
* `themeHolder` in `internal/cli/themeholder.go` gets `onScreen() bool`, which tells without reading the state file or
  uploading whether values may be sent: true with no theme, and otherwise only when no check is due and the theme is
  not due. It does not change the holder

The existing tests that pin the minute are replaced by those of R10. Several existing tests change their expected
reports, because every round of a new second now writes the clock, and the tests of the keeper with an interval of
10 seconds count waits, of which there are now ten times as many. A prototype of the schedule run against the suite
showed which tests these are, and they are listed in the tasks below.

## Files

- `internal/cli/run.go`: the schedule of rounds and ticks in `feed`, the tick, the shared clock helper, `round` taking
  its start, the removal of `clockPeriod`, and the comments of `feeder` and of the clock helper
- `internal/cli/themeholder.go`: `onScreen`
- `internal/cli/themeholder_test.go`: the unit test of `onScreen`
- `internal/cli/run_test.go`: the new tests of R10 for the moments of the clock, the screen lost and the failed write
  of a tick; the removal of the two tests that pin the minute; updated expectations of existing tests
- `internal/cli/runtheme_test.go`: the new test of R10 for a tick while the theme is not on the screen; updated
  expectations of existing tests
- `internal/cli/runreload_test.go`: updated expectations of existing tests
- `internal/cli/keeper_test.go`: the counts of waits of the tests with an interval of 10 seconds
- `e2e/run_test.go`: the end-to-end test of R11
- `e2e/commands_test.go`: the comment at the top that lists what the suite leaves out
- `README.md`: the sentence on the clock of `run`
- `docs/protocol.md`: the section on the clock

## Tasks

### The tests that fail before the change

- [x] In `internal/cli/themeholder_test.go`, add a test of `onScreen` with the fixture of the file: true for a holder
  with no theme; false for a holder with a theme before its first check; true after an `ensure` that found the hash in
  the state file or completed an upload; false after an `ensure` whose upload failed; false after `screenBack` and
  after `setTheme` with a theme of another hash, until the next `ensure`. It does not compile until `onScreen` exists
  (R3)
- [x] In `internal/cli/run_test.go`, add a helper that gives, for each report written, its kind and the moment of the
  fake clock at which it was written, using `afterWrite` of the fake transport, and a helper that decodes the time of a
  clock report as `expectClockReport` of `e2e/commands_test.go` does, in UTC since the harness runs in UTC (R10)
- [x] Add a test with `interval: 1s` that starts on a whole second, as the harness does, and stops after 5 waits:
  the writes are `CVCVCVCVCV`, from the five rounds at 0 to 4 seconds, each clock written at the start of its round
  with the time of that second, and the waits are five of one second each, so there is no tick apart from the rounds
  (R1, R2, R10)
- [x] Add a test with `interval: 5s` that stops after 11 waits: one clock report at each whole second from 0 to 10,
  with consecutive seconds, and value reports only at 0, 5 and 10 seconds, each directly after the clock of its second
  (R1, R2, R10)
- [x] Add a test with `interval: 300ms` that runs for a little over two seconds: clock reports written at 0, 1 and 2
  seconds only, each with the time of its second; value reports at every multiple of 300 milliseconds; the rounds at
  1.2 and 2.1 seconds write values and no clock, and the ticks at 1 and 2 seconds write only the clock (R1, R2, R10)
- [x] Add a test that sets the fake clock to 400 milliseconds after a whole second before the run, with
  `interval: 1s`: the clock and the values at the start; at the next whole second a tick that writes the clock alone;
  at 400 milliseconds after it a round that writes values and no clock; and so on for three seconds (R1, R2, R10)
- [x] Add a test whose first read of the source moves the clock by 2.5 seconds: the next round follows at once at
  2.5 seconds and writes the clock of second 2 and then the values; no clock report has the time of second 1; the
  round after it is at 3 seconds. This replaces `TestRunSetsTheClockByTheTimeARoundStartsNotByItsPlaceOnTheGrid`, and
  `TestRunDoesNotMakeUpMissedRounds` stays as it is (R1, R10)
- [x] Add a test with `interval: 5s` in which the value report of the first round fails and the first reopening
  succeeds: the ticks at 1 to 4 seconds write nothing and make no reopening, the round at 5 seconds reopens and writes
  the clock and the values, and the line is `screen lost: fake failure\nscreen back\n` (R3, R10)
- [x] Add a test with `interval: 300ms` in which the value report of the second round, at 300 milliseconds, fails and
  the reopening in the round at 600 milliseconds succeeds: that round writes the clock again, within second 0 whose
  clock was written at the start, and then the values (R2, R10)
- [x] Add a test with `interval: 5s` in which the clock written by the tick at 1 second fails: stderr holds
  `screen lost: fake failure\n` from that tick on, the ticks at 2 to 4 seconds write nothing and make no reopening,
  and the round at 5 seconds reopens and writes `screen back` (R3, R10)
- [x] In `internal/cli/runtheme_test.go`, add a test with `interval: 5s` and a theme whose first upload is refused, as
  `TestRunTriesAFailedUploadAgainAfterTenSeconds` sets it up: nothing is written by the ticks; the second upload starts
  at a place of a round on the grid of 5 seconds, the first one at least 10 seconds after the moment of the failure,
  and never at a whole second between two rounds, which the test checks by the moment of the fake clock at the first
  write of each upload; after it completes the clock and the values follow, so the kinds are `UCV`, and stdout holds
  `uploading theme` twice (R3, R4, R10)
- [x] In `e2e/run_test.go`, add a test that runs `run` with `interval: 3s` and the tags of `runConfig` against the fake
  screen, waits for four reports within 6 seconds, and checks that they are a clock report, the value report
  `valueReportOf42And1234`, and two clock reports whose times are one and two seconds after the time of the first; the
  first passes `expectClockReport` with the bounds taken around the start, as `TestRunSendsTheClockAndThenTheValues`
  does (R11)
- [x] Run `go test ./internal/cli/ ./e2e/` and see the new tests fail, other than those that only pin what holds today
  (R10, R11)

### The schedule

- [x] Add `onScreen` to `themeHolder` in `internal/cli/themeholder.go`, as the approach says (R3)
- [x] In `internal/cli/run.go`, add the clock helper used by round and tick, with the comment that the screen shows the
  time of the last clock report and does not advance it, and make `round` take its start from `feed` and use the
  helper in place of its condition on `clockPeriod`; remove `clockPeriod` and its comment (R2, R4, R6)
- [x] Add the tick to the feeder, as the approach says: nothing while lost or while `onScreen` is false, and otherwise
  the helper (R2, R3)
- [x] Change `feed` to the schedule of rounds and ticks of the approach, keeping the grid of the rounds and the rule
  that missed rounds are not made up, and update the comment of `feeder` so that it names the ticks of the clock
  between the rounds (R1, R4)
- [x] Leave the command `clock` in `internal/cli/commands.go` untouched, and confirm that its tests in
  `internal/cli/commands_test.go` and `e2e/commands_test.go` pass unchanged (R5)

### The existing tests

- [x] In `internal/cli/run_test.go`, remove `TestRunSetsTheClockAgainAfterSixtySecondsAndNotBefore` and
  `TestRunSetsTheClockByTheTimeARoundStartsNotByItsPlaceOnTheGrid`, which the tests above replace, and change the
  expected writes of `TestRunWritesNoReportWhenEveryMetricIsUnavailable` from `C` to `CC`, one clock in each of the two
  rounds (R6, R10)
- [x] In `internal/cli/runreload_test.go`, change the expected reports of
  `TestRunUploadsTheThemeAfterAChangeFromTagsToThemeAndSendsItsTags`,
  `TestRunUploadsNothingAfterAChangeFromThemeToTags`,
  `TestARewriteOfTheSameThemeUploadsNothingAndAnEditedFolderIsUploaded` and
  `TestAReloadToAThemeWithNoCacheDirectoryIsRefusedWithItsLine` to the clock in every round of a new second, each to
  the exact sequence that the schedule gives, and keep what each test is about: the upload, the absence of one, and
  the line of the refusal (R4)
- [x] In `internal/cli/runtheme_test.go`, change the expected reports of
  `TestRunTriesAFailedUploadAgainAfterTenSeconds` to the exact sequence that the schedule gives, `UCVCVCV` in the
  prototype, and update the comment in `themeKinds` that explains why an upload makes the next round follow at once,
  if it no longer holds (R4)
- [x] In `internal/cli/keeper_test.go`, scale the counts of waits of the tests with an interval of 10 seconds by ten,
  so that each still covers the same span of time now that `run` waits once a second: 6 to 60 in
  `TestRunKeepsTheControllerAwakeWithAnIntervalOfTenSeconds` and `TestRunKeepsTheControllerAwakeWhileTheScreenIsLost`,
  3 to 30 in `TestRunAsksForNoReadingWhenNoProbeIsMappedAndKeepsTheSession`, 100 to 1000 in
  `TestRunWaitsLongerAfterEachFailedSetup`, and 10 to 100 in
  `TestRunSetsUpAgainTenSecondsAfterALostSessionAndLightsTheColourAgain`, `TestRunWritesTheLinesOfTheControllerOnceEach`
  and `TestRunWritesNoLineWhenThereIsNoController`. Where a test still fails after that, fix the count it relies on
  rather than what it checks of the keeper, and write in a comment of the test which span of time the count covers
  (R4)
- [x] Run `go test ./...` and fix any other test whose expected reports change for the clock alone, in the same way:
  the expected reports follow the schedule, and what the test checks otherwise stays (R4, R12)

### What the project says

- [x] In `e2e/commander_test.go`, change `TestRunSendsTheKeptProbesBetweenReadingsOfTheController` to count only the
  value reports that follow the change, because a clock report now stands between value reports once a second (R4)
- [x] In `README.md`, change the sentence of `sensordeck run` to say that it sets the clock of the screen when it
  starts and once a second, and keep the rest of the sentence, that it keeps running when the screen is unplugged and
  plugged in again (R7)
- [x] In `docs/protocol.md`, add to the section `## Clock`, after the example of the report, a paragraph that says the
  screen shows the time of the last clock report it received and does not advance it, as seen on the device on
  2026-11-15, so a program sends the clock once a second for the seconds on the screen to change (R8)
- [x] In the comment at the top of `e2e/commands_test.go`, change the item on the clock of `run` so that it names only
  the reopening that fails before it succeeds, and no longer the clock of `run` after 60 seconds (R9)
- [x] Search the code, the tests, `README.md` and `docs/` for `keeps time`, `once a minute`, `60 seconds`, `sixty`,
  `drift` and `clockPeriod`, outside `changes/`, and confirm that none says the screen keeps time itself or that `run`
  sets the clock once a minute (R6)

## Verification

- [x] `go test ./...` passes (R12)
- [x] `go vet ./...` reports nothing (R12)
- [x] `gofmt -l .` prints no file name (R12)
- [x] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing (R12)
- [x] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing (R12)
- [ ] `npm run check-changes` succeeds once the work is committed

## Commit Points

- [x] `bugfix-0017 Write the plan of sending the clock of run every second`: `plan.md` and the entry of `record.md`
- [ ] `bugfix-0017 Send the clock of run once a second`: the new tests, the schedule, `onScreen` and the updated
  existing tests, with the suite passing
- [ ] `bugfix-0017 Say that run sends the clock once a second`: `README.md`, `docs/protocol.md` and the comment of
  `e2e/commands_test.go`
