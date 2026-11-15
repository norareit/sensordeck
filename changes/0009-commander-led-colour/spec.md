# A static colour for the LEDs of the Commander Core XT in run

## Context

Unit `0005-commander-core-xt-probes` reads the temperature probes of the Corsair iCUE Commander Core XT, and every
reading enters software mode and returns to hardware mode at once. On the device the LEDs of the fans on the
controller go dark while it is in software mode, so each reading makes them flicker. Unit `0008-commander-read-period`
made `run` read the controller at most once per period, of 1 minute by default, so the LEDs flicker once a minute
instead of once a second, and named driving the LEDs in software mode as the later remedy. This is that unit.

With a new key of the config, `run` holds the controller in software mode for as long as it runs, lights the LEDs of
the fans in one static colour from the config, reads the probes in every round, and keeps the controller awake with
at least one exchange a second. On every way `run` ends, it returns the controller to hardware mode, so that the
lighting stored in the controller comes back. Without the key, `run` behaves as unit 0008 says.

### Provenance and licensing

The facts below come from three sources, and from nothing else:

- A black-box trace with `strace` of the hidraw traffic of OpenLinkHub on the human's machine, a capture that is kept
  locally and is not committed. OpenLinkHub is licensed under the GPL-3.0. It was run, and its code was not read
- Tests that the orchestrating session made by hand on the device, with the human watching the LEDs
- liquidctl's public description of the protocol of this family of controllers, read for facts only

No code of OpenLinkHub, liquidctl or OpenRGB is read, translated or copied for this unit. The project is MIT, as
`AGENTS.md` requires. What a device expects on the wire is a fact about the device, and the documents of this unit
state those facts in the project's own words. The earlier facts of units 0005, 0006 and 0008 still hold.

### What is known

Offsets of a write count the 96 data bytes after the report ID byte, and offsets of an answer the bytes read, from 0.
A write is 97 bytes as in unit 0005: the report ID `0x00`, the command, and `0x00` up to 96 data bytes. An answer
echoes the second byte of its command in byte 1 and carries a status in byte 2, as units 0005 and 0008 describe.

Channels and endpoints, from the trace:

- Every command is `08 <command> <channel> ...`: the byte after the command is a channel. An endpoint is opened on a
  channel with `08 0D <channel> <endpoint>`, read with `08 08 <channel> <endpoint>` and closed with
  `08 05 01 <channel> <endpoint>`. The commands `08 08 00` and `08 05 01` of unit 0005 are the read and the close of
  channel 0, with zeros in the place of the endpoint
- OpenLinkHub opened the LED endpoint `0x22` on channel 0 once and kept it open for its whole run. It made every read
  of a sensor endpoint on channel 1, each as four exchanges: close, open, read, close, so `08 05 01 01 21`,
  `08 0D 01 21`, `08 08 01 21`, `08 05 01 01 21`. Every one of these answers, the close of an endpoint that was not
  open among them, had status `00`
- After setup it read the endpoints `0x1A`, `0x17` and `0x21` in that order, each in those four exchanges, every
  1.000 seconds, and wrote nothing else unless the lighting changed. The controller stayed in software mode with the
  colour shown for the minutes of the trace. At its end it wrote `08 01 03 00 01` and nothing else
- OpenLinkHub wrote the full output report, 385 bytes with the report ID, where sensordeck writes 97

The LED endpoint, from the device tests:

- In software mode, `08 0D 00 22` answered status `00`. A colour write of 16 triples `FF 00 00` then answered status
  `00`, and the human saw all fans turn red at once
- While `0x22` was open on channel 0, opening `0x21` on channel 0 answered status `03`. After a close of channel 0,
  opening `0x21` on channel 0 worked
- Afterwards `08 01 03 00 01` answered `00` and the fans went back to the lighting stored in the controller, the
  human's icy blue
- So in hardware mode the controller shows its stored lighting, and in software mode only what the host sends. With
  nothing sent, the LEDs are dark, which is the flicker of units 0005 and 0008

