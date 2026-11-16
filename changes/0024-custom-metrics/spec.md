# Custom metrics defined in the config file

## Context

The metrics of sensordeck are a fixed table built into it, in `internal/sensors`: `IsName` accepts a name when it is a
row of that table, `Unit` gives its unit, and the sources of Linux and Windows read only those names. Hardware the
project does not know cannot be shown: an AMD or Intel GPU on Linux, the fans and the sensors of a board, the
temperature of a water loop, or a value that a script or a web service gives. This unit lets the user define further
metrics in the config file of `run`, each read from a file, from a chip of hwmon, or from a command:

```yaml
metrics:
  water.temp:   {file: /sys/class/hwmon/hwmon5/temp1_input, scale: 0.001}
  gpu.amd.temp: {hwmon: amdgpu, input: temp1_input, scale: 0.001, unit: "°C"}
  fan.pump:     {hwmon: nct6799, input: fan2_input, unit: RPM}
  outside.temp: {command: "curl -s https://example.org/t | jq .temp", every: 5m}
```

What exists today and bears on the unit:

- A source reads on demand, only the metrics that `wanted` names, and gives nothing for a metric it cannot read. `run`
  reads in each round the metrics that are mapped, through the theme's tags or the key `tags`, and sends each value
  through `sensors.ToValue`, which rounds to the nearest whole number with halves away from zero and clamps to 0 to
  65535, because the screen takes a value as 16 bits unsigned (`docs/protocol.md`). A metric that has no value is not
  sent, and the tile keeps the value it was sent last
- The theme compiler and the key `tags` check a metric name with `IsName`. `docs/theme-folder.md` says that the
  compiler checks that a name is well formed, and not that the machine can read it, so that a theme can be shared
- `sensors` lists the readings in the order of the table of metrics, a name in no row last, as
  `<name>\t<value with one decimal>\t<unit>`. By R15 of unit 0004 it read no config. By R27 of unit 0018 it reads, on
  Windows only, the key `librehardwaremonitor_url` from the config file at the place where `run` reads it without
  `--config`; a file that is missing gives the default silently, and one that cannot be read or is not valid gives one
  line on standard error and the default. It has no `--config` flag
- A reload of the config while `run` runs (unit 0013) applies the theme and the tags. A change of `interval`,
  `commander_interval`, `commander_color` or `librehardwaremonitor_url` is not applied, and `run` writes that a restart
  applies it. A config that is not valid, or whose theme does not compile, is not applied, with one line
- The LibreHardwareMonitor source of unit 0018 reports an outage once: a line when a reading first fails, none while it
  keeps failing, and a line when it works again
- `internal/linuxsensors` finds the hwmon chips of a driver by the `name` file of each `hwmonN` directory, and orders
  them by the resolved path of their `device`, because the number N follows the order in which the drivers load and
  changes between boots
- `nvidia-smi` is the one child process that a source starts, on Linux and on Windows, with a time limit of 2 seconds.
  On Windows the background run of unit 0019 detaches from its console, so each child that is a console program gets a
  console of its own: the human saw on 2026-11-16 a console window open about every second while the background run
  ran, which is the run of `nvidia-smi` in each round

## Requirements

### The key `metrics`

- **R1** The config file has an optional key `metrics`, a mapping from a metric name to a definition, with at least one
  entry. A name given twice is refused, as in `tags`. The list of keys in the error of an unknown key, and in the error of
  a root that is not a mapping, names `metrics`
- **R2** A custom metric name is one or more segments joined by single dots, each segment one or more of the characters
  `a` to `z`, `0` to `9`, `_` and `-`, and the name begins with a letter. Every name of the table of metrics has that
  form
- **R3** A name that `IsName` accepts is a built-in metric and cannot be defined: the error names the metric and says
  that it is built in and cannot be replaced. This includes a numbered name with no hardware behind it on the machine,
  such as `nvme.9.temp`
