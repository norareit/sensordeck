# A static colour for the LEDs of the Commander Core XT in run

## Approach

Four parts change, each with one job:

* `internal/config`: the key `commander_color`, its checks and the texts that list the keys (R1 to R3)
* `internal/fakecommander`: a fake controller that knows channels, endpoints, modes, LEDs and a software-mode timeout
  (R22)
* `internal/commandercore`: a session of the controller, its setup, its readings and its teardown, on Linux (R5 to
  R10, R12, R13, R18)
* `internal/cli`: a keeper that holds the session in a goroutine of its own beside the rounds of `run`, with the
  keepalive, the back-off and the lines of standard error (R4, R11, R13 to R16)

Then the end-to-end tests (R26) and the documents (R19 to R21). Without `commander_color` nothing of the keeper is
made, and `run` reads the controller through `periodicSource` as unit 0008 made it (R4). `sensors` does not change
(R17).

### The config

```go
// Colour is a colour of the LEDs, as commander_color gives it.
type Colour struct{ Red, Green, Blue uint8 }

type Config struct {
	Interval          time.Duration
	CommanderInterval time.Duration // zero when the file does not give it
	CommanderColour   *Colour       // the key commander_color; nil when the file does not give it
	Mappings          []Mapping     // ordered by tag, ascending
}
```

The identifiers spell colour as the project spells utilisation, and the doc comment names the key. A nil pointer is the
key not given, so every existing test that compares a whole `Config`, such as `TestParseTheExample`, keeps what it
expects (R27).

`Parse` collects the node of `commander_color` in its loop over the keys and checks it after `commander_interval`,
with `parseCommanderColour(node *yaml.Node) (*Colour, error)`:

* a nil node gives nil
* a scalar node whose short tag is `!!null`, or whose value is empty, gives the first text of R2. An unquoted `#00c8ff`
  arrives as such a node, since YAML reads the rest of the line as a comment
* the text is the value of a scalar node and the empty text otherwise; a text that is not `#` followed by exactly six
  hexadecimal digits, of either case, gives the second text of R2 with the text quoted by `%q`. The digits are decoded
  with `encoding/hex`, which takes both cases
* otherwise the colour

After the colour, a file that gives both `commander_color` and `commander_interval` gets the third text of R2. Checking
the colour before the conflict and after every other key keeps the order of today for the problems that were already
found, and makes one of them reported, as R18 of unit 0004 asks.

The texts of R3 become `want a mapping with the keys commander_color, commander_interval, interval and tags` and
`unknown key "<key>", the keys are commander_color, commander_interval, interval and tags`.

### The fake controller

The fake keeps, under its mutex, the endpoint open on each channel, the mode, the LED ports, the colour writes, what
its LEDs show and a software-mode timeout. It answers a command so, in this order:

1. a mode command, `08 01 03 00 02` or `08 01 03 00 01`, sets the mode and answers status `00`. Entering software mode
   makes the LEDs dark; returning to hardware mode makes them show the stored lighting. A mode command closes no
   channel
2. while the mode is not software, `NoMode` included, every other command answers status `03` (D18)
3. `08 0D <channel> <endpoint>`, an open: status `03` when the channel has an endpoint open, the same one included,
   which is the case D11 describes; otherwise it opens the endpoint and answers `00`
4. `08 08 <channel> ...`, a read: answered by the endpoint open on the channel, whatever the bytes after the channel.
   `0x17` gives the answer of the fan speeds of today, `0x21` the probes of today, `0x20` the data type `0F 00`, the
   count of the ports and their entries, mode and LED count, both low byte first, written as far as they fit into the
   96 bytes, and `0x1A` the data type `09 00`, the count 6 and six bytes `01`. Any other open endpoint answers status
   `00` and zeros. With no endpoint open on the channel there is no answer, as there is none today
5. `08 05 01 <channel> ...`, a close: closes the channel, whatever is open on it, and answers `00`. The `08 05 01` of
   unit 0005 is the close of channel 0
6. `08 06 00 ...`, a colour write: with `0x22` open on channel 0 it keeps the 96 data bytes, makes the LEDs show the
   colour of the triple at bytes 9 to 11, and answers `00`; otherwise it answers status `03`
7. any other command gets no answer, as today

The 0005 steps that `Fail` and `OnStep` take keep their meaning: they are the mode commands, and the open, read and
close of `0x17` and of `0x21` on channel 0 alone, worked out from the endpoint open on channel 0 before the command
changes it. The new failures and actions are chosen by the data bytes a command starts with:

