# Running `run` as a systemd user service

## Approach

The work falls into four parts, all in `internal/cli`, and none changes what `run` does.

* **The seam** is three new fields of `environment`, which `Main` fills from the process and the harness of the unit
  tests fills with fakes (D13): the name of the system, a function that gives the path of the executable, and a
  function that makes a call. The real call is in a file of its own, `internal/cli/systemctl.go`, with no build tag,
  because `os/exec` builds on every system and R17 is decided by the field and not by the build
* **The unit file** is pure computation in `internal/cli/unitfile.go`: the quoting of R5, the content of R2, the reading
  of the output of `show` and the three lines of R16. It touches no file and makes no call, so it is tested with plain
  values
* **The commands** are in `internal/cli/service.go`: `service` and its three subcommands, which put the paths, the check
  of the config, the file and the calls together, and the check of the system that comes first in each of them
* **The check of the config** is the load of `run` (D4). The prefix that `run` puts in front of an error of the load,
  `config file <path>: `, moves into one function of `internal/cli/runconfig.go` that `run` and `install` both call,
  in a commit of its own that changes no behaviour (R7, R18)

None of the items of "Beyond the request" is built, as the record decided.

The spec fits the tree. `run` reads the config file with `readNamedFile("config file", path)`, which gives
`cannot read the config file <path>: <cause>`, and then gives `config file <path>: <error of loadConfig>`.
`os.UserConfigDir()` fails for a relative `XDG_CONFIG_HOME`, so the directory it gives is absolute. The existing
`ownConfigDirectory` and `configPath` of `internal/cli/run.go` give the default config file of R4. The persistent flags
`--device` and `--trace` of the root command are accepted by every subcommand already, so the `service` commands accept
them with no work (D15).

### The names and texts the implementation uses

These are decided here, so that the tests and the code agree. `<cause>` is the cause alone, through the existing
`innerError`, so that a path is not written twice.

| What | Exact text |
|---|---|
| Use line and Short of `service` | `service`, `Run sensordeck run as a systemd user service` |
| Use line and Short of `service install` | `install`, `Write, enable and start the systemd user service of run` |
| Usage of `--config` of `install` | `name this config file in the service instead of the one of the user` |
| Use line and Short of `service status` | `status`, `Show whether the service is installed, enabled and active` |
| Use line and Short of `service uninstall` | `uninstall`, `Stop, disable and remove the systemd user service` |
| No config directory (R1) | `cannot find the config directory: <cause>` |
| No path of the executable (R3) | `cannot find the path of the executable: <cause>` |
| A path with a control character (R5) | `cannot write the path <path as %q> into the unit file: it holds a control character` |
| A config file that cannot be read (R7) | `cannot read the config file <path>: <cause>`, as `run` |
| The same when the file does not exist (R7) | `cannot read the config file <path>: <cause>; sensordeck theme use <theme> creates it` |
| A config that does not load (R7) | `config file <path>: <error of loadConfig>`, as `run` |
| A unit file or its directory that cannot be written (R10) | `cannot write the unit file <unit file>: <cause>` |
| A call that failed (R10, R11) | `systemctl --user <arguments joined by spaces> failed: <cause of the call>` |
| A unit file that cannot be removed (R14) | `cannot remove the unit file <unit file>: <cause>` |
| A unit file whose presence cannot be checked | `cannot read the unit file <unit file>: <cause>` |
| Another system (R17) | `the service is not supported on <runtime.GOOS> yet; on Windows a startup task is planned` |
| The lines of `install` (R9) | `wrote <unit file>`, then `sensordeck.service is enabled and started` |
| The line of `uninstall` with no unit file (R12) | `sensordeck.service is not installed` |
| The line of `uninstall` (R13) | `sensordeck.service is stopped, disabled and removed` |
| The lines of `status` (R15, R16) | `installed: no`; or `installed: yes, <unit file>`, `enabled: <value>`, `active: <value> (<value>)` |

`<cause of the call>` is the standard error of the call with the white space at its two ends removed when that is not
empty, and otherwise the text of the error of the call: `exit status <n>` for a call that ended with a status other
than 0, and the error of the start, such as `exec: "systemctl": executable file not found in $PATH`, for one that could
not start (R10, R11).

The spec does not name a failure of the check whether the unit file exists, other than its absence, in `status` and
`uninstall`. Such a check is made with `os.Lstat`; an error that is not `fs.ErrNotExist` fails the command with exit
status 1 and the text of the table, and makes no call. A link at the path counts as a unit file that exists.

### The seam

