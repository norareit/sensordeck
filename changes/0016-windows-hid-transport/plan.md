# Plan: the HID transport on Windows

## Approach

The work comes in two halves, committed apart as D10 and the standards ask. The first half is refactoring that
changes no behaviour: the text of the screen lines and of the two errors of the discovery, which the Windows transport
would otherwise repeat, moves into a package of its own, and the Commander package is split into its protocol and a
node per system. The second half is the Windows code: a new package of the Windows HID layer with its seam, a fake of
the seam, the Commander on Windows, the wiring of the command line, the CI job and the documents.

What the tree on `feature-0016-windows-hid-transport` holds, and what the plan builds on:

* `internal/hidraw` is limited to Linux. `Screen.Line` writes `<node>\t%04x:%04x\t<name>\t<serial>` with `-` for an
  empty name or serial, `openDiscovered` fails with `no supported screen found` or with
  `more than one supported screen is connected, name one with --device:\n<lines>`, and `internal/cli/commands.go`
  holds `noScreenMessage` with the same first text
* `internal/cli/device_linux.go` and `internal/cli/device_other.go` (`!linux`) choose the transport, and
  `sources_linux.go` and `sources_other.go` (`!linux`) the sources. `unsupported()` is in `device_other.go`. The
  `environment` of `internal/cli/main.go` holds `listScreens`, `openTransport` and `newSources` as functions, and
  `harness.run` in `internal/cli/commands_test.go` fills them with fakes
* `internal/commandercore` is limited to Linux in every file. `findNode` (in `discover.go`) uses `hidraw.List`;
  `Source.Read` and `Lighting.Start` call `findNode` and `hidraw.OpenNode`; `exchange(file *os.File, ...)` writes
  `encode(command)` (97 bytes), sets the read deadline and reads into 384 bytes, and words its failures
  `write failed: %v`, `short write of %d bytes`, `no answer within %s` and `read failed: %v`. `Start` words a failed
  opening `cannot open %s: %w`. `Session` holds an `*os.File`, and `teardown` closes it
* The tests of the Commander package are all limited to Linux. `source_test.go` and `session_test.go` use
  `internal/fakecommander` and build `Source` and `Lighting` from `SysfsRoot` and `DevRoot`; `source_test.go` calls
  `runSequence` with a file of `hidraw.OpenNode`. `protocol_test.go` uses nothing of Linux. `discover_test.go` tests
  `findNode` against a sysfs tree
* `layout_test.go` holds `TestLinuxCodeIsLimitedToLinux`, whose list names `internal/commandercore` as a whole, and
  `TestOnlyLinuxFilesOfTheCommandsNameTheLinuxCode`, which forbids every file of `internal/cli` in a Windows build to
  import the Commander package
* `release_test.go` holds `TestTheCIWorkflowRunsTheSuite`, which requires exactly one job, on `ubuntu-24.04`, running
  `suiteCommands`, with exact action versions by `checkExactActions`. `TestTheReleaseWorkflowRunsTheSuiteBeforeGoreleaser`
  checks the call of the CI workflow and the permissions, and needs no change (R38)
* `golang.org/x/sys v0.50.0` is a direct requirement. In it `windows.Errno` is an alias of `syscall.Errno`,
  `ERROR_ACCESS_DENIED` is 5, `ERROR_SHARING_VIOLATION` 32 and `ERROR_OPERATION_ABORTED` 995, and
  `CM_Get_Device_Interface_List(deviceID string, interfaceClass *GUID, flags uint32) ([]string, error)`,
  `CM_GET_DEVICE_INTERFACE_LIST_PRESENT`, `CreateFile`, `CreateEvent`, `ReadFile`, `WriteFile`, `WaitForSingleObject`,
  `CancelIoEx`, `GetOverlappedResult`, `CloseHandle` and `NewLazySystemDLL` are bound. `LazyProc.Call` panics on a
  procedure that cannot be found, while `LazyProc.Find` gives an error that names the procedure and the DLL
* `smartmonitor.ErrReadTimeout` is `no report arrived within the time limit`

### Choices the spec leaves to the plan

