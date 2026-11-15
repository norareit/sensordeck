# Starting `run` at logon on Windows

## Context

Step 8 of the roadmap of the initial plan is the service mode: a systemd user service on Linux and a startup task on
Windows. Unit `0014-systemd-user-service` made the Linux half: `sensordeck service install`, `status` and `uninstall`
write a systemd user unit, enable and start it, report it and remove it. On every other system the four commands fail at
once with `the service is not supported on <system> yet; on Windows a startup task is planned`. Units
`0016-windows-hid-transport` and `0018-windows-sensors` made `run` work on Windows with the screen, the Commander Core
XT and the sensors of the machine. This unit is the Windows half of step 8, and the last part of step 7: the same three
commands on Windows manage a task of the Task Scheduler that starts `run` at the user's logon, hidden.

What the code gives today, which this unit builds on:

- `service install` on Linux takes `--config`, loads the config file as `run` does at its start, theme compilation
  included, writes the unit file with the resolved absolute path of the executable and `--config` with the absolute path
  of the config file (D3 of unit 0014), and makes the calls `daemon-reload`, `enable` and `restart`, so that running it
  again overwrites the unit and restarts the service. `status` and `uninstall` take the presence of the unit file as the
  mark of an installed service and make no call without it
- The calls of `systemctl --user`, the path of the executable and the name of the system come through three fields of
  the environment of the commands, which `Main` fills from the process and the unit tests fill with fakes. A call gives
  its standard output, its standard error and its error, and a failed call is reported with the arguments and the
  standard error of the call
- `run` reads its config file from `--config` or from `sensordeck\config.yaml` in `os.UserConfigDir()`, which on Windows
  is `%AppData%`. The state file of the uploads is `sensordeck\uploaded-themes.json` in `os.UserCacheDir()`, which on
  Windows is `%LocalAppData%`
- `run` ends with exit status 1 only at its start: a config that cannot be read or loaded, a theme that does not
  compile, a state file directory that cannot be found, sources that cannot be made, or no screen. Once running it
  survives a lost screen, a lost controller and an invalid reload, with lines on standard error, and does not end
  because of them. Every one of its lines goes to standard output or standard error
- `run` stops when the context of `notifyStop` ends, which `Main` makes with `signal.NotifyContext` for `os.Interrupt`
  and `syscall.SIGTERM`. A stop ends the rounds, the keeper of `commander_color` ends its session of the controller with
  the teardown, which returns the controller to hardware mode, `run` closes the screen, writes `stopped` and ends with
  exit status 0. An upload that has begun runs to its end first, about 10 seconds
- On Windows there is no SIGTERM. Go maps a Ctrl+C and a Ctrl+Break of the console to `os.Interrupt`, and the console
  events `CTRL_CLOSE_EVENT`, `CTRL_LOGOFF_EVENT` and `CTRL_SHUTDOWN_EVENT` to `syscall.SIGTERM`, but for these three the
  process is still terminated by Windows unless it exits first [os/signal]. Windows gives a process 5 seconds after
  `CTRL_CLOSE_EVENT`, and sends the logoff and shutdown events to services only: an interactive program is terminated at
  logoff without them [HandlerRoutine]. So today `run` in a console on Windows stops cleanly on Ctrl+C, stops within 5
  seconds or is cut off when its window is closed, and is killed with no teardown at logoff, by the Task Manager, by
  `taskkill /F` and by `schtasks /End`
- The CI workflow has a job on `windows-2025` that builds and vets everything and tests the packages written for
  Windows, named one by one; the test of the workflows checks that list. The layout test of the project allows only the
  seam files of the Windows packages to import `golang.org/x/sys/windows`, and only `hid.dll` and `kernel32.dll` to be
  loaded by name

The sources cited are listed under "Sources" at the end, by the names in square brackets.

"The task" below is the task of the Task Scheduler named `sensordeck`, in the root folder of the Task Scheduler. "The
task file" is the file of R3. "A call" is a run of `schtasks.exe`, found on the `PATH`, with the arguments that follow
it. "A background run" is a `run` started with `--background`, the flag of R18. "The stop event" is the named event of
R23. "The log" is the file of R20.

## Requirements

### The task definition

- **R1** On Windows, `sensordeck service install`, `status` and `uninstall` manage the task, as R6 to R17 say. By its
  definition in R2, the task runs as the user who ran `install`, with that user's interactive token and the least
  privilege, and starts at that user's logon. No command of this unit asks for elevation or for a password
