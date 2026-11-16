# Recovering a screen whose firmware hung after a USB reset

## Context

On Linux the screen is reached through its hidraw node, and `sensordeck run` treats a failed write as the screen lost:
it writes `screen lost: ` and the error, and reopens the node in the next round, which writes `screen back` (R22 of
unit `0004-linux-sensors-and-run`). The screen is bus powered: it is off while the PC is off and comes on with it.

### What was observed

After a normal cold boot of the human's Linux PC, the screen shows its theme but no values, with its default time
12:00:00 and date 2022-01-01. Every write to its hidraw node fails after the kernel's limit of 5 seconds with
`write /dev/hidraw3: connection timed out`. `run` writes `screen lost: ...` and `screen back` every 5 seconds or so,
for as long as it runs. Only unplugging the screen and plugging it in again made it work.

The orchestrating session established these facts on the human's machine on 2026-11-16, each observed on the device:

1. **The firmware hangs on a USB reset while it runs.** A `USBDEVFS_RESET` ioctl on the usbfs node of a screen that
   worked made every later write time out at once. Disabling and enabling its port of the root hub, by writing 1 and
   then 0 to `usb1-port4/disable`, did the same. At boot the firmware sets the device up and Linux resets it once
   more, which explains the hang at every boot; after a replug only Linux enumerates the fresh device, and it works
2. **The hung state.** Writes of output reports to the interrupt OUT endpoint time out. `HIDIOCSOUTPUT`, the output
   report by the control endpoint, times out too. `HIDIOCSFEATURE` is accepted but does nothing, because the report
   descriptor, `06 00 ff 09 01 a1 01 09 01 15 00 26 ff 00 95 40 75 08 81 02 09 01 15 00 26 ff 00 95 40 75 08 91 02 c0`,
   declares one input and one output report of 64 bytes and no feature report. No input report arrives. A further USB
   reset does not help. The disable switch of the port does not cut its power, the screen stayed lit, and the port has
   no ACPI power resources
3. **A USB runtime suspend and resume recovers it.** With the service stopped, so that no program held the hidraw node
   open, `power/control` of the screen's USB device was set to `auto`; its value there was `on`, with
   `power/autosuspend_delay_ms` 2000. `power/runtime_status` read `suspended` within 5 seconds. Setting `control` back
   to `on` resumed the device, and the clock and the values worked again at once. This was observed once, on a screen
   hung by the reset of fact 1

The USB device of the screen on that machine has `bmAttributes` `0x80`, bus powered without remote wakeup, and no
`power/wakeup` file, read on 2026-11-16.

### Steps to reproduce

1. Install the current udev rule, which does not touch `power/control`, and the service of unit 0014
2. Switch the PC off and on again, with the screen plugged in
3. The screen shows its theme with 12:00:00 and 2022-01-01, and `journalctl --user -u sensordeck` shows `screen lost:
   write /dev/hidraw<n>: connection timed out` and `screen back` alternating every few seconds

Alternatively, on a screen that works, a `USBDEVFS_RESET` ioctl on its usbfs node, with root, gives the same state at
once. sensordeck itself never sends one.

### Root cause

The firmware of the screen stops serving its OUT endpoint, and its control endpoint for output reports, when it gets a
USB reset while it is already running, which happens at every boot because the firmware of the PC and then Linux reset
it. The reopening of `run` does not help, because the node opens fine and the device stays hung. This is read from the
observations above; the boot sequence as the cause of the reset at boot is an inference from facts 1 and from the
replug that works, not an observation of the boot itself.

### The fix

The kernel suspends a USB device whose `power/control` is `auto` once no driver holds it busy for its autosuspend
delay, and resumes it when it is opened. Fact 3 shows that a suspend and resume clears the hang. So:

* The udev rule of the project sets `power/control` to `auto`, and the autosuspend delay, for the screen's USB device.
  A screen hung at boot then suspends by itself before `run` first opens it, because the service starts at login and
  no program holds the node open before, and the opening of `run` resumes it
* `run` recovers a screen that hangs while it holds the node open, or before it could suspend: after a write times
  out it closes the node, waits until the device is suspended, and opens it again, which resumes it
* Where the rule is missing or old, sensordeck says what to do, once, in words, instead of writing `screen lost` and
  `screen back` forever

How usbhid keeps an open device resumed: on opening, usbhid marks the interface as needing remote wakeup, and the USB
core refuses to autosuspend a device whose interface needs remote wakeup when the device cannot wake the host. The
screen cannot (its `bmAttributes` has no remote wakeup bit), so it is not suspended while a program holds its hidraw
node open, whatever `power/control` says, and `run`, which holds the node for as long as it runs, is not interrupted by
the autosuspend. This is read from the behaviour of the kernel's USB power management and the attributes of the device,
and is not yet observed under `run` with `power/control` at `auto`.