```go
// Port is one entry of the answer of the endpoint 0x20: the mode of a port of the controller and its LED count.
type Port struct{ Mode, LEDs uint16 }

// Lighting is what the LEDs of the controller show.
type Lighting int

const (
	StoredLighting Lighting = iota // in hardware mode, and before any mode command
	Dark                           // in software mode before a colour write
	SentColour                     // in software mode after a colour write
)

func (controller *Controller) SetPorts(ports ...Port)                           // default: the seven entries of the host
func (controller *Controller) SetOpenEndpoint(channel, endpoint byte)            // as a killed program leaves it
func (controller *Controller) SetSoftwareModeTimeout(timeout time.Duration)      // zero, the default, is off
func (controller *Controller) FailCommand(prefix []byte, failure Failure)        // every later command with the prefix
func (controller *Controller) OnCommand(prefix []byte, action func())           // every command with the prefix, before its answer
func (controller *Controller) OpenEndpoint(channel byte) byte                    // 0 when none is open
func (controller *Controller) ColourWrites() [][]byte                           // the 96 data bytes of each one kept
func (controller *Controller) Lighting() (Lighting, [3]byte)                     // the colour for SentColour
```

The default ports are those of "What is known": mode 0 with 0 LEDs, mode 3 with 0 LEDs twice, and mode 2 with 4 LEDs
four times. A failure of `FailCommand` applies after the failure of `Fail`, with the same `applyFailure`, so that a test
sets one or the other. `BadStatus` stays `0x01`, which is the status that is not `00` of R22.

The timeout is a `time.AfterFunc` that every command on interface 0 resets while the mode is software. When it fires
with no command in between, the fake goes to hardware mode, closes every channel and shows the stored lighting, so that
the next session starts dark, as the device would (R22).

### The session of the controller

All of it is in `internal/commandercore`, which is limited to Linux (R18).

`protocol.go` gains the pure parts, each unit tested in `protocol_test.go`:

```go
const (
	channelZero, channelOne byte = 0x00, 0x01
	ledEndpoint   = 0x22
	portsEndpoint = 0x20
	maxLEDs       = (answerSize - 9) / 3 // 29: the triples that fit into a write of 97 bytes
)

var (
	enterSoftware  = []byte{0x08, 0x01, 0x03, 0x00, 0x02}
	returnHardware = []byte{0x08, 0x01, 0x03, 0x00, 0x01}
	portsDataType  = []byte{0x0F, 0x00}
	colourDataType = []byte{0x12, 0x00}
)

func openCommand(channel, endpoint byte) []byte  // 08 0D <channel> <endpoint>
func readCommand(channel, endpoint byte) []byte  // 08 08 <channel> <endpoint>
func closeCommand(channel, endpoint byte) []byte // 08 05 01 <channel> <endpoint>

// ledCount gives the LED count n of an answer of the endpoint 0x20 that is valid, by R7 of unit 0009.
func ledCount(answer []byte) (int, error)

// colourCommand gives the 96 data bytes of the colour write for n LEDs, by R8 of unit 0009.
func colourCommand(count int, red, green, blue byte) []byte
```

`steps` keeps its eight literal commands and its test, since they are the form of unit 0005; `enterSoftware` and
`returnHardware` are the same bytes as its steps 1 and 8, and `steps` is written with them so that the two modes are
written once.

`ledCount` refuses a data type other than `0F 00` with `data type <b3> <b4>` in hexadecimal, such as `data type 06 00`,
a count *c* with 6 + 4·*c* > 96 with `count <c> leaves no room for its entries`, and an *n* of 0 or more than 29 with
`LED count <n> is outside 1 to 29`.

`source.go`: `exchange` gives an error in place of false, as one refactoring commit before the session, so that the
session can name what failed and the teardown can tell an answer that came from one that did not:

```go
// exchangeError is a failed exchange. answered tells whether an answer came, which decides whether a teardown closes
// channel 0.
type exchangeError struct {
	answered bool
	problem  string
}
```

Its problems are `write failed: <error>`, `short write of <count> bytes`, `no answer within <limit>` for a deadline
that passed, `read failed: <error>` for any other read error, all with `answered` false, and `answer of <count> bytes`,
`echo <byte>` and `status <byte>`, bytes in hexadecimal as `03`, with `answered` true. `runSteps` and `runSequence`
treat an error as they treat false today, so that the reading of unit 0005 does not change (R27).

A new file `session.go` holds the session:

