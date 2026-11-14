# Linux transport and a first command line

## Approach

The unit adds four packages and changes one, in the order in which each builds on the one before:

* `internal/smartmonitor`, changed: the USB ID constants, the error value of a passed time limit and the progress
  function of `Upload` (R5 to R7)
* `internal/fakescreen`, new: the fake screen that the tests of the transport and the end-to-end tests share (R29). It
  is a normal package and not a `_test.go` file, because two packages import it. It imports `testing` for its
  cleanup and its failures, and is imported by tests only
* `internal/hidraw`, new: the discovery and the transport on Linux (R8 to R14)
* `internal/cli`, new: the commands, the tracing and the reading of the environment (R15 to R24, R35)
* `e2e`, new: the end-to-end tests, in `_test.go` files only (R33, R34)

The root of the module gets `main.go`, package `main`, whose `main` is `os.Exit(cli.Main())` and nothing else (R1,
D5). Every Go file of `internal/fakescreen`, `internal/hidraw` and `e2e`, and the one file of `internal/cli` that names
`internal/hidraw`, starts with `//go:build linux` (R4). A Windows build then leaves those packages out of `./...`
entirely, which the two commands of R3 prove.

### The changes to the package of the protocol

A new file `identity.go` holds:

```go
const (
	VendorID  uint16 = 0x0483
	ProductID uint16 = 0x0065
)
```

`upload.go` gains, beside `Transport`:

```go
// ErrReadTimeout is the error of a Read whose time limit passed.
var ErrReadTimeout = errors.New("no report arrived within the time limit")
```

The comment of `Read` in `Transport` becomes: Read returns one report, or an error once the time limit has passed,
for which `errors.Is(err, ErrReadTimeout)` is true; for every other error it is false (R6).

The signature of `Upload` becomes:

```go
func Upload(transport Transport, theme []byte, pause func(time.Duration), progress func(stage Stage, block, blocks int)) error
```

`Upload` computes `blocks := dataBlocks(theme)` once at its start. Its list of steps becomes a list of pairs of a stage
and the function that runs it. For each step but the data blocks it calls `progress(stage, 0, len(blocks))` before
running it; `sendDataBlocks` takes the blocks and the progress function and calls `progress(StageData, number,
len(blocks))` before `sendItem` of each block. The retries inside `sendItem` and `reopen` stay where they are, so
the progress function is called once per stage or block whatever the retries (R7). The calls on the transport do not
change.

### The fake screen

`internal/fakescreen` makes a pseudo terminal pair for each screen with `golang.org/x/sys/unix`, which the trial for
this plan confirmed in the sandbox: a report sent to the master before the node is opened is kept and read later, a
read of at most 64 bytes on the node gives one report of 64, a read deadline on the node works, the bytes pass
unchanged in both directions once the slave is raw, a read after the master is closed gives `io.EOF`, and a regular
file refuses a read deadline with `os.ErrNoDeadline`.

```go
type Fake struct {
	SysfsRoot string // <temp>/sys
	DevRoot   string // <temp>/dev
	// unexported: the test, a mutex, the screens
}

func New(test testing.TB) *Fake // makes <temp>/sys/class/hidraw and <temp>/dev, and closes every pty in test.Cleanup
func (fake *Fake) Add(number int, name, serial string) *Screen

type Screen struct{ /* unexported */ }

func (screen *Screen) Node() string                                    // <DevRoot>/hidraw<number>, where it is now
func (screen *Screen) Send(report smartmonitor.Report)                 // writes the 64 bytes to the master now
func (screen *Screen) SendBytes(data []byte)                           // writes the bytes as given, for a short report
func (screen *Screen) AnswerAfter(count int, report smartmonitor.Report) // sends the report once the count-th written report has arrived
func (screen *Screen) MoveAfter(count int, number int)                 // moves the screen once the count-th written report has arrived
func (screen *Screen) Move(number int)                                 // takes the screen away and brings it back as hidraw<number>
func (screen *Screen) Remove()                                         // takes the screen away for good
func (screen *Screen) Received() []byte                                // every byte written to the node, over all moves
```

How each part works:

* **Making a pty.** Open `/dev/ptmx` with `os.OpenFile(path, os.O_RDWR|syscall.O_NOCTTY, 0)`, unlock it with
  `unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0)` and get its number with `unix.IoctlGetInt(fd, unix.TIOCGPTN)`.
  The file descriptor is reached only through `SyscallConn().Control`, never through `Fd()`, because `Fd()` puts the
  file into blocking mode and a blocked read could then not be ended by `Close`
* **Raw mode.** The fake opens the slave `/dev/pts/<n>` itself and keeps it open for as long as the screen is there,
  without ever reading from it. The kernel resets the termios of a pty slave on its last close, so holding it keeps
  the raw settings when the transport closes and opens the node. On that held slave it sets with `TCGETS` and
  `TCSETS`: `Iflag` without `IGNBRK|BRKINT|PARMRK|ISTRIP|INLCR|IGNCR|ICRNL|IXON`, `Oflag` without `OPOST`, `Lflag`
  without `ECHO|ECHONL|ICANON|ISIG|IEXTEN`, `Cflag` without `CSIZE|PARENB` and with `CS8`, `VMIN` 1 and `VTIME` 0
* **Sysfs and the node.** `Add` writes `<SysfsRoot>/class/hidraw/hidraw<number>/device/uevent` with the six lines of
  the example in the spec, `HID_NAME` and `HID_UNIQ` holding the name and serial given, and makes
  `<DevRoot>/hidraw<number>` a symbolic link to the slave
* **Receiving.** One goroutine per pty reads the master into a buffer of the screen, under a mutex, and records the
  time of the last arrival. After each read it counts the complete reports, the length of everything received
  divided by 65, and carries out each rule of `AnswerAfter` and `MoveAfter` whose count has been reached, once. The
  goroutine ends when its read fails, which closing the master causes
* **Moving and removing.** `Move` deletes the sysfs directory and the link of the old number, closes the old master
  and the held slave, makes a new pty, sysfs entry and link under the new number, and starts a reader on it. The
  received bytes and the count of reports carry on. `Remove` does the first half only. Both are safe to call from
  the reader goroutine, which is where `MoveAfter` calls `Move`
* **Reading back.** `Received` waits until no byte has arrived for 200 milliseconds, polling every 20 milliseconds,
  and returns a copy. That covers both the transport tests in the same process and the end-to-end tests after the
  executable has ended
* **Failure.** Every failure to make a pty, a file or a link is `test.Fatalf`, so the tests fail and are not skipped
  (R29)

### The Linux transport

`internal/hidraw/discover.go`:

```go
type Screen struct {
	Node            string
	Vendor, Product uint16
	Name, Serial    string
}

func (screen Screen) Line() string // R15: "<node>\t%04x:%04x\t<name or ->\t<serial or ->"
func Discover(sysfsRoot, devRoot string) ([]Screen, error)
```

`Discover` reads `<sysfsRoot>/class/hidraw` with `os.ReadDir`. A directory that does not exist,
`errors.Is(err, fs.ErrNotExist)`, gives nil and no error, and any other error is returned wrapped with the path. An
entry counts when its name is `hidraw` followed by one or more decimal digits, all of them digits. For each entry it
reads `device/uevent` with `os.ReadFile`, passes over the entry on any error, and hands the content to the pure
function `parseUevent(content string) (screen Screen, ok bool)`, which splits it into lines, each at its first `=`,
and accepts the `HID_ID` line only when it has exactly three fields that `strconv.ParseUint(field, 16, 32)` reads, with
bus 3 and the IDs of R5. The result is sorted by the number of the entry name with `slices.SortFunc` on a slice the
function made itself (R8, R9). The discovery never opens a node (R10).

`internal/hidraw/transport.go`:

```go
type Transport struct {
	open func() (*os.File, error) // the way it was opened first, used again by Reopen
	file *os.File                 // nil while no node is open
}

func OpenDiscovered(sysfsRoot, devRoot string) (*Transport, error)
func OpenPath(path string) (*Transport, error)
func (transport *Transport) Write(report smartmonitor.Report) error
func (transport *Transport) Read(timeout time.Duration) (smartmonitor.Report, error)
func (transport *Transport) Reopen() error
func (transport *Transport) Close() error
```

