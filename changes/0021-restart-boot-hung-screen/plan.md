# Plan: restarting a screen whose firmware hung at boot

## Approach

The suspend recovery of unit 0020 is replaced by a restart of the screen's firmware. The Linux transport gains the
restart as a function of `internal/hidraw` that takes everything it touches as a seam: the usbfs opening and the ioctl,
the hidraw node to close and reopen, the clock and the wait. Its tests drive it with a sysfs tree of `fakeusb` and fakes
of those seams, with a time the test moves. The real usbfs seam is a small file of its own that opens the node and
makes the ioctl `USBDEVFS_CONTROL`.

The command line keeps one small interface over the system, as unit 0020 did, renamed from `screenPower` to
`screenRestarter` since it no longer reads any power state: it tells a timeout and makes a restart. On Linux it is
backed by the restart of `internal/hidraw` on the transport that `openTransport` opened; on Windows and the other
systems it tells no failure a timeout. The feeder of `run` and the one-shot commands depend on the interface only, and
their tests use a fake restarter whose outcomes the test sets.

The hang state of the feeder stays beside `lost`, with its lines said once in a hang. The time of the last restart that
sent a request is kept outside the hang state, so that the minute of R15 holds across hangs as well as within one.

The one-shot commands keep the default handling of Ctrl+C, as the comment of `notifyStop` in `main.go` requires, and
pass `context.Background()` to the restart. A SIGINT or SIGTERM ends the process at once, which ends the requests and
the wait before the next of them (R5). `run` passes its stop context, which a signal ends.

### Layout of the sysfs and dev trees the code reads

Under the sysfs root the tree of unit 0020 stays as it is: `class/hidraw/hidraw<n>` links to
`devices/pci0000:00/0000:00:14.0/usb1/1-4/1-4:1.0/0003:0483:0065.0001/hidraw/hidraw<n>`, and the USB device `1-4` has a
`uevent` with `DEVTYPE=usb_device`. The USB device holds `busnum` and `devnum`, each a decimal number and a line feed as
the kernel writes them (`1\n` and `5\n` in `fakeusb`). The usbfs node is `<dev root>/bus/usb/001/005`, the numbers
written with `%03d`. The flat tree of `internal/fakescreen` has no USB device, so the end-to-end tests see a restart
that is not possible and, since no write of theirs times out, never make one.

### Names fixed by this plan

* `internal/hidraw`, file `usb.go` (the file `power.go` renamed): `TimedOut` and `USBDevice` as they are, and
  `isUSBDevice`. `Recovery`, `Suspended` and `readPowerFile` are removed
* `internal/hidraw`, file `restart.go`:
    * `type RestartOutcome int` with the constants `RestartNotPossible`, `NotRestarted`, `RestartedOpen` and
      `RestartedNotOpen`
    * `type RestartResult struct { Outcome RestartOutcome; Index int; Err error }`, where `Index` is the index the
      screen left the bus at, set for the two restarted outcomes, and `Err` the error of R3 step 1, set for
      `RestartNotPossible`
    * `type ControlRequest struct { RequestType, Request uint8; Value, Index, Length uint16; Timeout time.Duration }`
    * `type Usbfs interface { Open(path string) (UsbfsNode, error) }` and
      `type UsbfsNode interface { Control(request ControlRequest) error; Close() error }`
    * `type Screen interface { Close() error; Reopen() error }`, which `*Transport` satisfies
    * `type Restart struct { SysfsRoot, DevRoot, Node string; Usbfs Usbfs; Screen Screen; Now func() time.Time;
      Wait func(stop context.Context, duration time.Duration) bool }` and its method
      `Run(stop context.Context) RestartResult`
    * the constants `firstMissingString = 4`, `lastStringIndex = 255`, `requestTimeout = time.Second`,
      `scanLimit = 10 * time.Second`, `reopenPoll = 100 * time.Millisecond` and `reopenLimit = 10 * time.Second`
