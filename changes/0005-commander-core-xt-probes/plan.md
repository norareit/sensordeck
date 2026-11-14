# The temperature probes of the Corsair Commander Core XT

## Approach

The unit adds three packages, all limited to Linux, extends four, and changes two documents and the udev rule file.
Two refactorings come first, each in a commit of its own that changes no behavior, so that the driver and the fake
controller do not repeat what the screen already knows:

* `internal/hidraw` gets `List`, which reads every `hidraw<N>` entry under the sysfs root into a value with its
  number, its node and the `HID_ID`, `HID_NAME`, `HID_PHYS` and `HID_UNIQ` of its `uevent` file, ordered by the
  number. `Discover` becomes a filter over `List`. It also gets `OpenNode`, the opening of a node for reading and
  writing with `O_NOCTTY` and the check that it takes a read deadline, returning the error of the system unchanged;
  the existing `openNode` wraps it with `openError`. The driver of the controller uses both, so the reading of sysfs
  and the opening of a node live in one place (R3, R4)
* `internal/fakepty`, new, takes the pseudo terminal helpers out of `internal/fakescreen` (`makeTerminal`, `makeRaw`,
  `control`), so that the fake controller is made of the same terminals (R17)

Then the unit builds:

* `internal/sensors`, extended: the row `commander.<n>.temp` and `CommanderTemperature(number int) string` (R1, R19)
* `internal/fakecommander`, new: the fake controller of R17
* `internal/commandercore`, new: the driver, split by concern:
    * `protocol.go`, pure: the eight commands, the encoding of a write, the check of an answer and the parsing of the
      temperature answer (R5, R6, R9)
    * `discover.go`: the node of interface 0 through `hidraw.List` (R3)
    * `source.go`: the `Source` that runs the sequence on an opened node, with the stop and the restoring of hardware
      mode (R2, R4, R5 to R8, R10)
* `internal/cli`, extended: the sources take the dev root, the controller is the last source, and `sensors` catches
  SIGINT and SIGTERM while it reads (R12, R13, R22)
* `layout_test.go`, extended: the three new packages are limited to Linux and only Linux files of the commands import
  the driver (R11)
* `e2e`, extended: the tests of R20, R21 and R23
* `udev/70-sensordeck.rules`, `docs/commander-core-xt.md` and `README.md` (R14 to R16)

### The listing of hidraw entries

```go
// Entry is one hidraw<N> entry of <sysfs root>/class/hidraw whose device/uevent could be read.
type Entry struct {
	Number          int
	Node            string // <dev root>/hidraw<N>
	Bus             uint64 // from HID_ID, like Vendor and Product; ValidID is false when HID_ID is missing or malformed
	Vendor, Product uint64
	ValidID         bool
	Name, Phys      string // HID_NAME, HID_PHYS
	Serial          string // HID_UNIQ
}

// List gives the entries ordered by Number. A missing directory gives none and no error; any other failure to read
// the directory is the error of today's Discover. An entry whose uevent cannot be read is left out.
func List(sysfsRoot, devRoot string) ([]Entry, error)
```

`Discover` keeps its signature and its result: it takes the entries of `List` with `ValidID`, bus 3 and the vendor
and product of the screen, and makes a `Screen` of each with the `Vendor` and `Product` constants of `smartmonitor` as
today. The package comment says that the package is the transport of the screen and the listing of hidraw entries
that the controller shares. `isSupportedID` becomes the parsing of `HID_ID` into three numbers, read as hexadecimal
as today.

```go
// OpenNode opens the node for reading and writing, without making it the controlling terminal, and checks that it
// takes a time limit for reading. Its error is the error of the system, or the failure of that check.
func OpenNode(path string) (*os.File, error)
```

The existing tests of `Discover` and of the opening stay unchanged and must pass unchanged; that is the proof that
the refactoring kept the behavior.

### The metric

`metricKinds` gets `{prefix: "commander.", suffix: ".temp", numbered: true, unit: "°C"}` as its last row, after
`nvme.`. `CommanderTemperature(number)` gives `"commander.<number>.temp"`. The config takes the names through
`sensors.IsName` and needs no change (R1).

### The fake controller

`internal/fakecommander`, every file `//go:build linux`:

