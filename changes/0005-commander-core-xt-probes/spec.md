# The temperature probes of the Corsair Commander Core XT

## Context

The two temperatures the human most wants on the screen are the front intake of the case and the power connector of
the GPU. Neither has a hwmon driver on Linux: both are probes plugged into a Corsair iCUE Commander Core XT, a fan and
lighting controller on USB. Unit `0004-linux-sensors-and-run` left the controller out and named a later unit for it.
This is that unit.

The human decided earlier that liquidctl, the tool that reads this controller today, must not be needed by
sensordeck, and chose a driver written in Go inside sensordeck. liquidctl is licensed under the GPL-3.0. Its code was
not read for this unit and must not be read, translated or copied for it. The driver is written from the facts below
alone. They come from a black-box trace with `strace` of the hidraw traffic of one run of `liquidctl status` on the
human's machine, a capture that is kept locally and is not committed, and from the sysfs of that machine. Facts about
how a device talks are not protected by copyright, and the documents of this unit state them in the project's own
words.

The unit is built in the sandbox, which has no controller. Every requirement is checked by automated tests with a
fake controller made of a directory tree and pseudo terminals, as the fake screen of unit 0003 is. The check on the
real controller is the human's.

### What was observed

- The controller is the USB device `1b1c:0c2a`, whose `HID_NAME` is `Corsair CORSAIR iCUE COMMANDER CORE XT`
- It has two hidraw nodes, one for each of its USB interfaces 0 and 1. On the host they are `hidraw9` and `hidraw10`.
  The `uevent` file of the entry under `/sys/class/hidraw` holds `HID_ID=0003:00001B1C:00000C2A` for both, and
  `HID_PHYS` ends in `/input0` for the node of interface 0 and in `/input1` for the other. liquidctl used the node of
  interface 0 only
- On the host, the user reaches both nodes through an ACL that the `uaccess` tag of a udev rule of liquidctl gives
- Every write is 97 bytes: the report ID byte `0x00` followed by 96 bytes. Every read gives 96 bytes. Bytes not named
  below are `0x00`
- liquidctl wrote eight commands, each followed by one read of the answer. The written bytes after the report ID are:

  | Step | Bytes | Meaning |
  |---|---|---|
  | 1 | `08 01 03 00 02` | enter software mode |
  | 2 | `08 0D 00 17` | open the endpoint `0x17`, the fan speeds |
  | 3 | `08 08 00` | read the open endpoint |
  | 4 | `08 05 01` | close the endpoint |
  | 5 | `08 0D 00 21` | open the endpoint `0x21`, the temperatures |
  | 6 | `08 08 00` | read the open endpoint |
  | 7 | `08 05 01` | close the endpoint |
  | 8 | `08 01 03 00 01` | return to hardware mode |

- In every answer, byte 0 is `0x00`, byte 1 is the second written byte of its command (`01`, `0D`, `08` or `05`), and
  byte 2 is `0x00`. Byte 2 is presumably a status that is not `0x00` on an error, which was not observed
- The answer of step 6 is `00 08 00 10 00 02 00 75 01 00 E3 00` followed by zeros: the data type `10 00` in bytes 3
  and 4, the count `02` in byte 5, and from byte 6 three bytes for each probe: a status byte, `00` for a connected
  probe, and the temperature in tenths of a degree Celsius, 16 bits, low byte first. `0x0175` is 37.3 °C and `0x00E3`
  22.7 °C, and liquidctl wrote them as `Temperature 0` and `Temperature 1`
- The answer of step 3 is `00 08 00 06 00 06` followed by zeros: the data type `06 00`, the count 6, and from byte 6
  six fan speeds of 16 bits, all 0, because no fan is connected
- The bytes of an answer past what its command fills keep the content of an earlier answer. The answer of step 1
  held the temperatures of an earlier run
- The answers of steps 1 and 8 came after about 65 milliseconds, the others after about 2 milliseconds
- A probe status other than `00`, and a byte 2 other than `00`, were not observed

The human thinks that probe 0 is at the power connector of the GPU, because it is the warmer one and rose from 31.0
to 37.3 °C under load, and probe 1 at the front intake, at about 22 °C. This is not verified.

## Requirements