* `internal/hidraw`, file `usbfs.go`: `type SystemUsbfs struct{}`, whose `Open` opens the path with `os.O_RDWR` and
  whose node's `Control` makes the ioctl. The ioctl number is the constant `usbdevfsControl`, `_IOWR('U', 0, struct
  usbdevfs_ctrltransfer)`, computed as `3<<30 | size<<16 | 'U'<<8` with `size` the `unsafe.Sizeof` of a Go struct laid
  out as the kernel's: `bRequestType uint8, bRequest uint8, wValue uint16, wIndex uint16, wLength uint16,
  timeout uint32` (milliseconds) and `data unsafe.Pointer`, which gives `0xc0185500` on 64-bit systems. The data is a
  buffer of `Length` bytes kept alive across the call with `runtime.KeepAlive`. A failed ioctl gives
  `os.NewSyscallError("ioctl USBDEVFS_CONTROL", errno)`, so that `errors.Is(err, syscall.ENODEV)` finds it
* `internal/cli`, file `hang.go`:
    * the interface `screenRestarter` with `timedOut(err error) bool` and `restart(stop context.Context) restartResult`
    * `type restartOutcome int` with `restartNotPossible`, `notRestarted`, `restartedOpen`, `restartedNotOpen`, and
      `type restartResult struct { outcome restartOutcome; index int; err error }`
    * `noScreenRestarter`, whose `timedOut` is false and whose `restart` gives `restartNotPossible` with the error
      `restarting the screen is supported on Linux only`
    * the texts: `hungText = "the screen has hung"`; `restartingLine = "restarting the screen to recover it"`; the
      format of R9 `restartedFormat = "screen restarted at string descriptor %d"`; the line of R11
      `notRestartedLine`; the line of R12 `stillHungLine`; and `cannotRestartFormat = " and sensordeck cannot restart
      it: %v. Installing the current udev rule lets sensordeck recover it, see ..."`, the words of R13 from
      ` and sensordeck cannot restart it: ` to the end, which the line of R13 (`hungText` before it) and the error of
      R17 share, so that the text exists once
    * `restartAfterHang(err error, restarter screenRestarter) error`, which replaces `hangError`, and `hangedError`
      and `observedTransport` as they are
* `internal/cli`, file `device_linux.go`: `linuxRestarter` holding the two roots and the `*hidraw.Transport`. Its
  `restart` builds a `hidraw.Restart` with `Node` read from the transport at the call, `hidraw.SystemUsbfs{}`, the
  transport as `Screen`, `time.Now` and `waitFor`, and maps the result
* `internal/fakeusb`: `New` writes `busnum` and `devnum` in place of the power files, and keeps the sysfs and dev
  roots and the number; `(*Device).SetDevnum(content string)` rewrites `devnum`; `(*Device).Unplug()` removes the
  directory `devices/pci0000:00` of the tree, the link `class/hidraw/hidraw<n>` and the node, as the kernel does when
  the device leaves the bus. A test that makes the screen come back calls `New` again with another number.
  `Set` and `Remove` of the power files are removed

### The steps of `Restart.Run`

1. `USBDevice(SysfsRoot, DevRoot, Node)`; when not found, `RestartNotPossible` with the error `no USB device in sysfs
   for <node>`. Read `busnum` and `devnum` of the device, trimmed and parsed with `strconv.Atoi`; a failure is
   `RestartNotPossible` with the error of the reading or the parsing. `Usbfs.Open` of the usbfs path; its error is
   `RestartNotPossible` with that error as it is (it holds the path and, for a node without the rule, `permission
   denied`). Nothing is closed and nothing sent in any of these cases
2. `Screen.Close()`
3. Note `begin := Now()`. For each index from 4 to 255: return the scan when `stop.Err() != nil`; stop the scan when
   `Now().Sub(begin) >= 10s`; send `ControlRequest{RequestType: 0x80, Request: 6, Value: 0x0300 | index, Index: 0x0409,
   Length: 255, Timeout: time.Second}`; the screen has left when the error is `syscall.ENODEV`, or when after the
   request `devnum` cannot be read or reads another number than in step 1. Any other result goes on to the next index
4. Close the usbfs node
5. Not left: `NotRestarted`, with nothing reopened. Left: note `left := Now()`, then loop: return when
   `stop.Err() != nil`; `Screen.Reopen()`, and `RestartedOpen` when it succeeds; `RestartedNotOpen` once
   `Now().Sub(left) >= 10s`; otherwise `Wait(stop, 100ms)`, returning when it gives false

A stop gives `NotRestarted` before the screen left and `RestartedNotOpen` after; the callers tell a stop by their own
stop context and write nothing for it.