The colour write, from the trace and the device test: `08 06 00`, the command `06` on channel 0; bytes 3 and 4 a
length, low byte first, `32 00` for 16 LEDs; bytes 5 and 6 `00 00`; bytes 7 and 8 the data type `12 00`; then one
triple of red, green and blue per LED. The length counts the two bytes of the data type and three bytes per LED. For a
static colour OpenLinkHub wrote a colour write of all zero triples and, about 2 milliseconds later, the one with the
colour, and wrote neither again until the colour changed. For an animated effect it wrote one about every
22 milliseconds. The device test shows that one write with the colour lights the LEDs.

The LED count, from the trace: a read of the endpoint `0x20` on channel 1 answered the data type `0F 00` in bytes 3
and 4, a count of 7 in byte 5, and from byte 6 seven entries of 4 bytes each, for the connector EXT and the ports 1
to 6: a mode of 16 bits and an LED count of 16 bits, both low byte first. On the host the entries were mode `00` with
0 LEDs for EXT, mode `03` with 0 LEDs for ports 1 and 2, and mode `02` with 4 LEDs for ports 3 to 6: 16 LEDs, three
front fans and one back fan, which matches the 16 triples of the device test. Mode `02` reads as connected and `03` as
disconnected.

The other answers of the trace: the read of `0x1A` answered the data type `09 00`, a count of 6 and six bytes `01`,
whose meaning is unknown. The reads of `0x17` and `0x21` answered as in unit 0005.

Not used, from the trace: at setup OpenLinkHub also wrote `08 02 13`, `08 14 0N 01` for N from 1 to 6, writes with the
command `06` to the endpoints `0x18`, data type `07 00`, and `0x1E`, data type `0D 00`, opened on channel 1, and
`08 15 01`. Their meaning is unknown, and some may concern the fans. The device test lit the LEDs without them.

The keepalive: liquidctl's description says that the device must be woken after some seconds without data, without
saying how many, and that `08 01 03 00 01` returns it to hardware mode. The timeout is unknown. The trace shows that
reads once a second hold software mode.

Fans: in unit 0005 the controller reported a speed of 0 on all six fan outputs of the host, so the fans of the host
are not run by the controller. What the controller does with fans that it does run, in software mode with no fan
settings sent, is not known.

The unit is built in the sandbox, which has no controller and no screen. Every requirement is checked by automated
tests with a time the test moves, sources the test gives, the fake screen and the fake controller. The check on the
device is the human's.

## Requirements

"R*n* of unit 000*k*" names a requirement of the spec of that unit, and "the controller" means what it means in unit
0005. Every requirement of units 0004 to 0008 that this spec does not change still holds. Commands are written as
their data bytes after the report ID byte, and every byte not named is `0x00`. "An exchange" is one write of a command
and one read of its answer, as R5 of unit 0005 and R1 of unit 0006 make it: a write of 97 bytes, a read into a buffer
of 384 bytes, and a time limit of 1 second for the answer, which is a value the code takes so that a test gives a
shorter one. An exchange "succeeds" when the answer is valid by R2 of unit 0006: at least 96 bytes, the echo in byte 1
and status `00` in byte 2, plus the checks that a requirement below adds for its answer. "The colour" is the value of
`commander_color`. "A session" lasts from the write of `08 01 03 00 02` that its setup makes to the end of its
teardown.

### The config file

- **R1** The config file of R17 of unit 0004 has a fourth key, `commander_color`, a YAML string of `#` followed by six
  hexadecimal digits, in upper or lower case, which give red, green and blue in that order. For example:

  ```yaml
  interval: 1s
  commander_color: "#00c8ff"
  tags:
    cpu.temp: 1
    commander.1.temp: 18
    commander.2.temp: 19
  ```

  The key is valid whether or not a metric of the controller is mapped in `tags`. `tags` still needs at least one
  entry, as R18 of unit 0004 says
- **R2** The checks of R18 of unit 0004 gain these problems, with these texts after the prefix `config file <path>: `:
    - a `commander_color` that is empty or null, which is what an unquoted `#` gives, since it starts a comment in
      YAML: `commander_color: no colour given, write it in quotes such as "#00c8ff", because # starts a comment in
      YAML`
    - any other `commander_color` that is not as R1 says, including one that is not a scalar:
      `commander_color: "<text>" is not a colour such as "#00c8ff"`, with the text as written in the file and the
      empty text for a value that is not a scalar
    - both `commander_color` and `commander_interval` given: `commander_interval: cannot be used with commander_color,
      which reads the controller in every round`

  When a file has several problems one of them is reported, as R18 of unit 0004 says
