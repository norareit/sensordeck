# Plan: recovering a screen whose firmware hung after a USB reset

## Approach

The Linux transport learns three things of sysfs, each a plain function of `internal/hidraw`: whether the error of a
write is a timeout, which USB device the hidraw entry of a node belongs to, and what that device's `power/` files say.
The command line reaches them through one small interface of its own, `screenPower`, which `openTransport` gives beside
the transport. On Linux it is backed by those functions; on Windows and the other systems it says that no failure is a
timeout and that no recovery is possible. The feeder of `run` and the one-shot commands depend on the interface only,
so the tests of `run` drive it with a fake transport and, on Linux, with the real reading of a sysfs tree in a temporary
directory.

The feeder keeps its `lost` flag, and a hang is a second piece of state beside it: a hang is always a lost screen, so the
ticks of unit 0017 already write nothing during it. What differs is what a failed write writes, what the round does
when the screen is lost, and that the first write that succeeds during a hang writes `screen recovered`. The writes of
an upload pass through the theme holder and not through the feeder's `write`, so the feeder hands the holder, and uses
itself, a thin wrapper of the transport that tells the feeder of every write that succeeded. The holder gives the error
of a failed upload back instead of a flag, so that the feeder can tell a timed-out upload.

A test-support package builds the sysfs tree as the kernel lays it out, so that the tests of `internal/hidraw` and of
`run` share one picture of it.

### Layout of the sysfs tree the code reads

Under the sysfs root, `class/hidraw/hidraw<n>` is a link to
`devices/pci0000:00/0000:00:14.0/usb1/1-4/1-4:1.0/0003:0483:0065.0001/hidraw/hidraw<n>`, whose `device` is a link to the
HID device `../..`. Of the ancestors of the HID device, the interface `1-4:1.0` has a `uevent` with
`DEVTYPE=usb_interface`, and the USB device `1-4` has a `uevent` with `DEVTYPE=usb_device` and a directory `power` with
`control`, `autosuspend_delay_ms` and `runtime_status`. The USB device of an entry is found by resolving the links of
`class/hidraw/hidraw<n>/device` and walking up from there to the first directory whose `uevent` holds the line
`DEVTYPE=usb_device`, stopping at the resolved sysfs root. The flat tree of `internal/fakescreen`, in which `device` is a
plain directory, has no such ancestor, so the end-to-end tests see a recovery that is not possible and never a timeout.

### Names fixed by this plan

* `internal/hidraw`: `TimedOut(err error) bool`; `USBDevice(sysfsRoot, devRoot, node string) (string, bool)`;
  `Recovery(device string) (delay time.Duration, possible bool)`; `Suspended(device string) bool`; the method
  `(*Transport).Node() string`, the path of the node opened last
* `internal/cli`, file `hang.go`: the interface `screenPower` with `timedOut(err error) bool` and
  `recovery() (hangRecovery, bool)`; the struct `hangRecovery` with `delay time.Duration` and `suspended func() bool`;
  `noScreenPower`, which answers false to both; the constants of the texts of R5, R6, R7 and R13; the function
  `hangError(err error, power screenPower) error` of R13; and the wrapper `observedTransport`
* `internal/fakeusb`, Linux only, test support: `New(test testing.TB, sysfsRoot, devRoot string, number int) *Device`
  building the tree above and a regular file `<dev root>/hidraw<n>` as the node; `(*Device).Set(name, content string)`
  writing a file of `power/`; `(*Device).Remove(name string)` deleting one; and the field `Node`

## Files

* `udev/70-sensordeck.rules`
* `udev_test.go` (new, package `main` at the root, beside `docs_test.go`)
* `internal/fakeusb/fakeusb.go` (new)
* `internal/hidraw/power.go` (new), `internal/hidraw/power_test.go` (new)
* `internal/hidraw/transport.go`, `internal/hidraw/transport_test.go`
* `internal/cli/hang.go` (new), `internal/cli/hang_test.go` (new)
* `internal/cli/device_linux.go`, `internal/cli/device_windows.go`, `internal/cli/device_other.go`
* `internal/cli/main.go`, `internal/cli/commands.go`, `internal/cli/run.go`, `internal/cli/background.go`,
  `internal/cli/themeholder.go`
* `internal/cli/commands_test.go`, `internal/cli/fake_test.go`, `internal/cli/themeholder_test.go`
* `internal/cli/runhang_linux_test.go` (new), `internal/cli/hang_linux_test.go` (new)
* `docs/protocol.md`, `README.md`

## Tasks

### 1. The udev rule