"R*n* of unit 000*k*" names a requirement of the spec of that unit. "The controller" is a Commander Core XT, and "a
reading of the controller" is one run of the sequence of R5 that gives its metrics. Offsets of an answer count from 0
over the 96 bytes read. "Software mode" and "hardware mode" are what the commands of steps 1 and 8 set.

### Metrics

- **R1** The table of metrics of R1 of unit 0004 gets one more row, last, after `nvme.<n>.temp`:

  | Metric | Unit | Source |
  |---|---|---|
  | `commander.<n>.temp` | `°C` | the probe *n* − 1 of the controller, as liquidctl's `Temperature <n − 1>` |

  So `commander.1.temp` is the first probe and `commander.2.temp` the second. The rule of R2 of unit 0004 for
  numbered metrics holds for the new pattern, so `commander.0.temp` and `commander.01.temp` are no metric names, and
  `commander.3.temp` is a metric name that is unavailable on a controller with two probes. The config of `run`
  therefore accepts these names in `tags` with no other change
- **R2** The metrics of the controller follow R3 of unit 0004: they are read on demand, and a missing controller, a
  node that cannot be opened, for instance for lack of access, and every failure of R6 make them unavailable, which is
  never an error of `sensors` or `run`

### Finding the controller

- **R3** The controller is found as the screen is found by R7 of unit 0003, under the same sysfs root and dev root,
  and so through `SENSORDECK_SYSFS_ROOT` and `SENSORDECK_DEV_ROOT` in the tests: an entry `hidraw<N>` of
  `<sysfs root>/class/hidraw` whose `device/uevent` holds a `HID_ID` line with the bus `0003`, the vendor `1B1C` and
  the product `0C2A`, read as hexadecimal numbers, and a `HID_PHYS` line whose value ends in `/input0`. Its node is
  `<dev root>/hidraw<N>`. An entry with any other `HID_PHYS`, which covers the node of interface 1, is never opened
  and never written. When several entries qualify, the one with the lowest *N* is used. `--device` names the screen
  only, and has no effect on the controller
- **R4** The node is opened for reading and writing at the start of every reading of the controller and closed at
  its end. A reading opens no node when none of the metrics of the controller is wanted

### Reading the probes

- **R5** A reading of the controller writes the eight commands of the table of "What was observed", in that order,
  each as 97 bytes in one write: `0x00`, then the bytes of the table, then `0x00` up to 96 bytes. After each write it
  reads one answer, waiting no longer than the time limit of the reply, which is 1 second and is a value the source
  takes, so that a test gives a shorter one. The driver writes nothing to the controller but these commands, and
  never a command of steps 2 to 7 outside a sequence that step 1 opened
- **R6** An answer is valid when 96 bytes were read, its byte 1 is the second written byte of its command, and its
  byte 2 is `0x00`. For the answer of step 6, the data type in bytes 3 and 4 must also be `10 00`, and the count *c* in
  byte 5 must leave room for its entries: 6 + 3·*c* ≤ 96. The answer of step 3 is checked as every other answer and
  its content is not used. A failed or short write, an answer that does not arrive within the time limit, and an
  answer that is not valid end the sequence: no further command of steps 2 to 7 is written, and R7 follows. Every
  metric of the controller is then unavailable, except when the answer of step 6 was already valid
- **R7** Once the write of step 1 has been made, whatever its result, the write of step 8 is made before the reading
  returns, followed by one read of its answer within the time limit. This holds when a write or an answer of steps 1
  to 7 failed, and when the stop of R8 came. A failure of step 8 does not make the probes that step 6 gave
  unavailable
- **R8** The reading takes the stop of the `Source` contract of unit 0004. When stop is done before step 1, nothing
  is written. When it is done later, the reading writes no further command of steps 2 to 7, and makes step 8 as R7
  says. On SIGINT or SIGTERM, `run` therefore leaves the controller in hardware mode, and still ends within the
  3 seconds of R23 of unit 0004. R22 makes `sensors` do the same
- **R9** From a valid answer of step 6, the entry of probe *i*, for *i* from 0 to *c* − 1, is the three bytes at
  offset 6 + 3·*i*. A probe whose status byte is `0x00` gives `commander.<i + 1>.temp` with the value of the next two
  bytes, low byte first, read as a signed 16-bit number and divided by 10. A probe whose status is not `0x00` is
  unavailable and keeps its number, so that the other probe is not renumbered. For the answer of "What was observed"
  the readings are `commander.1.temp` 37.3 and `commander.2.temp` 22.7