- **R3** The two texts of R3 of unit 0008 name the new key, listing the four keys in the order of Go's string
  comparison:
    - a file whose content is not a mapping: `want a mapping with the keys commander_color, commander_interval,
      interval and tags`
    - a key other than the four: `unknown key "<key>", the keys are commander_color, commander_interval, interval and
      tags`
- **R4** Without `commander_color`, `run` behaves exactly as units 0004 to 0008 say, the period of unit 0008 included,
  and nothing of R5 to R19 applies

### The session

- **R5** With `commander_color`, the sequence of R5 of unit 0005 is never made by `run`, and the controller is reached
  through sessions alone. The first setup starts after the screen has been opened by R19 of unit 0004, so that a config
  that is not valid or a screen that cannot be opened writes nothing to the controller. The node is found as R3 of
  unit 0005 says, opened for reading and writing at the setup, kept open for the session and closed after its
  teardown. A node that is not found or cannot be opened ends the setup with nothing written, and is no session
- **R6** A setup makes these exchanges, in this order:
    1. `08 01 03 00 02`, enter software mode
    2. `08 05 01 01 20`, `08 0D 01 20`, `08 08 01 20` and `08 05 01 01 20`: the read of the endpoint `0x20` on
       channel 1, whose read answer R7 checks
    3. `08 0D 00 22`, open the LED endpoint on channel 0
    4. the colour write of R8

  A setup whose exchanges all succeed starts a session that is held. The first exchange that does not succeed ends
  the setup, no further exchange of the setup is made, and the teardown of R12 follows
- **R7** The answer of the read of `0x20` succeeds only when bytes 3 and 4 are `0F 00` and its count *c* in byte 5
  satisfies 6 + 4·*c* ≤ 96. Entry *i*, for *i* from 0 to *c* − 1, is the four bytes at offset 6 + 4·*i*: a mode of 16
  bits and an LED count of 16 bits, both low byte first. The LED count *n* of the controller is the sum of the LED
  counts of the entries whose mode is 2. A count *n* of 0 or more than 29 makes the setup fail at this step. For the
  answer of the host, *n* is 16
- **R8** The colour write is the command `08 06 00`, then the length 2 + 3·*n* in bytes 3 and 4, low byte first, then
  `00 00` in bytes 5 and 6, then `12 00` in bytes 7 and 8, then *n* triples from byte 9, each the red, green and blue
  of the colour, and zeros up to 96 data bytes. For 16 LEDs and `#ff0000` the data bytes start with
  `08 06 00 32 00 00 00 12 00` followed by 16 times `FF 00 00`. It is written once in a session, at its setup, and
  never again in that session
- **R9** A reading of a session makes, for each of the endpoints `0x1A`, `0x17` and `0x21` in that order, the four
  exchanges `08 05 01 01 <endpoint>`, `08 0D 01 <endpoint>`, `08 08 01 <endpoint>` and `08 05 01 01 <endpoint>`, twelve
  exchanges in all. The answer of the read of `0x21` succeeds only when it passes the checks of R2 of unit 0006 on its
  data type and count, and gives the probes as R9 of unit 0005 says. The answers of `0x1A` and `0x17` are checked as
  every answer is and their content is not used. The first exchange that does not succeed ends the reading and the
  session, gives no probe, and the teardown of R12 follows
- **R10** In a session `run` writes no command but those of R6, R8, R9 and R12. Among others it never writes
  `08 02 13`, `08 14 0N 01`, a command to the endpoints `0x18` or `0x1E`, `08 15 01`, a colour write of zero triples,
  or a command that opens on channel 0 an endpoint other than `0x22`. Nothing is written to the node of interface 1
