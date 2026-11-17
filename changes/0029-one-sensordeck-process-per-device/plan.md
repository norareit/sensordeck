# Plan: one sensordeck process at a time on the controller and on the screen

## Approach

A new package `internal/devicelock` holds what the two systems share: the error of a lock held by another process,
the text that names the holder, the lookup of the holder in `<proc root>/locks` and the content of the lock file of
Windows. Its system files take the lock: `flock` on an opened node on Linux, `LockFileEx` on a file of the cache
directory on Windows. The places that open a device take it: on Linux the opening of `hidraw.Transport` and the
opening of the finder of `commandercore`, so that a `Reopen` takes it again and a close ends it; on Windows the
opening of the transport in `internal/cli/device_windows.go` and the opening of the finder of `commandercore`. The
command line then words the refusals: `session.open` for the screen (R12, R13), `sensors` for the controller (R9),
and the keeper writes the line of R10 with no change of its own, because the error of the setup already has the text.

What the tree on `bugfix-0029-one-sensordeck-process-per-device` holds, and what the plan builds on:

* `internal/hidraw/transport.go` holds `OpenDiscovered(sysfsRoot, devRoot)`, `OpenPath(path)`, `openWith`,
  `openDiscovered`, `OpenNode(path)`, which the fakes' tests also use and which stays without a lock, and
  `openNode(path)`, which words the failure with `openError`. `Transport.Reopen` closes the node and runs the opening
  again; `Close` closes it. `internal/hidraw/restart.go` reopens through `Reopenable.Reopen`
* `internal/commandercore/node.go` holds `Finder{find; open}` and the interface `node{send; receive; close}`.
  `hidraw_linux.go` holds `Hidraw(sysfsRoot, devRoot)`, whose `open` calls `hidraw.OpenNode` and gives
  `cannot open <path>: ...`. `hid.go` holds `HID(seam)` and `hidNode{device}`; `finder_windows.go` holds `Windows()`
* `internal/commandercore/source.go` holds `Source{Finder; TimeLimit}` and `Read`, which gives nil for a missing
  controller and for a node that cannot be opened. `session.go` holds `Lighting.Start`, which gives the error of the
  opening as it is, and `Session`, which keeps its node open from the setup to the teardown
* `internal/cli/sources.go` holds `sourceSet{others; controller; monitor; startSession; controllerProblem}` and
  `commanderSources(finder)`; `sources_linux.go` and `sources_windows.go` hold `newSources(sysfsRoot, devRoot,
  procRoot)`
* `internal/cli/main.go` holds `environment.openTransport func(sysfsRoot, devRoot, device string)` and `run`, which
  writes `sensordeck: <error>` and gives 1. `internal/cli/commands.go` holds `session.open`, which the commands
  `clock`, `values`, `upload`, `listen`, `raw` and `run` (and `startRun` of a background run) call, and no other
  command; `sensorsCommand` and `listReadings`. `internal/cli/device_linux.go` and `device_windows.go` hold
  `openTransport`
* `internal/cli/keeper.go` writes `controller setup failed: <error>` once until a setup succeeds and schedules the
  retry at 10 seconds, doubling up to 5 minutes. `internal/cli/periodic.go` counts a reading that gave nothing as a
  reading, and `roundUnits` sends no value for a metric with no reading, so the screen keeps what it showed (R11)
* `internal/cli/background.go` writes `sensordeck: <error>` and `starting again in 10s` to the log after a failed start
* The e2e tests run the executable on pseudo terminals whose slave `/dev/pts/<n>` is behind the symbolic links of the
  fake tree. `flock` on a pseudo terminal works as on a hidraw node, and the lock is on the inode of `/dev/pts/<n>`,
  so a service of the developer that locks `/dev/hidraw<n>` does not touch it. `environmentFor(test, tree, "")` sets
  `SENSORDECK_PROC_ROOT` to the fake tree, which has no `locks`, so the holder of a refusal there is empty unless the
  test writes one

The design decisions the spec leaves to the plan:

* **The error.** `devicelock.InUseError{Holder string}`, whose `Error()` is
  `in use by another sensordeck process` followed by the holder text of R6. It names no device, and the callers put
  `the screen is ` or `the controller is ` before it. Every other failure to take a lock is a plain error of the form
  `cannot lock <path>: <cause>` (R5), or on Windows also `cannot create the lock file <path>: <cause>`
