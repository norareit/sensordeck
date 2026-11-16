# Restarting a screen whose firmware hung at boot

## Context

Unit `0020-recover-hung-screen` made `sensordeck run` recover a screen whose writes time out by letting its USB device
suspend and opening it again, and its udev rule sets `power/control` to `auto` for that. Its spec took the hang at every
cold boot for the hang that a USB reset of a running screen causes, and inferred that Linux resets the screen at boot.
That inference was wrong, and the recovery does not clear the hang at boot: `run` closes the node, waits for the
suspend, opens it again, and the next write times out again, every 7 seconds or so, for as long as it runs. The README
and `docs/protocol.md` say that a suspend and resume recovers a hung screen in general, which is false.

### What was observed

These facts were established on the human's Linux machine on 2026-11-16, by the orchestrating session and by another
session of the human's, with sudo, each observed on the device. Facts 1 to 3 of the spec of unit 0020 stand as they
were observed there; only its inference about the boot is withdrawn.

1. **Two hangs that look the same.** In both, every write of an output report to the interrupt OUT endpoint times out
   after the kernel's 5 seconds, and the screen shows its theme with 12:00:00 and 2022-01-01. The *reset hang* follows a
   USB reset of a screen that runs, and a runtime suspend and resume cleared it once (fact 3 of unit 0020). The *boot
   hang* is there after every cold boot, and a suspend and resume does not clear it
2. **The trigger of the boot hang is the firmware of the PC.** The screen, plugged in while the GRUB menu showed, when
   the firmware of the PC (UEFI) drives USB, hung in the same way once Linux ran. So the boot hang follows the setup of
   the screen by UEFI before Linux, and not a reset by Linux. Nothing in Linux can prevent it
3. **What does not clear the boot hang.** The control endpoint still answers. These requests were accepted and did not
   help: CLEAR_FEATURE(ENDPOINT_HALT) on both endpoints, a halt and an unhalt of the OUT endpoint, the HID requests
   SET_IDLE and SET_PROTOCOL, SET_INTERFACE, SET_CONFIGURATION(1) repeated, and the command `01 reset` sent as
   SET_REPORT on the control endpoint. No endpoint reports itself halted. A suspend and resume, woken by opening the
   node or by setting `power/control` to `on`, a USB reset, and a USB reset followed by a suspend did not help either.
   A replug clears it
4. **Asking for string descriptors the screen does not have makes its firmware restart.** The screen has the string
   descriptors 0 to 3. The request is GET_DESCRIPTOR with bmRequestType `0x80`, bRequest 6, wValue `0x0300` with the
   index in its low byte, wIndex `0x0409` and wLength 255, sent for one index at a time
    - On a screen that worked, the indexes 4 to 32 were answered with a stall, the request for 33 failed with "no such
      device" (`ENODEV`), the kernel logged a USB disconnect, and the screen enumerated again 1.3 seconds later and
      worked
    - On a screen hung at boot, a scan of the indexes 0 to 255 made it disconnect and enumerate again, and writes were
      accepted afterwards. Other requests of fact 3 had been sent to it before, so this is not yet a clean observation
5. **A screen that restarts under Linux is set up by Linux alone,** which is what a replug does, and a replug is what
   was seen to clear both hangs

Not established: whether the restart is triggered by index 32, by index 33, or by the number of stalled requests in a
row; a run on a screen hung at boot with nothing sent to it first; whether the restart clears the reset hang, which is
likely, since it is a restart of the firmware as a replug is, but not observed; and whether either hang happens on
Windows.

### Observed behavior

After a cold boot with the rule and the service of unit 0020 installed, the screen shows its theme with 12:00:00 and
2022-01-01 and no values. `run` writes `screen hung: write /dev/hidraw<n>: connection timed out` and
`suspending the screen to recover it` once, and then suspends and reopens the screen every 7 seconds or so without
effect, for as long as it runs. Only a replug makes the screen work.

