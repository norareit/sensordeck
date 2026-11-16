---
kind: bug
status: in-progress
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
- 2026-11-16T14:52:50Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  observed on the human's machine on 2026-11-16 by this session and another session of the human's, what was observed
  once or is not established is marked as such, the restart uses only GET_DESCRIPTOR requests and never a USB reset,
  and every requirement can be tested in the sandbox with a fake usbfs seam and temporary sysfs and dev trees. The
  removal of the suspend recovery of unit 0020 is accepted for the reasons of D4. Items 1 and 3 of "Beyond the
  request" are declined. Item 2 is not taken into the unit: the orchestrating session adds the amendment to the record
  of `0020-recover-hung-screen` itself when it merges this unit. The device check is the human's and the unit does
  not depend on it. The approval that follows is made under that delegation
- 2026-11-16T14:52:51Z [human] approved the spec. The status is `approved`
- 2026-11-16T14:58:25Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote `plan.md` against the tree of the branch:
  the udev rule and its test; the restart as a function of `internal/hidraw` behind seams for usbfs, the hidraw node,
  the clock and the wait, with the real ioctl `USBDEVFS_CONTROL` in a file of its own; the interface of the command
  line renamed from `screenPower` to `screenRestarter`; the one-shot commands; the feeder of `run`; and the two
  documents, in five tasks and four commits, with tasks 3 and 4 sharing one. The plan reads R8 together with R15: the
  time of the last restart that sent a request is kept across hangs, so a hang that begins less than a minute after a
  restart makes its first restart only once the minute has passed, and the line `restarting the screen to recover it`
  is written with the first restart the hang makes rather than at a round that makes none. The one-shot commands keep
  the default handling of Ctrl+C, which ends the restart with the process, as R5 asks. The status stays `approved`
- 2026-11-16T15:05:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on the branch. The status is
  `in-progress`
- 2026-11-16T15:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built tasks 1 to 5 of `plan.md` in four
  commits. Two deviations from the plan, both in `plan.md` as built: the interface that `Restart` takes for the
  transport is named `Reopenable` and not `Screen`, since `Screen` is already the type that discovery gives in
  `internal/hidraw`; and the fake restarter of the tests of `internal/cli` is in `hang_test.go` with the tests of
  `restartAfterHang`, as the plan has it, and its constructor sets it on the harness. No end-to-end test was added, by
  D10. The restart against a real screen and the real ioctl `USBDEVFS_CONTROL` were not run: only the number of the
  ioctl is tested, on amd64, and the device check is the human's, by D13