```go
// ErrNoController is the setup that found no node of the controller.
var ErrNoController = errors.New("no controller found")

// Lighting makes the sessions that light the LEDs of the controller in one colour.
type Lighting struct {
	SysfsRoot, DevRoot string
	TimeLimit          time.Duration
	Red, Green, Blue   byte
}

// Start makes a setup and gives the session it starts. On every error the setup has ended, with its teardown once
// the first command was written, and the node is closed.
func (lighting Lighting) Start(stop context.Context) (*Session, error)

// Session is a held session: the node open, the controller in software mode, the LEDs lit.
type Session struct {
	file      *os.File
	timeLimit time.Duration
}

// Read makes one reading and gives every available probe. On every error the session has ended with its teardown and
// the node is closed.
func (session *Session) Read(stop context.Context) ([]sensors.Reading, error)

// End makes the teardown with the close of channel 0 and closes the node.
func (session *Session) End()
```

`Start` gives `ErrNoController` when `findNode` finds none, and `cannot open <node>: <error>` when `hidraw.OpenNode`
fails, both with nothing written (R5). With stop done before the first exchange it closes the node and gives
`errStopped`, with nothing written (R13). Otherwise it makes the exchanges of R6 in order: enter, the close, open, read
and close of `0x20` on channel 1, whose read answer `ledCount` checks, the open of `0x22` on channel 0, and the colour
write of `colourCommand` (R6 to R8). Before every exchange after the first, a stop that is done ends the setup with
`errStopped`.

A step is named in its error as `<name>: <problem>`: `enter software mode`, `close of endpoint 0x20 on channel 1`,
`open of endpoint 0x20 on channel 1`, `read of endpoint 0x20 on channel 1`, `open of endpoint 0x22 on channel 0` and
`colour write`, and in a reading the same forms for `0x1A`, `0x17` and `0x21`. The names are made from the command by
one function, `stepName(command []byte) string`, so that the text and the bytes cannot disagree. A check of `ledCount`
or of `temperatures` is named after the read it checks, as `read of endpoint 0x20 on channel 1: LED count 30 is outside
1 to 29` and `read of endpoint 0x21 on channel 1: not an answer of the temperatures` (R15).

`Read` checks the stop before each of its twelve exchanges, the first included, and makes the four exchanges of R9 for
`0x1A`, `0x17` and `0x21` on channel 1, the answers checked by `valid`, and that of the read of `0x21` by `temperatures`
as well, whose probes it gives through `readingsOf` with every name wanted (R9). Nothing else is ever written, and only
to the node of interface 0 that `findNode` gives (R10).

The teardown is one private function, `teardown(file *os.File, timeLimit time.Duration, closeChannel bool)`: the close
`08 05 01 00 22` when `closeChannel`, then `08 01 03 00 01`, each by `exchange` with its result not used, then the close
of the node. A setup or a reading that ends by an error calls it with `closeChannel` false only when the error is an
`exchangeError` whose `answered` is false, and true for every other error: an answer that is not valid, a failed check
and a stop (R12, D12). `End` calls it with `closeChannel` true.

### The keeper in run

The keeper is not limited to a platform and names no Linux code (R18). A new file `internal/cli/keeper.go` holds:

```go
// errNoController is the setup that found no node of the controller, which writes no line.
var errNoController = errors.New("no controller found")

// heldSession is a held session of the controller. After an error of read it has ended with its teardown.
type heldSession interface {
	read(stop context.Context) ([]sensors.Reading, error)
	end()
}

const (
	keepalivePeriod = time.Second
	firstRetry      = 10 * time.Second
	longestRetry    = 5 * time.Minute
)

// sessionKeeper holds the session of the controller in a goroutine of its own, beside the rounds of run.
type sessionKeeper struct {
	start        func(stop context.Context) (heldSession, error) // a setup; on an error it has ended
	probesMapped bool
	now          func() time.Time
	after        func(duration time.Duration) <-chan time.Time
	stderr       io.Writer
	requests     chan chan []sensors.Reading
	done         chan struct{}

	// The fields below belong to the goroutine of keep.
	held     heldSession // nil while no session is held
	next     time.Time   // the next keepalive while held, the next setup otherwise
	retry    time.Duration
	lost     bool // "controller lost" was written and "controller back" is owed
	reported bool // "controller setup failed" was written since the last setup that succeeded
}

func newSessionKeeper(...) *sessionKeeper

// keep holds the session until stop is done, and ends with a teardown of a held session. It closes done.
func (keeper *sessionKeeper) keep(stop context.Context)

// readings asks for a reading and gives its wanted probes, or none when no session is held or the reading failed.
func (keeper *sessionKeeper) readings(stop context.Context, roundStart time.Time, wanted func(name string) bool) []sensors.Reading
```

`keep` makes the first setup at once, then loops: it arms `after(next - now)`, at least zero, and selects on stop, the
timer and a request (R11, R14).

