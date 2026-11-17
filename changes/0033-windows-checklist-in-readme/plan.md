# What the Windows checklist found, in the README

The record of the task states what is true when it is done. "The record" below is
`changes/0033-windows-checklist-in-readme/record.md`. "The controller section" is the section
`## The Corsair iCUE Commander Core XT` of `README.md`, "the sensors section" is its section
`## Windows: the sensors of the machine`, and "the Windows part" is the part `### Windows` of its section
`## Running as a service`. The bullets of the record's "What is true when the task is done" are traced as these
requirements:

- **R1** The paragraph of the controller section that begins "Other programs, such as iCUE, liquidctl and OpenLinkHub,
  are not kept out" keeps every fact it holds and says further: that with `commander_color`, iCUE running at the same
  time fights `run` over the controller, the LEDs changing between the colour of iCUE and that of `commander_color`
  every now and then, and `run` writing lines such as `controller setup failed` and `controller lost` to its log; that
  a user who lets sensordeck drive the controller with `commander_color` closes iCUE and keeps it from starting with
  Windows, with no menu or option of iCUE named; and that other monitoring programs that read the controller can make
  the LEDs flicker, as was seen with HWiNFO, while LibreHardwareMonitor was not seen to do so, with nothing said of a
  program that was not tried
- **R2** The Windows part says, in one sentence linked to `#the-corsair-icue-commander-core-xt`, that a task that runs
  `run` with `commander_color` needs iCUE closed and kept from starting with Windows, as that section says, and does
  not repeat its facts
- **R3** The sentence of the Windows part that begins "Each start of the task, at each logon and at each restart after
  an error" names the starts there are, at each logon, by `sensordeck service install` and by
  `schtasks /Run /TN sensordeck`, and each new try of a failed start that `run` makes every 10 seconds, and no longer
  says that the task is restarted after an error. The upload of the theme of the config, the values about 10 seconds
  after the start and the one reboot of the screen stay as they are
- **R4** The paragraph of the Windows part that begins "`schtasks /End /TN sensordeck`, the Task Manager, logging off
  and shutting down end `run` with no teardown" keeps what it says and says further that a `run` ended this way from
  outside is not started again until the next logon, `sensordeck service install` or `schtasks /Run /TN sensordeck`
- **R5** The README claims no restart of the task by the Task Scheduler after a `run` that ended with an error
- **R6** The part `### Linux` of `## Running as a service` stays as it is
- **R7** The sentences that say Windows has not been checked on a real machine say what was checked and what was not,
  each as far as its section concerns it, and no more: the last sentence of the second paragraph of the README, the last
  line of the sensors section, and the sentences that open the Windows part. Checked on a real Windows machine, on
  2026-11-16 and 2026-11-17: `devices`; `sensors`, whose readings were listed but not compared value by value; `run` in
  the foreground and as the startup task, with `minimal-dark-probes` showing live values; the lock between sensordeck
  processes; a replug of the screen while `run` runs; and the start of Windows after a boot from Linux. Not checked: the
  readings through LibreHardwareMonitor, and a comparison value by value of the natively read readings or of those of
  `nvidia-smi` with the Task Manager. The sentence that says the Windows part is written from the documentation of
  Windows may stay where it still holds
- **R8** Nothing else of the README, of `docs/` or of the code changes
- **R9** The five commands of the suite in `AGENTS.md` succeed

## Approach

The change is text in one file. No test reads the README, so there is no failing test to write first; the checks are
the diff, read against the record, and the suite. Each change is made inside the paragraph the record names, keeping
the line width of about 120 characters that the README uses, so that the diff shows only the paragraphs concerned.

Two readings of the record are fixed here, so that the implementer need not decide them:

- The Windows part opens with "On Windows sensordeck runs `run` as a task of the Task Scheduler. None of what follows
  has been checked on a real Windows machine yet. It is written from the documentation of Windows, and a wrong point is
  to be corrected." The first sentence says nothing of a check and stays. The second is the one R7 rewrites. The third
  still holds for what was not checked, such as the settings of the task for a battery and its `RestartOnFailure`, so
  it stays, narrowed to say that what was not checked is written from the documentation of Windows and that a wrong
  point is to be corrected
- "As far as its section concerns it" in R7 is applied so: the second paragraph of the README covers the transport, the
  sensors and the startup task, so it names every item of both lists; the sensors section names `sensors`, `run` with
  `minimal-dark-probes` showing live values, and both items of what was not checked; the Windows part names `run` as the
  startup task with `minimal-dark-probes` showing live values, the lock between sensordeck processes, a replug of the
  screen while `run` runs and the start of Windows after a boot from Linux, and names nothing as not checked, since
  neither item of that list concerns the task