### Expected behavior

`run` restarts a hung screen through its control endpoint, finds it again once it has enumerated, and goes on as after
a reconnect, within a few seconds of the first write that timed out. The documents say what is known of the two hangs
and of the restart, and claim no more.

### Steps to reproduce

1. Install the udev rule of unit 0020 and the service of unit 0014
2. Switch the PC off and on again with the screen plugged in, or plug the screen in while the GRUB menu shows
3. The screen shows its theme with 12:00:00 and 2022-01-01, and `journalctl --user -u sensordeck` shows `screen hung`
   and `suspending the screen to recover it`, and no `screen recovered`

### Root cause

The boot hang is a state of the screen's firmware that follows its setup by the firmware of the PC (fact 2), and that
none of the requests or power transitions of fact 3 clears. Unit 0020 recovers by a suspend and resume, which clears
only the reset hang. What goes wrong inside the firmware is not known.

### The fix

A restart of the screen's firmware clears the hang, as a replug does (facts 4 and 5), and requests for string
descriptors that the screen does not have make it restart. So `run` sends those requests through the usbfs node of
the screen's USB device, from index 4 upwards, until the screen leaves the bus, then waits for it to enumerate again,
opens it and goes on. The udev rule gives the user of the session access to that usbfs node. The suspend recovery of
unit 0020, its rule line and its claims in the documents are removed.

## Requirements

### The udev rule

- **R1** `udev/70-sensordeck.rules` keeps its two rules for hidraw unchanged, and holds a third rule, with a comment
  above it that says why it is there, which is this line:
  `SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTR{idVendor}=="0483", ATTR{idProduct}=="0065", MODE="0660", TAG+="uaccess"`
- **R2** The rule file holds no line that sets `power/control` or `power/autosuspend_delay_ms`

### The restart

The USB device of the screen is the USB device in sysfs to which the hidraw entry of the node opened last belongs, under
the sysfs root that sensordeck reads (`SENSORDECK_SYSFS_ROOT`, or `/sys`), found as unit 0020 finds it, for a node found
by discovery and for a node given with `--device` alike. Its usbfs node is `<dev root>/bus/usb/<BBB>/<DDD>`, under the
dev root that sensordeck reads (`SENSORDECK_DEV_ROOT`, or `/dev`), where `<BBB>` and `<DDD>` are the numbers in its
`busnum` and `devnum` files, written with three digits.

- **R3** A restart is these steps, in order:
    1. Find the USB device of the screen, read its `devnum`, and open its usbfs node for reading and writing. When one
       of these fails, the restart is not possible, it gives the error, and nothing is sent
    2. Close the hidraw node
    3. Send the request of fact 4 through the usbfs node, by the ioctl `USBDEVFS_CONTROL` with a timeout of 1 second,
       for the indexes 4, 5, 6 and so on, one at a time, until the screen has left the bus. It has left the bus when a
       request fails with `ENODEV`, or when, after a request, the USB device's directory in sysfs no longer exists or
       its `devnum` differs from the one read in step 1. A request that is answered, stalled or fails with another
       error is followed by the next index
    4. Stop sending at index 255, after which no index is left, or when 10 seconds have passed since step 3 began,
       whichever comes first. The screen has then not restarted
    5. Close the usbfs node
    6. When the screen left the bus, open the screen again the way it was opened at the start, every 100 milliseconds,
       until an opening succeeds or 10 seconds have passed since it left. A node found by discovery is found by
       discovery anew, so its new hidraw node is found whatever its number

  A restart gives its outcome: not possible (with the error), not restarted, restarted and open again, or restarted
  and not open again. It gives the index whose request the screen left the bus at
- **R4** The restart sends nothing but the requests of R3 step 3. sensordeck never sends a USB reset, never unbinds or
  detaches a driver, never claims an interface, and writes no file in sysfs. usbhid stays bound to the screen
