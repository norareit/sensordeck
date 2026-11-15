# Starting `run` at logon on Windows

## Approach

The work falls into six parts. The first is a refactoring that changes no behaviour, and the others add behaviour on
top of it. Everything but the calls into Windows is in `internal/cli` and is tested on Linux through the environment of
the commands, with the system set to `windows` by its field (R34).

* **The refactoring** (one commit, no change of behaviour): the start of `run` and its feeding are split into two
  methods, so that a background run can repeat the start; the call of a program and the text of its failure are
  shared by `systemctl` and `schtasks`; the config path of `install` is one method that both systems use
* **The package of the Windows calls** is `internal/winbackground`: the stop event of R23 and R24 and the detach of
  R25, behind a seam file that alone imports `golang.org/x/sys/windows`, with its test limited to Windows, the layout
  test and the Windows job of CI widened for it (R31 to R33)
* **The log** is a writer in `internal/cli/runlog.go` that puts the time in front of each line, writes the line whole,
  and rotates at 1 MiB (R20 to R22)
* **The background run** is the flag `--background` of `run` and the work around the start in
  `internal/cli/background.go`: the log, the stop event, the detach and the retry (R17 to R19, R23, R25, R26)
* **The task file** is pure computation in `internal/cli/taskfile.go`: the quoting, the XML of R2 in UTF-16 with CR LF,
  and the reading of the output of `/Query /XML` (R2, R5, R14)
* **The commands on Windows** are in `internal/cli/startuptask.go`, chosen by `service.go` by the system, with the
  stop of R16 (R1, R3, R4, R6 to R16, R27)

None of the items of "Beyond the request" is built, as the record decided.

The spec fits the tree. `os.UserCacheDir()` is `env.userCacheDir`, already used by the state file with the error
`cannot find the cache directory: <cause>`. `executablePath` gives the executable of R3 of unit 0014, and the config
path of `unitFileFor` gives the config file of R4 of unit 0014. `checkConfig` is the check of R7 of unit 0014, hint
included. `golang.org/x/sys/windows` v0.50.0 has `CreateEvent`, which gives a valid handle together with
`ERROR_ALREADY_EXISTS` for an event that exists, `OpenEvent`, `SetEvent`, `WaitForMultipleObjects` and `CloseHandle`,
and has no `FreeConsole`, which is therefore found in `kernel32.dll` with `NewLazySystemDLL`, which the layout test
allows.

### Decisions of the plan

The spec leaves these open, and they are decided here so that the tests and the code agree. None of them changes what
the spec makes true.

* **A failed check of the stop event.** An error of the opening by R24 other than that no event of the name exists
  fails `install` and `uninstall` with exit status 1 and the error of the table, before the stop and while waiting
  alike, and makes no further call. In `status` it is `active: unknown`, as R14 says
* **A stop event that vanishes before it is signalled.** A background run that ended between the check and the signal
  needs no stop: the signal finds no event and the stop has succeeded
* **The `Enabled` element.** It is read as an `xs:boolean` of the schema, its space trimmed: `true` and `1` are
  `enabled`, `false` and `0` are `disabled`, and any other value is `unknown`. A root element other than `Task`, or a
  `Settings` element with no `Enabled` child, follow R14: the first is `unknown`, the second `enabled`
* **A failure of a background run after its start.** The feeding of `run` fails only when the clock cannot be encoded.
  In a background run that error is written to the log as `sensordeck: <error>` and the run ends with exit status 1,
  with nothing on standard error, because R19 sends every line of `run` to the log. The function `run` of `main.go`
  writes nothing for an error that is `errLogged`, the sentinel a background run returns once it wrote its error to
  the log
* **A stop event that cannot be created** for another reason than that it exists ends the background run with
  `sensordeck: <error of the table>` in the log and exit status 1, as an error of the start that the retry cannot
  repair would
* **A partial line at the end.** The log writes a line once its line feed arrives. At its close it writes what is left
  with a line feed added, so that no text of `run` is lost
