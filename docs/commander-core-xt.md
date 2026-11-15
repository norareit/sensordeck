# The temperature probes and the LEDs of the Corsair iCUE Commander Core XT

This document describes how the computer reads the temperature probes of a Corsair iCUE Commander Core XT and lights
its LEDs in one colour, as far as sensordeck uses the controller. It is written in the project's own words.

## How it was learned

The facts below come from four sources, and from nothing else:

* A black-box trace, made with `strace`, of the hidraw traffic of liquidctl during one run of its status command on one
  machine, and the sysfs of that machine (unit 0005). It gave the eight commands, the endpoints `0x17` and `0x21` and the
  temperatures
* A trace of sensordeck itself on the same machine (unit 0006). It gave the sizes of the reports, together with the
  report descriptor of interface 0 in that sysfs
* A black-box trace, made with `strace`, of the hidraw traffic of OpenLinkHub on the same machine. It gave the channels,
  the endpoints `0x20`, `0x1A`, `0x21` and `0x22`, the colour write, the LED count and the rate of its readings
* Tests made by hand on the device with the LEDs watched (units 0008 and 0009). They showed the effect of software mode
  on the LEDs, the status `0x03` of commands outside software mode and of a second endpoint opened on a channel, and that
  one colour write lights the LEDs

liquidctl's public description of the protocol was read for facts only, for the keepalive. No code of liquidctl,
OpenLinkHub or OpenRGB was read, translated or copied. The traces themselves are kept locally and are not part of the
project. Everything here is therefore what these showed, and what is not shown is named under "Unknown" and
"Assumptions".

## Identity

* USB ID `1b1c:0c2a`, `HID_NAME` `Corsair CORSAIR iCUE COMMANDER CORE XT`
* The controller has two USB interfaces, 0 and 1, and the kernel makes one hidraw node for each. Both entries under
  `/sys/class/hidraw` hold `HID_ID=0003:00001B1C:00000C2A` in their `uevent` file. The `HID_PHYS` line ends in `/input0`
  for interface 0 and in `/input1` for interface 1. The tool of the trace used the node of interface 0 only, and so
  does sensordeck. The node of interface 1 is never opened, because nothing is known of what it does
* On Windows the HID class driver makes one collection for each top-level collection of each interface, and the path of
  a collection of interface 0 holds the field `mi_00`. Interface 0 is the first collection, in the order of the paths,
  of vendor ID `1b1c` and product ID `0c2a` whose path holds that field, with usage page `0xFF42`, usage `0x0001` and
  reports of 385 bytes in both directions

## Reports

The report descriptor of interface 0, read on the host from `/sys/class/hidraw/hidraw9/device/report_descriptor`, is

```
06 42 FF 09 01 A1 01 09 01 15 80 25 7F 75 08 96 80 01 81 02 09 01 15 80 25 7F 75 08 96 80 01 91 02 C0
```

It names the vendor usage page `0xFF42` and declares no report ID. It declares one input report and one output report,
each of 384 (`0x180`) bytes of 8 bits.

A read of a hidraw node gives one report, at most as many bytes of it as the buffer of the read holds, and the rest of
that report is discarded. The other tool read into a buffer of 96 bytes, so its trace showed answers of 96 bytes: they
were the first 96 bytes of reports of 384, cut by its buffer. Because the device declares no report ID, byte 0 of a
read is the first byte of the report, and the offsets below count from it.

A write to the node is 97 bytes: the report ID byte `0x00` followed by 96 bytes. For a device without report IDs,
hidraw takes the report ID `0x00` off, so that 96 bytes go to the device. Bytes of a command that are not named below
are `0x00`.

sensordeck reads each answer into a buffer of the 384 bytes of the input report, so that nothing of it is discarded.
An answer of at least 96 bytes may be valid, and only its first 96 bytes are interpreted.

On Windows a write is the full output report of 385 bytes: the report ID byte `0x00`, the 96 bytes of the command and 288
bytes `0x00`. The HID API refuses a shorter buffer. A read is the full input report of 385 bytes with the report ID first,
and the report ID byte is dropped, so that the answer is the same 384 bytes as on Linux.

## The commands

A reading is eight commands, each written in one write and followed by a read of its answer. The bytes are those after
the report ID byte.

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

sensordeck makes the same eight steps in a reading, so that the controller is out of hardware mode for as short a time
as it can be: it enters software mode, reads and returns at once. What the controller does in software mode when
nobody sends it settings is known for the LEDs, which are dark, and not for the fans it runs, as "Software mode and the
LEDs" and "Unknown" say.

## Channels and endpoints

