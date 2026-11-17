---
kind: task
status: in-progress
timestamp: 2026-11-17T14:51:03Z
---

# Alarm colours in the built-in theme minimal-dark-probes

Unit `0028-number-alarm-colour` added the key `alarm` of a number widget, which shows the value in another colour past
a threshold. This task gives the five temperatures of the built-in theme `minimal-dark-probes` an alarm, at thresholds
the human chose, so that the theme shows them in red when they run hot. `minimal-dark` is not changed. "R*n* of unit
0028" below names a requirement of `changes/0028-number-alarm-colour/spec.md`, and "the record of unit 0023" is
`changes/0023-minimal-dark-probes-theme/record.md`, whose rules on the built-in themes and their tests this task
follows.

## What is true when the task is done

* In the `theme.yaml` of `minimal-dark-probes`, these five number widgets have the key `alarm`, written as the last key
  of the widget, as `alarm: {above: <n>, color: "#ff3030"}`, and every other line of the file is as it was:

  | Widget, by its label | Metric | `above` |
  |---|---|---|
  | CPU | `cpu.temp` | 90 |
  | GPU | `gpu.temp` | 85 |
  | DISK | `nvme.2.temp` | 70 |
  | GPU CABLE | `commander.1.temp` | 60 |
  | INTAKE | `commander.2.temp` | 35 |

  So the five lines read, in the order of the file:

  ```yaml
    - {type: number, font: big, metric: cpu.temp, x: 20, y: 44, alarm: {above: 90, color: "#ff3030"}}
    - {type: number, font: big, metric: gpu.temp, x: 176, y: 44, alarm: {above: 85, color: "#ff3030"}}
    - {type: number, font: small, metric: nvme.2.temp, x: 76, y: 233, alarm: {above: 70, color: "#ff3030"}}
    - {type: number, font: big, metric: commander.1.temp, x: 176, y: 176, alarm: {above: 60, color: "#ff3030"}}
    - {type: number, font: small, metric: commander.2.temp, x: 240, y: 233, alarm: {above: 35, color: "#ff3030"}}
  ```

* The alarm colour is `#ff3030` for all five. It is the colour of every example of an alarm in the README and in
  `docs/theme-folder.md`, and the project names no other default
* The layout is unchanged: no widget is added, removed or moved, no `x`, `y`, `digits`, `font` or other key changes,
  and no key `align` is added. The alignment of the five numbers is left as the theme file states it, with no key.
  What that means for how they are drawn is under "The alignment of the five numbers" below
* The background, the fonts and the folder are unchanged: the folder still holds `theme.yaml` and `background.png`
  only, and `background.png` is still the drawing of the shared generator
* The theme compiles with no error and no warning of the reader of `internal/themefile` when read back, with 43 widgets,
  the 38 of today and one alarm widget for each of the five, within the limit of 62. Its tags, as R13 of unit 0028
  gives them, make this tags file, byte for byte. It was written by the compiler on the host from a copy of the folder
  with the five lines above:

  ```yaml
  # Paste this block into the config file of sensordeck run.
  tags:
    commander.1.temp: {alarms: [{above: 60, normal: 1, alarm: 2}]}
    commander.2.temp: {alarms: [{above: 35, normal: 3, alarm: 4}]}
    cpu.temp: {alarms: [{above: 90, normal: 5, alarm: 6}]}
    cpu.util: 7
    gpu.mem.used: 8
    gpu.temp: {alarms: [{above: 85, normal: 9, alarm: 10}]}
    gpu.util: 11
    net.down: 12
    net.up: 13
    nvme.2.temp: {alarms: [{above: 70, normal: 14, alarm: 15}]}
    ram.used: 16
    ram.util: 17
  ```

  The five temperatures have an alarm pair and no plain tag, since no number without `alarm` and no bar shows them.
  The tags of the other metrics move, so a config file with a `tags` block copied from the old tags file of this theme
  no longer matches it. A config file with the key `theme` takes its tags from the compiler and needs no change
* `docs/minimal-dark-probes.png` is the preview that `sensordeck theme preview minimal-dark-probes
  docs/minimal-dark-probes.png` draws after the change. The preview draws the normal state, as R27 of unit 0028 says,
  so the five numbers show 42 in the colour of their font
* `minimal-dark` is unchanged: its `theme.yaml`, its `background.png`, its tags and `docs/minimal-dark.png`

### The alignment of the five numbers

A number without `align` is centred in its box, and the five numbers are so centred today. An alarm number is
left-aligned, and without `align` it is `left` (R4 and D5 of unit 0028: the method that hides the widget not shown
needs the left alignment, and `align: center` is an error). Giving the five numbers an alarm therefore moves each value
from the middle of its box to its left edge, on the screen as in the preview, with no key of the theme file changed.
The preview drawn on the host from the copy above shows CPU, GPU and GPU CABLE about half a digit further left, and
the INTAKE value next to its label, as the DISK value is. This follows from the request and cannot be avoided while the
five numbers have an alarm. Moving the boxes to place the values as they were is a change of the layout, which the
human decided not to make now, and is not part of this task.

### What the tests pin

As the record of unit 0023 set out, the tests go through the one table of the built-in themes in
`internal/builtinthemes` and the table of the preview images, and change only in the row of `minimal-dark-probes`:

* The `theme.yaml` of `minimal-dark-probes` is the text of today with the five lines above, verbatim, and that of
  `minimal-dark` is unchanged
* The compilation of each theme gives exactly its tags and its tags file, `minimal-dark-probes` the tags file above.
  The row of a theme states its plain tags and its alarm pairs, so that the test checks both and not only a list of
  metric names numbered from 1, which no longer fits a theme with an alarm number. The row of `minimal-dark` keeps its
  tags of today
