# Reading the Commander Core XT less often in run

## Approach

The period is a matter of `run` alone (D8), so the driver of the controller, `internal/commandercore`, does not
change (R9). Three parts change:

* `internal/config`: the key `commander_interval`, its checks and the two texts that list the keys (R1 to R3)
* `internal/cli`: the sources are given with the controller apart from the others, and the loop of `run` reads the
  controller through a small type that keeps its readings and the start of the round that made them (R4 to R8, R10)
* the documents: `README.md` and `docs/commander-core-xt.md` (R11, R12)

### The config

`Config` holds what the file holds, as its doc comment says, so the new field is what the file gives and is zero
without the key. The period itself is a method, so that the default lives beside the parsing and every existing test
that compares a whole `Config`, such as `TestParseTheExample`, keeps what it expects (R17).

```go
// DefaultCommanderInterval is the shortest time between two readings of the controller when the file does not say.
DefaultCommanderInterval = time.Minute

type Config struct {
	Interval          time.Duration
	CommanderInterval time.Duration // zero when the file does not give it
	Mappings          []Mapping     // ordered by tag, ascending
}

// CommanderPeriod is the shortest time between two readings of the controller in run.
func (config Config) CommanderPeriod() time.Duration
```

`CommanderPeriod` gives `CommanderInterval` when it is not zero, and otherwise `max(DefaultCommanderInterval,
Interval)` (R1). A file cannot give a `commander_interval` of zero, because it would be shorter than the interval,
which is at least 100 milliseconds, so zero is free to mean "not given".

`Parse` collects the node of `commander_interval` in its loop over the keys, beside those of `tags` and `interval`,
and checks it last, after the interval, with `parseCommanderInterval(node *yaml.Node, interval time.Duration)
(time.Duration, error)`. It follows `parseInterval`: a nil node gives 0; the text is the value of a scalar node and the
empty text otherwise; a text that `time.ParseDuration` refuses gives `commander_interval: "<text>" is not a duration
such as 1m or 30s`; a duration below the interval gives `commander_interval: <text> is shorter than the interval
<interval>`, with `interval.String()`; a duration equal to the interval is valid (R2). Checking it last keeps the order
in which the problems of a file are found today, and is the one order in which "the interval" of the text is known.

The two texts of R3 become `want a mapping with the keys commander_interval, interval and tags` and `unknown key
"<key>", the keys are commander_interval, interval and tags`.

### The sources

`environment.newSources` gives a `sourceSet` instead of a `[]sensors.Source`, so that `run` can tell the controller
from the others without knowing anything of its metrics:

```go
// sourceSet is the sources of the readings, with the controller apart, because run reads it less often than the
// others.
type sourceSet struct {
	others     []sensors.Source
	controller sensors.Source
}

// all gives every source, the controller last, in the order in which sensors reads them.
func (set sourceSet) all() []sensors.Source
```

`all` appends the controller to `slices.Clip(set.others)`, so that it never writes into the array of `others`. The
type is in a new file, `internal/cli/sources.go`, which is not limited to a platform, because `main.go` names it.
`sources_linux.go` gives `sourceSet{others: linuxsensors.Sources(sysfsRoot, procRoot), controller: controller}`, and
`sources_other.go` gives `sourceSet{}` with the error of today. `sensors` reads `sources.all()`, which is the list of
today in the order of today, so its output and its one reading of the controller do not change (R10).

### The loop of run

A new file `internal/cli/periodic.go` holds the type that reads the controller at most once per period:

```go
// periodicSource reads a source at most once per period and gives the readings of its last reading in between.
type periodicSource struct {
	source   sensors.Source
	period   time.Duration
	read     bool              // whether a reading was made
	lastRead time.Time         // the start of the round that made the last reading
	kept     []sensors.Reading // exactly what the last reading gave
}

// readings reads the source when no reading was made yet or when roundStart is at least the period after the start
// of the round that made the last one, and gives the kept readings.
func (periodic *periodicSource) readings(stop context.Context, roundStart time.Time, wanted func(name string) bool) []sensors.Reading
```

