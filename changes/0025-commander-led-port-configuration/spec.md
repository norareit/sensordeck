# Repairing the LED port configuration of the Commander Core XT that iCUE rewrote

## Context

Unit `0009-commander-led-colour` made `run` hold the Corsair iCUE Commander Core XT in software mode with
`commander_color` and light the LEDs of its fans in one colour. On the human's controller this stopped working after
Corsair iCUE had run on Windows: the LEDs stay dark in every session, although every exchange of the setup succeeds.
The controller stores an LED port configuration, in the endpoint `0x1E`, that iCUE rewrites and that unit 0009 never
reads or writes. This unit makes `run` repair it at the start of a session.

### Provenance and licensing

The facts below come from the same kinds of sources as those of unit 0009, and from nothing else:

- The black-box trace with `strace` of the hidraw traffic of OpenLinkHub of 2026-11-15, the capture that unit 0009
  used, kept locally and not committed. It was read for the traffic alone. OpenLinkHub is licensed under the GPL-3.0,
  and its code was not read
- Tests that the orchestrating session made by hand on the device on 2026-11-16, with a script of its own and the
  human watching the LEDs

No code of OpenLinkHub, liquidctl, OpenRGB or iCUE is read, translated or copied for this unit. The project is MIT, as
`AGENTS.md` requires. What a device expects on the wire is a fact about the device, and the documents of this unit
state those facts in the project's own words.

### What was observed

On the human's controller, a Commander Core XT with firmware 1.4.62, on 2026-11-16:

- After the human had run iCUE on Windows, once killing it while it ran, the LEDs were dark whenever sensordeck held
  the controller in software mode with `commander_color`. This held on Linux and on Windows, with every version of
  sensordeck, those that had lit the LEDs before included, and after a full power cut
- The setup of unit 0009 was accepted all the same: every answer had status `00`, the colour write
  `08 06 00 32 00 00 00 12 00` with 16 triples included
- In hardware mode the stored lighting showed. While iCUE ran, the LEDs were lit

### What is known of the endpoint `0x1E`

Offsets count as in unit 0009: the bytes of a command after the report ID byte, and the bytes of an answer as read,
from 0.

- The read of `0x1E` on channel 1, as `08 0D 01 1E` and `08 08 01 1E`, answered after iCUE had run with the data type
  `0D 00` in bytes 3 and 4, the count `07` in byte 5, then `06 06 06 06 06 06 06`, and then
  `00 00 01 03 01 03 01 03 01 03`. Seven is the count of connections of the answer of `0x20`, EXT and the ports 1 to 6.
  The bytes after the seven `06` are taken to be stale bytes of an earlier answer, which is not established
- In the trace, OpenLinkHub wrote at setup, after the read of `0x20` and the open of `0x22` on channel 0:
  `08 05 01 01 1E`, `08 0D 01 1E`, the write `08 06 01 0E 00 00 00 0D 00 07 00 00 00 01 03 01 03 01 03 01 03`,
  `08 05 01 01 1E`, and then `08 15 01`, each answered with status `00`. It did not read `0x1E`. The write is the
  command `06` on channel 1, laid out as the colour write of unit 0009 is: a length of 14 in bytes 3 and 4, low byte
  first, `00 00` in bytes 5 and 6, and from byte 7 the data: the data type `0D 00`, the count 7, and then one entry
  for each connection in the order of the entries of `0x20`, `00` for a connection with no LEDs (EXT and the ports 1
  and 2) and `01 03` for one with LEDs (the ports 3 to 6). The length counts the bytes of the data. The entries are
  of two lengths, so the layout of the data is read from this one example
- At that time a software colour still showed. The same trace has `08 14 0N 01` for N from 1 to 6 earlier in its
  setup, and a write of `07 00 00` to the endpoint `0x18`, whose meaning is unknown and which may concern the fan modes
- The test script of the orchestrating session wrote exactly that configuration in those four exchanges, then
  `08 15 01`, then opened `0x22` on channel 0 and wrote the colour: the LEDs lit, red. The same with `08 14 0N 01` for
  N from 1 to 6 before it lit too, green, so `08 14` is not needed. A read of `0x1E` afterwards answered
  `0D 00 07 00 00 00 01 03 01 03 01 03 01 03` from byte 3
- The configuration is stored in the controller: after the test, the colour of sensordeck lit the LEDs again, as it
  will until iCUE rewrites the configuration. iCUE's configuration, too, outlasted a full power cut