### What the feeder of `run` does in a hang

The feeder holds `restarter screenRestarter` in place of `power`, `lastRestart time.Time` (the start of the last restart
that sent a request, zero before any), and `hang hangState` with the fields `active`, `restartingSaid`, `restarted` (a
restart of this hang made the screen leave the bus), `awaitingOpen` (that restart did not open it again), and the said
flags `notRestartedSaid`, `stillHungSaid` and `cannotRestartSaid`.

* `failed`: a timeout outside a hang begins one and writes `screen hung: ` and the error (R7); a timeout in a hang whose
  `restarted` is set writes the line of R12 once (R12); another failure ends a hang and, for a write, writes
  `screen lost: ` as today (R14)
* `reopen` in a hang calls `recoverHung`, which:
    * with `awaitingOpen` set, only reopens the transport, and clears the flag when that succeeds (R9)
    * with `lastRestart` less than a minute before now, only reopens the transport (R8, R15)
    * otherwise writes `restarting the screen to recover it` once in the hang, notes the time, and calls `restart`. When
      the stop ended it, it gives false and writes nothing more. By outcome: `restartNotPossible` writes the line of
      R13 once in the hang and reopens, leaving `lastRestart` as it was (R13); `notRestarted` sets `lastRestart`, writes
      the line of R11 once in the hang and reopens (R11); `restartedOpen` sets `lastRestart` and `restarted`, writes the
      line of R9 and gives true (R9); `restartedNotOpen` does the same, sets `awaitingOpen` and gives false (R9)
* After `reopen` gives true, the round makes the theme check and the clock due, as today, without `screen back` (R9)
* `writeSucceeded` writes `screen recovered` and clears the hang state, as today; `lastRestart` stays (R10, R15)

## Files

* `udev/70-sensordeck.rules`, `udev_test.go`
* `internal/fakeusb/fakeusb.go`
* `internal/hidraw/power.go` renamed to `internal/hidraw/usb.go`, `internal/hidraw/power_test.go` renamed to
  `internal/hidraw/usb_test.go`
* `internal/hidraw/restart.go` (new), `internal/hidraw/restart_test.go` (new)
* `internal/hidraw/usbfs.go` (new), `internal/hidraw/usbfs_test.go` (new)
* `internal/cli/hang.go`, `internal/cli/hang_test.go`, `internal/cli/hang_linux_test.go`
* `internal/cli/device_linux.go`, `internal/cli/device_windows.go`, `internal/cli/device_other.go`
* `internal/cli/main.go`, `internal/cli/commands.go`, `internal/cli/run.go`, `internal/cli/background.go`
* `internal/cli/commands_test.go`, `internal/cli/runhang_linux_test.go`
* `docs/protocol.md`, `README.md`

## Tasks

### 1. The udev rule

- [ ] In `udev_test.go`, rename the test to `TestTheUdevRuleHoldsTheHidrawRulesAndTheUsbfsRule`, make its third
  wanted line the line of R1, and add a check that no line holds `power/control` or `power/autosuspend_delay_ms`. Run
  it and see it fail (R23)
- [ ] In `udev/70-sensordeck.rules`, replace the comment and the line of the third rule by a comment that says the rule
  gives the user of the active session access to the screen's USB device node, through which sensordeck restarts the
  firmware of a hung screen, and the line of R1. The two hidraw rules and their comments stay as they are (R1, R2, D5)
- [ ] Commit: `bugfix-0021 Give access to the usbfs node of the screen in the udev rule`

### 2. The restart in the Linux transport

- [ ] In `internal/fakeusb/fakeusb.go`, add `busnum` and `devnum` to the USB device, the stored roots and number,
  `SetDevnum` and `Unplug`, as "Names fixed by this plan" says. Keep `Set` and `Remove` for now, since the tests of
  unit 0020 use them until task 3 (R24)
