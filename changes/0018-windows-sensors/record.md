---
kind: feature
status: implemented
timestamp: 2026-11-15T23:04:03Z
---

# The sensors of the machine on Windows

## Log

- 2026-11-15T23:04:03Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for the second part of step 7 of the roadmap, against the README, `internal/sensors`,
  `internal/linuxsensors`, the choice of sources in `internal/cli`, the config, the layout and workflow tests, units 0004,
  0007 and 0016, the Windows API as Microsoft documents it, `golang.org/x/sys/windows` v0.50.0, and the source of
  LibreHardwareMonitor at its tags v0.8.5 to v0.9.6 and at `master`, with the document of its issue 2253. Decided: the
  memory, the CPU load and the network natively through `GlobalMemoryStatusEx`, `GetSystemTimes` and `GetIfTable2Ex`
  behind a seam, with a physical interface being one with `HardwareInterface` and `ConnectorPresent` and without
  `FilterInterface`, keyed by its LUID; the two-sample computations and the GPU source shared with Linux, moved in
  commits that change no behaviour, and a CR before the LF of `nvidia-smi` dropped; the temperatures and `cpu.clock` from
  the formatted `Value` of LibreHardwareMonitor's `/data.json`, the hardware told by `SensorId`, the CPU sensors by
  name and the drive and module temperatures by index 0, numbered by the numbers LibreHardwareMonitor gives; the key
  `librehardwaremonitor_url` with the default `http://localhost:8085/data.json`, read by `sensors` on Windows too; a time
  limit of 500 milliseconds; one line per outage of the monitor; and tests of the commands over a fake seam and an HTTP
  test server in place of an end-to-end test of the executable. The status is `draft`
- 2026-11-15T23:08:11Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the native
  metrics use the calls of the system, the facts of the LibreHardwareMonitor document are cited and marked confirmed or
  assumed, the shared computations move in a commit that keeps behaviour, and every requirement can be tested on Linux
  with a fake seam and documents written in the tests. LibreHardwareMonitor is MPL-2.0, so reading it was allowed, and
  no code of it is used. `CPU Package` for Intel stays in R21. Items 1 to 5 of "Beyond the request" are declined.
  Everything on Windows is the human's to check and the unit does not depend on it. The approval that follows is made
  under that delegation
- 2026-11-15T23:08:12Z [human] approved the spec. The status is `approved`
- 2026-11-15T23:15:29Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  `feature-0018-windows-sensors`, with every requirement traced to a task. Read the spec as follows where it leaves a
  detail to the plan: the host of R12 is the host without the port, so that `http://:8085/data.json` is refused; the
  line of R27 for a config that cannot be read or is not valid is the error of `run` without the prefix
  `sensordeck: `; the sentence of R2 on a procedure that cannot be found is kept for `GetIfTable2Ex` too, which
  `golang.org/x/sys/windows` binds with a panic, by recovering that panic alone as the failure of the call; and the
  proxy test of R40 also checks that the transport has no proxy, because Go never proxies a loopback address. The
  status stays `approved`
- 2026-11-15T23:16:30Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-15T23:26:31Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built tasks 1 to 10 of the plan in
  ten commits, from `e3655fb` to `96e60df`, with the tests of each written with its code and all tasks ticked. Details the
  plan left open and that the code settled: `internal/nvidiasmi/source_test.go` holds its own copy of `names` besides
  `everything` and `read`, because its tests use it; the assertions of `procstat_test.go` and `netdev_test.go` that the
  moved source holds no sample after an unwanted reading (`previous != nil`) are dropped, because the field is private to
  `internal/sampling`, and what they showed, no wait and no clock asked, is still checked; the sensors of the drive and
  the module are found by their `SensorId` alone, and their `Type` decides in the reading of the value, so a
  `temperature/0` whose type is not `Temperature` gives no value and keeps its number; the dial error of a refused
  connection names the address and port, as Go gives it, and is the text of the error of the client of R17
