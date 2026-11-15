# Reading the Commander Core XT less often in run

## Context

Unit `0005-commander-core-xt-probes` added the temperature probes of the Corsair iCUE Commander Core XT as the metrics
`commander.<n>.temp`, and unit `0006-commander-answer-size` made them readable on the human's controller. Every reading
of the controller enters software mode and returns to hardware mode at once, by D2 of unit 0005. On the device, the
human saw that the LEDs of the fans on the controller flicker once at each reading: in software mode nothing drives the
LEDs, and the return to hardware mode lights them again. The status command of liquidctl makes them flicker once in the
same way. With `sensordeck run` at its interval of 1 second and a probe mapped, the LEDs flicker every second.

The orchestrating session tried on the device to read the temperatures without software mode: it opened the endpoint
`0x21` and read it, the steps 5 and 6 of unit 0005, with no step 1 before them. The controller answered every command
with byte 2 `0x03` and stale data in the rest of the answer. A reading without software mode is therefore not possible
with what is known of the controller. The same trial confirms the assumption of D4 of unit 0005 that byte 2 of an
answer is a status, and that an answer whose byte 2 is not `0x00` is an error: the checks of R6 of unit 0005 reject
those answers, as they should.

Until a later unit drives the LEDs while the controller is in software mode, the human chose to read the probes
rarely, and chose once a minute. This unit makes `run` read the controller at most once per period, of 1 minute unless
the config says otherwise, and send the readings of the last reading of the controller in the rounds between. The
probes of a case change slowly, so a reading a minute old is still a true picture of them. `sensordeck sensors` reads
once and stays as it is.

The licensing rule of unit 0005 still holds: no code of liquidctl is read, translated or copied. The facts above come
from the human's observation on the device and from the orchestrating session's trial on it.

The unit is built in the sandbox, which has no controller and no screen. Every requirement is checked by automated
tests with a time the test moves, sources the test gives, the fake screen and the fake controller. The check on the
real device is the human's.

## Requirements

"R*n* of unit 000*k*" names a requirement of the spec of that unit. "The controller", "a reading of the controller"
and "the metrics of the controller" mean what they mean in unit 0005: a reading of the controller is one call of its
source, which runs the sequence of R5 of unit 0005 when a metric of the controller is wanted, and the metrics of the
controller are the `commander.<n>.temp`. "The period" is the time of R1. "The kept readings" are those of R4. Every
requirement of units 0004 to 0007 that this spec does not change still holds.

### The config file

- **R1** The config file of R17 of unit 0004 has a third key, `commander_interval`, the shortest time between two
  readings of the controller in `run`, in the syntax of Go's `time.ParseDuration`, such as `1m` or `30s`. It must be
  at least the interval of `run`, the one the file gives or the default of 1 second. Without the key the period is
  1 minute, or the interval when the interval is longer than 1 minute. For example:

  ```yaml
  interval: 1s
  commander_interval: 1m
  tags:
    cpu.temp: 1
    commander.1.temp: 18
  ```

- **R2** The checks of R18 of unit 0004 gain two problems, with these texts after the prefix `config file <path>: `:
    - a `commander_interval` that is no duration, including one that is not a YAML scalar:
      `commander_interval: "<text>" is not a duration such as 1m or 30s`, with the text as written in the file and the
      empty text for a value that is not a scalar
    - a `commander_interval` shorter than the interval: `commander_interval: <text> is shorter than the interval
      <interval>`, with the text as written in the file and the interval as Go's `time.Duration` writes it, such as
      `1s` or `100ms`

  A `commander_interval` equal to the interval is valid. When the file has several problems one of them is reported,
  as R18 of unit 0004 says
- **R3** Two texts of R18 of unit 0004 name the new key, so that they list the three keys in the order of Go's string
  comparison:
    - a file whose content is not a mapping: `want a mapping with the keys commander_interval, interval and tags`
    - a key other than the three: `unknown key "<key>", the keys are commander_interval, interval and tags`

  Every other text of R18 of unit 0004 stays as it is

