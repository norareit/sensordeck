---
kind: feature
status: draft
timestamp: 2026-11-14T21:58:00Z
---

# The temperature probes of the Corsair Commander Core XT

## Log

- 2026-11-14T21:58:00Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the facts of a
  black-box trace of the hidraw traffic of one run of liquidctl and from the sysfs of the host, without reading any
  code of liquidctl. The status is `draft`
- 2026-11-14T22:05:00Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decisions of
  the orchestrating session under the delegation in `AGENTS.md`: item 2 of "Beyond the request", `sensors` catching
  SIGINT and SIGTERM while it reads so that hardware mode is restored, is taken into the unit as R22, with its tests
  in R23, and D13 now leaves only SIGKILL and a loss of power as the limitation; R8 refers to R22. Items 1, 3 and 4 are
  declined and stay in the list. The status stays `draft`

## Amendments