- **R10** A reading of several metrics of the controller at the same moment, as `sensors` makes and as a round of
  `run` makes, runs the sequence once

### Platforms and layout

- **R11** The driver and the fake controller are limited to Linux by build constraints, and the test of the layout of
  unit 0003 checks this as it does for the Linux sources and the fake screen. The code of the commands names the
  driver only in files limited to Linux, as R14 of unit 0004 says. In a build for Windows nothing changes: `sensors`
  and `run` fail as R14 of unit 0004 says. The five commands of the suite of `AGENTS.md`, the build for Windows among
  them, succeed

### The commands

- **R12** `sensordeck sensors` writes the lines of R15 of unit 0004 for the metrics of the controller as for every
  other metric. R15 of unit 0004 no longer says that it opens no device node: it opens no node of the screen, takes no
  account of `--device`, and opens the node of the controller as R4 says
- **R13** `sensordeck run` sends a mapped metric of the controller to its tag as any other metric, by R21 of unit 0004
- **R22** `sensordeck sensors` catches SIGINT and SIGTERM while it reads its metrics, as `run` does by R23 of unit
  0004, and passes the stop to every source. A signal during the reading of the controller therefore ends it as R8
  says, with the write of step 8, and the process does not end before step 8 has been written and its answer read or
  its time limit passed. After a signal during the reading, `sensors` writes nothing to standard output, writes
  `sensordeck: interrupted` to standard error and ends with the exit status 1, no later than 3 seconds after the
  signal. A signal before or after the reading keeps the effect it has today

### Access on Linux

- **R14** `udev/70-sensordeck.rules` holds, besides the rule of the screen, a rule of the same form for the USB ID
  `1b1c:0c2a`, with a comment that names the controller, so that a user without liquidctl reaches its nodes. The rule
  of the screen does not change

### The documents

- **R15** A document of its own in `docs/`, beside the protocol of the screen, describes the controller as far as this
  unit uses it, in the project's own words: the identity of the device and its two interfaces, the size of the
  reports, the eight commands, the layout of an answer and of the data of the endpoints `0x17` and `0x21`, the stale
  bytes of an answer, and the times of the answers. It says that it was learned from a black-box trace of the hidraw
  traffic of another tool on one machine, and that no code of that tool was read. It has a section "Unknown" and a
  section "Assumptions", which name at least: the meaning of byte 0 and byte 2 of an answer, the meaning of a probe
  status other than `00`, whether the temperature is signed, which probe sits at which port of the controller, and
  whether the fan steps 2 to 4 are needed before the temperature steps
- **R16** `README.md` names `commander.<n>.temp` in the table of metrics of "Showing sensor values on the screen",
  with its unit and the controller as its source, and says that *n* counts the probes of the controller from 1. The
  section "Linux: access to the screen" says that the rule file also gives access to the controller, and that its
  metrics are left out when the controller cannot be opened

### Tests

- **R17** A fake controller, test support in the style of the fake screen of unit 0003, makes the sysfs entries of
  both interfaces under the sysfs root, with the `HID_ID` and `HID_PHYS` of "What was observed", and their nodes under
  the dev root, backed by pseudo terminals. It answers each written command as the controller did, with probes the
  test sets, and keeps every write it received on each node and the mode the last mode command set. A test can make
  it give no answer to a chosen step, an answer whose byte 1 is wrong, an answer whose byte 2 is not `0x00`, a data
  type other than `10 00`, and a count too large for the report
- **R18** Unit tests against the fake controller cover: the readings and the eight writes of R5, byte for byte, for the
  answer of "What was observed"; nothing written to the node of interface 1; a probe with a status other than `00`,
  whose number the other probe keeps; a negative temperature; no controller; a node that cannot be opened; only an
  entry of interface 1; two qualifying entries, of which the lower number is used; each failure the fake can give, at
  steps 1, 3 and 6 at the least, each ending with the write of step 8 and with no write of steps 2 to 7 after the
  failure, and the readings unavailable; a failed step 8 after a valid step 6, which keeps the readings; stop done
  before step 1, which writes nothing, and stop done during the sequence, which writes step 8; and a reading in which
  no metric of the controller is wanted, which opens no node. The tests of the time limit take a time limit of
  100 milliseconds or less
