# Running `run` as a systemd user service

## Context

Step 8 of the roadmap of the initial plan is the service mode: a systemd user service on Linux and a startup task on
Windows, with the theme uploaded again only when it changed. Unit `0013-theme-key-and-upload-on-change` made the
first half: `run` takes the theme from the key `theme` of its config file, uploads it only when the hash stored for
the serial of the screen differs, and applies a changed config file within a round, so that `theme use` switches the
theme of a running `run`. This unit is the second half on Linux: `sensordeck service install`, `status` and
`uninstall`, which make `run` start with the user's systemd instance and keep running unattended. The architecture of
the initial plan names `sensordeck service install` among the commands of the CLI.

What the code gives today, which this unit builds on:

- `run` reads its config file from `--config` or from `sensordeck/config.yaml` in `os.UserConfigDir()`, which on Linux
  is `$XDG_CONFIG_HOME`, or `~/.config` without it. It parses the file and, with `theme`, compiles the theme before it
  opens the screen, and fails with exit status 1 and an error that names the config file when either fails. It then
  opens the screen, and fails with exit status 1 when it finds none or cannot open it
- `run` stops on SIGINT and SIGTERM. A stop ends the rounds, and with `commander_color` the keeper ends its session of
  the controller with the teardown, which returns the controller to hardware mode. `run` then closes the screen,
  writes `stopped` and ends with exit status 0. It sends the screen nothing at the stop, so the screen keeps the last
  values and the theme it holds. An upload that has begun runs to its end first, which takes about 10 seconds. The
  end-to-end tests check the stop on SIGTERM, with and without the controller
- Once running, `run` survives a lost screen, a lost controller and an invalid reload of its config file, each with
  lines on standard error and a retry, and does not end because of them
- `run` writes its lines to standard output and standard error, unbuffered, so each line reaches a pipe when it is
  written
- The commands take the outside world through one environment that `Main` fills from the process and the unit tests
  fill with fakes. The end-to-end tests build the executable and run it with a `PATH` they set, and put a fake
  `nvidia-smi` on it for the tests that need one
- On a system other than Linux, the commands that open a device fail with `<system> is not supported yet`, and every
  command is present

"The unit file" below is the file of R1. "The service" is the systemd user unit `sensordeck.service`. "A call" is a run
of `systemctl --user` with the arguments that follow it.

## Requirements

### The unit file

- **R1** The unit file is `systemd/user/sensordeck.service` in the directory that `os.UserConfigDir()` gives: on Linux
  `$XDG_CONFIG_HOME/systemd/user/sensordeck.service`, or `~/.config/systemd/user/sensordeck.service` without that
  variable. When `os.UserConfigDir()` fails, the `service` commands fail with exit status 1 and an error that says the
  config directory cannot be found, and make no call
- **R2** The content of the unit file that `install` writes is exactly this, where `<executable>` and `<config>` are
  the arguments of R3 and R4, written as R5 says:

  ```ini
  # Written by sensordeck service install, which overwrites it when run again.
  # Put changes in a drop-in instead: systemctl --user edit sensordeck.service

  [Unit]
  Description=sensordeck: sensor values on a USB screen
  StartLimitIntervalSec=0

  [Service]
  Type=exec
  ExecStart=<executable> run --config <config>
  Restart=on-failure
  RestartSec=10s
  StandardOutput=journal
  StandardError=journal

  [Install]
  WantedBy=default.target
  ```

  It ends in one line feed and has no other line, no `After=`, no `Wants=`, no `Requires=` and no `Environment=`
- **R3** `<executable>` is the path of the running executable that `os.Executable()` gives, with its symbolic links
  resolved and made absolute. When either step fails, `install` fails with exit status 1 and an error that says the
  path of the executable cannot be found, and writes nothing
- **R4** `<config>` is the config file that `install` checks by R7: the path given with `--config` made absolute and
  cleaned, or without the flag the path `run` reads without it, `sensordeck/config.yaml` in `os.UserConfigDir()`. Its
  symbolic links are not resolved, so that a config file kept as a link into a repository of the user's is read
  through the link
