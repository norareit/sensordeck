# Plan: custom metrics defined in the config file

## Approach

The work begins with two commits that prepare the ground: the lookup of hwmon chips by name moves out of
`internal/linuxsensors` into a package that every system builds, and every child process that reads a metric gets the
flag that keeps a console window from opening on Windows. Then come, in this order, the name rule, the theme compiler,
the key `metrics` of the config, the package that reads custom metrics, `run`, `sensors`, the end-to-end tests and the
documents. Each step leaves the suite passing.

What the tree on `feature-0024-custom-metrics` holds, and what the plan builds on:

* `internal/sensors/metrics.go` holds the table `metricKinds`, `IsName`, `Unit`, the name builders, `InTableOrder` and
  `position`. `source.go` holds `Reading`, `Source` and `ReadAll`; `convert.go` holds `ToValue`
* `internal/config/config.go` reads the keys `theme`, `tags`, `interval`, `commander_interval`, `commander_color` and
  `librehardwaremonitor_url`, names them in the error of an unknown key and in `errNotAMapping`, and refuses a key given
  twice with `refuseRepeatedKeys`. `parseTags` refuses a name that `sensors.IsName` refuses with
  `tags: "<name>" is not a metric name`
* `internal/themecompile/document.go` refuses, in `readNumberKeys`, a metric that `sensors.IsName` refuses with
  `metric: "<name>" is not a metric name`. `document_test.go` has the case "metric misspelt" with `cpu.tmp`, which has
  the form of R2 and becomes accepted
* `internal/linuxsensors/hwmon.go` has `entry{number, name, path, device}` and `entriesNamed(sysfsRoot, driver)`, which
  leaves out a chip whose `device` cannot be resolved, orders the rest by the resolved device path and then by N, and
  is used by `cpuTemperatures` and `numberedTemperatures`. The field `name` is written and never read
* `internal/nvidiasmi/source.go` starts `nvidia-smi` with `exec.CommandContext`, `WaitDelay` set to `pipeGrace` (500
  ms), and no `SysProcAttr`. Its tests are limited to Linux and include a run killed with a child holding its output
* `internal/cli/runconfig.go` holds `loadConfig`, which parses the file and, with a theme, compiles it and takes its
  tags as the mappings, `loadConfigFile`, `configFileError`, `restartKeys` and `configWatcher`. `run`, its reload and
  `service install` (`checkConfig` in `service.go`) all load through `loadConfig`
* `internal/cli/run.go` holds `prepareRun`, `serveRun`, which builds the `feeder` and is the one place that does, and the
  `feeder`, whose `reload` applies the mappings and the theme and whose `read` sends every reading of a mapped metric
  through `sensors.ToValue`. The `stderr` of `run` is a `lockedWriter`
* `internal/cli/commands.go` holds `sensorsCommand`, which reads the config only through `monitorURL`, only when the set
  has a monitor (Windows), with no `--config` flag, and `monitorOutcome`. `sensors_test.go` has
  `TestSensorsOnASetWithoutAMonitorNeverLooksForTheConfig`, which R25 makes false
* The harness of `internal/cli/commands_test.go` has `configDir`, `configDirErr` and `configDirCalls`, and
  `windowssensors_linux_test.go` has `windowsHarness` and `writeDefaultConfig`
* `layout_test.go` allows `golang.org/x/sys/windows` only in three seam files, limits `internal/fakesensors` and `e2e`
  to Linux, and forbids `os`, `os/exec` and `syscall` in `internal/themecompile`
* `e2e/main_test.go`'s `environmentOf` sets `XDG_CACHE_HOME` and not `XDG_CONFIG_HOME`, so a `sensors` run of the
  end-to-end tests inherits the config directory of the user who runs them. `exec.Cmd` keeps the last of two values of
  one variable, so a variable that a test gives overrides one that `environmentOf` sets before it.
  `e2e/theme_upload_test.go` has `themeSetup`, `expectUploadAt` and `expectClockThenValue`

### Choices the spec leaves to the plan