- **R23** A unit test of `sensors`, with sources the test gives and a stop the test ends during the reading, covers
  the output, the error line and the exit status of R22, and that the stop reached the sources. An end-to-end test
  runs `sensors` against the fake controller made to give no answer to step 6, sends SIGINT once step 6 has arrived,
  and checks that step 8 arrived after it, that the fake is in hardware mode, that the exit status is 1 with the line
  of R22, and that the process ended within 3 seconds of the signal
- **R19** The unit tests of R2 of unit 0004 cover `commander.1.temp`, `commander.0.temp` and `commander.01.temp`,
  and the test of the units of each row covers the new row
- **R20** An end-to-end test runs `sensors` against the tree of R34 of unit 0004 together with the fake controller
  giving the answer of "What was observed". Its standard output is that of R15 of unit 0004 followed by the two lines
  `commander.1.temp\t37.3\t°C` and `commander.2.temp\t22.7\t°C`, and afterwards the fake has received the eight
  commands once on the node of interface 0 and is in hardware mode. The existing end-to-end tests of `sensors` and
  `run`, without a controller, keep their exact output
- **R21** An end-to-end test runs `run` against the fake screen and the fake controller with a config that maps
  `commander.1.temp` to tag 4 and `commander.2.temp` to tag 5. A value report holds the pairs (4, 37) and (5, 23), and
  after SIGTERM the process ends with status 0 and the fake controller is in hardware mode

## Decisions

- **D1** The driver is written from the observed traffic alone, and no code of liquidctl is read, translated or
  copied. liquidctl is GPL-3.0 and this project is MIT, as `AGENTS.md` requires. What a device expects on the wire is a
  fact about the device, and the documents state those facts in the project's own words
- **D2** Every reading enters software mode, reads and returns to hardware mode at once, as liquidctl did, so that the
  controller spends as little time as possible out of hardware mode. In hardware mode the controller drives its fans
  and lighting by itself, from its own settings; what it does in software mode with nobody sending it fan or lighting
  settings is not known, and a machine whose fans stop or race because a monitor ended at the wrong moment is a far
  worse failure than a missing temperature. That is why R7 restores hardware mode after every failure and on a stop,
  and why `run` does not keep the controller in software mode between rounds, which would have saved the two slow
  mode changes of about 65 milliseconds each per round. A round of `run` that reads the controller takes about
  140 milliseconds longer; with the shortest interval of 100 milliseconds the rounds then run late, which R21 of unit
  0004 already handles
- **D3** The sequence is the eight steps liquidctl wrote, the fan steps 2 to 4 included, although this unit uses no
  fan speed. It is the one sequence known to work on the device. Leaving the fan steps out may well work too, but is
  not observed, and it saves about 4 milliseconds a reading. The question is recorded as unknown in the document of
  R15, so that a later unit can drop the steps after a check on the device
- **D4** An answer is checked on its echo in byte 1 and on byte 2, and the temperature answer on its data type and
  count. Because the bytes past an answer keep stale content, an answer that the controller did not fill could still
  look like temperatures; the echo, which every answer overwrites, and the data type tell such an answer apart. Byte 2
  is taken as a status, an assumption: it was `0x00` in every answer seen, and if the assumption is wrong the metrics
  are missing on the device, which the human's check shows at once and a correction mends cheaply, while a stale value
  shown as a reading would go unnoticed. Byte 0 is not checked, because nothing is known of it
- **D5** A probe whose status is not `00` is unavailable, an assumption: only `00` was observed, for two connected
  probes. Reading the value of a probe that the controller marks otherwise could put a false temperature on the screen,
  and an unavailable metric keeps the last true value there, as D3 of unit 0004 decided
- **D6** The temperature is read as a signed 16-bit number. Both readings seen were positive, so the sign is an
  assumption. Signed costs nothing for any temperature a case reaches, while unsigned would turn a probe below 0 °C
  into more than 6000 °C; R4 of unit 0004 clamps a negative value to 0 for the screen anyway
- **D7** The metric names are `commander.<n>.temp`, numbered from 1 as `ram.<n>.temp` and `nvme.<n>.temp` are, so the
  registry of metrics takes them as one more numbered row of the same shape and the config accepts them with no new
  rule. `commander.1.temp` is the probe that the controller lists first and liquidctl calls `Temperature 0`. Names that
  say where a probe sits, such as `gpu.connector.temp`, were rejected: which port is which is not verified, and the
  place of a probe is the user's choice of cabling, which the tags of `run` already map to what the theme shows.
  `commander` names the controller family and not the model, which keeps the name short; a second model is a later
  unit's decision
