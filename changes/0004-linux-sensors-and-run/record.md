---
kind: feature
status: approved
timestamp: 2026-11-14T19:03:59Z
---

# Linux sensors and the run command

## Log

- 2026-11-14T19:03:59Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec. The status is
  `draft`
- 2026-11-14T19:06:22Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decisions of
  the orchestrating session under the delegation in `AGENTS.md`: the YAML package is the maintained fork
  `go.yaml.in/yaml/v3` at v3.0.6 or newer, the newest version on the module proxy, in place of the archived
  `gopkg.in/yaml.v3`, in R16, R25, D18 and the new D19; item 9 of "Beyond the request", the status line of the
  README, is taken into the unit as R37 and D20; the other eight items are declined and stay in the list. The status
  stays `draft`
- 2026-11-14T19:06:57Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: every
  requirement can be tested in the sandbox with fake trees, a fake `nvidia-smi` and the fake screen, and the tag 1 and
  tag 2 example matches what the human saw on the device. It was amended on the two decisions the entry above
  records, and the eight items left under "Beyond the request" are declined. The approval that follows is made under
  that delegation
- 2026-11-14T19:06:57Z [human] approved the spec. The status is `approved`
- 2026-11-14T19:14:39Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  unit's branch, after a trial of `go.yaml.in/yaml/v3` v3.0.6, which reports a key given twice only when a mapping
  is decoded into a map, and of the time limit of a program whose child keeps its output open. The status stays
  `approved`

## Amendments