- **R4** A definition is a mapping with exactly one of the source keys `file`, `hwmon` and `command`, and these other
  keys, each optional unless said:
    - `scale`, a number, 1 without the key
    - `offset`, a number, 0 without the key
    - `unit`, a string, empty without the key, for `sensors` only
    - `input`, with `hwmon` only, where it is required
    - `index`, with `hwmon` only, 1 without the key
    - `every`, with `command` only
- **R5** `file` is an absolute path, as the system the config runs on reads it. `hwmon` is the name of a chip, a
  non-empty string without `/`. `input` is the name of a file of the chip's directory, a non-empty string that is not `.`
  or `..` and holds no `/`. `index` is a whole number of 1 or more. `command` is a non-empty string. `scale` and `offset` are finite numbers. `unit` holds no
  tab, line feed or other control character. `every` is a duration of at least 1 second, such as `30s` or `5m`
- **R6** Each problem of `metrics` is refused with an error in the existing form, the config file named in front by
  the caller: `metrics:` followed by the metric name and the key at fault when there is one, and what was wanted. A
  file with several problems gives one of them, as today
- **R7** The key `tags` accepts a name that is built in or defined under `metrics`, and refuses any other with an
  error that names it
- **R8** Loading the config as `run` does, at its start, at a reload and in `service install`, refuses a metric of the
  theme that is neither built in nor defined under `metrics`, with an error that names the metric and the theme. A
  reload refused this way is not applied, as for any config that is not valid
- **R9** The theme compiler accepts a metric name that has the form of R2, and refuses any other as it does today. It
  does not know the config, so a theme that names a custom metric compiles on any machine
- **R10** The key `metrics` is valid on Linux and on Windows, and every source form is accepted on both

### The value of a custom metric

- **R11** The number of a source is the first match, in its text, of an optional `-` or `+`, one or more decimal digits,
  and optionally a `.` followed by one or more decimal digits. A text with no match, or whose match does not give a
  finite number, has no number. Exponents, a comma as the decimal mark and `NaN` or `Inf` are not numbers
- **R12** The value of a custom metric is its number multiplied by `scale`, plus `offset`. `sensors` writes that value
  with one decimal, as it writes a built-in one. `run` sends it through `sensors.ToValue`, as it sends a built-in one, so
  a fraction is rounded half away from zero, a value below 0.5 is sent as 0 and a value above 65535 as 65535
- **R13** A custom metric that has no value is not sent, so its tile keeps the value it was sent last, as for a built-in
  one

### The sources

- **R14** A `file` metric reads the file at each reading and takes its number. A file that cannot be read, or holds no
  number, gives no value
- **R15** A `hwmon` metric looks, at each reading, among the directories `hwmonN` of `class/hwmon` under the sysfs root,
  for those whose `name` file holds the chip name, without its trailing line feed. It orders them by the resolved path of
  their `device`, compared by bytes, the order of the numbered built-in metrics, with a chip that has no `device` after
  those that have one, by N, and takes the chip at the place `index` in that order, counted from 1. It reads the file
  `input` of that directory and takes its number. Fewer chips of the name than `index`, none among them, or an input
  that cannot be read or holds no number, gives no value
- **R16** A `command` metric runs its command through the shell of the system: `/bin/sh -c <command>` on Linux, by that
  absolute path, and on Windows the program of the environment variable `ComSpec`, or `cmd.exe` when it is not set, with
  the command line `/d /s /c "<command>"` given as it is, so that the quoting of the command is the user's. The command
  runs in the directory of the config file, with the environment of sensordeck and no standard input
- **R17** A run of a command that exits with status 0 and whose standard output has a number gives the value of the
  metric. A run that cannot start, exits with another status, outlasts its time limit or gives no number gives the metric
  no value. The value of a metric is that of its last finished run
- **R18** A run's time limit is 10 seconds, or `every` when that is shorter. At the limit the run is killed, on Linux
  with every process of its process group, which the command is started as the leader of, so that the programs of a
  pipeline end with it. A killed run does not keep a reading waiting more than half a second for a child that holds its
  output open, as for `nvidia-smi`