### The command `run`

- **R4** The first round of `run` makes a reading of the controller. A later round makes one when it starts at least
  the period after the start of the last round that made one, measured on the same clock as the rounds and the clock
  report of R20 of unit 0004. Every other round makes no reading of the controller and writes nothing to
  it. A reading of the controller asks for the mapped metrics of the controller as R21 of unit 0004 says, and what it
  gives replaces the kept readings as a whole: the kept readings are exactly the readings of the last reading of the
  controller
- **R5** Every round, whether or not it made a reading of the controller, sends a pair for each mapped metric of the
  controller that has a kept reading, with the value of the kept reading converted by R4 of unit 0004, ordered by tag
  together with the pairs of the other metrics, as R21 of unit 0004 says. A mapped metric of the controller without a
  kept reading is left out of the round, as an unavailable metric is
- **R6** A reading of the controller that gives no reading of a mapped metric, because the controller is missing, its
  node cannot be opened, or the sequence failed by R6 of unit 0005, or that gives a reading of some mapped metrics
  only, still counts as a reading for R4. The metrics it did not give are left out of every round until the next
  reading of the controller, which comes a full period later and not at the next round
- **R7** Every source other than that of the controller is read in every round as R21 of unit 0004 says. The period
  changes nothing of the moments of the rounds, of the clock of R20 of unit 0004, or of what `run` writes to standard
  output and standard error; the line `running: <count> metrics every <interval>` of R19 of unit 0004 stays as it is
- **R8** While the screen is lost, by R22 of unit 0004, the rounds in which `run` calls `Reopen` read no source, so
  they make no reading of the controller. The round that follows a reopening that succeeds makes a reading of the
  controller only when R4 says so, and sends the kept readings by R5 otherwise. The kept readings and the start of the
  last round that made a reading of the controller are kept for the whole run, across a lost screen
- **R9** Nothing of the reading of the controller changes: the discovery, the eight commands, their order, the time
  limit, the sizes of the reports, the checks of an answer, the return to hardware mode after a failure or a stop, and
  the stop of R8 of unit 0005 stay as units 0005 and 0006 say. A reading of the controller still runs the sequence
  once, R10 of unit 0005

### The command `sensors`

- **R10** `sensordeck sensors` is unchanged: it makes one reading of every source, the controller among them, as R15
  of unit 0004 and R12 and R22 of unit 0005 say, and reads no config

### The documents

- **R11** The section "Showing sensor values on the screen" of `README.md` says, after the paragraph on `interval`:
  that `commander_interval` is the shortest time between two readings of the Commander Core XT, of at least the
  interval and of 1 minute without the key; that each reading makes the LEDs of the fans on the controller flicker
  once, which is why it is read less often; that between two readings `run` sends the values of the last reading; and
  that `sensordeck sensors` reads the controller once each time it runs
- **R12** `docs/commander-core-xt.md` states, in the project's own words:
    - that on the device the LEDs of the fans on the controller flicker once at each reading, because nothing drives
      them in software mode, and that the status command of the other tool makes them flicker in the same way
    - that the endpoint `0x21` was opened and read without step 1 before it, and that every answer then had byte 2
      `0x03` and stale data in the rest, so that a reading without software mode is not possible with what is known
    - that byte 2 of an answer is a status, `0x00` in the answers of a sequence that step 1 opened and `0x03` for the
      commands made without it
    - that `sensordeck run` reads the controller at most once per period, of 1 minute unless the config says otherwise,
      to make the LEDs flicker less often

  The bullet of "Unknown" on the meaning of byte 2 and the bullet of "Assumptions" that byte 2 is a status are
  corrected to match, and the bullet of "Unknown" on what the controller does in software mode is corrected to say
  that it does not drive the LEDs, while what it does with the fans stays unknown. Nothing else of the document changes

### Tests