* **The shared lookup of chips** is a new package `internal/hwmonchips` with no build constraint. `Named(sysfsRoot,
  name string) []Chip` gives every `hwmonN` directory of `<sysfsRoot>/class/hwmon` whose N is digits and whose `name`
  file, without its trailing line feed, is the name, as `Chip{Number int; Path string; Device string}`, `Device` the
  resolved path of `device` or empty when it cannot be resolved. The order is that of R15: the chips with a device by
  their device compared by bytes and then by N, then those without by N. `entriesNamed` of `linuxsensors` keeps its
  name and gives the chips that have a device, which is the order it gives today, and `entry` is replaced by
  `hwmonchips.Chip`
* **The flag on Windows** is set by a new package `internal/childprocess` with `NoWindow(command *exec.Cmd)`. In
  `nowindow_windows.go` it sets `CreationFlags |= createNoWindow`, a constant `0x08000000` of its own, on the
  `SysProcAttr` of the command, making one when it is nil. It uses `syscall` and not `golang.org/x/sys/windows`, so
  that `layout_test.go` stays as it is. `nowindow_other.go` (`!windows`) does nothing
* **The name rule** is `sensors.IsWellFormedName(name string) bool`, the form of R2, beside `IsName`
* **The definitions** are `config.Metric`, a comparable struct, so that R28 compares two definitions with `==`:

    ```go
    type Metric struct {
        Name    string
        File    string        // the file form
        Hwmon   string        // the hwmon form, with Input and Index
        Input   string
        Index   int           // 1 without the key
        Command string        // the command form, with Every
        Every   time.Duration // DefaultEvery without the key; zero for the other forms
        Scale   float64       // 1 without the key
        Offset  float64
        Unit    string
    }
    ```

    `Config.Metrics []Metric` is in the order of the file, empty without the key. `config.DefaultEvery` is 10 seconds
    and `config.MinimumEvery` 1 second. `Config.IsMetric(name string) bool` tells whether a name is built in or defined,
    and is the one place that encodes R7 and R8
* **The errors of `metrics`** follow the form of `tags`. Each text below follows `config file <path>: ` as today:
    * `metrics: want a mapping from metric name to definition, with at least one entry`
    * a name given twice: the error of `refuseRepeatedKeys`, as for `tags`
    * `metrics: "<name>" is not a metric name: segments of a to z, 0 to 9, _ and - joined by dots, starting with a letter`
    * `metrics: <name>: is a built-in metric and cannot be replaced`
    * `metrics: <name>: want a mapping with one of the keys file, hwmon and command`, for a definition that is not a
      mapping; a key given twice in a definition gives the error of `refuseRepeatedKeys`
    * `metrics: <name>: unknown key "<key>", the keys are command, every, file, hwmon, index, input, offset, scale and unit`
    * `metrics: <name>: want exactly one of the keys file, hwmon and command`, for none or several
    * `metrics: <name>: <key>: only with hwmon`, for `input` and `index`, and `metrics: <name>: every: only with command`
    * `metrics: <name>: input: needed with hwmon`
    * `metrics: <name>: file: want an absolute path`, checked with `filepath.IsAbs` of the system that runs
    * `metrics: <name>: hwmon: want a chip name, a non-empty string without /`
    * `metrics: <name>: input: want the name of a file of the chip, a non-empty string other than . and .. without /`
    * `metrics: <name>: index: want a whole number of 1 or more`
    * `metrics: <name>: command: want a non-empty string`
    * `metrics: <name>: <key>: want a finite number`, for `scale` and `offset`
    * `metrics: <name>: unit: want a string without tabs, line feeds or other control characters`
    * `metrics: <name>: every: "<text>" is not a duration such as 30s or 5m` and
      `metrics: <name>: every: <text> is shorter than 1s`

    A string key takes a scalar whose tag is `!!str`. `scale` and `offset` take a scalar tagged `!!int` or `!!float`
    that decodes into a finite `float64`, so `.inf`, `.nan` and a quoted number are refused. `index` takes a scalar
    tagged `!!int` that decodes into an `int64` of 1 or more. A control character is one that `unicode.IsControl`
    accepts. The checks of one metric run in the order of the list, so that a file with several problems gives the
    first. `metrics` is parsed before `tags`, which needs the definitions
