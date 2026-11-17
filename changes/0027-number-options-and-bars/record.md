---
kind: feature
status: implemented
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
- 2026-11-17T11:16:03Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on `feature-0027-number-options-and-bars`. The status is `in-progress`
- 2026-11-17T11:20:24Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in five commits following the plan: the reader and the dump, the compiler, the preview, the end-to-end tests and the documents. On `cf5b3a5` the suite of `AGENTS.md` was run: `go test ./...` passed, `go vet ./...` reported nothing, `gofmt -l .` printed no file name, and `GOOS=windows GOARCH=amd64 go build ./...` and `go vet ./...` reported nothing. The tests were written after the code of their task and not before it, in the compiler and the preview tasks. The committed images `docs/minimal-dark.png` and `docs/minimal-dark-probes.png` were drawn again, and the probes image was looked at to see the numbers centred. Not checked: the device, which is the human's (D15). The status stays `in-progress`
- 2026-11-17T11:22:41Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a full review of `changes/0027-number-options-and-bars` at `e9f46beb91170c10df9dcaf25e022737fe997d6d` against its merge base with `origin/master`. Returned P1 and C1 in `.reviews/changes/0027-number-options-and-bars/review-round-1.md`. Independently ran all five suite commands, the committed change check, an uncached end-to-end run and the diff whitespace check; all passed. Viewed both regenerated built-in preview images. Could not verify the physical screen or a real Windows session, nor independently establish the internal test/code editing order. The status stays `in-progress`
- 2026-11-17T11:24:11Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, took the findings of `review-round-1.md`, accepted by the definition of the orchestrator and not as a decision of the human. C1 (nonblocking) is accepted and fixed: the folder document now says that no number or bar gets tag 0, and that the preview draws from the top left of the box except that a number is placed in its box by its `align`. P1 (nonblocking) is accepted as a statement of fact and needs no change: the tests of the compiler and preview tasks were written after their code, as the earlier entry says, and the order of work cannot be repaired afterwards. Not re-run after this documentation edit: the suites, since only `docs/theme-folder.md` and this record changed. The status stays `in-progress`
- 2026-11-17T11:25:26Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a delta review of `changes/0027-number-options-and-bars` at `20aa995` against `e9f46beb91170c10df9dcaf25e022737fe997d6d`, the commit named by the first review. Checked the changes concerning P1 and C1 and returned no new findings in `.reviews/changes/0027-number-options-and-bars/review-round-2.md`. Independently ran all five suite commands, an uncached end-to-end run, the committed change check and the delta whitespace check; all passed. Could not verify the physical screen, a real Windows session or the internal test/code editing order. The status stays `in-progress`

## Amendments
- 2026-11-17T11:11:42Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  the human saw on the screen on 2026-11-17, what is a guess or the preview's own choice is marked as such, the bar
  values of the tests went to the tags in byte 2 of the bar records, and every requirement can be tested in the
  sandbox. `center` (D2) and `thousands: true` (D4) are accepted. Items 1 to 4 of "Beyond the request" are declined.
  The device check is the human's and the unit does not depend on it. The approval that follows is made under that
  delegation
- 2026-11-17T11:27:32Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the findings: C1 of
  round 1 was corrected, P1 was recorded by the implementer as a departure that cannot be undone, and round 2 returned
  none, so no blocking finding is open. The timestamps of three entries of the implementer, which did not match their
  commits, were corrected to the times of the commits that added them. Every task of `plan.md` is ticked, and the
  five commands of the suite passed on the branch. On the host, the preview of a trial theme showed a number aligned
  left, one aligned right, one at opacity 0.4, one in the thousands mode as `1.2`, and a bar filled from the left. The
  check on the device is the human's and follows the merge. The status is `implemented`