- The answer of `0x20` on the host gives mode `00` with 0 LEDs for EXT, mode `03` with 0 LEDs for the ports 1 and 2, and
  mode `02` with 4 LEDs for the ports 3 to 6, as unit 0009 records

Not established: what `03` means, which is taken to be an LED type and likely stands for the model of fan with 4 LEDs
of the host, and what a fan with another LED count needs; what `06` means, the value that iCUE left; whether
`08 15 01` is needed, since it was sent in both phases that lit and never left out; whether a write of `0x1E` wears the
memory of the controller; and what iCUE does with the configuration that sensordeck writes.

### Steps to reproduce

1. On a machine with a Commander Core XT whose fans have 4 LEDs each, run iCUE, so that it rewrites `0x1E`
2. End iCUE, and run `sensordeck run` with `commander_color` in its config file
3. The LEDs stay dark while `run` holds the session, and no line on standard error says why

### Root cause

In software mode the controller lights only the ports that the configuration stored in `0x1E` enables. iCUE leaves a
configuration there that does not enable them for a host that sends colours as sensordeck does, so the colour write
succeeds and lights nothing. Unit 0009 never writes `0x1E`, by its R10, because the LEDs lit without it while the
configuration that an earlier run of OpenLinkHub had stored was still there. This is established by the observations
above: the LEDs were dark with iCUE's configuration in every version, and lit after the configuration of OpenLinkHub
was written, with nothing else changed but `08 15 01`. What in iCUE's configuration disables the ports is not known.

### The fix

At each setup, after the read of `0x20`, `run` reads `0x1E`. When its content is not the configuration that enables
exactly the ports with LEDs, `run` writes that configuration, followed by `08 15 01`, before it opens `0x22` and
writes the colour, and writes a line to standard error that names iCUE as a likely cause. When the content is already
that configuration, nothing is written to `0x1E`. A port with an LED count other than 4 makes `run` leave `0x1E` as it
is, with a line, since what such a fan needs is not known. Windows reaches the controller through the same protocol
package, so the fix holds there too.

The unit is built in the sandbox, which has no controller. Every requirement is checked by automated tests with the
fake controller. The check on the device, that the LEDs light after iCUE has rewritten `0x1E`, is the human's, and the
automated loop does not wait for it.

## Requirements

"R*n* of unit 000*k*" names a requirement of the spec of that unit. Every requirement of unit 0009 and of the units
after it that this spec does not change still holds. Commands are written as their data bytes after the report ID
byte, every byte not named is `0x00`, and "an exchange" and "succeeds" mean what they mean in unit 0009. "A read of an
endpoint on channel 1" is the four exchanges of R6 of unit 0009: `08 05 01 01 <endpoint>`, `08 0D 01 <endpoint>`,
`08 08 01 <endpoint>` and `08 05 01 01 <endpoint>`.

### The configuration

- **R1** The entries of the answer of `0x20` are those of R7 of unit 0009, *c* of them. Entry 0 is named `EXT` and
  entry *i*, for *i* from 1, is named `port <i>`. An entry "has LEDs" when its mode is 2 and its LED count is not 0.
  "The wanted configuration" is the 3 + *c* + *k* bytes `0D 00`, *c*, and then for each entry in order `01 03` when it
  has LEDs and `00` when it does not, *k* being the number of entries that have LEDs. For the entries of the host it is
  `0D 00 07 00 00 00 01 03 01 03 01 03 01 03`
- **R2** "The configuration write" is the command `08 06 01`, then the length of the wanted configuration in bytes 3 and
  4, low byte first, then `00 00` in bytes 5 and 6, then the wanted configuration from byte 7, and zeros up to 96 data
  bytes. For the entries of the host it starts with `08 06 01 0E 00 00 00 0D 00 07 00 00 00 01 03 01 03 01 03 01 03`
- **R3** The configuration is "known" when every entry that has LEDs has an LED count of 4. When it is not known, no
  command to `0x1E` and no `08 15 01` is written in the setup, and the setup goes on as R6 of unit 0009 says

### The setup

- **R4** A setup makes these exchanges, in this order, which replaces the list of R6 of unit 0009:
    1. `08 01 03 00 02`, enter software mode
    2. the read of `0x20` on channel 1, whose read answer R7 of unit 0009 checks
    3. when the configuration is known, the read of `0x1E` on channel 1. Its read answer succeeds when it is valid as
       every answer is, whatever its data type and content
    4. when the configuration is known and bytes 3 to 3 + *L* − 1 of that read answer are not the wanted configuration,
       *L* being its length: `08 05 01 01 1E`, `08 0D 01 1E`, the configuration write, `08 05 01 01 1E`, and then
       `08 15 01`. When they are the wanted configuration, none of these is written
    5. `08 0D 00 22`, open the LED endpoint on channel 0
    6. the colour write of R8 of unit 0009

  The rest of R6 of unit 0009 holds: a setup whose exchanges all succeed starts a session that is held, and the first
  exchange that does not succeed ends the setup, no further exchange of the setup is made, and the teardown of R12 of
  unit 0009 follows. The configuration write and `08 15 01` are exchanges like the others, so a write that fails or an
  answer that does not come leaves out the close of channel 0 in that teardown