- **R5** Each of the two paths is written in double quotes. Inside the quotes, `\` is written `\\`, `"` is written
  `\"`, `%` is written `%%` and `$` is written `$$`, so that systemd reads back exactly the path, with no specifier and
  no variable expanded. A path that holds a line feed, a carriage return or another control character below 0x20 is
  refused by `install` with exit status 1 and an error that names the path, and nothing is written

### `sensordeck service install`

- **R6** `sensordeck service install` takes no argument and the flag `--config <path>`. It opens no device and reads no
  sensor
- **R7** Before it writes anything, `install` reads the config file of R4 and loads it as `run` does at its start: it
  parses it with the parser of `run` and, with `theme`, compiles the theme, a relative folder being taken relative to
  the directory of the config file. When the file cannot be read, does not parse or names a theme that does not
  compile, `install` fails with exit status 1 and the error that `run` gives for it at its start, writes no unit file
  and makes no call. When the file does not exist, the error also says that `sensordeck theme use <theme>` creates it
- **R8** After the check, `install` creates the directory of the unit file when it is missing, and writes the unit file
  of R2, replacing a file that exists at that path whatever it holds. It then makes these calls in this order, each
  only after the one before succeeded:
    1. `daemon-reload`
    2. `enable sensordeck.service`
    3. `restart sensordeck.service`
- **R9** `install` writes these lines to standard output: `wrote <unit file>` after the file is written, and
  `sensordeck.service is enabled and started` after the last call succeeded. It then ends with exit status 0
- **R10** When the unit file cannot be written, `install` fails with exit status 1 and an error that names the unit
  file and the cause, and makes no call. When a call fails, `install` makes no further call and fails with exit status
  1 and an error that names the call, as `systemctl --user <arguments>`, and gives the text the call wrote to its
  standard error, or its exit status when it wrote none. The unit file it wrote is left in place
- **R11** When `systemctl` cannot be started, as when it is not on the `PATH`, the call fails as R10 says, with the error
  of the start as the cause

### `sensordeck service uninstall`

- **R12** `sensordeck service uninstall` takes no argument. When the unit file does not exist, it writes
  `sensordeck.service is not installed` to standard output, makes no call, removes nothing and ends with exit status 0
- **R13** When the unit file exists, `uninstall` makes the call `disable --now sensordeck.service`, then removes the
  unit file, then makes the call `daemon-reload`, and writes `sensordeck.service is stopped, disabled and removed` to
  standard output and ends with exit status 0
- **R14** When the call `disable --now` fails, `uninstall` removes nothing, makes no further call and fails as R10 says.
  When the removal fails, it fails with exit status 1 and an error that names the unit file and the cause, and makes
  no further call. When the last `daemon-reload` fails, it fails as R10 says, and the unit file stays removed

### `sensordeck service status`

- **R15** `sensordeck service status` takes no argument. When the unit file does not exist, it writes the one line
  `installed: no` to standard output, makes no call and ends with exit status 0
- **R16** When the unit file exists, `status` makes the one call
  `show --property=UnitFileState --property=ActiveState --property=SubState sensordeck.service`, reads the lines
  `<property>=<value>` that it writes to standard output, and writes these three lines to standard output and ends with
  exit status 0:

  ```
  installed: yes, <unit file>
  enabled: <UnitFileState>
  active: <ActiveState> (<SubState>)
  ```

  A property that the output does not hold, or holds with an empty value, is written as `unknown`, with
  `active: unknown` when `ActiveState` is missing or empty, and no parentheses then. When the call fails, `status`
  fails as R10 says

### Other systems

- **R17** The command `service` and its three subcommands are present on every system and appear in the help as on
  Linux. On a system other than Linux each of them fails at once with exit status 1 and the error
  `the service is not supported on <system> yet; on Windows a startup task is planned`, where `<system>` is the name
  `runtime.GOOS` gives, and reads no file, writes no file and makes no call. The flag `--config` of `install` is
  accepted there too

### `run` under systemd

- **R18** `run` is not changed by this unit. Its behaviour on SIGTERM, described in the context, is the stop of the
  service: the screen keeps the last values and its theme, the controller is back in hardware mode, `run` ends with
  exit status 0 and systemd does not restart it

### The documents

