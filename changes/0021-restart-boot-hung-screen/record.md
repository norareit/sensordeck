---
kind: bug
status: draft
timestamp: 2026-11-16T14:50:42Z
supersedes: 0020-recover-hung-screen
---

# Restarting a screen whose firmware hung at boot

## Log

- 2026-11-16T14:50:42Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of the
  orchestrating session that the suspend recovery of unit 0020 does not clear the hang at every cold boot, and from the
  facts it and another session of the human's established on the device on 2026-11-16, against the udev rule and its
  test, the Linux transport and its reading of the USB device in sysfs, the feeder of `run`, the one-shot commands, the
  test tree of `fakeusb`, `docs/protocol.md`, the README and the spec and record of unit 0020. Decided: the restart by
  requests for string descriptors from index 4 up to 255 within 10 seconds, through the usbfs node with usbhid bound;
  the udev rule gives that node to the session user by `uaccess`, and the suspend recovery and its rule line are
  removed, with keeping it as a fallback offered as item 1 of "Beyond the request"; `run` restarts at the first round
  of a hang and then at most once a minute, writes the index the screen left at, and writes each line of a failure
  once in a hang; the one-shot commands restart and end with an error, and do not write again; no end-to-end test. Not
  reproduced by the specifier: the observations are the sessions' on the device. The status is `draft`
