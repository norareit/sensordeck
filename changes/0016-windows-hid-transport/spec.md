# The HID transport on Windows

## Context

Step 7 of the roadmap of the initial plan is Windows support: HID on Windows, and a sensor source on Windows. This unit
is its first part, the HID transport. After it, the screen (`0483:0065`) and the Corsair iCUE Commander Core XT
(`1b1c:0c2a`, USB interface 0) are found and driven on Windows by the commands that open a device on Linux today:
`devices`, `clock`, `values`, `upload`, `listen`, `raw` and `run`, and the Commander in `sensors` and `run`. On
Windows every one of these fails today with `windows is not supported yet`.

On Linux the screen is reached through the package `internal/hidraw`, which implements the `Transport` of
`internal/smartmonitor` (`Write`, `Read` with a time limit whose expiry is `smartmonitor.ErrReadTimeout`, `Reopen`) and
adds `Serial` and `Close` for the command line. The code of the commands chooses the transport in two files limited to
Linux, and a third, limited to every other system, makes the commands fail. `run` depends on the transport in three
ways only: a failed `Write` is the screen lost; `Reopen` fails while the screen is gone and succeeds once it is back;
a `Read` whose time limit passed is told apart from every other failure. The upload depends on the same: it reopens
after the reset and after the closing block, whatever error a try gives. No code looks at which error of the system a
failure carries, beyond the wording of a failed opening.

The Commander is driven by the package `internal/commandercore`, which is limited to Linux as a whole. It holds the
protocol (the commands, their order, the checks of the answers, the session of `commander_color`) and, mixed with it,
the access to the node: it finds the node through `internal/hidraw.List`, opens it with `hidraw.OpenNode`, and writes
and reads an `*os.File` directly, 97 bytes per write and a read into a buffer of 384 bytes. It has no transport
contract of its own. To run on Windows, its protocol must be separated from the access to the node, without a change
of what it does on Linux.

The unit is built in a sandbox with no Windows and no device. The Windows calls cannot run there; they are built and
vetted with `GOOS=windows`, and everything around them is tested on Linux. Every behaviour on a real Windows machine is
the human's to check, and the automated loop does not wait for that check. The section "Unverified until the human
runs it" lists what is open until then.

### What is known of HID on Windows

These are facts of the Windows HID API as Microsoft documents it.

- The HID class driver makes one device interface of the class that `HidD_GetHidGuid` gives for each top-level
  collection of each HID interface. Its path is opened with `CreateFileW`. The list of the paths of the present
  interfaces of a class is given by `CM_Get_Device_Interface_List` with `CM_GET_DEVICE_INTERFACE_LIST_PRESENT`, and by
  `SetupDiGetClassDevsW` with `DIGCF_PRESENT|DIGCF_DEVICEINTERFACE` followed by `SetupDiEnumDeviceInterfaces` and
  `SetupDiGetDeviceInterfaceDetailW`
- A path can be opened with a desired access of zero, which reads no reports but lets `HidD_GetAttributes`,
  `HidD_GetProductString`, `HidD_GetSerialNumberString` and `HidD_GetPreparsedData` be called. This works also for
  devices that the system holds, such as keyboards and mice, whose paths refuse an opening for reading and writing
- `HidD_GetAttributes` gives the vendor ID and the product ID. `HidD_GetPreparsedData`, `HidP_GetCaps` and
  `HidD_FreePreparsedData` give the caps of the collection: its usage page, its usage, and `InputReportByteLength` and
  `OutputReportByteLength`
- A report is read and written with the report ID as its first byte, which is `0x00` for a device that uses no numbered
  reports. The two lengths of the caps count that byte. For the screen's reports of 64 bytes they are 65, and for the
  Commander's reports of 384 bytes they are 385
- `WriteFile` sends one output report and takes a buffer of exactly `OutputReportByteLength` bytes. A shorter buffer is
  refused. `ReadFile` gives one input report of `InputReportByteLength` bytes, with the report ID first