## Requirements

### The udev rule

- **R1** `udev/70-sensordeck.rules` keeps its two rules for hidraw unchanged, and holds a third rule, with a comment
  above it that says why it is there, which is this line:
  `SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTR{idVendor}=="0483", ATTR{idProduct}=="0065", ATTR{power/autosuspend_delay_ms}="2000", ATTR{power/control}="auto"`

### What a timed-out write is

- **R2** On Linux a write to the screen has timed out when its error is, or wraps, the error of the system
  `ETIMEDOUT`. Any other failure of a write is not a timeout, and on Windows no failure is
- **R3** The USB device of the screen is the USB device in sysfs to which the hidraw entry of the open node belongs,
  under the sysfs root that sensordeck reads (`SENSORDECK_SYSFS_ROOT`, or `/sys`). It is found the same way for a node
  found by discovery and for a node given with `--device`. The recovery of R5 is possible only when its
  `power/control` reads `auto` and its `power/autosuspend_delay_ms` reads a whole number of 0 or more. `run` reads them
  anew at each round of a hang, and a command when its write timed out, so that a rule applied while `run` runs is
  used. When the USB device is not found, or a
  file cannot be read, the recovery is not possible

### What `run` does

A hang is the time from a write of `run` that timed out to the first write of `run` that succeeds after it. The writes
of `run` are those of the clock, of the values and of an upload of the theme.

- **R4** A write of `run` that times out writes `screen hung: ` and the text of the error to standard error, in place
  of the line `screen lost: ` of R22 of unit 0004, but only for the first timed-out write of a hang. The rest of the
  round is as for a lost screen: no further report of that round is written, and the ticks of unit 0017 write nothing
  until a round has opened the screen again. A timed-out write of an upload writes the line `theme upload failed: `
  that it writes today before the line `screen hung: `
- **R5** At the time of every round during a hang, `run` does instead of the round, when the recovery is possible:
  close the node; read `power/runtime_status` of the USB device every 100 milliseconds until it reads `suspended`, or
  until the limit has passed, which is the autosuspend delay read for R3 plus 5 seconds; then open the node again the
  way it was opened at the start, which resumes the device. The first such recovery of a hang writes
  `suspending the screen to recover it` to standard error before it closes the node
- **R6** When the limit of R5 passes without `suspended`, `run` writes, once in a hang,
  `the screen did not suspend within <limit>, another program may hold it open` to standard error, with the limit
  written as Go writes a duration, and opens the node again all the same
- **R7** When the recovery is not possible, the first round of the hang writes this line to standard error, once in a
  hang: `the screen has hung. Installing the current udev rule lets sensordeck recover it, see "Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen. Unplugging the screen once recovers it now`.
  That round and the later ones of the hang reopen the screen as R22 of unit 0004 does today
- **R8** A round that opened the node during a hang goes on as after a screen that came back, without the line
  `screen back`: the clock is due, the theme check of R7 of unit 0013 is due, and the round then writes as any round
  does. A failed opening writes no line, and the next round tries again as R5 or R7 says
- **R9** The first write of `run` that succeeds during a hang ends it and writes `screen recovered` to standard error.
  A later timed-out write begins a new hang, with its lines once again
- **R10** During a hang, `run` writes no line `screen lost: ` or `screen back`. A write that fails for another reason
  during a hang is the screen lost, as R22 of unit 0004 says, and ends the hang without a line `screen recovered`
- **R11** A stop by SIGINT or SIGTERM during the wait of R5 ends the wait at once, and `run` stops as R23 of unit 0004
  says
- **R12** A write that fails without a timeout, outside a hang, is handled as today, with `screen lost: ` and
  `screen back`

### What the other commands do

- **R13** `clock`, `values`, `upload` and `raw`, whose write times out, end with an error whose text is the text of the
  error of the write, followed by `: the screen has hung. ` and then:
    - when the recovery of R3 is possible: `It recovers by itself once no program has held it open for a few seconds, so run the command again`
    - otherwise the text of R7 after `the screen has hung. `

  They do not close, wait and reopen themselves. `devices` is unchanged
- **R14** sensordeck never sends a USB reset, never writes to the `disable` file of a port or to `authorized`, never
  unbinds a driver, and writes no file in sysfs. The recovery is the closing and the opening of the hidraw node alone