* **The shared screen texts** go into a new package `internal/screenlist`, with no build constraint and no I/O:
  `Line(path string, vendor, product uint16, name, serial string) string`, `const NoneFound = "no supported screen
  found"`, and `Several(lines []string) error`, which gives the error of several screens. `hidraw.Screen.Line`,
  `hidraw.openDiscovered` and `noScreenMessage` of the command line use them. The texts are unchanged
* **The node of the Commander.** The protocol runs on an unexported interface `node` with three methods:
  `send(command []byte) error`, which writes the command as one output report and gives an error whose text is the
  whole problem; `receive(timeLimit time.Duration) ([]byte, error)`, which gives the bytes of one input report after the
  report ID, `errNoAnswer` when the time limit passed, or an error whose text is the whole problem; and `close() error`.
  `exchange(node, command, timeLimit)` makes `no answer within %s` of `errNoAnswer` and takes the text of every other
  error as the problem. The Linux node is `hidrawNode{file *os.File}`: `send` writes `encode(command)` and words
  `write failed: %v` and `short write of %d bytes`; `receive` sets the deadline, reads into 384 bytes, gives
  `errNoAnswer` for `os.ErrDeadlineExceeded` and words every other failure `read failed: %v`. Every text and every byte
  of Linux stays as it is
* **The finding of interface 0** is an exported struct `Finder` with two unexported fields, `find func() (string,
  error)` and `open func(path string) (node, error)`. `find` gives the path, `ErrNoController`, or another error that
  says why nothing was found (R34). `open` gives a node or an error worded `cannot open <path>: ...`. `Source` and
  `Lighting` lose `SysfsRoot` and `DevRoot` and gain `Finder Finder`. `Hidraw(sysfsRoot, devRoot string) Finder`, in a
  file limited to Linux, finds with `findNode` and opens with `hidraw.OpenNode`, wording the failure `cannot open %s:
  %w` as `Start` does today. `HID(seam winhid.Seam) Finder`, in a file with no build constraint, is the Windows one,
  and `Windows() Finder`, in a file limited to Windows, gives `HID(winhid.Windows())`. `Finder.Problem() error` gives
  the error of `find`, and nil when it found a path or gave `ErrNoController`; `sensors` writes it (R26, R34)
* **The Windows HID layer** is a new package `internal/winhid`. Its seam is the interface `Seam` in `seam.go`, with no
  build constraint, and its real implementation, `Windows() Seam`, is in `seam_windows.go`, limited to Windows, the
  only file of the module that imports `golang.org/x/sys/windows` besides its test. The seam holds, by R3:
    * `Paths() ([]string, error)`
    * `OpenQuery(path string) (Handle, error)` and `OpenReadWrite(path string) (Handle, error)`, where `Handle` is a
      `uintptr` type of the package
    * `Attributes(Handle) (vendor, product uint16, err error)`, `Product(Handle) (string, error)`, `Serial(Handle)
      (string, error)` and `Caps(Handle) (Caps, error)`, where `Caps` holds `UsagePage`, `Usage`, `InputLength` and
      `OutputLength`, all `uint16`
    * `StartRead(Handle, []byte) (Operation, error)` and `StartWrite(Handle, []byte) (Operation, error)`. An error
      means that nothing is pending. `Operation` has `Wait(milliseconds uint32) bool`, true when it completed within
      the wait and false when the wait ended without completion or failed; `Cancel()`; and `Reap() (count uint32, code
      syscall.Errno)`, which waits for the completion, gives its code, 0 for success, and frees what the operation held
    * `Close(Handle) error`
* **The error codes** that the constraint-free code names are constants of `syscall.Errno` in `winhid`:
  `errorAccessDenied = 5`, `errorSharingViolation = 32` and `errorOperationAborted = 995`. The test limited to Windows
  compares them with the constants of `golang.org/x/sys/windows`. The text of an error is `syscall.Errno`'s, which is
  the text of the system on Windows, so a test on Linux compares with `fmt.Sprint(syscall.Errno(code))` and not with a
  text of Windows
* **The completion of an operation** is classified by the pure `complete(operation direction, length int, result
  result) (int, error)`, where `result` holds `inTime`, `cancelled`, `code` and `count`. A start that failed is a
  result in time, not cancelled, with the code of the failure. `run(seam, operation, handle, buffer, limit)` starts,
  waits `milliseconds(limit)`, cancels when the wait gave false, always reaps, and then classifies. `milliseconds`
  rounds up to a whole millisecond, gives 0 for a limit of zero or less, and at most `0xFFFFFFFE`, so that it never
  asks for the infinite wait `0xFFFFFFFF`