- On a handle opened with `FILE_FLAG_OVERLAPPED`, `ReadFile` and `WriteFile` return at once with `ERROR_IO_PENDING`
  while the operation runs; it signals the event of its `OVERLAPPED` when it completes. `WaitForSingleObject` waits
  for the event with a time limit in milliseconds. `CancelIoEx` asks for the cancellation of the operation, which then
  completes with `ERROR_OPERATION_ABORTED`, unless it had completed already. `GetOverlappedResult` with its wait set
  gives the result once the operation has completed. Until then the buffer and the `OVERLAPPED` belong to the system
- The path of a collection of one interface of a composite USB device holds `MI_<nn>`, the number of the interface in
  two hexadecimal digits, in its device instance part, as in `\\?\hid#vid_1b1c&pid_0c2a&mi_00#...`. The case of the path
  is not fixed. A collection of an interface with several top-level collections holds `COL<nn>` beside it. A device of
  one interface, like the screen, has no `MI_` field
- `CreateFileW` with a share mode of read and write lets other programs open the same path, unless one of them opened
  it with a share mode that excludes it, which makes the opening fail with `ERROR_SHARING_VIOLATION`

### Existing pure-Go HID packages

The initial plan asks to evaluate existing pure-Go HID packages before writing a layer. These were looked at:

| Package | Licence | Windows without cgo | Why not used |
|---|---|---|---|
| `github.com/karalabe/hid`, `github.com/ethereum/hid`, `github.com/bearsh/hid` | BSD 3-clause, with hidapi | no, it wraps hidapi with cgo | cgo is excluded; without cgo it enumerates no devices |
| `github.com/sstallion/go-hid` | BSD 3-clause | no, it wraps hidapi with cgo | cgo is excluded |
| `rafaelmartins.com/p/usbhid` and its fork `github.com/buglloc/usbhid` | BSD 3-clause | yes | reads block with no time limit |
| `github.com/telesma-app/hid`, formerly `github.com/go-ctap/hid` | Apache 2.0 | yes | before v1 with an API that may change, more dependencies, a newer Go than `go.mod` names |

No GPL package was considered, as `AGENTS.md` excludes it. D1 gives the decision.

## Requirements

"The Windows transport" is the transport of the screen that this unit adds. "The seam" is the narrow set of calls into
the system on which it rests, defined by R3. "A collection" is one device interface of the HID class, with its path.
"Interface 0" is the USB interface 0 of the Commander. "The protocol package" is `internal/smartmonitor`. "The
Commander package" is `internal/commandercore`. "The time limit of a write" is 5 seconds, by D8. `\t` stands for one
tab character.

### Build and dependencies

- **R1** No Go file of the module imports `C`, and `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` succeeds.
  The direct requirements of `go.mod` and its `go` directive do not change. The Windows calls use
  `golang.org/x/sys/windows`, which is already a requirement, and `THIRD_PARTY_LICENSES` does not change
- **R2** `hid.dll` is loaded lazily from the system directory, with `windows.NewLazySystemDLL`. A DLL that cannot be
  loaded, or a procedure that cannot be found, gives an error that names it, at the first call that needs it, and never
  a panic. No other DLL is loaded by name; the calls that `golang.org/x/sys/windows` already binds are used through it
- **R3** The calls into the system are behind a seam that holds nothing else: list the paths of the present collections
  of the HID class; open a path with zero access or for reading and writing; read the attributes, the product string,
  the serial string and the caps of an open handle; start an overlapped read or write, wait for its completion with a
  time limit, cancel it and reap it; close a handle. The files of the seam are limited to Windows by a build
  constraint. Everything else of the Windows transport and of the Commander on Windows, which is the matching of paths,
  the choice of collections, the check of the report lengths, the framing of reports, the classification of the
  completion of an operation, the wording of errors, the discovery and `Reopen`, is in files without a build constraint
  for Windows, and is tested on Linux by `go test ./...` against a fake of the seam
- **R4** The protocol package does not change, and neither does the `transport` interface of the command line. On a
  system that is neither Linux nor Windows, the commands that open a device, `sensors` and `run` fail as they do today,
  with `<system> is not supported yet`
