---
kind: bug
status: draft
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
