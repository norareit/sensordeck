---
kind: feature
status: implemented
timestamp: 2026-11-15T23:39:29Z
---

# Starting `run` at logon on Windows

## Log

- 2026-11-15T23:39:29Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for the Windows half of step 8 of the roadmap and the last part of step 7, against the specs
  of units 0014, 0016 and 0018, the service code of `internal/cli` and its seam, the stop of `run`, the CI workflow and
  the layout test, and the Microsoft documentation of the Task Scheduler and the console, which the spec cites. Decided:
  a task of the Task Scheduler created by `schtasks.exe` from an XML definition that `install` writes, running as the
  user with an interactive token and the least privilege, with no time limit and not stopped on batteries; the presence
  of that file as the mark of an installed task, as the unit file on Linux; a hidden flag `--background` of `run`, which
  only the task passes, that writes the lines of `run` to a log of two files of at most 1 MiB in `%LocalAppData%`,
  detaches from the console with `FreeConsole`, and tries a failed start again every 10 seconds, so that the restart
  does not depend on the Task Scheduler; a named event by which `install` and `uninstall` stop a running background run
  cleanly, with `schtasks /End` after 30 seconds, and by which `status` tells whether it runs; `conhost.exe --headless`,
  a GUI-subsystem build, the `Run` key and the Startup folder rejected. The checks on Windows are listed as unverified
  and the unit does not depend on them. The status is `draft`
- 2026-11-15T23:43:05Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the task needs no
  administrator rights by the documentation it cites, the log and the stop event stand in for the journal and the
  signal, Linux behaviour is unchanged, and every requirement can be tested on Linux through the service seam. What the
  documentation does not settle is listed as unverified until the human runs it, and the unit does not depend on it.
  Items 1 to 6 of "Beyond the request" are declined. The approval that follows is made under that delegation
- 2026-11-15T23:43:05Z [human] approved the spec. The status is `approved`
- 2026-11-15T23:48:12Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  the branch: a refactoring with no change of behaviour first, then the package `internal/winbackground` for the stop
  event and the detach, the log, the background run, the task file and the commands on Windows, and the README. The
  plan decides what the spec leaves to the implementation: a failed check of the stop event fails `install` and
  `uninstall` and is `unknown` in `status`, an event gone before the signal is a stop that succeeded, `Enabled` is
  read as an `xs:boolean`, an error after the start of a background run and a stop event that cannot be created are
  written to the log with exit status 1, and the exact texts. The status stays `approved`
- 2026-11-15T23:49:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on the branch `feature-0019-windows-startup-task`. No
  review exists yet. The status is `in-progress`
- 2026-11-15T23:56:47Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in the six commits
  of the plan after the first and the documents, and ticked each task as it was done. Decisions of the implementation,
  none of which changes what the spec makes true: the refactoring of the first commit moved the `lockedWriter` to the
  top of the `RunE` of `run` as the plan says, and a background run does not wrap its standard error in it because the
  log is locked itself; `runInBackground` makes the retry loop and `holdBackgroundStop` holds the event, detaches and
  turns a signal of the event into the end of the context; a `Close` of the event of `winbackground` ends its waiting
  goroutine through an unnamed event, as the plan says. The tests of the Windows calls (`internal/winbackground`) and
  every check on Windows are not run here and are open for the human, as the spec lists. The tests of the other parts
  run on Linux with the system set to `windows`
- 2026-11-15T23:56:47Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran the suite on the tree of
  `d13489f`, whose later commit changes documents only: `go test ./...` passed with `-count=1` for `internal/cli`, `e2e`
  and the root package, `go vet ./...` and `gofmt -l .` reported nothing, and `GOOS=windows GOARCH=amd64 go build ./...`
  and `go vet ./...` succeeded and reported nothing. The test of `internal/winbackground` was only compiled and vetted
  for Windows, not run. No review has been made yet, so the unit stays `in-progress`
- 2026-11-15T23:58:54Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a full review of
  `0019-windows-startup-task` at `397204d16c79e37bd47e17e45655d8f981d35f7a`, against merge base
  `d4e8444acf1d617c25cf67fc2c222bb2371a8e1f`. Returned no findings on spec conformance, plan fidelity or code quality;
  the full review is `.reviews/0019-windows-startup-task/review-round-1-397204d.md`. Independently ran
  `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...`,
  `GOOS=windows GOARCH=amd64 go vet ./...` and `npm run check-changes`, all successfully. The Windows-only event tests,
  real Windows and device checks, and Windows CI execution could not be verified on this Linux host. The status stays
  `in-progress`
- 2026-11-16T00:00:42Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the findings: round 1
  returned none, so no blocking finding is open. The timestamps of two entries of the implementer, which did not match
  their commits, were corrected to the time of the commit that added them. Every task of `plan.md` is ticked, and the
  five commands of the suite passed on the branch. On the Linux host, `service status` of the executable built from
  the branch reported the human's installed, enabled and running systemd service, and `devices` found the screen.
  Everything on Windows is the human's to check and follows the merge. The status is `implemented`