### The documents

- **R15** `docs/protocol.md` has a section on the hang of the firmware and its recovery that states facts 1 to 3 of
  this spec, with the date 2026-11-16, the report descriptor, and the attributes of the USB device; says that fact 3
  was observed once, and that the cause of the reset at boot is an inference; says that the screen is not suspended
  while a program holds its node open because it cannot wake the host, as read and not observed; and lists as unknown
  whether a suspend and resume of a screen that works is harmless, what the screen shows while suspended, and whether
  the hang happens on Windows
- **R16** The section "Linux: access to the screen" of the README shows the whole rule file, or the three rules,
  instead of saying the rule is one line; says what the third rule does and why, in a few sentences that point to the
  section of R15; and says that a rule file without the line that sets `power/control` is old and must be installed
  again, with how to tell, such as `grep power/control /etc/udev/rules.d/70-sensordeck.rules`, and that a replug
  applies it
- **R17** The README has a section "Troubleshooting" with an entry for a screen that shows its theme but no values
  after boot, with 12:00:00 and 2022-01-01 and `screen hung` in the log: that the screen hung after the USB reset of
  the boot, that installing the current udev rule lets sensordeck recover it, and that unplugging it once recovers it
  now
- **R18** The README says, where it describes how `run` behaves when the screen is unplugged and plugged in again, that
  on Linux `run` recovers a hung screen as R5 says, with the lines it writes

### Tests

- **R19** A test checks that `udev/70-sensordeck.rules` holds the line of R1, and its two rules for hidraw
- **R20** Unit tests of the Linux side, with a sysfs tree in a temporary directory laid out as the kernel lays it out,
  a class entry `hidraw<n>` that leads to a USB device holding `power/control`, `power/autosuspend_delay_ms` and
  `power/runtime_status`, cover: the USB device found for a discovered node and for a `--device` node; the recovery
  possible with `auto` and `2000`, and not possible with `on`, with `-1`, with a missing file and with no USB device;
  and the classification of R2, where an error of `os.File.Write` that wraps `ETIMEDOUT` is a timeout and one that
  wraps `EIO` is not
- **R21** Unit tests of `run`, with the fake transport whose writes time out as R2 says and a recovery whose sysfs
  files the test sets, with the time the test moves, cover:
    - a timed-out clock write with the recovery possible: `screen hung: `, `suspending the screen to recover it`, the
      node closed, the status read until the test makes it `suspended`, the node opened again, the theme check and the
      clock written, and `screen recovered`, with no `screen lost` and no `screen back`
    - the limit of R5 passing: the line of R6, the node opened all the same
    - a recovery that does not help: the second timed-out write writes no second `screen hung` and no second
      `suspending` line
    - the recovery not possible: the line of R7 once over several rounds, plain reopenings, and `screen recovered` once
      a write succeeds
    - a timed-out write of an upload: `theme upload failed: ` and then `screen hung: `
    - a stop during the wait of R5, which ends `run` without waiting for the limit
    - a write that fails with another error, which writes `screen lost: ` and `screen back` as today
    - ticks during a hang, which write nothing
- **R22** Unit tests of `clock`, with a write that times out, cover the two texts of R13
- **R23** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1** The first timed-out write already counts as a hang, rather than a count of several in a row. A screen that
  works has never been seen to time out, each timed-out write costs 5 seconds, and a recovery costs at most a few
  seconds of a screen that shows nothing new. Waiting for three would add 10 seconds to every recovery and protect
  against a case nobody has seen. Should a screen that works ever time out once, the cost is one needless suspend
- **D2** The rule sets `autosuspend_delay_ms` to 2000 explicitly rather than leaving the kernel's default. The default
  comes from the parameter `usbcore.autosuspend`, which some systems set to -1 to turn autosuspend off, and the recovery
  would then never see `suspended`. 2000 is the value under which fact 3 was observed. A shorter delay would recover a
  little faster but suspends the screen sooner between two commands, which is not observed to be harmless; a longer one
  only makes the recovery slower. The delay is set before `control` in the rule, so that the device never runs on
  `auto` with another delay. The delay costs nothing while `run` runs, because the open node keeps the device resumed
- **D3** The rule matches the USB device itself, by `DEVTYPE` and the IDs, and has no `ACTION` match. The attributes
  belong to the USB device and not to the hidraw node or the interface. Without an `ACTION` match a `change` event, such
  as the one of `udevadm trigger`, applies it as well as a replug, and writing the same values again is harmless