- **R13** Unit tests of the config cover: a `commander_interval` that is given, with its value; the default of
  1 minute without the key; the default with an `interval` of 2 minutes, which is 2 minutes; a `commander_interval`
  equal to the interval, which is valid; a `commander_interval` shorter than an `interval` the file gives and shorter
  than the default interval, each with the text of R2; one that is no duration and one that is a mapping, each with
  the text of R2; and the two texts of R3. The existing tests of the texts of R18 of unit 0004 are changed to the texts
  of R3, and the others stay
- **R14** Unit tests of `run`, with a fake transport, a time the test moves and sources the test gives, one of them
  standing for the controller and counting the calls it gets, cover:
    - with an interval of 1 second and no `commander_interval`: the controller is read in the first round, not in the
      rounds up to 59 seconds after it, again in the round at 60 seconds, and three times over the rounds of the first
      121 seconds
    - between two readings of the controller, every value report holds the pairs of the kept readings, while a source
      whose value changes in every round is read in every round and sends its new value
    - a reading of the controller that gives nothing: no pair of the controller in that round and in the rounds up to
      the next reading, and no call of the controller before the period has passed
    - a reading that gives one of two mapped probes: only its pair is sent until the next reading
    - a next reading that gives no probe after one that gave both: the pairs of the controller are left out from then on
    - a `commander_interval` equal to the interval: the controller is read in every round
    - a round that starts late, after a long round: the period is measured from the start of the round that read the
      controller, as the clock of R20 of unit 0004 is
    - a lost screen and a reopening that succeeds before the period has passed: the round after the reopening sends
      the kept readings without reading the controller
- **R15** An end-to-end test runs `run` against the fake screen, the tree of R35 of unit 0004 and the fake controller
  with a config of `interval: 100ms`, `commander_interval: 100ms` and `tags` of `cpu.temp: 1` and
  `commander.1.temp: 4`. Once a value report holding (4, 37) has arrived, the test sets the first probe of the fake
  to 40.0 °C, and a value report holding (4, 40) arrives within 3 seconds. The fake has then received step 1 at least
  twice. After SIGTERM the process ends with status 0 and the fake is in hardware mode
- **R16** An end-to-end test runs `run` as R15 says, with `commander_interval: 1h`. Once a value report holding (1, 42)
  and (4, 37) has arrived, the test sets the first probe of the fake to 40.0 °C and replaces the input of `Tctl` at
  once, through a new file renamed into place, by 43000. A value report holding (1, 43) and (4, 37) arrives within
  3 seconds, and the three value reports after it each hold (4, 37). The fake has then received exactly the eight
  commands of one reading. After SIGTERM the process ends with status 0 and the fake is in hardware mode
- **R17** The existing unit tests and end-to-end tests of the suite pass, with no change of what they expect beyond
  the texts of R3. The end-to-end test of R21 of unit 0005 keeps its config and its result, because its first round
  reads the controller

## Decisions

- **D1** The controller is read at most once per period, of 1 minute by default, the choice of the human. Reading
  without software mode was tried on the device and does not work, as the context says, and keeping the controller
  in software mode between readings is rejected for the reasons of D2 of unit 0005, so each reading costs one flicker
  of the LEDs, and the only lever left before the LEDs are driven in software mode is how often it happens. The probes
  of a case, the intake air and a power connector, change over minutes, so a reading of the last minute is close to
  the truth, while a flicker every second is a fault anyone near the machine sees
- **D2** The kept readings are sent in every round, R5, and not only when a reading of the controller gives a new
  value. This is how `run` treats every metric that has a value today: it sends it in every round whether or not it
  changed, and leaves out only what is unavailable, D3 of unit 0004. Sending only at a reading would leave the screen
  without the probes for up to a minute after the screen is unplugged or reboots and comes back, R22 of unit 0004,
  since a screen that rebooted may have lost the values it showed. The cost is a few bytes of the value reports that
  are written anyway