When it reads, it sets `kept` to what `source.Read(stop, wanted)` gave, whatever that is, nil included, and
`lastRead` to `roundStart` and `read` to true (R4, R6). It does not look at what the reading gave, so a reading that
gives nothing or some of the probes counts as a reading and the next comes a full period later (R6, D3, D4). Otherwise
it does not call the source at all (R4).

The feeder changes so:

* its field `sources` becomes `others []sensors.Source` and `controller *periodicSource`. `runCommand` fills them from
  the `sourceSet`, with `period: settings.CommanderPeriod()`. The line `running: ...` does not change (R7)
* `round` takes `start := feeding.now()` as its first statement, before the branch of the lost screen, and passes it to
  `read`. That is the moment the round starts, on the clock of the rounds and of the clock report (R4, D7). A round
  whose reopening fails returns before `read`, as today, so it reads no source and leaves the controller and its
  kept readings as they are. A round whose reopening succeeds goes on with the round, R22 of unit 0004, and reads the
  controller only when the period has passed since `lastRead`, which the feeder kept across the lost screen (R8)
* `read(stop, start)` builds `wanted` from the mappings as today, reads `sensors.ReadAll(stop, feeding.others,
  wanted)`, appends `feeding.controller.readings(stop, start, wanted)`, and turns the readings into pairs ordered by
  tag as today (R5, R7). The controller is still asked for the mapped names only, as R21 of unit 0004 says (R4)

Every other source is read in every round, as before (R7, D9). The moments of the rounds, the clock and the output do
not change, because `feed` and `writeClock` do not change (R7).

R8 says that the rounds in which `run` calls `Reopen` read no source. In the tree, as R22 of unit 0004 says, a round
whose reopening fails is given up, and a round whose reopening succeeds writes the clock and goes on with the round.
The plan reads R8 by the second half of that rule, which is the one its second sentence states: the reading that
follows a reopening that succeeds is that of the same round, and it reads the controller only when R4 says so. This
keeps `TestRunReopensALostScreenOncePerRound` as it is (R17).

### The test harness

`harness` keeps `sources []sensors.Source` for the others and gains `controller sensors.Source`, which `newHarness`
sets to `&fakeSource{}`, a controller that gives nothing, so that no existing test needs to set it. Its `newSources`
gives `sourceSet{others: harness.sources, controller: harness.controller}`.

The tests of `run` use a controller made by a helper `twoProbes() *fakeSource`, with the names `commander.1.temp`
and `commander.2.temp` and the values 37 and 23, a config `controllerTags` of `tags:\n  cpu.temp: 1\n
commander.1.temp: 4\n  commander.2.temp: 5\n`, with `interval` and `commander_interval` before `tags` where a test
needs them, and a helper `runWithController(others, controller *fakeSource, stopAfterWaits int) *harness` built as
`runWith` is. `fakeSource` already counts its calls in `readsAt`, and its `onRead` records the moment of each call from
`harness.clock.current` or moves the clock.

### The end-to-end tests

`runningSetup` of `e2e/run_test.go` keeps its signature and builds its tree with a new helper
`runningTree(test, fake) (*fakesensors.Tree, string)`, which makes the tree of today, the `k10temp` entry with `Tctl`
of 42000 and the CPU frequency, and gives the tree and the path of the `k10temp` entry. This is the tree of R35 of unit
0004 that R15 and R16 name.

R16 replaces `temp1_input` of that entry while `run` reads it. `fakesensors` gains `(*Tree).ReplaceInput(entry string,
number int, input string)`, which writes `<entry>/temp<number>_input` with the input and a trailing newline, as `Hwmon`
writes it, through a temporary file in the same directory renamed into place. The rename of `NetDev` and this one are
the same knowledge, so it is extracted into a private `(*Tree).replace(path, content string)` that both call. The
extraction is its own commit, before the new helper, since it changes no behaviour.