* `openDiscovered(sysfsRoot, devRoot)` runs `Discover`, and gives the error `no supported screen found` for none and
  `more than one supported screen is connected, name one with --device:` followed by `"\n"` and the `Line` of each
  screen joined by `"\n"` for several. With one it calls `openNode` on its node. `OpenDiscovered` and the `open` of
  its `Transport` are both this function, so a Reopen does the discovery anew (R11, R12, D14)
* `openNode(path)` opens the path with `os.OpenFile(path, os.O_RDWR|syscall.O_NOCTTY, 0)`. `O_NOCTTY` matters only for
  the fake, whose node is a terminal that must not become the controlling terminal of the process, and changes
  nothing on a hidraw node. Then it calls `file.SetReadDeadline(time.Time{})`; when that fails it closes the file and
  returns `cannot open <path>: the node does not support a time limit for reading: <error>`, with `%w` (R13, D10)
* `openError(path string, err error) error` makes the errors of R14 as a value of an unexported type whose `Unwrap`
  gives `err`. Its text takes the inner error of a `*fs.PathError` where `err` is one, so that the path is not
  written twice. The type is what lets the text be exact while `errors.Is` still reaches the error of the system
* `Write` builds a 65 byte buffer, `0x00` and the report, and fails when no node is open, on the error of the write,
  and when fewer than 65 bytes were taken (R12)
* `Read` fails when no node is open, sets the read deadline to `time.Now().Add(timeout)`, and reads into a buffer of
  64 bytes. `errors.Is(err, os.ErrDeadlineExceeded)` gives `smartmonitor.ErrReadTimeout`. Any other error, `io.EOF`
  included, which is what a read of no bytes gives, is wrapped with the path and not with `ErrReadTimeout`. The bytes
  read are copied into a zero `Report`, which fills a short read up with `0x00` (R12, D11)
* `Reopen` closes the open file, ignoring the error of closing, sets `file` to nil, calls `open` and keeps the file it
  gives, so a failed Reopen leaves no node and a later one tries again (R12)
* `Close` closes the file when there is one and sets it to nil

`OpenDiscovered` and `OpenPath` return `nil, err` on failure, and the caller in `internal/cli` must return a nil
interface in that case and not the nil pointer, so that its check of the error is the only check.

### The command line

`internal/cli` is built around an environment that `Main` fills from the process and the unit tests fill with fakes:

```go
type transport interface {
	smartmonitor.Transport
	Close() error
}

type environment struct {
	stdout, stderr io.Writer
	sysfsRoot      string
	devRoot        string
	now            func() time.Time
	sleep          func(time.Duration)
	listScreens    func(sysfsRoot, devRoot string) ([]string, error)
	openTransport  func(sysfsRoot, devRoot, device string) (transport, error)
}

func Main() int                                   // reads os.Args[1:], os.Stdout, os.Stderr, the two variables of R24, time.Now, time.Sleep and the platform functions, and calls run
func run(args []string, env environment) int
```

`Main` is the one place where the variables of R24 are read; an empty value means `/sys` or `/dev` (R24, D17).

The platform functions live in two files:

* `device_linux.go`, `//go:build linux`: `listScreens` calls `hidraw.Discover` and returns the `Line` of each screen;
  `openTransport` calls `hidraw.OpenPath(device)` when `device` is not empty and `hidraw.OpenDiscovered` otherwise
  (R4, R11)
* `device_other.go`, `//go:build !linux`: both return `fmt.Errorf("%s is not supported yet", runtime.GOOS)` (R3)

`run` takes `start := env.now()`, builds the root command and executes it with `SetArgs(args)`, `SetOut(stdout)` and
`SetErr(stderr)`. On an error it writes `sensordeck: <error>\n` to stderr and returns 1, otherwise 0 (R16, D8). The
root command has `Use: "sensordeck"`, a short description, `SilenceErrors` and `SilenceUsage` set,
`CompletionOptions.DisableDefaultCmd` set, no `Run`, so that cobra writes the help to stdout with status 0 when no
command is given, and the two persistent flags `--device` (string) and `--trace` (bool), each with a usage text
(R16, R17, D7). Each of the six commands has a short description.

Each command's `RunE` first checks its arguments, and `upload` reads its file, then calls a helper `open` that calls
`env.openTransport` with the value of `--device`, wraps the result in the tracing transport when `--trace` is set,
and defers `Close` (R18, D23). `devices` calls `env.listScreens` and never `open` (R15).