- [x] In `udev_test.go`, write a test that reads `udev/70-sensordeck.rules` and fails unless it holds, each as a whole
  line, the two hidraw rules as they stand and the line of R1 exactly. Run it and see it fail (R19)
- [x] Add to `udev/70-sensordeck.rules`, after the two hidraw rules and a blank line, a comment that says the rule lets
  the kernel suspend the screen when no program holds it open, which clears the hang of its firmware after a USB reset
  and which `run` relies on to recover it, and then the line of R1. The two hidraw rules stay as they are (R1, D2, D3)
- [x] Commit: `bugfix-0020 Set the screen to autosuspend in the udev rule`

### 2. The Linux side: timeout, USB device, power files

- [x] Write `internal/fakeusb/fakeusb.go` with the package comment that it is test support, as `internal/fakescreen`
  says of itself, and the build tag `linux`. `New` builds the tree of "Layout of the sysfs tree the code reads" under the
  roots, with `uevent` files for the HID device (the `HID_ID`, `HID_NAME` and `HID_UNIQ` of a screen, so that `List`
  and `Discover` find it), the interface and the USB device, and `power/control` `auto`,
  `power/autosuspend_delay_ms` `2000` and `power/runtime_status` `active` (R20)
- [x] In `internal/hidraw/power_test.go`, with `fakeusb`, write tests that fail before the code exists:
    - `USBDevice` gives the directory `1-4` for the node of a screen that `Discover` found, and for a `--device` path
      that is a symbolic link to that node in another directory (R3, R20)
    - `USBDevice` gives false for a node with no entry, and for the flat entry of `writeEntry` in `discover_test.go`,
      whose `device` has no USB ancestor (R3, R20)
    - `Recovery` gives 2 seconds and true with `auto` and `2000`; false with `on`; false with `-1`; false with
      `power/control` removed; false with `power/autosuspend_delay_ms` removed; and true with `0` and a delay of 0.
      Values are read with the trailing line feed the kernel writes, so the tests write `auto\n` and `2000\n` (R3, R20)
    - `Suspended` is true for `suspended\n`, false for `active\n`, and false when the file is missing (R5)
    - `TimedOut` is true for `&os.PathError{Op: "write", Path: "/dev/hidraw3", Err: syscall.ETIMEDOUT}` and for an
      error that wraps it with `fmt.Errorf("...: %w", ...)`, and false for the same with `syscall.EIO` and for `nil` (R2,
      R20)
- [x] Write `internal/hidraw/power.go` (build tag `linux`) with the four functions of "Names fixed by this plan".
  `USBDevice` finds the entry whose node is the same file as the path, by the lookup that `serialOfNode` makes today,
  which moves into a function `entryOfNode(sysfsRoot, devRoot, path string) (Entry, bool)` that both use. `Recovery`
  trims white space, requires `control` to be `auto`, and parses the delay with `strconv.Atoi`, requiring 0 or more.
  `TimedOut` is `errors.Is(err, syscall.ETIMEDOUT)`. None of them writes a file (R2, R3, R14)
- [x] In `internal/hidraw/transport_test.go`, add a test that `Node` is the discovered node after `OpenDiscovered`,
  the given path after `OpenPath`, and the new node after a `Reopen` that found the screen under another number, as
  `TestReopenFindsTheScreenUnderAnotherNumber` moves it (R3)
- [x] In `internal/hidraw/transport.go`, have the opening function give the path of the node with the file and the
  serial, keep it in a field of `Transport`, and add `Node`. A `Reopen` that fails leaves the field as it was, since a
  later `Reopen` sets it again (R3)
- [x] Run `go test ./internal/hidraw/...` and the five commands of the suite
- [x] Commit: `bugfix-0020 Read the USB power state of the screen's node`

### 3. The seam of the command line and the one-shot commands

- [x] In `internal/cli/hang_test.go` (all systems), write tests of `hangError` with a fake `screenPower` that fail
  before the code exists: a timed-out error with the recovery possible gives the text of the error, `: the screen has
  hung. ` and the text of R13 for a possible recovery; with the recovery not possible, the text of R7 after
  `the screen has hung. `; an error that is no timeout is given back as it is; and `errors.Is` still finds the error of
  the write through the result (R13)
- [x] Write `internal/cli/hang.go` with the `screenPower` interface, `hangRecovery`, `noScreenPower`, the constants and
  `hangError`. The line of R7 and the error of R13 share one constant for the words after `the screen has hung. `, so
  that the text exists once (R7, R13)