The sentence of R2 goes at the end of the paragraph of the Windows part that begins "Running `sensordeck service
install` again stops the running `run`", which is where the part talks about `commander_color` and the starts of the
task, as a sentence of its own after the one on the reload of `run`. No other placement is needed, since R2 asks for one
sentence and no new paragraph.

## Files

- `README.md`: the second paragraph (the one that begins "On Windows the screen and the Commander Core XT work"), the
  last line of the sensors section, the paragraph of the controller section that begins "Other programs, such as iCUE",
  and the Windows part: its opening paragraph, the paragraph that begins "Running `sensordeck service install` again"
  and the paragraph that begins "`schtasks /End /TN sensordeck`"

No other file changes, apart from this plan and the record.

## Tasks

- [x] In the controller section, extend the paragraph that begins "Other programs, such as iCUE, liquidctl and
  OpenLinkHub, are not kept out", keeping its three sentences as they are, with: a sentence that with
  `commander_color`, iCUE running at the same time fights `run` over the controller, so that the LEDs change between
  the colour of iCUE and the colour of `commander_color` every now and then, and `run` writes lines such as
  `controller setup failed` and `controller lost` to its log; a sentence that to let sensordeck drive the controller
  with `commander_color`, close iCUE and keep it from starting with Windows, naming no menu or option of iCUE; and a
  sentence that other monitoring programs that read the controller can make the LEDs flicker, as was seen with
  HWiNFO, and that LibreHardwareMonitor running at the same time was not seen to do so. Name no other program as
  tried (R1)
- [x] In the Windows part, in the paragraph that begins "Running `sensordeck service install` again", replace "Each
  start of the task, at each logon and at each restart after an error, uploads the theme" with a sentence whose starts
  are: at each logon, by `sensordeck service install`, by `schtasks /Run /TN sensordeck`, and each new try of a failed
  start that `run` makes every 10 seconds, as the paragraph on a failed start says. Keep "uploads the theme of the
  config, so the values appear about 10 seconds after the start and the screen reboots once" as it is (R3, R5)
- [x] At the end of the same paragraph, after the sentence on the reload of `run`, add one sentence: a task that runs
  `run` with `commander_color` needs iCUE closed and kept from starting with Windows, as
  [The Corsair iCUE Commander Core XT](#the-corsair-icue-commander-core-xt) says. Repeat none of the facts of that
  section (R2)
- [x] In the paragraph that begins "`schtasks /End /TN sensordeck`, the Task Manager, logging off and shutting down end
  `run` with no teardown", keep its two sentences and add that a `run` ended this way from outside is not started
  again until the next logon, `sensordeck service install` or `schtasks /Run /TN sensordeck`. Fold this into the
  existing sentence on `schtasks /Run /TN sensordeck` or put it before it, so that the paragraph does not name that
  command twice to say the same thing (R4)
- [x] Search the README for "restart", "again" and "RestartOnFailure" in the Windows part and in the second paragraph,
  and see that no sentence says the Task Scheduler starts the task again after an error or after a kill. The paragraph
  on a failed start, which says `run` starts again every 10 seconds within its own process, stays (R5)
- [x] Rewrite the opening of the Windows part: keep "On Windows sensordeck runs `run` as a task of the Task
  Scheduler."; replace "None of what follows has been checked on a real Windows machine yet." with a sentence that it
  was checked on a real Windows machine on 2026-11-16 and 2026-11-17: `run` as the startup task with
  `minimal-dark-probes` showing live values, the lock between sensordeck processes, a replug of the screen while `run`
  runs and the start of Windows after a boot from Linux; and narrow "It is written from the documentation of Windows,
  and a wrong point is to be corrected." to say that what was not checked is written from the documentation of Windows
  and that a wrong point is to be corrected (R7)
- [x] Replace the last line of the sensors section, "The sensors of Windows have not yet been checked on a real Windows
  machine.", with sentences that say: on a real Windows machine, on 2026-11-16 and 2026-11-17, `sensors` listed its
  readings, which were not compared value by value, and `run` showed live values with `minimal-dark-probes`; the
  readings through LibreHardwareMonitor were not checked, and neither the natively read readings nor those of
  `nvidia-smi` were compared value by value with the Task Manager (R7)
- [x] Replace the last sentence of the second paragraph of the README, "The Windows transport, the sensors of Windows and
  the startup task have not yet been checked on a real Windows machine.", with sentences that say what was checked on a
  real Windows machine on 2026-11-16 and 2026-11-17, `devices`, `sensors` with its readings listed but not compared
  value by value, `run` in the foreground and as the startup task with `minimal-dark-probes` showing live values, the
  lock between sensordeck processes, a replug of the screen while `run` runs and the start of Windows after a boot from
  Linux, and what was not, the readings through LibreHardwareMonitor and a comparison value by value of the natively
  read readings or of those of `nvidia-smi` with the Task Manager. Leave the rest of the paragraph as it is (R7)
- [x] Run `grep -n "not yet been checked\|has been checked\|not been checked" README.md` and see that no sentence is
  left that says Windows, its sensors or its startup task has not been checked at all (R7)
- [x] Run the five commands of the suite in `AGENTS.md` and see each succeed: `go test ./...`, `go vet ./...`,
  `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` (R9)

## Verification

- [x] `git diff master --stat` lists only `README.md`, this plan and the record (R8)
- [x] `git diff master -- README.md` shows changes only in the paragraphs named under "Files", and none in the part
  `### Linux` of `## Running as a service` (R6, R8)
- [x] Each fact of the paragraph "Other programs, such as iCUE" and of the paragraph "`schtasks /End /TN sensordeck`"
  before the change is still in it after the change (R1, R4)
- [x] The five commands of the suite in `AGENTS.md` succeed (R9)
- [x] `npm run check-changes` passes on the committed branch

## Commit Points

- [x] `task-0033 Plan the README changes from the Windows checklist`: this plan and its log entry in the record
- [x] `task-0033 Write what the Windows checklist found into the README`: the change of `README.md`
- [x] `task-0033 Record the implementation`: the ticked plan and the log entries of the record