```go
// Fake makes controllers under the given roots, which the fake screen and the fake sensors may share.
func New(test testing.TB, sysfsRoot, devRoot string) *Fake

// Add makes a controller whose node of interface 0 is hidraw<interface0> and whose node of interface 1 is
// hidraw<interface1>, each with a sysfs entry that holds the HID_ID 0003:00001B1C:00000C2A, the HID_NAME of "What was
// observed", and a HID_PHYS ending in /input0 and /input1. Only the node of interface 0 answers.
func (fake *Fake) Add(interface0, interface1 int) *Controller

// AddInterfaceOne makes only the entry and the node of interface 1, as hidraw<number>.
func (fake *Fake) AddInterfaceOne(number int) *Controller

type Probe struct {
	Status byte
	Tenths int16 // the temperature in tenths of a degree, written low byte first
}

type Failure int

const (
	NoAnswer      Failure = iota + 1
	WrongEcho             // byte 1 of the answer is not the second written byte of the command
	BadStatus             // byte 2 of the answer is 0x01
	WrongDataType         // bytes 3 and 4 are 06 00; meaningful for step 6
	CountTooLarge         // byte 5 is 31, so that 6 + 3·31 > 96; meaningful for step 6
)

type Mode int

const (
	NoMode Mode = iota // no mode command has arrived
	SoftwareMode
	HardwareMode
)

func (controller *Controller) SetProbes(probes ...Probe)        // default: {0, 373} and {0, 227}
func (controller *Controller) Fail(step int, failure Failure)   // the answer of that step, 1 to 8
func (controller *Controller) OnStep(step int, action func())  // runs action once that step arrived, before its answer
func (controller *Controller) RemoveNode()                      // the sysfs entries stay, the node of interface 0 goes
func (controller *Controller) Writes() [][]byte                 // the writes on interface 0, 97 bytes each, after a quiet period
func (controller *Controller) WritesOnInterfaceOne() [][]byte   // the same for interface 1
func (controller *Controller) WaitWrites(count int, limit time.Duration) ([][]byte, bool)
func (controller *Controller) Mode() Mode                       // the mode the last mode command set
```

A node is a pseudo terminal from `internal/fakepty`, its slave held open and linked as `<dev root>/hidraw<N>`, as the
fake screen does. A reader goroutine on the master splits what arrives into writes of 97 bytes and answers each
complete write of interface 0 with one write of 96 bytes on the master. It knows which step a write is from its bytes
and the endpoint it opened last: `08 01 03 00 02` is step 1 and sets software mode, `08 0D 00 17` step 2, `08 08 00`
with `0x17` open step 3, `08 05 01` with `0x17` open step 4, `08 0D 00 21` step 5, `08 08 00` with `0x21` open step 6,
`08 05 01` with `0x21` open step 7, and `08 01 03 00 01` step 8, which sets hardware mode. A mode command sets the
mode even when its answer is made to fail. Any other write gets no answer.

The answers are those of "What was observed": `00 <second written byte> 00` followed by zeros for steps 1, 2, 4, 5, 7
and 8; `00 08 00 06 00 06` and twelve zero bytes for step 3; and `00 08 00 10 00 <count>` followed by the three bytes
of each probe for step 6. With the default probes the answer of step 6 is byte for byte the one of "What was
observed". A failure set for a step changes its answer as the constant says, and `NoAnswer` sends nothing.

The writes are kept with the time they arrived, as in the fake screen, and `Writes` waits until nothing has arrived
for 200 milliseconds before it gives them, so that it serves a reading in the same process and an executable that has
ended. Cleanup closes every terminal.

### The driver

`internal/commandercore`, every file `//go:build linux`.

`protocol.go`:

```go
const (
	answerSize = 96
	writeSize  = 1 + answerSize
	// maxProbes is the largest count whose entries fit into an answer: 6 + 3·30 = 96.
	maxProbes = (answerSize - 6) / 3
)

// steps are the eight commands of the sequence, after the report ID byte, in their order.
var steps = [8][]byte{ {0x08, 0x01, 0x03, 0x00, 0x02}, {0x08, 0x0D, 0x00, 0x17}, {0x08, 0x08, 0x00},
	{0x08, 0x05, 0x01}, {0x08, 0x0D, 0x00, 0x21}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01},
	{0x08, 0x01, 0x03, 0x00, 0x01} }

func encode(command []byte) []byte                 // 97 bytes: 0x00, the command, zeros
func valid(command, answer []byte) bool            // R6 for every answer: 96 bytes, echo in byte 1, byte 2 is 0x00
func temperatures(answer []byte) ([]probe, bool)   // R6 for step 6, then R9
type probe struct {
	available bool
	celsius   float64 // int16(answer[low] | answer[high]<<8) / 10
}
```