```go
// In environment:
	// system is the name of the system, as runtime.GOOS gives it.
	system string
	// executable gives the path of the running executable, as os.Executable does.
	executable func() (string, error)
	// systemctl runs systemctl --user with the arguments and gives its standard output and standard error. Its error
	// is nil when the call succeeded, an *exec.ExitError when it ended with another status, and the error of the start
	// when it could not start.
	systemctl func(arguments ...string) (stdout, stderr string, err error)
```

`Main` fills them with `runtime.GOOS`, `os.Executable` and `runSystemctl`. `runSystemctl`, in
`internal/cli/systemctl.go`, runs `exec.Command("systemctl", append([]string{"--user"}, arguments...)...)` with two
`bytes.Buffer` as its standard output and standard error and gives them with the error of `Run`.

The harness of `internal/cli/commands_test.go` gets the fields `system` (`linux` unless a test sets it),
`executablePath` and `executableErr`, `calls [][]string`, which records the arguments of each call, `answer
func(arguments []string) (stdout, stderr string, err error)`, which gives the result of a call and succeeds with no
output when it is nil, and `configDirCalls` and `executableCalls`, counts of the calls of `userConfigDir` and
`executable`, so that R17 can be checked to touch nothing.

### The unit file

```go
// unitName is the name of the service.
const unitName = "sensordeck.service"

// unitFilePath gives the unit file in the config directory of the user.
func unitFilePath(configDirectory string) string // <configDirectory>/systemd/user/sensordeck.service

// quoteUnitPath gives the path in double quotes with \, ", % and $ escaped as R5 says, \ first, or an error naming the
// path when it holds a character below 0x20.
func quoteUnitPath(path string) (string, error)

// unitFileContent gives the content of R2 for the two paths, each quoted by quoteUnitPath.
func unitFileContent(executable, config string) (string, error)

// showProperties reads the lines <property>=<value> of the output of show, split at the first =. A line with no = is
// left out.
func showProperties(output string) map[string]string

// statusLines gives the three lines of R16 for the unit file and the properties, with unknown for a property that is
// missing or empty, and no parentheses when ActiveState is.
func statusLines(unitFile string, properties map[string]string) []string
```

The content of R2 is written in the code as one raw string literal with the two paths put in by `fmt.Sprintf`, so that
it reads as the spec writes it.

### The commands

```go
// serviceCommand is service and its three subcommands. On another system than Linux each of them fails at once.
func (current *session) serviceCommand() *cobra.Command
```

* `service` has `Args: cobra.NoArgs` and a `RunE` that gives the error of R17 on another system and the help of the
  command on Linux, which is what `theme` shows today, so that `sensordeck service` with no subcommand fails on Windows
  as R17 asks. Each subcommand has `Args: cobra.NoArgs`, and its `RunE` starts with `current.supported()`, which gives
  the error of R17 when `env.system` is not `linux`, before anything else
* `install`, in this order: the unit file from `env.userConfigDir` (R1); the executable from `env.executable`, then
  `filepath.EvalSymlinks`, then `filepath.Abs` (R3); the config file from `--config` or `configPath("")`, then
  `filepath.Abs`, with no link resolved (R4); the content of the unit file, which refuses a path by R5; the read of the
  config file with `readNamedFile`, adding the hint when `errors.Is(err, fs.ErrNotExist)`; the load with the shared
  function of the approach (R7); `os.MkdirAll` of the directory with `0o755` and `os.WriteFile` with `0o644` (R8,
  R10); the line `wrote`; the three calls, each through one function `call(arguments ...string) (string, error)` that
  gives the standard output or the error of the table (R8, R10, R11); the last line (R9). The hint is added on the
  error of the read alone, because an error of the compilation of a missing theme folder may also wrap
  `fs.ErrNotExist`
* `uninstall`: the unit file (R1); the check of its presence with `os.Lstat`; the line of R12 when it is absent; the
  call `disable --now sensordeck.service`; `os.Remove`; the call `daemon-reload`; the line of R13 (R12 to R14)
* `status`: the unit file (R1); the check of its presence; the line `installed: no` when it is absent; the call of R16;
  the lines of `statusLines` of `showProperties` of its standard output (R15, R16)

The shared load is:

```go
// loadConfigFile loads the content of the config file at the path as run does at its start, its error naming the file.
func (current *session) loadConfigFile(path string, content []byte) (runConfig, error)
```

It is `loadConfig` with the error given as `config file <path>: <error>`. `run` calls it in place of its own
`fmt.Errorf`, and `install` calls it after the read.

### The end-to-end tests