`arguments.go` holds the pure parsers, each giving the error texts of the spec word for word:

* `parseValues(args []string) ([]smartmonitor.Value, error)`, with `parseNumber(text string) (uint64, bool)` that
  accepts one or more decimal digits or `0x`/`0X` and one or more hexadecimal digits and nothing else, and reads them
  with `strconv.ParseUint` in base 10 or 16 and bit size 64, so that leading zeros stay decimal (R20, D24). A number
  too large for 64 bits is still a number, which `ParseUint` gives as its largest value with a range error and which the range checks refuse. The range checks follow, the tag first
* `parseRawBytes(args []string) (smartmonitor.Report, error)` (R35)
* `parseSeconds(text string) (int, error)`: one or more decimal digits read with `strconv.Atoi`, 1 or more, and an
  error of `Atoi` is the same error text (R22). `--seconds` is a string flag, default `5` on `listen` and `1` on
  `raw`, so that the text of the error is ours and not that of pflag

`trace.go` holds:

```go
func traceLine(elapsed time.Duration, body string) string // fmt.Sprintf("%12.6f %s", elapsed.Seconds(), body)
func readBody(report smartmonitor.Report) string         // "< " + fmt.Sprintf("% X", report[:])

type tracingTransport struct {
	inner   transport
	out     io.Writer
	elapsed func() time.Duration
}
```

with `Write`, `Read`, `Reopen` writing the six line forms of R23 after the inner call returns, and `Close` passing
through without a line (R23, D16). The listening commands write `traceLine(elapsed(), readBody(report))` to stdout for
each report, so the read line of R22 and R35 is the same function as the read line of the trace. `elapsed` is
`func() time.Duration { return env.now().Sub(start) }`.

The commands, in `commands.go`:

* `clock`: `cobra.NoArgs`; `moment := env.now()`, `EncodeClock(moment)`, one write, `clock sent: ` and
  `moment.Format("2006-01-02 15:04:05")` (R19)
* `values`: `parseValues`, `EncodeValues`, the writes, the line of R20 (R20)
* `upload`: `cobra.ExactArgs(1)`; `os.ReadFile`, with `cannot read the theme file <path>: <error>` where the error is
  the inner error of the `*fs.PathError`, as for R14; then `smartmonitor.Upload(transport, theme, env.sleep,
  progress)` where `progress` writes the stage line of R21 and keeps `blocks`, and on success the line
  `upload complete: <size> bytes in <blocks> data blocks`. The error of `Upload` is returned as it is (R21)
* `listen`: `cobra.NoArgs`; `parseSeconds`, `open`, then the shared function `receive(transport, deadline, env,
  elapsed)` with `deadline := env.now().Add(seconds)` taken after the opening, then `reports received: <count>` (R22)
* `raw`: `parseRawBytes`, `parseSeconds`, `open`, one write, `receive` as for `listen`, `reports received: <count>`
  (R35)

`receive` loops: `remaining := deadline.Sub(env.now())`, stop when it is 0 or less, `Read(remaining)`, stop on
`errors.Is(err, smartmonitor.ErrReadTimeout)`, return any other error, otherwise write the read line and count. So no
read waits beyond the deadline (R22, R35).

The text of `devices` with no screen, `no supported screen found`, is the same as the error of R11; the command writes
it from a constant of its own, because the two are separate decisions that happen to read alike today.

The fake transport of the unit tests of `internal/cli`, in `commands_test.go`, holds a queue of reports to read and a
clock: a read with an empty queue moves the clock forward by its time limit and returns `ErrReadTimeout`. It records
every write, reopen and close, and a fake `openTransport` records the device it was asked for and whether it was
called at all.

### The end-to-end tests

`e2e/main_test.go` has `TestMain`, which builds the executable once into a temporary directory with
`exec.Command("go", "build", "-o", <path>, ".")`, the working directory `..` and the environment `os.Environ()` plus
`CGO_ENABLED=0`, and fails the whole run when the build fails. A helper `run(test, fake, args...)` runs the
executable with the environment plus `SENSORDECK_SYSFS_ROOT=<fake.SysfsRoot>` and `SENSORDECK_DEV_ROOT=<fake.DevRoot>`,
under a context with a time limit of 30 seconds, and returns stdout, stderr and the exit status from
`exec.ExitError.ExitCode()` (R33, D19).