- [x] Change `environment.openTransport` to `func(sysfsRoot, devRoot, device string) (transport, screenPower, error)`.
  In `device_linux.go`, give a `linuxPower` holding the two roots and the transport's `Node` method: `timedOut` calls
  `hidraw.TimedOut`; `recovery` calls `hidraw.USBDevice` with the current node and then `hidraw.Recovery`, and gives a
  `hangRecovery` whose `suspended` calls `hidraw.Suspended` on the device found. Both read sysfs at each call, never
  caching (R2, R3). In `device_windows.go` and `device_other.go`, give `noScreenPower{}` (R2, non-goal Windows)
- [x] Change `session.open` to give the `screenPower` as well, and pass it on in `run.go` and `background.go` to
  `serveRun`. `listen` discards it
- [x] In `commands_test.go`, give the harness a field `power screenPower`, given by its `openTransport`, which is
  `noScreenPower{}` unless a test sets it. Add to `fakeTransport` in `fake_test.go` a field `failErr error`, the error
  of a write that `failWrites` fails, `errFake` when nil
- [x] In `internal/cli/hang_linux_test.go` (build tag `linux`), write tests of `clock` with the harness, a `linuxPower`
  on a `fakeusb` tree whose node it is given, and a write that fails with the `ETIMEDOUT` error of task 2: with `auto`
  and `2000` the command ends with status 1 and the error of R13 for a possible recovery; with `on` it ends with the
  error that holds the text of R7; and in both the transport is opened once and never reopened (R13, R22)
- [x] In `commands.go`, pass the error of a write of `clock`, `values` and `raw`, and the error of the upload of
  `upload`, through `hangError` with the power of the opening. `devices` and `listen` do not change (R13)
- [x] Run the five commands of the suite
- [x] Commit: `bugfix-0020 Tell a hung screen in the one-shot commands`

### 4. The recovery of `run`

- [x] In `internal/cli/themeholder_test.go`, change the tests that read the third result of `ensure` to read an
  error, and add one that a failed upload gives the error of the upload, which wraps the error of the write
- [x] Change `themeHolder.ensure` to give `failure error` in place of `failed bool`, the error of `upload`, still after
  the line `theme upload failed: ` that `upload` writes (R4)
- [x] In `internal/cli/runhang_linux_test.go` (build tag `linux`), with the harness of `run_test.go`, `failWrites`
  and `failErr` set to the `ETIMEDOUT` error, a `linuxPower` on a `fakeusb` tree, and `onWait` to change
  `runtime_status` and to stop, write tests that fail before the code exists. Each checks the whole of standard error,
  and the counts `closed` and `reopens` of the fake transport:
    - a timed-out clock write with `auto` and `2000`: `screen hung: ` and the error, then
      `suspending the screen to recover it`; at the next round the node is closed, the waits are of 100 milliseconds
      until the test writes `suspended`, the node is opened again, the clock is written, and `screen recovered`
      follows; no `screen lost` and no `screen back` (R4, R5, R8, R9, R10, R21)
    - the same with the theme of `runtheme_test.go` on the screen, and the state file emptied by the test during the
      hang: the round that opened the node again reads the state file and uploads the theme before the clock, which
      shows that the theme check was due (R8, R21)
    - the status never `suspended`: after waits of 100 milliseconds adding up to 7 seconds, the line
      `the screen did not suspend within 7s, another program may hold it open`, and the node opened all the same (R6,
      R21)
    - a recovery that does not help, with the status `suspended` from the start and every write timing out for three
      rounds: one `screen hung` line, one `suspending` line, two closes and two reopenings, and no line of R6 (R5, R9,
      R21)
    - the same with the status never `suspended`: the line of R6 once over the two recoveries (R6)
    - the recovery not possible, with `power/control` `on`, over three rounds: the line of R7 once, a plain reopening in
      each round and no close of R5, and `screen recovered` once a write succeeds (R7, R8, R9, R21)
    - the recovery not possible because the tree has no USB device for the node: the same as with `on` (R3, R7)
    - a `power/control` set to `auto` by the test between two rounds of a hang: the round after it recovers as R5 says,
      so the files are read at each round (R3)
    - a timed-out write of an upload, with the theme of `runtheme_test.go`: `theme upload failed: ` and then
      `screen hung: ` (R4, R21)
    - a stop during the wait of R5: `run` ends with `stopped`, the waits stop at once, no reopening follows, and the
      limit is not waited for (R11, R21)
    - a write that fails with `errFake` outside a hang: `screen lost: fake failure` and `screen back`, as today (R12,
      R21)
    - a write that fails with `errFake` during a hang: `screen lost: fake failure`, no `screen recovered`, and
      `screen back` at the next reopening (R10)
    - ticks during a hang, with an interval of 5 seconds: no write between the timed-out write and the round that
      recovers (R4, R21)
    - a second hang after `screen recovered`: its lines are written again (R9)