`temperatures` gives false when the data type is not `10 00` or 6 + 3·*c* > 96, and otherwise one `probe` for each
*i* from 0 to *c* − 1, unavailable when its status byte is not `0x00`.

`discover.go`:

```go
// findNode gives the node of the lowest numbered entry with a valid HID_ID of bus 3, vendor 0x1B1C and product
// 0x0C2A, and a HID_PHYS ending in /input0, or false.
func findNode(sysfsRoot, devRoot string) (string, bool)
```

`source.go`:

```go
// ReplyTimeLimit is how long a reading waits for each answer of the controller (D11).
const ReplyTimeLimit = time.Second

// Source reads the probes of the controller, opening its node anew for every reading (D9).
type Source struct {
	SysfsRoot, DevRoot string
	TimeLimit          time.Duration
}

func (source Source) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading
```

`Read` orchestrates and performs nothing itself:

1. When `wanted` is false for `CommanderTemperature(n)` for every *n* from 1 to `maxProbes`, it returns nil and
   opens nothing (R4). A name beyond `maxProbes` can never be given by a valid answer, so no wanted name is missed
2. When stop is done, it returns nil and writes nothing (R8)
3. It finds the node, and returns nil when there is none (R2, R3)
4. It opens the node with `hidraw.OpenNode`, returns nil when that fails (R2, R4), and closes it before it returns
5. It runs the sequence and gives the readings of the wanted probes that are available, in the order of their
   numbers (R9)

The sequence, a function of its own over the opened file and the time limit, gives the probes of a valid step 6 or
none:

* It writes step 1 and reads its answer. From then on step 8 is owed, whatever happens (R7)
* Before each write of steps 2 to 7 it checks stop, and ends the steps when stop is done (R8)
* A write that fails or writes fewer than 97 bytes, a read that fails or passes the time limit, and an answer that is
  not `valid`, end the steps (R6). The answer of step 6 must also pass `temperatures`; its probes are kept. The answer
  of step 3 is checked with `valid` and its content is not used
* After the steps, ended or not, it writes step 8 and reads one answer within the time limit, and a failure there
  changes nothing of the probes kept (R7)

A read sets the deadline of the file to now plus the time limit and reads once into a buffer of 96 bytes. A count
other than 96 is an answer that is not valid. Stop does not end a read in progress: a read waits at most one time
limit, so a reading that is stopped ends after at most two, the read under way and step 8, which is the bound of D11.
The stop is checked before the writes only, which is all that R8 asks.

### The commands

`environment.newSources` becomes `func(sysfsRoot, devRoot, procRoot string) ([]sensors.Source, error)`, and `sensors`
and `run` pass the dev root. `sources_linux.go` gives the sources of `linuxsensors.Sources` followed by
`commandercore.Source{SysfsRoot: sysfsRoot, DevRoot: devRoot, TimeLimit: commandercore.ReplyTimeLimit}`, so the
controller comes last as its row does (R1, R13). `sources_other.go` takes the new parameter and is otherwise
unchanged. `--device` is not passed to the sources (R3, R12).

`sensors` calls `notifyStop` after `newSources` and right before `sensors.ReadAll`, passes its context to `ReadAll`,
reads `stop.Err()` once `ReadAll` has returned and cancels then, so that a signal before or after the reading keeps
the default handling. When the stop was done it writes nothing to standard output and returns the error
`interrupted`, which `run` writes as `sensordeck: interrupted` with the status 1 (R22). The comment of `notifyStop` in
`main.go` says that `run` and the reading of `sensors` call it. `run` already passes its stop to the sources (R8,
R13).

### The tests

* `internal/sensors/metrics_test.go`: `commander.1.temp` and `commander.10.temp` accepted, `commander.0.temp` and
  `commander.01.temp` refused, the new row in the test of units, `CommanderTemperature(2)` (R19)