- **R5** The suite of `AGENTS.md`, all five commands, succeeds

### Finding the devices

- **R6** The listing of the collections takes the paths that `CM_Get_Device_Interface_List` gives for the class of
  `HidD_GetHidGuid` with `CM_GET_DEVICE_INTERFACE_LIST_PRESENT`, and orders them by path, compared without regard to
  case. It opens each path with zero access and a share mode of read and write, and reads its vendor ID, its product
  ID, its product string, its serial string, its usage page, its usage and its two report lengths. It frees the
  preparsed data and closes the handle in every case. A path that cannot be opened or whose attributes or caps cannot
  be read is left out, with no error. A product or serial string that cannot be read is empty
- **R7** A supported screen is a collection with vendor ID `0x0483`, product ID `0x0065`, an `InputReportByteLength`
  of 65 and an `OutputReportByteLength` of 65. The 65 is not written as a number of its own: it is one plus the length
  of the report of the protocol package
- **R8** Interface 0 is a collection with vendor ID `0x1B1C`, product ID `0x0C2A`, the field `mi_00` in its path by R9,
  usage page `0xFF42`, usage `0x0001`, and both report lengths of 385, which is one plus the 384 bytes of the report of
  the Commander package. The Commander on Windows uses the first such collection in the order of R6. A collection of the
  Commander that does not match is never opened for reading and writing
- **R9** A path holds the field `mi_00` when the part between its first and its second `#` holds, among its fields
  separated by `&`, one that equals `mi_00` without regard to case. So `\\?\hid#vid_1b1c&pid_0c2a&mi_00#...`,
  `\\?\HID#VID_1B1C&PID_0C2A&MI_00#...` and `\\?\hid#vid_1b1c&pid_0c2a&mi_00&col01#...` hold it, and paths with
  `mi_01`, `mi_000`, `xmi_00`, no `mi_` field, or `mi_00` only after the second `#` do not
- **R10** On Windows, `sensordeck devices` writes one line for each supported screen, in the order of R6, in the format
  of the Linux line: `<path>\t<vendor>:<product>\t<name>\t<serial>`, with the IDs in four lower-case hexadecimal digits,
  the name the product string, and `-` for an empty name or serial. With no supported screen it writes what it writes
  on Linux
- **R11** Without `--device`, the Windows transport opens the one supported screen. With none it fails with
  `no supported screen found`, and with several with the error of Linux that lists their lines and asks for
  `--device`. The screen is opened for reading and writing, with a share mode of read and write and
  `FILE_FLAG_OVERLAPPED`
- **R12** With `--device <path>`, the Windows transport opens that path as it is given, in the way of R11, whatever its
  IDs, as the Linux transport opens the node given. It reads the caps of the opened handle, and when its two report
  lengths are not both 65 it closes the handle and fails with
  `cannot open <path>: its reports are <input> and <output> bytes, those of the screen are 65`
- **R13** The serial of the Windows transport is the serial string of the collection opened last, and is empty when it
  has none
- **R14** A failed opening for reading and writing fails with `cannot open <path>: <text>`, where the text is that of the
  system error. For `ERROR_ACCESS_DENIED` and `ERROR_SHARING_VIOLATION` the text is followed by
  `. Another program may hold the device, such as Corsair iCUE or the software of the screen's vendor; close it and try again`.
  The error of the system stays reachable with `errors.As`, so that `errors.Is` with `fs.ErrPermission` and
  `fs.ErrNotExist` holds as Go maps the Windows error codes

### Reports on Windows

- **R15** A `Write` of the Windows transport writes one buffer of the `OutputReportByteLength` of the open collection:
  the byte `0x00`, then the 64 bytes of the report. When fewer bytes are written it fails with
  `short write: <count> of <length> bytes`
- **R16** A `Read` of the Windows transport reads into a buffer of the `InputReportByteLength` of the open collection.
  A read of 0 bytes gives `io.EOF`. Otherwise the first byte, the report ID, is dropped, and the report is the bytes
  after it, filled up with `0x00` to 64 bytes when there are fewer, as on Linux
