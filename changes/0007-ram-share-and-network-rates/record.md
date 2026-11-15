---
kind: feature
status: implemented
timestamp: 2026-11-15T00:11:24Z
---

# The share of RAM in use and the network rates

## Log

- 2026-11-15T00:11:24Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the tags of the
  vendor theme that the human found on the device, from the specs of units 0004 and 0005 and from the sensor registry
  and the Linux sources of the tree, with the network rates made from two samples as `cpu.util` is. The status is
  `draft`
- 2026-11-15T00:12:12Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: every
  requirement can be tested in the sandbox with fake proc and sysfs trees, the names follow `cpu.util` and
  `gpu.util`, D4 counts the interfaces the kernel ties to a device, and D5 keeps the units of the table binary. The
  three items of "Beyond the request" are declined. The approval that follows is made under that delegation
- 2026-11-15T00:12:12Z [human] approved the spec. The status is `approved`
- 2026-11-15T00:14:35Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  the branch: three rows of the registry, `ram.util` from the one read of `meminfo`, and a network source in
  `internal/linuxsensors` that keeps its sample as `cpu.util` does and takes a wait and a clock, with the test
  helpers, the end-to-end tests and the README. Every requirement has a task. The status stays `approved`
- 2026-11-15T00:15:22Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-15T00:17:30Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built every task of the plan in
  the commit points it names. The tests of the new rows, of `ram.util` and of the network source were written with the
  code and not run first against a tree without it, so that no failing run of them is recorded. Not checked: a
  reading on the real machine, which is the human's. On the tree at `5cf55bc`, plus a later commit that changes only
  `plan.md` and this record, the five commands of the suite succeeded and reported nothing, `go test -count=5 -run
  TestRunSendsTheShareOfRAMAndTheNetworkRates ./e2e` passed, and `npm run check-changes` passed. The status stays
  `in-progress`, because no review has been made yet
- 2026-11-15T00:18:53Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a full review of unit
  `0007-ram-share-and-network-rates` at `0aa59a309c6044272dfd48f12b3492d9a3654aab`, against its merge base
  with `origin/master`. Returned P1 and P2, saved in `.reviews/0007-ram-share-and-network-rates/review-round-1.md`.
  Independently ran the five required suite commands, an uncached `go test -count=1 ./...`, the five-repeat
  network end-to-end test and `npm run check-changes`; all succeeded. Live host and physical-screen readings
  and Windows execution could not be verified. The status stays `in-progress`
- 2026-11-15T00:20:07Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, answered the two findings of
  `review-round-1.md`. P1 (tests written with the code, not run first) and P2 (task boxes ticked in one sweep) are
  both true, as the record already says. Neither can be applied: history cannot be redone without inventing earlier
  results, and ticking boxes now cannot restore incremental ticking. I propose declining both, as a request for an
  exception to the two rules of the implementer role for this unit, with the reasons that the finished code and its
  tests are verified on the final tree and that nothing in the delivered work depends on the order. The findings stay
  open until the human decides them. Nothing was changed. The status stays `in-progress`
- 2026-11-15T00:21:16Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the proposed
  declines of P1 and P2 of the review of round 1: both are granted as exceptions for this unit. The tests were not
  written before their code and the tasks were ticked in one commit, which the record discloses and which cannot be
  undone without inventing history. The review found no defect of the requirements or the code, and the five
  commands of the suite passed on `1c7ef26`. The timestamps of three entries of the implementer, which did not match
  their commits, were corrected to the times of the commits that added them. No blocking finding is open, every task
  of `plan.md` is ticked, and the check on the real machine is the human's and follows the merge. The status is
  `implemented`

## Amendments
