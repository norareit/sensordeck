---
kind: feature
status: approved
timestamp: 2026-11-17T11:32:23Z
---

# A number widget that changes colour past a threshold

## Log

- 2026-11-17T11:32:23Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of the
  orchestrating session and the observation on the device of 2026-11-17. The status is `draft`
- 2026-11-17T11:33:29Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec at the request of
  the orchestrating session: item 6 of "Beyond the request", the two values of an alarm pair always in one report of
  the value update, is now R21a with its tests, and D13 gives its reason. Items 1 to 5 are declined and stay in the
  list. The status stays `draft`
- 2026-11-17T11:33:51Z [human] approved the spec. The status is `approved`
- 2026-11-17T11:37:30Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  `feature-0028-number-alarm-colour`. Every requirement from R1 to R29 has a task. The status stays `approved`

## Amendments
- 2026-11-17T11:33:51Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its method is the
  one demonstrated on the human's screen on 2026-11-17 (a white 47, a red 95 and a white 52), it refuses the options
  that would keep a widget from being hidden, every existing `tags` block keeps its meaning, and every requirement can
  be tested in the sandbox. Item 6 of "Beyond the request" was taken, because a pair split across two reports would
  flicker every round, and the spec was amended for it; items 1 to 5 are declined. The device check is the human's and
  the unit does not depend on it. The approval that follows is made under that delegation