- **R5** Setups are made when R14 of unit 0009 says, and each makes steps 3 and 4 afresh. In a held session nothing is
  written to `0x1E` and no `08 15 01` is written
- **R6** R10 of unit 0009 holds with this change: a session writes no command but those of R4, of R9 and of R12 of
  unit 0009. Among others it still never writes `08 02 13`, `08 14 0N 01`, a command to the endpoint `0x18`, a colour
  write of zero triples, or a command that opens on channel 0 an endpoint other than `0x22`. Nothing is written to the
  node of interface 1
- **R7** The error of a setup that ends at an exchange of step 3 or 4 names the exchange, as R15 of unit 0009 asks:
  `read of endpoint 0x1e on channel 1`, `open of endpoint 0x1e on channel 1`, `close of endpoint 0x1e on channel 1`,
  `write of the LED port configuration` and `command 08 15 01`, followed by what failed, as the other exchanges are
  named today

### The lines

- **R8** `run` writes to standard error, besides the lines of R15 of unit 0009:
    - `controller LED ports configured: the stored configuration did not enable the ports with LEDs, iCUE may have
      changed it`, written on one line, after each setup whose configuration write succeeded, whatever the rest of the
      setup gives, and before any line that R15 of unit 0009 writes for that setup
    - `controller LED ports left as stored: <name> has <n> LEDs, and only fans with 4 LEDs are known`, with the name of
      R1 and the LED count of the first entry that has LEDs and a count other than 4, written for the first setup whose
      configuration is not known and not again in that run of `run`

  A setup whose configuration is known and already stored writes no line of R8. Standard output does not change

### Platforms

- **R9** The fix is in the protocol and the session of `internal/commandercore`, which Linux and Windows share by unit
  0016, so `run` with `commander_color` on Windows makes the same exchanges and writes the same lines. The five
  commands of the suite of `AGENTS.md` succeed

### The documents

- **R10** `docs/commander-core-xt.md` gains a section on the endpoint `0x1E`, the LED port configuration, which says in
  the project's own words: the read and its layout; the write and its layout, with the example of the host; the two
  configurations seen, the one of OpenLinkHub and the one left after iCUE had run, with the bytes after the seven `06`
  taken to be stale; that the configuration is stored in the controller and survives a power cut; that in software
  mode the colour lights only the ports it enables, which the observations of this unit show; the rule of R1 and R3,
  with `03` written only for ports with 4 LEDs; and `08 15 01` after the write. Its section "A session of run" gives
  the setup of R4, and its list of commands that sensordeck does not send no longer holds `0x1E` and `08 15 01`. Its
  "How it was learned" names the device tests of this unit. Its section "Unknown" names at least: the meaning of the
  type `03` and what a port with another LED count needs; the meaning of `06`; whether `08 15 01` is needed; whether
  the bytes after the seven `06` are stale; whether a write of `0x1E` wears the memory of the controller; and what iCUE
  does with the configuration that sensordeck writes. Nothing else that the document says is removed
- **R11** The section `## The Corsair iCUE Commander Core XT` of `README.md` says in plain words, besides what it says
  today:
    - that the controller stores which of its ports have LEDs, that iCUE rewrites this, and that afterwards the LEDs
      stay dark in software mode, on Linux and on Windows
    - that `run` checks it at the start of each session and repairs it when it does not match the fans the controller
      reports, writing `controller LED ports configured` with the rest of the line of R8, and that the repair stays
      stored in the controller until iCUE changes it again
    - that the repair is only made when every port with LEDs has 4 of them, the one kind of fan it was tested with, and
      that otherwise `run` writes `controller LED ports left as stored` and the LEDs may stay dark
    - that running iCUE and sensordeck at the same time is not supported, beside the other programs it names today

### Tests