- [x] In `run_test.go`, check that the tests of the lost screen still pass unchanged with `noScreenPower`, which is the
  path of every write failure on Windows (R12)
- [x] In `hang.go`, write `observedTransport`, which embeds `transport` and calls a function after every `Write` that
  succeeded. In `run.go`, give the feeder the power and a hang state beside `lost`: a flag that a hang is in progress,
  and the flags of the lines said once in it (the line of R5, of R6 and of R7). Then:
    - the feeder writes through an `observedTransport` of its transport, and gives it to the holder as well; its
      function writes `screen recovered` and ends the hang when one is in progress (R9)
    - a failed write, and a failed upload given back by `ensure`, go to one method of the feeder: a timeout outside a
      hang begins one and writes `screen hung: ` and the error; a timeout in a hang writes nothing; another failure ends
      a hang without a line and, for a write, writes `screen lost: ` as today. Every failure sets `lost` (R4, R10, R12)
    - in `round`, a lost screen in a hang goes to a method that does R5 to R7 and gives whether the node is open again;
      a lost screen outside a hang reopens as today with `screen back`. After an opening in a hang the round makes the
      theme check and the clock due, as after `screen back`, without the line (R5, R6, R7, R8)
    - the method of R5 reads `recovery()` anew; when it is not possible, it writes the line of R7 once in the hang and
      reopens. When it is, it writes the line of R5 once in the hang, closes the transport, and reads `suspended`
      before each wait of 100 milliseconds made with `feeding.wait`, until it reads true or the time from the close
      reaches the delay plus 5 seconds, then writes the line of R6 once in the hang if it did not read true, and
      reopens. A wait that the stop ended gives false at once, with no reopening (R5, R6, R7, R11, R14)
- [x] Update the comment of `feeder` to say how it handles a hang
- [x] Run the five commands of the suite
- [x] Commit: `bugfix-0020 Recover a hung screen in run by a suspend and a resume`

### 5. The documents

- [ ] In `docs/protocol.md`, add a section `## Hang after a USB reset` before `## Unknown`, holding facts 1 to 3 of the
  spec with the date 2026-11-16, the report descriptor, and `bmAttributes` `0x80` with no `power/wakeup`; that fact 3
  was observed once and the reset at boot as its cause is an inference; that the screen is not suspended while a
  program holds its node open, because it cannot wake the host, as read and not observed; and how sensordeck recovers.
  Add to `## Unknown` whether a suspend and resume of a screen that works is harmless, what the screen shows while
  suspended, and whether the hang happens on Windows (R15)
- [ ] In the README's section "Linux: access to the screen", replace the sentence that the rule is one line and its
  block with the whole rule file, and say in a few sentences what the third rule does and why, linking the section of
  `docs/protocol.md`. Say that a rule file without the line that sets `power/control` is old and is installed again,
  that `grep power/control /etc/udev/rules.d/70-sensordeck.rules` printing nothing tells it, and that a replug applies
  it (R16)
- [ ] In the README, after the sentence that `run` keeps running when the screen is unplugged and plugged in again, say
  that on Linux `run` recovers a hung screen by letting it suspend and opening it again, with the lines
  `screen hung: `, `suspending the screen to recover it` and `screen recovered`, and the line of R7 without the rule
  (R18)
- [ ] Add a section `## Troubleshooting` to the README before `## Making a release`, with an entry for a screen that
  shows its theme but no values after boot, with 12:00:00 and 2022-01-01 and `screen hung` in the log: that it hung on
  the USB reset of the boot, that installing the current udev rule lets sensordeck recover it, and that unplugging it
  once recovers it now (R17)
- [ ] Run the five commands of the suite (R23)
- [ ] Commit: `bugfix-0020 Document the hang of the screen and its recovery`

## Trace

| Requirement | Tasks |
|---|---|
| R1 | 1 |
| R2 | 2, 3 |
| R3 | 2, 3, 4 |
| R4 | 4 |
| R5, R6, R7, R8, R9, R10, R11 | 4 (R7 also 3) |
| R12 | 4 |
| R13 | 3 |
| R14 | 2, 4 |
| R15 | 5 |
| R16, R17, R18 | 5 |
| R19 | 1 |
| R20 | 2 |
| R21 | 4 |
| R22 | 3 |
| R23 | 2, 3, 4, 5 |
