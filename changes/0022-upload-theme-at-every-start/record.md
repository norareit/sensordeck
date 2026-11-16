---
kind: bug
status: draft
timestamp: 2026-11-16T19:59:40Z
supersedes: 0013-theme-key-and-upload-on-change
---

# The theme uploaded at every start of `run`

## Log

- 2026-11-16T19:59:40Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session that `run` on Linux sent the tags of one theme to a screen that Windows had given another,
  because the state file of unit 0013 is kept per machine and per operating system, and from the fix the human chose,
  against the spec and record of unit 0013, the spec of unit 0021, the README, the code of `run`, `upload`, the theme
  holder, the state file and its package, the run config, and the Linux and Windows transports. Decided: `run` uploads
  the theme of its config at every start; within one `run` it uploads again only after a reload whose compiled theme
  differs byte for byte from its last completed upload, kept in memory, which an upload that begins and a reload to
  `tags` clear; it uploads nothing when the screen comes back after a replug or the restart of a hung screen; `run` and
  `upload` no longer touch the state file, and an existing one is left alone and documented as deletable; the state
  file code, the hash and the serial of the open transport, which served only it, are removed, which supersedes R13 of
  unit 0016 too. Items 1 to 3 of "Beyond the request" are left out. The device check is the human's and the unit does
  not wait for it. The status is `draft`
- 2026-11-16T20:00:21Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decision of
  the orchestrating session under the delegation in `AGENTS.md`: item 1 of "Beyond the request" is taken, because the
  human's screen hangs at every cold boot, so the first upload of every boot times out, and the wait of 10 seconds after
  a restart that recovered the screen would be added to every boot for nothing. Added R21, R22 and the definition of a
  restart that recovered the screen, which is one in which the screen left the bus and its node was opened again,
  since the first write that succeeds is the upload itself. R6, R7, D8, a non-goal and item 1 were changed to match.
  Items 2 and 3 are declined and stay in the list. The status stays `draft`