* **A line larger than 1 MiB** is written to an empty `run.log` without a rotation, so that the rotation cannot repeat
  itself; this is what bounds the two files to 2 MiB plus two lines
* **The help texts on a system other than Linux and Windows** are those of Linux, as today
* **The wait of R16** uses `env.wait` with no stop for each step of 200 milliseconds and `env.now` for the deadline of
  30 seconds, so that the tests move the fake clock and make no real wait

### The names and texts the implementation uses

`<cause>` is the cause alone, through the existing `innerError` where the error is one of the file system, so that a
path is not written twice. `<event>` is `Local\sensordeck-background-run`.

| What | Exact text |
|---|---|
| Short of `service` on Windows | `Run sensordeck run as a startup task of the Task Scheduler` |
| Short of `service install` on Windows | `Write, create and start the startup task of run` |
| Usage of `--config` of `install` on Windows | `name this config file in the task instead of the one of the user` |
| Short of `service status` on Windows | `Show whether the startup task is installed, enabled and running` |
| Short of `service uninstall` on Windows | `Stop and delete the startup task and remove its file` |
| Usage of `--background` of `run`, hidden | `write to the log and start again after a failed start, as the startup task does` |
| No cache directory (R3, R20) | `cannot find the cache directory: <cause>` |
| No user (R4) | `cannot find the user: <cause>` |
| A refused path (R5) | `cannot write the path <path as %q> into the task file: it holds ", % or a control character` |
| A task file that cannot be written (R9) | `cannot write the task file <task file>: <cause>` |
| A task file whose presence cannot be checked | `cannot read the task file <task file>: <cause>` |
| A task file that cannot be removed (R12) | `cannot remove the task file <task file>: <cause>` |
| A call that failed (R9, R15, R16) | `schtasks <arguments joined by spaces> failed: <cause of the call>` |
| A stop event that cannot be signalled (R16) | `cannot signal the stop event <event>: <cause>` |
| A check of the stop event that failed | `cannot check the stop event <event>: <cause>` |
| A stop event that cannot be created | `cannot create the stop event <event>: <cause>` |
| A log that cannot be opened (R19) | `cannot open the log <log>: <cause>` |
| Another system (R27) | `the service is not supported on <runtime.GOOS>` |
| The lines of `install` (R7, R8) | `wrote <task file>`, then `the startup task sensordeck is created and started` |
| The line of `uninstall` with no task file (R10) | `the startup task sensordeck is not installed` |
| The line of `uninstall` (R11) | `the startup task sensordeck is stopped and removed` |
| The line after `/End` (R16) | `sensordeck run did not stop within 30 seconds and was ended with schtasks /End` |
| The line of a second background run (R23) | `another background run of sensordeck is running` |
| The lines of a failed start (R26) | `sensordeck: <error>`, then `starting again in 10s` |
| The lines of `status` (R13, R14) | `installed: no`; or `installed: yes, <task file>`, `enabled: <enabled>`, `active: <active>`, `log: <log>` |

`<cause of the call>` is what R9 says, and what `systemctl` gives today: the standard error of the call trimmed, and
the text of its error when that is empty.

### The refactoring

```go
// In systemctl.go: runProgram runs the program with the arguments and gives what it wrote to standard output and
// standard error, with the error of Run. runSystemctl is runProgram("systemctl", "--user", arguments...).
func runProgram(name string, arguments ...string) (string, string, error)

// In service.go: callFailure names the call, as "<program> <arguments joined by spaces>", and gives the trimmed
// standard error as the cause, or the error of the call when it wrote none. current.systemctl uses it with
// "systemctl --user".
func callFailure(program string, arguments []string, stderr string, err error) error

// In service.go: serviceConfigPath gives the config file named, or the one of the user, made absolute, its links kept.
// unitFileFor calls it in place of its own lines.
func (current *session) serviceConfigPath(named string) (string, error)
```

`run` is split in `internal/cli/run.go` as follows, with the `lockedWriter` around `env.stderr` moved to the top of
the `RunE`, before the start, so that it is put there once:

```go
// preparedRun is what a start of run made before it opens the screen.
type preparedRun struct {
	path     string
	stamp    fileStamp
	started  runConfig
	state    stateFile
	stateErr error
	sources  sourceSet
}

// prepareRun reads and loads the config file, finds the state file and makes the sources, as run does at its start.
func (current *session) prepareRun(named string) (preparedRun, error)

// serveRun writes the line running, feeds the screen until stop is done, ends the keeper and closes the transport, and
// writes stopped. Its error is the error of the feeding.
func (current *session) serveRun(stop context.Context, prepared preparedRun, opened transport, closeTransport func()) error
```

The `RunE` of `run` becomes `prepareRun`, `notifyStop`, `current.open()`, `serveRun`, in the order of today.

### The package of the Windows calls

`internal/winbackground` holds `doc.go`, with the package comment alone and no build constraint, so that the package
exists on every system, and `seam_windows.go` and `seam_windows_test.go`:

```go
// ErrExists is the error of Create when an event of the name exists already.
var ErrExists = errors.New("an event of the name exists already")

// Event is a named manual-reset event that this process created and holds until Close.
type Event struct { /* the handle, an unnamed event that Close sets, the channels signalled and done */ }

// Create creates the event, not signalled. When one of the name exists it closes the handle CreateEvent gave and
// gives ErrExists. A goroutine waits with WaitForMultipleObjects on the event and on the unnamed event of Close, and
// closes the channel of Signalled when the event was signalled.
func Create(name string) (*Event, error)
func (event *Event) Signalled() <-chan struct{}
// Close sets the unnamed event, waits for the goroutine to end, and closes both handles.
func (event *Event) Close() error

// Exists opens the event with SYNCHRONIZE and closes it at once. ERROR_FILE_NOT_FOUND is false with no error.
func Exists(name string) (bool, error)
// Signal opens the event with EVENT_MODIFY_STATE, sets it and closes it. ERROR_FILE_NOT_FOUND is false with no error.
func Signal(name string) (bool, error)
// FreeConsole is FreeConsole of kernel32.dll, found with NewLazySystemDLL.
func FreeConsole() error
```

`internal/cli/background_windows.go` fills the four fields of the environment below from the package, with the name
`Local\sensordeck-background-run`, and maps `ErrExists` to `errStopEventExists` of `internal/cli`.
`internal/cli/background_other.go`, `//go:build !windows`, gives functions that fail with `the stop event exists only
on Windows`; nothing calls them, because the code calls them only when `env.system` is `windows`.

### The seam

```go
// In environment:
	// schtasks runs schtasks.exe with the arguments, as systemctl runs systemctl --user.
	schtasks func(arguments ...string) (stdout, stderr string, err error)
	// currentUser gives the security identifier of the user on Windows, the Uid of os/user.Current.
	currentUser func() (string, error)
	// holdStopEvent creates the stop event and holds it until its close, or gives errStopEventExists.
	holdStopEvent func() (heldStopEvent, error)
	// findBackgroundRun tells whether the stop event can be opened, and closes it at once.
	findBackgroundRun func() (bool, error)
	// signalBackgroundRun signals the stop event. An event that does not exist is no error.
	signalBackgroundRun func() error
	// detachConsole detaches the process from its console.
	detachConsole func() error

// heldStopEvent is the stop event that a background run holds.
type heldStopEvent interface {
	signalled() <-chan struct{}
	close()
}
```

`Main` fills `schtasks` with `runSchtasks` of `internal/cli/schtasks.go`, which is `runProgram("schtasks.exe",
arguments...)`, `currentUser` with a function of `os/user.Current`, and the other four with the functions of
`background_windows.go` or `background_other.go`.

