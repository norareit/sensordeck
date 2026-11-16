# Wire protocol of the SmartMonitor screen

This document describes how the computer talks to the screen that version 1 of sensordeck targets. The protocol was
learned from the project the README credits. This document is written in the project's own words and holds no code or
text of that project.

## Identity

* USB ID `0483:0065`
* Manufacturer string `llhmi.com`, product string `llTechCo.,Ltd`
* Sold as "3.5 inch PC CPU Data Monitor" and as SmartMonitor
* The screen is 480 by 320 pixels

## Nature of the device

The device is a theme player and not a framebuffer. A theme holds the layout, the images, the fonts and widgets that
are bound to numeric tags. It is uploaded once to the 4 MB flash of the device. Afterwards the computer sends only
small value updates and the clock. A theme upload takes about 10 seconds and reboots the device.

## Transport

USB HID at full speed, one interface, two interrupt endpoints: `0x81` IN and `0x02` OUT. The computer sends HID output
reports of 64 bytes, and the device answers with input reports of 64 bytes. On Linux the device is reached through
`/dev/hidraw*`, where a report is written as the report ID byte `0x00` followed by the 64 bytes. The report ID byte
belongs to the transport and is no part of the 64 bytes this document speaks of. A "report" below is always the 64
bytes alone.

An input report is read from the node as 64 bytes without a report ID byte, because the device uses no numbered
reports. The screen is found through the `HID_ID` line of the `uevent` file of its entry under `/sys/class/hidraw`,
which holds the bus, the vendor ID and the product ID. hidraw needs the kernel's `usbhid` driver bound to the device.

On Windows the transport writes and reads the same reports through the HID API. A report is written as the report ID
byte `0x00` followed by the 64 bytes, and an input report is read as 65 bytes whose first byte, the report ID, is
dropped. Both lengths, 65, are those that the caps of the collection give, and a collection whose lengths are others is
not taken for the screen. The screen is found by its vendor ID and product ID among the HID collections that the system
lists.

## Value update

One report sets up to 20 values:

| Offset | Size | Content |
|---|---|---|
| 0 | 1 | `0x02` |
| 1 | 1 | the number of pairs in this report, 1 to 20 |
| 2 | 3 for each pair | the tag, one byte, then the value, 16 bits unsigned, high byte first |
| after the pairs | the rest | `0x00` up to 64 bytes |

For example, the pairs (tag `0x01`, value `0x1234`) and (tag `0xFF`, value `0x0001`) give `02 02 01 12 34 FF 00 01`
followed by 56 bytes of `0x00`. More than 20 pairs are sent as several reports.

## Clock

One report sets the clock:

| Offset | Content |
|---|---|
| 0 | `0x03` |
| 1 | `0x01` |
| 2 | `0x15` |
| 3 | the year minus 2000 |
| 4 | the month, 1 to 12 |
| 5 | the day of the month |
| 6 | the hour, 0 to 23 |
| 7 | the minute |
| 8 | the second |
| 9 | the ISO weekday, 1 for Monday to 7 for Sunday |
| 10 | `0x64` |
| 11 to 63 | `0x00` |

The year byte holds 2000 to 2255. For 2026-11-14 15:04:05, a Saturday, the report starts
`03 01 15 1A 0B 0E 0F 04 05 06 64`.

The screen shows the time of the last clock report it received and does not advance it, as seen on the device on
2026-11-15. A program sends the clock once a second for the seconds on the screen to change.

## Theme upload

The steps, in order:

1. The computer sends the six bytes `0x01 0x72 0x65 0x73 0x65 0x74`, which is `0x01` followed by the ASCII text
   `reset`. The device reboots and enumerates again, which takes about 2.5 seconds. The computer opens it again
2. The computer sends the six ASCII bytes `ymodem` and waits for the answer ACK, `0x06`
3. The computer sends a YMODEM header block: start byte SOH, block number 0, and 128 data bytes that hold the file name
   `img.dat`, a `0x00`, the size of the theme in decimal ASCII, and a `0x00`. Then it sends the theme in blocks of
   1024 data bytes with the start byte STX, the last one filled up with `0x1A`. Every block carries a CRC-16/XMODEM,
   high byte first. The device answers every block with ACK, `0x06`, or with NAK, `0x15`, on which the block is sent
   again