* On stop it ends a held session with `end` and returns (R12, R13)
* On the timer it makes a reading when a session is held, and a setup otherwise
* On a request it makes a reading when a session is held and sends its probes, and sends nil otherwise. A request taken
  is always answered. A request that arrives during a setup is taken after it, so a round waits for the setup (R16)

A reading takes `now()` as its start and sets `next` to one second after it (R11). When it fails, the session is gone:
unless stop is done it writes `controller lost: <error>` and sets `lost`, and it sets `next` to `now()` plus `retry` and
doubles `retry` up to five minutes (R14, R15). A setup that fails does the same with `controller setup failed:
<error>`, written only when the error is not `errNoController` and `reported` is not set, which it then sets. A setup
that succeeds holds the session, sets `next` one second after its end, `retry` back to ten seconds, clears `reported`,
and writes `controller back` and clears `lost` when `lost` is set (R14, R15). `retry` starts at ten seconds, so the
waits of a run of failures are 10, 20, 40, 80, 160, 300 and then 300 seconds. When stop is done no line is written for a
setup or a reading that it ended.

`readings` returns nil at once when no probe of the controller is mapped, so that such a round asks for nothing while
the keepalive goes on (R16). Otherwise it sends a reply channel of capacity one on `requests`, or gives nil when
`done` is closed or stop is done first, then waits for the reply and gives the readings that `wanted` names. The
readings are of a reading that starts after the request, so after the round started (R16). Every exchange is made by
the goroutine of `keep`, so two never run at the same time (R11).

`readings` has the signature of `periodicSource.readings`, and the feeder's field `controller` becomes an interface
that both satisfy, `controllerReader`, declared in `run.go`. The feeder does not change otherwise: the rounds whose
reopening fails return before `read`, and the round of a reopening that succeeds goes on with the round, as R22 of unit
0004 says. This is how the plan of unit 0008 read the same words of its R8, and R16 is read so too: the rounds in which
`run` calls `Reopen` instead of the round ask for nothing, and the keeper holds the session through them.

`probesMapped` comes from `sensors.IsCommanderTemperature(name string) bool`, a new function of `internal/sensors`,
which tells whether a metric name is a probe of the controller. The prefix and the suffix of the name become two
constants that the table of metrics, `CommanderTemperature` and the new function share.

### The wiring

* `sourceSet` gains `startSession func(stop context.Context, colour config.Colour) (heldSession, error)`, with a doc
  comment that says what it gives. `sources_linux.go` sets it to a function that makes a `commandercore.Lighting` with
  the roots, `commandercore.ReplyTimeLimit` and the colour, calls `Start`, gives `errNoController` for
  `commandercore.ErrNoController`, and wraps the session in a type `commanderSession` whose `read` and `end` call
  `Read` and `End`. `sources_other.go` does not change: it gives an error before `run` goes on, so a build for Windows
  fails as R14 of unit 0004 says (R18)
* `environment` gains `after func(duration time.Duration) <-chan time.Time`, set to `time.After` in `Main`
* `runCommand`, when `settings.CommanderColour` is not nil, makes the keeper after `current.open()` has succeeded, with
  a start that binds the colour, starts `go keeper.keep(keeping)` on a context `keeping` derived from stop with a
  cancel, and makes it the feeder's controller (R5). After `feed` returns, whatever it returns, it cancels `keeping` and
  waits for `done` before it closes the transport and writes `stopped`, so the teardown comes before the end of `run` on
  a stop and on a feeding error (R12, R13). Without the key it keeps the `periodicSource` of today (R4). The line
  `running: ...` does not change (R15)
* `runCommand` wraps the standard error in a `lockedWriter` of `run.go`, a writer with a mutex, which the feeder and
  the keeper share, because the two write their lines from two goroutines

### The fake clock of the unit tests

The tests of `run` with the keeper need time that moves only when the test moves it, with the keeper's timer firing at
its moment. `fakeClock` in `fake_test.go` gains a mutex, which `now` and `sleep` take, a pending timer, and a channel
`armed` of capacity one:

* `after(duration)` records a timer due at `now + duration` with a channel of capacity one, which replaces the pending
  one, and signals `armed` without blocking. The keeper has one timer at a time, so one pending timer is enough
* `sleep(duration)` moves to the target in steps: while the pending timer is due at or before the target, it drains
  `armed`, sets the time to the timer's moment, takes the timer off, sends the moment on its channel, and waits for
  `armed` or for `released`; then it sets the time to the target. With no pending timer it moves at once, as today
* `released` is the `Done` of the harness's stop, set by `newHarness`, so that a keeper that ended does not hold a
  sleep
