---
kind: bug
status: approved
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