- [ ] In `internal/hidraw/restart_test.go` (build tag `linux`), write tests with `fakeusb`, a fake `Usbfs` that records
  each path opened and each request, answers each request by a function of the test, and moves a fake clock by a
  duration of the test for each request, a fake `Screen` that counts its closes and whose `Reopen` succeeds when
  `Discover` under the roots finds a screen and records its node, and a `Wait` that moves the clock and counts. They
  fail before the code exists:
    - the path opened is `<dev root>/bus/usb/001/005` for the node of a discovered screen, and for a `Node` that is a
      symbolic link to that node in another directory (R3, R24)
    - each request has `RequestType` `0x80`, `Request` 6, `Value` `0x0300` with the index in its low byte, `Index`
      `0x0409`, `Length` 255 and a `Timeout` of 1 second, and the indexes go 4, 5, 6 in order (R3, R24)
    - with stalls (`syscall.EPIPE`) up to index 32 and `syscall.ENODEV` at 33, the scan ends at 33, the outcome is a
      restart with index 33, and nothing is sent after it; the same with the answer of 20 calling `Unplug`, giving index
      20; and with the answer of 9 calling `SetDevnum("6\n")`, giving index 9. In each, the hidraw node was closed
      before the first request and the usbfs node closed after the last (R3, R24)
    - a request answered with no error and one failing with `syscall.EIO` are followed by the next index (R3, R24)
    - every request stalled and the clock not moved: the indexes 4 to 255 are sent and the outcome is `NotRestarted`
      with no reopening; and each request moving the clock by 1 second: the requests for the indexes 4 to 13 are sent,
      none after 10 seconds, and the outcome is `NotRestarted` (R3, R24)
    - the opening of the usbfs node failing with `permission denied`, and a `Node` with no entry in sysfs: the outcome
      is `RestartNotPossible` with the error, no request is sent, and the hidraw node is not closed (R3, R24)
    - the wait: the answer of 33 calls `Unplug`, and the third wait calls `fakeusb.New` with the number 7: the outcome
      is `RestartedOpen`, `Reopen` found `<dev root>/hidraw7`, and the waits were of 100 milliseconds. With no `New`:
      the outcome is `RestartedNotOpen` after waits adding up to 10 seconds (R3, R24, D7, D8)
    - a stop: the test ends the stop context in the answer of index 6, and no request for 7 is sent; and in the first
      wait after the screen left, and no `Reopen` follows (R5, R24)
- [ ] Write `internal/hidraw/restart.go` with the names and the steps of "The steps of `Restart.Run`". It sends nothing
  but those requests, and writes no file (R3, R4, R5)
- [ ] In `internal/hidraw/usbfs_test.go` (build tag `linux`), test that on `amd64` and `arm64` `usbdevfsControl` is
  `0xc0185500`, and that `SystemUsbfs{}.Open` of a path that does not exist gives an error that holds the path (R3)
- [ ] Write `internal/hidraw/usbfs.go` (build tag `linux`) with `SystemUsbfs` as "Names fixed by this plan" says, with
  the ioctl made by `unix.Syscall(unix.SYS_IOCTL, ...)` of `golang.org/x/sys/unix`. It makes no other ioctl (R3, R4)
- [ ] Run `go test ./internal/hidraw/...` and the five commands of the suite
- [ ] Commit: `bugfix-0021 Restart the screen by asking for string descriptors it lacks`

### 3. The seam of the command line and the one-shot commands

The tree does not build between the first step of this task and the end of task 4, which share one commit.

- [ ] In `internal/cli/hang_test.go` (all systems), replace `timeoutPower` by a `fakeRestarter` with a function
  `timedOutBy`, the outcomes the test sets (one for each restart, the last repeated), the `fakeTransport` and the
  `fakeClock` of the harness, a duration that each restart moves the clock by, and the times at which each restart
  began. Its `restart`, except for `restartNotPossible`, closes the transport and moves the clock, and for
  `restartedOpen` reopens it. Replace the tests of `hangError` by tests of `restartAfterHang` that fail before the code
  exists: a timeout with each of `restartedOpen` and `restartedNotOpen` gives the text of the error,
  `: the screen has hung` and `. sensordeck restarted it, so run the command again`; with `notRestarted`,
  ` and did not restart. Unplugging the screen once recovers it`; with `restartNotPossible` and an error,
  ` and sensordeck cannot restart it: ` and the error and the rest of the line of R13; `errors.Is` finds the error of
  the write in each; an error that is no timeout, and `nil`, are given back as they are with no restart made (R17)