- **R19** In `run` a command never holds up a round. A reading starts a run of the command when the metric is wanted,
  no run of it is in progress, and none started within the last `every`, which is 10 seconds without the key. The run
  goes on in the background, and the reading gives the value of the last finished run. The first rounds before a first
  run finished send no value for the metric
- **R20** `sensors` runs each command once and waits for it, up to its time limit, before it writes its lines
- **R21** When `run` ends, a run in progress is killed as at its time limit
- **R22** A custom metric is read only while it is wanted, as every source is: in `run` while it is mapped, and in
  `sensors` always

### Reporting failures

- **R23** In `run` a custom metric that gives no value writes one line to standard error at the first reading of a
  streak of such readings, none while the streak lasts, and one line when it gives a value again. The first line starts
  with `metric <name>: ` and gives the cause: the file or the input that cannot be read and the error, the number of
  chips of the name when it is less than `index` (zero when there is none), no number in the text, the exit status of the command with the first line of its standard error when it wrote
  one, a command that could not start, or the time limit it outlasted. The second line says that the metric is read
  again and names it. For a command, a streak begins and ends with finished runs, and a run in progress changes nothing
- **R24** `sensors` writes the line of R23 for each custom metric that gives no value, and lists the others

### The commands

- **R25** `sensors` takes the flag `--config <path>`, as `run` does, and reads the config file it names, or else the
  file at the place where `run` reads it without the flag, on every system. It reads the custom metrics and, on
  Windows, `librehardwaremonitor_url` from it
- **R26** For `sensors` a default file that does not exist gives no custom metrics and no line. A file that cannot be
  read or is not valid, the default one or the one named, gives one line on standard error that names the file, and
  `sensors` lists the built-in metrics, as R27 of unit 0018 does for the URL
- **R27** `sensors` lists the custom metrics after the built-in ones, in the order of the file, each as
  `<name>\t<value with one decimal>\t<unit>`, with the `unit` of the definition, so a metric without one ends its line
  with a tab. The message that no sensor can be read is written only when neither a built-in nor a custom metric has a
  value
- **R28** A reload while `run` runs applies a change of `metrics`, with the theme and the tags. A metric whose
  definition is unchanged keeps its last value, its streak of R23 and its run in progress. A metric that is new or whose
  definition changed starts with no value and no streak, and a run in progress of a definition that changed or was
  removed is killed and its outcome dropped
- **R29** A theme and a `tags` block use a custom metric as a built-in one: the compiler gives it a tag, and `run` sends
  its value to that tag

### Child processes on Windows

- **R30** On Windows every child process that sensordeck starts to read a metric, the custom commands and `nvidia-smi`,
  is created with the flag `CREATE_NO_WINDOW`, so that no console window opens, in the background run of unit 0019 or
  any other

### The documents

- **R31** The README has a section on custom metrics that gives the key `metrics`, the name rule of R2 and R3, the three
  source forms and their keys, R11 and R12 with a note that the screen shows whole numbers from 0 to 65535, so that a
  negative value shows 0 and tenths are shown by a `scale` ten times larger, the shell of each system and the directory a
  command runs in, `every` and the time limit, the lines of R23, and that a reload applies `metrics`. It has these
  examples, each a config block:
    - an AMD GPU on Linux through `hwmon: amdgpu`: the edge temperature `temp1_input` and the junction temperature
      `temp2_input` in °C with `scale: 0.001`, the power `power1_average` in W with `scale: 0.000001` and the fan
      `fan1_input` in RPM, with how to find a chip's name (`cat /sys/class/hwmon/hwmon*/name`) and its inputs
    - `index` for a machine with two chips of one name, with the case of a Ryzen 7000 or 9000 desktop processor, whose
      integrated graphics has an `amdgpu` chip of its own beside that of an AMD card. The README says that the chips
      are counted from 1 in the order of their device paths, and gives a shell command that lists every chip's name and
      resolved device path in that order, for each name, such as a loop over `/sys/class/hwmon/hwmon*` that prints
      `name` and `readlink -f` of `device`, piped to `LC_ALL=C sort`, and that a chip with no device path comes last.
      It says that a GPU's device path ends in its PCI address, which `lspci -s <address>` names, so that the card and
      the processor's graphics are told apart. It shows the card read with the `index` that the command gives it
    - a fan of the board through the chip of its controller, such as `nct6799`, in RPM
    - a command, such as the `curl` and `jq` example of the context, with `every`