- **R2** The task definition that `install` writes is exactly this XML, where `<user>`, `<executable>` and `<config>`
  are the values of R4 and R5, each written with `&` as `&amp;`, `<` as `&lt;` and `>` as `&gt;`:

  ```xml
  <?xml version="1.0" encoding="UTF-16"?>
  <!-- Written by sensordeck service install, which overwrites it when run again. -->
  <Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
    <RegistrationInfo>
      <Description>sensordeck: sensor values on a USB screen</Description>
    </RegistrationInfo>
    <Triggers>
      <LogonTrigger>
        <Enabled>true</Enabled>
        <UserId><user></UserId>
      </LogonTrigger>
    </Triggers>
    <Principals>
      <Principal id="Author">
        <UserId><user></UserId>
        <LogonType>InteractiveToken</LogonType>
        <RunLevel>LeastPrivilege</RunLevel>
      </Principal>
    </Principals>
    <Settings>
      <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
      <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
      <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
      <AllowStartOnDemand>true</AllowStartOnDemand>
      <Enabled>true</Enabled>
      <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
      <RestartOnFailure>
        <Interval>PT1M</Interval>
        <Count>99</Count>
      </RestartOnFailure>
    </Settings>
    <Actions Context="Author">
      <Exec>
        <Command><executable></Command>
        <Arguments>run --config "<config>" --background</Arguments>
      </Exec>
    </Actions>
  </Task>
  ```

  Each line ends in CR LF, the last one included, and the file holds no other line. It is encoded in UTF-16
  little-endian and begins with the byte order mark `FF FE`
- **R3** The task file is `sensordeck\startup-task.xml` in the directory that `os.UserCacheDir()` gives, which on
  Windows is `%LocalAppData%`. When `os.UserCacheDir()` fails, the three commands fail with exit status 1 and an error
  that says the cache directory cannot be found, and make no call
- **R4** `<user>` is the security identifier of the user who runs `install`, as a string such as
  `S-1-5-21-1004336348-1177238915-682003330-1001`, as `os/user.Current` gives it on Windows. When it cannot be found,
  `install` fails with exit status 1 and an error that says the user cannot be found, and writes nothing
- **R5** `<executable>` is the path of the running executable, found as R3 of unit 0014 says, and `<config>` is the
  config file found as R4 of unit 0014 says: the path given with `--config` made absolute and cleaned, or without it
  `sensordeck\config.yaml` in `os.UserConfigDir()`, its links kept. A path that holds `"`, `%`, a line feed, a carriage
  return or another control character below 0x20 is refused by `install` with exit status 1 and an error that names the
  path, and nothing is written

### `sensordeck service install` on Windows

- **R6** `install` takes no argument and the flag `--config <path>`, opens no device and reads no sensor. Before it
  writes anything or makes a call, it checks the config file as R7 of unit 0014 says, with the same errors, and with a
  missing file the same hint that `sensordeck theme use <theme>` creates it
- **R7** After the check, when a background run is running by R24, `install` stops it as R16 says. It then creates the
  directory of the task file when it is missing, writes the task file of R2, replacing a file that exists at that path
  whatever it holds, writes `wrote <task file>` to standard output, and makes these calls in this order, each only after
  the one before succeeded:
    1. `/Create /TN sensordeck /XML <task file> /F`
    2. `/Run /TN sensordeck`
- **R8** After the last call succeeded, `install` writes `the startup task sensordeck is created and started` to
  standard output and ends with exit status 0
- **R9** When the task file cannot be written, `install` fails with exit status 1 and an error that names the task file
  and the cause, and makes no call. When a call fails, `install` makes no further call and fails with exit status 1 and
  an error that names the call, as `schtasks <arguments>`, and gives the text the call wrote to its standard error, with
  the space around it trimmed, or its error when it wrote none. The task file it wrote is left in place. When
  `schtasks.exe` cannot be started, the call fails in the same way, with the error of the start as the cause

### `sensordeck service uninstall` on Windows

- **R10** `uninstall` takes no argument. When the task file does not exist, it writes `the startup task sensordeck is
  not installed` to standard output, makes no call, stops nothing, removes nothing and ends with exit status 0