- **R17** On Windows the Commander writes buffers of 385 bytes: the byte `0x00`, the 96 bytes that it writes after the
  report ID on Linux, and 288 bytes `0x00`. It reads buffers of 385 bytes and drops the first byte, so that the answer
  it checks is the 384 bytes of the input report, as on Linux. On Linux the Commander writes and reads exactly the bytes
  it writes and reads today

### Time limits

- **R18** A read on Windows is an overlapped `ReadFile` whose completion is waited for at most the time limit, rounded
  up to a whole millisecond. When the wait ends without completion, the read is cancelled with `CancelIoEx` and reaped
  with `GetOverlappedResult` with its wait set, and only then does `Read` return. A reaped read that completed with data
  gives its report, as a read that completed in time does. A reaped read that completed with `ERROR_OPERATION_ABORTED`
  gives an error for which `errors.Is` with `smartmonitor.ErrReadTimeout` is true. Every other result gives an error
  for which it is false
- **R19** A write on Windows is an overlapped `WriteFile` waited for in the same way, with the time limit of a write.
  When the wait ends without completion, the write is cancelled and reaped. A reaped write that completed with all its
  bytes is a success; otherwise it fails with `write timed out after 5s`, an error for which `errors.Is` with
  `smartmonitor.ErrReadTimeout` is false
- **R20** No `Read`, `Write`, `Reopen` or `Close` of the Windows transport, and no exchange of the Commander on Windows,
  returns while an operation it started on the handle is still pending. The buffer and the `OVERLAPPED` of an operation
  are not reused, and stay reachable, until it has completed or been reaped
- **R21** The completion of an operation is classified by a function of plain values: whether the wait ended in time,
  whether the transport cancelled it, the Windows error code of the completion and the count of bytes. It gives a
  report or success, the timeout of R18, the failure of R19, or a failure that names the operation and carries the
  error code, worded `read failed: <text>` or `write failed: <text>` with the text of the system error. An
  `ERROR_OPERATION_ABORTED` of an operation that the transport did not cancel is such a failure, and not a timeout

### Loss and return

- **R22** `Reopen` of the Windows transport closes the handle it holds, if any, and opens again the way it was opened
  first: by the discovery of R11 anew without `--device`, by the same path with it. When the opening fails the
  transport holds no handle, `Write` and `Read` fail with an error that is not the timeout, and a later `Reopen` tries
  again. `Close` closes the handle, and a second `Close` does nothing
- **R23** With the Windows transport, `run`, `upload` and the other commands do what they do on Linux when the screen
  is lost and comes back, with no change of their code: a failed write is the screen lost, `run` writes `screen lost`
  and `screen back` and calls `Reopen` as it does on Linux, and the upload reopens after the reset and after the
  closing block as it does on Linux

### The Commander on Windows

- **R24** The Commander package holds its protocol once for both systems: the commands, their order, the checks of the
  answers, the time limits, the reading of the probes, the session of `commander_color` and the teardown. Only finding
  interface 0, opening it, and the framing of R17 differ between Linux and Windows. On Linux it does what it does
  today, and its tests and the end-to-end tests pass, changed only where they name what the node is
- **R25** On Windows, the Commander is opened for reading and writing with a share mode of read and write and
  `FILE_FLAG_OVERLAPPED`, its reads wait as R18 says with the time limit the Commander package gives them today, and its
  writes as R19 says. With no collection of interface 0 by R8, a reading gives no probes and the setup of a session
  gives `commandercore.ErrNoController`, as on Linux when no node is found. A failed opening makes the probes
  unavailable, as on Linux
- **R26** On Windows, `sensordeck sensors` lists the probes of the Commander, `commander.<n>.temp`, which are its only
  readings. When the Commander is missing it writes `no sensor could be read` to standard error, as on Linux
- **R27** On Windows, `sensordeck run` runs with the Commander as its only source of readings. It sets the clock,
  uploads the theme when it is not on the screen, and sends the probes of the Commander that the config maps, with
  `commander_interval` and `commander_color` as on Linux, and `controller lost` and `controller back` as on Linux. A
  mapped metric of any other source is not sent, as for a metric that cannot be read on Linux

