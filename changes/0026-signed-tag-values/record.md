---
kind: bug
status: in-progress
timestamp: 2026-11-16T23:45:33Z
supersedes: 0004-linux-sensors-and-run
---

# Sending the value of a tag as a signed 16-bit number

## Log

- 2026-11-16T23:47:15Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session of the values that the human's screen showed on 2026-11-17 for 65535, 40000, 32768, 32767
  and 0 sent with `values`, which establish that the screen reads the value of a tag as a 16-bit two's complement
  number, against `docs/protocol.md`, the README, `sensors.ToValue`, the arguments of `values` and their tests, and the
  specs of the units 0002, 0003, 0004, 0007, 0011 and 0024. Decided: rounding as today and a clamp to -32768 to 32767;
  `values` takes -32768 to 65535, with an optional `-` before a decimal or hexadecimal number, so that every raw value
  that worked keeps its bytes; the documents say that the value is signed. The unit supersedes unit 0004, whose R4 it
  replaces, and changes requirements and decisions of the units 0002, 0003, 0007, 0011 and 0024, which the spec names.
  The check on the device of a negative custom metric is the human's, and the unit does not wait for it. The status is
  `draft`
- 2026-11-16T23:47:46Z [human] approved the spec. The status is `approved`
- 2026-11-16T23:49:49Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  branch: `smartmonitor.Value` takes an `int16` written as its 16-bit pattern, `sensors.ToValue` clamps to the signed
  range, `values` reads an optional `-` with a parser of its own, and the tests and documents follow. The fix takes
  more than a couple of steps, so it has a plan. The status stays `approved`
- 2026-11-17T00:05:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work from the plan. The status is
  `in-progress`
- 2026-11-17T00:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built every task of the plan. Not in the plan and added to it:
  `TestValuesAreSentAsTheScreenTakesThem` in `internal/custommetrics/number_test.go` called `ToValue` and did not
  compile with the new type, so it now wants -5 for -5 and 32767 for 70000. The commits differ from the commit points:
  `parseValue` and its tests went into the first commit with the encoding, and the move to `compileTheme` and the new
  end-to-end test went into one commit, so the move was not a commit of its own. On `4bacbc3`, `go test ./...` (the
  end-to-end suite included), `go vet ./...`, `gofmt -l .`, and `go build` and `go vet` for `GOOS=windows GOARCH=amd64`
  all passed and reported nothing. The grep of `README.md` and `docs/` for "unsigned" and "65535" finds only the
  observation of 65535 in `docs/protocol.md` and the raw range in the README. The check on the device of a negative
  custom metric is not made, and is the human's. The status stays `in-progress`
- 2026-11-16T23:54:17Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, completed a full review of
  `0026-signed-tag-values` at `d6e5e9cdb0b2a45e42a01693d31f1318d2b126ff` against merge base
  `24170d8a962d0ad41db90a9dd8afde5704afee36` with `origin/master`. Returned C1; the complete findings are in
  `.reviews/0026-signed-tag-values/review-round-1.md`. Independently ran the five required Go checks, the uncached
  end-to-end suite and `npm run check-changes`, all successfully. The physical-device check and Windows runtime
  behavior were not verified. The status stays `in-progress`
- 2026-11-17T01:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, read `review-round-1.md`. It has one
  finding, C1, blocking: the move of the compiling of the theme into `compileTheme` and the new end-to-end test share
  a commit, against the rule that a commit does not change both structure and behavior. Proposed decline, not a
  decision: the move leaves what `TestRunSendsCustomMetricsToTheTagsOfTheTheme` checks unchanged, and that test passes
  on the final tree, which `go test -count=1 ./e2e` as the review ran it confirms. The deviation is already logged, and
  splitting it means rewriting commits of a branch whose later commits depend on it, for no change in the tree.
  The finding stays open until the human decides it. No file other than this record changed. The status stays
  `in-progress`

## Amendments
- 2026-11-16T23:47:45Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are the
  values the human saw on the screen on 2026-11-17, the fix clamps to the signed range and keeps every argument of
  `values` that worked, and every requirement can be tested in the sandbox. The time of the specifier's entry was
  corrected to its commit. Items 1 to 3 of "Beyond the request" are declined: item 1 belongs to the coming unit on the
  options of number widgets, and the other two are not needed for a correct range. The device check is the human's
  and the unit does not depend on it. The approval that follows is made under that delegation