- **R11** When the task file exists, `uninstall` stops a background run that is running by R24 as R16 says, then makes
  the call `/Delete /TN sensordeck /F`, then removes the task file, writes `the startup task sensordeck is stopped and
  removed` to standard output and ends with exit status 0
- **R12** When the stop fails by R16 or the call fails, `uninstall` fails as R9 says for a call, makes no further call
  and removes nothing. When the removal of the task file fails, it fails with exit status 1 and an error that names the
  task file and the cause, and the task stays deleted

### `sensordeck service status` on Windows

- **R13** `status` takes no argument. When the task file does not exist, it writes the one line `installed: no` to
  standard output, makes no call and ends with exit status 0
- **R14** When the task file exists, `status` makes the one call `/Query /TN sensordeck /XML`, checks the stop event by
  R24, writes these four lines to standard output and ends with exit status 0:

  ```
  installed: yes, <task file>
  enabled: <enabled>
  active: <active>
  log: <log>
  ```

  `<enabled>` is `disabled` when the element `Enabled` that is a child of `Settings` in the output of the call holds
  `false`, `enabled` when it holds `true` or the element is absent, which is the default of the schema, and `unknown`
  when the output cannot be read as the XML of a task. The output is read as UTF-16 little-endian when it begins with
  the byte order mark `FF FE`, and as UTF-8 otherwise, whatever encoding its declaration names. `<active>` is `running`
  when R24 finds a background run, `stopped` when it finds none, and `unknown` when the check fails. `<log>` is the path
  of the log by R20
- **R15** When the call fails, `status` fails as R9 says for a call

### Stopping a background run

- **R16** To stop a background run, `install` and `uninstall` signal the stop event and wait until R24 no longer finds a
  background run, checking at least every 200 milliseconds, for at most 30 seconds. When it has not ended by then, they
  make the call `/End /TN sensordeck` and write `sensordeck run did not stop within 30 seconds and was ended with
  schtasks /End` to standard output. When that call fails, the stop fails as R9 says for a call. When the stop event
  cannot be signalled, the stop fails with exit status 1 and an error that names the stop event and the cause
- **R17** A background run that is stopped by the stop event stops as `run` stops on SIGTERM on Linux: the keeper ends
  its session of the controller with the teardown, which returns the Commander to hardware mode, the screen keeps its
  last values and its theme, `stopped` is written to the log, and it ends with exit status 0. An upload that has begun
  runs to its end first

### The background run

- **R18** `run` takes the flag `--background`, which is hidden from the help, on every system. Without it, `run` is not
  changed by this unit. The task is the only thing that passes it
- **R19** A background run first opens the log by R20 and R21. When the log cannot be opened, it ends with exit status 1
  and the error of the opening on standard error. Every line that `run` would write to standard output or standard error
  is then written to the log and nowhere else, the error with which a start fails included
- **R20** The log is `sensordeck\run.log` in the directory that `os.UserCacheDir()` gives, which on Windows is
  `%LocalAppData%\sensordeck\run.log` and on Linux `~/.cache/sensordeck/run.log` or
  `$XDG_CACHE_HOME/sensordeck/run.log`. The directory is created when it is missing, and the lines are appended to the
  file that exists
- **R21** Each line of the log begins with the local time at which it was written, as `2006-01-02T15:04:05-07:00`, and a
  space, followed by the line as `run` writes it in a terminal. A line is written to the file whole, in one write
- **R22** The log and its one older file `run.log.1`, in the same directory, together hold at most 2 MiB plus two lines.
  Before a line is written that would make `run.log` larger than 1 MiB, `run.log` is closed and renamed to `run.log.1`,
  replacing it, and a new empty `run.log` is opened. A log that is larger than 1 MiB when it is opened is renamed in the
  same way first. When the renaming fails, `run.log` is opened again emptied instead, and when that fails too, the line
  is not written and the next line tries again
- **R23** On Windows, a background run then creates the stop event, a manual-reset event named
  `Local\sensordeck-background-run`, and holds it until it ends. When an event of that name exists already, the run
  writes `another background run of sensordeck is running` to the log and ends with exit status 1. The stop event being
  signalled ends the run as R17 says. `os.Interrupt` and `syscall.SIGTERM` stop it too, as without the flag
- **R24** "R24 finds a background run" means that an event named `Local\sensordeck-background-run` can be opened in the
  session of the user who runs the command. The command closes its handle at once after the check, and holds none while
  it waits by R16
