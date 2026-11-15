# The clock of run sent every second

## Context

Unit `0004-linux-sensors-and-run` made `sensordeck run` set the clock of the screen at the start, once a minute and
after a reopening, and unit `0013-theme-key-and-upload-on-change` added the clock after a completed upload. Decision D13
of unit 0004 gave the reason for the minute: the screen keeps time itself, so the minute only corrects drift. That
statement is in the comment of the clock period of `run` as well, and the README says that `run` sets the clock once a
minute.

The statement is false. The evidence D13 cites is the check on the device after unit 0003, and what that check showed,
as the context of unit 0004 records, is that the clock report sets the clock of the screen. It did not show that the
screen advances the time it was given.

### What was observed

On 2026-11-15 the human ran `sensordeck run` on the real screen with the built-in theme `minimal-dark`, and with a
variant of it, both of which show the time as `HH:mm:ss`. The values on the screen changed every second, at the default
interval of 1 second. The time shown changed only once a minute.

The orchestrating session then sent the time in the format the samples of the vendor use, with a leading space, such as
` 4:5:6`. It made no difference, so the format of the time is not the cause.

### Steps to reproduce

1. Put the theme `minimal-dark` on the screen with `sensordeck theme use minimal-dark` and a config without `interval`
2. Run `sensordeck run` and watch the time in the bar at the bottom of the screen
3. The numbers of the tiles change every second, and the time changes once a minute, each time to the current time

### Root cause

The screen shows the time of the last clock report it received and does not advance it. `run` sends the clock report at
the start, after a reopening, after an upload, and otherwise only in the first round that starts at least 60 seconds
after the last one, so the time on the screen moves once a minute.

This is read from the observation on the device, where the time moved exactly as often as `run` sent the clock, and has
not been reproduced in the sandbox, which has no screen. Whether one of the bytes of the clock report whose meaning is
unknown, `0x01 0x15` at offsets 1 and 2 or `0x64` at offset 10, would make the screen advance its time is not known and
was not tried. The fix below does not depend on it.

### The fix

`run` sends the clock once a second, at the start of each whole second of the time, whatever the interval of its
rounds. The rounds keep their moments and what they do. Between them, `run` wakes at each whole second and writes only
the clock report.

## Requirements

### When the clock is written

- **R1** Besides its rounds, `run` has a tick at each whole second of the time. After a round or a tick, the next round
  or tick starts at the earlier of two moments: the next place of a round, which R21 of unit 0004 gives as it does
  today, and the first whole second after the start of the round or tick that ended. A moment that has already passed
  is followed at once, and whole seconds that passed during a round or a tick are not made up. A round and a tick that
  fall at the same moment are one round
- **R2** A round or a tick writes the clock report of `EncodeClock` for the current local time when the screen is open,
  the theme of the config is on the screen, and either the clock is due or the whole second of the time at the start of
  that round or tick differs from the whole second of the last clock report written. The clock is due, as today, at
  the start, after the screen came back and after a completed upload. A round writes the clock before its value reports
- **R3** A tick does nothing but R2. It does not read the config file again, does not reopen a lost screen, does not
  check or upload the theme, reads no source and no controller, and writes no value report. While the screen is lost,
  or while the theme of the config is not on the screen or a check of it is due, a tick writes nothing. A failed write
  of a tick is the screen lost, with the line of R22 of unit 0004, and the next round reopens it
- **R4** The rounds keep everything R19 to R22 of unit 0004 and the later units give them, but for the moments of the
  clock: their moments on the grid of the interval, the reload of the config, the reopening, the check and the upload
  of the theme, the readings of the sources and of the controller, the value reports, and the lines on standard output
  and standard error. In particular the clock is not written while the theme of the config is not on the screen, as R9
  of unit 0013 says
- **R5** The command `clock` is unchanged

### What the project says

- **R6** No comment, document or test of the project says that the screen keeps time itself, or that `run` sets the
  clock once a minute. The comment of the code that schedules the clock says that the screen shows the time of the last
  clock report and does not advance it
- **R7** The README says that `run` sets the clock of the screen when it starts and once a second
- **R8** The section on the clock of `docs/protocol.md` says that the screen shows the time of the last clock report it
  received and does not advance it, as seen on the device on 2026-11-15, so that a program sends the clock once a second
  for the seconds to change
- **R9** The comment at the top of the end-to-end tests of the commands, which lists what they leave out, no longer names
  the clock of `run` after 60 seconds

### Tests

- **R10** Unit tests of `run`, with the fake transport and the time the test moves, cover:
    - an interval of 1 second that starts on a whole second: every round writes the clock and then the values, and
      there is no tick apart from the rounds
    - an interval of 5 seconds: a clock report at each whole second, with consecutive seconds, and value reports only
      in the rounds 5 seconds apart, each after the clock of its second
    - an interval of 300 milliseconds: one clock report at each whole second, written at that second, and value reports
      every 300 milliseconds, with no clock in a round that starts within a second whose clock was written
    - a start at 400 milliseconds after a whole second with an interval of 1 second: the clock at the start, the next
      clock at the next whole second in a tick that writes nothing else, and rounds at 400 milliseconds after each
      second that write values and no clock
    - a round that runs for 2.5 seconds: the next round follows at once and writes the clock of its own second, and the
      seconds that passed during the long round get no clock report
    - a tick while the screen is lost, which writes nothing and reopens nothing, and the clock in the round whose
      reopening succeeds, even within a second whose clock was written before the loss
    - a tick while the theme of the config is not on the screen, which writes nothing and uploads nothing
    - a failed write of the clock in a tick, which marks the screen lost with the line of R22 of unit 0004