- **R19** `README.md` has a section on running sensordeck as a service, which says:
    - that it is for Linux, and that the config file is to be set first, as the quick start says
    - `sensordeck service install`, with `--config`, what it checks, the path of the unit file, and that it enables
      and starts the service at once and at each later login
    - that running `install` again overwrites the unit file and restarts the service, which is the way to apply a moved
      or replaced executable or another `--config`, and that a change of the unit is made in a drop-in, with
      `systemctl --user edit sensordeck.service`, which `install` does not overwrite
    - `sensordeck service status`, and `systemctl --user status sensordeck.service` for more
    - the log, `journalctl --user -u sensordeck`, with `-f` to follow it, and that it holds the lines `run` writes to
      standard output and standard error
    - that `sensordeck theme use` and edits of the config file reach the running service within a round, by the reload
      of `run`, and that a change of `interval`, `commander_interval` or `commander_color` needs
      `systemctl --user restart sensordeck.service`
    - that the service is restarted 10 seconds after it ended with an error, as when no screen was found at its start,
      with no limit to the number of restarts, and that a stop with `systemctl --user stop sensordeck.service` leaves
      the screen with its last values and returns the Commander Core XT to hardware mode
    - that running `sensordeck run` or another command that writes to the screen by hand while the service runs is not
      supported, because both write to the screen, and that the service is stopped first
    - `sensordeck service uninstall`
    - that a user service runs only while the user has a session, that `loginctl enable-linger` makes it start at boot
      and keep running after the user logged out, that `loginctl disable-linger` undoes it, and that lingering is the
      user's choice, which `install` does not make
    - that the udev rule of the README gives access to the screen to the user of the active local session, so that a
      service that starts at boot by lingering cannot open the screen until that user logs in, and is restarted every
      10 seconds until then
- **R20** The status line of the README names the service among what works on Linux, and no longer says that the
  service is still to come. It says that Windows is still to come. "Planned usage" is kept as it is

### Tests