* `keeping`, set by the tests of the keeper, makes the first `sleep` wait for the first `armed` before anything else,
  because the first round of a run with no probe mapped may reach its wait before the keeper has armed its first timer

Without a keeper nothing calls `after`, so every existing test moves the clock as today (R27). The tests of the keeper
read the time with `now` and move it with `sleep` alone, never through `current`.

The stand-in for the session is `fakeLighting` in a new file `internal/cli/keeper_test.go`. It records the moment and
the colour of every setup and the moment of every reading on the harness's clock, takes the result of each setup and
reading from a queue of errors, nil once the queue is empty, gives the probes `commander.1.temp` of 37 and
`commander.2.temp` of 23, counts the teardowns, one for every `end` and one for every reading that fails, and runs an
optional `onRead(count int)` from the goroutine of the keeper. The harness gains `startSession`, which its `newSources`
puts into the `sourceSet`.

## Files

- `internal/config/config.go`, `internal/config/config_test.go`: the key, its checks, the texts of R3 (R1 to R3, R23)
- `internal/fakecommander/fakecommander.go`, `internal/fakecommander/fakecommander_test.go`: R22
- `internal/commandercore/protocol.go`, `internal/commandercore/protocol_test.go`: the commands, `ledCount`,
  `colourCommand`
- `internal/commandercore/source.go`: `exchange` with an error
- `internal/commandercore/session.go`, `internal/commandercore/session_test.go`: new, the session and R24
- `internal/sensors/metrics.go`, `internal/sensors/metrics_test.go`: `IsCommanderTemperature`
- `internal/cli/keeper.go`: new, the keeper, `heldSession`, `errNoController`
- `internal/cli/keeper_test.go`: new, `fakeLighting` and the tests of R25
- `internal/cli/run.go`: `controllerReader`, `lockedWriter`, the keeper in `runCommand`
- `internal/cli/sources.go`, `internal/cli/sources_linux.go`: `startSession`, `commanderSession`
- `internal/cli/main.go`: `after`
- `internal/cli/fake_test.go`: the fake clock with its timer
- `internal/cli/commands_test.go`: the harness gives `after` and `startSession`
- `e2e/lighting_test.go`: new, the tests of R26
- `udev/70-sensordeck.rules`: the comment (R19)
- `README.md`: R20
- `docs/commander-core-xt.md`: R21

## Tasks

### The config

- [ ] In `internal/config/config_test.go`, change the four cases of `TestParseErrors` that hold the texts of R3 of unit
  0008 to the texts of R3, and add (R1, R2, R3, R23):
    - a test that `commander_color: "#00c8ff"` gives a `CommanderColour` of 0, 200, 255 and that `"#00C8FF"` gives the
      same, both with `cpu.temp: 1` as the only tag, which also shows the key valid with no metric of the controller
      mapped
    - a test that a file without the key gives a nil `CommanderColour`
    - cases of `TestParseErrors` with valid tags: `commander_color: #00c8ff` unquoted, `commander_color: ""` and
      `commander_color: null`, each with the first text of R2; `"#00c8f"`, `"00c8ff"`, `"#00c8fg"` and `{a: 1}`, each
      with the second text, the text as written and the empty text for the mapping; and `commander_color: "#00c8ff"`
      with `commander_interval: 1m`, with the third text

  `TestParseTheExample` and the other tests stay as they are. The new tests fail to compile until the next task
- [ ] In `internal/config/config.go`, add `Colour`, the field `CommanderColour`, the case `commander_color` of the loop
  of `Parse`, `parseCommanderColour` called after `parseCommanderInterval`, the check of both keys after it, and the
  texts of R3, as "The config" says, until `go test ./internal/config` passes (R1, R2, R3)
- [ ] Run the five commands of the suite and commit: `feature-0009 Add commander_color to the config file`

### The fake controller