- **R11** An end-to-end test runs `run` with `interval: 3s` against the fake screen and checks that the first four
  reports are the clock, the values, and two more clock reports, whose times are one and two seconds after the whole
  second of the first
- **R12** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1** The clock is sent once a second, on its own schedule, rather than in every round. In every round alone, a user
  with `interval: 5s` would see the seconds jump by five, and one with `interval: 100ms` would send ten clock reports a
  second for a time the screen shows in whole seconds. A clock with seconds needs one report a second, whatever the
  interval
- **D2** Between rounds `run` wakes for a tick that writes only the clock, rather than making every wake-up a full round
  with the values left out. A full round each second would read the config file again, try a reopening and check the
  theme each second, which changes the cadence that units 0004, 0008 and 0013 decided and the README describes, and
  makes the word round mean two things. The tick changes only the clock. The price is that the tick needs to know
  whether the screen is open and the theme is on it without doing the work of a round, and that a lost screen still
  waits for the next round to be reopened, as it does today
- **D3** The ticks are on the whole seconds of the time, rather than on a grid of one second from the start. A clock
  report holds whole seconds and the screen shows the last one, so a report written just after the second changed keeps
  the screen at most a few milliseconds behind. A grid from the start would put the writes at any fraction of the
  second, which leaves the screen up to a second behind and, when the writes fall close to the change of a second,
  lets a few milliseconds of delay make the screen skip or repeat a second. The seconds are those of the time of the
  system, as the clock report is: a wait to the next whole second is never longer than a second, so a step of the time
  of the system costs at most one tick
- **D4** A clock report is due when the whole second differs from that of the last one, not when it is later. When the
  time of the system is set back, the screen follows at once instead of standing still until the time catches up
- **D5** The clock stays due at the start, after the screen came back and after an upload, even within a second whose
  clock was already written. The second half of D13 of unit 0004 still holds: after a reopening or an upload the screen
  may have rebooted and lost its time
- **D6** There is no key in the config for the period of the clock. One report of 64 bytes a second is no load on the
  screen or the bus, and a period longer than a second only makes a clock with seconds wrong
- **D7** This unit supersedes unit `0004-linux-sensors-and-run` in its statements on the clock of `run`: R20, D13 and the
  case of R32 that tests the clock again after 60 seconds and not before. The tests of unit 0004 that pin the minute,
  the clock again after sixty seconds and the clock measured by the start of a round, are replaced by those of R10. The
  context of unit 0013, which describes the feeder as sending the clock once a minute, is superseded in that statement,
  and its requirements stand. Unit 0008 compares the measuring of the controller's period with that of the clock of R20,
  which stays a correct account of how its own period is measured. The specs of these units are frozen and are not
  edited
- **D8** The unit has a plan, although it is a bug. The fix changes the schedule of the feeder, adds to the theme holder
  a way to tell whether the theme is on the screen, changes the expected reports of existing tests in several files,
  adds an end-to-end test and corrects three documents, which is more than a couple of steps, and the order of the
  tests before the change is worth agreeing
- **D9** The check on the device, that the seconds on the screen change every second, is the human's, as `AGENTS.md`
  says. The unit does not wait for it: R10 and R11 check what can be checked without the screen

## Non-goals

- Finding out whether a byte of the clock report makes the screen advance its own time. That needs experiments on the
  device, and once a second works whatever the answer
- Sending the clock less often for a theme that shows no seconds. Every theme is sent the clock once a second
- Any change to the moments of the rounds, to the interval, or to what a round reads and sends
- Any change to the command `clock`, which sets the clock once
- Editing the specs or plans of units 0004, 0008 and 0013, which are frozen
- The Windows transport, which unit 0016 is about. The schedule of the clock is the same code on both systems

## Beyond the request

1. An amendment in the record of unit 0004 that says D13 and R20 are superseded by this unit. It costs one list item,
   made on the branch of this unit. Without it, a reader of unit 0004 learns that D13 is false only by finding this
   unit, whose `supersedes` points back to 0004 and not forward from it
2. Starting the grid of the rounds on a whole second, so that with an interval of whole seconds a round and a tick fall
   at the same moment and `run` wakes once a second instead of twice. It costs a first round up to a second later, and
   changes the moments of the rounds that units 0004 and 0008 decided. Without it, `run` wakes twice a second with the
   default interval and writes the same reports, which no one sees
3. Sending the clock once a minute when the theme of the config shows no seconds. It costs a reading of the formats of
   the date and time widgets of the compiled theme, and has no answer for a config with `tags`, whose theme is unknown.
   Without it, such a theme is sent 59 reports a minute it does not need, which no one sees