- **R32** The README's section on `sensors` says that it lists custom metrics after the built-in ones and takes
  `--config`; its section on the config file of `run` lists `metrics` among the keys and its reload paragraph says a
  reload applies it; its Windows section says that `sensors` reads the config named with `--config` too, and that the
  commands run through `cmd.exe` with no console window; the list of commands shows the flag of `sensors`
- **R33** `docs/theme-folder.md` says that a metric name is a built-in one or a custom one defined in the config of
  `run`, that the compiler checks the form of R2 only, and that `run` refuses a theme whose metric is neither built in
  nor defined

### Tests

- **R34** Unit tests on Linux cover:
    - the name rule of R2 and R3, with names accepted and refused, a built-in name refused with its error, and every
      name of the table having the form of R2
    - each error of R1 to R7, and a valid definition of each source form with its defaults
    - R8 at the start and at a reload, and R9 with a theme naming a custom metric and one naming a malformed name
    - the number of R11 on texts with leading text, signs, fractions, several numbers, no number, exponents and a number
      too large for a float, and R12 with `scale` and `offset`, negative, fractional and too large values sent as R12
      says
    - `file` metrics with temporary files, and `hwmon` metrics with a temporary sysfs tree whose chip changes its number,
      two chips of one name with the order of R15 whose numbers N are in the opposite order of their device paths, read
      with `index` 1, 2 and without the key, `index` 3 with two chips and with none, and its line of R23, a chip
      without `device`, and the errors of `index` that is 0, negative, not whole, or given without `hwmon`
    - `command` metrics with fake commands written as shell scripts in temporary directories: a value, a nonzero
      exit, no number, a run past its time limit that is killed together with a child it started, the working
      directory, a round that does not wait for a run in progress, and `every` with a fake clock
    - the lines of R23 for a streak and its end, and R28 for an unchanged, a changed and a removed definition
    - `sensors` with the default config, `--config`, a missing default file, and a file that is not valid
- **R35** End-to-end tests run the executable on Linux with a config that defines a `file` metric over a temporary file,
  a `hwmon` metric over the fake sysfs tree, and a `command` metric over a fake script: `sensors` lists the three after
  the built-in metrics with their units, and `run` sends their values to the tags of a theme that names them
- **R36** The five commands of the suite in `AGENTS.md` succeed

## Unverified until the human runs it

Each is open until the human checks it on Windows. The unit does not depend on it, and a wrong one is corrected by a
later unit.

- That the background run opens no console window any more, for `nvidia-smi` and for a custom command
- That a command such as `powershell -NoProfile -Command "..."` and one with quotes run through `cmd.exe /d /s /c` as
  typed
- That a command killed at its time limit leaves no program of it running, which R18 does not promise on Windows

## Decisions

- **D1 Custom names need no prefix.** A name such as `water.temp` reads naturally in a theme and in `sensors`, and a
  prefix such as `custom.` would be in every name for the sake of a clash that may never come. The clash it would guard
  against, a later release adding a built-in name the user defined, is refused by R3 with an error that names the
  metric, and is mended by renaming it in two files. Rejected: a required prefix, for that cost on every name
- **D2 A custom metric cannot shadow a built-in one.** The orchestrating session preferred it: a theme's `gpu.temp` would
  otherwise mean what the config of each machine says it means, and a definition left over in a config would silently
  replace a reading that works. The error names the metric, so the fix is a new name. An explicit way to replace a
  built-in one is left to the human under "Beyond the request"
