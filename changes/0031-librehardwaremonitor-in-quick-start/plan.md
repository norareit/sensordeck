# Setting up LibreHardwareMonitor in the quick start

The record of the task states what is true when it is done. "The record" below is
`changes/0031-librehardwaremonitor-in-quick-start/record.md`. "The Windows section" is the section
`## Windows: the sensors of the machine` of `README.md`, and "the Windows step" is the new second step of its section
`## Quick start`. The bullets of the record are traced as these requirements:

- **R1** The quick start has the Windows step as its second step. The udev rule of Linux stays first, setting the
  theme becomes the third step and `sensordeck run` the fourth. The opening sentence says that the first step is for
  Linux only and the second for Windows only
- **R2** The Windows step says why LibreHardwareMonitor is needed, names `cpu.temp`, `cpu.ccd<n>.temp`, `cpu.clock`,
  `ram.<n>.temp` and `nvme.<n>.temp`, says that the CPU temperature tile and the disk tile of `minimal-dark` show
  nothing without it, and says that sensordeck neither installs nor starts it
- **R3** The Windows step gives the six numbered sub-steps of the record, in its order and with its content
- **R4** The words of the menus are written as LibreHardwareMonitor writes them, with `>` between a menu and its item
- **R5** In the Windows section, the paragraph that begins "To turn on its web server" is replaced by a sentence that
  links to `#quick-start`, and every fact of that paragraph is in the Windows step
- **R6** Everything else of the Windows section stays as it is, and the Windows step links to it rather than restating
  its rules beyond what the sub-steps need
- **R7** Nothing else of the README, of `docs/` or of the code changes, and the sentences that say the sensors of
  Windows have not been checked on a real Windows machine stay
- **R8** The five commands of the suite in `AGENTS.md` succeed

## Approach

The change is text in one file. No test reads the README, so there is no failing test to write first; the checks are
the diff, read against the record, and the suite. The Windows step is a numbered item of the quick start with a short
paragraph of why and a nested numbered list of the six sub-steps, indented by three spaces under the item as the
existing items continue their lines, so that the Markdown renders it as one step. The links use the anchors the README
already uses: `#windows-the-sensors-of-the-machine` for the Windows section and `#quick-start` for the quick start. The
key `librehardwaremonitor_url` and the line `LibreHardwareMonitor cannot be read at <URL>` are not given anchors of
their own, so their links point at the Windows section.

## Files

- `README.md`: the section `## Quick start` and the paragraph of the Windows section that begins "To turn on its web
  server"

No other file changes, apart from this plan and the record.

## Tasks

- [ ] In the opening paragraph of `## Quick start`, change the end of its last sentence, "and the first step is for
  Linux only", to say that the first step is for Linux only and the second for Windows only. Leave the rest of the
  paragraph as it is (R1)
- [ ] After step 1, the udev rule, insert step 2, for Windows only, that sets up LibreHardwareMonitor. Its opening
  says that without it sensordeck has no CPU temperature, no CPU clock and no temperatures of the NVMe drives and the
  memory modules on Windows; that `cpu.temp`, `cpu.ccd<n>.temp`, `cpu.clock`, `ram.<n>.temp` and `nvme.<n>.temp`
  appear only with it; that the CPU temperature tile and the disk tile of `minimal-dark` show nothing without it; and
  that sensordeck neither installs nor starts it, with a link to the Windows section (R1, R2, R6)
- [ ] Under step 2, write the six numbered sub-steps of the record, in its order, each with the content the record
  gives it: (1) download `LibreHardwareMonitor.zip` of the latest release from
  `https://github.com/LibreHardwareMonitor/LibreHardwareMonitor/releases`, linked, at 0.9.5 or newer for the
  temperatures of the memory modules, and extract it to a folder where it stays, since its options are kept in a file
  beside its executable and the startup of sub-step 4 names its path; (2) start `LibreHardwareMonitor.exe`, which runs
  as an administrator, as its driver needs, with Windows asking for that consent, and with the groups CPU, RAM and
  Storage Devices of its Options menu on, as they are by default; (3) check Options > Remote Web Server > Run, which
  turns on the web server sensordeck reads; (4) check Options > Run On Windows Startup, which starts it with Windows at
  each logon, Options > Start Minimized, which with Options > Minimize To Tray, on by default, keeps its window in the
  notification area, and Options > Minimize On Close, so that closing the window hides it instead of ending it, and
  say that it must keep running for the readings to be there; (5) open `http://localhost:8085/data.json` in a browser
  on the same machine, which shows a JSON document, and when it does not load, the web server is not running or its
  port, address or authentication was changed, and the key `librehardwaremonitor_url` of the Windows section, linked,
  gives sensordeck the address; (6) run `sensordeck.exe sensors`, which now lists `cpu.temp` and `cpu.clock`, and
  `nvme.<n>.temp`, `ram.<n>.temp` and `cpu.ccd<n>.temp` where the machine has such drives, modules with a thermal
  sensor and CCDs, and when it writes the line `LibreHardwareMonitor cannot be read at <URL>` instead, the web server
  cannot be read, as the Windows section, linked, says (R3, R4, R6)
- [ ] Renumber the step that sets the theme to 3 and the step that starts `sensordeck run` to 4, and change nothing
  else of their text. Leave the paragraphs after the list as they are (R1, R7)
- [ ] In the Windows section, replace the paragraph that begins "To turn on its web server" with one sentence that
  says how to set LibreHardwareMonitor up is the Windows step of the [quick start](#quick-start). Leave every other
  paragraph, list and code block of the section as it is (R5, R6)
- [ ] Check the move: each fact of the removed paragraph is in the Windows step, namely starting it as an
  administrator, which its driver needs (sub-step 2); checking Options > Remote Web Server > Run (sub-step 3); keeping
  it running and Options > Run On Windows Startup starting it with Windows (sub-step 4); and the groups CPU, RAM and
  Storage Devices on, as they are by default (sub-step 2) (R5)
- [ ] Check every menu word against the list of the record: `Start Minimized`, `Minimize To Tray`,
  `Minimize On Close`, `Run On Windows Startup`, `Remote Web Server`, `Run`, written with `>` between menu and item
  as the README writes Options > Remote Web Server > Run today (R4)
- [ ] Run the five commands of the suite in `AGENTS.md` and see each succeed: `go test ./...`, `go vet ./...`,
  `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` (R8)

## Verification

- [ ] `git diff master --stat` lists only `README.md`, this plan and the record
- [ ] `git diff master -- README.md` shows changes only in the section `## Quick start` and in the one paragraph of
  the Windows section, and the sentence "The sensors of Windows have not yet been checked on a real Windows machine."
  is still there
- [ ] The five commands of the suite in `AGENTS.md` succeed
- [ ] `npm run check-changes` passes on the committed branch

## Commit Points

- [ ] `task-0031 Plan the LibreHardwareMonitor step of the quick start`: this plan and its log entry in the record
- [ ] `task-0031 Set up LibreHardwareMonitor in the quick start`: the change of `README.md`
- [ ] `task-0031 Record the implementation`: the ticked plan and the log entries of the record