- **R12** The fake controller of R22 of unit 0009 also keeps an LED port configuration, stored apart from the mode, the
  channels and the colour, so that it survives a return to hardware mode, the software-mode timeout and a close and
  reopen of its node. A test sets its bytes, and the default is the wanted configuration of the default entries,
  `0D 00 07 00 00 00 01 03 01 03 01 03 01 03`. The read of `0x1E` on channel 1 answers them from byte 3. A write
  `08 06 01` answers status `00` and stores the bytes of its data, from byte 7 for the length in bytes 3 and 4, when
  `0x1E` is open on channel 1, and status `03` otherwise. `08 15 01` answers status `00`. Like every other command,
  these answer `03` outside software mode. A test can also set "the lighting configuration", by default the default
  configuration, or none: after a colour write the LEDs show the colour when none is set or when the stored
  configuration equals it, and stay dark otherwise. A test can read the stored configuration. Its own tests show each
  of these, and the tests of the fake of unit 0009 keep what they expect
- **R13** Unit tests of the protocol cover: the wanted configuration and the configuration write of R1 and R2 for the
  entries of the host, with the bytes of the trace; for entries with LEDs on EXT and the ports 1 and 2 only; for an
  entry of mode 2 with 0 LEDs, and of mode 3 with 4 LEDs, both of which are `00`; and the rule of R3 for an entry of
  mode 2 with 8 LEDs
- **R14** Unit tests of the session against the fake controller cover:
    - with the default configuration stored: the bytes of every write of a setup, in order, which hold the read of
      `0x1E` and no configuration write and no `08 15 01`
    - with the configuration of R10 after iCUE stored, `0D 00 07 06 06 06 06 06 06 06 00 00 01 03 01 03 01 03 01 03`:
      the bytes of every write of the setup in the order of R4, the stored configuration of the fake afterwards equal
      to the wanted one, the LEDs showing the colour, and the setup telling that it wrote the configuration
    - a stored configuration that differs from the wanted one only in its last byte, which is written
    - entries with a port of 8 LEDs: no command to `0x1E` and no `08 15 01`, the setup succeeds, and it tells that the
      configuration is not known and which entry made it so
    - a failure of the read of `0x1E`, of the configuration write and of `08 15 01`, each with status `03` and with no
      answer, each followed by the teardown of R4 and no further command of the setup, with the error naming the
      exchange as R7 says
    - the exchanges of R4 at every setup of a session that is set up again, and nothing written to `0x1E` by readings
- **R15** Unit tests of `run` cover the lines of R8: the line of a configuration write after each setup that made one,
  before a `controller setup failed` line of the same setup; no line when the configuration was already stored; and the
  line of a configuration that is not known written once over several setups
- **R16** End-to-end tests run `run` against the fake screen, the tree of R35 of unit 0004 and the fake controller, with
  the config of R26 of unit 0009, and cover:
    - with the configuration after iCUE stored in the fake and the lighting configuration set to the default: the line
      of R8 on standard error, the stored configuration equal to the wanted one, the LEDs showing `#ff0000` while `run`
      runs, and over the whole run exactly one configuration write and one `08 15 01`
    - with the default configuration stored: no line of R8, no configuration write and no `08 15 01`
- **R17** The existing unit and end-to-end tests of the suite pass. Those that list the writes of a setup are changed to
  hold the read of `0x1E` of R4, and no other expectation of them changes

## Decisions

- **D1** The unit is written from observed traffic and device tests, and no code of OpenLinkHub, liquidctl, OpenRGB or
  iCUE is read, translated or copied, as the context says and `AGENTS.md` requires
- **D2** The unit supersedes `0009-commander-led-colour`, because it changes requirements of that unit rather than
  adding behaviour beside them: R6, the exchanges of a setup; R10, which forbade every command to `0x1E` and `08 15 01`;
  R20 and R21, what the README and the controller document say; and R22 and R24, the fake and the tests of the session.
  The other requirements of unit 0009 stay true and are not restated
- **D3** `run` reads `0x1E` and writes it only when its content differs, R4, rather than writing it at every setup as
  OpenLinkHub did. The configuration is stored in the controller and survives a power cut, so it is likely kept in
  flash, whose wear nobody has measured. A setup is made at every start of `run` and after every lost session, so
  writing every time would add a write of that memory at each start and each retry for nothing. The read costs four
  exchanges of about 2 milliseconds each, and writes nothing. In the case nobody supports, iCUE running beside `run`,
  rewrites happen at most once per setup, which the back-off of R14 of unit 0009 makes at most once in 5 minutes once
  failures repeat