The expected reports of the complete upload are taken from `smartmonitor.Upload` run in the test process against a
recording transport that answers ACK to every read and reopens at once. Unit 0002's own tests prove that these are the
reports of its R20, so the end-to-end test checks that the executable carries them to the node unchanged, rather than
encoding YMODEM a second time. It also checks independently that the first report is the reset command, the second
the ymodem command, and the third starts with `01 00 FF 69 6D 67 2E 64 61 74`.

The rules of the fake for the complete upload, counted in reports as unit 0002's R20 orders them: reset is report 1,
ymodem 2, the header 3 to 5, the data blocks 6 to 22, 23 to 39 and 40 to 56, the end of transmission 57, and the
closing block 58 to 60. So `MoveAfter(1, <other number>)` and `AnswerAfter` with ACK at 2, 5, 22, 39, 56 and 57.

### The layout test

`layout_test.go` in the root package walks the module from `.` with `filepath.WalkDir`, skipping `node_modules`,
`local-themes` and every directory whose name starts with a dot, and parses each `.go` file with
`parser.ImportsOnly`. It checks: no file imports `C` (R1); every file under `internal/hidraw`, `internal/fakescreen`
and `e2e` is excluded from a Windows build; and every file under `internal/cli` that imports
`github.com/norareit/sensordeck/internal/hidraw` is excluded from a Windows build (R4). Exclusion is decided with a
copy of `build.Default` with `GOOS` set to `windows` and its `MatchFile(dir, name)`, which reads both the
`//go:build` line and a `_linux` suffix.

## Files

- `internal/smartmonitor/identity.go`, `identity_test.go`: new, the USB ID constants
- `internal/smartmonitor/upload.go`, `upload_test.go`: `ErrReadTimeout`, the comment of `Read`, the progress function
- `go.mod`, `go.sum`: `github.com/spf13/cobra v1.10.2` and `golang.org/x/sys v0.50.0` as direct requirements, and the
  indirect ones that `go mod tidy` adds, which are `github.com/spf13/pflag` and `github.com/inconshreveable/mousetrap`
- `internal/fakescreen/fakescreen.go`: new, the fake screen
- `internal/hidraw/discover.go`, `discover_test.go`: new, the discovery and the line of R15
- `internal/hidraw/transport.go`, `transport_test.go`: new, the transport
- `internal/cli/main.go`: new, `Main`, `environment`, `transport`, `run` and the root command
- `internal/cli/commands.go`, `commands_test.go`: new, the six commands and `receive`
- `internal/cli/arguments.go`, `arguments_test.go`: new, the parsers
- `internal/cli/trace.go`, `trace_test.go`: new, the tracing
- `internal/cli/device_linux.go`, `device_other.go`: new, the choice of the transport
- `main.go`, `layout_test.go`: new, the executable and the layout check
- `e2e/main_test.go`, `e2e/commands_test.go`: new, the end-to-end tests
- `udev/70-sensordeck.rules`: new, the rule
- `README.md`: a new section `## Linux: access to the screen`, placed before `## Credits`
- `docs/protocol.md`: the sections "Transport" and "Assumptions"
- `AGENTS.md`: the subsection "Testing"

## Tasks

### The package of the protocol

- [x] In `identity_test.go`, test that `VendorID` is `0x0483` and `ProductID` is `0x0065`, and write `identity.go`
  so it passes (R5)
- [x] In `upload_test.go`, add the kind `progress` to the recorded calls of the fake transport, with the stage, block
  and count, and a recording progress function that appends to the same list, so that the order of the progress
  calls among the calls on the transport is checked. Pass it in every existing call of `Upload` (R7)
- [x] Test the complete upload of the theme of 2500 bytes: the recorded list is the existing expected list with a
  progress entry before the first call of each stage and of each of the three data blocks, block numbers 1 to 3 for
  the data blocks and 0 otherwise, and the count 3 in every entry. A progress entry for each reopening stands before
  its reopen call (R7, R32)