- **R5** A stop by SIGINT or SIGTERM during the requests or the wait of R3 ends them at once, before the next request
  or the next attempt to open

### What a timed-out write is

- **R6** On Linux a write to the screen has timed out when its error is, or wraps, the error of the system `ETIMEDOUT`,
  as R2 of unit 0020 says. Any other failure of a write is not a timeout, and on Windows no failure is

### What `run` does

A hang is the time from a write of `run` that timed out to the first write of `run` that succeeds after it, as in unit
0020. The writes of `run` are those of the clock, of the values and of an upload of the theme.

- **R7** The first write of a hang that times out writes `screen hung: ` and the text of the error to standard error.
  The rest of the round is as unit 0020 has it: no further report of that round is written, the ticks of unit 0017 write
  nothing until a round has opened the screen again, and a timed-out write of an upload writes `theme upload failed: `
  before `screen hung: `
- **R8** At the time of the first round of a hang, `run` writes `restarting the screen to recover it` to standard error
  and makes a restart. At the time of a later round of the hang, it makes a restart when no restart that sent a request
  began in the last minute; otherwise the round opens the screen again and writes as for a lost screen, without the
  lines `screen lost: ` and `screen back`
- **R9** A restart in which the screen left the bus writes `screen restarted at string descriptor <index>` to standard
  error, with the index of R3. When the screen is open again, the round goes on as after a screen that came back,
  without the line `screen back`: the clock is due, the theme check of R7 of unit 0013 is due, and the round then
  writes as any round does. When it is not open again, the round ends, and later rounds open the screen as for a lost
  screen until it opens, without a restart while the opening fails
- **R10** The first write of `run` that succeeds during a hang ends it and writes `screen recovered` to standard error.
  A later timed-out write begins a new hang, with its lines once again
- **R11** When a restart did not make the screen leave the bus, `run` writes
  `the screen did not restart, trying again every minute. Unplugging the screen once recovers it` to standard error,
  once in a hang, and opens the screen again as for a lost screen
- **R12** When a write times out after a restart in which the screen left the bus, in the same hang, `run` writes
  `the screen is still hung after its restart, trying again every minute. Unplugging the screen once recovers it` to
  standard error, once in a hang
- **R13** When a restart is not possible, `run` writes this line to standard error, once in a hang, with the error of
  R3 step 1 in place of `<error>`:
  `the screen has hung and sensordeck cannot restart it: <error>. Installing the current udev rule lets sensordeck recover it, see "Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen. Unplugging the screen once recovers it now`.
  The round then opens the screen again as for a lost screen. A restart that is not possible does not count for the
  minute of R8, so the next round of the hang tries again, and a rule applied while `run` runs is used
- **R14** During a hang, `run` writes no line `screen lost: ` or `screen back`, and no line of R11, R12 or R13 more than
  once. A write that fails for another reason during a hang is the screen lost, as R22 of unit 0004 says, and ends the
  hang without a line `screen recovered`. A write that fails without a timeout outside a hang is handled as today
- **R15** `run` never makes two restarts less than a minute apart, except after a restart that was not possible, and
  no round of a hang waits on the screen without a time limit
- **R16** `run` no longer closes the node to wait for a suspend, reads no file of `power/` and writes no line
  `suspending the screen to recover it` or `the screen did not suspend within ...`

### What the other commands do

- **R17** `clock`, `values`, `upload` and `raw`, whose write times out, make a restart and end with an error whose text
  is the text of the error of the write, followed by `: the screen has hung` and then:
    - when the screen left the bus: `. sensordeck restarted it, so run the command again`
    - when it did not: ` and did not restart. Unplugging the screen once recovers it`
    - when the restart is not possible: ` and sensordeck cannot restart it: ` and the rest of the line of R13 from
      `<error>` on

  They do not write again what timed out. `devices` and `listen` are unchanged

### The documents