4. The computer sends EOT, `0x04`, and waits for ACK. Then it sends an empty block with number 0, which ends the
   session
5. The device reboots again. Opening it again needs several attempts, and an I/O error while it enumerates is normal

## YMODEM block

This is the public layout of the YMODEM protocol:

| Offset | Size | Content |
|---|---|---|
| 0 | 1 | the start byte: SOH, `0x01`, for 128 data bytes, or STX, `0x02`, for 1024 data bytes |
| 1 | 1 | the block number |
| 2 | 1 | the one's complement of the block number, which is 255 minus the block number |
| 3 | 128 or 1024 | the data |
| after the data | 1 | the high byte of the CRC of the data |
| last | 1 | the low byte of the CRC of the data |

A block is therefore 133 or 1029 bytes long. The CRC is computed over the data bytes alone.

## CRC-16/XMODEM

Polynomial `0x1021`, initial value `0x0000`, no reflection of input or output, no final XOR. The CRC of the nine ASCII
bytes `123456789` is `0x31C3`.

## Hangs of the firmware

The firmware of the screen can get into a state in which every write of an output report to the interrupt OUT endpoint
times out after the kernel's limit of 5 seconds, and the screen shows its theme with the default time 12:00:00 and date
2022-01-01. Two hangs that look the same were seen, and the facts below were observed on the device on 2026-11-16, on a
Linux machine, unless said otherwise.

### The reset hang

1. **The firmware hangs on a USB reset while it runs.** A `USBDEVFS_RESET` ioctl on the usbfs node of a screen that
   worked made every later write time out at once. Disabling and enabling its port of the root hub, by writing 1 and
   then 0 to `usb1-port4/disable`, did the same
2. **The hung state.** Writes of output reports to the interrupt OUT endpoint time out. `HIDIOCSOUTPUT`, the output
   report by the control endpoint, times out too. `HIDIOCSFEATURE` is accepted and does nothing, because the report
   descriptor declares one input and one output report of 64 bytes and no feature report. No input report arrives. A
   further USB reset does not help. The disable switch of the port does not cut its power, and the port has no ACPI
   power resources. The report descriptor is
   `06 00 ff 09 01 a1 01 09 01 15 00 26 ff 00 95 40 75 08 81 02 09 01 15 00 26 ff 00 95 40 75 08 91 02 c0`
3. **A USB runtime suspend and resume cleared it once.** With no program holding the hidraw node open, `power/control`
   of the USB device was set to `auto`, with `power/autosuspend_delay_ms` at 2000. `power/runtime_status` read
   `suspended` within 5 seconds. Setting `control` back to `on` resumed the device, and the clock and the values
   worked again at once. This was observed once, on a screen hung by the reset of fact 1

The USB device has `bmAttributes` `0x80`, bus powered without remote wakeup, and no `power/wakeup` file.

### The boot hang

After every cold boot the screen shows its theme with 12:00:00 and 2022-01-01, and every write times out. A suspend and
resume does not clear it.

* **Its trigger is the firmware of the PC.** The screen, plugged in while the GRUB menu showed, when the firmware of
  the PC (UEFI) drives USB, hung in the same way once Linux ran. So the hang follows the setup of the screen by UEFI
  before Linux, and not a reset by Linux. Nothing in Linux can prevent it
* **The inference that Linux resets the screen at boot was wrong.** An earlier version of this document took the hang at
  boot for the reset hang and inferred that Linux resets the device once more at boot. The plugging in at the GRUB
  menu shows that no reset by Linux is needed
* **What does not clear it.** The control endpoint still answers. These requests were accepted and did not help:
  CLEAR_FEATURE(ENDPOINT_HALT) on both endpoints, a halt and an unhalt of the OUT endpoint, the HID requests SET_IDLE
  and SET_PROTOCOL, SET_INTERFACE, SET_CONFIGURATION(1) repeated, and the command `01 reset` sent as SET_REPORT on the
  control endpoint. No endpoint reports itself halted. A suspend and resume, woken by opening the node or by setting
  `power/control` to `on`, a USB reset, and a USB reset followed by a suspend did not help either. A replug clears it