### The documents

- **R28** The README says, in the place of its sentences that Windows is not supported yet:
    - that on Windows the screen and the Commander Core XT work with the commands that open a device, `sensors` and
      `run`, with no driver and no rule to install
    - that the sensors of the machine are not read on Windows yet, so that `sensors` lists only the probes of the
      Commander and `run` sends only them, besides the clock and the theme
    - that `--device` on Windows takes a device path as `devices` writes it
    - that Corsair iCUE, or the vendor's software of the screen, may hold the device and make the opening fail, and that
      running it at the same time as sensordeck is not supported
    - that the service is not available on Windows yet
    - that the Windows transport has not yet been checked on a real Windows machine
- **R29** The README's quick start and the line of the Windows archive under "A release archive" say what R28 says
  works on Windows, in place of saying that only the theme commands work there
- **R30** `docs/protocol.md`, under "Transport", says how a report is written and read on Windows: with the report ID
  `0x00` first, 65 bytes each way, the lengths taken from the caps. `docs/commander-core-xt.md` says that on Windows the
  writes are the full output report of 385 bytes and the reads the full input report of 385 bytes with the report ID
  first, and how interface 0 is told apart there, under "Identity" and "Assumptions" as R8 and R9 say

### Tests

- **R31** Unit tests that run on Linux in `go test ./...` cover, against a fake of the seam where a call is needed:
    - R9, with each of the paths it names
    - the choice of R7 and R8 among collections of several devices, interfaces, usages and lengths, and the order of R6
    - the listing of R6 leaving out a path that fails to open or to give its attributes or caps
    - R10 to R13, with no, one and two screens, and `--device` with right and wrong lengths
    - the wording of R14 for each error code it names and for another one
    - the framing of R15, R16 and R17, a short write, a read of 0 bytes, a short read
    - every case of R18, R19 and R21, the race of a read that completed after the wait ended among them, and the
      rounding of the time limit
    - R20, as a fake of the seam that records whether an operation was still pending when the call returned
    - R22 and R25

## Unverified until the human runs it

Each of these is open until the human checks it on a Windows machine with the devices. The unit does not depend on
it, and a wrong one is corrected by a later unit.

- That the paths, the IDs, the usage page and usage `0xFF42`/`0x0001` and the report lengths of the screen and the
  Commander are those of R7 to R9 on a real machine, so that `devices` lists the screen and `sensors` reads the probes
- That the screen and the Commander accept and answer the writes of R15 and R17, the Commander a full output report of
  384 bytes. The trace of OpenLinkHub in `docs/commander-core-xt.md` shows that it took one for the colour write
- Which error a read or a write on the handle of an unplugged device gives (the guesses of D7), and whether a pending
  read ends at the unplug or at its time limit. Either way R23 holds, since a write follows within a round
- That the screen enumerates again after its reboot within the reopen attempts of the upload, about 10.5 seconds in
  all, and that the screen in the state after the reset is found by the same discovery
- That `CancelIoEx` ends a pending HID read at once, so that a time limit is not overrun by much
- That iCUE, when it runs, makes the opening fail or lets it succeed, and what the README's warning should say then

## Decisions

- **D1 A thin layer of our own, no HID package.** The transport must give a read with a time limit whose expiry is told
  apart from every other failure, and must never leave an operation pending, because `run`, `listen` and the upload
  depend on both. `rafaelmartins.com/p/usbhid` and `github.com/buglloc/usbhid` read with no time limit, so a timeout
  would leave a goroutine blocked on the handle and a read pending that takes the next report. The cgo wrappers of
  hidapi are excluded by the rule of no cgo, and without cgo they find no device. `github.com/telesma-app/hid` has
  timeouts, but it is before v1 with an API that may change, brings more modules and their licences into
  `THIRD_PARTY_LICENSES`, and needs Go 1.27.3 where `go.mod` names 1.27; its framing and error handling would still have
  to be checked against the contract of the protocol package. The layer needs about a dozen calls, all of them in
  `golang.org/x/sys/windows` or in `hid.dll`, and R3 needs a seam under it anyway so that its logic is tested on Linux