* **The time limit of a write** is `winhid.WriteTimeLimit = 5 * time.Second`, and the text of R19 is made from it.
  The timeout of a read is `fmt.Errorf("%w after %s", smartmonitor.ErrReadTimeout, limit)`
* **An open collection** is `winhid.Device`, made by `winhid.Open(seam, path) (*Device, error)`: it opens for reading
  and writing, words a failure by R14, reads the caps and the serial string of the handle, and closes the handle when
  the caps cannot be read. `WriteReport(payload []byte, limit time.Duration) error` writes a buffer of the output
  length: `0x00`, the payload, zeros. `ReadReport(limit time.Duration) ([]byte, error)` reads a buffer of the input
  length and gives the bytes after the report ID, or `io.EOF` for a read of 0 bytes. `Caps()`, `Serial()` and an
  idempotent `Close()` complete it. The screen's `Transport` and the Commander's node on Windows both use it
* **The near misses of the screen** come out of `winhid` as lines. `listScreens` of the `environment` changes to
  `func(sysfsRoot, devRoot string) ([]string, []string, error)`, the screens and the near misses; the Linux one gives
  no near misses, and `devices` writes them to standard error after `no supported screen found` when there is no
  screen. `sourceSet` gains `controllerProblem func() error`, which the command line fills from `Finder.Problem` on
  both systems; on Linux it is always nil, because the Linux `find` gives no other error
* **The fake of the seam** is a new package `internal/fakehid`, with no build constraint, so that the tests of
  `winhid` and of the Commander package use it on Linux and in the job of R36. Since it imports `winhid`, the tests of
  `winhid` that use it are in the package `winhid_test`, and the tests of unexported pure functions in `winhid`
* **The commit of the shared command line code.** The type `commanderSession` and the making of the controller source
  and of `startSession` from a `Finder` move into `internal/cli/sources.go` as `commanderSources(finder
  commandercore.Finder) sourceSet`, so that `sources_linux.go` and `sources_windows.go` each only choose the finder and
  the other sources

## Files

- `internal/screenlist/screenlist.go`, `internal/screenlist/screenlist_test.go`: new, the texts of the screen lines
  and of the two errors of the discovery
- `internal/hidraw/discover.go`, `internal/hidraw/transport.go`: use `internal/screenlist`
- `internal/commandercore/protocol.go`, `source.go`, `session.go`: no build constraint, the protocol on a `node`, and
  `Finder`; `vendor` and `product` move here from `discover.go`
- `internal/commandercore/discover.go`: renamed `hidraw_linux.go`, with `findNode`, `hidrawNode` and `Hidraw`
- `internal/commandercore/hid.go`, `hid_test.go`: new, no build constraint, the Windows finding, near misses and node
- `internal/commandercore/finder_windows.go`: new, `Windows() Finder`
- `internal/commandercore/protocol_test.go`: the build constraint is removed
- `internal/commandercore/source_test.go`, `session_test.go`, `discover_test.go`: keep `linux`, build `Source` and
  `Lighting` with `Hidraw(...)`, and wrap the opened file in `hidrawNode`. `discover_test.go` is renamed
  `hidraw_linux_test.go`
- `internal/winhid/seam.go`, `collections.go`, `operation.go`, `device.go`, `transport.go`, `errors.go`: new, no
  build constraint
- `internal/winhid/seam_windows.go`, `seam_windows_test.go`: new, limited to Windows
- `internal/winhid/paths.go`: new, no build constraint
- `internal/winhid/operation_test.go`, `errors_test.go`, `paths_test.go`: new, package `winhid`
- `internal/winhid/collections_test.go`, `device_test.go`, `transport_test.go`: new, package `winhid_test`
- `internal/fakehid/fakehid.go`, `fakehid_test.go`: new, the fake of the seam
- `internal/cli/main.go`, `commands.go`, `commands_test.go`, `sources.go`, `sources_linux.go`, `device_linux.go`,
  `sensors_test.go`: the changed `listScreens`, `controllerProblem`, `commanderSources`, the near misses in `devices`
  and the problem in `sensors`
