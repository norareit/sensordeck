---
kind: feature
status: draft
timestamp: 2026-11-15T12:11:23Z
---

# The theme in the config file, uploaded when it changes

## Log

- 2026-11-15T12:11:23Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for the first half of step 8 of the roadmap, against the specs of units 0004, 0008, 0009
  and 0010 to 0012 and the code of `internal/cli`, `internal/config`, `internal/hidraw`, the upload of
  `internal/smartmonitor` and the fake screen. Decided: `theme` and `tags` exclude each other; the hash is the SHA-256
  of the compiled file, with no cache of compiled files; an empty serial is a key like any other; the entry is removed
  before an upload; a failed upload is retried after 10 seconds doubling to 5 minutes; the check is made at the start,
  after a reload that changes the theme and after the screen came back; the config file is checked by its modification
  time and size each round; a reload applies the theme and the tags only; `theme use` edits one line and writes only a
  config that `run` accepts. The device check is the human's and the unit does not wait for it. The status is `draft`
