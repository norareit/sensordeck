---
kind: bug
status: implemented
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
- 2026-11-16T23:14:58Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on
  `bugfix-0025-commander-led-port-configuration`, from the approved spec and the plan. The status is `in-progress`
- 2026-11-16T23:18:45Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in the six tasks of
  the plan, with the tests of each written before its code. Tasks 3 and 4 are one commit, since the new three results
  of `Lighting.Start` break the build of `internal/cli` until the keeper takes them. The fake gained `UnavailableStatus`
  and the stored and lighting configurations, the protocol `wantedConfiguration`, `portTypeProblem` and
  `configurationCommand`, the session `repairConfiguration` and `writeConfiguration`, and the keeper `setupNotice`
  and the two lines of R8. The suites were run on the tree of the commit that describes the documents:
  `go test ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=windows GOARCH=amd64 go vet ./...` all reported nothing but passing tests. Not checked: the device, and Windows,
  which are the human's. No requirement was left out, and the plan needed no addition. The status stays `in-progress`
- 2026-11-16T23:20:56Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a full review of unit
  `0025-commander-led-port-configuration` at `30599af75566e61cacb34ab45fcdfd0e117b4f6e` against the merge base
  with `origin/master`. Returned no findings on spec conformance, plan fidelity or code quality. Saved the review in
  `.reviews/0025-commander-led-port-configuration/review-round-1.md`. Independently ran the five required suite
  commands, an uncached `go test -count=1 ./...`, `npm run check-changes` and the diff whitespace check; all passed.
  Could not verify physical-device behavior, Windows runtime behavior or tests-first ordering within commits.

## Amendments
- 2026-11-16T23:08:20Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  observed on the human's controller on 2026-11-16 and 2026-11-17, among them that writing OpenLinkHub's configuration
  to 0x1E made the colour of a test script and then of `run` show again, it writes only the configuration observed to
  work and only when the stored one differs, and every requirement can be tested in the sandbox with the fake
  controller. Items 1 and 2 of "Beyond the request" are declined: `08 15 01` was part of every setup that lit the
  LEDs, and a user who wants to keep iCUE's configuration leaves out `commander_color`, without which sensordeck does
  not touch it. The device check is the human's and the unit does not depend on it. The approval that follows is made
  under that delegation
- 2026-11-16T23:22:48Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the findings: round 1
  returned none, so no blocking finding is open. The timestamps of two entries of the implementer, which did not match
  their commits, were corrected to the times of the commits that added them. Every task of `plan.md` is ticked, and
  the five commands of the suite passed on the branch. The repair on the device, after iCUE has rewritten 0x1E, is the
  human's check and follows the merge. The status is `implemented`
