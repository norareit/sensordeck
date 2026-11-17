---
kind: bug
status: draft
timestamp: 2026-11-17T12:05:47Z
---

# One sensordeck process at a time on the controller and on the screen

## Log

- 2026-11-17T12:05:47Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of the
  orchestrating session and the observations on the human's machine of 2026-11-16 and 2026-11-17. The sequence was not
  reproduced in the sandbox, which has no device; the root cause is stated from the reported log lines and the code.
  The lock on Linux is `flock` on the device node rather than the lock file under `$XDG_RUNTIME_DIR` the request
  suggested, for the reasons of D2. The status is `draft`

## Amendments