- **R25** On Windows, after R23, a background run detaches from its console with `FreeConsole`, so that the console
  window that Windows made for it at its start closes. A failure of the detach is ignored
- **R26** When the start of a background run fails, with any error with which `run` ends with exit status 1 today, the
  run writes `sensordeck: <error>` and then `starting again in 10s` to the log, waits 10 seconds, and starts again, the
  config file read anew, for as long as it is not stopped. A stop while it waits ends it at once with `stopped` and exit
  status 0. A stop by R17 at any time also ends it with exit status 0

### Other systems

- **R27** The command `service` and its three subcommands are present on every system, with help texts that name a
  systemd user service on Linux and a startup task on Windows. On a system other than Linux and Windows each of them
  fails at once with exit status 1 and the error `the service is not supported on <system>`, where `<system>` is the
  name `runtime.GOOS` gives, and reads no file, writes no file and makes no call. On Linux they do what unit 0014 says,
  with no change but the help texts

### The documents

- **R28** The section of `README.md` on running as a service covers Linux as it does today and Windows in a part of its
  own, which says:
    - that `sensordeck service install` creates the task `sensordeck` of the Task Scheduler, which runs `sensordeck run`
      at each logon of the user, hidden, with no administrator rights, and starts it at once; with `--config`, what it
      checks as on Linux, and that the task names the absolute path of the executable and of the config file
    - where the task file and the log are, that the log holds the lines of `run` with their time, and that it is kept to
      two files of at most 1 MiB each, `run.log` and `run.log.1`
    - that running `install` again stops the running `run`, replaces the task and starts it again, which is the way to
      apply a moved or replaced executable, another `--config`, or a change of `interval`, `commander_interval` or
      `commander_color`, and that `sensordeck theme use` and edits of the config file reach the running `run` within a
      round
    - `sensordeck service status`, its four lines, and that the task is seen in the Task Scheduler too
    - that a background run whose start fails, as when no screen is found, writes the error to the log and starts again
      every 10 seconds, with no limit
    - that `sensordeck service uninstall` stops the running `run` so that the Commander Core XT is back in hardware mode
      and the screen keeps its last values, then deletes the task, and that it ends `run` with `schtasks /End` when it
      has not stopped within 30 seconds
    - that `schtasks /End /TN sensordeck`, the Task Manager, logging off and shutting down end `run` with no teardown,
      so that the Commander Core XT may stay in the colour of `commander_color` until `run` starts again, and that
      `schtasks /Run /TN sensordeck` starts the task again
    - that a console window may flash at the logon when the task starts, before `run` detaches from it
    - that running `sensordeck run` or another command that writes to the screen by hand while the task runs is not
      supported, as on Linux
    - that the task of one user is named `sensordeck` for the whole machine, so that a second user of the same machine
      cannot install one too
    - that this has not yet been checked on a real Windows machine
- **R29** The README no longer says that the service is not available on Windows yet, or that the startup task on
  Windows is still to come: the introduction and the list of features name the startup task on Windows, and the lines of
  `service` under "Commands" name both systems
- **R30** The README describes the task as R2 writes it and what the documentation cited says, and claims nothing as
  seen on Windows

### CI on Windows

- **R31** The calls into Windows that this unit adds, the stop event of R23 and R24 and the detach of R25, are behind a
  seam in a package of their own that holds nothing else, and follow the layout rules of the project: only its seam file
  imports `golang.org/x/sys/windows`, and it loads no library by name but `kernel32.dll` from the system directory. The
  layout test allows that seam file and its test
- **R32** A test of that package limited to Windows creates the stop event under a name unique to the test, finds it as
  R24 says from a second opening, signals it and sees a waiter on it end, closes it, and then no longer finds it. It
  does not call `FreeConsole`
- **R33** The Windows job of the CI workflow tests that package too, named in its list beside the packages it names
  today, and the test of the workflows checks the new list

### Tests