The two tests are in `e2e/commander_test.go`, with a helper that starts `run` with the screen, the tree of
`runningTree`, a fake controller `fakecommander.New(test, fake.SysfsRoot, fake.DevRoot).Add(9, 10)` and the config
`interval: 100ms\ncommander_interval: <period>\ntags:\n  cpu.temp: 1\n  commander.1.temp: 4\n`, and gives what the
tests need. They find reports with `waitForReport` and `valuePairs` of `e2e/run_test.go`. The writes of the fake are
taken with `controller.Writes()` after the SIGTERM, because a controller that is read every 100 milliseconds is never
quiet while `run` runs, and a count of writes taken later can only be larger.

## Files

- `internal/config/config.go`: the key, its checks, `DefaultCommanderInterval`, `CommanderPeriod`, the texts of R3
- `internal/config/config_test.go`: the tests of R13
- `internal/cli/sources.go`: new, `sourceSet`
- `internal/cli/sources_linux.go`, `internal/cli/sources_other.go`: give a `sourceSet`
- `internal/cli/main.go`: the type of `newSources`
- `internal/cli/commands.go`: `sensors` reads `sources.all()`
- `internal/cli/periodic.go`: new, `periodicSource`
- `internal/cli/run.go`: the feeder reads the controller through `periodicSource`
- `internal/cli/commands_test.go`: the harness gives a `sourceSet`
- `internal/cli/sensors_test.go`: the test that `sensors` reads the controller once, after the others
- `internal/cli/run_test.go`: the tests of R14
- `internal/fakesensors/fakesensors.go`: `replace` and `ReplaceInput`
- `e2e/run_test.go`: `runningTree`
- `e2e/commander_test.go`: the tests of R15 and R16
- `README.md`: the text of R11
- `docs/commander-core-xt.md`: the text of R12

## Tasks

### The config

- [x] In `internal/config/config_test.go`, change the four cases of `TestParseErrors` named `empty`, `comment`,
  `list` and `unknown key` to the texts of R3, and add (R3, R13):
    - a test that `commander_interval: 30s` with `interval: 1s` gives a `CommanderInterval` of 30 seconds and a
      `CommanderPeriod` of 30 seconds
    - a test that a file without the key gives a `CommanderInterval` of 0 and a `CommanderPeriod` of 1 minute, and
      that one with `interval: 2m` and no key gives a `CommanderPeriod` of 2 minutes
    - a test that `interval: 500ms` with `commander_interval: 500ms` is valid, with a `CommanderPeriod` of 500
      milliseconds
    - cases of `TestParseErrors` with valid tags: `interval: 2s` with `commander_interval: 1s` gives
      `commander_interval: 1s is shorter than the interval 2s`; `commander_interval: 500ms` without `interval` gives `commander_interval:
      500ms is shorter than the interval 1s`; `commander_interval: soon` gives `commander_interval: "soon" is not a
      duration such as 1m or 30s`; `commander_interval: {a: 1}` gives `commander_interval: "" is not a duration such
      as 1m or 30s`; and `commander_interval: -1m` gives `commander_interval: -1m is shorter than the interval 1s`

  `TestParseTheExample` and the other tests stay as they are. The new tests fail to compile until the next task
- [x] In `internal/config/config.go`, add `DefaultCommanderInterval`, the field `CommanderInterval`, the method
  `CommanderPeriod`, the case `commander_interval` of the loop of `Parse`, `parseCommanderInterval` called after
  `parseInterval`, and the texts of R3, as "The config" says, until `go test ./internal/config` passes (R1, R2, R3)
- [x] Commit: `feature-0008 Add commander_interval to the config file`

### The sources apart

- [ ] In `internal/cli/sensors_test.go`, add a test that `sensors` with a harness whose `sources` give `cpu.temp` and
  whose `controller` is a `fakeSource` giving `commander.1.temp` of 37.3 prints the line of `cpu.temp` and then that
  of `commander.1.temp`, and that the controller got exactly one call (R10). It fails to compile until the next task
- [ ] Add `internal/cli/sources.go` with `sourceSet` and `all`, as "The sources" says. Change the type of `newSources`
  in `internal/cli/main.go` to `func(sysfsRoot, devRoot, procRoot string) (sourceSet, error)`, make
  `sources_linux.go` and `sources_other.go` give a `sourceSet`, make `sensors` in `internal/cli/commands.go` read
  `sources.all()`, and make the harness in `internal/cli/commands_test.go` give `sourceSet{others: harness.sources,
  controller: harness.controller}` with `controller: &fakeSource{}` set in `newHarness`. In `run.go`, the feeder takes
  `sources.all()` in this task, so that `run` behaves as before. The new test and every existing test pass (R10, R17)