* **R7 and R8 keep the existing texts where there is one.** `tags` keeps `tags: "<name>" is not a metric name`, which
  names it. The theme check of R8 is in `loadConfig` after the compilation:
  `theme: <value>: the metric "<name>" is neither built in nor defined under metrics`, with the value of the key
  `theme` as the file gives it, checked for the names of the compiled tags in their order
* **The reading** is a new package `internal/custommetrics` with no build constraint, which imports `config`,
  `hwmonchips`, `childprocess` and `sensors`:
    * `number.go`: `Number(text string) (float64, bool)`, the first match of `[-+]?[0-9]+(\.[0-9]+)?` parsed with
      `strconv.ParseFloat`, false for no match or a result that is not finite (R11)
    * `outcome.go`: `outcome{value float64; cause string}`, an empty cause meaning a value, and the causes of R23:
        * `cannot read <path>: <cause>`, with the `Err` of the `*fs.PathError`, for a file or an input
        * `<count> hwmon chips named <chip>, fewer than index <index>`
        * `no number in <path>` for a file or an input, and `no number in the output of the command`
        * `the command exited with status <status>` and, when its standard error has a line, `: <first line>`, the
          line without a trailing CR
        * `the command could not start: <error>`
        * `the command outlasted its time limit of <limit>`, the limit written as `time.Duration` writes it
    * `read.go`: `readFile(definition)`, `readHwmon(sysfsRoot, definition)` and the value of R12,
      `number*Scale + Offset`
    * `command.go`: `TimeLimit` (10 seconds), `timeLimit(definition)` (`min(TimeLimit, Every)`), and
      `runCommand(stop context.Context, text, directory string, limit time.Duration) (outcome, bool)`, which runs the
      command of `shellCommand` with `Dir` the directory, the inherited environment, no standard input, standard
      output and standard error in buffers and `WaitDelay` 500 milliseconds. The bool is false when `stop` was done
      before the run ended, so that its outcome is dropped (R21, R28). The time limit is a context of its own derived
      from `stop`, so the two are told apart
    * `shell_linux.go`: `shellCommand(limited context.Context, text string) *exec.Cmd` runs `/bin/sh -c <text>` with
      `Setpgid` set, and its `Cancel` sends `SIGKILL` to the process group, `-Process.Pid` (R16, R18)
    * `shell_windows.go`: the program is `ComSpec`, or `cmd.exe` when it is empty, and `SysProcAttr.CmdLine` is the
      program in double quotes followed by ` /d /s /c "<text>"`, so that the text reaches `cmd.exe` as it is; it calls
      `childprocess.NoWindow` (R16, R30)
    * `shell_other.go` (`!linux && !windows`): `/bin/sh -c <text>` with no process group, for the build only
    * `streak.go`: `streak{failing bool}` with `observe(stderr io.Writer, name string, result outcome)`, which writes
      `metric <name>: <cause>` at the first failure and `metric <name>: read again` at the first value after one, as
      `monitorOutage` does (R23)
    * `set.go`: `Set`, made by `NewSet(definitions []config.Metric, sysfsRoot, directory string, now func() time.Time,
      stderr io.Writer)`, is the source of `run` and implements `sensors.Source`. It holds the metrics in the order of
      the file, each with its definition, its last value, its streak and, for a command, the start of its last run,
      a run in progress with its cancel and a channel of capacity 1 for its outcome. `Read` touches only the wanted
      metrics. A `file` or `hwmon` metric is read at once and observed. A `command` metric first takes the outcome of
      a run that finished, with a receive that does not block, observes it and keeps its value or clears it; it then
      starts a run in a goroutine when no run is in progress and none started within `Every` by `now`, and gives the
      value of the last finished run. The goroutine runs `runCommand` under a context of the set and counts in a
      `sync.WaitGroup`. `Replace(definitions)` keeps each metric whose definition is `==` to a new one, makes the
      others anew, and cancels the run in progress of each metric that is not kept, whose outcome nobody reads.
      `Close()` cancels the context of the set and waits for the group. Only the goroutine of the feeder calls `Read`,
      `Replace` and `Close`, so the state needs no lock
    * `once.go`: `ReadOnce(stop context.Context, definitions []config.Metric, sysfsRoot, directory string) []Result`,
      for `sensors`, reads each metric in the order of the file and runs each command, waiting for it up to its time
      limit. `Result{Name, Unit string; Value float64; Cause string}`. It gives nothing for a run that `stop` ended