- **R11** While a session is held, at most 1 second passes between the end of the setup or the start of one reading
  and the start of the next reading. When 1 second has passed since the later of the two with no reading started, the
  session starts one itself. This holds whatever the interval is, while the screen is lost by R22 of unit 0004, and
  while a round waits for a source, for the screen or for its next start, so the keepalive does not wait for a round.
  Two readings, or a reading and a setup or teardown, never run at the same time on the node: one that is asked for
  while another runs starts after it ends
- **R12** A teardown writes `08 05 01 00 22`, the close of channel 0, and then `08 01 03 00 01`, the return to hardware
  mode, each as one exchange whose result is not used. The close is left out when the exchange that ended the setup
  or the session had a write that failed or no answer within the time limit, and the return to hardware mode is never
  left out. The node is closed after it. A teardown follows every setup once `08 01 03 00 02` has been written,
  whatever its answer, and ends every session: after an exchange that does not succeed, on SIGINT or SIGTERM, and on
  any other end of `run`
- **R13** On SIGINT or SIGTERM, an exchange in progress ends at its answer or its time limit, no exchange of a setup
  or a reading is made after it, and the teardown of R12 follows. `run` then writes `stopped` and ends with the exit
  status 0 as R23 of unit 0004 says, within 3 seconds of the signal, also when the controller stopped answering. A
  stop before a setup has written anything writes nothing to the controller
- **R14** A setup is made when `run` starts, by R5, and again after each setup that fails and each session that ends
  before `run` does. The next setup starts 10 seconds after the end of the one that failed or of the teardown of the
  session that ended. Each further setup that fails doubles that wait, up to 5 minutes. A setup that succeeds sets it
  back to 10 seconds. The setups are made by the session and not by the rounds, so they do not depend on the screen.
  A setup that succeeds writes the colour by R8, so the colour comes back after a replug of the controller
- **R15** `run` writes to standard error:
    - `controller lost: ` and a text that names the exchange or the check that failed, when a held session ends by an
      exchange that does not succeed
    - `controller back`, when a setup succeeds after such a loss
    - `controller setup failed: ` and a text that names what failed, when a setup fails for another reason than a
      node that is not found. The line is written for the first such failure and not again until a setup succeeds,
      so a controller that keeps failing does not fill a log. For R7 the text names *n*

  A node that is not found writes no line. Standard output, the line of R19 of unit 0004 included, does not change
- **R16** A round that reads its sources by R21 of unit 0004 and R8 of unit 0008, with at least one metric of the
  controller mapped, asks the session for a reading, and the readings of the controller in that round are the mapped
  probes of a reading that starts after the round started. A round that asks while a setup runs waits for its end.
  When no session is held, or the reading does not succeed, the metrics of the controller are unavailable in that
  round. A round with no metric of the controller mapped asks for no reading, and the session is held all the same,
  for the colour. The rounds in which `run` calls `Reopen` ask for nothing, and the session is held through them
- **R17** `sensordeck sensors` is unchanged, R10 of unit 0008. Running `sensors` or another program against the
  controller while `run` holds a session is not supported, and the README says what happens then by R21

### Platforms and layout

- **R18** The session is limited to Linux by build constraints, as R11 of unit 0005 says of the driver, and the test of
  the layout of unit 0003 checks it. The config and its checks are not limited to a platform. In a build for Windows
  `run` fails as R14 of unit 0004 says. The five commands of the suite of `AGENTS.md` succeed
- **R19** The comment of the rule of the controller in `udev/70-sensordeck.rules` says that sensordeck reads its
  temperature probes and lights its LEDs. The rules themselves do not change

### The documents

