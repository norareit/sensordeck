---
kind: task
status: in-progress
timestamp: 2026-11-17T22:08:14Z
---

# What the Windows checklist found, in the README

On 2026-11-17 the human ran the checklist of the device checks on Windows with a build of `master` at unit
`0032-screen-theme-size-limit`. Three of its results are true of Windows and are not in `README.md`, and one of them
shows a sentence of the README to be false. None of them asks for a change of the code. This task writes them into
the README. It is a change of `README.md` only.

## What the checklist found

* **iCUE.** With iCUE and `sensordeck run` with `commander_color` running at the same time, the two programs fought
  over the Commander Core XT: the colour of iCUE differed from the colour of `commander_color`, and the LEDs changed
  between them every now and then. With iCUE running when `run` started, `run` wrote `controller setup failed`, then
  `controller lost` and then `controller back`, with details of the endpoint and the channel after the first two. With
  iCUE closed, everything worked with no such line
* **HWiNFO.** Running HWiNFO at the same time as `run` made the LEDs of the fans flicker. HWiNFO talks to the controller
  too. LibreHardwareMonitor, which the quick start sets up, running at the same time did not make them flicker
* **The startup task after a kill.** After `run` of the startup task was ended in the Task Manager, the Task Scheduler
  did not start it again. The task that `sensordeck service install` writes holds `RestartOnFailure` with an interval
  of one minute and 99 tries, which D3 of unit `0019-windows-startup-task` wrote as a second line of defence for a run
  that crashed, and whose notion of failure the documentation of Windows leaves open. A process ended from outside was
  not treated as such a failure. What starts `run` of the task is the logon, `sensordeck service install` and
  `schtasks /Run /TN sensordeck`; a background run whose start fails tries the start again by itself every 10 seconds
  (R26 of unit 0019), within the same process. The human decided that a start of `run` again after a kill is not
  needed, so the code stays as it is

## What is true when the task is done

* The paragraph of the section `## The Corsair iCUE Commander Core XT` that begins "Other programs, such as iCUE,
  liquidctl and OpenLinkHub, are not kept out" keeps every fact it holds today and says further:
    * that with `commander_color`, iCUE running at the same time fights `run` over the controller: the LEDs change
      between the colour of iCUE and the colour of `commander_color` every now and then, and `run` writes lines such as
      `controller setup failed` and `controller lost` to its log
    * that a user who lets sensordeck drive the controller with `commander_color` closes iCUE and keeps it from
      starting with Windows. The README names no menu or option of iCUE for this, since none was read
    * that other monitoring programs that read the controller can make the LEDs flicker, and that this was seen with
      HWiNFO. That LibreHardwareMonitor was not seen to do so may be said as well. Nothing is said of a program that
      was not tried
* The part `### Windows` of the section `## Running as a service` says, in one sentence linked to
  `#the-corsair-icue-commander-core-xt`, that a task that runs `run` with `commander_color` needs iCUE closed and kept
  from starting with Windows, as that section says. It does not repeat the facts of that section
* In the part `### Windows`, the sentence that begins "Each start of the task, at each logon and at each restart after an
  error" no longer says that the task is restarted after an error. It names the starts there are: at each logon, by
  `sensordeck service install` and by `schtasks /Run /TN sensordeck`, and each new try of a failed start that `run`
  makes every 10 seconds, as the paragraph on the failed start says. That each start uploads the theme of the config,
  that the values appear about 10 seconds after it and that the screen reboots once stay as they are. This corrects a
  false statement of the README: the Task Scheduler was seen not to start the task again after a kill, and is not
  known to start it again after an error
* The paragraph of the part `### Windows` that begins "`schtasks /End /TN sensordeck`, the Task Manager, logging off and
  shutting down end `run` with no teardown" keeps what it says and says further that a `run` ended this way from outside
  is not started again until the next logon, `sensordeck service install` or `schtasks /Run /TN sensordeck`
* The README claims no restart of the task by the Task Scheduler after a `run` that ended with an error, since whether
  `RestartOnFailure` makes one is not known
* The section on Linux of `## Running as a service`, where systemd does restart the service after an error, stays as it
  is
* The sentences of the README that say Windows has not been checked on a real machine say what was checked and what
  was not, and no more. They are the last sentence of the second paragraph of the README, which begins "The Windows
  transport, the sensors of Windows and the startup task have not yet been checked", the last line of
  `## Windows: the sensors of the machine`, and the two sentences that open the part `### Windows` of
  `## Running as a service`. What they say, each as far as its section concerns it:
    * checked on a real Windows machine, on 2026-11-16 and 2026-11-17: `devices`; `sensors`, whose readings were listed
      but not compared value by value; `run` in the foreground and as the startup task, with `minimal-dark-probes`
      showing live values; the lock between sensordeck processes; a replug of the screen while `run` runs; and the
      start of Windows after a boot from Linux
    * not checked: the readings through LibreHardwareMonitor, and a comparison value by value of the natively read
      readings or of those of `nvidia-smi` with the Task Manager
  The sentence of `### Windows` that says the part is written from the documentation of Windows and that a wrong point
  is to be corrected may stay where it still holds, or go where the check makes it false. This changes documentation
  by the decision of the orchestrating session, which took the item under the delegation in `AGENTS.md`