- [ ] Run the five commands of the suite in `AGENTS.md`, the two Windows ones among them, and commit:
  `feature-0008 Give the sources with the controller apart`

### The period in run

- [ ] In `internal/cli/run_test.go`, add `twoProbes`, `controllerTags` and `runWithController` as "The test harness"
  says, and these tests, each recording the moments of the calls of the controller in its `onRead` where it checks
  them (R14):
    - with no `commander_interval` and a stop after 121 waits, so rounds at 0 to 120 seconds: the controller is called
      at exactly the start, 60 and 120 seconds, three calls (R4, R14)
    - with a stop after 3 waits and an `onRead` of the other source that sets `cpu.temp` to the number of its call:
      the three value reports are those of the pairs (1, *n*), (4, 37) and (5, 23) for *n* of 1 to 3, and the
      controller got one call (R5, R7, R14)
    - a controller with no values and a stop after 60 waits, so rounds at 0 to 59 seconds: every value report is that
      of the one pair (1, 42), and the controller got one call (R6, R14)
    - a controller that gives only `commander.1.temp` at its first call, with `commander_interval: 2s` and an `onRead`
      that sets `commander.2.temp` to 23 at its second call, and a stop after 3 waits: the reports of the rounds at 0
      and 1 second hold (1, 42) and (4, 37), and that of the round at 2 seconds (1, 42), (4, 37) and (5, 23) (R4, R6,
      R14)
    - with `commander_interval: 2s`, an `onRead` that removes both values at the second call, and a stop after 4 waits:
      the reports of the rounds at 0 and 1 second hold (1, 42), (4, 37) and (5, 23), and those of the rounds at 2 and 3
      seconds only (1, 42) (R4, R6, R14)
    - with `interval: 1s` and `commander_interval: 1s` and a stop after 3 waits: the controller got three calls (R4,
      R14)
    - a controller whose `onRead` records the moment and then moves the clock by 500 milliseconds, stopped at its third
      call: the calls are at the start, 60 and 120 seconds, and not at 61 and 122 seconds as a period measured from the
      end of the reading would give (R4, D7, R14)
    - an other source whose `onRead` moves the clock by 1.6 seconds at its 60th call, the round at 59 seconds, so
      that the next round starts late at 60.6 seconds, with the controller stopped at its third call: the calls are at
      the start, 60.6 and 121 seconds, and not at 120 seconds as a period on the grid would give (R4, D7, R14)
    - a screen lost at the first value report, with `failWrites` of `false, true`, `reopenResults` of `errFake, nil`
      and a stop after 3 waits: the writes are `CCV`, the last that of (1, 42), (4, 37) and (5, 23), and the
      controller got one call, so neither the failed reopening nor the round of the reopening that succeeded read it
      (R8, R14)

  The tests of the controller's calls fail until the next task, since the feeder reads every source in every round
- [ ] Add `internal/cli/periodic.go` with `periodicSource`, as "The loop of run" says, and change the feeder of
  `internal/cli/run.go`: the fields `others` and `controller`, filled in `runCommand` from the `sourceSet` and
  `settings.CommanderPeriod()`, `start := feeding.now()` as the first statement of `round`, and `read(stop, start)`
  that joins the readings of the others and of `controller.readings`. Update the doc comments of `feeder` and `read`
  to say that the controller is read at most once per period and its kept readings are sent in between. The new tests
  and every existing test of `internal/cli` pass, among them `TestRunReopensALostScreenOncePerRound` and the tests of
  the clock unchanged (R4 to R8, R17)
- [ ] Run the five commands of the suite and commit: `feature-0008 Read the controller at most once per period in
  run`

### The end-to-end tests