- `internal/cli/device_windows.go`, `sources_windows.go`: new, limited to Windows by their names
- `internal/cli/device_other.go`, `sources_other.go`: the constraint becomes `!linux && !windows`
- `layout_test.go`: the Commander package is no longer Linux-only as a whole; new checks of the seam and of `hid.dll`
- `.github/workflows/ci.yml`, `release_test.go`: the job on `windows-2025`
- `README.md`, `docs/protocol.md`, `docs/commander-core-xt.md`: the documents

## Tasks

### 1. The shared texts of the screen discovery (refactoring)

- [x] Add `internal/screenlist/screenlist_test.go` with the tests of `Line` (the format of the Linux line, `-` for an
  empty name and an empty serial, the IDs in four lower-case digits), of `NoneFound`, and of `Several` with two lines,
  whose text is `more than one supported screen is connected, name one with --device:\n<line>\n<line>` (R10, R11)
- [x] Add `internal/screenlist/screenlist.go` with `Line`, `NoneFound` and `Several`, no build constraint (R10, R11)
- [x] Make `hidraw.Screen.Line` call `screenlist.Line`, `openDiscovered` use `screenlist.NoneFound` and
  `screenlist.Several`, and `noScreenMessage` of `internal/cli/commands.go` be `screenlist.NoneFound`. Remove
  `orDash` from `hidraw`. The tests of `hidraw` and of the command line pass unchanged (R4)
- [x] Run the suite and commit: `feature-0016 Share the texts of the screen discovery`

### 2. The Commander package split into its protocol and a node (refactoring)

- [x] In `internal/commandercore`, add the interface `node`, `errNoAnswer` and `Finder` as the choices above say, and
  change `exchange`, `runSequence`, `runSteps` and `Session` to take a `node` in place of the `*os.File`.
  `teardown` closes the node. `Source.Read` calls `finder.find` and `finder.open` and gives no probes on either error;
  `Lighting.Start` gives the error of `find` as it is, `ErrNoController` included, and the error of `open` as it is
  (R24)
- [x] Move `findNode` into `hidraw_linux.go` (renamed from `discover.go`), with `hidrawNode` and `Hidraw`, which keep
  every byte and every text of Linux as the choice above says. Move `vendor` and `product` into `protocol.go`. Remove
  the build constraint from `protocol.go`, `source.go` and `session.go` (R24, D10)
- [x] In the tests limited to Linux, build `Source` and `Lighting` with `Finder: Hidraw(sysfsRoot, devRoot)`, and in
  `TestAStopDoneAtTheStartOfTheSequenceWritesNothing` pass `hidrawNode{file: file}`. Rename `discover_test.go` to
  `hidraw_linux_test.go`. Remove the build constraint of `protocol_test.go`, which uses nothing of Linux. Nothing else
  of the tests changes (R24, R37)
- [x] Move `commanderSession` and the making of `controller` and `startSession` out of `sources_linux.go` into
  `commanderSources(finder commandercore.Finder) sourceSet` in `internal/cli/sources.go`, and make `newSources` of
  Linux call it with `commandercore.Hidraw(sysfsRoot, devRoot)` and add the Linux sources (R24)
- [x] In `layout_test.go`, take `internal/commandercore` out of the list of `TestLinuxCodeIsLimitedToLinux` and add a
  test that every file of the module that imports `internal/hidraw` or `internal/linuxsensors` is excluded from a
  Windows build. In `TestOnlyLinuxFilesOfTheCommandsNameTheLinuxCode`, drop the Commander package from the forbidden
  imports (R3, R24)
- [x] Run the suite, check that the end-to-end tests pass unchanged, and commit:
  `feature-0016 Split the Commander package into its protocol and a node`

### 3. The fake of the seam and the pure parts of the Windows layer

- [x] Add `internal/winhid/seam.go` with `Seam`, `Handle`, `Caps` and `Operation`, as the choices above say (R3)
- [x] Add `internal/fakehid/fakehid.go`: a `Seam` built from a list of collections, each with its path, IDs, product
  and serial strings, caps, an error for each of the opening with zero access, the attributes, the caps, the strings
  and the opening for reading and writing, and a function that makes the input report that follows a write. It
  records every opening with its kind, every closing, every wait with its milliseconds, every cancel, every start and
  reap, and whether an operation is pending. A test sets how an operation ends: completed in time, completed only after
  the wait, completed only at the reap after a cancel with a code and a count, or failed at the start with a code. A
  method tells whether any operation is pending and whether every handle opened is closed. Add
  `internal/fakehid/fakehid_test.go` for what the fake records (R31)