* `internal/commandercore/protocol_test.go`, plain values: the 97 bytes of each step; `valid` for a short answer, a
  wrong echo and byte 2 not `0x00`; `temperatures` for the answer of "What was observed", a wrong data type, the
  count 31 and the count 30, a probe with status `01`, and `0x FF9C` read as −10.0
* `internal/commandercore/discover_test.go`, a sysfs tree written by the test: only interface 1, other devices,
  a malformed `HID_ID`, two qualifying entries
* `internal/commandercore/source_test.go`, against `internal/fakecommander` with a time limit of 50 milliseconds, the
  cases of R18 one by one (listed in the tasks)
* `internal/cli/sensors_test.go`: the interrupted reading of R23, and the dev root passed on
* `e2e/commander_test.go`: R20, R21 and R23

## Files

- `internal/hidraw/discover.go`: `Entry`, `List`, `Discover` as a filter over it, the parsing of `HID_ID`
- `internal/hidraw/discover_test.go`: tests of `List`
- `internal/hidraw/transport.go`: `OpenNode`, used by `openNode`
- `internal/hidraw/transport_test.go`: a test that `OpenNode` gives the error of the system unchanged
- `internal/fakepty/fakepty.go`, `internal/fakepty/fakepty_test.go`: new
- `internal/fakescreen/fakescreen.go`: uses `internal/fakepty`
- `internal/sensors/metrics.go`, `internal/sensors/metrics_test.go`: the new row
- `internal/fakecommander/fakecommander.go`, `internal/fakecommander/fakecommander_test.go`: new
- `internal/commandercore/protocol.go`, `discover.go`, `source.go` and a `_test.go` file for each: new
- `internal/cli/main.go`: the signature of `newSources`, the comment of `notifyStop`
- `internal/cli/commands.go`: `sensors` with the stop and the dev root
- `internal/cli/run.go`: the dev root
- `internal/cli/sources_linux.go`, `internal/cli/sources_other.go`: the dev root and the controller
- `internal/cli/commands_test.go`: the harness checks the dev root
- `internal/cli/sensors_test.go`: `fakeSource` records the stops it got; the tests of R23
- `layout_test.go`: the new packages and the new import
- `e2e/commander_test.go`: new
- `udev/70-sensordeck.rules`: the rule of the controller
- `docs/commander-core-xt.md`: new
- `README.md`: the metric, the access and the status line

## Tasks

### The refactorings

- [x] In `internal/hidraw`, add `Entry` and `List`, and add unit tests of `List` in `discover_test.go`: the fields of
  an entry with every key, an entry with a malformed `HID_ID` kept with `ValidID` false, an entry without `uevent`
  left out, the order by number, a missing directory. Make `Discover` filter the entries of `List`. The existing
  tests of `Discover` pass unchanged (R3)
- [x] In `internal/hidraw/transport.go`, extract `OpenNode` from `openNode`, which keeps its errors. Add a test that
  `OpenNode` on a missing path gives an error for which `errors.Is(err, fs.ErrNotExist)` holds and whose text is the
  one of the system. The existing tests of the opening pass unchanged (R4)
- [ ] Add `internal/fakepty` with `Open(test testing.TB) (master, slave *os.File, slavePath string)`, which makes a
  pseudo terminal, opens its slave with `O_NOCTTY` and sets it raw, moving `makeTerminal`, `makeRaw` and `control`
  from `internal/fakescreen` with their comments. Add a test that bytes written to the master arrive unchanged on the
  slave and the other way, `0x0A` and `0x0D` among them. Make `internal/fakescreen` use it. The tests of
  `fakescreen`, `hidraw` and `e2e` pass unchanged (R17)

### The metric

- [ ] Add to `metrics_test.go`: `commander.1.temp` and `commander.10.temp` among the accepted names,
  `commander.0.temp` and `commander.01.temp` among the refused, `"commander.1.temp": "°C"` in the test of units, and
  `CommanderTemperature(2) == "commander.2.temp"` in the test of numbered names. Then add the row and the function to
  `metrics.go` (R1, R19)

### The fake controller

