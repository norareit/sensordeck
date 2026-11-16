---
kind: bug
status: in-progress
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
- 2026-11-16T23:08:20Z [human] approved the spec. The status is `approved`
- 2026-11-16T23:12:16Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  `bugfix-0025-commander-led-port-configuration`. The fix needs a plan, since it takes the fake controller, the
  protocol, the setup, the lines of `run`, end-to-end tests and two documents. In six commits: the fake learns `0x1E`,
  `08 15 01`, a lighting configuration and a failure of status `03`; the protocol builds the wanted configuration and
  the configuration write; the setup reads `0x1E` and rewrites it when it differs, and `Lighting.Start` gives what it
  did with the configuration beside the session and the error; the keeper writes the lines of R8 from a notice of its
  own; the end-to-end tests; the documents. Every requirement has a task, and every task traces to a requirement. The
  status stays `approved`
- 2026-11-16T23:30:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on
  `bugfix-0025-commander-led-port-configuration`, from the approved spec and the plan. The status is `in-progress`

## Amendments
- 2026-11-16T23:08:20Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  observed on the human's controller on 2026-11-16 and 2026-11-17, among them that writing OpenLinkHub's configuration
  to 0x1E made the colour of a test script and then of `run` show again, it writes only the configuration observed to
  work and only when the stored one differs, and every requirement can be tested in the sandbox with the fake
  controller. Items 1 and 2 of "Beyond the request" are declined: `08 15 01` was part of every setup that lit the
  LEDs, and a user who wants to keep iCUE's configuration leaves out `commander_color`, without which sensordeck does
  not touch it. The device check is the human's and the unit does not depend on it. The approval that follows is made
  under that delegation