- [x] Add `internal/winhid/paths_test.go`, with each path of R9 and its expected answer, and
  `internal/winhid/paths.go` with `InterfaceZero(path string) bool`: the part between the first and the second `#`,
  split on `&`, holds a field equal to `mi_00` without regard to case (R9, R31)
- [x] Add `internal/winhid/operation_test.go`, a table over `complete` and `milliseconds`: a read and a write in time;
  a read in time of fewer bytes; a short write in time, `short write: <count> of <length> bytes`; a read completed with
  data after the wait ended (the race), which gives its bytes; a cancelled read reaped with `errorOperationAborted`,
  for which `errors.Is` with `smartmonitor.ErrReadTimeout` is true; an `errorOperationAborted` of a read not
  cancelled, `read failed: <text>` and not the timeout; a cancelled write reaped with all its bytes, a success; a
  cancelled write reaped with fewer bytes or a code, `write timed out after 5s`, not the timeout of a read; a start
  that failed; another code, worded `read failed: <text>` and `write failed: <text>`, with the code reachable with
  `errors.As`; and the rounding of 0, 1 ns, 1 ms, 1.5 ms, a negative limit and a limit beyond `0xFFFFFFFE` ms (R18,
  R19, R21, R31)
- [x] Add `internal/winhid/operation.go` with `complete`, `milliseconds` and `run`. The cases of `run` against
  `fakehid` are tested through `Device` in `device_test.go` below, because `operation_test.go` is in the package
  `winhid` and cannot import the fake (R18, R19, R20)
- [x] Add `internal/winhid/errors_test.go`, with the wording of R14 for `errorAccessDenied`, `errorSharingViolation`
  and another code, and the code reachable with `errors.As` for each, and `internal/winhid/errors.go` with the opening
  error, its exact text and `Unwrap` to the `syscall.Errno`, on the pattern of `hidraw.openErrorValue` (R14, R31)
- [x] Add `internal/winhid/collections_test.go` (`winhid_test`): `List` orders by path without regard to case, leaves
  out a path that fails to open, to give its attributes or its caps, keeps a collection whose strings fail with empty
  strings, opens every path with zero access only, and closes every handle in every case; `Screens` keeps the
  collections of `0483:0065` with 65 and 65, gives the others of `0483:0065` as near misses, and ignores other devices;
  `NearMissLines` gives the lines of R33 in the order of R6 (R6, R7, R32, R33, R31)
- [x] Add `internal/winhid/collections.go` with `Collection` (path, IDs, name, serial, caps), `List(seam Seam)
  ([]Collection, error)`, `Screens(collections []Collection) (screens, nearMisses []Collection)`, `Collection.Line()`
  by `screenlist.Line`, and `NearMissLines`. The length of the screen is `1 + len(smartmonitor.Report{})` and is
  written nowhere as 65 (R6, R7, R10, R32, R33)
- [x] Add `internal/winhid/device_test.go` (`winhid_test`): `Open` opens for reading and writing only the path given,
  reads its caps and serial, words a failed opening by R14, and closes the handle when the caps fail; `WriteReport`
  writes a buffer of the output length with `0x00`, the payload and zeros, with `WriteTimeLimit`; `ReadReport` reads a
  buffer of the input length, drops the first byte, gives `io.EOF` for 0 bytes, and gives the bytes of a short read;
  `run` cancels and reaps when the wait ends without completion, reaps without cancelling when the operation completes
  in time, waits the time limit rounded up, and starts nothing more after a failed start; and after each call no
  operation is pending (R15, R16, R18, R19, R20, R31)
- [x] Add `internal/winhid/device.go` with `Device`, `Open`, `WriteReport`, `ReadReport`, `Caps`, `Serial` and
  `Close`, keeping the buffer of each operation referenced by the call until `Reap` returned (R15, R16, R20)