- **D3 The compiler checks the form, and `run` checks the definition.** The compiler knows no config, and a theme is
  meant to be shared (`docs/theme-folder.md`), so it cannot ask for a definition. `run` reads the theme and the
  definitions from the same file, so a metric of the theme that is neither built in nor defined is a mistake it can
  name, such as a typo in a custom name, which would otherwise be a tile that never shows a value. A built-in name the
  machine lacks stays allowed, as today, because the machine and not the file decides it. Rejected: treating an
  undefined custom name as a metric that cannot be read, for the silent tile; and checking the definitions in `theme
  use`, which sets a theme before the user may have written its metrics, and whose result `run` checks anyway
- **D4 The `hwmon` form is in this unit.** The number N of `hwmonN` changes between boots on a usual machine, so a
  `file` path through it reads another chip or nothing after a reboot, in a service that nobody watches. The AMD GPU,
  the first case of the request, is behind such a path. The lookup by `name` and the order by device already exist for
  the built-in metrics, so the form costs a definition key and a lookup. `index` chooses among several chips of one name,
  because every Ryzen 7000 and 9000 desktop processor has integrated graphics with an `amdgpu` chip of its own, so that
  the AMD GPU example is ambiguous on most AMD hosts without it; the human's 9950X3D has one. It counts in the order of
  the device paths because that order follows the hardware and not the boot, and it starts at 1 as the numbered
  built-in metrics do. Fewer chips than `index` is a metric with no value and a line of R23, and not an error of the
  config, because the chips present are known only when the machine is read, and a config is shared between systems.
  Rejected: choosing by the device path itself, which is longer to write and to get right; and listing the chips in
  `sensors`, which would add a second kind of output to a command that lists readings
- **D5 `offset` is in this unit.** A file source has no other way to shift a value, and the cases are real: the
  calibration of a cheap probe of a water loop, and a value in kelvin. It is one key with a default of 0. A command can
  shift its own output, but the key costs nothing there either
- **D6 Commands run through the shell.** The request's own example is a pipeline, which needs a shell, and a single
  string is what a user copies from a terminal. `/bin/sh` is named by its absolute path because the `PATH` of a service
  may be short, and the end-to-end tests set their own. On Windows `cmd.exe` is the shell every machine has; its command
  line is given verbatim because the quoting that Go applies to an argument is not the one `cmd.exe` reads, and `/s`
  keeps the user's quotes. A user who wants PowerShell writes it as the command. The config file belongs to the user and
  only the user's own processes read it, so running its commands as the user gives no one rights they did not have.
  Rejected: an argument list, which cannot express the pipeline, left under "Beyond the request" as a second form
- **D7 Commands run in the background, and a round takes the last value.** A command may take seconds, and a round of
  `run` must not wait for it, or the clock and the other tiles stall. The value of the last finished run is what the
  metric is until the next one finishes. A failed run clears it, so that `sensors` and the lines of R23 never present an
  old value as current; on the screen it changes nothing, because the tile keeps its last value either way
- **D8 `every` is 10 seconds without the key, and at least 1 second.** A command costs a process and may reach a
  service on the web, so the default is a pace that still follows a local sensor and is not a request in every
  round to a service on the web. A user who wants a slower pace for the web, or each second for a local script, says so. One
  second keeps a typo such as `1ms` from starting a process in every round. `every` is not tied to `interval`: shorter
  than the interval it simply means each round
- **D9 The time limit is 10 seconds, or `every` when shorter.** One run of a command at a time, and a run never lasts
  past the time its next run is due, so a hung command cannot pile up. Ten seconds is ample for a request to a service.
  The process group on Linux is what makes the limit end a pipeline, whose shell alone would otherwise be killed and
  leave `curl` running, a process left behind at each run of a hung command. Windows has no process group to kill, and
  a job object is left under "Beyond the request"
- **D10 One rule of failure lines for every custom source.** The request asks for the rule of the LibreHardwareMonitor
  source for commands. A file or a chip that disappears is the same kind of outage for a service, and one rule for the
  three keeps the log quiet and tells when the metric works again
