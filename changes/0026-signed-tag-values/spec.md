# Sending the value of a tag as a signed 16-bit number

## Context

The value update of the screen carries one 16-bit value for each tag. `docs/protocol.md` calls that value unsigned,
and every unit since the protocol was written took it for a fact: unit 0004 made `run` clamp every reading to 0 to
65535 before it sends it, the `values` command of unit 0003 takes 0 to 65535, and unit 0024 sends custom metrics
through the same clamp, so that a negative custom metric shows 0. On the human's screen the value is not unsigned. The
screen reads it as a signed number, so a reading above 32767 shows as a negative number, and a negative reading, which
the screen could show, is sent as 0.

### What was observed

On the human's screen, on 2026-11-17, with test themes whose number widgets show the tags sent, and with
`sensordeck values` sending raw values:

| Value sent | Bytes of the value | The number widget showed |
|---|---|---|
| 65535 | `FF FF` | `-1` |
| 40000 | `9C 40` | `25536`, with its minus sign clipped by the box of the widget, so -25536 |
| 32768 | `80 00` | `-32768` |
| 32767 | `7F FF` | `32767` |
| 0 | `00 00` | `0` |

Every one of these is the value read as a 16-bit two's complement number, high byte first. The screen draws the minus
sign of a negative value, with the `-` glyph that a number record holds, and the box of the widget can clip it.

### Steps to reproduce

1. Upload a theme with a number widget for a tag, such as tag 1
2. Run `sensordeck values 1=65535`: the widget shows `-1` and not `65535`
3. Run `sensordeck run` with a custom metric read from a file that holds `40000`, mapped to a tag of the theme: the
   widget shows -25536 and not 40000
4. Change the file to hold `-5`: the widget shows `0` and not `-5`

### Root cause

The screen reads the value of a tag as a 16-bit two's complement number, while sensordeck takes it as unsigned. This is
established by the observations above, made on the device with known bytes. In the code, `sensors.ToValue` clamps a
reading to 0 to 65535, so a reading from 32768 to 65535 is sent as a bit pattern that the screen shows as that reading
minus 65536, and a negative reading is sent as 0. The `values` command takes 0 to 65535 and has no way to write a
negative value in decimal.

### The fix

A reading is rounded as today, to the nearest whole number with halves away from zero, then clamped to -32768 to 32767,
and sent as a 16-bit two's complement number, high byte first. The `values` command takes a value from -32768 to 65535:
a negative value is sent as its two's complement, and a value from 0 to 65535 as that 16-bit pattern, as today. The
documents say that the value is signed.

The check on the device that a negative custom metric shows with its minus sign is the human's, and the unit does not
wait for it.

### The requirements this unit replaces

The main one is R4 of unit `0004-linux-sensors-and-run`, the rounding and the clamp to 0 to 65535, with its examples in
R31 and the remark on GPU memory in its D1. Besides it:

- Unit `0002-device-protocol`: the table of the value update, which says 16 bits unsigned, and the word unsigned in R8.
  The bytes that R8 gives for a value from 0 to 32767 do not change
- Unit `0003-linux-transport-and-command-line`: in R20, the range of the value, the rule that a number has no sign as
  far as the value is concerned, and the error on the range. R32 still holds, `0x1F=0XFFFF` included
- Unit `0007-ram-share-and-network-rates`: D6, where a rate above 65535 KiB/s is sent as 65535. It is now a rate above
  32767 KiB/s, sent as 32767
- Unit `0011-theme-compiler`: the reason in D5 for the limit of 5 digits, a value of 16 bits unsigned, and the non-goal
  on negative values. The limit itself does not change, see "Beyond the request"
- Unit `0024-custom-metrics`: R12, where a value below 0.5 is sent as 0 and one above 65535 as 65535, the note of R31,
  D13, and the non-goal on showing negative numbers

## Requirements

### The value of a reading

- **R1** `sensors.ToValue` gives the value of a value update for a reading by rounding it to the nearest whole number,
  halves away from zero, and then clamping it to -32768 to 32767. So 51.5 gives 52, 51.49 gives 51, 0.5 gives 1,
  -0.4 gives 0, -0.5 gives -1, -5 gives -5, -51.5 gives -52, 32767.4 gives 32767, 32767.5 gives 32767, 40000 gives
  32767, 70000 gives 32767, -32768.4 gives -32768, -32768.5 gives -32768, -40000 gives -32768, positive infinity gives
  32767 and negative infinity gives -32768