- **R18** `docs/protocol.md` replaces its section "Hang after a USB reset" by a section on the hangs of the firmware
  that:
    - describes the reset hang with facts 1 to 3 of unit 0020 as they were observed, the report descriptor and the
      attributes of the USB device, and says that the suspend and resume cleared it once
    - describes the boot hang: its symptom, its trigger (fact 2 of this spec, with the plugging in at the GRUB menu),
      and the list of fact 3 of what did not clear it
    - says that the inference of unit 0020, that Linux resets the screen at boot, was wrong
    - describes the restart: the request of fact 4 with its fields, the string descriptors 0 to 3 that the screen has,
      the two observations of fact 4 with their date, and that the observation on a screen hung at boot was not clean
    - describes how sensordeck uses it, as R3 and R8 say, through the usbfs node, with usbhid bound, and with no USB
      reset and no file of sysfs written
    - lists as not established the questions of the context of this spec: index 32, index 33 or the count of stalls;
      a clean run on a screen hung at boot; whether the restart clears the reset hang; whether either hang happens on
      Windows
    - says that on Windows `HidD_GetIndexedString` sends the same request through the HID handle, and that sensordeck
      does not use it

  The list "Unknown" of `docs/protocol.md` keeps its items, with the one on Windows made about both hangs
- **R19** The section "Linux: access to the screen" of the README shows the three rules of the file as they are after
  R1 and R2; says that the third rule gives the user of the session access to the screen's USB device node, which
  `run` and the commands that write use to restart a hung screen, with a link to the section of R18; says that a rule
  file without that line is old and is installed again, and that a file with a line that sets `power/control` is the
  old rule of the version before; gives how to tell, which is that
  `grep 'SUBSYSTEM=="usb".*uaccess' /etc/udev/rules.d/70-sensordeck.rules` prints nothing for an old rule; and says
  that a replug applies the rule
- **R20** The README, where it describes how `run` behaves when the screen is unplugged and plugged in again, says that
  on Linux `run` restarts a hung screen as R8 and R9 say, with the lines it writes, and no longer says that it lets the
  screen suspend
- **R21** The entry of the section "Troubleshooting" of the README for a screen that shows its theme but no values
  after boot says that the screen hung because the firmware of the PC set it up before Linux, that `run` restarts it
  with the current udev rule, what the line of R11 or R12 means, and that unplugging it once recovers it. It does not
  say that a USB reset at boot causes the hang
- **R22** No document of the project claims that a suspend and resume recovers a hung screen in general, or that
  sensordeck relies on one

### Tests

- **R23** A test checks that `udev/70-sensordeck.rules` holds the line of R1 and its two rules for hidraw, and no line
  that holds `power/control` or `power/autosuspend_delay_ms`
- **R24** Unit tests of the restart, with a sysfs tree in a temporary directory laid out as the kernel lays it out,
  whose USB device holds `busnum` and `devnum`, a dev root in a temporary directory, and a fake in place of the opening
  of the usbfs node and of the ioctl, cover:
    - the path of the usbfs node from `busnum` and `devnum`, for a discovered node and for a `--device` node
    - the fields of each request, and the indexes sent in order from 4
    - the scan ending at a request that fails with `ENODEV`, at the USB device's directory removed, and at its
      `devnum` changed, each with the index given
    - stalls and other errors followed by the next index
    - the scan ending at index 255 and at the time limit of 10 seconds, with the time the test moves, as not restarted
    - the opening of the usbfs node failing, and the USB device not found, as not possible, with nothing sent
    - the wait for the screen: a new hidraw entry with another number, found by discovery, opened; and no entry within
      10 seconds, as restarted and not open again
    - a stop during the requests and during the wait