- **D11 A reload applies `metrics`.** A definition says what a tile shows, as the theme and the tags do, which a reload
  applies, and not how often or with which device `run` works, which is what the keys a restart applies have in common.
  It also keeps R8 consistent: a reload that adds a metric to the theme and its definition to `metrics` is applied as
  one. A restart on Windows means reinstalling the task, which is a high price for a scale. Unchanged definitions keep
  their state so that an edit of the theme starts no command and repeats no line
- **D12 `sensors` reads the config on every system and takes `--config`.** It must know the custom metrics to list
  them, and they are in the config. The default place is where `run` reads it, as `sensors` already does on Windows, and
  `--config` serves a user whose `run` reads another file, such as a service installed with one. A config that is not
  valid does not stop `sensors`, which is the command run while a config is written, as unit 0018 decided for the URL
- **D13 The values go through `sensors.ToValue`, as built-in ones do.** The screen takes 16 bits unsigned, and the rule
  that rounds and clamps a built-in value is the one rule there is. A negative custom value shows as 0; `offset` lets a
  user who needs one shift it into range. `sensors` writes the value before the rounding, so the user sees what was read
- **D14 The number is the first decimal number of the text.** A sysfs file holds one number with a line feed, and a
  command can be made to print one with `jq` or `awk`, so the first number is all the forms need. The grammar leaves out
  exponents and `NaN`, which no sensor file writes and which a theme could not show
- **D15 The Windows console fix is part of this unit.** The custom commands start children in the same way as
  `nvidia-smi`, so the flag belongs in the one place where sensordeck starts a child to read a metric, and setting it
  for the commands and not for `nvidia-smi` would leave that place with two behaviours. The bug the human saw on
  2026-11-16 is fixed by the same line. Rejected: a separate bug unit, which would touch the same code twice and leave
  the new commands opening windows until it lands
- **D16 Custom metrics are listed in the order of the file.** It is the order the user chose and is stable. The table
  order of the built-in metrics does not reach them, and `InTableOrder` already puts a name in no row last

## Non-goals

- Replacing a built-in metric, so that `gpu.temp` reads an AMD GPU in a built-in theme; see "Beyond the request"
- Finding hardware by itself: an AMD or Intel GPU, the fans and the board sensors are not added to the table of
  built-in metrics
- Parsing JSON, XML or other structured output; a command uses `jq` or another tool to print the number
- Showing decimals, negative numbers or text on the screen, which the protocol does not carry
- A cap on what a command writes, a check of what a command does, or a sandbox for it
- Units for anything but `sensors`; the unit of a custom metric does not reach a theme
- Ending, on Windows, the programs that a killed command started
- A check on the device or on Windows by the automated loop

## Beyond the request

1. Taken into the unit by the amendment recorded in `record.md`, as the key `index` of R4, R5, R15, R23, R31 and R34
   and D4. A key `index` for the `hwmon` form, which takes the *n*-th chip of the name in the order of R15. Cost: a
   key, its check and a test. Without it, a machine with two chips of one name, such as an AMD processor's graphics beside an AMD
   card, reads the first and uses a `file` path for the other, which may change between boots
2. A way to replace a built-in metric, such as a key `replaces: true` in a definition, so that the built-in themes show
   an AMD GPU as `gpu.temp`. Cost: the key, a rule for which source wins, and tests. Without it, a user copies a
   built-in theme and names the custom metric in it, as the README says for `minimal-dark-probes`
3. A second form of `command` as a list of arguments, run without a shell. Cost: the form, its check and tests. Without
   it, a command is a shell line, and a user who wants no shell writes `exec` in front
4. A key `timeout` for a command. Cost: a key and its check. Without it, a command that needs more than 10 seconds fails
   with the time limit in its line
5. Running a command on Windows in a job object that ends its children at the time limit. Cost: Windows calls behind a
   seam that the sandbox cannot run. Without it, a program that a killed command started on Windows may keep running
6. A `file` path with a glob pattern that must match one file, such as one through the stable PCI path of a device.
   Cost: the matching, the error of several matches and tests. Without it, a stable path is the `hwmon` form or a path
   without a variable part