Every command is `08 <command> <channel> ...`: the byte after the command is a channel. An endpoint is opened on a
channel with `08 0D <channel> <endpoint>`, read with `08 08 <channel> <endpoint>` and closed with
`08 05 01 <channel> <endpoint>`. The eight commands above are those of channel 0, with zeros in the place of the
endpoint byte of a read and a close, which worked.

A channel holds one open endpoint. While the endpoint `0x22` was open on channel 0, opening `0x21` on channel 0 answered
status `03`. After a close of channel 0, the open worked. The status `03` is therefore also the answer to an open on a
channel that has an endpoint open. The trace of OpenLinkHub made every read of a sensor endpoint on channel 1, as four
exchanges: close, open, read, close, such as `08 05 01 01 21`, `08 0D 01 21`, `08 08 01 21` and `08 05 01 01 21`. Every
answer of the trace had status `00`, the close of an endpoint that was not open among them.

## The layout of an answer

Byte 0 is `0x00`. Byte 1 is the second written byte of the command, so `01`, `0D`, `08` or `05`: the answer echoes its
command. Byte 2 is a status: `0x00` in the answers of a sequence that step 1 opened, and `0x03` for commands made without it, as
"Software mode and the LEDs" describes. The data of the endpoint starts at byte 3.

### The endpoint `0x17`, the fan speeds

The answer of step 3 was `00 08 00 06 00 06` followed by zeros: the data type `06 00` in bytes 3 and 4, the count 6 in
byte 5, and from byte 6 six fan speeds of 16 bits. All were 0, because no fan was connected. sensordeck checks this
answer as every other and does not use its content.

### The endpoint `0x21`, the temperatures

The answer of step 6 was `00 08 00 10 00 02 00 75 01 00 E3 00` followed by zeros: the data type `10 00` in bytes 3 and
4, the count 2 in byte 5, and from byte 6 three bytes for each probe. The first byte of an entry is a status, `00` for
a connected probe. The next two are the temperature in tenths of a degree Celsius, 16 bits, low byte first. `0x0175` is
37.3 °C and `0x00E3` is 22.7 °C. The entries fit into the 96 bytes that sensordeck interprets only for a count of 30 or
less.

### The endpoint `0x1A`

The read of `0x1A` answered the data type `09 00` in bytes 3 and 4, the count 6 in byte 5 and six bytes `01`. Their
meaning is unknown. sensordeck checks the answer as every other and does not use its content.

### The endpoint `0x20`, the ports

The read of `0x20` on channel 1 answered the data type `0F 00` in bytes 3 and 4, a count *c* in byte 5, and from byte 6
*c* entries of 4 bytes: a mode of 16 bits and an LED count of 16 bits, both low byte first. They are the entries for the
connector EXT and the ports 1 to 6. On the host they were mode `00` with 0 LEDs for EXT, mode `03` with 0 LEDs for ports
1 and 2, and mode `02` with 4 LEDs for ports 3 to 6, so 16 LEDs: three front fans and one back fan.

sensordeck takes the LED count as the sum of the LED counts of the entries whose mode is 2, and accepts a count from 1 to
29, for 6 + 4·*c* up to 96. This assumes that mode 2 means connected and mode 3 disconnected, and that the triples of a
colour write are taken in the order of the entries, the LEDs of a port one after the other. A single colour does not
depend on the order, and nothing has checked it.

### The endpoint `0x22`, the LEDs, and the colour write

The endpoint `0x22` is the LED endpoint. In software mode, `08 0D 00 22` answered status `00`, and a colour write then
answered status `00` and lit the fans. The colour write is the command `06` on channel 0: `08 06 00`, then in bytes 3
and 4 a length, low byte first, `32 00` for 16 LEDs, then `00 00` in bytes 5 and 6, then the data type `12 00` in bytes 7
and 8, and from byte 9 one triple of red, green and blue for each LED. The length counts the two bytes of the data type
and three bytes for each LED. A write of 97 bytes holds at most 29 triples. For 16 LEDs and `#ff0000` the data bytes start
with `08 06 00 32 00 00 00 12 00` and hold 16 times `FF 00 00`.

For a static colour OpenLinkHub wrote a colour write of all zero triples and, about 2 milliseconds later, the one with
the colour, and wrote neither again until the colour changed. For an animated effect it wrote one about every 22
milliseconds. It wrote the full output report of 385 bytes with the report ID. sensordeck writes one colour write of 97
bytes, without the write of zero triples, because the device test lit the LEDs with it.

## Stale bytes

The bytes of an answer past what its command fills keep the content of an earlier answer. The answer of step 1 held the
temperatures of an earlier run. An answer is therefore checked on its echo, which every answer overwrites, and the
answer of step 6 on its data type.

## Times

The answers of steps 1 and 8, the mode changes, came after about 65 milliseconds, and the others after about 2
milliseconds. A reading takes about 140 milliseconds. sensordeck waits at most 1 second for an answer.