- [ ] Write `internal/fakecommander/fakecommander_test.go` first, writing to the node as a driver would: the eight
  commands get the eight answers of "What was observed" byte for byte, and the mode is software after step 1 and
  hardware after step 8; `SetProbes` changes the answer of step 6; each `Failure` changes the answer of the step it
  is set for as its comment says, and `NoAnswer` sends nothing within 100 milliseconds; `OnStep` runs its action after
  the write of its step and before the answer; a write on interface 1 is kept
  in `WritesOnInterfaceOne` and gets no answer; both sysfs entries hold the `HID_ID` and the `HID_PHYS` of their
  interface; `RemoveNode` leaves the entries and removes the node (R17)
- [ ] Write `internal/fakecommander/fakecommander.go` as "The fake controller" describes, until its tests pass (R17)

### The driver

- [ ] Write `internal/commandercore/protocol_test.go` with the cases of "The tests", then `protocol.go` (R5, R6, R9)
- [ ] Write `internal/commandercore/discover_test.go`: no `class/hidraw` directory, an entry of the screen, only an
  entry of interface 1, a malformed `HID_ID`, and entries 7 and 3 that both qualify, of which `hidraw3` is given; then
  `discover.go` (R3)
- [ ] Write `internal/commandercore/source_test.go` against the fake controller with a time limit of 50 milliseconds,
  with one test for each of: (R5, R9, R10, R18)
    - both probes wanted: the readings `commander.1.temp` 37.3 and `commander.2.temp` 22.7, and the eight writes byte
      for byte, once
    - nothing written to the node of interface 1 in that reading
    - probe 1 with status `01`: only `commander.2.temp`, with its number
    - probe 1 at −10.0 °C, `Tenths: -100`
    - only `commander.2.temp` wanted: only that reading, after a full sequence of eight writes
    - no controller, and `class/hidraw` missing: no readings
    - `RemoveNode`: no readings
    - only `AddInterfaceOne`: no readings and nothing written to it
    - controllers added as `Add(7, 8)` and `Add(3, 4)`: the one at 3 gets the eight writes, the one at 7 none
    - `NoAnswer`, `WrongEcho` and `BadStatus` at steps 1, 3 and 6, and `WrongDataType` and `CountTooLarge` at step 6:
      no readings, the writes are those of the steps up to the failed one followed by step 8, and the mode is hardware
    - `NoAnswer` at step 8: both readings, and eight writes
    - stop done before the reading: nothing written and no readings
    - stop done by `OnStep(3, cancel)`, so that it is done after step 3 arrived and before its valid answer is sent:
      no readings, the writes are steps 1, 2, 3 and 8, and the mode is hardware. The answer of step 3 is valid, so
      only the stop can have ended the steps
    - `wanted` false for every name: no readings, `Writes` empty, and `wanted` asked for `commander.<n>.temp` names
      only. An opening cannot be seen on a pseudo terminal, and in `Read` an opening is always followed by the write
      of step 1 unless stop is done, which this test does not do, so no write is the evidence that no node was
      opened
- [ ] Write `internal/commandercore/source.go` as "The driver" describes, until its tests pass (R2, R4 to R10)

### The commands

- [ ] Change `newSources` to take the dev root in `main.go`, `sources_linux.go`, `sources_other.go`, `commands.go`,
  `run.go` and the harness of `commands_test.go`, which fails when the dev root is not `/devroot`. Add the controller
  as the last source in `sources_linux.go` (R12, R13)
- [ ] Add to `sensors_test.go`: `fakeSource` keeps the stop of each call; a test whose source cancels the harness stop
  in `onRead`, and that expects status 1, no standard output, `sensordeck: interrupted\n` on standard error, and that
  the stop the source got is done; and a test that a reading without a signal gives the source a stop that is not
  done. Then change `sensors` in `commands.go` and the comment of `notifyStop` in `main.go` as "The commands" says
  (R22, R23)
- [ ] Add `internal/commandercore`, `internal/fakecommander` and `internal/fakepty` to the list of
  `TestLinuxCodeIsLimitedToLinux`, and `github.com/norareit/sensordeck/internal/commandercore` to the imports that
  `TestOnlyLinuxFilesOfTheCommandsNameTheLinuxCode` refuses in a file of `internal/cli` that is part of a Windows
  build (R11)