- **R34** Unit tests that run on Linux in `go test ./...` cover, with the system set to `windows` by the fake of the
  system, and with the calls of `schtasks.exe`, the user, the executable, the stop event and the time behind fakes, the
  cache and config directories in temporary directories:
    - R2 to R5: the task file byte for byte, the byte order mark and the CR LF included, with and without `--config`, a
      relative `--config`, paths with a space, `&`, `<` and `>`, a path with `"`, one with `%` and one with a line feed
      refused, a user that cannot be found, and a cache directory that cannot be found
    - R6: a missing config file and its hint, a file that does not parse and a theme that does not compile, each with no
      file written and no call made
    - R7 to R9: the two calls in order after the file is written, a task file that exists replaced, a running background
      run stopped before the file is written, a failure of each call with the calls after it not made, an error text of
      the call and an empty one, and `schtasks.exe` that cannot be started
    - R10 to R12: no task file, the order of the stop, the call and the removal, and a failure of each step
    - R13 to R15: no task file, the four lines with the task enabled, disabled and with no `Enabled` element, an output
      in UTF-16 with its byte order mark and one in UTF-8, an output that is not XML, `running`, `stopped` and
      `unknown`, and a failure of the call
    - R16: a run that ends within the time with no `/End`, one that does not end with `/End` and its line, a failure of
      `/End`, and a stop event that cannot be signalled
    - R19 to R22: the time at the start of each line, a line written whole, the rotation at 1 MiB, a log larger than 1
      MiB at its opening, a failed renaming that empties the log, and a log that cannot be opened
    - R23 and R26: the stop by the fake stop event with the teardown of the controller, an existing stop event, a start
      that fails and is tried again after 10 seconds with the two lines, and a stop during the wait
    - R27: each of the four commands with the system set to `darwin`, with no file read or written and no call made
- **R35** The end-to-end tests run the built executable on Linux with `run --background`, `XDG_CACHE_HOME` set to a
  temporary directory and the fake screen, and check that the lines of `run` are in the log with their time and not on
  standard output or standard error, that a run with no screen writes the error and `starting again in 10s` to the log
  and keeps running, and that SIGTERM ends both with `stopped` in the log and exit status 0
- **R36** The five commands of the suite in `AGENTS.md` succeed

## Unverified until the human runs it

Each of these is open until the human checks it on Windows 10 and Windows 11 with the devices. The unit does not depend
on it, and a wrong one is corrected by a later unit.

- That a standard user, not an administrator, can create, run, end, query and delete the task with `schtasks.exe` and
  the XML of R2, which the documentation says for a task that runs as its creator with an interactive token [Security
  contexts], and that a `LogonTrigger` with the user's own `UserId` needs no administrator rights, which the
  documentation does not say
- That `schtasks /Create /XML` accepts the file of R2 in UTF-16 with its byte order mark, a `UserId` given as a security
  identifier, a `LogonTrigger` with no `StartBoundary`, an unquoted `Command` that holds a space, and a `Count` of 99
- That `schtasks /Query /XML` writes the task as UTF-16 with a byte order mark or as UTF-8, as R14 reads it, and what it
  writes for a task disabled in the Task Scheduler
- That the task starts at the logon and runs `run` with the arguments of R2, that the Task Scheduler expands no `%` in a
  path (R5 refuses them because it may), and that `run` opens the screen and the Commander from the task's token
- How long the console window shows at the logon before `FreeConsole` closes it, on Windows 10 and on Windows 11 with
  the Windows Terminal as the default terminal, and whether a window of the Windows Terminal stays open after the detach
- That `schtasks /End` ends the process with no console event, which the setting `AllowHardTerminate` suggests
  [settingsType] and the page of `/End` does not say [schtasks end], and that the task shows as running while a
  background run runs and as ready after it ended
- Whether `RestartOnFailure` restarts the task after `run` ended with an exit status other than 0, or only when the task
  could not be started. R26 does not depend on it
- What the Commander Core XT does when `run` is killed with no teardown at logoff or shutdown: whether it keeps the
  colour of `commander_color`, and whether it returns to hardware mode at the next boot by itself
- That the text `schtasks.exe` writes to standard error, which Windows localizes in the code page of the console, is
  readable in the error of R9
- That the job of R33 passes on GitHub. The loop does not push, so the job first runs when the human pushes

## Decisions