- **R25** Unit tests of `run`, with the fake transport whose writes time out as R6 says and a fake restart whose
  outcome the test sets, with the time the test moves, cover:
    - a timed-out clock write and a restart that makes the screen leave and open again: `screen hung: `,
      `restarting the screen to recover it`, `screen restarted at string descriptor <index>`, the theme check and the
      clock written, and `screen recovered`, with no `screen lost` and no `screen back`
    - a restart that did not make the screen leave: the line of R11 once over several rounds, the plain reopenings in
      between, and the next restart not before a minute has passed
    - a write that times out after a restart: the line of R12 once, and the next restart not before a minute
    - a restart that is not possible: the line of R13 once over several rounds, a restart tried in each round, and
      `screen recovered` once a write succeeds
    - a restart after which the screen is not open again: the rounds that open it until it opens, with no restart
    - a timed-out write of an upload: `theme upload failed: ` and then `screen hung: `
    - a write that fails with another error, which writes `screen lost: ` and `screen back` as today
    - ticks during a hang, which write nothing
    - none of the lines of R16
- **R26** Unit tests of `clock`, with a write that times out, cover the three texts of R17
- **R27** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1** The restart goes through the usbfs node of the USB device and not through hidraw. hidraw has no request for a
  string descriptor by index, only the ioctls for the names the kernel cached, and the request is addressed to the
  device and not to the interface, so usbfs passes it with usbhid bound and no interface claimed. Unbinding usbhid was
  rejected: it would need root and break the hidraw node that the rest of sensordeck uses
- **D2** The scan asks from index 4 upwards until the screen leaves the bus, rather than for one index. Which index, or
  which count of stalls, triggers the restart is not established, and a scan that goes on until the screen leaves
  works in each of the three cases. Index 4 is the first the screen does not have; asking for 0 to 3 would only be
  answered. The bound is 255 because wValue holds the index in one byte. The time limit of 10 seconds bounds a screen
  that answers each request only after the timeout; the stalls seen take far less, so the limit is not reached by a
  screen that behaves as observed
- **D3** The line of R9 names the index that the screen left at, so that the human's device check, and any later user's
  log, establishes the index that fact 4 leaves open, at no cost
- **D4** The suspend recovery of unit 0020 and its rule line are removed, rather than kept as a fallback. The restart
  is a restart of the firmware, as a replug is, and a replug clears both hangs; the request it rests on is answered by
  the control endpoint, which still works in both. So the restart is likely to clear the reset hang too, though that is
  not observed. The reset hang needs a USB reset of a running screen, which sensordeck never sends and the boot does
  not send (fact 2), so it is the rare one. Keeping the suspend as a fallback would keep two recoveries, two rule lines,
  two sets of lines and their tests for that rare case, and it would keep the screen autosuspending between commands,
  which has never been shown to be harmless. Should the reset hang be seen and not be cleared by the restart, the
  line of R12 says so and a replug clears it. Item 1 of "Beyond the request" keeps the fallback
- **D5** The rule grants the access by `uaccess`, as the hidraw rules do, so that the user of the active session gets
  it and no group needs to exist. Access to the usbfs node lets that user send any request to the screen, a USB reset
  included. The screen's hidraw node already lets the same user hang it and upload any theme to it, so this grants no
  power over another device and none over the system. The rule matches the USB device by `DEVTYPE` and the IDs with no
  `ACTION` match, as the rule of unit 0020 did, so a `change` event applies it as well as a replug
- **D6** The first restart of a hang is made at its first round, as the recovery of unit 0020 was, and later ones at
  most once a minute. A restart that did not help costs a write that times out (5 seconds) and a restart (under 2
  seconds) for nothing, and restarting the screen every 7 seconds is the cycling that this unit is reported for. The
  rounds in between open the screen and write as for a lost screen, so that a replug is taken up at once. A minute is
  short enough for a restart that failed by chance to be tried again soon, and long enough that a screen that cannot
  be restarted costs one attempt a minute. A restart that was not possible sent nothing, so it does not count, and a
  rule applied while `run` runs is used at the next round