- [x] Test NAK and then ACK on data block 2 and on the ymodem command, and a reopening that fails three times after
  the reset: there is exactly one progress entry for that block or stage, and it stands before the first write or
  reopen of it (R7, R32)
- [x] Test that `Upload` of a failing ymodem command makes three progress calls, the reset, the reopening after the
  reset and the ymodem command, and none after (R7)
- [x] In `upload.go`, add `ErrReadTimeout` and the comment of `Read` of R6, and change `Upload` and `sendDataBlocks`
  as the Approach says, so that the tests pass and the existing ones still do (R6, R7)
- [x] Commit: `feature-0003 Report the stages of the upload and name the passed time limit`

### The fake screen and the Linux transport

- [x] Add `golang.org/x/sys` at v0.50.0 with `go get` and write `internal/fakescreen/fakescreen.go` as the Approach
  says, `//go:build linux` on its first line (R2, R29, D18)
- [x] In `discover_test.go`, with trees written by a helper of the test into `test.TempDir()`, and no node made under
  the dev root, so that a discovery that opened a node would fail: no `class/hidraw` directory; an empty one; entries
  of other devices only; one screen with all its fields; `hidraw10` and `hidraw2` given in that order of creation,
  coming back as `hidraw2` first; `HID_ID` in lower case; a missing `HID_NAME` and an empty `HID_UNIQ`, both empty;
  an entry without `uevent`; a malformed `HID_ID`, once with two fields and once with a field that is no hexadecimal;
  vendor and product right on bus `0005`; entries named `hidraw` and `hidrawx1`, passed over; and a `class/hidraw`
  that is a regular file, giving an error (R8, R9, R10, R30)
- [x] Test `Screen.Line` for the screen of the spec on `/dev/hidraw3`, giving
  `/dev/hidraw3\t0483:0065\tllhmi.com llTechCo.,Ltd\t-`, and for one with a serial and no name (R15, R32)
- [x] Write `discover.go` so the tests pass (R5, R8, R9, R10)
- [x] In `transport_test.go`, against the fake screen: a write that arrives as 65 bytes, `0x00` and the report; a
  read of a report the fake sent; two reports sent one after the other and read as two; one byte `0x06` sent and read
  as a report of `0x06` and 63 of `0x00`; a read with a limit of 50 milliseconds and nothing sent, giving an error
  for which `errors.Is(err, smartmonitor.ErrReadTimeout)` is true; a read after `Remove`, giving an error for which it
  is false; a reopening after `Move` to another number, after which a write arrives at the screen; a reopening after
  `Remove`, which fails, after which a write and a read fail, and a reopening after `Add` of a new screen, which
  succeeds; `OpenDiscovered` with no screen and with two, giving the texts of R11 to the character; and `OpenPath` on
  the node of one of two screens, whose write arrives there and not at the other (R11, R12, R31)
- [x] Test that `OpenPath` on a regular file in `test.TempDir()` fails with an error that says the node does not
  support a time limit for reading (R13)
- [x] Test `openError` with `&fs.PathError{Op: "open", Path: "/dev/hidraw3", Err: syscall.EACCES}`, giving the text of
  R14 to the character and `errors.Is(err, fs.ErrPermission)` true, and with `syscall.ENOENT`, giving
  `cannot open /dev/hidraw3: no such file or directory` and `errors.Is(err, fs.ErrNotExist)` true. Test that
  `OpenPath` on a path that does not exist gives the second form (R14, R31)
- [x] Write `transport.go` so the tests pass (R11 to R14, D10, D11, D14)
- [x] Commit: `feature-0003 Add the fake screen and the hidraw transport`

### The executable

- [x] Add `github.com/spf13/cobra` at v1.10.2 with `go get` (R2, D7)
- [x] In `arguments_test.go`, test `parseValues`: `1=0x1234 255=1` gives the two pairs; `010=1` gives tag 10;
  `0x1F=0XFFFF` gives tag 31 and value 65535; no arguments; `1`, `1=2=3`, `=1`, `1=`, `-1=1`, `+1=1`, ` 1=1`, `0x=1`,
  `0b1=1` and `1=1.0`, each the error naming the argument in double quotes; `256=1`, the tag error; `1=65536`, the
  value error; and numbers beyond 64 bits in decimal and hexadecimal, in the tag and in the value, the range error of the field (R20, R32)