A new file `e2e/service_test.go` holds a helper that makes a temporary directory with an executable `systemctl`, a
script for `/bin/sh` that uses only built-ins of the shell: it appends its arguments, `$*`, as one line to a log file
whose path is written into the script; it writes `fake failure of $*` to standard error and ends with status 1 when
`$*` is the text it was told to fail at, also written into the script; and for `show` it writes
`UnitFileState=enabled`, `ActiveState=active` and `SubState=running`. The tests run the built executable with
`XDG_CONFIG_HOME` set to a temporary directory and `PATH` set to the directory of the fake alone, so that no real
`systemctl` is ever run (R22). The path of the executable that the unit file names is compared with
`filepath.EvalSymlinks` of the built executable, so that a temporary directory reached through a link does not fail
the test.

## Files

- `internal/cli/main.go`: the three fields of `environment`, filled by `Main`; `serviceCommand` added to the root
- `internal/cli/systemctl.go`: new, `runSystemctl`
- `internal/cli/unitfile.go`, `internal/cli/unitfile_test.go`: new, the pure functions of the unit file
- `internal/cli/service.go`, `internal/cli/service_test.go`: new, the `service` commands
- `internal/cli/runconfig.go`, `internal/cli/runconfig_test.go`: `loadConfigFile`
- `internal/cli/run.go`: `run` calls `loadConfigFile`, with its behaviour unchanged
- `internal/cli/commands_test.go`: the fields of the harness; `service` in the test of the help
- `e2e/service_test.go`: new, the end-to-end tests of R22
- `README.md`: the section of R19 and the status line of R20
- `changes/0014-systemd-user-service/record.md`: the log

## Tasks

### The shared load of the config

- [x] Add to `internal/cli/runconfig_test.go` a test of `loadConfigFile`: a config with `tags` loaded, and the error
  `config file <path>: <error>` for a file that does not parse and for a theme that does not compile (R7, R18)
- [x] Write `loadConfigFile` in `internal/cli/runconfig.go`, make `run` in `internal/cli/run.go` call it in place of
  its own `fmt.Errorf`, and make the tests pass, with the tests of `run` and the end-to-end tests of the stop on SIGTERM
  unchanged and passing (R7, R18)
- [x] Commit, as a change that keeps the behaviour of `run`

### The unit file

- [x] Write `internal/cli/unitfile_test.go`: `unitFilePath` under a directory; `quoteUnitPath` of a plain path, of
  paths with a space, `"`, `\`, `%` and `$`, of `\$` and `%%` in a path written as `\\$$` and `%%%%`, and its error
  for a path with a line feed, a carriage return and a tab, naming the path; `unitFileContent` against the whole
  content of R2 written in the test byte for byte, ending in one line feed; `showProperties` of the three lines, of a
  line with no `=`, of a value holding `=`; `statusLines` for the three properties, for each missing, for each empty,
  and for `ActiveState` missing with `SubState` given, written `active: unknown` (R1, R2, R5, R16)
- [x] Write `internal/cli/unitfile.go` as the approach says, and make the tests pass (R1, R2, R5, R16)
- [x] Commit

### The seam and `service install`

- [ ] Add `system`, `executable` and `systemctl` to `environment` in `internal/cli/main.go`, filled in `Main`, and write
  `internal/cli/systemctl.go`. Give the harness of `internal/cli/commands_test.go` the fields of the approach, with
  `system` set to `linux` in `newHarness` (R21, D13)
- [ ] Write the tests of `install` in `internal/cli/service_test.go`, with a helper that gives a harness whose config
  directory is a temporary one holding `sensordeck/config.yaml` with `theme: minimal-dark`, and whose executable is a
  file in another temporary directory: the unit file byte for byte without `--config`, naming the default config file,
  and with an absolute `--config` and a relative one, made absolute against the working directory of the test; a
  `--config` that is a link to a file kept as the link; an executable given through a link resolved to the file; an
  executable and a config path in directories whose names hold a space, `"`, `\`, `%` and `$`, written as R5 says; a
  config path with a line feed refused with no file and no call; the config directory that cannot be found, with no
  call; the executable that cannot be found, and one whose link leads nowhere, with no file written; the two lines of
  standard output and exit status 0; the three calls in order after the file is written, checked by the answer of the
  first call finding the file; a unit file that exists with other content replaced; the directory of the unit file
  created when it is missing; a unit file that cannot be written, as when `systemd/user` is a file, with no call; a
  failure of each of the three calls, with the calls after it not made, the error naming the call, and the unit file
  left in place; the standard error of a call as the cause, trimmed, and an empty one giving `exit status <n>` from a
  real `*exec.ExitError` made by running `sh -c 'exit 3'`; and a call that cannot start giving the error of the start
  (R1 to R4, R6, R8 to R11, R21)
