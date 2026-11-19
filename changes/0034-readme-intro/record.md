---
kind: task
status: implemented
timestamp: 2026-11-19T22:07:33Z
---

# A short intro for the README

The opening of `README.md`, everything above `## Supported devices`, lists every subcommand and every caveat of
Windows in two long paragraphs, holds a dated account of what was checked on a real Windows machine, and says in its
headline that sensordeck works on Linux only. A reader who opens the repository cannot tell quickly what sensordeck
is, that it works on Linux and on Windows, or how to start. The README also does not say that the project was built
with AgentRite, while the records of that work are in `changes/`. This task rewrites the opening and the quick start
into a short form and moves the detail they hold into the sections where it belongs. It is a change of `README.md`
and of `docs/` only.

The human supplied a photo of the screen running `minimal-dark-probes`, with the tile GPU CABLE above its threshold
and so drawn in red.

## What is true when the task is done

* The headline says that sensordeck turns the screen into a hardware dashboard on Linux and on Windows
* Directly under the headline is the photo of the real screen, stored in `docs/`, with alternative text that says it is
  a photo of `minimal-dark-probes` with one temperature past its threshold in red
* Under the photo, a short list says what sensordeck does: the readings it shows, the probes of the Commander Core XT
  with colours past a threshold, themes as plain folders that can be previewed as PNG, running as a service on both
  systems, and a single executable with no driver or vendor software
* A table, short enough to read at a glance, shows what works on Linux and what on Windows: driving the screen, the
  sources of the sensors of the machine, the probes of the Commander Core XT, and how it runs as a service
* A section `## Built with AgentRite` says that the project was specified, planned, built and reviewed by agents
  through [AgentRite](https://agentrite.dev), and links `changes/`, where the records of each unit are
* The quick start is three numbered steps: install (with the udev rule on Linux and LibreHardwareMonitor on Windows,
  each a link to the section that tells how), `sensordeck theme use minimal-dark`, and `sensordeck run`. The paragraphs
  under the steps on what the tiles show and on a tile with no value stay
* The six sub-steps that set up LibreHardwareMonitor move, with their content unchanged, into
  `## Windows: the sensors of the machine`, and the sentence there that pointed at the quick start for them points
  nowhere else
* Every fact of the old opening that is not in a later section is kept in the README: that the Windows build needs no
  driver and no rule, that `--device` takes a device path on Windows as `devices` writes it, that iCUE or the vendor's
  software may hold the device and running either at the same time is not supported, and the account of what was and
  was not checked on a real Windows machine on 2026-11-16 and 2026-11-17. They go in one section on the two systems,
  after `## Supported devices`
* The two preview images stay in `## Built-in themes`, with the sentence that says they are previews and not photos.
  The copy of the `minimal-dark` preview in the opening goes, since that section shows it
* Every link of the README to `#quick-start` and to a moved part still lands where the text it stands in says
* No other section of the README changes, and no code changes

## Outside the task

* Rewording the rest of the README. Only the opening, the quick start and the moved parts are touched
* A preview of a theme with chosen values, so that a render could show a colour past a threshold

## Log

- 2026-11-19T22:07:33Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the record of the task from the
  request of the human and the restructure the human approved in the session: a new opening, a section on AgentRite
  linking `changes/`, the long Windows quick start moved into the Windows section, and the dated account of the checks
  on Windows moved out of the opening. The link of AgentRite and the photo were given by the human. The status is
  `draft`
- 2026-11-19T22:07:51Z [human] approved the record. The status is `approved`
- 2026-11-19T22:08:30Z [Claude Code, anthropic/claude-opus-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-19T22:11:51Z [Claude Code, anthropic/claude-opus-5-5] as the implementer, changed `README.md` and added
  `docs/screen-photo.png`, the photo of the human scaled to 800 pixels wide, in `3e0197b`. The opening is the headline
  on Linux and Windows, the photo, a list of five points, a table of the two systems and a link to the quick start;
  `## Built with AgentRite` follows it; the facts of the old opening that no later section held, and its account of
  the checks on Windows, are in a new section `## Linux and Windows` after `## Supported devices`; the quick start is
  three steps; the setup of LibreHardwareMonitor is the new part `### Setting up LibreHardwareMonitor` at the end of
  `## Windows: the sensors of the machine`. Deviation: in the moved setup, its opening sentence is reworded to stand
  as the start of a part rather than a step, and "sub-step 4" reads "step 4"; the six steps are otherwise unchanged.
  Verified by reading the diff: every removed sentence of the opening is either kept in `## Linux and Windows` or
  stated in `## Commands`, `## Built-in themes` or `## Showing sensor values on the screen`; the links to
  `#quick-start` and `#setting-up-librehardwaremonitor` land on those headings. On `3e0197b`, with the toolchain
  `go1.27.0`, `go test -count=1 ./...` passed in all 32 packages with tests, and `go vet ./...`, `gofmt -l .`,
  `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` reported nothing. A first run
  of `go test ./...`, made while the toolchain and modules were still downloading, failed in `internal/notices`, which
  passed alone and in the two full runs after it. Not done: a review. The status stays `in-progress`
- 2026-11-19T22:14:13Z [human] reviewed the change and accepted the new opening of the README as it is, with no
  finding, in place of a review by the reviewer
- 2026-11-19T22:14:13Z [Claude Code, anthropic/claude-opus-5-5] set the unit at the request of the human, with no open
  finding and no plan. The status is `implemented`
