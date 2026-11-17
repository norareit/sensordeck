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
- 2026-11-17T11:11:13Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the word of the
  orchestrating session: the direction of the fill of a bar was observed, on a photo of the screen at 50 and at 100,
  from the left edge. The context, R18 and D12 now record the fill from the left as observed, and keep the exact pixels
  of its edge and a box taller than wide as unknown. The session confirmed that the values of the tests went to the
  tags in byte 2 of the bar records, so R8 stands. The status stays `draft`

## Amendments
- 2026-11-17T11:11:42Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  the human saw on the screen on 2026-11-17, what is a guess or the preview's own choice is marked as such, the bar
  values of the tests went to the tags in byte 2 of the bar records, and every requirement can be tested in the
  sandbox. `center` (D2) and `thousands: true` (D4) are accepted. Items 1 to 4 of "Beyond the request" are declined.
  The device check is the human's and the unit does not depend on it. The approval that follows is made under that
  delegation