* **The holder text.** `devicelock.HolderText(pid int, commandLine string, commandLineKnown bool) string` gives
  ` (process <pid>: <command line>)`, or ` (process <pid>)` when the command line is not known or is empty. A holder
  that is not found is the empty string, which the callers pass on as it is
* **The lookup on Linux.** `devicelock.holderOfNode(procRoot string, major, minor uint32, inode uint64) string` reads
  `<procRoot>/locks`, and for the first line whose second field is `FLOCK` (a line whose second field is `->`, a
  waiter, is skipped) and whose sixth field is `<major>:<minor>:<inode>`, with major and minor in lower-case
  hexadecimal of at least two digits and the inode in decimal, takes the fifth field as the process ID. A process ID
  that is not a whole number above 0 gives no holder. The command line is `<procRoot>/<pid>/cmdline` split at the NUL
  bytes, the empty last part dropped, joined with one space. An unreadable or empty `cmdline` gives the process ID
  alone, and an unreadable `locks` or no matching line gives no holder. Device and inode come from the `Stat_t` of the
  opened file, with `unix.Major` and `unix.Minor` of its `Dev`. The format was checked in the sandbox:
  `30: FLOCK  ADVISORY  WRITE 496 00:85:627 0 EOF`
* **The lock on Linux.** `devicelock.LockNode(file *os.File, procRoot string) error` calls
  `unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)` inside `file.SyscallConn().Control`, never through `file.Fd()`, which
  would put the node into blocking mode and break the read deadlines of both transports. Its result goes through the
  pure `lockResult(err error, path string, holder func() string) error`: nil for nil, an `*InUseError` with
  `holder()` for `unix.EWOULDBLOCK`, and `cannot lock <path>: <err>` for any other error. The lock ends with the close
  of the node, so no unlock is written. Go opens the node close-on-exec, so a command of a custom metric inherits
  nothing (R4)
* **The lock on Windows.** `devicelock.Acquire(name string) (release func(), err error)` takes the directory
  `sensordeck` of `os.UserCacheDir()`, creates it with `os.MkdirAll`, and calls `acquireIn(directory, name)`, which
  opens `<directory>\<name>` with `os.OpenFile(O_RDWR|O_CREATE, 0o644)` and calls `windows.LockFileEx` with
  `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY` on one byte at the offset 2^62 (`OffsetHigh` `0x40000000`), far
  beyond the content, so that another process can still read the content while the lock is held (R6).
  `ERROR_LOCK_VIOLATION` reads the content and closes the file and gives an `*InUseError` with the holder of the
  content. Once held, the file is truncated and the content of the process is written at offset 0. `release` closes
  the file, which ends the lock; a second call does nothing
* **The content of the lock file.** `devicelock.encodeContent(pid int, commandLine string) []byte` gives
  `<pid>\n<command line>\n`, and `devicelock.holderOfContent(content []byte) string` gives the holder text of a
  content of exactly those two lines with a pid that is a whole number above 0, and the empty string for any other
  content. The command line written is `strings.Join(os.Args, " ")`. Both live in a file with no build constraint, so
  that their tests run on Linux
* **The screen on Windows.** `openTransport` of `device_windows.go` acquires `screen.lock` before it opens the screen,
  releases it when the opening fails, and gives the transport wrapped in `heldTransport{transport; release}`, defined
  in a file with no build constraint, whose `Close` closes the inner transport and then releases, once, and whose
  `Reopen` is the inner one, so that `run` holds the lock for its whole run while the screen is lost too (R14)
* **The controller on Windows.** `HID(seam winhid.Seam, acquire func() (func(), error))`. Its `open` acquires before
  `winhid.Open` and releases when the opening fails; `hidNode` gains `release func()`, called by `close` after the
  close of the device. `Windows()` passes `func() (func(), error) { return devicelock.Acquire("controller.lock") }`, and
  the tests pass a function that gives a release that does nothing
* **The controller's text.** `commandercore` words a refusal once, in `node.go`:
  `controllerOpenError(err error) error` gives `fmt.Errorf("the controller is %w", err)` for an error that holds an
  `*devicelock.InUseError`, and the error as it is otherwise. Both finders' `open` give their error through it, so the
  keeper's line is `controller setup failed: the controller is in use by another sensordeck process<holder>` (R10)
  with no change to the keeper
