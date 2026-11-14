---
kind: feature
status: in-progress
timestamp: 2026-11-14T19:03:59Z
---

# Linux sensors and the run command

## Log

- 2026-11-14T19:03:59Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec. The status is
  `draft`
- 2026-11-14T19:06:22Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decisions of
  the orchestrating session under the delegation in `AGENTS.md`: the YAML package is the maintained fork
  `go.yaml.in/yaml/v3` at v3.0.6 or newer, the newest version on the module proxy, in place of the archived
  `gopkg.in/yaml.v3`, in R16, R25, D18 and the new D19; item 9 of "Beyond the request", the status line of the
  README, is taken into the unit as R37 and D20; the other eight items are declined and stay in the list. The status
  stays `draft`
- 2026-11-14T19:06:57Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: every
  requirement can be tested in the sandbox with fake trees, a fake `nvidia-smi` and the fake screen, and the tag 1 and
  tag 2 example matches what the human saw on the device. It was amended on the two decisions the entry above
  records, and the eight items left under "Beyond the request" are declined. The approval that follows is made under
  that delegation
- 2026-11-14T19:06:57Z [human] approved the spec. The status is `approved`
- 2026-11-14T19:14:39Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  unit's branch, after a trial of `go.yaml.in/yaml/v3` v3.0.6, which reports a key given twice only when a mapping
  is decoded into a map, and of the time limit of a program whose child keeps its output open. The status stays
  `approved`
- 2026-11-14T19:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-14T20:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in the seven commits
  of the plan. Two things differ from the plan as written: the commit `Add the command sensors` holds the harness
  fields and the third root, but its `main.go` does not yet name `run`, and the next commit adds it, so each of the
  two commits builds and passes on its own; and the layout test writes the name of the archived YAML package in two
  pieces, so that `grep -rn 'gopkg.in/yaml' --include='*.go' .`, a verification item of the plan, finds nothing
- 2026-11-14T20:12:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran the suite on the final tree:
  `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=windows GOARCH=amd64 go vet ./...` all passed or reported nothing, and so did `CGO_ENABLED=0 go build ./...`
  and `go mod tidy -diff`. `go.mod` holds `go 1.27`, no `toolchain` line, and cobra, x/sys and `go.yaml.in/yaml/v3`
  v3.0.6 as its direct requirements. The end-to-end suite passed three times in a row with `-count=3`. Not checked: a
  real device, a real `nvidia-smi`, the sensors of the host, and anything on Windows beyond the build and vet
- 2026-11-14T19:25:02Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a full review of
  `changes/0004-linux-sensors-and-run` at `2d2ffba4e7148743aad11123d11a21c4fdba40bc`, against the merge base
  `6e1c57771f204a06eb3c531a752a508308163b61` with `origin/master`. Returned S1, S2 and C1, saved in
  `.reviews/changes/0004-linux-sensors-and-run/review-round-1-20261114T192402Z.md`. Ran the five project suite
  commands, the static build, the module tidy diff and the change check, all passing; the focused GPU tests also
  passed but did not exercise their intended timeout and cancellation scenarios. Not verified: a physical screen,
  the host sensors, a real NVIDIA driver, or execution on Windows. The status stays `in-progress`
- 2026-11-14T19:26:33Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a delta review of
  `changes/0004-linux-sensors-and-run` at `7893d972a98846a09d7de5fe0fbf6069545416b8`, compared against
  `2d2ffba4e7148743aad11123d11a21c4fdba40bc`. The delta contains only the preceding review entry, with no
  implementation or test corrections. Returned the unchanged findings S1, S2 and C1, with no new findings, in
  `.reviews/changes/0004-linux-sensors-and-run/review-round-2-20261114T192605Z.md`. Reran the five project suite
  commands, the static build, the module tidy diff and the change check successfully. Physical-device behavior,
  host sensors, real NVIDIA driver output and execution on Windows remain unverified. The status stays
  `in-progress`
- 2026-11-14T19:27:53Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a delta review of
  `changes/0004-linux-sensors-and-run` at `6ce21b22defcd7cba2018c2a7fc17c3fb63f7818`, compared against
  `7893d972a98846a09d7de5fe0fbf6069545416b8`. Only the preceding review entry changed; there are no code or test
  corrections. Returned the unchanged findings S1, S2 and C1, with no new findings, saved in
  `.reviews/changes/0004-linux-sensors-and-run/review-round-3-20261114T192724Z.md`. Reran the five project suite
  commands, the static build, the module tidy diff and the change check successfully. Physical-device behavior,
  host sensors, real NVIDIA driver output and execution on Windows remain unverified. The status stays
  `in-progress`
- 2026-11-14T19:30:19Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, corrected S1 of the reviews in `.reviews/0004-linux-sensors-and-run/`, accepted by the orchestrating
  session under the delegation in `AGENTS.md`, in `c8389a3`: `cpuTemperatures` now tests whether the metric of a sensor is
  wanted before it reads the input, so a config that maps only `cpu.clock` reads no k10temp input. The regression test
  makes the Tctl and Tccd1 inputs named pipes, which block a read, and fails with the old code
- 2026-11-14T19:30:19Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, corrected S2, accepted by the orchestrating session under the delegation in `AGENTS.md`, in `c8389a3`: the
  clock deadline of a round is measured from the time the round actually starts, and the grid stays the schedule only.
  The approach in `plan.md` is corrected to say so, because the nominal time of a round that follows a long one is
  earlier than its real start. The new fake-time test (reopening at 2.5s, a round of 1.6s ending at 62.6s) fails with
  the old code and passes
- 2026-11-14T19:30:19Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, corrected C1, accepted by the orchestrating session under the delegation in `AGENTS.md`, in `c8389a3`: the fake
  `nvidia-smi` script runs with the PATH the test had, so it can start `sleep`, while the lookup of the program under
  test still finds only the fake, which a test asserts. The markers use a shell builtin. The tests of the time limit
  with a child holding the output, of the stop signal and of the not-run case now assert that the program started,
  that it did not run to its end, and that the time is at least the limit or the stop; a test shows that the marker
  detects a run
- 2026-11-14T19:30:19Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran the suite on the tree of `c8389a3`: `go test -count=1 ./...` including the end-to-end suite, `go vet ./...`,
  `gofmt -l .`, and the build and vet with `GOOS=windows GOARCH=amd64` passed or reported nothing. Not checked: a real
  device, a real `nvidia-smi`, and execution on Windows. The status stays `in-progress`

## Amendments