- [ ] In `internal/fakecommander/fakecommander_test.go`, add tests (R22):
    - in software mode, the open, read and close of `0x20`, `0x1A`, `0x17` and `0x21` on channel 1 give the answers of
      "The fake controller", the default ports being those of the host, and `SetPorts` with three ports of mode 2 and
      8 LEDs changes the answer of `0x20`
    - an open on a channel that has an endpoint open answers status `03` and leaves it open, for another endpoint and
      for the same one, and after a close of the channel the open answers `00`; `SetOpenEndpoint(0, 0x21)` makes the
      open of `0x22` on channel 0 answer `03`, and `OpenEndpoint` tells what is open
    - before any mode command and after `08 01 03 00 01`, an open, a read, a close and a colour write each answer status
      `03`, and the mode commands answer `00`
    - `Lighting` gives `StoredLighting` before any command, `Dark` after `08 01 03 00 02`, `SentColour` with `FF 00 00`
      after a colour write with `0x22` open on channel 0, and `StoredLighting` after `08 01 03 00 01`; a colour write
      with nothing open on channel 0 answers `03` and is not kept; `ColourWrites` holds the 96 data bytes of the one
      kept
    - with `SetSoftwareModeTimeout(100 * time.Millisecond)`, entering software mode, opening `0x22` and a colour write,
      then waiting 300 milliseconds: the mode is hardware, channel 0 is closed and `Lighting` gives `StoredLighting`,
      and a new `08 01 03 00 02` gives `Dark`; commands every 50 milliseconds for 300 milliseconds keep it in software
      mode; and with no timeout set the mode stays software after 300 milliseconds
    - `FailCommand` with `NoAnswer`, `WrongEcho` and `BadStatus` on `08 08 01 21` changes the answer of that command
      only, and `OnCommand` on `08 0D 00 22` runs its action after the write and before the answer

  Every existing test of the fake stays as it is. The new tests fail to compile until the next task
- [ ] In `internal/fakecommander/fakecommander.go`, add the channels, the new answers, `Port`, `Lighting` and the
  methods of "The fake controller", keep `Fail` and `OnStep` for the steps of unit 0005 on channel 0, and update the
  doc comments of the package, of `Controller` and of `reply`, until `go test ./internal/fakecommander
  ./internal/commandercore ./e2e` passes, which shows that the tests of units 0005, 0006 and 0008 keep what they expect
  (R22, R27)
- [ ] Run the five commands of the suite and commit: `feature-0009 Teach the fake controller channels, LEDs and modes`

### The session of the controller

- [ ] In `internal/commandercore/source.go`, make `exchange` give an `exchangeError` in place of false, as "The session
  of the controller" says, and make `runSteps` and `runSequence` treat an error as they treated false. Behaviour does
  not change, and every test of the package passes as it is. Run the five commands and commit: `feature-0009 Give the
  exchanges of the controller an error that says what failed`
- [ ] In `internal/commandercore/protocol_test.go`, add tests of `openCommand`, `readCommand` and `closeCommand` for
  `0x20` on channel 1 and `0x22` on channel 0; of `colourCommand` for 16 LEDs and `#ff0000`, whose bytes start with
  `08 06 00 32 00 00 00 12 00` and hold 16 times `FF 00 00` and zeros after them, and for 24 LEDs, with the length
  `4A 00`; and of `ledCount` for the answer of the host, 16, for entries of mode 3 with LEDs, which do not count, and
  for each text of its three refusals, including counts of 0 and 30 and a count of 23 entries (R7, R8)
- [ ] In `internal/commandercore/protocol.go`, add the constants, the commands, `ledCount` and `colourCommand`, and
  write steps 1 and 8 of `steps` with `enterSoftware` and `returnHardware`, until the tests pass (R6, R7, R8)
- [ ] Add `internal/commandercore/session_test.go`, with the fake controller and a time limit of 50 milliseconds, and
  these tests (R24, and R5 to R10, R12, R13):
    - `Start`, `Read` and `End` for the host and `#ff0000` give exactly the writes of R6, R9 and R12 in order, 7, 12 and
      2, compared byte by byte with `encode` of the commands, the colour write included, and leave the fake in hardware
      mode with channel 0 closed
    - with three ports of mode 2 and 8 LEDs, the colour write holds 24 triples
    - with LED counts of 0 and 30, `Start` fails with the text of `ledCount`, the writes being the enter, the close,
      open and read of `0x20` and the teardown with the close
    - `BadStatus` on `08 01 03 00 02`, on `08 08 01 20`, on `08 0D 00 22` and on `08 06 00` each makes `Start` fail with
      the name of that step, and the writes are those of the setup up to that step and the teardown with the close
    - after a `Start` that succeeds, `BadStatus` on `08 08 01 21` makes `Read` fail with the name of that read, the
      writes of the reading ending with that read and followed by the teardown with the close
    - `NoAnswer` on `08 0D 00 22` makes `Start` fail, and on `08 08 01 21` makes `Read` fail, each followed by the
      teardown without the close, `08 01 03 00 01` alone
    - `SetOpenEndpoint(0, 0x21)` makes `Start` fail with `open of endpoint 0x22 on channel 0: status 03`, the teardown
      closes channel 0, and the next `Start` succeeds
    - a stop by `OnCommand` on `08 0D 00 22` ends `Start` with no colour write and the teardown with the close; a stop
      by `OnCommand` on `08 08 01 17` ends `Read` with no further exchange of the reading and the teardown with the
      close; a stop done before `Start` writes nothing
    - with no controller `Start` gives `ErrNoController` and nothing is written
    - nothing is written to the node of interface 1 over a `Start`, a `Read` and an `End`
    - `Read` gives `commander.1.temp` of 37.3 and `commander.2.temp` of 22.7 for the default probes, the answer of unit
      0005

  The tests fail to compile until the next task