- **R21** The unit tests cover at least, with the calls behind a fake that records them and gives a chosen result, the
  path of the executable from a fake, and the config and unit directories in temporary directories:
    - R1 to R5: the content of the unit file, byte for byte, with and without `--config`, a relative `--config`, a
      config path that is a symbolic link kept as it is, an executable path through a symbolic link resolved, paths with
      a space, `"`, `\`, `%` and `$` written as R5 says, a path with a line feed refused, and a config directory that
      cannot be found
    - R7: a missing config file and its hint, a file that does not parse, a theme that does not compile, and in each
      case no file written and no call made
    - R8 to R11: the three calls in order after the file is written, a unit file that exists replaced, a failure of each
      call with the calls after it not made and the error naming the call, an error text of the call and an empty one,
      and `systemctl` that cannot be started
    - R12 to R14: no unit file, the order of the call, the removal and the second call, and a failure of each step
    - R15 and R16: no unit file, the three lines from an output of the call, a property missing and an empty one, and a
      failure of the call
    - R17: each of the four commands with the system set to `windows`, by a fake of the system, with no file read or
      written and no call made
- **R22** The end-to-end tests run the built executable with `XDG_CONFIG_HOME` set to a temporary directory and a
  `PATH` that holds only a fake `systemctl`, which records its arguments and fails when told to, so that no real
  `systemctl` is ever run. They check: `install` after `theme use minimal-dark`, with the unit file naming the built
  executable and the config file, and the three calls; `install` with no config file and with a config whose theme
  does not compile, with no unit file and no call; `install` with a call that fails; `status` before and after
  `install`; and `uninstall` with its calls and the unit file removed
- **R23** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 A user unit, written by sensordeck.** The screen and the controller are reached through hidraw nodes that the
  udev rule gives to the user of the session, and the config file, the themes and the state file are the user's, so
  the service runs as the user, with no root. Writing the unit file and calling `systemctl --user` is what the request
  asks for and needs no library: a D-Bus client of systemd was rejected as a dependency for three calls
- **D2 The executable path is resolved.** A service outlives the shell that installed it. `os.Executable()` gives the
  file that runs, which is the path that keeps working while the file stays, and resolving its links makes the unit
  name the real file rather than a link that a package manager or the user may point elsewhere. The cost is that a
  link updated to a new release is not followed, and running `install` again fixes the path, as R19 says. Keeping the
  link was rejected, because the request asks for the resolved path
- **D3 The unit always names the config file.** The request asks for `--config` only when it is given to `install`.
  The spec writes it always, with the path `install` checked, because the environment of the systemd user instance
  is not the one of the shell: a `XDG_CONFIG_HOME` set in a shell profile is not seen by the service, which would then
  read another file than the one `install` checked and `theme use` wrote. Naming the file makes R7 a check of the file
  the service reads. The cost is that a later change of `XDG_CONFIG_HOME` is not followed until `install` is run again.
  The unit file itself is put by `os.UserConfigDir()` of the shell, as the request says; when the instance looks in
  another directory, the call `enable` fails with the error of `systemctl`, which says the unit was not found, and the
  user sees it
- **D4 The check is the load of `run`.** `run` ends at its start both for a file that does not parse and for a theme
  that does not compile, so a check of the parser alone would install a service that fails at once and is restarted
  every 10 seconds for ever. The same code as `run` gives the same errors, so a config that `install` accepts is one
  `run` starts with. The check needs no screen and does not open one, so a screen that is unplugged at `install` does
  not stop it, and the service then waits for it by R2. The state file is not checked: without the cache directory
  `run` fails, which is a failure the journal shows
- **D5 Overwrite, enable and restart.** Running `install` again is the way to fix the unit after the executable moved
  or the config path changed, so it must replace the file and make the running service use it. `enable --now`, which
  the request names, does not restart a running service, so a running one would keep the old executable. One sequence,
  `daemon-reload`, `enable`, `restart`, does both cases, because `restart` starts a service that is not running, and
  has one path to test. Refusing an existing file, or asking, was rejected: the file is sensordeck's own, as its first
  lines say, and a change of the user's goes in a drop-in, which `install` does not touch
- **D6 `Restart=on-failure`, `RestartSec=10s`, no start limit.** `run` ends with an error only at its start: a missing
  screen, an unreadable or invalid config, or a theme that no longer compiles. Once running it handles a lost screen
  and an invalid reload itself. A failure at the start is often one that time cures, such as a screen not yet
  enumerated at boot or a lingering service before the user's login, so the service retries for ever. 10 seconds is
  the first wait of the project's other retries. `StartLimitIntervalSec=0` removes the limit of starts, which with the
  defaults of systemd would not be reached at one start per 10 seconds, but is stated so that a changed default on a
  distribution does not leave the screen dead. `on-failure` and not `always`, so that a `run` that ended with exit
  status 0 after a SIGINT sent by hand stays stopped; a stop by systemd is never restarted either way
- **D7 `Type=exec`.** It makes `install` fail at the call `restart` when the executable cannot be run, such as after it
  was deleted, rather than report a start and fail later in the journal. `Type=simple` was rejected for that reason.
  `Type=notify` was rejected because it needs code in `run` for a readiness that nothing waits for. `Type=exec` needs
  systemd 240 or later, from 2018
- **D8 No dependency on a target but `default.target`.** The screen is on USB and not part of the graphical session,
  and `run` waits for a missing screen by D6. `graphical-session.target` was rejected because it is reached only in
  some desktop sessions and never for a lingering service at boot, so the service would not start at all there.
  `WantedBy=default.target` starts it with the user's systemd instance, at login or at boot with lingering. `After=`
  a device unit was rejected because the hidraw node has no stable name to wait for
- **D9 The output goes to the journal by the unit.** `StandardOutput=journal` and `StandardError=journal` are the
  defaults of systemd for a service, and are written so that a changed default of the manager does not lose the log.
  `run` writes its lines unbuffered, so each reaches the journal when it is written, and `journalctl --user -u
  sensordeck` shows them. No prefix of a level, such as `<3>`, is added: it would change the lines `run` writes, which
  are the same in a terminal
- **D10 The stop is the stop of `run`.** systemd stops the service with SIGTERM and waits 90 seconds by default
  before SIGKILL, longer than an upload of about 10 seconds and the teardown of the controller, so the unit sets no
  `KillSignal=` and no `TimeoutStopSec=`. `run` already stops cleanly on SIGTERM and needs no change
- **D11 `status` reports three fields.** Passing through `systemctl --user status` was rejected: it ends with exit
  status 3 for a stopped service and 4 for an unknown one, which `status` would have to map, its output is that of
  systemd and changes with its version, and it can start a pager. `systemctl --user show` gives the properties as
  `Key=Value` lines and exit status 0 for a loaded unit or not, so `status` reports what the request asks, installed,
  enabled and active, in lines that can be tested. The README points to `systemctl --user status` for the rest. A
  missing unit file is reported with no call, so that `status` works with no systemd
- **D12 `uninstall` of nothing succeeds.** The aim of `uninstall` is that no service is left, which holds already when
  there is no unit file, so it reports it and succeeds. A unit enabled from another directory is not sensordeck's
  and is not looked for
- **D13 The calls behind a narrow seam.** The commands take, through the environment, a function that runs
  `systemctl --user` with arguments and gives its standard output, its standard error and whether it succeeded, and a
  function that gives the path of the executable. `Main` fills them with the real ones. The unit tests then check the
  file and the calls with no systemd, and the end-to-end tests check the real start of a process through a fake
  `systemctl` on the `PATH`, as the project does for `nvidia-smi`. The system for R17 is taken from the environment
  too, so the Windows path is tested on Linux
- **D14 Present and failing on other systems.** Hiding the command on Windows was rejected: the user would get
  `unknown command`, which does not say that the service is planned, and the help and the README would differ by
  system. A command that is present and fails with a message says what is missing and where it is going, and keeps one
  command tree. The text names the startup task because it is the Windows half of step 8 of the roadmap
- **D15 `--device` and `--trace` are not written into the unit.** The node number of a hidraw device changes between
  boots and plugs, so the service finds the screen by discovery, and a trace of every report would fill the journal.
  The two flags of every command are accepted by the `service` commands and have no effect on them, as on `theme use`.
  A user who wants them adds them in a drop-in
- **D16 Lingering is the user's.** Lingering keeps the user's processes running with no session, which is a choice
  about the machine, and `loginctl enable-linger` may need the policy of the system. `install` does not make it, and
  the README says how and what it costs with the udev rule, by R19
- **D17 The threat model.** The unit file, the config file and the executable belong to the user who runs `install`,
  and are trusted as theirs. The quoting of R5 is there so that any path the user has works, not to defend against a
  hostile path. The unit does not defend against another user who can write these files or the user's systemd
  directory
- **D18 No check on a real systemd in this unit.** Whether the unit file starts, restarts and stops as described on
  the developer's machine, with the screen and the controller, and how lingering and the udev rule meet at boot, is
  the human's to check later. The loop does not wait for it, the unit is complete without it, and a finding of it is a
  new unit. The README describes the unit as written and what systemd documents, and claims nothing as seen on the
  device

## Non-goals

- A startup task on Windows, and the service on any system but Linux
- A system-wide service run by root, or a unit in `/etc/systemd`
- Socket or D-Bus activation, and readiness notification with `Type=notify`
- Enabling or disabling lingering
- Changing `run`, its signals, its exit statuses or its output
- Following a symbolic link to the executable that is later changed, other than by running `install` again
- Reading or merging a drop-in, or a unit file of sensordeck in any other directory than that of R1
- A unit of another name, or several services for several screens
- Passing environment variables, `--device` or `--trace` to the service

## Beyond the request

1. `install` printing a line after a successful install that the service runs only while the user has a session, and
   naming `loginctl enable-linger`. Cost: one line and its test. Without it, the user learns it from the README, and a
   service that stopped at logout is seen and fixed with one command
2. `status` comparing the executable of the unit file with the running one, and saying that `install` is to be run
   again when they differ. Cost: reading the `ExecStart=` line back with the quoting of R5, and tests. Without it, a
   moved executable makes the service fail, which `status` shows as failed and the journal explains
3. `status` writing the last lines of the journal, through `journalctl --user -u sensordeck -n 10`. Cost: a second
   command behind the seam and its tests. Without it, the user runs `journalctl`, which the README names
4. Refusing `--device` and `--trace` on the `service` commands with an error, rather than accepting them with no effect.
   Cost: a check and a test. Without it, a user who gave them sees in the unit file that they are not there
