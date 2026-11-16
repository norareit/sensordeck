---
kind: bug
status: approved
timestamp: 2026-11-16T13:25:49Z
supersedes: 0004-linux-sensors-and-run
---

# Recovering a screen whose firmware hung after a USB reset

## Log

- 2026-11-16T13:25:49Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session of a screen that shows no values after a cold boot, and the three facts it established on
  the device on 2026-11-16, against the udev rule, the Linux transport and its discovery, the feeder of `run` and its
  theme holder, the upload, the one-shot commands, the fake screen and `internal/fakepty`, `docs/protocol.md`, the
  README and the specs of units 0004, 0013 and 0017. Read on the human's machine: the screen's USB device has
  `bmAttributes` `0x80` and no `power/wakeup`, so it cannot wake the host and usbhid keeps it resumed while its node is
  open. Decided: the rule sets `autosuspend_delay_ms` to 2000 and `control` to `auto` on the USB device, with no
  `ACTION` match; the first timed-out write is a hang; `run` recovers at the next round by closing the node, waiting
  for `suspended` up to the delay plus 5 seconds, and reopening; the lines of a hang are written once each, and
  `screen recovered` at the first write that succeeds; without the rule, one line of what to do; the one-shot commands
  report and do not recover; no end-to-end test, because no write to the fake screen's terminal can time out. The
  status is `draft`
- 2026-11-16T13:28:21Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  the orchestrating session and the human observed on the device on 2026-11-16, what was observed once or inferred is
  marked as such, sensordeck sends no USB reset and writes no sysfs, and every requirement can be tested in the
  sandbox with a temporary sysfs tree and a fake transport. That opening the node resumes the device, and that a
  healthy screen suspends harmlessly, go to the human's check after the next cold boot; the unit does not depend on it.
  Item 3 of "Beyond the request" is not taken into the unit: the orchestrating session adds the amendment to the record
  of `0004-linux-sensors-and-run` itself when it merges this unit. Items 1 and 2 are declined. The approval that
  follows is made under that delegation
- 2026-11-16T13:28:22Z [human] approved the spec. The status is `approved`
- 2026-11-16T13:31:57Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote `plan.md` against the tree of the
  branch: five commits, for the udev rule, the reading of the USB power state in the Linux transport, the seam of the
  command line with the one-shot commands, the recovery of `run`, and the documents, with each requirement traced to
  its tasks. The status stays `approved`
