---
kind: feature
status: in-progress
timestamp: 2026-11-15T00:29:10Z
---

# Reading the Commander Core XT less often in run

## Log

- 2026-11-15T00:29:10Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the flicker of
  the LEDs of the fans that the human saw on the device at each reading of the controller, from the trial of the
  orchestrating session on the device that a reading without software mode answers with byte 2 `0x03`, from the
  decisions of the orchestrating session on the period, and from the specs of units 0004 to 0007 and the config, the
  sensor contract, the driver of the controller and the loop of `run` in the tree. The status is `draft`
- 2026-11-15T00:29:50Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the period of
  1 minute is the one the human chose, the default keeps every valid config valid, waiting a full period after a
  failure keeps a controller that answers badly from flickering every second, and every requirement can be tested in
  the sandbox with a fake clock and the fake controller. Item 1 of "Beyond the request" is declined. The approval that
  follows is made under that delegation
- 2026-11-15T00:29:51Z [human] approved the spec. The status is `approved`
- 2026-11-15T00:34:03Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  the unit's branch. The controller is given apart from the other sources and read through a type of `run` that keeps
  its readings, and the driver does not change. The plan reads the first sentence of R8 by R22 of unit 0004: a round
  whose reopening fails reads no source, and one whose reopening succeeds goes on with the round, so that
  `TestRunReopensALostScreenOncePerRound` keeps what it expects. The new period is a method of the config, so that the
  field holds what the file gives and the existing tests that compare a whole config keep what they expect, as R17
  asks
- 2026-11-15T00:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`

## Amendments