- **R20** `README.md` has a section with the heading `## The Corsair iCUE Commander Core XT`, which says in plain words:
    - that the controller has a hardware mode, in which it runs its fans and shows the lighting stored in it, and a
      software mode, in which it shows only what a program sends it, and is dark when nothing is sent
    - that reading the probes needs software mode, so without `commander_color` each reading of `run` makes the LEDs
      flicker once, at most once per period of `commander_interval`, of 1 minute by default, and that `sensors` makes
      them flicker once each time it runs
    - what `commander_color` does: `run` holds software mode for as long as it runs, lights the LEDs of every
      connected port in that one colour, reads the probes in every round and keeps the controller awake once a second;
      that the value must be in quotes, and that it cannot be used with `commander_interval`
    - what the user sees: at the start a moment of darkness and then the colour; while running the colour, steady;
      after Ctrl+C or SIGTERM the stored lighting again
    - that when the controller is unplugged or stops answering, `run` returns it to hardware mode if it can, writes
      `controller lost`, leaves its metrics out, keeps the screen going, tries again after 10 seconds and then less
      often, up to every 5 minutes, and writes `controller back` and lights the colour again when it succeeds
    - that another program that talks to the controller at the same time, such as iCUE, liquidctl, OpenLinkHub or
      `sensordeck sensors`, is not supported: the commands of both mix, readings fail, the LEDs may show the stored
      lighting or go dark, and `run` sets the colour again at its next setup
    - that what the controller does with fans that it runs, in software mode with no fan settings sent, is not known,
      that sensordeck sends no fan settings, and that the fans of the machine it was tested on are not run by it
    - that at most 29 LEDs can be lit, and that a controller with more makes `run` write `controller setup failed`
    - that a killed process, as with SIGKILL, or a loss of power can leave the controller in software mode with the
      colour or dark, until a program returns it to hardware mode or it is replugged
    - a complete example config with `interval`, `commander_color` and `tags` mapping both probes, and a second one
      with `interval`, `commander_interval` and `tags`, each noted as the choice of one way
    - that the controller needs the second rule of `udev/70-sensordeck.rules`, with the line of that rule

  The paragraph on `commander_interval` in "Showing sensor values on the screen" says that the period applies when
  `commander_color` is not given, and points to the new section. Nothing else of the README that units 0004 to 0008
  wrote is removed
- **R21** `docs/commander-core-xt.md` describes the controller as far as sensordeck uses it, in the project's own words.
  Its title covers the lighting as well as the probes. It keeps what it says today of the identity, the report sizes
  and descriptor of unit 0006, the eight commands of unit 0005, the stale bytes, the times, and the reading without
  software mode that unit 0008 tried. It adds:
    - the channel model of "What is known": the channel byte, and the open, read and close of an endpoint on a
      channel, with the commands of unit 0005 read as those of channel 0
    - the LED endpoint `0x22`, the status `03` for a second endpoint opened on the same channel, and the colour write
      with its layout
    - the layout of the answers of `0x20` and of `0x1A`, and the rule of R7 for the LED count, with the assumption
      that mode 2 means connected and that the triples are taken in the order of the entries, the LEDs of a port one
      after the other, which a single colour does not depend on and nothing has checked
    - hardware mode and software mode as the device tests showed them
    - the keepalive: what liquidctl's description says, that the timeout is unknown, that reads once a second held
      software mode in the trace, and that sensordeck keeps one reading a second
    - the commands of a session in order: the setup of R6, the readings of R9 and the teardown of R12
    - the commands of the trace that sensordeck deliberately does not send, by R10, and that the device test lit the
      LEDs without them
    - the provenance of each fact: which of the trace of liquidctl of unit 0005, the trace of sensordeck of unit 0006,
      the trace of OpenLinkHub or the device tests of units 0008 and 0009 it comes from, and that the code of none of
      these programs was read

  Its statements that this unit makes untrue are corrected: that what the controller does in software mode when nobody
  sends it settings is not known, which is now known for the LEDs; and that `run` reads the controller at most once per
  period, which now holds only without `commander_color`. Its section "Unknown" names at least: the software-mode
  timeout; the meaning of the setup commands of R10; the meaning of the answer of `0x1A`; modes of an entry of `0x20`
  other than 2 and 3; the order in which triples map to ports; whether the trailing endpoint byte of a read or a close
  matters; whether the device needs the full output report for a colour write of more than 29 LEDs; and what the
  controller does with fans it runs in software mode

### Tests

