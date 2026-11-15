# The temperature probes of the Corsair iCUE Commander Core XT

This document describes how the computer reads the temperature probes of a Corsair iCUE Commander Core XT, as far as
sensordeck uses the controller. It is written in the project's own words.

## How it was learned

The facts below come from a black-box trace, made with `strace`, of the hidraw traffic of another tool during one run of
its status command on one machine, and from the sysfs of that machine. The sizes of the reports come from the report
descriptor of interface 0 in that sysfs and from a trace of sensordeck itself on the same machine. No code of that tool
was read, translated or copied. The trace itself is kept locally and is not part of the project. Everything here is therefore what one run
showed, and what is not shown is named under "Unknown" and "Assumptions".

## Identity

* USB ID `1b1c:0c2a`, `HID_NAME` `Corsair CORSAIR iCUE COMMANDER CORE XT`
* The controller has two USB interfaces, 0 and 1, and the kernel makes one hidraw node for each. Both entries under
  `/sys/class/hidraw` hold `HID_ID=0003:00001B1C:00000C2A` in their `uevent` file. The `HID_PHYS` line ends in `/input0`
  for interface 0 and in `/input1` for interface 1. The tool of the trace used the node of interface 0 only, and so
  does sensordeck. The node of interface 1 is never opened, because nothing is known of what it does

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
nobody sends it settings is not known.

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

## Stale bytes

The bytes of an answer past what its command fills keep the content of an earlier answer. The answer of step 1 held the
temperatures of an earlier run. An answer is therefore checked on its echo, which every answer overwrites, and the
answer of step 6 on its data type.

## Times

The answers of steps 1 and 8, the mode changes, came after about 65 milliseconds, and the others after about 2
milliseconds. A reading takes about 140 milliseconds. sensordeck waits at most 1 second for an answer.

## Software mode and the LEDs

On the device, the LEDs of the fans on the controller flicker once at each reading. Nothing drives them while the
controller is in software mode, and the return to hardware mode lights them again. The status command of the other tool
makes them flicker in the same way.

A reading without software mode was tried: the endpoint `0x21` was opened and read with no step 1 before it. Every answer
then had byte 2 `0x03` and stale data in the rest, so a reading without software mode is not possible with what is known.
Byte 2 of an answer is therefore a status, `0x00` in the answers of a sequence that step 1 opened and `0x03` for the
commands made without it.

To make the LEDs flicker less often, `sensordeck run` reads the controller at most once per period, of 1 minute unless
the config file says otherwise.

## Unknown

* The meaning of byte 0 of an answer, which was `0x00`. It is not checked
* The meaning of a byte 2 other than `0x00` and `0x03`, the two values seen
* The meaning of a probe status other than `00`, which was not seen
* Whether the temperature is signed
* Which probe sits at which port of the controller
* Whether the fan steps 2 to 4 are needed before the temperature steps. They are made, because that is the one
  sequence seen to work
* What the controller does with the fans in software mode when nobody sends it fan or lighting settings. It does not
  drive the LEDs
* What bytes 96 to 383 of an answer hold. They are not interpreted
* Whether the device needs a write of the full output report of 384 bytes. It was given 96 bytes after the report ID,
  as the other tool gave it, and answered them

## Assumptions

* An answer whose byte 2 is not `00` is not valid, byte 2 being the status that "Software mode and the LEDs" describes
* A probe whose status is not `00` has no reading, and is left out without renumbering the others
* The temperature is a signed 16-bit number. Both readings seen were positive
* The node of interface 0 is told apart by a `HID_PHYS` that ends in `/input0`, as the kernel's USB HID driver writes it
