---
kind: bug
status: draft
timestamp: 2026-11-16T23:06:16Z
supersedes: 0009-commander-led-colour
---

# Repairing the LED port configuration of the Commander Core XT that iCUE rewrote

## Log

- 2026-11-16T23:06:16Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session of LEDs that stay dark in a session after iCUE has run, with the facts it observed on the
  human's controller on 2026-11-16, against the trace of OpenLinkHub of 2026-11-15, read for the traffic alone, which
  gave the close, open, write and close of `0x1E` and the order of `08 15 01`, the test script of the orchestrating
  session, the README, `docs/commander-core-xt.md`, the spec of unit 0009, and the protocol and session of
  `internal/commandercore`, the keeper of `run` and `internal/fakecommander` in the tree, which answers no read of
  `0x1E` with content and no write on channel 1. Decided: the read of `0x1E` at each setup and a write only when its
  bytes differ from the wanted configuration, to spare the controller's memory; the type `03` written only when every
  port with LEDs has 4 of them, and `0x1E` left as stored with a line otherwise; `08 15 01` only after a write; a line
  naming iCUE after each rewrite; a fake that stays dark with iCUE's configuration. The unit supersedes unit 0009,
  whose R6 and R10 it changes. No code of OpenLinkHub, liquidctl, OpenRGB or iCUE was read. The status is `draft`

## Amendments