The harness of `internal/cli/commands_test.go` gets: the calls of `schtasks` recorded in `calls` and answered by
`answer`, as those of `systemctl`, because a test runs one system; `user` and `userErr`; `cacheDirCalls` and
`userCalls`; `stopEvent`, a fake whose channel a test closes, `stopEventErr`, and `holds`, the count of its creations;
`backgroundRun func(check int) (bool, error)`, which answers the checks in turn and finds nothing when it is nil;
`signals` and `signalErr`; `detaches`; and `steps []string`, which records in order `signal`, `end of the event` and
each call as `schtasks <arguments>`, so that a test can check the order of R7, R11 and R16.

### The log

```go
// runLog is the log of a background run: run.log in a directory, with run.log.1 beside it.
type runLog struct {
	mutex   sync.Mutex
	path    string
	file    io.WriteCloser // nil after a failed reopening, so that the next line tries again
	size    int64
	pending []byte // what was written after the last line feed
	now     func() time.Time
	open    func(path string, truncate bool) (io.WriteCloser, int64, error) // os.OpenFile with append, or with truncate
	rename  func(from, to string) error                                     // os.Rename
}

// logPath gives sensordeck/run.log in the cache directory.
func logPath(cacheDirectory string) string

// openRunLog creates the directory with 0o755, opens the log with 0o644 for appending, and rotates it first when it is
// larger than 1 MiB.
func openRunLog(path string, now func() time.Time) (*runLog, error)

// Write keeps what follows the last line feed for later, and writes each whole line with its time in front in one
// write, rotating first when the line would make run.log larger than 1 MiB and run.log is not empty.
func (log *runLog) Write(data []byte) (int, error)

// Close writes what is pending with a line feed and closes the file.
func (log *runLog) Close() error
```

The rotation closes the file, renames `run.log` to `run.log.1`, and opens a new `run.log`; when the renaming fails it
opens `run.log` truncated; when that fails too, the line is dropped and `file` is nil, so that the next line opens it
again by the same steps. `Write` always reports the whole of `data` as written, because `run` has nowhere else to put a
line. The time is `now().Format("2006-01-02T15:04:05-07:00")`.

### The background run

```go
// runInBackground does what run does with --background: it opens the log, holds the stop event and detaches on
// Windows, and repeats the start until it succeeds or a stop comes.
func (current *session) runInBackground(named string) error
```

In this order: the cache directory and `openRunLog`, whose errors are returned as they are and so reach standard
error; `env.stdout` and `env.stderr` both set to the log, and the log closed when the run ends; on Windows
`holdStopEvent`, with `errStopEventExists` giving the line of R23 and `errLogged`, and another error the line
`sensordeck: <error>` and `errLogged`; on Windows `detachConsole`, its error ignored; `notifyStop`, wrapped in a
`context.WithCancel` whose cancel a goroutine calls once the event is signalled; then the loop of R26: `prepareRun`
and `current.open()`; on an error, the two lines of R26 and `env.wait(stop, 10*time.Second)`, and on a stop during the
wait the line `stopped` and nil; on success `serveRun`, whose error is written as `sensordeck: <error>` and gives
`errLogged`. The event is closed when the run ends.

### The task file

```go
// taskName is the name of the task in the root folder of the Task Scheduler.
const taskName = "sensordeck"

// taskFilePath gives sensordeck/startup-task.xml in the cache directory.
func taskFilePath(cacheDirectory string) string

// checkTaskPath gives the error of the table for a path that holds ", % or a character below 0x20.
func checkTaskPath(path string) error

// taskFileContent gives the bytes of R2: the text written in the code as one raw string literal with \n, the three
// values escaped for XML, the line feeds made CR LF, encoded as UTF-16 little-endian behind FF FE.
func taskFileContent(user, executable, config string) ([]byte, error)

// taskEnabled reads the output of /Query /XML, as UTF-16 little-endian behind FF FE and as UTF-8 otherwise, with the
// CharsetReader of encoding/xml giving its input unchanged so that the declaration is ignored, and gives enabled,
// disabled or unknown.
func taskEnabled(output string) string

// taskStatusLines gives the four lines of R14.
func taskStatusLines(taskFile, enabled, active, log string) []string
```

### The commands on Windows