* **The refusal seen by `sensors`.** `commandercore.Source` gains `Refused func(err error)`, called, when it is not
  nil, with the error of an opening that holds an `*devicelock.InUseError`. `sourceSet` gains
  `reportController func(refused func(err error)) sensors.Source`, nil where nothing can be reported, which
  `commanderSources` sets to give the source with `Refused` set. `sensors` replaces `sources.controller` with it when it
  is not nil, as it adds the monitor, and keeps the error. `run` keeps `sources.controller`, with no `Refused`, so it
  writes no line (R11)
* **The refusal of the screen.** `session.open` turns an error that holds an `*devicelock.InUseError` into
  `the screen is <InUseError>, stop it first, or <stop text>`, where the stop text is
  `stop the service with systemctl --user stop sensordeck` and, when `env.system` is `windows`,
  `stop the startup task with sensordeck service uninstall`. `run` prefixes `sensordeck: ` as for every error, and the
  background run writes it to its log with the same prefix before `starting again in 10s` (R13)
* **The proc root reaches the openings.** `environment.openTransport` gains `procRoot` after `devRoot`, as
  `newSources` has it; `hidraw.OpenDiscovered(sysfsRoot, devRoot, procRoot)`, `hidraw.OpenPath(path, procRoot)` and
  `commandercore.Hidraw(sysfsRoot, devRoot, procRoot)` gain it too. `device_windows.go` ignores it, as it ignores the
  other roots

## Files

- `internal/devicelock/devicelock.go` (new, no constraint): the package comment, `InUseError`, `HolderText`
- `internal/devicelock/proclocks.go` (new, no constraint): `holderOfNode` and the parsing of `locks` and `cmdline`
- `internal/devicelock/content.go` (new, no constraint): `encodeContent`, `holderOfContent`
- `internal/devicelock/flock_linux.go` (new): `LockNode`, `lockResult`
- `internal/devicelock/lockfile_windows.go` (new): `Acquire`, `acquireIn`
- `internal/devicelock/devicelock_test.go`, `proclocks_test.go`, `content_test.go` (new), `flock_linux_test.go`
  (new), `lockfile_windows_test.go` (new, built by the vet of Windows and run by the human on Windows)
- `internal/hidraw/transport.go`, `internal/hidraw/transport_test.go`, `internal/fakescreen/fakescreen_test.go`: the
  lock in `openNode`, the proc root
- `internal/commandercore/node.go`, `hidraw_linux.go`, `hid.go`, `finder_windows.go`, `source.go`, and their tests
  `source_test.go`, `session_test.go`, `hid_test.go`, `hidraw_linux_test.go`
- `internal/cli/main.go`, `commands.go`, `sources.go`, `sources_linux.go`, `sources_windows.go`, `device_linux.go`,
  `device_windows.go`, `device_other.go`, `heldtransport.go` (new)
- `internal/cli/commands_test.go`, `sensors_test.go`, `keeper_test.go`, `background_test.go`, `run_test.go`,
  `heldtransport_test.go` (new)
- `e2e/lock_test.go` (new)
- `README.md`

## Tasks

### 1. The package devicelock

- [ ] Write `internal/devicelock/devicelock_test.go`: `HolderText(42, "sensordeck run --config c.yaml", true)` is
  ` (process 42: sensordeck run --config c.yaml)`, `HolderText(42, "", false)` and `HolderText(42, "", true)` are
  ` (process 42)`; `(&InUseError{Holder: " (process 42)"}).Error()` is
  `in use by another sensordeck process (process 42)`, and with an empty holder
  `in use by another sensordeck process` (R6)
- [ ] Write `internal/devicelock/proclocks_test.go` with a fake proc root in a temporary directory: a `locks` with a
  `POSIX` line, a `->` line and a `FLOCK` line for `00:85:627` with pid 496, and `496/cmdline` of
  `sensordeck\x00run\x00--config\x00c.yaml\x00`, gives ` (process 496: sensordeck run --config c.yaml)` for
  major 0, minor 0x85, inode 627; the same without `496/cmdline` gives ` (process 496)`; an empty `cmdline` gives
  ` (process 496)`; another inode, another device, a missing `locks` and a pid field of `-1` give `` (R6, R20)