- **D4** The rule does the setting and sensordeck only reads, rather than sensordeck writing `power/control` itself.
  The files of sysfs belong to root, and sensordeck runs as the user. This is why an old rule cannot be repaired by
  sensordeck and R7 asks for it to be installed again
- **D5** The recovery is a closing, a wait for `suspended` and an opening, not a USB reset or a cut of the port's
  power. Fact 1 shows that a reset is the cause and fact 2 that it does not help, and the disable switch of the port
  cuts no power on the machine where it was tried. The opening resumes the device by the same runtime resume as setting
  `control` to `on` did in fact 3, which is the kernel's doing and not observed through the opening itself
- **D6** The recovery is made at the time of the next round, as the reopening of a lost screen is today, rather than at
  once inside the write. The round is where `run` already reopens, holds the stop and writes the lines, and the wait of
  at least the autosuspend delay is no worse a round later
- **D7** The limit of the wait is the autosuspend delay plus 5 seconds, read each time, rather than a fixed limit. Fact 3
  saw `suspended` within 5 seconds with a delay of 2 seconds, and a user who sets another delay gets a limit that fits
  it. After the limit `run` opens the node all the same, because another program that holds it open may close it, and
  the line of R6 says what is likely wrong
- **D8** The lines of a hang are written once in each hang, and a hang ends only at a write that succeeds. The report
  was of `screen lost` and `screen back` every 5 seconds forever, which says nothing that the first line did not say. A
  reopening that succeeds is not a recovery, since the node of a hung screen opens fine, so `screen recovered` waits for
  a write that succeeds, and `screen back` is not written during a hang. The line of R7 is per hang and not per run
  because a hang is rare and a second one is news
- **D9** The one-shot commands report a hang and do not recover it. With the rule installed, a screen that hangs while
  no program holds it open suspends by itself within the delay, and the next command's opening resumes it, so a
  recovery in the command would only repeat what the next run of the command gets anyway. Their text says so. This
  follows from fact 3, where the recovery happened with no program holding the node, and is not observed for the
  commands themselves. `devices` writes nothing, so it cannot see a hang, and stays as it is
- **D10** No end-to-end test covers the recovery. The fake screen is a pseudo terminal, and no write to a terminal fails
  with `ETIMEDOUT`. Making one fail needs a time limit on the write in the transport, which on the device never fires,
  because the kernel blocks a write to a hidraw node until the device answers or its own 5 seconds pass whatever the
  file's mode: a branch that only the tests would reach. The unit tests of R20 to R22 cover the decision and the lines,
  and the device check covers the rest
- **D11** This unit supersedes unit 0004 in R22 for a write that times out on Linux, as R4, R8 and R10 say. R22 holds
  for every other failure. The spec of unit 0004 is frozen and is not edited
- **D12** The unit has a plan, although it is a bug. It changes the udev rule, the Linux transport's knowledge of sysfs,
  the feeder of `run`, the errors of four commands, the tests in several packages and two documents, which is more than
  a couple of steps
- **D13** The check on the device, that the screen shows values after a cold boot with the new rule installed by the
  human with sudo, and that `run` recovers a screen hung by a reset while it runs, is the human's, as `AGENTS.md` says.
  The loop does not wait for it

## Non-goals

- Windows. Whether the screen hangs the same way there is not known, and the recovery uses Linux runtime power
  management. On Windows a timed-out write stays the screen lost
- Preventing the reset at boot, or the hang itself. Both are in the firmware of the PC and of the screen
- A hang after the system wakes from sleep. It has not been observed; should it happen, the same symptom gets the same
  recovery
- Telling the user about an old rule before a hang happens, at the start of `run` or in `devices`
- sensordeck writing to sysfs, or asking for root
- Finding a hang without a write, such as by reads. `run` writes every second
- Editing the frozen specs of units 0004, 0013 and 0017

## Beyond the request

1. An end-to-end test of the recovery, by a write time limit in the transport that a fake screen which stops reading
   its terminal would reach. It costs a branch of the transport that only the tests reach, as D10 says, and a test that
   waits several seconds. Without it, the unit tests cover the recovery and the device check confirms it
2. A line at the start of `run` on Linux when the screen's `power/control` is not `auto`, saying that the udev rule is
   old. It costs one read of sysfs and one line at each start for the users without the rule. Without it, a user with
   an old rule learns of it at the first hang, from the line of R7, and unplugging recovers the screen meanwhile
3. An amendment in the record of unit 0004 that says R22 is superseded by this unit for a write that times out. It
   costs one list item made on the branch of this unit. Without it, a reader of unit 0004 finds the change only from
   this unit's `supersedes`
