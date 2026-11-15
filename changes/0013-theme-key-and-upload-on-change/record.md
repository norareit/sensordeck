---
kind: feature
status: approved
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
- 2026-11-15T12:14:10Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decision of
  the orchestrating session under the delegation in `AGENTS.md`: item 1 of "Beyond the request" is taken, because
  without it a manual `upload` of another file leaves the state file naming the earlier theme, so that `run` skips the
  upload it needs. Added R35 to R38 and D20: `upload` removes the screen's entry before the upload and records the hash
  after a completed one, through the same code as `run`. R8 now says what a failed record after a completed upload
  does, R30, R32, R33, D17, a non-goal and item 1 were changed to match. Items 2 to 4 are declined and stay in the
  list. D8 and D15 stand as written. The status stays `draft`
- 2026-11-15T12:14:53Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: it reuses the
  upload and the reconnect of `run`, decides each open question of the request with a reason, and every requirement
  can be tested in the sandbox with the fake screen. Item 1 of "Beyond the request" was taken, because without it a
  manual upload would leave the state file false, and the spec was amended for it; items 2 to 4 are declined. D8 and
  D15 stand as written. The device check is the human's and the unit does not depend on it. The approval that follows
  is made under that delegation
- 2026-11-15T12:14:53Z [human] approved the spec. The status is `approved`