- **R22** The fake controller of R17 of unit 0005 and R4 of unit 0006 answers, besides the eight commands it answers
  today, the commands of R6, R8, R9 and R12 on channels 0 and 1. It keeps which endpoint is open on each channel, and
  answers status `03` to an open on a channel that has another endpoint open. It answers the read of `0x20` with entries
  the test sets, by default those of the host, the read of `0x1A` with the answer of "What is known", and the read of
  `0x21` on channel 1 with the probes the test sets. It answers status `03` to every command but a mode command while
  it is not in software mode. It keeps the colour writes it received and tells what its LEDs show: the stored lighting
  in hardware mode, the last colour in software mode after a colour write, and dark in software mode before one. It
  has a software-mode timeout that the test sets, off by default: when no command arrives for that time in software
  mode it goes to hardware mode, closes every channel and forgets the colour. A test can make it give no answer, a
  wrong echo or a status that is not `00` to a chosen command. Its tests show each of these, and every test of units
  0005, 0006 and 0008 that uses it keeps what it expects
- **R23** Unit tests of the config cover: a colour in lower and in upper case, with its red, green and blue; the
  colour with no metric of the controller mapped; an unquoted colour, an empty and a null value, each with the first
  text of R2; `#00c8f`, `00c8ff`, `#00c8fg` and a mapping, each with the second text; both keys, with the third text;
  and the texts of R3. The existing tests of the texts of R3 of unit 0008 are changed to these, and the others stay
- **R24** Unit tests of the session against the fake controller cover: the bytes of every write of a setup, of a
  reading and of a teardown, for the entries of the host and `#ff0000`; an LED count from other entries, such as three
  ports of 8 LEDs giving 24 triples; counts of 0 and 30, which fail the setup at R7 with a teardown; a failure of each
  of the four steps of a setup and of a reading at `0x21`, each followed by the teardown with the close, and a write
  of no further command of the setup or the reading; no answer, followed by a teardown without the close; a status
  `03` to the open of `0x22`, made by the test leaving an endpoint open on channel 0, after which the next setup
  succeeds; a stop during a setup and during a reading, each followed by the teardown; no node, which writes nothing;
  nothing written to the node of interface 1; and the probes of a reading for the answer of unit 0005
- **R25** Unit tests of `run`, with a time the test moves, sources the test gives and a session or a stand-in for it
  that counts what it is asked, cover: with an interval of 10 seconds, readings at most 1 second apart; the same while
  the screen is lost; a reading in every round with an interval of 100 milliseconds and a probe mapped; no reading asked
  by a round when no probe is mapped, with the keepalive going on; a failed session, after which the metrics of the
  controller are left out of the rounds; the waits of R14 for a run of failed setups, 10, 20, 40, 80, 160 and 300
  seconds and then 300, and the return to 10 seconds after a setup that succeeds; the colour written again after a
  setup that succeeds; the lines of R15, the one of a failed setup written once; and a stop and a feeding error, each
  ending with a teardown
- **R26** End-to-end tests run `run` against the fake screen, the tree of R35 of unit 0004 and the fake controller,
  with a config of `interval: 100ms`, `commander_color: "#ff0000"` and `tags` of `cpu.temp: 1` and
  `commander.1.temp: 4`. They cover:
    - a value report holding (4, 37); then the test sets the first probe to 40.0 °C and a value report holding (4, 40)
      arrives within 1 second
    - after SIGTERM: exit status 0 within 3 seconds, the fake in hardware mode with channel 0 closed, and over the
      whole run exactly one write of `08 01 03 00 02` and one colour write, of 16 triples `FF 00 00`
    - with `interval: 5s` and a software-mode timeout of the fake of 1.5 seconds: 4 seconds after the colour write the
      fake is still in software mode, shows the colour, and has received at least three reads of `0x21`
    - the node of the controller removed while `run` runs: `controller lost: ` on standard error, and value reports
      with the pair of tag 1 still arriving
    - a config with an unquoted colour, with no screen: the first error of R2 and exit status 1
- **R27** The existing unit and end-to-end tests of the suite pass, with no change of what they expect beyond the texts
  of R3

## Decisions

- **D1** The unit is written from observed traffic, the device tests and a public description, and no code of
  OpenLinkHub, liquidctl or OpenRGB is read, translated or copied, as the context says and `AGENTS.md` requires