- [ ] Add `internal/commandercore/session.go`, with `//go:build linux`, holding `ErrNoController`, `errStopped`,
  `Lighting`, `Start`, `Session`, `Read`, `End`, `stepName` and `teardown`, as "The session of the controller" says,
  until the tests of the package pass (R5 to R10, R12, R13)
- [ ] Run the five commands of the suite and commit: `feature-0009 Hold a session of the controller with its LEDs lit`

### The keeper in run

- [ ] In `internal/sensors/metrics_test.go`, add a test that `IsCommanderTemperature` is true for `commander.1.temp`
  and `commander.12.temp` and false for `commander.0.temp`, `commander.temp`, `cpu.temp` and `ram.1.temp`. Add the
  function and the two constants to `internal/sensors/metrics.go`, used by the table and by `CommanderTemperature`,
  until the tests of the package pass (R16)
- [ ] In `internal/cli/fake_test.go`, give `fakeClock` the mutex, `after`, `armed`, `released` and `keeping` of "The
  fake clock of the unit tests". In `internal/cli/commands_test.go`, give the environment `after: harness.clock.after`,
  set `released` in `newHarness`, add the field `startSession` and put it into the `sourceSet`. Add `after` to
  `environment` in `internal/cli/main.go`, set to `time.After` in `Main`, and `startSession` to `sourceSet` in
  `internal/cli/sources.go`, so that the package compiles. Every existing test of `internal/cli` passes as it is (R27)
- [ ] Add `internal/cli/keeper_test.go` with `fakeLighting`, a config `colourTags` of `commander_color: "#ff0000"` and
  `tags` of `cpu.temp: 1`, `commander.1.temp: 4` and `commander.2.temp: 5`, with `interval` before it where a test needs
  it, and a helper that gives a harness with `twoMetrics`, the stand-in, `keeping` set and a stop after a given number of
  waits. Add these tests (R25):
    - with `interval: 10s` and a stop after 6 waits: the moments of the readings, preceded by the end of the first
      setup, are never more than 1 second apart, and the last is at least 59 seconds after the start (R11)
    - the same with the screen lost at the first value report and every reopening failing (R11)
    - with `interval: 100ms` and a stop after 30 waits: a reading at the start of every round, each 100 milliseconds
      after the one before, and the value reports hold (4, 37) and (5, 23) (R16)
    - with `tags` of `cpu.temp: 1` alone and `interval: 10s`, a stop after 3 waits: the readings are exactly 1 second
      apart from the end of the setup on, none of them asked by a round (R11, R16)
    - a reading that fails at the third reading: the value reports after it hold (1, 42) alone, and `controller lost:
      fake failure` is on standard error (R15, R16)
    - setups that fail eight times: their moments are the start and then 10, 20, 40, 80, 160, 300 and 300 seconds after
      the one before; with three that fail, one that succeeds and a reading that then fails, the next setup is 10
      seconds after that failure (R14)
    - a setup after a lost session gets the colour 255, 0, 0 again, so every setup of the stand-in records that colour
      (R14)
    - the lines: three failed setups, then one that succeeds, then a reading that fails, then a setup that succeeds give
      exactly `controller setup failed: fake failure`, `controller lost: fake failure` and `controller back`, each once,
      in that order; setups that give `errNoController` write no line (R15)
    - a stop after 3 waits with a session held ends with exactly one teardown, `stopped` on standard output and status
      0; a clock in the year 2256, which fails the feeding at its first clock report, ends with exactly one teardown and
      status 1 (R12, R13)
    - without `commander_color`, the stand-in gets no setup and the controller of the `sourceSet` is read as unit 0008
      says (R4)

  The tests fail until the next task
- [ ] Add `internal/cli/keeper.go` with `errNoController`, `heldSession`, the constants and `sessionKeeper`, as "The
  keeper in run" says. In `internal/cli/run.go`, add `controllerReader` and `lockedWriter`, make the feeder's
  `controller` a `controllerReader`, and make `runCommand` start and end the keeper as "The wiring" says. Update the doc
  comment of `feeder` to say that the controller is read through the keeper when the config gives a colour. In
  `internal/cli/sources_linux.go`, set `startSession` with `commanderSession`. Every test of `internal/cli` passes, and
  `internal/cli/commands.go`, where `sensors` is, does not change (R4, R5, R11 to R17)