- **D1 A task of the Task Scheduler, created with `schtasks.exe`.** The three mechanisms that start a program at a
  user's logon with no administrator rights are a task, a value under the `Run` key of `HKEY_CURRENT_USER`, and a
  shortcut in the user's Startup folder. All three run as the user, in the user's session, where the screen and the
  Commander are opened. The task is chosen because it is the only one with the two other properties the request weighs:
  the Task Scheduler can start the task on demand (`/Run`), end it (`/End`) and report it (`/Query`), and it can restart
  a task that failed (`RestartOnFailure`) [RestartOnFailure]. A `Run` value and a shortcut start the program once at the
  logon, are neither restarted nor stopped by Windows, and give no state to report; Windows also delays them after the
  logon and lets the user turn them off under the startup apps of the settings, where `status` cannot see it. A shortcut
  needs the shell's COM interface to be written, which a registry value and a task do not. The task is created from an
  XML definition because the switches of `schtasks /Create` cannot express the settings of R2, such as the batteries,
  the time limit and the restart [schtasks create], and because a file that `install` writes gives `status` and
  `uninstall` their mark of an installed task, as the unit file does on Linux. The Task Scheduler's COM interface was
  rejected for the reason that unit 0014 rejected a D-Bus client: three calls of a program do the work, behind the same
  kind of seam
- **D2 No administrator rights by the principal.** The Microsoft documentation says that a user who is not an
  administrator registers a task with no password when the task runs in that user's own security context with the
  interactive logon type, and that the creator of a task may read, update, delete and run it [Security contexts].
  `InteractiveToken` also runs the task in the user's session, which is where the HID devices are opened and where the
  `Local\` namespace of R23 is the user's. `LeastPrivilege` is the run level a process with no elevation may register.
  The other logon types were rejected: a stored password is a secret sensordeck would have to ask for, and S4U needs the
  privilege to log on as a batch job, which a standard user does not have by default [Security contexts], and runs
  outside the session. The trigger names the user by the security identifier, which does not change with a renamed
  account and holds no character to escape
- **D3 The settings of R2.** `ExecutionTimeLimit` `PT0S` runs the task with no time limit [ExecutionTimeLimit], where
  the default is 72 hours. `DisallowStartIfOnBatteries` and `StopIfGoingOnBatteries` are `true` by default
  [settingsType], which would stop the screen on a laptop that is unplugged, so both are `false`.
  `MultipleInstancesPolicy` `IgnoreNew` is the default and is written so that a second start never runs a second `run`
  beside the first. `RestartOnFailure` takes an `Interval` of at least one minute [Interval] and a `Count` that is a
  byte; one minute and 99 tries are written as a second line of defence for a run that crashed. Whether the Task
  Scheduler counts an exit status other than 0 as a failure is not documented, so nothing depends on it: R26 makes the
  restart that matters. The priority is left at the default of the Task Scheduler, and no `WorkingDirectory` is set,
  because every path that `run` reads is absolute
- **D4 The retry is made by `run` itself.** On Linux, systemd starts `run` again 10 seconds after a start that failed,
  with no limit (D6 of unit 0014), and that is what keeps a screen that is unplugged at the logon, or a config being
  edited, from leaving the screen dead. The Task Scheduler gives no documented equivalent: `RestartOnFailure` has a
  minimum of one minute, a limit of tries, and an undocumented notion of failure. `run --background` therefore tries its
  start again every 10 seconds, the wait of systemd, so the behaviour is the same on both systems and is tested on Linux
  with fakes. The retry is limited to the background run, so that `run` in a terminal still fails at once with its
  error, as today. A repeating trigger every minute was rejected: it is slower, and it would start `run` again after a
  stop that was meant
- **D5 The window: `run` detaches itself.** A console program that is started with no console, as by the Task Scheduler,
  gets a new console and its window [Console allocation policy]. The `Hidden` setting of a task hides it in the Task
  Scheduler's list only. Four ways round it were weighed:
    - An executable of the GUI subsystem has no console, but it is the command line too, which a shell would no longer
      wait for and whose output would be lost, so the request excludes it. A second, GUI-subsystem launcher in the
      archive is item 2 of "Beyond the request"
    - `conhost.exe --headless` runs a program with a console that has no window, but the flag is not documented by
      Microsoft, so it may change with an update, and security products such as Elastic Security report its use as a
      sign of an attack [Elastic], so an antivirus may flag or block the task. Rejected
    - The manifest setting `consoleAllocationPolicy` `detached` makes Windows give no console to a console program
      started outside one, which is what is needed, but only from Windows 11 24H2 [Console allocation policy], and it
      needs a manifest resource in the build. Item 1 of "Beyond the request"
    - `FreeConsole`, called by `run` when it has the flag that only the task passes, is a documented call of the console
      API, and the console, and with it its window, is expected to close once no process is attached to it
  `FreeConsole` is chosen. Its cost is that the window shows for the moment between the start of the process and the
  call, which the README says, and that the Windows Terminal as the default terminal of Windows 11 may behave otherwise,
  which is unverified. The detach comes after the log is open and the stop event is held, so that a log that cannot be
  opened is still reported on the console. With no console the background run gets no console event at all, which
  changes nothing: logoff and shutdown kill an interactive program anyway [HandlerRoutine]
- **D6 The log is a file with two generations.** With no journal and no console, the lines of `run` would be lost. The
  file is in `%LocalAppData%\sensordeck`, beside the state file of the uploads and the task file, because it belongs to
  the machine and is not to roam with the profile. Each line takes the time, which the journal adds on Linux and a file
  does not. `run` writes few lines, one for each outage and each upload, so 1 MiB holds a long history; the bound of two
  files of 1 MiB keeps it from growing on a machine whose screen stays unplugged for months, where a line is written
  every 10 seconds. A rename to `run.log.1` keeps the last lines readable after a rotation; the file is closed before it
  is renamed, so that the rename does not depend on the share mode the file was opened with. The Windows event log was
  rejected: writing to it needs an event source that only an administrator can register. `--background` does the same on
  Linux, where nothing passes it, so that the end-to-end tests can run the real executable with it
- **D7 The stop is a named event.** A clean stop needs the background run to end its own rounds: the teardown of the
  Commander and the end of an upload in progress happen inside `run`. Windows has no signal another process can send for
  it: `schtasks /End` and `taskkill /F` terminate the process, `taskkill` without `/F` sends a close message to windows
  that a console program does not have, and `GenerateConsoleCtrlEvent` reaches only processes that share the console of
  the caller, which a detached background run does not. A named event is a documented means for two processes of one
  session to signal each other, and needs one creation in `run` and one opening in the command. The same event answers
  whether a background run is running, because it exists only while a process holds it, so `status` needs no text of
  `schtasks /Query`, which Windows localizes. The wait of 30 seconds is longer than an upload and a teardown, and `/End`
  after it keeps `uninstall` from hanging on a `run` that does not answer. `install` stops a running background run
  first, because `/Create /F` replaces the definition but leaves the running process, and `/Run` with `IgnoreNew` would
  then start nothing, so the old executable and config would stay in use
- **D8 `status` reads the definition, not the text.** The text of `schtasks /Query` is localized, so a word such as
  `Running` or `Disabled` cannot be read on every machine. `/Query /XML` gives the definition, whose element names are
  not localized, and `Enabled` there is what the Task Scheduler shows as enabled or disabled. The running state comes
  from the stop event by D7. The fourth line, `log`, is how `status` points the user to the log, which on Linux is the
  journal that the README names
- **D9 The quoting of the paths.** The executable is the `Command` of the action, a path on its own. The config file is
  in the `Arguments`, which the Go runtime of `run` splits by the rules of the Windows C runtime, so it is written in
  double quotes. A path on Windows cannot hold `"`, and a path that is a file does not end in `\`, so no escape is
  needed inside the quotes, and a path with `"` is refused rather than escaped. A `%` is refused because the Task
  Scheduler may expand environment variables in the action, and there is no escape for it; such a path is rare on
  Windows and the refusal names it. The XML escapes are those of the XML specification for text
