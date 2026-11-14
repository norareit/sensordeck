---
kind: feature
status: in-progress
timestamp: 2026-11-14T21:54:29Z
---

# The temperature probes of the Corsair Commander Core XT

## Log

- 2026-11-14T21:54:29Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the facts of a
  black-box trace of the hidraw traffic of one run of liquidctl and from the sysfs of the host, without reading any
  code of liquidctl. The status is `draft`
- 2026-11-14T21:55:57Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decisions of
  the orchestrating session under the delegation in `AGENTS.md`: item 2 of "Beyond the request", `sensors` catching
  SIGINT and SIGTERM while it reads so that hardware mode is restored, is taken into the unit as R22, with its tests
  in R23, and D13 now leaves only SIGKILL and a loss of power as the limitation; R8 refers to R22. Items 1, 3 and 4 are
  declined and stay in the list. The status stays `draft`
- 2026-11-14T21:56:33Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: every
  requirement can be tested in the sandbox with the fake controller, and the eight commands and the layout of the
  answers match the hidraw trace of the host byte for byte, with 37.3 and 22.7 decoded as liquidctl printed them. The
  timestamps of the two entries above, which did not match their commits, were corrected to the times of those
  commits. Item 2 of "Beyond the request" is taken as R22 and R23, and items 1, 3 and 4 are declined. The approval
  that follows is made under that delegation
- 2026-11-14T21:56:33Z [human] approved the spec. The status is `approved`
- 2026-11-14T22:00:41Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  the branch `feature-0005-commander-core-xt-probes`. It starts with two refactorings that keep the behavior, the
  listing of hidraw entries and the opening of a node in `internal/hidraw` and the pseudo terminals of the fake screen
  in `internal/fakepty`, so that the driver and the fake controller repeat neither. Every task names the requirements
  it delivers, and every requirement from R1 to R23 has a task. The status stays `approved`
- 2026-11-14T22:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work in round 1. The status is
  `in-progress`

## Amendments
