---
kind: bug
status: in-progress
timestamp: 2026-11-15T22:04:29Z
supersedes: 0004-linux-sensors-and-run
---

# The clock of run sent every second

## Log

- 2026-11-15T22:04:29Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session that the time on the real screen moves once a minute under `run`, against the feeder of
  `run`, its tests, the README, `docs/protocol.md` and the specs of units 0004, 0008 and 0013. The root cause is read
  from the observation on the device and not reproduced in the sandbox; D13 of unit 0004 rested on a check that showed
  only that the clock report sets the clock. Decided: the clock is sent once a second, in ticks on the whole seconds of
  the time that write only the clock, while the rounds keep their moments and their work; a clock report is due when
  its second differs from the last one, and stays due at the start, after a reopening and after an upload. The unit
  supersedes 0004 in R20, D13 and the clock case of R32, and the statement on the clock in the context of 0013. The
  unit has a plan, by D8. The check on the device is the human's, and the unit does not wait for it. The status is
  `draft`
- 2026-11-15T22:05:54Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its cause is the
  human's observation on the device, its fix sends the clock at each whole second without changing the rounds, and
  every requirement can be tested in the sandbox with fake time. The unit keeps a `plan.md` as D8 decides. Items 2 and
  3 of "Beyond the request" are declined. Item 1 is not taken into the unit: the orchestrating session adds the
  amendment to the record of `0004-linux-sensors-and-run` itself when it merges this unit. The device check is the
  human's and the unit does not depend on it. The approval that follows is made under that delegation
- 2026-11-15T22:05:54Z [human] approved the spec. The status is `approved`
- 2026-11-15T22:14:30Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote `plan.md` against the tree
  of the branch: a tick at each whole second beside the rounds of `feed`, one helper for the clock of round and tick,
  and `onScreen` on the theme holder for the tick. A throwaway prototype of the schedule, run against the suite and
  reverted, named the existing tests whose expected reports or counts of waits change, which the plan lists. The
  status stays `approved`
- 2026-11-15T22:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-15T22:41:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the schedule of rounds and
  ticks in `feed`, `onScreen` on the theme holder, the tests of R10 and R11, the scaled counts of waits of the keeper
  tests, the updated expected reports of the existing tests, and the README, protocol and test-comment changes. Added
  a task to `plan.md` for work it did not list: `TestRunSendsTheKeptProbesBetweenReadingsOfTheController` in
  `e2e/commander_test.go` took the next three reports after a change to be value reports, which fails now that a clock
  report stands between them once a second, so it counts the value reports only. Two details of the tests differ from
  the plan: the test of a tick while the theme is not on the screen runs 16 waits and expects `UCVC`, because the
  failed upload advanced the fake clock by a second and so the retry came at the round at 15 seconds, and a clock
  report of the tick follows the upload. The tests of R10 were written after the schedule and not before; they fail
  against the tree without the schedule and `onScreen`, which was checked by stashing `run.go` and `themeholder.go`
- 2026-11-15T22:50:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran the five commands of the suite on the final tree: `go test ./...` passed with no failing package, `go vet ./...`, the build and the vet for Windows reported nothing, and `gofmt -l .` printed no file name. The end-to-end suite also passed four times in a row with `-count=4`. The check on the device, that the seconds on the screen change every second, is not done, and is the human's. The status stays `in-progress`
- 2026-11-15T22:55:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, made one commit, `13c87b6`, for the work that the plan splits over its last two commit points, because the documents and the code were done together, and ran `npm run check-changes` on it, which reported nothing. The status stays `in-progress`
- 2026-11-15T22:23:29Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, completed a full review of
  `0017-clock-every-second` at `38849817278d54e226ed82b6bf036987f66ed43f` against its merge base with
  `origin/master`. Returned P1; the full findings and verification are in
  `.reviews/0017-clock-every-second/review-round-1.md`. The five suite commands, the change check and a fresh run of
  the CLI and end-to-end tests passed. The physical-screen behavior, historical test order and historical repeated
  test runs could not be independently verified; Windows execution was not performed. The status stays `in-progress`
