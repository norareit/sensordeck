---
kind: feature
status: in-progress
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
- 2026-11-15T00:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-15T00:50:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built every task of the plan in
  the commit points it names. The tests of the new rows, of `ram.util` and of the network source were written with the
  code and not run first against a tree without it, so that no failing run of them is recorded. Not checked: a
  reading on the real machine, which is the human's. On the tree at `5cf55bc`, plus a later commit that changes only
  `plan.md` and this record, the five commands of the suite succeeded and reported nothing, `go test -count=5 -run
  TestRunSendsTheShareOfRAMAndTheNetworkRates ./e2e` passed, and `npm run check-changes` passed. The status stays
  `in-progress`, because no review has been made yet

## Amendments