- [ ] Write `internal/devicelock/content_test.go`: `holderOfContent(encodeContent(42, "sensordeck run"))` is
  ` (process 42: sensordeck run)`; an empty content, `x\nsensordeck\n`, `0\nsensordeck\n`, a content of one line and
  a content of three lines give `` (R6, R20)
- [ ] Write `internal/devicelock/flock_linux_test.go`: two openings of one temporary file, `LockNode` of the first
  gives nil and of the second an `*InUseError`; after the close of the first, `LockNode` of the second gives nil
  (R1, R3); an opening through a symbolic link to the file is refused while the other holds it (R1); with `/proc` as
  the proc root and `/proc/locks` readable, the holder of the refusal is the process of the test, its pid and its
  command line, and the test is skipped when `/proc/locks` cannot be read (R6); `lockResult` gives nil for nil, an
  `*InUseError` whose holder is what the function gave for `unix.EWOULDBLOCK`, and `cannot lock /dev/x: bad file
  descriptor` for `unix.EBADF` (R5)
- [ ] Write `internal/devicelock/lockfile_windows_test.go`: `acquireIn` of a temporary directory and `screen.lock`
  succeeds and the file holds the content of the process; a second `acquireIn` of the same file in the same process
  gives an `*InUseError` whose holder names the process; the content stays readable with `os.ReadFile` while held;
  after `release`, `acquireIn` succeeds again; a second `release` does nothing; a directory that is a file gives an
  error that names the path (R2, R3, R5, R6). It is compiled by `GOOS=windows GOARCH=amd64 go vet ./...` and run by
  the human on Windows
- [ ] Write `devicelock.go`, `proclocks.go`, `content.go`, `flock_linux.go` and `lockfile_windows.go` as the approach
  says, until the tests pass and both builds and vets of the suite succeed (R1 to R6)
- [ ] Commit: `bugfix-0029 Add the advisory locks of the devices and the naming of their holder`

### 2. The locks on Linux

- [ ] In `internal/hidraw/transport_test.go`, test with the fake screen: while a transport of `OpenPath(node,
  procRoot)` is open, a second `OpenPath` of the node, and an `OpenPath` of the `/dev/pts/<n>` the link resolves to,
  give an `*devicelock.InUseError` (the test checks it with `errors.As`) and leave the fake screen with no report;
  after the first `Close`, the second opening succeeds (R1, R3); `OpenDiscovered` is refused the same way; a
  transport whose `Reopen` finds the node held by another transport gives the `*InUseError`, holds no node, and a later
  `Reopen` after the other's `Close` succeeds (R14)
- [ ] Change `openNode` to take the proc root and call `devicelock.LockNode` after `OpenNode`, closing the file and
  giving the error of the lock as it is when it fails; pass the proc root through `OpenDiscovered`, `OpenPath` and
  `openDiscovered`, and update `internal/fakescreen/fakescreen_test.go` and the other callers. `OpenNode` stays
  without a lock (R1, R3, R5)
- [ ] In `internal/commandercore/source_test.go`, with the fake controller and a file of the test that holds the
  `flock` of `hidraw9`: `Source.Read` gives no reading, writes nothing to the controller, and calls `Refused` once
  with an error whose text is `the controller is in use by another sensordeck process`; with `Refused` nil it does the
  same and calls nothing (R7, R9, R11); with `OnStep(3, ...)` trying the lock of `hidraw9` from the test, the try is
  refused, so the reading holds the lock while it runs, and after the reading the test takes it (R7)
- [ ] In `internal/commandercore/session_test.go`: `Lighting.Start` while the test holds the lock gives an error that
  holds an `*devicelock.InUseError`, whose text is `the controller is in use by another sensordeck process`, and writes
  nothing; while a session is held the test is refused the lock, after a failed keepalive, a setup that failed and
  `End` it is not (R8, R10)
- [ ] Add `controllerOpenError` to `node.go`; give `Hidraw` the proc root, and call `devicelock.LockNode` in its
  `open` after `hidraw.OpenNode`, closing the file on a failure and giving its error through `controllerOpenError`;
  add `Refused` to `Source` and call it in `Read` when the error of the opening holds an `*InUseError` (R1, R3, R7,
  R8, R9)
