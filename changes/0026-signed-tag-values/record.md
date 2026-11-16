---
kind: bug
status: draft
timestamp: 2026-11-16T23:45:33Z
supersedes: 0004-linux-sensors-and-run
---

# Sending the value of a tag as a signed 16-bit number

## Log

- 2026-11-16T23:47:09Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session of the values that the human's screen showed on 2026-11-17 for 65535, 40000, 32768, 32767
  and 0 sent with `values`, which establish that the screen reads the value of a tag as a 16-bit two's complement
  number, against `docs/protocol.md`, the README, `sensors.ToValue`, the arguments of `values` and their tests, and the
  specs of the units 0002, 0003, 0004, 0007, 0011 and 0024. Decided: rounding as today and a clamp to -32768 to 32767;
  `values` takes -32768 to 65535, with an optional `-` before a decimal or hexadecimal number, so that every raw value
  that worked keeps its bytes; the documents say that the value is signed. The unit supersedes unit 0004, whose R4 it
  replaces, and changes requirements and decisions of the units 0002, 0003, 0007, 0011 and 0024, which the spec names.
  The check on the device of a negative custom metric is the human's, and the unit does not wait for it. The status is
  `draft`

## Amendments