- **D7** The wait for the screen to come back is 10 seconds, polled every 100 milliseconds, for an enumeration observed
  to take 1.3 seconds. The poll also covers the moment between the hidraw node appearing and udev granting access to
  it, in which an opening fails. A screen not back within 10 seconds is left to the reopening of later rounds, as a
  screen that was unplugged is
- **D8** The screen is found again by discovery, as every reopening of a discovered screen is. Discovery accepts only one
  supported screen, so the IDs find it, and a comparison of the serial would add nothing while sensordeck drives one
  screen. The hidraw number and the device number may change at the restart, which is why the usbfs path is read anew
  at each restart and the node is found anew
- **D9** The one-shot commands restart a hung screen and end with an error, rather than only advising as in unit 0020,
  or restarting and writing again. No suspend recovers the screen any more, so without the restart a user who does not
  run `run` would have to replug. They do not write again because `raw` is a tool of diagnosis whose bytes should be
  sent once and seen to be, an upload is recorded in the state file and is better started by the user than repeated
  behind them, and one rule for the four commands is simpler to read than four. The restart waits for the screen to come
  back, so the command run again works at once
- **D10** No end-to-end test covers the restart, for the reason D10 of unit 0020 gives: the fake screen is a pseudo
  terminal, no write to it fails with `ETIMEDOUT`, and no usbfs node of it takes the ioctl. The unit tests of R24 to R26
  cover the steps, the decisions and the lines, and the device check covers the rest
- **D11** This unit supersedes unit 0020 in its udev rule (R1), the conditions and the steps of its recovery and their
  lines (R3, R5 to R7 and R11), the texts of the commands (R13) and the documents (R15 to R18). Its R2, R4, R8 to R10,
  R12 and R14 hold in the form this spec gives them. The specs of units 0004 and 0020 are frozen and are not edited
- **D12** The unit has a plan, although it is a bug. It changes the udev rule, the Linux transport, the feeder of `run`,
  four commands, the tests in several packages and two documents, which is more than a couple of steps
- **D13** The device check, a cold boot with the new rule installed by the human with sudo and the screen recovering
  without a replug, with the index that `run` writes, is the human's, as `AGENTS.md` says. The loop does not wait for it

## Non-goals

- Windows. Whether either hang happens there is not known. `HidD_GetIndexedString` would send the same request through
  the HID handle, and is left for a unit of its own once a hang is seen on Windows. On Windows a timed-out write stays
  the screen lost
- Preventing the boot hang. Its trigger is the firmware of the PC (fact 2), which no software of the project reaches,
  and settings of the firmware of the PC or a hub with switched power are not answers for other users
- Establishing which index triggers the restart before this unit is built. The scan of R3 works whatever it is
- A screen given with `--device` whose hidraw node gets another number after the restart. It is reopened by its path,
  as after the reboot of an upload today, and a restart then ends as not open again
- Telling the user about an old rule before a hang happens, at the start of `run` or in `devices`
- sensordeck asking for root, or writing to sysfs
- Editing the frozen specs of units 0004, 0013, 0017 and 0020

## Beyond the request

1. Keep the suspend recovery of unit 0020 as a fallback for when the usbfs node cannot be opened, and keep its rule
   line. It costs the code, the lines and the tests of unit 0020 alongside the new ones, and the screen autosuspending
   between commands. Without it, a user with the rule of unit 0020 whose screen gets the rare reset hang sees the line
   of R13 and replugs once or installs the current rule
2. An amendment in the record of unit 0020 that says this unit supersedes it, with the requirements D11 names. It
   costs one list item made on the branch of this unit. Without it, a reader of unit 0020 finds the change only from
   this unit's `supersedes`
3. A hidden diagnostic command that makes a restart of a screen that works and writes the index it left at, so that
   the human can establish the index of fact 4 without waiting for a hang. It costs a command, its tests and its text.
   Without it, the line of R9 gives the index at the first hang that `run` restarts