### The end-to-end tests

- [ ] Write `e2e/commander_test.go`, `//go:build linux`, with the fake controller made by `fakecommander.New` on the
  roots of the fake screen (R20, R21, R23):
    - `sensors` with `fullTree` and `Add(9, 10)`: the exact output of `TestSensorsListsTheReadingsOfTheHost` followed
      by `commander.1.temp\t37.3\t°C\n` and `commander.2.temp\t22.7\t°C\n`, an empty standard error, status 0, the
      eight commands once on interface 0, nothing on interface 1, and hardware mode
    - `run` with a screen as `hidraw3`, `Add(9, 10)` and a config of `interval: 100ms` that maps `commander.1.temp`
      to 4 and `commander.2.temp` to 5: a value report `02 02 04 00 25 05 00 17` followed by zeros arrives within
      `reportLimit`; after SIGTERM the status is 0 and the mode is hardware
    - `sensors` with an empty proc root, a `PATH` without `nvidia-smi`, and `Add(9, 10)` with `NoAnswer` at step 6:
      once `WaitWrites` has seen six writes, SIGINT; the status is 1, the standard output empty, the standard error
      `sensordeck: interrupted\n`, the process ended within 3 seconds of the signal, the seventh and last write is
      step 8, and the mode is hardware
- [ ] Check that the existing tests of `e2e/sensors_test.go` and `e2e/run_test.go` pass with their outputs unchanged
  (R20)

### Access and documents

- [ ] Add to `udev/70-sensordeck.rules`, after the rule of the screen, a comment that names the Corsair iCUE
  Commander Core XT and its USB ID `1b1c:0c2a`, and the line
  `SUBSYSTEM=="hidraw", ATTRS{idVendor}=="1b1c", ATTRS{idProduct}=="0c2a", MODE="0660", TAG+="uaccess"`. The first
  two lines and the rule of the screen stay as they are (R14)
- [ ] Write `docs/commander-core-xt.md` in the project's own words, with the sections: how it was learned (a
  black-box trace of the hidraw traffic of another tool on one machine, no code of that tool read); identity (USB ID,
  `HID_NAME`, the two interfaces and how `HID_PHYS` tells them apart, which one is used); reports (97 bytes written
  with the report ID, 96 read); the eight commands; the layout of an answer (bytes 0 to 2, the echo); the data of the
  endpoint `0x17` and of `0x21`; stale bytes; the times of the answers; "Unknown", which names at least the meaning
  of bytes 0 and 2 of an answer, the meaning of a probe status other than `00`, which probe sits at which port, and
  whether steps 2 to 4 are needed before the temperature steps; and "Assumptions", which names at least that byte 2
  is a status, that a probe status other than `00` means no reading, and that the temperature is signed (R15)
- [ ] In `README.md`: add the row `commander.<n>.temp` with `°C` and the source "the temperature probes of a Corsair
  iCUE Commander Core XT" after the `nvme.<n>.temp` row, and a sentence after the table that *n* counts the probes of
  the controller from 1; in "Linux: access to the screen", say that the rule file also holds a rule for the
  controller, which gives access to it, and that its metrics are left out when it cannot be opened; in the status
  line at the top, name the probes of the controller among what works on Linux. No other text of the README changes
  (R16)

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [ ] `grep -rn "liquidctl" internal e2e` finds nothing that is code or text of liquidctl, only the name where a
  comment needs it (D1)
- [ ] `npm run check-changes` passes

## Commit Points

- [ ] `feature-0005 Write the plan` with `plan.md` and the entry of `record.md`
- [ ] `feature-0005 List the hidraw entries and open a node in one place`, the first two tasks, with no change of
  behavior
- [ ] `feature-0005 Move the pseudo terminals of the fake screen into a package of their own`, with no change of
  behavior
- [ ] `feature-0005 Add the metric of the probes of the controller`
- [ ] `feature-0005 Add the fake controller`
- [ ] `feature-0005 Read the probes of the Commander Core XT`, the driver and its tests
- [ ] `feature-0005 Read the controller in sensors and run, and catch signals while sensors reads`, the commands and
  the layout test
- [ ] `feature-0005 Test the controller end to end`
- [ ] `feature-0005 Document the controller and give access to it`, the udev rule, the document and the README
