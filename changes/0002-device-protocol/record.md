---
kind: feature
status: implemented
timestamp: 2026-11-14T14:51:19Z
---

# Device protocol

## Log

- 2026-11-14T14:51:19Z [Claude Code, anthropic/claude-fable-5-2] as the specifier, wrote the spec. The status is `draft`
- 2026-11-14T14:53:55Z [Claude Code, anthropic/claude-fable-5-2] read the spec under the delegation in `AGENTS.md`: every known
  fact of the protocol is in it, the byte examples and CRC values of R10, R13, R14 and R16 were computed again and
  hold, and every requirement can be tested without the device. Nothing was changed. None of the six items of
  "Beyond the request" is taken: item 1 is decided once the device has shown what it answers, and items 2 to 6 have
  no user before there is a command line. The approval that follows is made under that delegation
- 2026-11-14T14:53:55Z [human] approved the spec. The status is `approved`
- 2026-11-14T15:01:56Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  branch. Every requirement R1 to R29 has a task, and the byte examples and CRC values of R13, R14 and R16 were
  computed again and hold. The status stays `approved`
- 2026-11-14T15:02:30Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-14T15:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in the four commits
  of the plan and ticked every task. On the tree of `9b9f6f6`, `go test ./...` passes, `go vet ./...` reports nothing
  and `gofmt -l .` prints no file name. `go list -deps ./internal/smartmonitor` lists no package outside the standard
  library, there is no `go.sum`, and the grep of the plan for `time.Sleep`, `time.After`, `time.NewTimer` and
  `time.Tick` under `internal/` finds nothing. The tests of the upload were written after `upload.go`, not before it,
  which departs from the order of the plan. Not checked: any behavior on the real device, which this unit does not
  cover
- 2026-11-14T15:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, noted that the commits carry no
  `Co-Authored-By` trailer, because the standards forbid it. The status stays `in-progress`
- 2026-11-14T15:06:26Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a full review of
  `changes/0002-device-protocol/` at `ba8c8ec69c6d395e7661a5790a79d9efb5346ba5` against merge base
  `d06da9e62a806f1fd847fdcd45da5834b0f4f9ad`. Returned P1, P2 and C1 in
  `.reviews/changes/0002-device-protocol/review-20261114T150521Z.md`. The three project suite commands passed,
  including an uncached test run with module-network access disabled; `npm run check-changes` passed on the
  committed implementation. Device behavior and Windows execution were not verified. The status stays `in-progress`
- 2026-11-14T15:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accepted C1 by the definition of
  the orchestrator, which accepts findings the implementer agrees with, and not as a decision of the human. The
  parameter of every test function and of the helper `expectUploadError` is now `test` and not `t`, a naming-only
  change. `go test -count=1 ./...` passes, `go vet ./...` reports nothing and `gofmt -l .` prints no file name
- 2026-11-14T15:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accepted P2 by the definition of
  the orchestrator, and not as a decision of the human. The fake answers ACK to an unscripted read so that a test
  scripts only its exceptions, and the tests that end in an error check the last recorded call. `plan.md` was
  corrected to describe that, instead of the exhaustion check it promised
- 2026-11-14T15:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, proposes declining P1. The
  upload tests were written after `upload.go`, and the record already says so. Editing the code now cannot change the
  order, and no spec requirement or rule of the standards is broken by it, so there is nothing to change. The finding
  stays open until the human decides it
- 2026-11-14T15:07:54Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, logged the decision on P1, which the
  orchestrating session [Claude Code, anthropic/claude-fable-5-2] made under the delegation in `AGENTS.md`, and not
  the human in person: the proposed decline of P1 is accepted. The order in which the upload tests were written cannot
  be changed afterwards, the record states the departure, and no requirement of the spec or rule of the standards is
  broken. The status stays `in-progress`
- 2026-11-14T15:08:19Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a delta review of
  `changes/0002-device-protocol/` at `0b7d654cd3935b818f5ad19d9434815352d0f0f1` against the previous review's
  commit `ba8c8ec69c6d395e7661a5790a79d9efb5346ba5`. Verified the C1 and P2 corrections and the recorded P1
  disposition, and returned no new findings. The review is saved in
  `.reviews/changes/0002-device-protocol/review-20261114T150819Z.md`. The uncached tests with module-network
  access disabled passed, vet reported nothing, gofmt printed no filenames, and the change check passed on the
  committed work. Device behavior and Windows execution were not verified. The status stays `in-progress`
- 2026-11-14T15:09:05Z [Claude Code, anthropic/claude-fable-5-2] under the delegation in `AGENTS.md`, after the second review returned
  no finding and with P1 decided, set the status to `implemented`. The assumptions of D5 to D10 stay unconfirmed
  until the unit that adds the Linux transport is checked on the device

## Amendments