- [ ] In `internal/cli/hang_linux_test.go` (build tag `linux`), replace the tests of unit 0020 by tests of `clock` with
  the harness, its writes failing with `timeoutError` and a `fakeRestarter` whose `timedOutBy` is `hidraw.TimedOut`:
  status 1 and the whole of standard error for each of the three texts of R17, one restart made, and one write tried,
  not a second. Keep `TestAnotherFailureOfAWriteIsGivenAsItIs` with no restart made (R17, R26)
- [ ] Rewrite `internal/cli/hang.go` with the names of "Names fixed by this plan". `screenPower`, `hangRecovery`,
  `noScreenPower`, `hungPrefix`, `hungByItself`, `hungNoRecover` and `hungSuspending` are removed (R13, R16, R17)
- [ ] Change `environment.openTransport` to give a `screenRestarter`. In `device_linux.go` give `linuxRestarter` in
  place of `linuxPower`; in `device_windows.go` and `device_other.go` give `noScreenRestarter{}` (R3, R6, non-goal
  Windows)
- [ ] Change `session.open`, `run.go` and `background.go` to pass the restarter where they passed the power, and the
  harness of `commands_test.go` to have a field `restarter screenRestarter` in place of `power`, `noScreenRestarter{}`
  when nil
- [ ] In `commands.go`, have `clock`, `values`, `raw` and `upload` call `restartAfterHang` with `context.Background()`
  inside it where they called `hangError`. `devices` and `listen` do not change (R17, D9)
- [ ] Rename `internal/hidraw/power.go` to `usb.go` and `power_test.go` to `usb_test.go` with `git mv`, remove
  `Recovery`, `Suspended`, `readPowerFile` and their tests `TestRecovery` and `TestSuspended`, and rename the helper
  `screenWithPower` to `usbScreen`. Remove `Set`, `Remove` and the power files from `fakeusb` (R16)

### 4. The restart in `run`

- [ ] Rewrite `internal/cli/runhang_linux_test.go` (build tag `linux`) with the harness of `run_test.go`, `failWrites`
  and `failErr` set to `timeoutError`, and a `fakeRestarter` whose `timedOutBy` is `hidraw.TimedOut` and whose
  restarts take 2 seconds. Each test checks the whole of standard error, the restarts made and the times they began,
  and the counts `closed` and `reopens` of the fake transport, and fails before the code exists:
    - a timed-out clock write and a restart `restartedOpen` with index 33: `screen hung: ` and the error,
      `restarting the screen to recover it`, `screen restarted at string descriptor 33`, the clock written in that
      round, and `screen recovered`; no `screen lost` and no `screen back` (R7, R8, R9, R10, R14, R25)
    - the same with the theme of `runtheme_test.go` and the state file emptied by the test during the hang: the round
      of the restart uploads the theme before the clock, which shows that the theme check was due (R9, R25)
    - restarts `notRestarted` and writes that time out for 70 seconds of rounds: the line of R11 once, a plain
      reopening in each round between the restarts, a second restart 60 seconds or more after the first and none
      before it (R8, R11, R15, R25)
    - a restart `restartedOpen` whose next writes time out for 70 seconds of rounds: the line of R12 once, and the next
      restart 60 seconds or more after the first (R8, R12, R15, R25)
    - restarts `restartNotPossible` with an error `open /dev/bus/usb/001/005: permission denied` over three rounds: the
      line of R13 with that error once, a restart tried in each round, a plain reopening after each, and
      `screen recovered` once a write succeeds (R8, R13, R25)
    - a restart `restartedNotOpen` and `reopenResults` failing twice and then succeeding: the line of R9, no restart in
      the two rounds whose opening fails, the opening in the third, the clock written, and `screen recovered` (R9, R25)
    - a timed-out write of an upload, with the theme of `runtheme_test.go`: `theme upload failed: ` and then
      `screen hung: ` (R7, R25)
    - a write that fails with `errFake` outside a hang: `screen lost: fake failure` and `screen back`, as today; and
      during a hang: `screen lost: fake failure`, no `screen recovered`, and `screen back` at the next reopening (R14,
      R25)
    - ticks during a hang, with an interval of 5 seconds: no write between the timed-out write and the round of the
      restart (R7, R25)
    - a second hang more than a minute after the first restart: its lines are written again and its first round makes a
      restart (R10, R25)
    - a stop during the restart, which the fake makes by ending the stop context: `run` ends with `stopped` and writes
      none of the lines of R9 and R11 (R5)
    - in every test, none of the lines of R16 is written (R16, R25)