- **D4** The comparison is of the bytes of the read answer that the configuration write would fill, from byte 3 for the
  length of the wanted configuration, R4. The read answer is not parsed: its layout is known only from the configuration
  of OpenLinkHub, and the content that iCUE left does not fit it (seven `06` for seven connections of two possible
  lengths). Any difference, iCUE's configuration, another program's, another type byte or a data type that is not
  `0D 00`, makes the write, which is what brings the LEDs back. A configuration that enables the right ports with
  another type is rewritten with `03`, which is the cost of not knowing what the types mean, and only happens on a
  controller whose fans all have 4 LEDs
- **D5** The type byte is `03` for a port with LEDs, the one value seen to light the fans, and it is written only when
  every port with LEDs has 4 of them, R3, which is the one fan seen. For other counts three ways were weighed. Writing
  `03` anyway could store a configuration that is wrong for those fans, in a memory that persists, and overwrite a
  configuration that was right for them, so that a setup that works today would stop working. Skipping those ports
  would write `00` for them and disable LEDs that may light today. Refusing the session would take away the colour and
  the probes from a controller on which unit 0009 may work. Leaving `0x1E` as stored with a line keeps the behaviour of
  unit 0009 on hardware nobody has seen and tells the user why the LEDs may stay dark, and a later unit can add a type
  once one is observed
- **D6** "Has LEDs" is mode 2 with an LED count other than 0, R1, the same entries that R7 of unit 0009 counts the LEDs
  of, so that the ports enabled are those that the colour write sends triples for. An entry of another mode is `00`
  whatever its count, as R7 of unit 0009 does not count it
- **D7** `08 15 01` follows the configuration write, R4, because both phases of the device test that lit the LEDs sent
  it and none left it out, and OpenLinkHub sent it in the same place. It is sent only after a write, since unit 0009
  lit the LEDs without it while the stored configuration was right. Its meaning is unknown, and testing without it is
  item 1 of "Beyond the request"
- **D8** The configuration is written as the device test wrote it: the close, the open, the write and the close of
  `0x1E` on channel 1, then `08 15 01`, and only then the open of `0x22` and the colour write. OpenLinkHub opened
  `0x22` before it wrote `0x1E`; the order of the device test is kept because it is the one sensordeck's own commands
  were seen to light with, and `0x1E` is on channel 1 while `0x22` is on channel 0, so either order keeps one endpoint
  per channel
- **D9** `08 14 0N 01` and the write to `0x18` are not sent, R6. The device test lit the LEDs without `08 14`, and the
  write to `0x18` probably concerns the fan modes, which sensordeck does not touch, as unit 0009 decided
- **D10** The line of a rewrite names iCUE, R8, because iCUE is the one program seen to change the configuration, and a
  user who sees the line after using iCUE learns why the colour was gone. It is written for each rewrite and not once,
  because each one is a write of the controller's memory and means that something changed the configuration again. The
  line of a configuration that is not known is written once in a run, as the line of a failed setup is in unit 0009,
  because it says the same at every setup
- **D11** The setup does not read `0x1E` back after the write. The device test read back the configuration that was
  written, and whether the LEDs light is the check that matters, which only the human can make
- **D12** A setup that ends after its configuration write still writes the line of R8, because the write has been
  made and stays stored whatever the rest of the setup gives
- **D13** The fake controller models the bug by its lighting configuration, R12: with the configuration after iCUE
  stored, the colour write leaves the LEDs dark, so that the end-to-end test of R16 shows the effect of the fix and not
  only its bytes. The model is the one observed: dark with iCUE's configuration, lit with OpenLinkHub's

## Non-goals

- Types other than `03`, and fans with an LED count other than 4
- Colours per port, effects, and any change of the colour write or of the readings of a session
- `08 14 0N 01`, the endpoint `0x18`, `08 02 13`, fan modes and fan settings
- Writing `0x1E` in `sensors` or without `commander_color`; `sensors` does not change
- Restoring iCUE's configuration at the end of a session, or coordinating with iCUE in any other way
- Running iCUE and sensordeck at the same time
- The check on the device and on Windows, which is the human's

## Beyond the request

1. A device test of the configuration write without `08 15 01`, and dropping it when it is not needed. It costs a run
   of a test script with the human watching, after iCUE has rewritten the configuration, and a small later unit.
   Without it `run` sends one command more after each rewrite, whose effect is unknown and which lit the LEDs in every
   test it was part of
2. An option in the config file that turns the repair off, for a user who wants iCUE's configuration kept. It costs a
   key, its checks and tests. Without it such a user loses iCUE's configuration whenever `run` starts with
   `commander_color`, which iCUE rewrites when it runs again, and can leave `commander_color` out to keep it