- **D2 `CM_Get_Device_Interface_List` and not the SetupDi calls.** The request names `SetupDiGetClassDevsW`,
  `SetupDiEnumDeviceInterfaces` and `SetupDiGetDeviceInterfaceDetailW`. `golang.org/x/sys/windows` binds
  `CM_Get_Device_Interface_List`, which gives the same list of present interface paths as Go strings. The SetupDi
  interface calls are not bound there, and declaring them by hand means laying out
  `SP_DEVICE_INTERFACE_DETAIL_DATA_W`, whose size field must be 8 on 64-bit Windows and 6 on 32-bit, a known source of
  failures. Using the bound call loads no `setupapi.dll`, and leaves less code that cannot run in the sandbox
- **D3 Listing with zero access.** Opening every HID path for reading and writing fails for keyboards and mice and
  touches devices that are not ours. Zero access reads what R6 needs from every collection and sends nothing to any
  device. Only the chosen collection is opened for reading and writing
- **D4 Interface 0 by the `MI_00` field of the path.** The interface number is the rule on Linux, where `HID_PHYS`
  ends in `/input0`, and it is in the device instance part of the path on Windows. The usage alone cannot tell the two
  interfaces apart, because the report descriptor of interface 1 is not known. Reading the number from the properties
  of the parent USB device with the configuration manager was rejected as more calls behind the seam for the same
  number. The match is on whole `&`-separated fields of the instance part, without regard to case, because Windows
  writes the path in either case and `mi_000` or `xmi_00` must not match. hidapi reads the interface number from the
  same field
- **D5 The collection by usage and lengths.** Windows makes one path per top-level collection, so one interface may
  give several. For the Commander, the usage page `0xFF42` and usage `0x0001` of the report descriptor of interface 0 in
  `docs/commander-core-xt.md` and the lengths of 385 pick the one the protocol speaks to. For the screen the report
  descriptor is not recorded in the project, so the lengths of 65 alone pick it among the paths of `0483:0065`; this
  also keeps a second collection of one screen from being taken for a second screen. Requiring a usage page for the
  screen was rejected, since none is known
- **D6 The lengths from the caps, checked.** The framing of R15 to R17 takes the lengths from the caps of the open
  collection and not from constants, because `WriteFile` refuses a buffer of another length. The check of R7 and R8
  makes a device whose lengths are not what the protocol packages expect not found, rather than found and failing at its
  first write with an error of the system. With `--device`, R12 says so in words, because the user named the path
- **D7 The Windows errors.** `run` and the upload need no particular error, only the three ways of the Context, so the
  Windows errors are not translated into Linux errors: each failure carries its Windows error code and the text of the
  system, as the Linux transport carries the errno. What is a fact and what is a guess:
    - Fact: an operation cancelled with `CancelIoEx` completes with `ERROR_OPERATION_ABORTED`, so R18 makes that the
      timeout, but only for an operation the transport cancelled itself, since a removal may abort an operation too
    - Fact: a completion with data after the wait ended is a real report, so R18 gives it and loses nothing
    - Fact of Go: `syscall.Errno` maps `ERROR_ACCESS_DENIED` to `fs.ErrPermission`, and `ERROR_FILE_NOT_FOUND` and
      `ERROR_PATH_NOT_FOUND` to `fs.ErrNotExist`, which R14 keeps reachable
    - Guess: a read or a write on the handle of an unplugged device fails with `ERROR_DEVICE_NOT_CONNECTED`, and
      sometimes `ERROR_GEN_FAILURE`; an opening of its path fails with `ERROR_FILE_NOT_FOUND` or `ERROR_PATH_NOT_FOUND`;
      and the discovery does not find it. None of these is relied on: any failure of a write is the screen lost, and any
      failure of `Reopen` is the screen still gone
    - Guess: iCUE may hold the Commander, and the vendor's software the screen, with a share mode that gives
      `ERROR_SHARING_VIOLATION`, or open them shared. R14's wording covers the first, the README the second
