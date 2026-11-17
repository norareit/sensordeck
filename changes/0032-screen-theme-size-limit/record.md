---
kind: bug
status: draft
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

## Amendments