* **`run`** makes the set in `serveRun` with the directory of the config file, `current.env.now` and the `stderr` of
  `run`, puts it after the other sources, after the monitor on Windows, keeps it in the field `custom` of the
  `feeder`, calls `custom.Replace` in `reload` when a config is applied, and calls `Close` once `feed` returned, before
  the transport is closed
* **`sensors`** replaces `monitorURL` with `sensorsConfig(named string) (path string, settings config.Config, found
  bool)`. A name given with `--config` is read as it is. Without it the path is that of `configPath("")`; a config
  directory that cannot be found, or a default file that does not exist, gives no settings and no line. A file that
  cannot be read, a named one included, or that is not valid gives one line, the error of `readNamedFile` or of
  `configFileError`, and no settings. The monitor takes the URL of the settings, or the default. The custom metrics
  are read with `ReadOnce` after `ReadAll`, under the same stop, and an interrupted reading fails as today. The lines
  are written in this order: the line of the config, the line of the monitor, then `metric <name>: <cause>` for each
  custom metric with no value, in the order of the file; then either the built-in readings in the order of the table
  followed by the custom ones in the order of the file, or, when neither has a value, the problem of the controller and
  `no sensor could be read`
* **The end-to-end tests** get `XDG_CONFIG_HOME` set to a new temporary directory in `environmentOf`, before the
  variables of the test, so that no `sensors` run reads the config of the user who runs the suite

## Files

* `internal/hwmonchips/chips.go`, `chips_linux_test.go`: new, the shared lookup of chips
* `internal/linuxsensors/hwmon.go`: `entriesNamed` over `hwmonchips.Named`
* `internal/childprocess/nowindow_windows.go`, `nowindow_other.go`, `nowindow_windows_test.go`: new
* `internal/nvidiasmi/source.go`: `childprocess.NoWindow` on the command
* `internal/sensors/metrics.go`, `metrics_test.go`: `IsWellFormedName`
* `internal/themecompile/document.go`, `document_test.go`: the form of R2 for a metric
* `internal/config/config.go`, `config_test.go`: the key `metrics`, `Metric`, `IsMetric`, the keys in the errors
* `internal/config/metrics.go`, `metrics_test.go`: new, the parsing of `metrics`
* `internal/custommetrics/*.go`: new, as the choices above list, with `number_test.go`, `streak_test.go`,
  `read_linux_test.go`, `command_linux_test.go`, `set_linux_test.go`, `once_linux_test.go` and
  `shell_windows_test.go`
* `internal/cli/runconfig.go`, `runconfig_test.go`: the check of R8
* `internal/cli/run.go`, `run_test.go`, `runreload_test.go`: the set in the feeder
* `internal/cli/commands.go`, `sensors_test.go`, `windowssensors_linux_test.go`, `service_test.go`: `sensors` with
  `--config` and custom metrics, and R8 in `service install`
* `e2e/main_test.go`: `XDG_CONFIG_HOME` in `environmentOf`
* `e2e/custom_metrics_test.go`: new, R35
* `README.md`, `docs/theme-folder.md`: R31 to R33

## Tasks

### 1. The lookup of hwmon chips, shared (refactoring)

- [x] Add `internal/hwmonchips/chips_linux_test.go`, with trees of `fakesensors`: chips of one name ordered by the
  device path whose numbers N are in the opposite order, two chips with one device path ordered by N, a chip without
  `device` after those with one, two chips without `device` by N, a chip of another name left out, a directory such
  as `hwmonX` left out, and no `class/hwmon` giving nothing (R15)
