---
kind: task
status: approved
timestamp: 2026-11-17T15:31:56Z
---

# Setting up LibreHardwareMonitor in the quick start

On Windows, `cpu.temp`, `cpu.ccd<n>.temp`, `cpu.clock`, `ram.<n>.temp` and `nvme.<n>.temp` come from the web server
of LibreHardwareMonitor, as unit `0018-windows-sensors` made them, and without it they are missing. The quick start of
`README.md` does not say so: it links to the section "Windows: the sensors of the machine", whose paragraph that
begins "To turn on its web server" holds the only instructions for setting LibreHardwareMonitor up, and those leave
out where to get it, how to keep it out of the way after each logon, and how to check that it works. A Windows user who
follows the quick start therefore gets a screen whose CPU temperature tile and disk tile of `minimal-dark` stay empty.

This task makes the quick start give the setting up of LibreHardwareMonitor as one step of ordered sub-steps, and
moves the paragraph that says it today into that step, so that the README says it in one place. It is a change of
`README.md` only. "The Windows section" below is the section `## Windows: the sensors of the machine` of the README.

## What LibreHardwareMonitor's options are called

These facts were read from the source of LibreHardwareMonitor at its release tag v0.9.6, the latest release, of
2026-02-14 (`LibreHardwareMonitor/UI/MainForm.Designer.cs`, `MainForm.cs`, `StartupManager.cs` and
`Resources/app.manifest`), and from its release page. LibreHardwareMonitor is under the MPL 2.0; it was read for the
names of its menus and options, and nothing of it is copied into the project.

* The release page `https://github.com/LibreHardwareMonitor/LibreHardwareMonitor/releases` offers for v0.9.6 the two
  archives `LibreHardwareMonitor.zip` and `LibreHardwareMonitor.NET.10.zip`
* The executable asks Windows for the rights of an administrator by its manifest (`requireAdministrator`), so Windows
  asks for the consent of an administrator at each start by hand
* It keeps its options in `LibreHardwareMonitor.config`, a file beside its executable, so an option set once holds
  at every later start, the web server included
* The menu Options holds `Start Minimized` (off by default), `Minimize To Tray` (on by default), `Minimize On Close`
  (off by default) and `Run On Windows Startup`. With `Start Minimized` and `Minimize To Tray` both on, it starts
  with its window hidden and its icon in the notification area
* `Run On Windows Startup`, chosen while it runs as an administrator, creates a task of the Task Scheduler that starts
  it at the logon of the user with the highest rights, so it starts after a reboot with no prompt. The task names the
  path of the executable
* The menu Options > Remote Web Server holds `Run` (off by default), `Interface / Port` and `Authentication`

## What is true when the task is done

* The section `## Quick start` of `README.md` has, as its second step, a step for Windows only that sets up
  LibreHardwareMonitor. The steps of today follow it, renumbered: the udev rule of Linux stays the first step, setting
  the theme becomes the third and starting `sensordeck run` the fourth. The sentence that opens the section, which says
  today that the first step is for Linux only, says that the first step is for Linux only and the second for Windows
  only
* The Windows step says why: without LibreHardwareMonitor sensordeck has no CPU temperature, no CPU clock and no
  temperatures of the NVMe drives and the memory modules on Windows. It names the metrics that appear only with it,
  `cpu.temp`, `cpu.ccd<n>.temp`, `cpu.clock`, `ram.<n>.temp` and `nvme.<n>.temp`, and says that the CPU temperature
  tile and the disk tile of `minimal-dark` show nothing without it. It says that sensordeck neither installs nor starts
  it, as the Windows section says today