- [ ] In `internal/fakesensors/fakesensors.go`, extract the write to a temporary file and the rename of `NetDev` into
  `(*Tree).replace(path, content string)`, which writes `<path>.tmp` and renames it to `path`, and make `NetDev` call
  it. `NetDev` keeps its doc comment and its behaviour, and the tests of the network pass as they are. Commit:
  `feature-0008 Extract the rename into place of the fake sensors`
- [ ] Add `(*Tree).ReplaceInput(entry string, number int, input string)` to `internal/fakesensors/fakesensors.go`,
  with a doc comment, calling `replace` with `<entry>/temp<number>_input` and the input with a trailing newline
- [ ] In `e2e/run_test.go`, add `runningTree` as "The end-to-end tests" says and make `runningSetup` use it, so that
  the tests of `run` keep their tree and pass as they are
- [ ] In `e2e/commander_test.go`, add the helper that starts `run` with the controller and the config of a given
  period, and (R15, R16):
    - a test with `commander_interval: 100ms`: once a value report holds (4, 37), it calls
      `controller.SetProbes(fakecommander.Probe{Tenths: 400}, fakecommander.Probe{Tenths: 227})`; a later value report
      holds (4, 40) within `reportLimit`; after SIGTERM the status is 0, `controller.Writes()` holds the write of step
      1, `controllerWrite(1)`, at least twice, and the controller is in hardware mode (R15)
    - a test with `commander_interval: 1h`: once a value report holds (1, 42) and (4, 37), it sets the probes to 40.0
      °C as above and calls `ReplaceInput` on the `k10temp` entry for `temp1` with `43000`; a later value report holds
      (1, 43) and (4, 37) within `reportLimit`; the three value reports after it, awaited within `reportLimit` by
      polling `screen.Reports()`, each hold (4, 37); after SIGTERM the status is 0,
      `expectWrites(test, controller.Writes(), 1, 2, 3, 4, 5, 6, 7, 8)` holds, and the controller is in hardware mode
      (R16)

  `TestRunSendsTheProbesOfTheController` stays as it is (R17)
- [ ] Run the five commands of the suite and commit: `feature-0008 Test the period of the controller end to end`

### The documents

- [ ] In `README.md`, in "Showing sensor values on the screen", after the paragraph on `interval`, add a paragraph
  saying that `commander_interval` is the shortest time between two readings of the Commander Core XT, at least the
  interval and 1 minute without the key (or the interval when it is longer); that each reading makes the LEDs of the
  fans on the controller flicker once, which is why it is read less often; that between two readings `run` sends the
  values of the last reading; and that `sensordeck sensors` reads the controller once each time it runs (R11)
- [ ] In `docs/commander-core-xt.md`, in the project's own words (R12):
    - add a section "Software mode and the LEDs" after "Times" that says that on the device the LEDs of the fans on the
      controller flicker once at each reading, because nothing drives them in software mode, and that the status
      command of the other tool makes them flicker in the same way; that the endpoint `0x21` was opened and read with
      no step 1 before it, and that every answer then had byte 2 `0x03` and stale data in the rest, so that a reading
      without software mode is not possible with what is known; that byte 2 of an answer is a status, `0x00` in the
      answers of a sequence that step 1 opened and `0x03` for the commands made without it; and that `sensordeck run`
      reads the controller at most once per period, of 1 minute unless the config says otherwise, so that the LEDs
      flicker less often
    - change the bullet of "Unknown" on the meaning of byte 2 to say that byte 2 is a status, `0x00` in the answers
      of a sequence that step 1 opened and `0x03` for commands made without it, and that the meaning of any other
      value is not known
    - change the bullet of "Unknown" on software mode to say that the controller does not drive the LEDs in software
      mode, and that what it does with the fans when nobody sends it settings is not known
    - change the bullet of "Assumptions" that byte 2 is a status, so that it holds only what is still assumed: that
      an answer whose byte 2 is not `00` is not valid, byte 2 being the status that the new section describes

  Nothing else of the document changes
- [ ] Commit: `feature-0008 Document the period of the controller`

### Before handing over

- [ ] Run the five commands of the suite and `npm run check-changes`, and check every requirement R1 to R17 against
  the tree, ticking this task only when all pass (R1 to R17)