- **R2** The value of a pair of a value update is written as a 16-bit two's complement number, high byte first, in the
  layout of `docs/protocol.md`, which does not otherwise change. So -1 is `FF FF`, -5 is `FF FB`, -32768 is `80 00`,
  32767 is `7F FF`, 0 is `00 00`, and a reading of -0.4 is `00 00`
- **R3** `run` sends every value through R1 and R2, those of the built-in metrics and of the custom metrics alike. A
  custom metric whose value is -5 is sent as `FF FB`, and one whose value is 40000 as `7F FF`

### The `values` command

- **R4** The value of an argument `<tag>=<value>` of `sensordeck values` is a number as R20 of unit 0003 defines it,
  decimal or hexadecimal, optionally with one `-` directly before it. Nothing else is a value: no `+`, no second `-`,
  no space after the `-`, and a `-` alone is not a number. `-0` is 0. The tag keeps the rules of R20 of unit 0003 and
  takes no sign
- **R5** A value from -32768 to 65535 is accepted. A negative value is sent as its 16-bit two's complement, and a value
  from 0 to 65535 as that 16-bit pattern, so that `-1` and `65535` give `FF FF`, and `-32768`, `-0x8000`, `32768` and
  `0x8000` all give `80 00`. A value outside that range is refused, before the transport is opened as R18 of unit 0003
  says, with `invalid argument "<argument>": the value is outside -32768 to 65535`. A number too large for 64 bits,
  with or without the `-`, is outside that range. A value that is not a number gives the error
  `invalid argument "<argument>": want <tag>=<value>`, as today
- **R6** What `values` writes to standard output does not change. So `sensordeck values 1=-1 2=32767` writes 65 bytes to
  the node, `00 02 02 01 FF FF 02 7F FF` followed by 56 bytes of `0x00`, and prints `values sent: 2, reports written: 1`

### The documents

- **R7** `docs/protocol.md` says in the table of the value update that the value is 16 bits signed, two's complement,
  from -32768 to 32767, high byte first, and gives the example of tag 1 with the value -1, `02 01 01 FF FF`. It states
  the observations of 2026-11-17 of the table of this spec: the five values sent and what the screen showed, that the
  screen draws the minus sign with the `-` glyph of the number record, and that the box of a widget can clip it. The
  example of the pairs `0x1234` and `0x0001` stays
- **R8** Where the README says that values are sent as whole numbers, rounded, it says that they are clamped to -32768
  to 32767, that a negative value shows with its minus sign, and that a reading above 32767, such as a network rate of
  40000 KiB/s, shows 32767 while `sensordeck sensors` shows the reading
- **R9** The note of the README on the values of custom metrics says that the screen shows whole numbers from -32768 to
  32767, as for every metric, so that `run` rounds the value, a negative value shows with its minus sign, and tenths are
  shown by a `scale` ten times larger. It no longer says that a negative value shows 0
- **R10** The README says what `sensordeck values` takes as a value: -32768 to 32767 for the number the screen shows,
  and 32768 to 65535 as the raw 16-bit pattern, which the screen shows as that value minus 65536. No other text of the
  README or of `docs/` says that a value is unsigned or that its range is 0 to 65535. The tables of metrics do not
  mention the range and do not change

### The tests

- **R11** Unit tests cover R1 with every example named there, and R2 with the encoding of -1, -5, -32768, 32767 and 0
  in a value update
- **R12** Unit tests cover R4 and R5: `1=-1`, `1=-32768`, `1=-0x8000`, `1=32768`, `1=65535`, `1=0xFFFF` and `1=-0`
  accepted with the patterns that R5 gives; `1=-32769`, `1=-0x8001`, `1=65536`, and a negative number too large for
  64 bits refused with the error on the range; `1=-`, `1=--1`, `1=+1`, `1=- 1` and `1=-0x` refused as not a number.
  The existing tests of the error on the range give its new text