* The Windows step gives these sub-steps, numbered and in this order:
  1. Download the archive `LibreHardwareMonitor.zip` of the latest release from its release page, linked, at version
     0.9.5 or newer, which the temperatures of the memory modules need. Extract it to a folder where it stays, since
     its options are kept in a file beside its executable and the startup of the fourth sub-step names its path
  2. Start `LibreHardwareMonitor.exe`. It runs as an administrator, which its driver needs, and Windows asks for that
     consent. The groups CPU, RAM and Storage Devices of its Options menu must be on, as they are by default
  3. Check Options > Remote Web Server > Run, which turns on the web server sensordeck reads
  4. Check Options > Run On Windows Startup, which starts it with Windows at each logon, and Options > Start Minimized,
     which with Options > Minimize To Tray, on by default, keeps its window out of the way in the notification area.
     Check Options > Minimize On Close as well, so that closing its window hides it instead of ending it. It must keep
     running for the readings to be there
  5. Open `http://localhost:8085/data.json` in a browser on the same machine. It shows a JSON document. When it does
     not load, the web server is not running, or its port, address or authentication was changed, and then the key
     `librehardwaremonitor_url` of the Windows section, linked, gives sensordeck the address
  6. Run `sensordeck.exe sensors`. It now lists `cpu.temp` and `cpu.clock`, and `nvme.<n>.temp`, `ram.<n>.temp` and
     `cpu.ccd<n>.temp` where the machine has such drives, modules with a thermal sensor and CCDs. When it writes the
     line `LibreHardwareMonitor cannot be read at <URL>` instead, the web server cannot be read, as the Windows section
     says, linked
* The words of the menus are written as LibreHardwareMonitor writes them, as listed above, with `>` between the menu
  and its item, as the README does today
* In the Windows section, the paragraph that begins "To turn on its web server" is replaced by a sentence that says how
  to set LibreHardwareMonitor up is the Windows step of the quick start, linked to `#quick-start`. Every fact of that
  paragraph is kept in the Windows step: starting it as an administrator, which its driver needs; checking Options >
  Remote Web Server > Run; keeping it running; Options > Run On Windows Startup starting it with Windows; and the groups
  CPU, RAM and Storage Devices of its menu on, as they are by default. This is a move, which the request asked for in
  place of saying the same twice, and no fact is removed
* Everything else of the Windows section stays as it is, the list of metrics that come from LibreHardwareMonitor, the
  key `librehardwaremonitor_url`, the sensors it reads, the version 0.9.5 for the memory modules and the lines that
  `sensors` and `run` write included. The Windows step links to it and does not restate its rules beyond what the
  sub-steps above need
* Nothing else of the README, of `docs/` or of the code changes. The sentences of the README that say the sensors of
  Windows have not been checked on a real Windows machine stay
* The five commands of the suite in `AGENTS.md` succeed

## Outside the task

* No change of the code, the messages that `sensors` and `run` write, or the default URL
* No installation or start of LibreHardwareMonitor by sensordeck, and no check of the order in which the two startup
  tasks start at a logon
* No instructions for the archive `LibreHardwareMonitor.NET.10.zip`, the nightly builds, or the authentication and the
  port of the web server beyond the link to `librehardwaremonitor_url`
* No check on a real Windows machine. Whether the steps read right against LibreHardwareMonitor on Windows is the
  human's to check

## Log

- 2026-11-17T15:31:56Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the record of the task from the
  request of the orchestrating session, against the README (its quick start, its section "Windows: the sensors of the
  machine", its installing section and the metrics of `minimal-dark`), the spec of unit 0018 and the source of
  LibreHardwareMonitor at v0.9.6 for the names of its options. The README already said how to turn on the web server
  and start it with Windows, in the Windows section; the record moves that paragraph into the new step of the quick
  start and leaves a link in its place, keeping every fact. No test reads the README text. The steps were not tried on
  Windows. The status is `draft`
- 2026-11-17T15:32:47Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, folded the one item of "Beyond the
  request", Options > Minimize On Close, into the fourth sub-step of the Windows step, since the orchestrating session
  took it, and removed the section "Beyond the request", which held nothing else. The status stays `draft`
- 2026-11-17T15:33:05Z [Claude Code, anthropic/claude-opus-5-5] read the record under the delegation in `AGENTS.md`: the move of the
  paragraph into the quick start keeps every fact and leaves a link, the option names are taken from the source of
  LibreHardwareMonitor, which is MPL and may be read, and nothing in it needs the device. The item of "Beyond the
  request" was taken into the record before this reading. The steps on Windows are the human's to try, and the task
  does not depend on them. The approval that follows is made under that delegation
- 2026-11-17T15:33:05Z [human] approved the record. The status is `approved`

## Amendments
