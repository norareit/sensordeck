---
kind: feature
status: in-progress
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
- 2026-11-17T11:11:42Z [human] approved the spec. The status is `approved`
- 2026-11-17T11:15:15Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  `feature-0027-number-options-and-bars`: the reader and the dump first, then the compiler, the preview, the end-to-end
  tests and the documents, in five commits, with every requirement traced to a task. The status stays `approved`
- 2026-11-17T11:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on `feature-0027-number-options-and-bars`. The status is `in-progress`
- 2026-11-17T12:30:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in five commits following the plan: the reader and the dump, the compiler, the preview, the end-to-end tests and the documents. On `cf5b3a5` the suite of `AGENTS.md` was run: `go test ./...` passed, `go vet ./...` reported nothing, `gofmt -l .` printed no file name, and `GOOS=windows GOARCH=amd64 go build ./...` and `go vet ./...` reported nothing. The tests were written after the code of their task and not before it, in the compiler and the preview tasks. The committed images `docs/minimal-dark.png` and `docs/minimal-dark-probes.png` were drawn again, and the probes image was looked at to see the numbers centred. Not checked: the device, which is the human's (D15). The status stays `in-progress`
- 2026-11-17T11:22:41Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a full review of `changes/0027-number-options-and-bars` at `e9f46beb91170c10df9dcaf25e022737fe997d6d` against its merge base with `origin/master`. Returned P1 and C1 in `.reviews/changes/0027-number-options-and-bars/review-round-1.md`. Independently ran all five suite commands, the committed change check, an uncached end-to-end run and the diff whitespace check; all passed. Viewed both regenerated built-in preview images. Could not verify the physical screen or a real Windows session, nor independently establish the internal test/code editing order. The status stays `in-progress`

## Amendments
- 2026-11-17T11:11:42Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  the human saw on the screen on 2026-11-17, what is a guess or the preview's own choice is marked as such, the bar
  values of the tests went to the tags in byte 2 of the bar records, and every requirement can be tested in the
  sandbox. `center` (D2) and `thousands: true` (D4) are accepted. Items 1 to 4 of "Beyond the request" are declined.
  The device check is the human's and the unit does not depend on it. The approval that follows is made under that
  delegation