- 2026-11-15T23:26:31Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran the suite of `AGENTS.md` on the
  tree at `96e60df`, which holds the code and the README: `go test ./...` reported all packages ok, including `e2e` and
  `internal/cli`, `go vet ./...` reported nothing, `gofmt -l .` printed no file name, `GOOS=windows GOARCH=amd64 go build
  ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` reported nothing, and `GOOS=darwin GOARCH=arm64 go build ./...`
  succeeded. `git diff origin/master` shows no change of `go.mod`, `go.sum`, `THIRD_PARTY_LICENSES` or
  `internal/cli/sources_other.go`, no Go file imports `C`, and no file of `internal/windowssensors` other than the two
  of the seam, nor any non-test file of `internal/sampling`, `internal/nvidiasmi` or `internal/librehardwaremonitor`,
  holds a `go:build` line. `npm run check-changes` passed. Not checked: everything on Windows, that is, the calls of the
  seam run on a real system, the job of R43 on GitHub, and LibreHardwareMonitor itself; the "Unverified until the human
  runs it" section of the spec lists them, and none of them has been run. The status stays `in-progress`
- 2026-11-15T23:28:27Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, completed a full review of
  `changes/0018-windows-sensors` at `3d2a828bb51113b9cfaa9836350de76e195f6e4f`, against merge base
  `d1e1f252f1ad97b5b4872d00599fdf8f78c1a668` with `origin/master`. Returned C1; the findings are saved in
  `.reviews/0018-windows-sensors/review-round-1.md`. Independently ran all five commands of the project suite, a fresh
  `go test -count=1 ./...` including end-to-end tests, the Darwin cross-build, compilation of the Windows seam tests,
  and `npm run check-changes`, all successfully. Could not verify real Windows execution, the GitHub Windows job,
  physical hardware or a real LibreHardwareMonitor server. The status stays `in-progress`
- 2026-11-15T23:30:36Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accepted finding C1 of
  `.reviews/0018-windows-sensors/review-round-1.md`, accepted by the definition of the orchestrator for this round and not
  decided by the human. `internal/librehardwaremonitor/tree.go` no longer uses `any`: the body is decoded into a
  `json.RawMessage`, each node into a map of `json.RawMessage` fields, and the strings and the children are decoded from
  those, so a field of another type still costs that field only and the children of a node that is no sensor are still
  searched. The fixtures of `tree_test.go` use the same representation. The plan's sentence on the decoding is corrected
  to match. The trailing blank line of this record that the review named is removed
- 2026-11-15T23:30:36Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran the suite of `AGENTS.md` on the
  tree at the commit that applies C1: `go test -count=1 ./...` reported all packages ok, `go vet ./...` and `gofmt -l .`
  reported nothing, and `GOOS=windows GOARCH=amd64 go build ./...` and `go vet ./...` reported nothing. Real Windows
  execution is not checked. The status stays `in-progress`
- 2026-11-15T23:31:37Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, completed a delta review of
  `changes/0018-windows-sensors` at `a8d62fc5fc3f0dad79daff4369a594c77fb15621`, compared against the previous review's
  commit `3d2a828bb51113b9cfaa9836350de76e195f6e4f`. Checked the correction of C1 and returned no new findings on any
  axis; the review is saved in `.reviews/0018-windows-sensors/review-round-2.md`. Independently ran
  `go test -count=1 ./...` including the end-to-end tests, `go vet ./...`, `gofmt -l .`, the Windows amd64 build and vet,
  and `npm run check-changes`, all successfully. Could not verify real Windows execution, the GitHub Windows job,
  physical hardware or a real LibreHardwareMonitor server. The status stays `in-progress`
- 2026-11-15T23:33:17Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the findings: those of
  round 1 were corrected, C1 of round 2 was corrected, and round 3 returned none, so no blocking finding is open. The
  timestamps of five entries of the implementer, which did not match their commits, were corrected to the times of the
  commits that added them. Every task of `plan.md` is ticked, and the five commands of the suite passed on the
  branch. On the Linux host, `sensors` of the executable built from the branch still listed every metric it listed
  before the move of the shared code, the two probes of the Commander included. Everything on Windows is the human's
  to check and follows the merge. The status is `implemented`
