---
kind: feature
status: draft
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