- [ ] Run the five commands of the suite, which include the test of the layout that keeps `internal/commandercore`
  out of a build for Windows and every file of `internal/cli` that names it in a Linux file, and commit:
  `feature-0009 Hold the session of the controller in run` (R18)

### The end-to-end tests

- [ ] Add `e2e/lighting_test.go`, with `//go:build linux`, a helper that starts `run` with the screen, the tree of
  `runningTree`, a fake controller `Add(9, 10)` and the config of R26 with a given interval, and a helper that encodes a
  command as the driver writes it. Add (R26):
    - a value report holds (4, 37); after `SetProbes` with 40.0 °C for the first probe, a value report holding (4, 40)
      arrives within 1 second
    - after SIGTERM: status 0 within 3 seconds, `Mode` hardware, `OpenEndpoint(0)` zero, and over the whole run one
      write of `08 01 03 00 02` in `Writes` and one entry in `ColourWrites`, with 16 triples `FF 00 00`
    - with `interval: 5s` and `SetSoftwareModeTimeout(1500 * time.Millisecond)`: once `ColourWrites` holds one entry,
      4 seconds later the mode is software, `Lighting` gives `SentColour` with `FF 00 00`, and the writes hold at least
      three of `08 08 01 21`
    - `RemoveNode` while `run` runs: value reports with the pair of tag 1 keep arriving for 2 seconds after it, and
      after SIGTERM standard error holds `controller lost: `
    - a config with `commander_color: #ff0000` unquoted and no screen: status 1 and the first text of R2 on standard
      error after `sensordeck: config file <path>: `
- [ ] Run the five commands of the suite and commit: `feature-0009 Test the colour of the controller end to end`

### The documents

- [ ] In `udev/70-sensordeck.rules`, change the comment of the second rule to say that sensordeck reads the
  temperature probes of the controller and lights its LEDs, and leave both rules as they are (R19)
- [ ] In `README.md`, add the section `## The Corsair iCUE Commander Core XT` after "Showing sensor values on the
  screen" and before "Credits", with every point of R20 in plain words, the two example configs among them, the one of
  `commander_color` with `cpu.temp`, `commander.1.temp` and `commander.2.temp`, and the line of the second rule of
  `udev/70-sensordeck.rules`. In "Showing sensor values on the screen", add to the paragraph on `commander_interval` that
  the period applies when `commander_color` is not given, with a link to the new section. Remove nothing (R17, R20)
- [ ] In `docs/commander-core-xt.md`, in the project's own words, with everything it says today kept (R21):
    - the title becomes `# The temperature probes and the LEDs of the Corsair iCUE Commander Core XT`, and the opening
      paragraph names the LEDs
    - "How it was learned" names the four sources with what each gave: the trace of liquidctl of unit 0005, the trace
      of sensordeck of unit 0006, the trace of OpenLinkHub and the device tests of units 0008 and 0009, and says that
      the code of none of these programs was read
    - a section "Channels and endpoints" after "The commands": the channel byte, the open, read and close of an
      endpoint on a channel, the eight commands as those of channel 0, and the status `03` of an open on a channel with
      an endpoint open
    - in "The layout of an answer", sections for the endpoints `0x20` and `0x1A`, the rule of R7 with its assumptions
      that mode 2 means connected and that the triples follow the order of the entries, and a section for the colour
      write and its layout
    - "Software mode and the LEDs" says what the device tests showed of hardware mode and software mode, and that `run`
      reads the controller at most once per period only without `commander_color`
    - a section "The keepalive" with what liquidctl's description says, the timeout unknown, the reads once a second
      of the trace, and the one reading a second of sensordeck
    - a section "A session of run" with the commands of the setup, of a reading and of the teardown in order, and the
      commands of the trace that sensordeck does not send, which the device test did without
    - in "Unknown", the bullet on the fans says that the LEDs are now known and the fans that the controller runs are
      not, the bullet on the full output report names the colour write of more than 29 LEDs, and the bullets of R21
      are added: the software-mode timeout, the setup commands not sent, the answer of `0x1A`, modes of `0x20` other
      than 2 and 3, the order of the triples, and whether the endpoint byte of a read or a close matters
- [ ] Commit: `feature-0009 Document the colour of the controller`

### Before handing over

- [ ] Run the five commands of the suite and `npm run check-changes`, and check every requirement R1 to R27 against the
  tree, ticking this task only when all pass (R1 to R27)