- **D2** With the colour, `run` holds software mode for its whole life, which D2 of unit 0005 rejected. The reason of
  that decision was that nothing was known of software mode with nobody sending settings. For the LEDs it is now known:
  they show what is sent, and the device test lit them. The remaining unknown is the fans that the controller runs,
  which on the host it does not; the key is therefore one the user chooses, the README states the unknown by R20, and
  without the key nothing changes. Holding software mode removes the flicker entirely, which the period of unit 0008
  only made rarer
- **D3** The readings of a session are the twelve exchanges of R9, the reads of `0x1A`, `0x17` and `0x21` on channel 1
  each closed before and after, exactly as the trace made them, although only `0x21` is used. They are the one traffic
  seen to hold software mode for minutes, and the timeout is unknown, so the keepalive copies them rather than a
  shorter set that might not count as data for the controller. They take about 24 milliseconds. Dropping `0x1A` and
  `0x17`, or the close before the open, is left to a later unit after a check on the device, as D3 of unit 0005 left
  the fan steps
- **D4** The keepalive keeps at most 1 second between readings, R11, the rate of the trace, because the timeout is
  unknown and a faster rate gains nothing that anyone has seen. It runs apart from the rounds because a round can wait:
  for `nvidia-smi` up to its time limit of 2 seconds, for the screen, or for an interval longer than 1 second, and the
  controller must not fall back to hardware mode or go dark meanwhile. Readings are made one at a time so that the
  answers on the node do not mix
- **D5** A round with a probe mapped makes its own reading, R16, as the orchestrating session decided, so that the
  probes are as fresh as every other metric. Using the reading of the keepalive instead would give values up to a
  second old for no saving that matters: at an interval of 1 second the keepalive then rarely fires, and at
  100 milliseconds the twelve exchanges take about a quarter of each round
- **D6** The LED count is read from `0x20` at each setup, R7, and not taken as the constant 16 of the host. Another
  machine has other fans, and a replug may change them, so a constant would leave LEDs dark or send triples for LEDs
  that are not there, with an effect nobody knows. Only entries of mode 2 count, the one mode seen with LEDs; on the
  host every other entry has 0 LEDs anyway. The cost is one read of four exchanges at setup
- **D7** The colour write is 97 bytes like every other write, as D5 of unit 0006 keeps them, so it holds at most 29
  triples: 9 + 3·29 = 96. A larger count fails the setup with a line, R7 and R15, rather than lighting some LEDs and
  leaving the others dark, which would look like a fault of the fans. Writing the full output report, as OpenLinkHub
  did, is item 1 of "Beyond the request". A count of 0 fails too, since there is nothing to light and holding software
  mode would then only add risk
- **D8** The order of the triples needs no decision in this unit: with one colour every triple is the same. The
  document records the assumed order of R21 for a later unit with colours per port
- **D9** The colour is written once per session, R8, because the device test shows that one write lights the LEDs and
  that they stay lit, and the trace shows OpenLinkHub writing a static colour only when it changed. The write of zero
  triples that OpenLinkHub made before it is not made: it has no observed effect but a moment of darkness, and the
  device test needed none
- **D10** The commands copy the forms of the trace, the endpoint byte at the end of a read and a close included. The
  close of channel 0 in the teardown, `08 05 01 00 22`, follows that form for the endpoint that was opened. Whether the
  endpoint byte matters is unknown: the `08 05 01` of unit 0005 worked without it
- **D11** No close of channel 0 is made before `0x22` is opened. A run that was killed may leave `0x22` open, and the
  open then answers `03`, as the device test showed for a second endpoint. The setup then fails, its teardown closes
  channel 0, and the next setup 10 seconds later succeeds, R24, so the case heals itself with the commands already
  needed, without a close that nothing has shown to answer `00` on channel 0 when nothing is open
- **D12** The teardown leaves out the close after a write that failed or an answer that did not come, R12, so that a
  stop while the controller does not answer costs at most two time limits, the exchange in progress and the return to
  hardware mode, and `run` ends within the 3 seconds of R23 of unit 0004. After an answer that came but was not
  valid, the controller is answering, and the close is made, which D11 needs. Answers take about 2 milliseconds, so
  only a controller that answers just before each time limit and then stops could bring a stop near 3 seconds. The
  return to hardware mode is always made, because it is what brings the stored lighting back and what the request requires on every end