- [ ] Add `procRoot` to `environment.openTransport` and to `openTransport` in `device_linux.go`, `device_windows.go`
  and `device_other.go`, pass `current.env.procRoot` in `session.open`, pass it to `commandercore.Hidraw` in
  `sources_linux.go`, and update the harness of `internal/cli/commands_test.go`, which checks that the proc root
  arrives as `/procroot`
- [ ] Run the suite, and commit: `bugfix-0029 Lock the nodes of the screen and the controller on Linux`

### 3. The locks on Windows

- [ ] Write `internal/cli/heldtransport_test.go`: `Close` of a `heldTransport` closes the inner transport and then
  calls the release once, a second `Close` calls neither again, `Reopen` calls the inner one and not the release, and
  `Write` and `Read` reach the inner transport (R2, R14)
- [ ] Write `internal/cli/heldtransport.go`, and in `device_windows.go` acquire `screen.lock` before the opening,
  release it when the opening fails, and give the `heldTransport` (R2, R14)
- [ ] In `internal/commandercore/hid_test.go`: `HID` with an acquirer that counts gives a node whose `close` releases
  once after the close of the device; an acquirer that gives an `*InUseError` makes `Source.Read` call `Refused` and
  open nothing on the fake seam, and makes `Lighting.Start` give the text of R10; an opening that fails after the
  acquire releases (R2, R3, R8, R9)
- [ ] Give `HID` the acquirer and `hidNode` the release, and make `Windows()` acquire `controller.lock` (R2)
- [ ] Run the suite, and commit: `bugfix-0029 Lock the screen and the controller on Windows`

### 4. The lines of the commands

- [ ] In `internal/cli/commands_test.go`, a table over `clock`, `values 1=2`, `upload <file>`, `listen`, `raw 01`,
  `run --config <file>` with `openErr` an `*devicelock.InUseError{Holder: " (process 42: sensordeck run)"}`, for
  `system` `linux` and `windows`: status 1, nothing on standard output, standard error exactly the line of R12 with
  ` (process 42: sensordeck run)` and the stop text of the system, and no write to the transport. For `run`, also no
  `running:`, no call to `startSession` and no reading of the controller (R12, R13, R20). An `openErr` that is not
  an `*InUseError` keeps its text as now
- [ ] In `internal/cli/background_test.go`: a background run whose opening is refused writes
  `sensordeck: the screen is in use by another sensordeck process (process 42: sensordeck run), stop it first, or stop
  the startup task with sensordeck service uninstall` and `starting again in 10s` to the log at each refused start,
  and runs once the opening succeeds (R13)
- [ ] Write the mapping in `session.open` (R12, R13)
- [ ] In `internal/cli/sensors_test.go`, with a set whose `reportController` gives a source that calls `refused` with
  `fmt.Errorf("the controller is %w", &devicelock.InUseError{Holder: " (process 42: sensordeck run)"})` and gives no
  reading: standard error is exactly
  `the controller is in use by another sensordeck process (process 42: sensordeck run), its probes are left out` once,
  standard output lists the other readings and the custom metrics as now, and the status is 0; with no other reading
  it writes that line and then `no sensor could be read` (R9, R20)
- [ ] Add `reportController` to `sourceSet` and set it in `commanderSources`; in `sensorsCommand` use it when it is
  not nil, keep the refusal, and write its line after the line of the monitor and before `listReadings` (R9)
- [ ] In `internal/cli/keeper_test.go`, with a start that gives the error of a refused controller with the holder
  ` (process 42: sensordeck run)`: the line `controller setup failed: the controller is in use by another sensordeck
  process (process 42: sensordeck run)` is written once over three refused setups, the setups come 10, 20 and 40
  seconds apart, and a setup that then succeeds holds the session (R10, R20)
- [ ] In `internal/cli/run_test.go`, check that a `run` without `commander_color` whose controller source gives no
  reading writes nothing to standard error and reads it again only after `commander_interval`; reuse the existing test
  of a reading that failed if it already checks both, and add the assertion it lacks otherwise (R11)