## Software mode and the LEDs

In hardware mode the controller shows the lighting stored in it. In software mode it shows only what the host sends, and
the LEDs are dark when nothing is sent. This is the flicker of a reading of unit 0005: the LEDs of the fans flicker once
at each reading, because the reading enters software mode and returns at once. The status command of liquidctl makes
them flicker in the same way. After a colour write in software mode all the fans turned the colour at once, and after
`08 01 03 00 01` they went back to the stored lighting.

A reading without software mode was tried: the endpoint `0x21` was opened and read with no step 1 before it. Every answer
then had byte 2 `0x03` and stale data in the rest, so a reading without software mode is not possible with what is known.
Byte 2 of an answer is therefore a status, `0x00` in the answers of a sequence that step 1 opened and `0x03` for the
commands made without it.

Without `commander_color`, `sensordeck run` reads the controller at most once per period, of 1 minute unless the config
file says otherwise, to make the LEDs flicker less often. With `commander_color` it holds software mode and sends the
colour, so the LEDs do not flicker.

## The keepalive

liquidctl's description says that the device must be woken after some seconds without data, without saying how many, and
that `08 01 03 00 01` returns it to hardware mode. The timeout is unknown. The trace of OpenLinkHub shows that reads once
a second held software mode with the colour shown for the minutes of the trace. sensordeck keeps one reading a second,
at most 1 second between the start of one reading and the start of the next, whatever the interval is.

## A session of run

With `commander_color`, `run` makes a session. The commands are those after the report ID byte.

* The setup: `08 01 03 00 02`; the read of `0x20` on channel 1 as four exchanges, `08 05 01 01 20`, `08 0D 01 20`,
  `08 08 01 20` and `08 05 01 01 20`; `08 0D 00 22`; and the colour write. The first exchange that does not succeed ends
  the setup
* A reading, made at least once a second: for each of `0x1A`, `0x17` and `0x21` in that order, the four exchanges close,
  open, read and close on channel 1. Only the answer of `0x21` is used. They are the traffic of the trace, copied whole
  because it is the one traffic seen to hold software mode, and take about 24 milliseconds
* The teardown, which follows every setup once `08 01 03 00 02` was written and ends every session: `08 05 01 00 22`, the
  close of channel 0, and `08 01 03 00 01`. The close is left out after a write that failed or an answer that did not
  come, so that a controller that stopped answering costs a stop at most two time limits. The return to hardware mode is
  never left out

OpenLinkHub also wrote at setup `08 02 13`, `08 14 0N 01` for N from 1 to 6, writes with the command `06` to the
endpoints `0x18`, data type `07 00`, and `0x1E`, data type `0D 00`, opened on channel 1, and `08 15 01`. sensordeck does
not send them, and nor does it send a colour write of zero triples or a command that opens another endpoint on channel
0. Their meaning is unknown, and the device test lit the LEDs without them.

## Unknown

* The meaning of byte 0 of an answer, which was `0x00`. It is not checked
* The meaning of a byte 2 other than `0x00` and `0x03`, the two values seen
* The meaning of a probe status other than `00`, which was not seen
* Whether the temperature is signed
* Which probe sits at which port of the controller
* Whether the fan steps 2 to 4 are needed before the temperature steps. They are made, because that is the one
  sequence seen to work
* What the controller does with the fans that it runs in software mode when nobody sends it fan settings. For the LEDs
  this is known: they show what is sent. On the host the controller reported a speed of 0 on all six fan outputs, so it
  runs none of the fans there
* What bytes 96 to 383 of an answer hold. They are not interpreted
* Whether the device needs a write of the full output report of 384 bytes. It was given 96 bytes after the report ID,
  as the other tool gave it, and answered them. Whether it needs it for a colour write of more than 29 LEDs, which does
  not fit into 97 bytes
* The software-mode timeout
* The meaning of the setup commands that sensordeck does not send, listed under "A session of run"
* The meaning of the answer of `0x1A`
* The modes of an entry of `0x20` other than 2 and 3
* The order in which the triples of a colour write map to ports
* Whether the trailing endpoint byte of a read or a close matters. The `08 05 01` of the first traces worked without it

## Assumptions

* An answer whose byte 2 is not `00` is not valid, byte 2 being the status that "Software mode and the LEDs" describes
* A probe whose status is not `00` has no reading, and is left out without renumbering the others
* The temperature is a signed 16-bit number. Both readings seen were positive
* The node of interface 0 is told apart by a `HID_PHYS` that ends in `/input0`, as the kernel's USB HID driver writes it
* On Windows interface 0 is told apart by the field `mi_00` of the path with its usage page, usage and report lengths. The
  usage page and usage are those of the report descriptor above, read on Linux, and none of this has been checked on a
  Windows machine