* The test of the preview images in `docs_test.go` passes with the committed `docs/minimal-dark-probes.png`
* The end-to-end test in `e2e/builtin_theme_test.go` that compiles `minimal-dark-probes` by its name expects the tags
  file above

### The documents

* The section "Built-in themes" of `README.md` says that `minimal-dark-probes` shows the five temperatures in red past
  their thresholds, and gives each threshold: CPU above 90, GPU above 85, DISK above 70, GPU CABLE above 60 and INTAKE
  above 35, in °C, with a link to the part of the README on the key `alarm`. It says that a copy of the theme changes
  the thresholds in its `theme.yaml`. The alt texts of the two images stay true, since the preview shows the normal
  state with 42
* Nothing else of the README or of `docs/` changes. The examples of the key `alarm` already use `#ff3030`

### Outside the task

* No change to the compiler, the preview, `run`, the config file or the sensors
* No change of the layout or the alignment of `minimal-dark-probes`, and no change of `minimal-dark`
* No alarm on the other numbers of the theme, such as the loads, the memory or the network, and no `below` threshold
* No check on the device. Whether the screen shows the alarm colours of the theme is the human's to check

The five commands of the suite in `AGENTS.md` succeed.

## Log

- 2026-11-17T14:51:03Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the record of the task from the
  request of the orchestrating session, against the README, `docs/theme-folder.md`, the records of units 0023, 0027
  and 0028, `internal/builtinthemes` and its tests, `docs_test.go` and `e2e/builtin_theme_test.go`. The thresholds are
  the human's; the colour is `#ff3030`, the one of every example of the project. Compiled a copy of the folder with the
  five alarms with the installed `sensordeck` on the host, which gave the tags file of the record, and drew its preview,
  which showed the five numbers left-aligned as R4 of unit 0028 makes every alarm number. The Go toolchain of the
  project could not be run on the host, so the tests were not run. The status is `draft`
- 2026-11-17T15:07:35Z [Claude Code, anthropic/claude-opus-5-5] read the record and checked its preview against the preview of the
  theme today: the five numbers move to the left of their boxes and no box moves. The human saw the effect, including
  the value of INTAKE close to its label, and approved the task as it stands, with no change of position
- 2026-11-17T15:07:35Z [human] approved the record. The status is `approved`
- 2026-11-17T15:08:46Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote `plan.md` against the tree of the
  branch, with each task traced to the bullets of this record, numbered R1 to R8 in the plan. The tests change first,
  and the table of the built-in themes states the plain tags and the alarm pairs of each theme. The status stays
  `approved`
- 2026-11-17T15:30:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-17T15:31:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, changed the tests first and saw
  the tests of `minimal-dark-probes` fail on its `theme.yaml`, its tags and its tags file, and those of `minimal-dark`
  pass. Then put the five lines in `theme.yaml` (`git diff --stat`: 5 insertions and 5 deletions in that file only)
  and the tests of `internal/builtinthemes` and `e2e/` passed
- 2026-11-17T15:32:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, found necessary work that the
  plan did not name: the end-to-end test of the preview of a built-in theme and the test of the preview images in
  `docs_test.go` drew the expected preview with `themeimage.Preview` alone, which draws both widgets of an alarm number,
  and since unit 0028 `sensordeck theme preview` hides the widget of the alarm state. Both tests failed against the
  preview the command wrote, so both now draw the expected preview with `themeimage.PreviewWithValues` and the alarm
  slots of the compilation hidden. The plan was extended with this task and `docs_test.go` in its files. The spec of the
  task says `docs_test.go` stays as it is; its intent, a preview of the normal state that the test pins, is kept
- 2026-11-17T15:33:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, checked the compiled theme with
  `themefile.Parse` in a scratch test that was not committed: 44 entries and no warning, which are the background and
  the 43 widgets, since the theme of today holds 38 widgets and each of the five alarms adds one. The count of the
  background among the entries was inferred and not read from a field. The preview written by the command was looked
  at: the five numbers show 42, CPU, GPU and GPU CABLE are left of where they were, and INTAKE is next to its label
- 2026-11-17T15:34:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran on the tree of commit
  `1a65b57` `go test ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=windows GOARCH=amd64 go vet ./...`, and all five succeeded. Not checked: the screen on the device, which is the
  human's
- 2026-11-17T15:15:44Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, performed a full review of unit
  `0030-probes-theme-alarm-colours` at commit `e6c79a2e8cb95153f244de7de958f4f7eecf583f`, against merge base
  `26c31e09005267c8cad63842691fc4632a2c3081` with `origin/master`. Returned non-blocking finding P1, saved in
  `.reviews/0030-probes-theme-alarm-colours/review-round-1.md`. Independently ran the five required suite commands,
  an uncached `go test -count=1 ./...` including the end-to-end suite, and `npm run check-changes`; all passed.
  Physical-device rendering and Windows execution were not verified. The transient failing test run and scratch
  count check were not independently reconstructible from committed history. The status stays `in-progress`
- 2026-11-17T15:45:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, applied finding P1 of
  `review-round-1.md`, which the definition of the orchestrator accepts, and which is not a decision of the human: the
  declaration of the unchanged files in `plan.md` no longer names `docs_test.go`, which the plan lists as changed. The
  review also found that the earlier log entry says the spec of the task forbids changing `docs_test.go`, which the
  text of the record does not say: the record says the test passes with the committed image, and it does, so the
  earlier entry overstated it. Only `plan.md` changed, so no suite was rerun for this correction. The status stays
  `in-progress`

## Amendments