- [ ] In `internal/cli/commands_test.go`, `devices` and `sensors` succeed with `openErr` an `*InUseError` and call
  `openTransport` not at all; check that `session.open` is called only by `clock`, `values`, `upload`, `listen`,
  `raw`, `run` and `startRun`, so that the `theme` and `service` commands take no lock of the screen (R15)
- [ ] Run the suite, and commit: `bugfix-0029 Refuse the commands while another sensordeck process has the device`

### 5. The end-to-end tests

- [ ] Write `e2e/lock_test.go`, every test with the environment of `environmentFor`:
    - `sensors` while `run` with `commander_color` holds a session, started after `waitForColourWrite`: status 0,
      standard output holds `cpu.temp\t42.0\t°C` and no line beginning `commander.`, standard error is exactly the line
      of R9 with no holder, and afterwards the fake controller is in software mode with the colour of the config, and
      its writes hold one write of the entry into software mode and no return to hardware mode (R7, R8, R9, R19)
    - a second `run` while the first has sent a value report, with `<ProcRoot>/locks` written by the test with a
      `FLOCK` line for the first run's pid and the device and inode of the screen's node, as `os.Stat` gives them,
      and `<ProcRoot>/<pid>/cmdline` with its arguments: status 1, standard output without `running:`, standard error
      exactly `sensordeck: the screen is in use by another sensordeck process (process <pid>: <arguments>), stop it
      first, or stop the service with systemctl --user stop sensordeck`, and the first run sends a value report after
      the refusal (R1, R6, R12, R13, R19)
    - `values 7=1234` while `run`, which maps no tag 7, runs: status 1, the line of R12 with no holder, and no report
      of the screen holds the tag 7 (R12, R19)
    - after a `run` with `commander_color` that ended on SIGTERM, and after one killed with SIGKILL while it held a
      session, `values 7=1234` succeeds and `sensors` lists `commander.1.temp` (R4, R19)
    - after a `run` with a custom metric whose command starts `<sleep> 5 >/dev/null 2>&1 &` with the absolute path of
      `sleep` found with `exec.LookPath` when the test starts, and that ended on SIGTERM, `values 7=1234` succeeds
      while the child still runs (R4)
- [ ] Run the suite; the e2e tests of earlier units run one process at a time and pass unchanged. Commit:
  `bugfix-0029 Test two sensordeck processes on the fake devices`

### 6. The README

- [ ] In "The Corsair iCUE Commander Core XT", replace the paragraph that begins `Another program that talks to the
  controller` with two: one sensordeck process at a time uses the controller; `sensordeck sensors` leaves out the
  probes while `run` holds a session or reads, writing its line, quoted; and a `run` whose setup finds the controller
  in use writes `controller setup failed` and tries again as after any failed setup. Then that iCUE, liquidctl and
  OpenLinkHub are not kept out, and talking to the controller from one of them at the same time as sensordeck is not
  supported: the commands of both mix, readings fail and the LEDs may show the stored lighting or go dark (R16)
- [ ] In the same section, after `sensordeck sensors` makes them flicker once each time it runs, and in "Showing
  sensor values on the screen", after `sensordeck sensors` reads the controller once each time it runs, say that this
  is so unless `run` has the controller, when `sensors` leaves the probes out (R17)
- [ ] In "Running as a service", replace the paragraph of Linux that begins `Running \`sensordeck run\`, or another
  command that writes to the screen, by hand while the service runs` and the paragraph of Windows that begins the same
  way for the task, each with: such a command refuses while the service (or the task) has the screen, with the line of
  R12 of that system, quoted with `<holder>` written as the process ID and command line; and a service (or a task)
  that starts while another sensordeck process has the screen refuses and starts again 10 seconds later (R17)
- [ ] In "Running as a service", after its first paragraph, add one that says that on Linux the lock is on the device
  node, so it holds between every sensordeck process that opens the device, whatever its user or environment, and that
  on Windows it is one lock for the screens and one for the controllers, for the processes of one user, so that two
  sensordeck processes cannot drive two screens at once on Windows (R18)
- [ ] Commit: `bugfix-0029 Document one sensordeck process at a time on each device`

### 7. Checks

- [ ] Run the five commands of the suite in `AGENTS.md` and `npm run check-changes`, and fix what they report (R21)
- [ ] Check that every requirement R1 to R21 has its task ticked, and that the checks of D10 on the device and on
  Windows are left to the human
