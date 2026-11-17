---
kind: feature
status: draft
timestamp: 2026-11-17T11:09:01Z
supersedes: 0010-theme-file-reader
---

# The options of a number widget, the bar widget and a preview that clips

## Log

- 2026-11-17T11:09:01Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session of what the human's screen showed on 2026-11-17 for bytes 11, 14 and 15 of a number, the
  clipping of glyphs to the box, a number with no value, and the fields of a bar, against the specs of the units 0010,
  0011, 0012 and 0026, the format and folder documents, the README and the code of `internal/themefile`,
  `internal/themeimage`, `internal/themecompile` and `internal/themedump`. Decided: `align` with `center` as its
  default, `opacity` as a fraction from 0 to 1 and `thousands` as a boolean on a number; a widget `bar` with `metric`,
  a box, `color` for the fill and the frame and `background` for the empty part, whose value is its metric's as it is,
  from 0 to 100, a metric of another range going through a custom metric with a `scale`; a preview that aligns
  numbers as byte 11 says, clips them to their box, applies their opacity, draws `1.2` for the thousands mode and draws
  bars filled to 42; the JSON names of the dump renamed for what is now known. The unit supersedes unit 0010, whose
  R3, R12, R13, R20 and R21 it changes in part, and changes requirements of the units 0011 and 0012, which the spec
  names. The device check is the human's, and the unit does not wait for it. The status is `draft`

## Amendments