- **D10 One task per machine, named `sensordeck`.** Task names are shared by the users of a machine. A name with the
  user in it was rejected for a case nobody has, two users of one machine with the screen; a second user's `install`
  fails with the error of `schtasks`, which says access is denied, and is seen at once
- **D11 The seam.** The calls of `schtasks.exe` come through a field of the environment beside the one of `systemctl`,
  with the same signature, so that a call is faked, recorded and failed in the unit tests as on Linux. The stop event,
  the user, the detach and the time come through fields of their own. The real stop event and the detach are in a
  package limited to the Windows calls, as the transport and the sensors of Windows are, so that the layout rules hold
  and its test runs on the Windows job of CI
- **D12 The message of other systems.** The text of R27 drops "yet; on Windows a startup task is planned", because
  Windows is no longer planned but done, and no other system is planned
- **D13 Unit 0014 stays as it is.** Its requirements hold for Linux unchanged. R17 of unit 0014 is replaced by R27 of
  this unit, which is a new behaviour on a new system and not a correction of the old record, so this unit does not
  supersede it
- **D14 The threat model.** The task file, the log, the config file and the executable belong to the user who runs
  `install`, and are trusted as theirs. Any process of the user's session can signal the stop event and stop the
  background run; that is the user's own process, with the same rights as the user to end it. The unit does not defend
  against another user who can write these files, or against an administrator