`service.go` keeps the tree. Each `RunE` calls the function of Linux on `linux`, the function of Windows on
`windows`, and gives the error of R27 on any other system before anything else. The Short texts and the usage of
`--config` are those of the table on `windows` and those of today elsewhere.

`startuptask.go` holds:

* `startupTaskInstall(named string) error`, in this order: the task file from `env.userCacheDir` (R3); the executable
  by `executablePath` (R5); the config file by `serviceConfigPath` (R5); the user by `env.currentUser` (R4);
  `taskFileContent`, which refuses a path by R5; `checkConfig` (R6); `stopBackgroundRun` (R7, R16); the directory and
  the file, with `0o755` and `0o644` (R7, R9); the line `wrote`; `/Create /TN sensordeck /XML <task file> /F` and
  `/Run /TN sensordeck`, each through `current.schtasks`, which gives the error of the table (R7, R9); the last line
  (R8)
* `startupTaskUninstall() error`: the task file (R3); its presence with `installed`, generalized to take the name of
  the file in its error; the line of R10; `stopBackgroundRun`; `/Delete /TN sensordeck /F`; `os.Remove`; the line of
  R11 (R10 to R12)
* `startupTaskStatus() error`: the task file (R3); its presence; `installed: no` (R13); `/Query /TN sensordeck /XML`
  (R15); `findBackgroundRun` as `running`, `stopped` or `unknown`; the log path from the same cache directory; the four
  lines (R14)
* `stopBackgroundRun() error`: `findBackgroundRun`, nothing to do when it finds none; `signalBackgroundRun`; checks
  with a wait of 200 milliseconds before each, until none is found or 30 seconds passed by `env.now`; then `/End /TN
  sensordeck` and its line (R16)

## Tasks

### 1. The refactoring

- [x] Add `runProgram` to `internal/cli/systemctl.go` and make `runSystemctl` call it (R9)
- [x] Add `callFailure` to `internal/cli/service.go` and make `current.systemctl` use it, with the text of today (R9)
- [x] Add `serviceConfigPath` to `internal/cli/service.go` and make `unitFileFor` use it (R5)
- [x] Split the `RunE` of `run` in `internal/cli/run.go` into `prepareRun` and `serveRun` as the approach says, with
  the `lockedWriter` at the top, the order of the calls unchanged (R26)
- [x] Run the five commands of the suite, which pass with no test changed (R36)
- [x] Commit: `feature-0019 Split the start of run and share the calls of the service, with no change of behaviour`

### 2. The package of the Windows calls

- [x] Add `internal/winbackground/seam_windows.go` and `internal/winbackground/seam_windows_test.go` to the allowed list
  of `TestOnlyTheSeamOnWindowsImportsTheWindowsPackage` in `layout_test.go` (R31)
- [x] Write `internal/winbackground/seam_windows_test.go`, `//go:build windows`: create an event under
  `Local\sensordeck-test-<pid>-<nanoseconds>`, check that `Create` of the same name gives `ErrExists`, that `Exists`
  finds it, that `Signal` gives true and the channel of `Signalled` closes within 5 seconds, that after `Close` `Exists`
  gives false and `Signal` gives false; `FreeConsole` is not called (R24, R32)
- [x] Write `internal/winbackground/doc.go` and `internal/winbackground/seam_windows.go` as the approach says, with
  `kernel32.dll` loaded by `windows.NewLazySystemDLL` for `FreeConsole` alone (R23, R24, R25, R31)
- [x] Add `./internal/winbackground` to the step `go test ./internal/winhid ./internal/commandercore
  ./internal/windowssensors` of the job `windows` in `.github/workflows/ci.yml`, after the three, and to the command
  that `checkWindowsJob` of `release_test.go` checks (R33)
- [x] Run `GOOS=windows GOARCH=amd64 go vet ./...`, which vets the test of the package, and `go test .` (R31 to R33)
- [x] Commit: `feature-0019 Add the stop event and the detach of a background run on Windows`

### 3. The log