- [ ] In `run.go`, give the feeder the restarter, `lastRestart` and the new `hangState` of "What the feeder of `run`
  does in a hang", rewrite `recoverHung` and `failed` as it says, and remove `suspendPoll`, `suspendMargin`, the closing
  of the transport and the wait for a suspend (R7 to R16)
- [ ] Update the comments of `feeder`, `reopen` and `recoverHung` to say that a hung screen is restarted, at most once a
  minute
- [ ] Check that the tests of the lost screen in `run_test.go` pass unchanged with `noScreenRestarter`, which is the
  path of every write failure on Windows (R6, R14)
- [ ] Run the five commands of the suite
- [ ] Commit: `bugfix-0021 Restart a hung screen in run and in the one-shot commands`

### 5. The documents

- [ ] In `docs/protocol.md`, replace the section `## Hang after a USB reset` by a section `## Hangs of the firmware`
  with the subsections `### The reset hang`, `### The boot hang`, `### The restart` and `### How sensordeck restarts the
  screen`, holding what R18 lists: facts 1 to 3 of unit 0020 with the date, the report descriptor, `bmAttributes`
  `0x80` and no `power/wakeup`, and that the suspend and resume cleared the reset hang once; the symptom of the boot
  hang, its trigger with the plugging in at the GRUB menu, and the list of fact 3 of this spec; that the inference of
  unit 0020 that Linux resets the screen at boot was wrong; the request of fact 4 with its fields, the string
  descriptors 0 to 3, the two observations with the date 2026-11-16, and that the one on a screen hung at boot was not
  clean; how sensordeck uses it as R3 and R8 say, through the usbfs node with usbhid bound, no USB reset and no file of
  sysfs written; the questions not established; and that `HidD_GetIndexedString` on Windows sends the same request
  through the HID handle and that sensordeck does not use it. Keep every item of `## Unknown`, with the item on Windows
  reworded as whether either hang happens on Windows. Remove the paragraphs on the kernel not suspending a busy device
  and on how sensordeck recovered by a suspend (R18, R22)
- [ ] In the README's section "Linux: access to the screen", show the three rules of the file after task 1; replace the
  paragraph of the third rule by one that says it gives the user of the session access to the screen's USB device
  node, which `run` and the commands that write use to restart a hung screen, linking
  `docs/protocol.md#hangs-of-the-firmware`; that a rule file without that line is old and is installed again, and that
  one with a line that sets `power/control` is the old rule of the version before; that
  `grep 'SUBSYSTEM=="usb".*uaccess' /etc/udev/rules.d/70-sensordeck.rules` prints nothing for an old rule; and that a
  replug applies the rule (R19)
- [ ] In the README, replace the paragraph on how `run` recovers a hung screen by one that says that on Linux `run`
  restarts a hung screen at the first round of a hang and at most once a minute after, with the lines `screen hung: `,
  `restarting the screen to recover it`, `screen restarted at string descriptor <index>` and `screen recovered`, and
  that it writes one line that says to install the current rule when it cannot restart it (R20)
- [ ] In the README's entry "The screen shows its theme but no values after boot", say that the screen hung because the
  firmware of the PC set it up before Linux, that `run` restarts it with the current udev rule, what the lines of R11
  and R12 mean, and that unplugging it once recovers it, with no USB reset at boot named as the cause (R21)
- [ ] Search `README.md` and `docs/` for `suspend`, `power/control` and `Hang after a USB reset`, and leave none that
  claims a suspend and resume recovers a hung screen in general, that sensordeck relies on one, or that links the old
  section (R22)
- [ ] Run the five commands of the suite (R27)
- [ ] Commit: `bugfix-0021 Document the hangs of the screen and its restart`

## Trace

| Requirement | Tasks |
|---|---|
| R1, R2 | 1 |
| R3 | 2, 3 |
| R4 | 2 |
| R5 | 2, 4 |
| R6 | 3, 4 |
| R7, R8, R9, R10, R11, R12, R13, R14, R15 | 4 (R13 also 3) |
| R16 | 3, 4 |
| R17 | 3 |
| R18, R19, R20, R21, R22 | 5 |
| R23 | 1 |
| R24 | 2 |
| R25 | 4 |
| R26 | 3 |
| R27 | 2, 4, 5 |
