---
kind: bug
status: in-progress
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
- 2026-11-16T20:00:51Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: it implements the
  human's choice of an upload at every start of `run`, removes the state file and its code, keeps an upload within
  one run only for a changed theme, and every requirement can be tested in the sandbox with the fake screen. Item 1 of
  "Beyond the request" was taken, because the screen hangs at every cold boot and the retry wait would otherwise add
  its seconds to every boot, and the spec was amended for it; items 2 and 3 are declined. The device check is the
  human's and the unit does not depend on it. The approval that follows is made under that delegation
- 2026-11-16T20:00:52Z [human] approved the spec. The status is `approved`
- 2026-11-16T20:04:08Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the branch: the
  theme holder of `run` uploads at every start and decides a reload by the bytes of its last completed upload, kept in
  memory; `upload` calls the upload directly; the state file, the package `uploadstate`, the hash of the config theme
  and the serial of the open Linux and Windows transports are removed, with the `--device` look-up of the sysfs entry
  and the roots `OpenPath` took only for it, while `entryOfNode` stays for the restart of unit 0021; a restart that
  recovered a hung screen clears the wait of a due upload; the README is rewritten last. Five commits, tests first in
  each. The fix needs a plan, since it takes far more than a couple of steps. The status stays `approved`
- 2026-11-16T20:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work in round 1, with no review of this unit yet. The status is `in-progress`
- 2026-11-16T20:55:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the five tasks of the plan in five commits, the last being `0e8b474`, in round 1 with no review of this unit yet. Deviations from the plan, none of which changes a requirement: the e2e test `TestRunAndUploadCreateNothingInTheCacheDirectory` was written in task 2, where its box now is ticked, because it fails until `upload` stops using the state file; and the tests of the hang cases of task 4 take the moment of the first upload that was recorded, since the report of a failed first write is not recorded by the fake. Tests were written before the code for the holder, the `run` tests and the tests of the upload made at once, whose new run tests were seen failing before `run.go` changed; the removal tasks 2 and 3 were done with the tests edited in the same step. Verified on `0e8b474`: `go test ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` all reported success and nothing, and `npm run check-changes` passed. Not checked: a run on the device and on Windows, which are the human's by D9 and D10. The status stays `in-progress`
- 2026-11-16T20:12:21Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a full review of unit 0022 at commit `40241cc746bf4922af122c6d58568698fa8b2016`, against merge base `ab51543b5410b40bca0dfaf27835120713194eba` with `origin/master`, in round 1. Returned no findings on spec conformance, plan fidelity or code quality. Saved the review in `.reviews/0022-upload-theme-at-every-start/round-1-20261116T201221Z.md`. Independently ran the five commands of the suite, `go test -count=1 ./...` including the end-to-end tests, `npm run check-changes` and the diff whitespace check successfully. Not verified: test-first execution history, the physical-device check of D9 and runtime behavior on Windows under D10. The status stays `in-progress`