- [x] Write `internal/cli/runlog_test.go`, against a temporary directory and a fake time: `logPath`; the time in front
  of each line in the format of R21; two lines given in one `Write` and one line given in three, each written whole in
  one write of the file, which a fake `open` that records the writes checks; a line that would pass 1 MiB rotating to
  `run.log.1`, replacing one that exists; a log larger than 1 MiB at its opening rotated first; a line larger than
  1 MiB in an empty log written with no rotation; a failed renaming, with `run.log.1` a directory that is not empty,
  that empties `run.log`; a fake `open` that fails after the failed renaming, the line dropped and the next line
  written once `open` succeeds again; the directory created when it is missing; a log that cannot be opened, with its
  directory a file, giving the error of the table; `Close` writing a pending partial line with a line feed (R20, R21,
  R22)
- [x] Write `internal/cli/runlog.go` as the approach says (R20, R21, R22)
- [x] Commit: `feature-0019 Add the log of a background run`

### 4. The background run

- [x] Add the six fields of the seam to `environment` in `internal/cli/main.go`, `heldStopEvent`,
  `errStopEventExists` and `errLogged`, fill them in `Main`, and make `run` of `main.go` write nothing for `errLogged`;
  write `internal/cli/schtasks.go`, `internal/cli/background_windows.go` and `internal/cli/background_other.go` (R18,
  R23, R24, R25)
- [x] Extend the harness of `internal/cli/commands_test.go` with the fields of the approach (R34)
- [x] Write `internal/cli/background_test.go`, with the cache directory a temporary one: `run --help` does not name
  `--background` (R18); on `linux` and on `windows` the lines of a run that starts, feeds and is stopped by
  `cancelStop` are in the log with their time, and stdout and stderr are empty (R19); a cache directory that cannot be
  found and a log that cannot be opened end with exit status 1 and the error on stderr (R19, R20); on `windows` the
  event is held before the detach, and the fake event signalled during the feeding ends the run with the teardown of
  the controller, the session of `startSession` ended, `stopped` in the log and exit status 0 (R17, R23, R25); a
  failing detach is ignored (R25); `errStopEventExists` gives the line of R23 in the log and exit status 1, with no
  detach and no source made (R23); another error of the creation gives its line in the log and exit status 1; on
  `linux` no event is held and no detach is made; a start that fails with no screen writes the two lines of R26, waits
  10 seconds, and starts again with the config file read anew, which the test changes during the wait, then feeds
  (R26); a stop during the wait ends at once with `stopped` and exit status 0 (R26); a signal of the fake event during
  the wait does the same (R26); a config that does not load at the first start is retried as well (R26)
- [x] Write `internal/cli/background.go` and the flag `--background`, hidden, in `internal/cli/run.go`, whose `RunE`
  calls `runInBackground` when the flag is given and does what it does today otherwise (R17, R18, R19, R23, R25, R26)
- [x] Write `e2e/background_test.go`, `//go:build linux`, with `XDG_CACHE_HOME` a temporary directory the test names:
  `run --background --config <file>` against the fake screen writes `running: 3 metrics every 100ms` with its time to
  `<XDG_CACHE_HOME>/sensordeck/run.log` and nothing to stdout or stderr, and ends on SIGTERM with `stopped` in the log
  and exit status 0; with no screen it writes `sensordeck: ` and the error and `starting again in 10s` to the log,
  still runs after them, and ends on SIGTERM with `stopped` in the log and exit status 0 (R35)
- [x] Run the five commands of the suite (R36)
- [x] Commit: `feature-0019 Add the background run of run with its log and its retry`

### 5. The task file

- [x] Write `internal/cli/taskfile_test.go`: `taskFilePath`; the bytes of `taskFileContent` for a user, an executable
  and a config compared with the expected XML of R2 written in the test, with `FF FE` first, every line ending in CR
  LF, the last included, and the decoded text equal; paths with a space, `&`, `<` and `>` escaped; a path with `"`,
  one with `%`, one with a line feed and one with a tab refused with the error of the table; `taskEnabled` for an
  output in UTF-16 with `FF FE` and one in UTF-8, each with a declaration naming UTF-16, with `Enabled` `true`,
  `false`, `1`, `0`, a value of neither, and no `Enabled` element, a root that is not `Task`, and an output that is not
  XML; `taskStatusLines` (R2, R5, R14)