- [x] Add `internal/hwmonchips/chips.go` with `Chip` and `Named`, as the choices above say (R15)
- [x] Make `entriesNamed` of `internal/linuxsensors/hwmon.go` give the chips of `hwmonchips.Named` that have a device,
  replacing `entry` with `hwmonchips.Chip`, with no change to the tests of `linuxsensors` (R15)
- [x] Run the suite, check that the tests of `linuxsensors` and the end-to-end tests pass unchanged, and commit:
  `feature-0024 Share the lookup of hwmon chips by name`

### 2. No console window for a child process on Windows

- [x] Add `internal/childprocess` with `NoWindow` as the choices above say, and `nowindow_windows_test.go`, which
  checks that `NoWindow` sets the flag on a command with no `SysProcAttr` and keeps the flags that one already has.
  It is built by `GOOS=windows go vet` and run on Windows only (R30)
- [x] Call `childprocess.NoWindow` on the command of `nvidia-smi` in `internal/nvidiasmi/source.go` (R30)
- [x] Run the suite and commit: `feature-0024 Start nvidia-smi with no console window on Windows`

### 3. The name rule

- [x] In `internal/sensors/metrics_test.go`, add `TestIsWellFormedName`: accepted `water.temp`, `gpu.amd.temp`,
  `fan.pump`, `a`, `x1.y_2.z-3`, `nvme.0.temp`; refused the empty name, `1abc`, `_a`, `.a`, `a.`, `a..b`, `A.b`,
  `a.B`, `a b`, `a/b`, `a.°c`. Add `TestEveryNameOfTheTableIsWellFormed`, over a name of each row as
  `TestUnitOfEachRow` makes them, numbered rows with 1 and 10 (R2, R34)
- [x] Add `IsWellFormedName` to `internal/sensors/metrics.go` (R2)
- [x] Run the suite and commit: `feature-0024 Add the rule of a well-formed metric name`

### 4. The theme compiler takes a custom metric

- [x] In `internal/themecompile/document_test.go`, make the case "metric misspelt" use `cpu..temp` and add a case
  `CPU.temp`, both refused with the existing text, and add a test that a theme naming `water.temp` compiles and that
  `Result.Tags` gives it a tag (R9, R29, R34)
- [x] In `readNumberKeys`, check the metric with `sensors.IsWellFormedName` (R9)
- [x] Run the suite and commit: `feature-0024 Let the theme compiler take a well-formed metric name`

### 5. The key metrics

- [x] Add `internal/config/metrics_test.go`: a valid definition of each form with its defaults (`Scale` 1, `Offset` 0,
  `Unit` empty, `Index` 1, `Every` 10 seconds) and with every key given; the order of the file kept; each error of
  the list in the choices above, with `nvme.9.temp` and `cpu.temp` refused as built in, `index` 0, `-1`, `1.5`,
  `"2"` and `index` without `hwmon`, `every` of `500ms` and of `x`, `scale` of `.inf`, `.nan` and `"1"`, a `unit`
  with a tab and one with a line feed, an `input` of `..` and of `a/b`, a relative `file`, and a definition that is
  a scalar. Add to `config_test.go` the new list of keys in the error of an unknown key and of a root that is not a
  mapping, `tags` taking a defined custom name and refusing an undefined one, and `IsMetric` for a built-in name, a
  defined one and an undefined one (R1 to R7, R10, R34)
- [x] Add `internal/config/metrics.go` with `Metric`, `DefaultEvery`, `MinimumEvery` and `parseMetrics`; in
  `config.go` read the key `metrics` before `tags`, add it to both lists of keys, give `parseTags` the definitions,
  add `Metrics` to `Config` and `IsMetric` (R1 to R7, R10)
- [x] Run the suite and commit: `feature-0024 Read the key metrics of the config file`

### 6. The reading of custom metrics

