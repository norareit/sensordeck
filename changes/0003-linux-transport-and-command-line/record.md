---
kind: feature
status: in-progress
timestamp: 2026-11-14T15:17:19Z
---

# Linux transport and a first command line

## Log

- 2026-11-14T15:17:19Z [Claude Code, anthropic/claude-fable-5-2] as the specifier, wrote the spec. The status is `draft`
- 2026-11-14T15:20:15Z [Claude Code, anthropic/claude-fable-5-2] as the specifier, amended the spec on the decision of the
  orchestrating session under the delegation in `AGENTS.md`: item 3 of "Beyond the request", the command `raw`, is
  taken into the unit as R35 and D25, with R3, R16, R18, R32 and R33 changed for it, and the other items are
  declined and stay in the list, renumbered 1 to 9. D18 now says that the sandbox was checked for `/dev/ptmx`. The
  status stays `draft`
- 2026-11-14T15:20:37Z [Claude Code, anthropic/claude-fable-5-2] read the spec under the delegation in `AGENTS.md`: every requirement
  can be tested without the device, the counts of trace lines in R33 were computed again and hold, and the sandbox
  was checked to have the pseudo terminals the fake screen needs. The `raw` command was taken in, as the entry above
  says, and the nine items left under "Beyond the request" are declined. The approval that follows is made under
  that delegation
- 2026-11-14T15:20:37Z [human] approved the spec. The status is `approved`
- 2026-11-14T15:27:28Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  unit's branch. Every requirement R1 to R35 has a task, and every task names the requirements it delivers. A
  trial in the sandbox confirmed what the fake screen relies on: a raw pseudo terminal behind a symbolic link
  passes the bytes unchanged, keeps a report sent before the node is opened, gives one report of 64 bytes per
  read and honours a read deadline, and cobra v1.10.2 with `golang.org/x/sys` v0.50.0 resolves without
  changing the directive `go 1.27`. The status stays `approved`
- 2026-11-14T15:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on the unit's branch. The status is `in-progress`
- 2026-11-14T16:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit by its plan in the five commits of its commit points and ticked every task. On `a707027` the following were run and reported: `go test -count=1 ./...` passed in all packages, including the end-to-end tests that build the executable and run it against the fake screen; `go vet ./...` and `gofmt -l .` reported nothing; `GOOS=windows GOARCH=amd64 go build ./...` and `go vet ./...` succeeded with no output; `CGO_ENABLED=0 go build ./...` and `go build .` succeeded; `go mod tidy -diff` reported no difference and `go.mod` holds `go 1.27`, no `toolchain` line and cobra and `golang.org/x/sys` as the only direct requirements; `grep` for `udevadm trigger` in the Markdown files outside `changes/` found nothing. `go test -race -count=5 ./internal/hidraw` also passed. Not run: any check on the real device, and Windows beyond the build and vet. The status stays `in-progress`
- 2026-11-14T16:10:30Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, noted three deviations of detail from the plan that change no requirement: the test helper of `internal/hidraw` is named `openFake` because `openDiscovered` is the function under test; `parseSeconds` accepts decimal digits only by reusing `parseNumber` and refusing a `0x` prefix; and `internal/fakescreen` calls `test.Fatalf` from the goroutine of the fake when a `MoveAfter` rule has to make a terminal, which `testing` does not allow off the test goroutine, so such a failure would surface as a panic and not as a clean failure. No test reached that path
- 2026-11-14T18:10:46Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a full review of `changes/0003-linux-transport-and-command-line` at `85e7ee9dc36c5f8d7e2a6911c4bc41566f880953`, against merge base `64aa559b3a610f1eac2c0260ae91f1e184a9ee32` with `origin/master`. Returned S1, P1 and C1, saved in `.reviews/changes/0003-linux-transport-and-command-line/review-20261114T180939Z.md`. Independently ran the five project checks, the cgo-disabled build, root build, module-tidiness check, change check, five race-test runs of the transport and one race-test run of all packages, including end-to-end tests; all passed. The real device and Windows execution could not be checked. The status stays `in-progress`
- 2026-11-14T18:30:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, outcome of the review of `85e7ee9`, accepted by the definition of the orchestrator and not as a decision of the human: S1 (blocking) is accepted. A number beyond 64 bits is now a number that the range checks refuse with the error of its field, and the tests cover both fields in both bases; the plan text that said otherwise is corrected. C1 (blocking) is accepted. `session.say` takes a string and the call sites format. P1 (nonblocking) is accepted and was a fault of the implementer: ticking the tasks duplicated sections of `plan.md`, which is restored to the single text of the planning commit with every box ticked, and no content was removed. On `63f34a1`: `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .` and the Windows vet reported clean, and `npm run check-changes` passed. The status stays `in-progress`
- 2026-11-14T18:13:00Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a delta review of `changes/0003-linux-transport-and-command-line` at `8695a831f4e5a95918612f170da93fa2e4493943`, compared against `85e7ee9dc36c5f8d7e2a6911c4bc41566f880953` named by the previous full review. Verified the corrections of S1, P1 and C1 and reviewed the intervening diff on all three axes; returned no new findings. The review is saved in `.reviews/changes/0003-linux-transport-and-command-line/review-20261114T181300Z.md`. Independently ran the five project checks, the cgo-disabled build, module-tidiness check, change check, CLI race tests and diff whitespace check; all passed. The physical device and Windows execution were not checked. The status stays `in-progress`

## Amendments
