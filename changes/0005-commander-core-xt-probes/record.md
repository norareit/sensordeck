---
kind: feature
status: implemented
timestamp: 2026-11-14T21:54:29Z
---

# The temperature probes of the Corsair Commander Core XT

## Log

- 2026-11-14T21:54:29Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the facts of a
  black-box trace of the hidraw traffic of one run of liquidctl and from the sysfs of the host, without reading any
  code of liquidctl. The status is `draft`
- 2026-11-14T21:55:57Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decisions of
  the orchestrating session under the delegation in `AGENTS.md`: item 2 of "Beyond the request", `sensors` catching
  SIGINT and SIGTERM while it reads so that hardware mode is restored, is taken into the unit as R22, with its tests
  in R23, and D13 now leaves only SIGKILL and a loss of power as the limitation; R8 refers to R22. Items 1, 3 and 4 are
  declined and stay in the list. The status stays `draft`
- 2026-11-14T21:56:33Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: every
  requirement can be tested in the sandbox with the fake controller, and the eight commands and the layout of the
  answers match the hidraw trace of the host byte for byte, with 37.3 and 22.7 decoded as liquidctl printed them. The
  timestamps of the two entries above, which did not match their commits, were corrected to the times of those
  commits. Item 2 of "Beyond the request" is taken as R22 and R23, and items 1, 3 and 4 are declined. The approval
  that follows is made under that delegation
- 2026-11-14T21:56:33Z [human] approved the spec. The status is `approved`
- 2026-11-14T22:00:41Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  the branch `feature-0005-commander-core-xt-probes`. It starts with two refactorings that keep the behavior, the
  listing of hidraw entries and the opening of a node in `internal/hidraw` and the pseudo terminals of the fake screen
  in `internal/fakepty`, so that the driver and the fake controller repeat neither. Every task names the requirements
  it delivers, and every requirement from R1 to R23 has a task. The status stays `approved`
- 2026-11-14T22:01:53Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work in round 1. The status is
  `in-progress`
- 2026-11-14T23:23:37Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in the order of the
  plan, in nine commits from the two refactorings to the documents, with the tests of each part written before its
  code. No task was added or removed. On `3736205`, `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .` (no file
  name), `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` all succeeded, and
  `npm run check-changes` passed; `grep -rn liquidctl internal e2e` found nothing. The driver and the fake controller
  were also run once with `-race`, which reported nothing. Not checked: the controller itself, which the sandbox does
  not have, so the check on the real device, including which probe is which, is the human's. The unit has not been
  reviewed, so the status stays `in-progress`
- 2026-11-14T23:25:27Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a full review of unit
  `0005-commander-core-xt-probes` at `861440096d147988f835d0955040b6a5654fecc7`, against merge base
  `557d0c555c8635ea7f126920b2c3bbe6f8dc6b3b` with `origin/master`. Returned S1; the full findings are in
  `.reviews/changes/0005-commander-core-xt-probes/round-1-review.md`. The five required checks, the change check and
  race-enabled driver, fake-controller and end-to-end tests passed. Could not verify the physical controller, a Windows
  runtime or the locally held trace. The status stays `in-progress`
- 2026-11-14T23:26:43Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accepted S1 of the review of round
  1, a finding the definition of the orchestrator for round 2 accepts and not a decision of the human. `runSequence`
  now returns before step 1 when stop is done, so a stop that arrives during discovery or opening writes nothing, and
  step 8 stays owed only once step 1 was attempted. The regression test
  `TestAStopDoneAtTheStartOfTheSequenceWritesNothing` failed before the fix with 2 writes and passes after it. On
  `f5dbb48`, `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .` (no file name) and the Windows build and vet all
  succeeded. The status stays `in-progress`
- 2026-11-14T23:27:06Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a delta review of unit
  `0005-commander-core-xt-probes` at `72eee62f4ec05e2defbcc3521f72fbae15ce9737`, from the previously reviewed
  `861440096d147988f835d0955040b6a5654fecc7`. Verified the correction of S1 and returned no new findings; the full
  review is in `.reviews/changes/0005-commander-core-xt-probes/round-2-review.md`. The five required checks and the
  change check passed. Could not verify the physical controller, a Windows runtime or the locally held trace, and did
  not replay the pre-fix regression failure. The status stays `in-progress`
- 2026-11-14T23:28:47Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the findings: S1 of
  round 1 was the one blocking finding and is corrected, and round 2 returned none, so no blocking finding is open.
  The timestamps of three entries of the implementer, which did not match their commits, were corrected to the times
  of the commits that added them. Every task of `plan.md` is ticked, and the five commands of the suite passed on
  `9b238d3`. The checks on the physical controller are the human's and follow the merge. The status is `implemented`

## Amendments