- [ ] Add `internal/custommetrics/number_test.go`: `42000\n`, `temp: 41.5 C`, `-3`, `+7`, `.5` giving 5, `1.2.3`
  giving 1.2, `12 34` giving 12, `1e3` giving 1, `1,5` giving 1, `NaN`, `Inf` and the empty text giving none, and
  a run of 400 digits giving none; and the value of R12 with `scale` 0.001, a negative `offset`, and values of
  -5, 0.4, 51.5 and 70000 sent by `sensors.ToValue` as 0, 0, 52 and 65535 (R11, R12, R34)
- [ ] Add `internal/custommetrics/read_linux_test.go`: a `file` metric over a temporary file, one that is missing and
  one that holds no number, with their causes; `hwmon` metrics over a tree of `fakesensors` with inputs written as
  files of the chip, as `fan2_input`, read again after the chip changed its number; two `amdgpu` chips whose numbers
  are in the opposite order of their device paths, read with `index` 1, 2 and without the key; `index` 3 with two
  chips and with none, with the causes `2 hwmon chips named amdgpu, fewer than index 3` and
  `0 hwmon chips named amdgpu, fewer than index 3`; a chip without `device` read after one with it; an input that is
  missing (R14, R15, R23, R34)
- [ ] Add `internal/custommetrics/command_linux_test.go`, with fake commands written as shell scripts in temporary
  directories: a value; a nonzero exit with and without a line on standard error; an exit of 0 with no number; a
  command that runs in the directory given, checked with `pwd`; a pipeline; a run past a limit of 300 milliseconds
  that starts a child with `(sleep 2; : > <marker>) &` and holds its output, which ends within the limit and a
  second, after which the marker is not there 2.5 seconds later; a `stop` done during a run giving false; a missing
  directory giving the cause of a command that could not start (R16, R17, R18, R34)
- [ ] Add `internal/custommetrics/streak_test.go`: one line at the first failure, none at the second, one line at the
  value after, none at the next value, and a new streak after that (R23, R34)
- [ ] Add `internal/custommetrics/set_linux_test.go`, with a fake clock: a read that does not wait for a command that
  sleeps 5 seconds and gives no value, then the value once the run finished; no second run within `every` and one
  after it; no second run while one is in progress; a failed run that clears the value with its line; a metric that
  is not wanted not read and its command not started; `Replace` keeping the value, the streak and the run of an
  unchanged definition, starting a changed one with no value and no streak, and killing the run of a changed and of a
  removed one, whose outcome is not taken; `Close` killing a run in progress and returning within a second (R19, R21,
  R22, R23, R28, R34)
- [ ] Add `internal/custommetrics/once_linux_test.go`: the three forms read in the order of the file with their units,
  each command waited for, a failing metric giving its cause, and a stop done during a run giving nothing (R20, R24)
- [ ] Add `internal/custommetrics/shell_windows_test.go`, which checks the program, the `CmdLine` and the flag of
  `shellCommand` with `ComSpec` set and unset. It is built by `GOOS=windows go vet` and run on Windows only (R16, R30)
- [ ] Add the files of `internal/custommetrics` as the choices above say (R11 to R23, R28, R30)
- [ ] Run the suite and commit: `feature-0024 Read custom metrics from a file, a chip or a command`

### 7. Custom metrics in run

- [ ] In `internal/cli/runconfig_test.go`, add: a theme folder naming a defined custom metric loads, with its tag; one
  naming an undefined custom metric is refused with the text of R8; one naming `nvme.9.temp` with no definition loads
  (R8, R29, R34)
- [ ] In `internal/cli/runreload_test.go`, add: a reload whose theme names an undefined metric is refused with its line
  and the running config kept; a reload that adds a custom metric to the theme and its definition is applied (R8,
  R28, R34)
- [ ] In `internal/cli/run_test.go`, add with `tags` and a `file` metric over a temporary file: its value sent to its
  tag through `ToValue`, a file that loses its number leaving the tag unsent with one line, and the line of a metric
  read again; with a `command` metric whose script sleeps, the clock and the other values sent in the rounds before
  its first run finished (R12, R13, R19, R23, R29, R34)
