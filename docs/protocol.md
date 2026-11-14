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

## Unknown

* How the brightness is set
* How device information or the firmware version is read back
* What the device sends on the IN endpoint without being asked
* Whether the touch layer is exposed
* What `0x01 0x15` at offsets 1 and 2 of the clock report mean
* What `0x64` at offset 10 of the clock report means

## Assumptions

Each of these is not yet confirmed on the device. The unit that adds the Linux transport is where they are checked.

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