- **R13** The end-to-end tests cover the example of R6, and one argument of `values` outside the range with no screen
  connected, which fails with the error of R5 and exit status 1. They cover `run` with a custom metric read from a file
  that holds `-5` and one that holds `40000`, mapped to tags of the theme, arriving as `FF FB` and `7F FF`

## Decisions

- **D1 Clamp to the signed range, as the clamp to the unsigned range was.** A reading above 32767 that is sent as its
  pattern shows a negative number, which is false, as 40000 showed -25536; clamping keeps the value at the largest
  that the screen shows, which reads as full, the reason D2 of unit 0004 gave for clamping rather than failing.
  Rejected: clamping to 0 to 32767, which fixes the readings above 32767 but keeps every negative reading at 0, while
  the screen shows a minus sign and unit 0024 made custom metrics such as an outside temperature, which go below 0
- **D2 `values` takes -32768 to 65535.** It is a low-level command, beside `raw`, made to put known bytes on the wire,
  and a value from 0 to 65535 is the 16-bit pattern that goes there. The human used it with raw values from 0 to 65535
  for the observations of this unit, R32 of unit 0003 tests `0x1F=0XFFFF`, and hexadecimal is how a pattern is written,
  so the range of today stays and every argument that worked keeps its bytes. The negative values are added so that the
  number the screen shows can be written as it is. Rejected: -32768 to 32767 only, which gives each value one spelling,
  but refuses `0xFFFF` and `0x8000` and every raw value of the experiments, and asks for `-0x1` where the pattern is
  meant. Rejected: clamping in `values`, because a low-level command sends what it was given or refuses it. The price
  is that two spellings give the same bytes, `-1` and `65535`, which the README says (R10)
- **D3 The sign is allowed before both forms.** One grammar, a number of R20 of unit 0003 with an optional `-`, is
  simpler to state and to test than a sign for decimal alone, and `-0x8000` reads as -32768. The tag takes no sign,
  because a tag is never negative. An argument that starts with `-`, such as `-1=5`, is taken for a flag by the parser of
  the command line and refused, as it is today
- **D4 The limits of the metrics move with the range.** A network rate is clamped at 32767 KiB/s, about 32 MiB/s or
  268 Mbit/s, where it was at 65535 KiB/s, and `gpu.mem.used` and `gpu.mem.total` at 32767 MiB, where D1 of unit 0004
  put the limit at 64 GiB. The 32607 MiB of the RTX 5090 of the host still fit. The units of the metrics stay, because
  changing one breaks every theme and every key `tags` that shows it; a pair of rates in MiB/s is item 2 of "Beyond
  the request" of unit 0007 and stays there
- **D5 The rounding does not change.** Halves go away from zero for negative values too, so -0.5 gives -1 and -51.5
  gives -52, which is what the rule of unit 0004 says and what `math.Round` does. A reading above -0.5 and below 0
  gives 0 and is sent as `00 00`, and never as a negative zero

## Non-goals

- The box of a number widget and the limit of 5 for `digits`: a negative value needs room for its minus sign, and
  the compiler does not change; see "Beyond the request"
- The built-in themes, whose boxes stay as they are
- Decimals on the screen, and the units of the metrics, which stay as D4 says
- A line on standard error or in `--trace` when a value is clamped
- The check on the device of a negative custom metric, which is the human's
- Windows, where the value update is the same and nothing of this unit differs

## Beyond the request

1. Say in `docs/theme-folder.md` that the minus sign of a negative value takes a place in the box of a number widget,
   so that a widget that shows negative values needs a `digits` one larger than its longest number. It costs a
   sentence. Without it, a theme author sees the minus sign clipped, as with 40000, and can give a larger `digits`
2. Raise the limit of `digits` from 5 to 6, so that -32768 fits in a box with its minus sign. It costs a change of the
   compiler, its error text, a test and `docs/theme-folder.md`, and replaces a requirement of unit 0011. Without it a
   negative value of five digits has its minus sign clipped, which only a custom metric below -9999 reaches
3. A line in `--trace` of `run` when a reading is clamped, naming the metric and the reading. It costs a line and a
   test. Without it the screen shows 32767 or -32768 while `sensordeck sensors` shows the reading, as it showed 65535
   before, which the user sees and can read with `sensors`