- [x] Write `internal/cli/taskfile.go` as the approach says (R2, R5, R14)
- [x] Commit: `feature-0019 Add the task file and the reading of the task`

### 6. The commands on Windows

- [x] Write `internal/cli/startuptask_test.go`, with the system `windows`, the config and executable of
  `serviceHarness`, a temporary cache directory and the user `S-1-5-21-1004336348-1177238915-682003330-1001`:
    - `install` writes the task file of `taskFileContent` with and without `--config`, with a relative `--config` made
      absolute, and with paths that hold a space, `&`, `<` and `>`; the two lines and the two calls in order (R2, R5,
      R7, R8)
    - a path of the executable or the config with `"`, `%` or a line feed, a user that cannot be found and a cache
      directory that cannot be found each fail with their error, with no file written and no call (R3, R4, R5)
    - a missing config file with its hint, a file that does not parse and a theme that does not compile each fail with
      no file written and no call (R6)
    - a task file that exists replaced; a running background run signalled and waited for before the file is written
      and before `/Create`, by `steps` and by the absence of the file at the signal (R7)
    - a task file that cannot be written, with its directory a file, fails with no call; a failure of `/Create` with no
      `/Run`, and of `/Run`, each with a standard error and with an empty one, the task file left in place; an error
      of a start, as `exec: "schtasks.exe": executable file not found in %PATH%`, as the cause (R9)
    - `uninstall` with no task file writes its line with no call, no signal and no check (R10); with the task file,
      the stop, `/Delete` and the removal in that order and the line (R11); a failed signal, a failed check and a
      failed `/Delete` each failing with the file kept and no further call; a task file that cannot be removed,
      a directory that is not empty at its path, failing after `/Delete`
      (R12)
    - `status` with no task file writes `installed: no` with no call (R13); with it, the four lines with the task
      enabled, disabled and with no `Enabled` element, an output in UTF-16 and in UTF-8, an output that is not XML,
      and `running`, `stopped` and `unknown`; a failure of `/Query` (R14, R15)
    - the stop of R16: a run that ends after three checks with no `/End` and waits of 200 milliseconds; one that never
      ends, with `/End` after 30 seconds of the fake clock and its line; a failure of `/End`; a stop event that cannot
      be signalled (R16)
    - the help of `service` and its subcommands on `windows` names the startup task, on `linux` the systemd user
      service, and `service` alone on `windows` shows the help (R27)
- [x] Change `TestEveryServiceCommandFailsOnAnotherSystemAndTouchesNothing` in `internal/cli/service_test.go` to the
  system `darwin` and the text of R27, and check that neither `userCacheDir` nor `currentUser` is called and that no
  stop event is held, found or signalled (R27)
- [x] Write `internal/cli/startuptask.go` and the choice by the system in `internal/cli/service.go`, with `installed`
  taking the name of the file in its error, as the approach says (R1, R3 to R16, R27)
- [x] Run the five commands of the suite (R36)
- [x] Commit: `feature-0019 Manage the startup task on Windows with the service commands`

### 7. The documents

- [x] Rewrite the section "Running as a service" of `README.md` with the part of Linux as it is, under a heading of its
  own, and a part on Windows that says each point of R28, describes the task as R2 writes it and claims nothing as
  seen on Windows (R28, R30)
- [x] Change the introduction, the list of features and the lines of `service` under "Commands" of `README.md`, so that
  they name the startup task on Windows and no longer say that the service is not available on Windows yet or that the
  task is still to come (R29)
- [x] Commit: `feature-0019 Document the startup task on Windows`

### 8. The suite

- [ ] Run the five commands of the suite in `AGENTS.md` and `npm run check-changes`, all of which succeed (R36)