- **D3** The kept readings are those of the last reading of the controller, replaced as a whole, R4 and R6, and not the
  last good value of each metric across readings. A reading that fails therefore leaves the metrics it did not give
  out of the rounds until the next reading, and the screen keeps its last value, by D3 of unit 0004. Keeping the value
  of an older reading across a failure was rejected: the screen would show the same value either way, but `run` would
  go on sending a value that may be minutes old as if it were current, and for as long as the controller stays away
  or keeps failing. The rule also needs no change of the contract of a source: `run` keeps what the source gave and
  needs no way to tell a failure from a controller with fewer probes
- **D4** A reading that fails waits the full period, R6, and is not made again at the next round. A failure is most
  likely a controller that is missing, has no access, or answers badly, and none of these mends itself within a
  second; retrying at every round would, for a controller that answers badly, bring back the flicker of every second
  that this unit removes, since step 1 was written and the mode changed in a failed sequence too. The cost is
  freshness: after a single failure the probes are missing from `run` for one period, and a controller plugged in
  while `run` runs is read up to a period later. The screen keeps its last values meanwhile, D3 of unit 0004, so a
  missed minute is hardly visible. A single retry at the next round was rejected as well, because it doubles the
  flicker of a controller that fails often and only helps the rare failure that one second mends
- **D5** The key is `commander_interval`, at the top of the file beside `interval`, with the same syntax as
  `interval`, so that the file stays a flat mapping of scalars and of `tags`. `commander` is the prefix of the metrics
  of the controller, so the key reads as the interval of those metrics. A mapping such as `commander: {interval: 1m}`
  was rejected as a structure for one value, which nothing else in the file has. A count of rounds, such as read the
  controller every 60th round, was rejected because it would change meaning with `interval`, while the human chose a
  time
- **D6** `commander_interval` must be at least the interval, R1 and R2. A period shorter than the interval cannot be
  honoured, because the controller is read only in a round, and accepting it would let the file say something that
  `run` does not do, so it is refused as a mistake, as D11 of unit 0004 refuses an empty `tags`. A period equal to the
  interval is valid and reads the controller in every round, the behaviour of units 0005 and 0006, for someone who
  prefers fresh probes to quiet LEDs. A fixed minimum of 1 second was rejected: it would refuse that choice for an
  interval below 1 second, and allow a period below an interval above 1 second. No upper bound is set, since a long
  period only makes the probes older, which the user chose. Without the key and with an interval above 1 minute the
  period follows the interval, R1, so that a file that is valid today stays valid
- **D7** The period is measured from the start of the round that read the controller to the start of a later round,
  R4, as the clock of R20 of unit 0004 is measured. A round that starts a little after its place on the grid then
  pushes a reading to the next round, so with an interval of 1 second two readings are between 60 and 61 seconds
  apart. Measuring on the grid instead would keep them at exactly 60 seconds, a precision no one would see, at the cost
  of a second rule of time in `run`
- **D8** The period is a matter of `run` alone, and the source of the controller and its driver do not change, R9 and
  R10. `sensors` reads once, so its one reading costs one flicker, and a period has nothing to save there. The driver
  keeps no state between readings, D9 of unit 0005, and keeping readings is the job of the loop that has rounds
- **D9** Other sources are read in every round, R7. Only the controller has a cost to the user for each reading; the
  other sources cost a file read or a run of `nvidia-smi` and change faster

## Non-goals

- Driving the LEDs, or the fans, of the controller while it is in software mode, which a later unit may do to remove
  the flicker
- Reading the controller without software mode, or keeping it in software mode between readings
- Any change to the sequence, the commands, the time limit, the checks of an answer or the return to hardware mode of
  units 0005 and 0006, among them dropping the fan steps 2 to 4
- A period for any source other than the controller
- Reading the controller in the background between rounds, or at another moment than the start of a round
- An option on the command line for the period, and reloading the config while `run` runs
- A change of `sensordeck sensors`

## Beyond the request

1. The line of `run` at the start names the period when a metric of the controller is mapped, such as
   `running: 3 metrics every 1s, the controller every 1m0s`. It costs a second form of the line of R19 of unit 0004
   and a change of the end-to-end tests that check it. Without it the period is what the config says, and the user
   who wonders why the probes change once a minute finds it in the README