- [x] Add `internal/winhid/transport_test.go` (`winhid_test`): `OpenDiscovered` with no screen, `no supported screen
  found`, with no screen and two near misses, the same followed by their lines, with a near miss beside a screen, no
  line, with one screen, opened, with two, the error of `screenlist.Several`; `OpenPath` with lengths of 65 and with
  lengths of 64 and 65, `cannot open <path>: its reports are 64 and 65 bytes, those of the screen are 65`, and the
  handle closed; `Serial` from the collection opened last, and empty when it has none; `Write` frames by R15 and
  `Read` by R16, filling up to 64 bytes; `Reopen` by the discovery anew and by the same path, a failed `Reopen` that
  leaves no handle, `Write` and `Read` then failing with an error that is not the timeout, a later `Reopen` that
  succeeds; `Close` twice; and nothing pending after any call (R10, R11, R12, R13, R15, R16, R20, R22, R33, R31)
- [x] Add `internal/winhid/transport.go` with `Transport` (`OpenDiscovered(seam)`, `OpenPath(seam, path)`, `Write`,
  `Read`, `Reopen`, `Serial`, `Close`), which implements the `transport` of the command line, on the pattern of
  `hidraw.Transport` (R11, R12, R13, R15, R16, R22, R23)
- [x] Run the suite and commit: `feature-0016 Add the Windows HID layer on a seam, with its fake`

### 4. The seam on Windows

- [ ] Add `internal/winhid/seam_windows_test.go`, limited to Windows: every procedure of `hid.dll` that the seam uses
  resolves with `Find`, and `List(Windows())` gives no error and a non-empty path for every collection; the three code
  constants equal those of `golang.org/x/sys/windows`; and `errors.Is` holds with `fs.ErrPermission` for the opening
  error of `ERROR_ACCESS_DENIED` and with `fs.ErrNotExist` for those of `ERROR_FILE_NOT_FOUND` and
  `ERROR_PATH_NOT_FOUND` (R2, R14, R35)
- [ ] Add `internal/winhid/seam_windows.go`: `hid.dll` by `windows.NewLazySystemDLL`, with `HidD_GetHidGuid`,
  `HidD_GetAttributes`, `HidD_GetProductString`, `HidD_GetSerialNumberString`, `HidD_GetPreparsedData`,
  `HidD_FreePreparsedData` and `HidP_GetCaps`, each called only after its `Find` succeeded, whose error is given as it
  is. `Paths` calls `HidD_GetHidGuid` and `windows.CM_Get_Device_Interface_List("", &guid,
  windows.CM_GET_DEVICE_INTERFACE_LIST_PRESENT)`. `OpenQuery` and `OpenReadWrite` call `windows.CreateFile` with an
  access of zero or `GENERIC_READ|GENERIC_WRITE`, `FILE_SHARE_READ|FILE_SHARE_WRITE`, `OPEN_EXISTING`, and
  `FILE_FLAG_OVERLAPPED` for reading and writing. `Attributes` lays out `HIDD_ATTRIBUTES` (`Size uint32`, then
  `VendorID`, `ProductID` and `VersionNumber` as `uint16`) with its size set; `Caps` lays out `HIDP_CAPS` (`Usage`,
  `UsagePage`, `InputReportByteLength`, `OutputReportByteLength`, `FeatureReportByteLength`, `Reserved [17]uint16`,
  then ten `uint16` counts), takes `HIDP_STATUS_SUCCESS` (`0x00110000`) as success, and frees the preparsed data on
  every path after it was got. The strings are read into a buffer of 256 `uint16` and cut at the first zero. An
  operation holds its buffer, a heap `windows.Overlapped` and a manual-reset event of `windows.CreateEvent`; its start
  takes `ERROR_IO_PENDING` and a synchronous success as started and closes the event on any other error; `Wait` is
  `WaitForSingleObject`, `Cancel` is `CancelIoEx`, and `Reap` is `GetOverlappedResult` with its wait set, then closes
  the event, gives the code, and keeps the buffer and the `Overlapped` alive until then with `runtime.KeepAlive`. No
  other DLL is loaded by name (R1, R2, R3, R6, R11, R18, R19, R20)
- [ ] In `layout_test.go`, add that only `internal/winhid/seam_windows.go` and its test import
  `golang.org/x/sys/windows`, and that both are excluded from a Linux build; and that no Go file of the module calls
  `NewLazyDLL`, `LoadLibrary` or `syscall.NewLazyDLL`, and that `NewLazySystemDLL` is called only with `"hid.dll"`
  (R2, R3)