### The restart

Asking for string descriptors the screen does not have makes its firmware restart, which is what a replug does. The
screen has the string descriptors 0 to 3. The request is GET_DESCRIPTOR, sent for one index at a time:

| Field | Value |
|---|---|
| bmRequestType | `0x80` |
| bRequest | 6 |
| wValue | `0x0300` with the index in its low byte |
| wIndex | `0x0409` |
| wLength | 255 |

Two observations, both on 2026-11-16:

* On a screen that worked, the indexes 4 to 32 were answered with a stall, the request for 33 failed with "no such
  device" (`ENODEV`), the kernel logged a USB disconnect, and the screen enumerated again 1.3 seconds later and worked
* On a screen hung at boot, a scan of the indexes 0 to 255 made it disconnect and enumerate again, and writes were
  accepted afterwards. Other requests had been sent to it before, so this observation is not clean

A screen that restarts under Linux is set up by Linux alone, as a replug does, and a replug was seen to clear both
hangs.

Not established: whether the restart is triggered by index 32, by index 33, or by the number of stalled requests in a
row; a clean run on a screen hung at boot, with nothing sent to it first; whether the restart clears the reset hang,
which is likely, since it is a restart of the firmware as a replug is, but not observed; and whether either hang
happens on Windows.

On Windows `HidD_GetIndexedString` sends the same request through the HID handle. sensordeck does not use it.

### How sensordeck restarts the screen

On Linux `run` restarts a screen whose write timed out at the first round of the hang, and then at most once a minute.
The commands that write make one restart and end with an error. A restart is made through the usbfs node of the screen's
USB device, `<dev>/bus/usb/<bus>/<device>` with the numbers of `busnum` and `devnum` written with three digits, which
the third rule of `udev/70-sensordeck.rules` gives to the user of the session. usbhid stays bound. sensordeck closes the
hidraw node, sends the request above through the ioctl `USBDEVFS_CONTROL` with a timeout of 1 second for the indexes
4, 5, 6 and so on, one at a time, until the screen has left the bus, and stops at index 255 or after 10 seconds. The
screen has left the bus when a request fails with `ENODEV`, or when the USB device is gone from sysfs or has another
`devnum`. It then opens the screen again, polling every 100 milliseconds for up to 10 seconds, and `run` writes the index
the screen left at. sensordeck sends no USB reset, unbinds no driver, claims no interface and writes no file in sysfs.

## Unknown

* Whether a suspend and resume of a screen that works is harmless
* What the screen shows while it is suspended
* Whether either hang happens on Windows
* How the brightness is set
* How device information or the firmware version is read back
* What the device sends on the IN endpoint without being asked
* Whether the touch layer is exposed
* What `0x01 0x15` at offsets 1 and 2 of the clock report mean
* What `0x64` at offset 10 of the clock report means

## Assumptions

Each of these is not yet confirmed on the device. They are checked on the device by the human with the commands of the
unit that adds the Linux transport, and stay unconfirmed until then.

* **Carrying a block in reports.** A YMODEM block is carried as consecutive reports of 64 bytes, the last filled up
  with `0x00`: 3 reports for 133 bytes and 17 for 1029. The alternatives are a length byte or a command byte in front
  of each piece
* **Fill of commands.** The reset command, the ymodem command and the end of transmission are filled up with `0x00` to
  64 bytes
* **Fill of the header block.** The 128 data bytes of the header block are filled up with `0x00` after the size
* **The answer.** The answer of the device is the first byte of an input report
* **No answer to the closing block.** The computer reads no answer to the closing block, because the device reboots
  at that point
* **The numbers of the upload.** A pause of 1 second before the first attempt to reopen the device, 500 milliseconds
  between attempts, 20 attempts, 10 seconds for an answer, and 10 writes of one item
* **A short input report.** An input report shorter than 64 bytes is filled up with `0x00`