- [ ] In `internal/cli/service_test.go`, add that `service install` refuses a config whose theme names an undefined
  metric, before any call to `systemctl` (R8)
- [ ] In `loadConfig` of `internal/cli/runconfig.go`, refuse a metric of the compiled tags for which
  `settings.IsMetric` is false (R8)
- [ ] In `internal/cli/run.go`, make the set in `serveRun`, add it to the sources and to the `feeder` as `custom`, call
  `Replace` in `reload` and `Close` after `feed` (R19, R21, R22, R28, R29)
- [ ] Run the suite and commit: `feature-0024 Send custom metrics in run`

### 8. Custom metrics in sensors

- [ ] In `internal/cli/sensors_test.go`, replace `TestSensorsOnASetWithoutAMonitorNeverLooksForTheConfig` with tests
  that read the config: the default file with a `file` metric listed after the built-in ones with its unit, and a
  metric without `unit` ending its line with a tab; `--config` naming another file; a missing default file giving no
  line; a missing named file and a file that is not valid each giving one line that names it and the built-in
  metrics; a failing custom metric giving its line on standard error; only a custom metric with a value giving no
  `no sensor could be read`; neither giving it (R24 to R27, R34)
- [ ] In `internal/cli/windowssensors_linux_test.go`, add that `sensors --config <file>` reads
  `librehardwaremonitor_url` from the named file (R25)
- [ ] In `internal/cli/commands.go`, add the flag `--config` to `sensors`, replace `monitorURL` with `sensorsConfig`
  and read and write the custom metrics, as the choices above say (R20, R22, R24 to R27)
- [ ] Run the suite and commit: `feature-0024 List custom metrics in sensors`

### 9. The end-to-end tests

- [ ] In `e2e/main_test.go`, set `XDG_CONFIG_HOME` in `environmentOf` to a new temporary directory, before the
  variables of the test, and say why in its comment
- [ ] Add `e2e/custom_metrics_test.go`: a config with a `file` metric over a temporary file, a `hwmon` metric over a
  chip of the fake sysfs tree, read with `index` 2 among two chips of one name, and a `command` metric over a fake
  script that prints a number, each with a `unit` but one; `sensors --config` lists them after the built-in metrics
  of `runningTree`, in the order of the file; `run --config` with a theme folder whose number widgets name the three,
  through `themeSetup`, uploads the theme and sends each value to the tag the compiler gave it (R29, R35)
- [ ] Run the suite and commit: `feature-0024 Cover custom metrics end to end`

### 10. The documents

- [ ] In `README.md`, add the section on custom metrics after "Windows: the sensors of the machine", with everything
  R31 lists and its four example blocks, the command that lists the chips being:
  `for chip in /sys/class/hwmon/hwmon*; do printf '%s %s\n' "$(cat "$chip/name")" "$(readlink -f "$chip/device")"; done | LC_ALL=C sort`
  (R31)
- [ ] In `README.md`, update the line of `sensors` in the list of commands, the section "Showing sensor values on the
  screen" for `sensors`, the keys and the reload of the config of `run`, and the Windows section, as R32 says (R32)
- [ ] In `docs/theme-folder.md`, update the paragraph on the names of the metrics as R33 says (R33)
- [ ] Commit: `feature-0024 Document custom metrics`

### 11. The suite

- [ ] Run the five commands of `AGENTS.md` and check that each succeeds (R36)

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [ ] `npm run check-changes` passes

## Commit Points

- [ ] `feature-0024 Share the lookup of hwmon chips by name`
- [ ] `feature-0024 Start nvidia-smi with no console window on Windows`
- [ ] `feature-0024 Add the rule of a well-formed metric name`
- [ ] `feature-0024 Let the theme compiler take a well-formed metric name`
- [ ] `feature-0024 Read the key metrics of the config file`
- [ ] `feature-0024 Read custom metrics from a file, a chip or a command`
- [ ] `feature-0024 Send custom metrics in run`
- [ ] `feature-0024 List custom metrics in sensors`
- [ ] `feature-0024 Cover custom metrics end to end`
- [ ] `feature-0024 Document custom metrics`