- [ ] Check `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...`, `GOOS=windows GOARCH=amd64 go vet ./...` and
  `GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/winhid` (R1, R35)
- [ ] Run the suite and commit: `feature-0016 Call the HID API of Windows behind the seam`

### 5. The Commander on Windows

- [ ] Add `internal/commandercore/hid_test.go`, no build constraint, against `fakehid`: with the collections of
  several devices, interfaces, usages and lengths, `find` gives the first of R8 in the order of R6; a collection of
  `1b1c:0c2a` that does not match is never opened for reading and writing; no collection, and only collections of
  interface 1 or of other devices, give `ErrNoController`; one near miss, two near misses, and a near miss beside a
  match, by R34 and its exact text, with the numbers in four upper-case digits and `; ` between near misses, and
  nothing beside a match; `Problem` gives that error and nil otherwise; a write of the node is 385 bytes, `0x00`, the
  96 bytes of the command and 288 zeros, with `WriteTimeLimit`; a read is of 385 bytes and gives the 384 after the
  first; the read waits the time limit of the `Source`; a `Source.Read` against a fake that answers the eight steps
  gives the probes and leaves nothing pending and every handle closed; a failed opening gives no probes; `Lighting.Start`
  gives `ErrNoController` with no collection and the error of R34 with a near miss (R8, R17, R20, R25, R32, R34, R31)
- [ ] Add `internal/commandercore/hid.go` with `HID(seam winhid.Seam) Finder`, the near misses and their error, and
  the node on a `winhid.Device`: `send` writes the command padded to 96 bytes with `WriteReport`, `receive` maps the
  timeout of `ReadReport` to `errNoAnswer`. The length of interface 0 is `1 + reportSize`, with usage page `0xFF42` and
  usage `0x0001` as constants (R8, R17, R25, R34)
- [ ] Add `internal/commandercore/finder_windows.go` with `Windows() Finder`, which gives `HID(winhid.Windows())`
  (R25)
- [ ] Run the suite and commit: `feature-0016 Drive the Commander on Windows`

### 6. The command line on Windows

- [ ] In `internal/cli/commands_test.go`, give the harness `nearMisses []string` and `controllerProblem error`, returned
  by `listScreens` and by the `controllerProblem` of the `sourceSet`. Add tests: `devices` with no screen and two near
  misses writes `no supported screen found` and the two lines to standard error and succeeds; `devices` with a screen
  and near misses writes only the screen; `sensors` with no reading and a problem writes the problem and then `no
  sensor could be read` to standard error; with no problem only the second line, as today (R10, R26, R33, R34)
- [ ] Change `listScreens` in `main.go`, `commands.go` and `device_linux.go` to give the near misses, nil on Linux,
  and write them in `devices` as the test says. Add `controllerProblem` to `sourceSet`, fill it in
  `commanderSources` from `finder.Problem`, and write its error in `sensors` before `noSensorMessage` (R10, R26, R33,
  R34)
- [ ] Add `internal/cli/device_windows.go`: `listScreens` from `winhid.List(winhid.Windows())` and `winhid.Screens`,
  the lines and `NearMissLines`; `openTransport` by `winhid.OpenPath` with `--device` and `winhid.OpenDiscovered`
  without, returning a nil interface on an error as Linux does. The roots are ignored (R10, R11, R12, D13)
- [ ] Add `internal/cli/sources_windows.go`: `newSources` gives `commanderSources(commandercore.Windows())` with no
  other source, and no error. The roots are ignored (R25, R26, R27, D13)
- [ ] Change the build line of `device_other.go` and `sources_other.go` to `//go:build !linux && !windows`, and check
  `GOOS=darwin GOARCH=arm64 go build ./...` and `GOOS=freebsd GOARCH=amd64 go vet ./internal/cli`, so that another
  system still fails with `<system> is not supported yet` (R4)
- [ ] Check that `internal/cli/keeper_test.go` still passes its test of `controller setup failed: fake failure`, which
  shows that a setup error that is not `errNoController`, as that of R34, is written once with no change of the keeper
  (R27, R34)
- [ ] Run the suite and commit: `feature-0016 Open the screen and the Commander on Windows`