- **D8 A write waits at most 5 seconds.** On Linux a write of an output report to hidraw through `usbhid` is an
  interrupt transfer that the kernel ends after 5 seconds, so 5 seconds keeps the behaviour of Linux. A shorter limit
  could fail a write that the device is slow to take after its reboot; a write with no limit could hang `run` for ever.
  Since unit `0017-clock-every-second`, `run` writes the clock once a second, so a write that waits its full limit
  delays the ticks after it by up to 5 seconds, as a slow write does on Linux
- **D9 `run` works on Windows now, with the Commander only.** Waiting for the sensors unit was rejected: with this unit
  `run` already sets the clock, uploads the theme and sends the probes, and it is the only command that exercises the
  loss and return of R23, which the human is to check. The theme's tiles of other metrics keep what they show, as
  for a metric that cannot be read on Linux, and the README says so
- **D10 The Commander package is split, its Linux bytes kept.** Its protocol runs on a node of two operations, write a
  command and read an answer within a time limit, and the finding, opening and framing are given to it per system. On
  Linux the node keeps writing 97 bytes and reading into 384, because that is the traffic checked on the device;
  writing the full 385 bytes on Linux too was rejected as a change of Linux behaviour that this unit does not need. The
  split is a refactoring that changes no behaviour, and is committed apart from the code of Windows, as the standards
  ask
- **D11 Names and order.** The name column of `devices` is the product string, as the request asks; on Linux it is
  `HID_NAME`, which the kernel makes from the manufacturer and the product, so the column differs between the systems.
  The order by path makes `devices` and the error of several screens stable from run to run, where the order of the
  list of the system is not documented
- **D12 Shared opening.** The screen and the Commander are opened with a share mode of read and write, as the request
  asks, so that sensordeck does not lock out other programs and is not refused by a program that opened the device
  shared. Talking to a device from two programs at once stays unsupported, as the README says of the Commander
- **D13 The roots are ignored on Windows.** `SENSORDECK_SYSFS_ROOT`, `SENSORDECK_DEV_ROOT` and `SENSORDECK_PROC_ROOT`
  are hooks of the Linux end-to-end tests. On Windows the transport and the Commander take no root

## Non-goals

- The sensors of the machine on Windows, through LibreHardwareMonitor or otherwise, which are the next unit
- The startup task on Windows, and `sensordeck service` on Windows
- End-to-end tests on Windows, and a run of any test on Windows
- Notifications of device arrival and removal, with `RegisterDeviceNotification` or otherwise; `Reopen` as `run`
  calls it is enough
- Flushing the input queue of the HID class driver, with `HidD_FlushQueue` or otherwise, which the Linux transport does
  not do with hidraw either
- Feature reports, numbered reports, and devices other than the screen and the Commander
- 32-bit Windows and Windows on ARM, for which the project builds no release
- A change of the protocol, of the commands, their flags or their output on Linux

## Beyond the request

1. When the listing finds collections of `0483:0065` but none with the lengths of R7, `no supported screen found`
   goes on with the lengths it found, and the same for the Commander in `sensors`. Cost: carrying the rejected
   collections out of the choice, and tests. Without it, a screen whose caps differ on a real machine shows as not
   found, and the human tells us, and the lengths are read with a debugging build
2. The name column of `devices` on Windows made as the kernel makes `HID_NAME`, the manufacturer string, a space, and
   the product string, so that the line is the same on both systems. Cost: one more call, `HidD_GetManufacturerString`,
   and a test. Without it, the column differs between the systems, which nothing reads
3. A bound on the reap after `CancelIoEx`, with the handle closed and the buffer left to the system when the cancel does
   not complete within a second. Cost: more states in R20 and their tests. Without it, a driver that does not honour the
   cancel would hang a call, which would be seen at the human's check and fixed then
4. A CI job on `windows-latest` that runs the unit tests of the Windows transport and the Commander, including the files
   limited to Windows, against the fake and with no device. Cost: a job in the workflow, and limiting the end-to-end
   tests, which need Linux, to Linux by a build constraint. Without it, the Windows-only files are built and vetted but
   never run before the human's check