- **D15 No check on Windows in this unit.** Everything that needs Windows is the human's, as "Unverified until the human
  runs it" lists. The loop does not wait for it, the unit is complete without it, and a finding of it is a new unit

## Non-goals

- A Windows service run by the service control manager, which needs administrator rights and runs outside the session
- A clean stop at logoff or shutdown on Windows, which needs a hidden window for the messages of the session's end (item
  3 of "Beyond the request")
- Commands to stop and start the task without removing it (item 4 of "Beyond the request")
- A task per user of the machine, a task in a folder of the Task Scheduler, or several tasks for several screens
- A `Run` value of the registry or a shortcut in the Startup folder, also as a fallback
- Passing `--device` or `--trace` to the task, as on Linux (D15 of unit 0014)
- Running the tests of `internal/cli` on the Windows job of CI
- The log on Linux beyond what `--background` gives: the service of unit 0014 keeps writing to the journal

## Beyond the request

1. The manifest setting `consoleAllocationPolicy` `detached` in the Windows executable, so that on Windows 11 24H2 and
   later no console window shows at all at the logon. Cost: a manifest resource built into the Windows executable, a
   step of the release and a check that the command line still works in a terminal. Without it, a window shows for a
   moment at each logon, which the README says
2. A second, GUI-subsystem executable in the Windows archive that the task starts and that starts `run` with no console.
   Cost: a second program to build, ship and test, and the task naming it. Without it, the same moment of a window as in
   item 1
3. A hidden window in the background run that receives the end of the session and stops `run` cleanly at logoff and
   shutdown. Cost: a message loop on Windows and tests on Windows only. Without it, the Commander may keep the colour of
   `commander_color` until the next start of `run`, which the README says
4. `sensordeck service stop` and `sensordeck service start`, on both systems: on Windows by the stop event and `schtasks
   /Run`, on Linux by `systemctl --user stop` and `start`. Cost: two commands and their tests. Without them, the user
   stops the task cleanly only with `uninstall`, and starts it with `schtasks /Run`, as the README says
5. A first line in the log at each start of a background run, with the version of sensordeck, so that the starts are
   told apart. Cost: one line and its test. Without it, the times of the lines show the starts
6. A test on the Windows job of CI that installs the task with the real `schtasks.exe`, queries it and uninstalls it.
   Cost: a test limited to Windows that changes the Task Scheduler of the runner, which runs as an administrator and so
   does not check R1. Without it, the calls are checked against fakes and on the human's machine

## Sources

- [os/signal]: the Go documentation of the package `os/signal`, section "Windows", https://pkg.go.dev/os/signal
- [HandlerRoutine]: HandlerRoutine callback function, Windows Console,
  https://learn.microsoft.com/en-us/windows/console/handlerroutine
- [Console allocation policy]: Console Allocation Policy, Windows Console,
  https://learn.microsoft.com/en-us/windows/console/console-allocation-policy
- [Security contexts]: Security Contexts for Tasks, Task Scheduler,
  https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks
- [schtasks create]: schtasks create,
  https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/schtasks-create
- [schtasks end]: schtasks end,
  https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/schtasks-end
- [settingsType]: settingsType Complex Type, Task Scheduler schema,
  https://learn.microsoft.com/en-us/windows/win32/taskschd/taskschedulerschema-settingstype-complextype
- [RestartOnFailure]: RestartOnFailure (settingsType) Element,
  https://learn.microsoft.com/en-us/windows/win32/taskschd/taskschedulerschema-restartonfailure-settingstype-element
- [Interval]: Interval (restartType) Element,
  https://learn.microsoft.com/en-us/windows/win32/taskschd/taskschedulerschema-interval-restarttype-element
- [ExecutionTimeLimit]: ExecutionTimeLimit (settingsType) Element,
  https://learn.microsoft.com/en-us/windows/win32/taskschd/taskschedulerschema-executiontimelimit-settingstype-element
- [Elastic]: Proxy Execution via Console Window Host, a detection rule of Elastic Security,
  https://www.elastic.co/guide/en/security/current/proxy-execution-via-console-window-host.html