### 7. The CI job on Windows

- [ ] In `release_test.go`, change `TestTheCIWorkflowRunsTheSuite` to require exactly two jobs: the job of the suite on
  `ubuntu-24.04` with the five commands as today, and a job on `windows-2025` whose steps check out and set up Go from
  `go.mod` with the same `uses` as the job of the suite, and run `go build ./...`, `go vet ./...` and `go test
  ./internal/winhid ./internal/commandercore` as three steps of their own. Both jobs pass `checkExactActions` (R36,
  R38)
- [ ] Add the job `windows` to `.github/workflows/ci.yml`: `runs-on: windows-2025`, `actions/checkout@v5.0.0`,
  `actions/setup-go@v6.0.0` with `go-version-file: go.mod`, then the three steps named `Build`, `Vet` and `Test`, with
  a comment that the job tests only the packages written for Windows, as D15 says. The release workflow does not
  change (R36, R37, R38)
- [ ] Check by `GOOS=windows GOARCH=amd64 go vet ./internal/winhid ./internal/commandercore` and `GOOS=windows
  GOARCH=amd64 go list -test -deps ./internal/winhid ./internal/commandercore` that the tests of the two packages build
  on Windows without `internal/fakecommander`, `internal/hidraw` or `internal/fakepty` (R37)
- [ ] Run the suite and commit: `feature-0016 Build and test the Windows code on a Windows runner`

### 8. The documents

- [ ] In `README.md`, replace the paragraph that says Windows is not supported yet with what R28 lists: the screen and
  the Commander Core XT work on Windows with the commands that open a device, `sensors` and `run`, with no driver and
  no rule; the sensors of the machine are not read on Windows yet, so `sensors` lists only the probes of the Commander
  and `run` sends only them, besides the clock and the theme; `--device` takes a device path as `devices` writes it;
  iCUE or the vendor's software of the screen may hold the device and make the opening fail, and running it at the same
  time is not supported; the service is not available on Windows yet; and the Windows transport has not yet been
  checked on a real Windows machine (R28)
- [ ] In `README.md`, change the line of the Windows archive under "A release archive" and the first sentence of
  "Quick start" to say what R28 says works on Windows, in place of the theme commands only, and say in "Quick start"
  that its first step is for Linux only (R29)
- [ ] In `docs/protocol.md` under "Transport", add how a report is written and read on Windows: the report ID `0x00`
  first, 65 bytes each way, the lengths taken from the caps of the collection. In `docs/commander-core-xt.md`, add under
  "Identity" that on Windows interface 0 is the collection whose path holds the field `mi_00`, with usage page
  `0xFF42`, usage `0x0001` and reports of 385 bytes, under "Reports" that on Windows a write is the full output report of
  385 bytes and a read the full input report of 385 bytes with the report ID first, and under "Assumptions" that
  interface 0 is told apart on Windows by the `mi_00` field and its usage and lengths (R30)
- [ ] Run the suite and commit: `feature-0016 Document the screen and the Commander on Windows`

## Verification

- [ ] `go test ./...` passes (R5)
- [ ] `go vet ./...` reports nothing (R5)
- [ ] `gofmt -l .` prints no file name (R5)
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing (R5)
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing (R5)
- [ ] `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` succeeds (R1)
- [ ] `git diff master -- go.mod go.sum THIRD_PARTY_LICENSES internal/smartmonitor` shows no change, the
  `transport` interface of `internal/cli/main.go` is unchanged, and `grep -rn '"C"'` finds no import of `C` (R1, R4)
- [ ] `npm run check-changes` passes

## Commit Points

- [ ] `feature-0016 Write the plan of the HID transport on Windows`
- [ ] `feature-0016 Share the texts of the screen discovery`
- [ ] `feature-0016 Split the Commander package into its protocol and a node`
- [ ] `feature-0016 Add the Windows HID layer on a seam, with its fake`
- [ ] `feature-0016 Call the HID API of Windows behind the seam`
- [ ] `feature-0016 Drive the Commander on Windows`
- [ ] `feature-0016 Open the screen and the Commander on Windows`
- [ ] `feature-0016 Build and test the Windows code on a Windows runner`
- [ ] `feature-0016 Document the screen and the Commander on Windows`
