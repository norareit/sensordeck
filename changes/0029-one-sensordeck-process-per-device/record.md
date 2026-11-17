---
kind: bug
status: in-progress
timestamp: 2026-11-17T12:05:47Z
---

# One sensordeck process at a time on the controller and on the screen

## Log

- 2026-11-17T12:05:47Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of the
  orchestrating session and the observations on the human's machine of 2026-11-16 and 2026-11-17. The sequence was not
  reproduced in the sandbox, which has no device; the root cause is stated from the reported log lines and the code.
  The lock on Linux is `flock` on the device node rather than the lock file under `$XDG_RUNTIME_DIR` the request
  suggested, for the reasons of D2. The status is `draft`
- 2026-11-17T12:06:09Z [human] approved the spec. The status is `approved`
- 2026-11-17T12:11:19Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the branch. It
  adds the package `internal/devicelock`, takes the locks where the screen and the controller are opened, and words the
  refusals in the command line; the format of `/proc/locks` it parses was checked in the sandbox. The status stays
  `approved`
- 2026-11-17T12:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`

## Amendments
- 2026-11-17T12:06:08Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its cause is the
  conflict the human and the orchestrating session saw on 2026-11-16 and 2026-11-17, the lock on the node itself (D2)
  is accepted over the lock file of the request because every process that opens the device sees it and the kernel
  releases it at any end, and every requirement can be tested in the sandbox with two processes against the fakes.
  Items 1 to 3 of "Beyond the request" are declined. The device check and Windows are the human's and the unit does
  not depend on them. The approval that follows is made under that delegation