* Nothing else of the README, of `docs/` or of the code changes
* The five commands of the suite in `AGENTS.md` succeed

## Outside the task

* No change of the code, the task file that `install` writes or its `RestartOnFailure`. A start of `run` again after a
  kill was decided not to be needed
* No check of whether `RestartOnFailure` starts the task again after a `run` that ended with exit status 1
* No instructions for a program other than iCUE, HWiNFO and LibreHardwareMonitor, and no menu names of iCUE or HWiNFO
* No amendment of the units whose device checks on Windows the human answered with this checklist

## Log

- 2026-11-17T22:08:14Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, recorded the report of the human
  that every other step of the checklist on Windows of 2026-11-17 passed: the boot from Linux into Windows with no
  replug, the theme with the date and the time, the startup task with no flash of a window, the lock between sensordeck
  processes, and a replug while `run` runs. With them the device checks on Windows left open by units
  `0016-windows-hid-transport`, `0018-windows-sensors` for its natively read sensors, `0019-windows-startup-task`,
  `0022-upload-theme-at-every-start`, `0024-custom-metrics`, `0025-commander-led-port-configuration`,
  `0029-one-sensordeck-process-per-device` and `0032-screen-theme-size-limit` are answered. Those units are not amended.
  The status is `draft`
- 2026-11-17T22:08:14Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the record of the task from the
  results of the checklist that need the README only, against the README (its section on the Commander Core XT and the
  part on Windows of its section on running as a service), the spec of unit 0019 (D3, D4 and R26) and the task file in
  `internal/cli/taskfile.go`. The task corrects one false statement of the README, that the task on Windows is started
  again after an error, and removes no fact. The sentences that say Windows has not been checked are left to item 1 of
  "Beyond the request". No test reads the README text. The status stays `draft`
- 2026-11-17T22:09:06Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, folded item 1 of "Beyond the request",
  which the orchestrating session took, into the requirements: the sentences that say Windows has not been checked
  are to say what was checked on 2026-11-16 and 2026-11-17 and what was not, as the orchestrating session listed it,
  and no more. Removed the section "Beyond the request", which held nothing else. The status stays `draft`
- 2026-11-17T22:09:46Z [Claude Code, anthropic/claude-opus-5-5] read the record under the delegation in `AGENTS.md`: every change
  is to the README, rests on what the human saw on Windows on 2026-11-16 and 2026-11-17, and claims nothing that was
  not checked, and the sentence on the restart after an error is corrected to what the task does. The approval that
  follows is made under that delegation
- 2026-11-17T22:09:46Z [human] approved the record. The status is `approved`
- 2026-11-17T22:10:24Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote `plan.md` against the README on
  the unit's branch. It traces the bullets of the record as R1 to R9, and fixes two readings: the sentence of
  `### Windows` that says nothing of a check stays and the one that says the part is written from the documentation of
  Windows stays, narrowed to what was not checked; and each section names the checked and unchecked items that concern
  it, so that the part `### Windows` names nothing as not checked. The status stays `approved`
- 2026-11-17T22:30:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-17T22:31:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, changed `README.md` only, as the
  plan says (R1 to R7). Verified: `grep` finds no "not yet been checked" or "not been checked" left; no sentence of the
  README says the Task Scheduler restarts the task after an error; the part `### Linux` is untouched. On the final
  tree `go test ./...`, `go vet ./...`, `gofmt -l .` (no output), `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=windows GOARCH=amd64 go vet ./...` all succeeded. Not yet run: `npm run check-changes` on the committed branch,
  and a review
- 2026-11-17T22:12:43Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a full review of
  `0033-windows-checklist-in-readme` at `f06afc705572ce4172a1366cb7ccf484ece35fba` against the merge base with
  `origin/master`. Returned non-blocking finding P1 in `.reviews/0033-windows-checklist-in-readme/review-1.md`.
  Independently ran all five commands of the project suite and `npm run check-changes`; all succeeded. Windows and
  physical-device observations were checked against the approved record, not independently repeated on hardware.
- 2026-11-17T22:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accepted finding P1 of
  `review-1.md` by the definition of the orchestrator, not as a decision of the human. It is not blocking. Ticked the
  change-check verification, which the reviewer and the implementer ran successfully on the committed branch, and the
  three commit points. Deviation: the ticked plan and the implementation log entry went into the commit of the README
  change, `f06afc7`, and the third commit point is this commit, which holds the reconciliation of the plan and this
  entry. The status stays `in-progress`

## Amendments
