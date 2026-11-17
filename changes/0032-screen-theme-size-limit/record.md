---
kind: bug
status: in-progress
timestamp: 2026-11-17T19:10:45Z
supersedes: 0011-theme-compiler
---

# The screen draws no widget whose data ends past 503 808 bytes

## Log

- 2026-11-17T19:10:45Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session, which carried the observation on the device of 2026-11-17 and the fix the human chose,
  against `process.md`, the standards, `AGENTS.md`, the README, `docs/`, units 0004, 0011, 0023, 0027, 0028 and 0030,
  and the compiler, the reader and the preview in `internal/`. On the host, the installed `sensordeck` compiled the
  two built-in themes and `theme dump` gave the bytes of each resource, which are the table of the context: 508 370
  bytes for `minimal-dark-probes` with 15 542 of texts, and 431 788 for `minimal-dark` with 13 072; no text of either
  overlaps an earlier widget, so all of them are drawn into the background and the two themes compile to 492 828 and
  418 716 bytes. The `theme.yaml` of `minimal-dark-probes` before unit 0030 compiled to 436 694 bytes. Nothing was
  uploaded to the screen. The unit supersedes unit 0011, for the reason D11 gives. The status is `draft`
- 2026-11-17T19:11:18Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its cause and
  limit are the measurements the orchestrating session and the human made on the screen on 2026-11-17, including the
  prediction that came true; the fix is the one the human chose; the limit is marked as a value inferred within the
  measured range; and the layering of any theme folder is kept by D2. The size of the theme before task 0030 that the
  orchestrating session gave the human, about 496 KB, was wrong; the spec's measured 436 694 bytes stands. Items 1 to 5
  of "Beyond the request" are declined. As D10 says, the unit is set to `implemented` only after the human has seen
  the date and the time on the screen with the five alarms. The approval that follows is made under that delegation
- 2026-11-17T19:11:18Z [human] approved the spec. The status is `approved`
- 2026-11-17T19:14:57Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the head of
  the default branch: the texts are decided in the one pass of the compiler and drawn into the background through
  `themeimage.Preview`, the limits are checked after the list, the records before the size, and the built-in
  previews, the end-to-end tests and the documents follow. It checked that rounding any 8-bit channel to RGB565 and
  expanding it back moves it by at most 4 in red and blue and 2 in green, the bounds of R6. Every requirement has a
  task. The status stays `approved`
- 2026-11-17T19:30:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on the branch `bugfix-0032-screen-theme-size-limit`. The status is `in-progress`
- 2026-11-17T20:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit. The texts are decided in `buildRecords` and drawn into the background by `drawTexts` through `themeimage.Preview`, the checks of 62 records and of 503 808 bytes follow the list, and the early count of widgets is removed. The tests were written together with the code and not each before it, but mutating `overlaps` to always say no failed 8 tests of the package. The first two tasks of the plan were committed together, since the limits were already in the tree: the commit points of those two are one commit. Before the images were written again, the preview of each built-in theme was compared with its committed image: `minimal-dark` differed at 3 590 pixels and `minimal-dark-probes` at 4 234, the largest differences being 4 in red, 2 in green and 4 in blue for both, the bounds of R6. The built-in themes compile to 418 716 bytes with 13 records and to 492 828 bytes with 19, with no text record, as the test of the built-in themes checks. On the final tree `go test ./...`, `go vet ./...`, `gofmt -l .`, and the build and the vet with `GOOS=windows GOARCH=amd64` all passed with no report. R14, the date and the time on the screen of the human, is not checked and is the human's, and the status stays `in-progress` until the human has checked it

## Amendments
