---
kind: feature
status: approved
timestamp: 2026-11-14T15:17:19Z
---

# Linux transport and a first command line

## Log

- 2026-11-14T15:17:19Z [Claude Code, anthropic/claude-fable-5-2] as the specifier, wrote the spec. The status is `draft`
- 2026-11-14T15:20:15Z [Claude Code, anthropic/claude-fable-5-2] as the specifier, amended the spec on the decision of the
  orchestrating session under the delegation in `AGENTS.md`: item 3 of "Beyond the request", the command `raw`, is
  taken into the unit as R35 and D25, with R3, R16, R18, R32 and R33 changed for it, and the other items are
  declined and stay in the list, renumbered 1 to 9. D18 now says that the sandbox was checked for `/dev/ptmx`. The
  status stays `draft`
- 2026-11-14T15:20:37Z [Claude Code, anthropic/claude-fable-5-2] read the spec under the delegation in `AGENTS.md`: every requirement
  can be tested without the device, the counts of trace lines in R33 were computed again and hold, and the sandbox
  was checked to have the pseudo terminals the fake screen needs. The `raw` command was taken in, as the entry above
  says, and the nine items left under "Beyond the request" are declined. The approval that follows is made under
  that delegation
- 2026-11-14T15:20:37Z [human] approved the spec. The status is `approved`
- 2026-11-14T15:27:28Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  unit's branch. Every requirement R1 to R35 has a task, and every task names the requirements it delivers. A
  trial in the sandbox confirmed what the fake screen relies on: a raw pseudo terminal behind a symbolic link
  passes the bytes unchanged, keeps a report sent before the node is opened, gives one report of 64 bytes per
  read and honours a read deadline, and cobra v1.10.2 with `golang.org/x/sys` v0.50.0 resolves without
  changing the directive `go 1.27`. The status stays `approved`

## Amendments