- **D8** The node of interface 0 is told apart by `HID_PHYS` in the same `uevent` file the discovery of the screen
  already reads, and not by the resolved sysfs path, whose `:1.0` component would need the entry to be followed as a
  link. The kernel's USB HID driver writes the interface number as `/input<number>` at the end of `HID_PHYS`, as both
  nodes on the host show. The node of interface 1 is never opened, because liquidctl did not use it and nothing is
  known of what it does
- **D9** The node is opened anew for every reading and closed after it. A replug of the controller under another
  hidraw number then needs nothing of `run`, no state is carried between readings, and a node that is closed between
  readings holds no stale input report at the next one. The cost is a look at sysfs and an open per reading, a
  fraction of a millisecond
- **D10** A missing controller, a node without access and every failure of the sequence make the metrics unavailable,
  as a missing hwmon file does, and never an error. A machine without the controller is the common case, and `run`
  must not stop because a probe does not answer once
- **D11** The time limit of 1 second for an answer is about 15 times the slowest answer seen. A sequence that fails
  ends after at most two time limits, the failed step and step 8, so `run` still ends within the 3 seconds of R23 of
  unit 0004 when the signal comes in the middle of a reading of a controller that stopped answering
- **D12** The controller gets its own udev rule line, R14, so that sensordeck does not depend on the rule of liquidctl,
  which the human's machine has today only because liquidctl is installed. It goes in the file of the screen, so the
  user installs one file for both, as described in the README
- **D13** `sensors` catches SIGINT and SIGTERM while it reads, R22, which unit 0004 left to the default handling.
  With the default handling a Ctrl-C between steps 1 and 8, a window of about 140 milliseconds, would end the process
  inside the sequence and leave the controller in software mode until the next reading or a replug, for the reasons
  of D2 a failure worth the small cost. An interrupted `sensors` ends with status 1 and writes no readings, because a
  list cut short by the user is no answer to the command, and a nonzero status tells a script so. The limitation that
  remains is a process killed with SIGKILL or a loss of power between steps 1 and 8, which can leave software mode in
  both commands and which no program can prevent. Taking this into the unit was item 2 of "Beyond the request",
  decided by the orchestrating session under the delegation in `AGENTS.md`
- **D14** The driver is for Linux only in this unit, over hidraw. Windows reaches HID devices in another way and is a
  later unit; the Windows build keeps compiling because the driver is limited to Linux by build constraints

## Non-goals

- Fan speeds, which no fan is connected to measure on the host; item 1 of "Beyond the request"
- Any control of fans, pumps or lighting, and any command not named in R5. The driver changes nothing of the settings
  of the controller
- Other Corsair controllers, such as the Commander Core or the Commander Pro, and more than one Commander Core XT at
  a time
- The controller on Windows
- Coordination with other programs that talk to the controller at the same time, such as liquidctl or OpenRGB. Their
  commands can interleave with those of sensordeck; the checks of R6 then make a reading unavailable, and the other
  program may see a wrong answer
- Telling which probe sits at which port of the controller, and names that say where a probe sits
- Showing the controller in `sensordeck devices`, which lists screens
- The traffic of the controller in `--trace`, which traces the screen; item 4 of "Beyond the request"

## Beyond the request

1. Fan speeds as metrics, `commander.fan.<n>` in rpm, from the answer of step 3 that the sequence reads anyway. It
   costs a row of the table, the parsing of six speeds and tests. Without it no fan speed of the controller is shown,
   and the host has no fan on the controller
2. Taken into the unit as R22 and R23, D13: `sensors` catches SIGINT and SIGTERM while it reads the controller, so
   that a Ctrl-C between steps 1 and 8 still restores hardware mode
3. A line on standard error from `sensors` when a controller is found but its node cannot be opened, pointing to the
   README section on access. It costs a way for a source to report a hint and a test. Without it the two metrics are
   silently missing and the user finds the cause in the README
4. `--trace` writes every report to and from the controller, as it does for the screen. It costs a second traced
   transport and tests. Without it a problem with the controller on a real machine is looked into with `strace`, as
   this unit's facts were gathered