- **D13** A lost session ends with a teardown and is set up again after a back-off of 10 seconds that doubles up to
  5 minutes, R14. Each setup that writes `08 01 03 00 02` costs a moment of darkness of the LEDs, and a controller that
  fails at once after each setup would otherwise flicker at the rate of the retries; the back-off makes that rare, at
  most once in 5 minutes once failures repeat, far less than the once a minute of unit 0008. The first retry after
  10 seconds brings a replugged controller back quickly. A missing node writes nothing and costs nothing on the device,
  but takes the same waits so that the rule stays one. While no session is held the probes are left out, and no
  reading of unit 0005 is made in its place, since that would bring back the flicker this unit removes
- **D14** The lines of R15 mirror the `screen lost` and `screen back` of R22 of unit 0004, so that a user who sees the
  colour vanish finds why. A failed setup is written once until a success, as D14 of unit 0004 writes no line for each
  failed reopening, so that a controller that keeps failing, or one with too many LEDs, does not fill a log. A missing
  node writes nothing, because a config used on a machine without the controller is a normal case
- **D15** The key is `commander_color`, beside `commander_interval`, a string `#rrggbb` as colours are commonly
  written. YAML reads an unquoted `#` as the start of a comment, so the most likely mistake gives an empty value, and
  R2 gives it a text of its own that says to quote it. The key is valid without a probe mapped, because the human may
  want only the lighting: the session needs no metric, and the keepalive keeps it, at a cost of twelve exchanges a
  second. Giving both keys is an error and not an interval ignored in silence, because with the colour the controller
  is read in every round and a file that names a period `run` does not keep says something false, as D6 of unit 0008
  refuses a period shorter than the interval
- **D16** `sensors` does not change, R17. It reads once and is short, and keeping it out of the session keeps the two
  commands independent. Running it while `run` holds a session is not supported: hidraw gives each answer to every
  open file of the node, so each program may read the other's answers, the checks then fail a reading, `sensors` ends
  with `08 01 03 00 01` and so puts the controller in hardware mode, after which every answer to `run` has status `03`
  and its session ends. At worst the readings fail and the LEDs show the stored lighting or go dark until the next
  setup of `run`, and nothing is damaged. Coordination between the two is item 2 of "Beyond the request"
- **D17** This unit adds behaviour behind a new key and leaves the behaviour of units 0005 and 0008 as it is without
  it, so it supersedes neither. Their specs stay true for a config without the key, and the statements of the shared
  document that become untrue are corrected by R21
- **D18** The fake controller answers `03` outside software mode, R22, as the trial of unit 0008 showed the device
  doing, so that the software-mode timeout of the fake has the effect it has on the device: the next exchange of the
  session fails. The commands of unit 0005 are written after step 1, so their tests are not affected

## Non-goals

- Fan control of any kind, fan settings, and the setup commands of R10
- Colours per port or per LED, effects, brightness, and changing the colour while `run` runs
- Changing the lighting stored in the controller for hardware mode
- A session in `sensors`, and any change of `sensors`
- Coordination with other programs that talk to the controller, such as iCUE, liquidctl, OpenLinkHub or `sensors`
- More than 29 LEDs, and writes of the full output report
- Other Corsair controllers, more than one Commander Core XT, and the controller on Windows
- The traffic of the controller in `--trace`, declined in units 0005 and 0006
- Fan speeds as metrics, and any use of the answers of `0x1A` and `0x17`
- An option on the command line for the colour, and reloading the config while `run` runs

## Beyond the request

1. Colour writes of the full output report, 385 bytes with the report ID, as OpenLinkHub wrote them, which hold up to
   125 triples. It costs a second size of write, its tests and a check on the device that the controller takes it.
   Without it a controller with more than 29 LEDs, such as six fans of 34 LEDs, fails the setup with a line, and its
   user keeps `commander_interval` and the flicker of unit 0008
2. A lock that `run` holds while it has a session, which `sensors` checks to leave the controller alone and say so. It
   costs a lock file in the runtime directory, its handling when `run` was killed, and tests. Without it running
   `sensors` beside `run` is unsupported as D16 says, and at worst fails readings and darkens or resets the LEDs until
   the next setup of `run`