- [ ] Add the tests of the check: a missing config file, with the hint; a file that does not parse; a theme that does
  not compile; each with no unit file written and no call made, and the error compared with the standard error that
  `run --config <same file>` gives in another harness, so that the text is the one of `run` (R7, R21)
- [ ] Write `serviceCommand` and `install` in `internal/cli/service.go`, add `serviceCommand` to the root in
  `internal/cli/main.go`, add `service` to the names that `TestHelpNamesTheCommandsAndTheOptions` checks, and make the
  tests pass; check that `install` opens no device and reads no sensor by `harness.opened` empty and `sourcesErr` set
  (R6 to R11)
- [ ] Commit

### `service uninstall` and `service status`

- [ ] Add to `internal/cli/service_test.go` the tests of `uninstall`: no unit file, with its line, no call and exit
  status 0; with a unit file, the call `disable --now sensordeck.service`, the unit file removed, checked by the answer
  of the second call finding it gone, then `daemon-reload`, the line and exit status 0; a failure of `disable --now`,
  with the unit file kept and no further call; a failure of the removal, as when the unit file is a directory that
  holds a file, with no further call; a failure of the last `daemon-reload`, with the unit file removed; and the config
  directory that cannot be found (R1, R12 to R14, R21)
- [ ] Add the tests of `status`: no unit file, with `installed: no`, no call and exit status 0; with a unit file, the
  one call of R16 and the three lines from an output that holds the three properties; an output with a property
  missing and one with a value empty; a failure of the call, as R10 says (R15, R16, R21)
- [ ] Write `uninstall` and `status` in `internal/cli/service.go`, and make the tests pass (R12 to R16)
- [ ] Commit

### Other systems

- [ ] Add to `internal/cli/service_test.go` a test that runs `service`, `service install`, `service install --config
  <missing file>`, `service status` and `service uninstall` with `system` set to `windows`: each fails with exit
  status 1 and the error of R17, with `configDirCalls`, `executableCalls` and `calls` at zero and the temporary config
  directory empty; and that `service --help` names the three subcommands with `system` set to `windows`, and `service`
  with no subcommand on `linux` ends with exit status 0 and its help (R17, R21)
- [ ] Write `supported` in `internal/cli/service.go` and call it first in the four commands, and make the tests pass
  (R17)
- [ ] Commit

### The end-to-end tests

- [ ] Write `e2e/service_test.go` with the fake `systemctl` of the approach, and the tests: `theme use minimal-dark` and
  then `service install`, with the two lines, the unit file naming the built executable and the config file of
  `XDG_CONFIG_HOME`, and the three calls in the log; `install` with no config file and with a config file of
  `theme: no-such-theme`, each with exit status 1, no unit file and an empty log; `install` with the fake failing at
  `enable sensordeck.service`, with exit status 1, the error naming the call and holding `fake failure`, and the log
  holding `daemon-reload` and the failed call alone; `status` before `install`, writing `installed: no` with an empty
  log, and after it, writing the three lines; and `uninstall` after `install`, with its line, the unit file removed and
  the two calls of R13 at the end of the log (R22)
- [ ] Commit

### The documents

- [ ] Add to `README.md` a section "Running as a service", after "The Corsair iCUE Commander Core XT" and before
  "Writing a theme", which says each point of R19 in the order the spec lists them, with the commands written as the
  spec writes them: `sensordeck service install` and `--config`, `sensordeck service status`,
  `systemctl --user status sensordeck.service`, `journalctl --user -u sensordeck` and `-f`,
  `systemctl --user restart sensordeck.service`, `systemctl --user stop sensordeck.service`,
  `systemctl --user edit sensordeck.service`, `sensordeck service uninstall`, `loginctl enable-linger` and
  `loginctl disable-linger`, and the path of the unit file on Linux; claim nothing as seen on the device (R19, D18)
- [ ] Change the status line of `README.md` to name `sensordeck service install`, `status` and `uninstall` among what
  works on Linux and to say that Windows is still to come, removing that the service is still to come; keep "Planned
  usage" as it is (R20)
- [ ] Commit

### Handing over

- [ ] Check each case that R21 lists against the unit tests written above, and add a test for any case missing (R21)
- [ ] Run the five commands of the suite in `AGENTS.md` and `npm run check-changes`, all succeeding (R18, R23)
- [ ] Log the implementation in `record.md`, tick the tasks, and commit

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
