---
kind: feature
status: in-progress
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