- [x] Test `parseRawBytes`: `03 01 15`, `030115` and `0301 15` give the same report of `03 01 15` and 61 of `0x00`;
  `ab` and `AB` give `0xAB`; no arguments; `0x03`, `3`, an empty argument and `0g`, each the error of pairs; 64 bytes
  in three arguments accepted; 65 bytes refused with `raw takes at most 64 bytes, got 65` (R35, R32)
- [x] Test `parseSeconds`: `1`, `5` and `05` accepted; `0`, `-1`, `1.5`, `x`, the empty text and a number of 30
  digits refused with the text of R22 (R22, R32)
- [x] In `trace_test.go`, with a fake transport and a fixed `elapsed`, test the six line forms of R23, and the two
  example lines of the spec to the character, with 412 microseconds for the reset report written and 1.503377 seconds
  for the ACK read; that `Close` writes nothing; and that the calls reach the inner transport unchanged (R23)
- [x] In `commands_test.go`, with the fake environment of the Approach, test: no command and `--help`, which write to
  stdout a help naming `devices`, `clock`, `values`, `upload`, `listen`, `raw`, `--device` and `--trace`, with status
  0, and no `completion` command; `values --help` with status 0; an unknown command and an unknown option, each one
  line `sensordeck: ...` on stderr with no usage text and status 1; `--device x values 1=1` and
  `values 1=1 --device x`, both opening `x`; `--trace` before and after the command; `devices` with two lines and
  with none, the second writing `no supported screen found` to stderr with status 0 and never opening a transport;
  the wrong arguments of `clock`, `values`, `upload` with a missing file, `listen` and `raw`, none of which opens the
  transport; `clock` with a fixed time; `values 1=0x1234 255=1`; `upload` of 2500 bytes with the output of R21 to the
  character and with an answer `0x43` to the ymodem command, giving stdout up to `ymodem command` and the error text
  of R21; `listen --seconds 1` with two reports queued, giving two read lines and `reports received: 2`; and
  `raw 03 01 15` with one report queued. Every command that opened the transport closed it (R15 to R22, R35)
- [x] Write `arguments.go`, `trace.go`, `main.go`, `commands.go`, `device_linux.go` and `device_other.go` in
  `internal/cli` as the Approach says, so the tests pass (R3, R4, R15 to R24, R35, D7, D8, D9, D16, D17, D23, D24)
- [x] Write `layout_test.go` in the root as the Approach says, and `main.go` with `package main` and the one call of
  `cli.Main` (R1, R4, D5)
- [x] Run `go mod tidy` and check that `go.mod` keeps `go 1.27` with no `toolchain` line, and has cobra and x/sys as
  its only direct requirements (R2)
- [x] Commit: `feature-0003 Add the executable and its commands`

### The end-to-end tests

- [x] Write `e2e/main_test.go` with `TestMain` and the helper `run` as the Approach says, `//go:build linux` on the
  first line (R33, D19)
- [x] In `e2e/commands_test.go`, `//go:build linux` on the first line, write the cases of R33: `devices` with no
  screen, with one on `hidraw3` and with two on `hidraw2` and `hidraw10`, each with the exact stdout, stderr and
  status 0
- [x] `values 1=0x1234 255=1`, the 65 bytes of R20 arriving and the line of R20, and 21 pairs, two reports arriving,
  the first with count 20 and the second with count 1 (R33)
- [x] `values 1` with no screen: stdout empty, stderr
  `sensordeck: invalid argument "1": want <tag>=<value>` and status 1 (R33)
- [x] `clock`: the time before and after the run, bytes 0 to 2 `03 01 15`, byte 10 `0x64`, bytes 11 to 63 `0x00`, and
  bytes 3 to 9 a time in `time.Local` between the start truncated to the second and the end, with byte 9 its ISO
  weekday (R33)
- [x] `--trace upload <theme>` of the theme of 2500 bytes where byte *i* is *i* modulo 256, against a screen on
  `hidraw3` with `MoveAfter(1, 7)` and the ACK rules of the Approach: status 0, the stdout of R21 to the character,
  the bytes that arrived being `0x00` and each expected report in order, and on stderr 60 lines with `>`, 6 with `<`
  and 2 with `* reopen` and nothing else, each matching R23 in its prefix of 12 characters and its 64 groups, and the
  write lines carrying the expected reports in order (R33)
- [x] `--trace upload <theme>` against a screen with `MoveAfter(1, 7)` and `AnswerAfter(2, 0x43)`: status 1, stdout
  ending with `ymodem command\n`, the last line of stderr
  `sensordeck: upload failed at the ymodem command: unexpected answer 0x43`, a trace line `< 43 ...`, and exactly
  130 bytes arrived (R33)
- [x] `listen --seconds 1` with two reports sent before the run: two read lines with those reports,
  `reports received: 2`, status 0 (R33)
- [x] `raw 03 01 15` with `AnswerAfter(1, <report starting 0x43>)`: the 65 bytes of R35 arrived, stdout one read line
  with that report and `reports received: 1`, status 0 (R33)
- [x] Two screens: `values 1=1` without `--device`, the error of R11 with both lines and status 1; with
  `--device <node of the second>`, the report arriving at the second and nothing at the first (R33)
- [x] Add a comment at the top of `e2e/commands_test.go` that lists what the suite leaves out and where it is covered,
  as R34 says (R34)
- [x] Commit: `feature-0003 Add the end-to-end tests`

### The udev rule and the documents

- [x] Write `udev/70-sensordeck.rules` with a comment line on what it does and why it sorts before
  `73-seat-late.rules`, and the rule line of R25 (R25, D4, D21)
- [x] Add the section `## Linux: access to the screen` to `README.md` before `## Credits`, with what R26 lists: hidraw
  and the udev rule, the rule line in a code block, the file, the copying to `/etc/udev/rules.d/` and the replugging,
  and the `usbhid` driver, which must stay bound, with the advice to remove a rule that unbinds it. It names no
  command to reload the rules. Nothing else of the README changes (R26, D21)
- [x] In the section "Transport" of `docs/protocol.md`, add that an input report is read from the node as 64 bytes
  without a report ID byte because the device uses no numbered reports, that the screen is found through the `HID_ID`
  line of the `uevent` file in sysfs, and that hidraw needs the `usbhid` driver bound (R27)
- [x] In the section "Assumptions" of `docs/protocol.md`, replace the sentence that names this unit as where they are
  checked with one saying that the human checks them on the device with the commands of this unit and that they stay
  unconfirmed until then, and add the assumption that an input report shorter than 64 bytes is filled up with `0x00`.
  Nothing else of the document changes (R27)
- [x] In the subsection "Testing" of `AGENTS.md`, name the five commands of the suite, the three it names now and
  `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...`, and replace the statement
  that there is no end-to-end suite with: the end-to-end tests are in `e2e/`, `go test ./...` runs them, they build
  the executable and run it against a fake screen made of a directory tree and pseudo terminals, they need Linux with
  `/dev/ptmx`, and `SENSORDECK_SYSFS_ROOT` and `SENSORDECK_DEV_ROOT` exist for them (R28)
- [x] Commit: `feature-0003 Add the udev rule and document the transport`

## Verification

- [x] `go test ./...` passes, with `-count=1` so that the end-to-end tests really run (R29 to R33)
- [x] `go vet ./...` reports nothing
- [x] `gofmt -l .` prints no file name
- [x] `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` succeed and report
  nothing (R3)
- [x] `CGO_ENABLED=0 go build ./...` and `go build .` succeed, the latter giving an executable named `sensordeck` (R1)
- [x] `go mod tidy -diff` reports no difference, and `go.mod` holds `go 1.27` and the two direct requirements (R2)
- [x] `go test ./internal/smartmonitor` still passes its import check (R4)
- [x] `grep -rn 'udevadm trigger' --include='*.md' . --exclude-dir=node_modules --exclude-dir=changes` finds nothing
  (R26)
- [x] `npm run check-changes` passes on the committed work

## Commit Points

- [x] `feature-0003 Report the stages of the upload and name the passed time limit`
- [x] `feature-0003 Add the fake screen and the hidraw transport`
- [x] `feature-0003 Add the executable and its commands`
- [x] `feature-0003 Add the end-to-end tests`
- [x] `feature-0003 Add the udev rule and document the transport`
