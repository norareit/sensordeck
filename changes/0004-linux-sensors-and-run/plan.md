# Linux sensors and the run command

## Approach

The unit adds four packages and extends three, in the order in which each builds on the one before:

* `internal/sensors`, new and not limited to a platform: the metric names and their units (R1, R2), the reading and
  the source (R3), and the conversion to a 16-bit value (R4)
* `internal/config`, new and not limited to a platform: the parsing and checking of the config file (R17, R18)
* `internal/fakesensors`, new, limited to Linux: test support that writes hwmon, cpufreq and proc trees and a fake
  `nvidia-smi` into temporary directories. It is a normal package and not a `_test.go` file because the tests of the
  Linux sources and the end-to-end tests both use it, as `internal/fakescreen` is (R28 to R30, R34, R35)
* `internal/linuxsensors`, new, limited to Linux: the sources of R5 to R12
* `internal/fakescreen`, extended: waiting for a count of reports (R33)
* `internal/cli`, extended: the third root, the commands `sensors` and `run`, and the files that choose the sources
  (R13 to R16, R19 to R25)
* `e2e`, extended: the cases of R34 and R35, and the comment of R36

Every Go file of `internal/fakesensors` and `internal/linuxsensors`, test files included, starts with
`//go:build linux`. A Windows build then leaves both packages out of `./...`, which the two Windows commands of the
suite prove (R14).

### The metrics

`internal/sensors/metrics.go` holds the one table of R1, as data, and everything that needs a name or a unit reads
it:

```go
// metricKind is one row of the table: a fixed name, or a numbered one made of prefix, number and suffix.
type metricKind struct {
	prefix, suffix string
	numbered       bool
	unit           string
}

var metricKinds = []metricKind{
	{prefix: "cpu.temp", unit: "°C"},
	{prefix: "cpu.ccd", suffix: ".temp", numbered: true, unit: "°C"},
	{prefix: "cpu.util", unit: "%"},
	// ... the twelve rows of R1, in its order
}

func IsName(name string) bool    // R2
func Unit(name string) string    // the unit of a name for which IsName is true
func CCDTemperature(number int) string   // "cpu.ccd<n>.temp"
func RAMTemperature(number int) string   // "ram.<n>.temp"
func NVMeTemperature(number int) string  // "nvme.<n>.temp"
```

and constants for the eight fixed names (`CPUTemperature = "cpu.temp"` and so on), which the sources use. A
numbered kind matches when the name has its prefix and its suffix and what lies between is a number by R2: one or
more decimal digits, the first not `0`. `IsName` and `Unit` are the same walk over `metricKinds`, so a name has a unit
exactly when it is a name.

`internal/sensors/source.go`:

```go
// Reading is a metric that has a value now. Its unit is Unit(Name).
type Reading struct {
	Name  string
	Value float64
}

// Source reads the metrics of one source at the moment it is called. It reads a metric only when wanted gives true
// for its name, and gives a reading for each wanted metric that is not unavailable, in the order of R1. Being
// unavailable is never an error.
type Source interface {
	Read(stop context.Context, wanted func(name string) bool) []Reading
}

// ReadAll asks each source in turn and joins what they give.
func ReadAll(stop context.Context, sources []Source, wanted func(name string) bool) []Reading
```

The `wanted` function is what lets a round of `run` read only the mapped metrics: a numbered metric gets its name
only once the source has found its entry, so the source asks for the name and not the caller. `stop` ends the run of
`nvidia-smi` early when `run` is stopped (R23).

`internal/sensors/convert.go`: `func ToValue(value float64) uint16` is `math.Round`, which rounds halves away from
zero, then clamping to 0 and 65535 (R4). An infinite value clamps like any other.

### The config

`internal/config/config.go`:

```go
// Mapping sends the value of a metric to a tag.
type Mapping struct {
	Name string
	Tag  uint8
}

type Config struct {
	Interval time.Duration
	Mappings []Mapping // ordered by tag, ascending
}

const DefaultInterval = time.Second
const MinimumInterval = 100 * time.Millisecond

// Parse reads the content of a config file. Its errors are the texts of R18 after the prefix, which the caller adds.
func Parse(content []byte) (Config, error)
```

It imports `go.yaml.in/yaml/v3` at v3.0.6 and works on a `yaml.Node` (D18). The trial for this plan, against v3.0.6,
showed what the steps below rely on:

* `yaml.Unmarshal` into a `yaml.Node` does not report a key given twice. Decoding a mapping node into a
  `map[string]yaml.Node` with `node.Decode` does, with the text `yaml: unmarshal errors:` and a line
  `line 2: mapping key "tags" already defined at line 1`. So each of the two mappings, the file and `tags`, is first
  decoded into such a map, only for that error, before its pairs are walked
* An empty file, or one of blanks or comments only, gives a node of kind 0. Otherwise the node is a document whose
  `Content[0]` is the root
* `ShortTag()` is `!!int` for `1`, `-1`, `0x10`, `0o10`, `1_000`, `+5` and also `010`, which the package reads as
  octal 8; it is `!!float` for `1.0` and for `99999999999999999999`, `!!str` for `"1"`, and `!!null` for a key with no
  value. `node.Decode` into an `int64` gives the value of an `!!int`

`Parse` checks in this order, and returns at the first problem:

1. `yaml.Unmarshal(content, &node)`: its error, as its text
2. a node of kind 0, or a root that is no `MappingNode`: `want a mapping with the keys interval and tags`
3. the root decoded into `map[string]yaml.Node`: its error, as its text
4. the pairs of the root in file order: a key whose `Value` is neither `interval` nor `tags` gives
   `unknown key "<key>", the keys are interval and tags`, written with `%q`
5. `tags`: missing, no `MappingNode`, or with no pair: the text of R18. Then decoded into `map[string]yaml.Node` for a
   key given twice. Then its pairs in file order: a key that `sensors.IsName` refuses; a value that is no
   `ScalarNode` with the short tag `!!int`, or whose decoding into an `int64` fails; a value below 0 or above 255,
   whose text names the tag as written in the file, `node.Value`. Then the pairs sorted by tag and then by name with
   `slices.SortFunc` on a slice `Parse` made: the first two neighbours with the same tag give the text of R18, the
   smaller name first
6. `interval`: missing gives `DefaultInterval`. A value that is no `ScalarNode` is taken as the text `""`. The text is
   read with `time.ParseDuration`; a failure gives the text of R18 with the text in `%q`, and a duration below
   `MinimumInterval` gives `interval: <text> is shorter than 100ms`, the text as written

The `Mappings` of the result are the sorted slice of step 5.

### The Linux sources

Each source reads its files with `os.ReadFile` and hands the content to a pure function that parses it, so that the
parsing has unit tests on plain strings as well as the tests on trees that R28 to R30 ask for. Every source returns
no reading for a metric whose file is missing or malformed, and never an error (R3).

`internal/linuxsensors/hwmon.go`:

```go
// entry is a hwmon device that R5 keeps.
type entry struct {
	number int    // N of hwmonN
	name   string // the content of name, without the trailing newline
	path   string // <sysfs root>/class/hwmon/hwmonN
	device string // <entry>/device resolved with filepath.EvalSymlinks
}

// entriesNamed gives the entries with the driver name, in the order of R5.
func entriesNamed(sysfsRoot, driver string) []entry
```

`entriesNamed` lists `<sysfs root>/class/hwmon` with `os.ReadDir` and takes the names that are `hwmon` followed by one
or more decimal digits, whatever the type of the entry, because on a real system the entries are symbolic links. A
missing directory, or any other error of the listing, gives no entries. It sorts with `slices.SortFunc` by the device
path compared as strings and then by the number (R5, D4). Each reading lists the directory anew (R3).

`labelledSensors(entryPath string) []labelled`, with `labelled{number int; label string}`, globs `temp*_label`, keeps
the files whose middle is decimal digits, reads each label with its trailing newline removed by
`strings.TrimSuffix(content, "\n")`, passes over a label that cannot be read, and orders them by the number. A
sensor's input is `temp<K>_input`, parsed by `parseMillidegrees(content string) (float64, bool)`: the trailing
newline removed, then an optional `-` and one or more decimal digits and nothing else, read with `strconv.ParseInt`
and divided by 1000 (R6). A `+` is refused.

Three sources live in this file:

* `cpuTemperatures{sysfsRoot}`: the first `k10temp` entry. The sensor whose label is `Tctl` gives
  `sensors.CPUTemperature`; a label `Tccd<n>` whose `<n>` makes `sensors.CCDTemperature(n)` a name by R2 gives that
  metric. When two sensors carry one label, the one with the lower number counts. The readings are `cpu.temp` first,
  then the CCDs with *n* ascending as numbers (R1, R6)
* `numberedTemperatures{sysfsRoot, driver, name func(int) string, input func(entryPath string) string}`, twice: for
  `spd5118` with `sensors.RAMTemperature` and the input `temp1_input`, and for `nvme` with
  `sensors.NVMeTemperature` and the input of the sensor whose label is `Composite`, an empty path when there is none.
  The *n* of an entry is its position in `entriesNamed`, counted from 1, before its input is read, so that an entry
  whose input fails keeps its number (R7)

A metric is read only when `wanted` gives true for its name.

`internal/linuxsensors/procstat.go`: `cpuUtilisation{procRoot string; wait func(time.Duration); previous *sample}`
with `sample{total, idle uint64}`. `parseStat(content string) (sample, bool)` takes the first line that starts with
`cpu `, splits it with `strings.Fields`, needs at least eight counters after `cpu`, refuses the line when any of its
counters is not one or more decimal digits, and sums the first eight into `total` and the fourth and fifth into
`idle`. `Read`, when `cpu.util` is wanted: with no previous sample it takes one, and when that fails it gives nothing
and keeps nothing; otherwise it calls `wait(200 * time.Millisecond)` and takes the second. With a previous sample it
takes one only. The new sample becomes the previous one whenever it was made. The value is
`100 * (growthTotal - growthIdle) / growthTotal`, the growths computed as `float64` differences so that a counter
that went back cannot wrap, clamped to 0 and 100, and a growth of the total of 0 or less gives no reading (R8, D7).
The source is a pointer and keeps its sample from one reading to the next, which is what makes only the first round
of `run` wait (R21).

`internal/linuxsensors/cpufreq.go`: `cpuClock{sysfsRoot}` globs
`<sysfs root>/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq`, keeps the paths whose directory after `cpu` is
decimal digits only, reads each with its trailing newline removed as digits only with `strconv.ParseUint`, divides by
1000 and gives the highest, or nothing when none is left (R9, D6).

`internal/linuxsensors/meminfo.go`: `memory{procRoot}`, with `parseMeminfo(content string, key string) (uint64, bool)`
taking the line whose first field is `<key>:` and needing exactly three fields, the second decimal digits and the
third `kB`. `ram.used` is `(MemTotal - MemAvailable) / 1048576` computed in `float64`, and `ram.total` is
`MemTotal / 1048576`, each given only when its lines are there and well formed and when it is wanted (R10, D8).

`internal/linuxsensors/nvidiasmi.go`: `gpu{timeLimit time.Duration}`. When none of the four GPU metrics is wanted it
does nothing, so the program does not run. Otherwise it makes a context with `context.WithTimeout(stop, timeLimit)`
and runs `exec.CommandContext(context, "nvidia-smi", <the two arguments of R11>)`, made anew for every reading so that
`PATH` is searched anew (R12). Standard output goes to a `bytes.Buffer`, standard error is discarded, and
`command.WaitDelay` is 500 milliseconds: once the context ends the process is killed, and `Wait` gives up on a pipe
that a child still holds after that delay. The trial for this plan, with a limit of 100 milliseconds and `sleep 5 &
wait` as the program, returned after 600 milliseconds, within the bound of R12. Any error of `Run` gives nothing.
`parseGPULine(output string) (fields [4]string, ok bool)` takes the text before the first newline, splits it at
commas and trims spaces with `strings.Trim(field, " ")`; other than four fields gives no reading. Each field that
matches `^[0-9]+(\.[0-9]+)?$` is read with `strconv.ParseFloat` and gives its metric when wanted (R11, D9).

`internal/linuxsensors/sources.go`:

```go
// Sources gives the sources of R1 in its order.
func Sources(sysfsRoot, procRoot string) []sensors.Source
```

in the order `cpuTemperatures`, `cpuUtilisation` with `time.Sleep`, `cpuClock`, `gpu` with 2 seconds, `memory`, the
`spd5118` and the `nvme` sources, which is the order of R1.

### The fake trees

`internal/fakesensors/fakesensors.go`:

```go
type Tree struct {
	SysfsRoot string // given, so that the end-to-end tests can share the root of the fake screen
	ProcRoot  string // <temp>/proc
	// unexported: the test
}

func New(test testing.TB, sysfsRoot string) *Tree

// Sensor is temp<Number>_input with Input and, when Label is not empty, temp<Number>_label with Label. Both are written
// with a trailing newline.
type Sensor struct {
	Number       int
	Label, Input string
}

// Hwmon makes <SysfsRoot>/class/hwmon/hwmon<number> with name and the sensors. With device not empty, <entry>/device
// is a link to <SysfsRoot>/devices/<device>, which it makes. It gives the path of the entry.
func (tree *Tree) Hwmon(number int, name, device string, sensors ...Sensor) string

func (tree *Tree) CPUFrequency(cpu int, content string) // devices/system/cpu/cpu<cpu>/cpufreq/scaling_cur_freq
func (tree *Tree) Stat(content string)                  // <ProcRoot>/stat, as given
func (tree *Tree) Meminfo(content string)               // <ProcRoot>/meminfo, as given

// NvidiaSMI makes a new temporary directory with an executable nvidia-smi, "#!/bin/sh\n" and the script, and gives
// the directory, for the PATH of a test.
func NvidiaSMI(test testing.TB, script string) string
```

Every failure is `test.Fatalf`. A test that needs a file missing removes it after the call.

### The commands

`environment` in `internal/cli/main.go` gains five fields, all filled in `Main` and by the harness of the unit tests:

```go
procRoot      string
userConfigDir func() (string, error)                                    // os.UserConfigDir
newSources    func(sysfsRoot, procRoot string) ([]sensors.Source, error)
notifyStop    func(parent context.Context) (context.Context, context.CancelFunc)
wait          func(stop context.Context, duration time.Duration) bool   // false when stop ended the wait
```

`Main` reads `SENSORDECK_PROC_ROOT` beside the other two with `valueOrDefault` and the default `/proc` (R13).
`notifyStop` is `signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)`, called by `run` alone, so that the
other commands keep the default handling of Ctrl+C. `wait` is a new function `waitFor` in `main.go` that selects on a
timer and `stop.Done()`.

`internal/cli/sources_linux.go`, `//go:build linux`, and `internal/cli/sources_other.go`, `//go:build !linux`, do
nothing but choose the sources: the first gives `linuxsensors.Sources(sysfsRoot, procRoot), nil`, the second
`nil, unsupported()`, the function of `device_other.go` (R14). They are the only files of the commands that import
`internal/linuxsensors`, and the layout test checks it.

`sensors` goes into `commands.go` beside the other commands. It calls `newSources`, then `sensors.ReadAll` with
`context.Background()` and a `wanted` that is always true, and writes each reading as
`name + "\t" + strconv.FormatFloat(value, 'f', 1, 64) + "\t" + sensors.Unit(name)`. With no reading it writes
`no sensor could be read` to standard error. It never calls `openTransport` and reads no config (R15).

The reading of a file and the unwrapping of its `*fs.PathError`, which `readTheme` does now, becomes
`readNamedFile(description, path string) ([]byte, error)` in `commands.go`, giving
`cannot read the <description> <path>: <error>`. `readTheme` calls it with `theme file`, and `run` with
`config file` (R16). This is its own commit, before the behaviour changes.

`internal/cli/run.go` holds the command `run` and its loop. The command, with the option `--config` of its own:

1. the path: `--config` when given, otherwise `userConfigDir()` joined with `sensordeck/config.yaml`, and its error
   wrapped as R16 says
2. `readNamedFile("config file", path)`, then `config.Parse`, whose error is wrapped as `config file <path>: %w`
3. `newSources(sysfsRoot, procRoot)`
4. `notifyStop(context.Background())`, with its cancel deferred
5. `current.open()`, the opening of the other commands, traced when asked (R19, R24)
6. `running: <count> metrics every <interval>`, the interval with `Duration.String`
7. the loop, then the close of the transport, then `stopped` on standard output, and the status 0 (R23)

So nothing is written to the device before the config is known valid, and on Windows a valid config fails on step 3
with `windows is not supported yet` (R14, R19).

The loop is a type of its own, so that its unit tests drive it without the command line:

```go
type feeder struct {
	transport transport
	sources   []sensors.Source
	config    config.Config
	now       func() time.Time
	wait      func(stop context.Context, duration time.Duration) bool
	stderr    io.Writer
	lastClock time.Time
	lost      bool
}

func (feeding *feeder) feed(stop context.Context) error
```

`feed` works as follows:

* `writeClock` encodes `EncodeClock(now())` and writes it, through `write`. An error of `EncodeClock` ends `feed` with
  that error, and the command fails with it after closing the transport
* `write` returns at once without writing when `stop` is done, and after a failed write sets `lost`, writes
  `screen lost: <error>` to standard error and tells the caller to end the round (R22, R23)
* At the start: `writeClock`, then the first round starts at `first := now()`, index 0 (R20, R21)
* A round, at its start time: when `lost`, it calls `Reopen` once; a failure ends the round with no line, a success
  writes `screen back`, clears `lost`, calls `writeClock` and goes on with the values (R22). When not lost and
  `start.Sub(lastClock) >= time.Minute`, `writeClock` before the values (R20). The values: `sensors.ReadAll` with a
  `wanted` that is true for the names of the mappings only, the readings put into a map by name, then for each
  mapping in tag order a `smartmonitor.Value` of its tag and `sensors.ToValue` of its reading when there is one, and
  `EncodeValues` of the pairs, each report through `write`. No pair writes no report (R21, D3, D17)
* After a round, with `end := now()`: the next index is the larger of the index plus 1 and
  `int(end.Sub(first) / interval)`, its start `first.Add(index * interval)`. When the start is after `end`,
  `wait(stop, start.Sub(end))`; otherwise the round follows at once. `feed` returns nil when `stop` is done or a wait
  returns false (R21, R23)

`stop` is also what `ReadAll` gets, so a run of `nvidia-smi` in progress is killed at a signal and the process ends
well within 3 seconds (R23).

### The fake screen

`internal/fakescreen/fakescreen.go` gains:

```go
// Reports gives the complete reports that arrived so far, over all moves, each without the report ID byte.
func (screen *Screen) Reports() []smartmonitor.Report

// WaitReports waits until at least count reports have arrived or the limit has passed, polling as Received does, and
// gives the reports and whether the count was reached.
func (screen *Screen) WaitReports(count int, limit time.Duration) ([]smartmonitor.Report, bool)
```

`Reports` cuts the received bytes into groups of 65 under the mutex and leaves out an incomplete last group (R33).

### The end-to-end tests

`e2e/main_test.go`: `run` keeps its signature and calls a new `runWith(test, fake, environment []string, args...)`,
which appends the given variables after the two roots, so that they override what the process inherits. A new
`start(test, fake, environment []string, args...) *process` starts the executable, and
`(process *process) stop(signal os.Signal) (result, time.Duration)` sends the signal and waits, giving the result and
the time from the signal to the end. A cleanup kills a process that is still running. The check of a clock report in
`TestClock` becomes `expectClockReport(test, report, before, after)`, which the case of `run` uses too; this
extraction belongs to the commit of `readNamedFile`.

The new cases go into `e2e/sensors_test.go` and `e2e/run_test.go`, `//go:build linux` on the first line, with the
trees made by `fakesensors` on the sysfs root of a `fakescreen.Fake`. `PATH` is set to the directory of
`fakesensors.NvidiaSMI`, or to an empty temporary directory where there is to be no `nvidia-smi`, so that no real one
is ever run.

## Files

- `go.mod`, `go.sum`: `go.yaml.in/yaml/v3` v3.0.6 as a third direct requirement
- `layout_test.go`: the two new Linux packages in the list that must be limited to Linux; the check of the commands
  covers the import of `internal/linuxsensors` too; a new check that no Go file imports `gopkg.in/yaml.v3`
- `internal/sensors/metrics.go`, `source.go`, `convert.go`, and `metrics_test.go`, `source_test.go`, `convert_test.go`:
  new
- `internal/config/config.go`, `config_test.go`: new
- `internal/fakesensors/fakesensors.go`: new
- `internal/linuxsensors/hwmon.go`, `procstat.go`, `cpufreq.go`, `meminfo.go`, `nvidiasmi.go`, `sources.go`, and a
  `_test.go` file for each: new
- `internal/fakescreen/fakescreen.go`: `Reports` and `WaitReports`; `fakescreen_test.go`: new
- `internal/cli/main.go`: the five fields of `environment`, the third root, `waitFor`, the two new commands in the
  root command
- `internal/cli/commands.go`: `readNamedFile`, the command `sensors`
- `internal/cli/run.go`: new, the command `run` and `feeder`
- `internal/cli/sources_linux.go`, `sources_other.go`: new
- `internal/cli/commands_test.go`: the harness fills the new fields; the help names the eight commands
- `internal/cli/fake_test.go`: writes and reopenings of `fakeTransport` that fail for a count of calls
- `internal/cli/main_test.go`: new, `waitFor`
- `internal/cli/sensors_test.go`, `run_test.go`: new
- `e2e/main_test.go`: `runWith`, `start`, `process`
- `e2e/commands_test.go`: `expectClockReport`, and the comment of R36
- `e2e/sensors_test.go`, `e2e/run_test.go`: new
- `README.md`: the status line and the section of R26
- `AGENTS.md`: the subsection "Testing"

## Tasks

### Preparation

- [x] In `internal/cli/commands.go`, extract `readNamedFile(description, path string)` from `readTheme`, which then
  calls it with `theme file`; the tests of `upload` with a missing file pass unchanged (R16)
- [x] In `e2e/commands_test.go`, extract the checks of `TestClock` on the 64 bytes of the report into
  `expectClockReport(test, report, before, after)`, which `TestClock` calls; the suite passes unchanged (R35)
- [x] Commit: `feature-0004 Extract the reading of a named file and the check of a clock report`

### The metrics and the config

- [x] In `internal/sensors/metrics_test.go`, test that `IsName` accepts the eight fixed names of R1 and
  `cpu.ccd1.temp`, `ram.2.temp`, `nvme.10.temp` and `nvme.9.temp`, and refuses `ram.0.temp`, `ram.01.temp`,
  `CPU.temp`, `cpu.tmp`, `ram..temp`, `ram.-1.temp`, `ram.1x.temp` and the empty name; and that `Unit` gives the unit
  of R1 for one name of each of the twelve rows (R1, R2, R31)
- [x] In `internal/sensors/convert_test.go`, test `ToValue` with 51.5 giving 52, 51.49 giving 51, 0.5 giving 1, -5
  giving 0, 70000 giving 65535, 65535.4 giving 65535, and `math.Inf(1)` giving 65535 (R4, R31)
- [x] In `internal/sensors/source_test.go`, test that `ReadAll` joins the readings of two fake sources in their order
  and passes `wanted` and `stop` to both (R1, R3)
- [x] Write `metrics.go`, `source.go` and `convert.go` as the Approach says, so the tests pass (R1 to R4, D1, D2, D5)
- [x] Add `go.yaml.in/yaml/v3` at v3.0.6 with `go get` (R25, D19)
- [x] In `internal/config/config_test.go`, test that the example of R17 gives 1 second and the mappings `cpu.temp` 1
  and `cpu.clock` 2 in that order; that `interval: 500ms` gives 500 milliseconds, `interval: 100ms` is accepted, and a
  file without `interval` gives 1 second; that the mappings come ordered by tag whatever the order of the file; and
  each error of R18 to the character after the prefix: a syntax error, giving the text of the parser; `tags` given
  twice and a metric given twice in `tags`, each giving a text that contains `already defined`; an empty file, a file
  of a comment only, and a list; the key `colour`; `tags` missing, `tags:` with no value, `tags: {}` and `tags: [1]`;
  the key `cpu.tmp`; the tags `1.0`, `"1"`, `x` and a mapping; the tags `256` and `-1`, each named as written; `1` for
  both `cpu.temp` and `cpu.clock`, given in that order and in the other, both naming `cpu.clock` first; `interval: 1`,
  `interval: soon` and `interval: [1s]`; and `interval: 99ms` and `interval: -1s` (R17, R18, R31)
- [x] Write `internal/config/config.go` as the Approach says, so the tests pass (R17, R18, D11, D12, D18)
- [x] Commit: `feature-0004 Add the metric names and the config`

### The Linux sources

- [ ] Write `internal/fakesensors/fakesensors.go` as the Approach says, `//go:build linux` on its first line (R28 to
  R30)
- [ ] In `layout_test.go`, add `internal/linuxsensors` and `internal/fakesensors` to the directories that must be
  limited to Linux; make the check of the commands refuse the import of `internal/linuxsensors` as it refuses that of
  `internal/hidraw` in a file of a Windows build, and name it for both; and add a test that no Go file of the module
  imports `gopkg.in/yaml.v3` (R14, R25)
- [ ] In `internal/linuxsensors/hwmon_test.go`, with trees of `fakesensors`, test: no `class/hwmon` directory, giving
  nothing from all three hwmon sources; `k10temp` with `Tctl` in sensor 1, `Tccd1` in sensor 3 and `Tccd2` in sensor 4,
  giving `cpu.temp`, `cpu.ccd1.temp` and `cpu.ccd2.temp` in that order; an input of `-1500`, giving -1.5; a malformed
  input `4x000`, leaving its metric out; a sensor without a label, giving nothing; two `spd5118` entries, `hwmon5` on
  `.../1-0051` and `hwmon3` on `.../1-0053`, and two `nvme` entries, `hwmon7` on the larger PCI path and `hwmon2` on
  the smaller, numbered by their paths and not their numbers; an `nvme` entry without `Composite` before another,
  which leaves `nvme.1.temp` out and still gives `nvme.2.temp`; an entry without a `device` link, passed over and not
  counted; an entry named `amdgpu` with `edge`, giving nothing; and a `wanted` that holds only `ram.2.temp`, giving
  that reading alone (R5 to R7, R28)
- [ ] Test `parseMillidegrees` on `51375\n`, `-1500\n`, `40000` without a newline, and the refused `+5`, ``, `4.5`,
  `4x000` and `1 2` (R6)
- [ ] Write `hwmon.go` as the Approach says, so the tests pass (R5 to R7, D4, D5)
- [ ] In `internal/linuxsensors/procstat_test.go`, with a `wait` of the test that records its duration and writes the
  second content into the tree, test: a first reading that takes two samples, waits 200 milliseconds once and gives
  the value of the two lines; a second reading after the file changed again, which uses the kept sample, does not
  wait and gives the value of the change; a growth of the total of 0, giving nothing; a line with seven counters and a
  line with a counter `1x`, giving nothing; counters whose growths give a value above 100, giving 100; a missing file,
  which gives nothing and does not wait; and a `wanted` without `cpu.util`, which reads nothing and does not wait
  (R8, R29)
- [ ] Write `procstat.go`, so the tests pass (R8, D7)
- [ ] In `internal/linuxsensors/cpufreq_test.go`, test three CPUs at 3000000, 4425361 and 1200000, giving 4425.361; the
  same with one file holding `abc`, which is passed over; a directory `cpufreq` under `cpu0x`, passed over; and no CPU
  at all, giving nothing (R9, R29)
- [ ] In `internal/linuxsensors/meminfo_test.go`, test the two lines of the spec, giving the used and total values of
  R34; a file without `MemAvailable`, giving `ram.total` only; a `MemTotal` of `9x kB`, giving nothing; and a missing
  file, giving nothing (R10, R29)
- [ ] Write `cpufreq.go` and `meminfo.go`, so the tests pass (R9, R10, D6, D8)
- [ ] In `internal/linuxsensors/nvidiasmi_test.go`, each case with `test.Setenv("PATH", ...)` and a fake of
  `fakesensors.NvidiaSMI`, test: `44, 17, 3256, 32607`, giving the four readings in the order of R1; a fake that exits
  with 1 unless its arguments are exactly the two of R11, giving the four readings; two lines, of which the first
  counts; `[N/A]` in the first field and `[Not Supported]` in the third, giving the other two; three fields, giving
  nothing; a field `4a`, giving the other three; `44, 17, 3256, 32607` and an exit status of 1, giving nothing; an
  empty directory on `PATH`, giving nothing; a `wanted` without a GPU metric, for which a fake that writes a file
  shows that it did not run; a `wanted` of two GPU metrics, for which the fake appends a line to a file at each run
  and runs once; and a time limit of 100 milliseconds with `sleep 3 & wait` as the script, where the reading gives
  nothing and returns within 1.1 seconds (R11, R12, R30)
- [ ] Test `parseGPULine` on the line of the spec, on a line with `\n` and a second line, and on `a,b,c,d,e` (R11)
- [ ] Write `nvidiasmi.go` as the Approach says, so the tests pass (R11, R12, D9)
- [ ] In `internal/linuxsensors/sources_test.go`, test that `Sources` over a tree of every source, with a fake
  `nvidia-smi`, gives the names in the order of R1 (R1)
- [ ] Write `sources.go`, so the test passes (R1, R12)
- [ ] Commit: `feature-0004 Add the Linux sources of the sensors`

### The command sensors

- [x] In `internal/fakescreen/fakescreen_test.go`, test that `WaitReports` of 2 gives the two reports written through
  the node, without their report ID byte, and true; that `WaitReports` of 3 with a limit of 100 milliseconds gives the
  two and false; and that a report cut short is not counted by `Reports` (R33)
- [x] Write `Reports` and `WaitReports`, so the tests pass (R33)
- [x] In `internal/cli/commands_test.go`, make the harness fill the new fields of `environment` with fakes: a proc
  root, a `userConfigDir` of the test, sources the test gives, a `notifyStop` whose context the test can cancel, and a
  `wait` that moves the fake clock by its duration and gives false when the context is done. Extend the test of the
  help to name `sensors` and `run`, and add `run --help` naming `--config` (R25)
- [x] In `internal/cli/main_test.go`, test `waitFor`: 10 milliseconds with a live context gives true, and a cancelled
  context gives false at once (R23)
- [x] In `internal/cli/sensors_test.go`, test: readings of two fake sources written in their order with the unit of
  each name and one decimal, 51.375 as `51.4`, 38.125 as `38.1` and 40 as `40.0`; no reading, giving empty stdout,
  `no sensor could be read` on stderr and status 0; `sensors x`, an error; and `--device x sensors`, which never
  opens a transport (R15)
- [x] Write the third root and the new fields in `main.go`, `sources_linux.go` and `sources_other.go`, and the command
  `sensors` in `commands.go`, as the Approach says, so the tests pass (R13 to R15, D10)
- [x] Commit: `feature-0004 Add the command sensors`

### The command run

- [x] In `internal/cli/fake_test.go`, give `fakeTransport` a count of writes to fail with `errFake` before the
  writes succeed again, and a queue of results for `Reopen`, so that a test fails one write and then two reopenings
  (R22, R32)
- [x] In `internal/cli/run_test.go`, with the harness, a config written into `test.TempDir()` and fake sources whose
  values the test sets per round, test the cases below (R32):
    - the config errors: no `--config` and a `userConfigDir` that fails, giving the text of R16; a missing file,
      naming the path once; a config with the tag 256, giving `config file <path>: ` and the text of R18; and `run x`.
      None of them opens the transport (R16, R18, R19)
    - the default path: with no `--config`, the file `<userConfigDir>/sensordeck/config.yaml` is read (R16)
    - the start: `running: 2 metrics every 1s` on stdout after the opening, the first write the clock report of the
      fake time, the second the values of the first round (R19 to R21)
    - the sources are asked only for the mapped names (R21)
    - the clock again at the round 60 seconds after it was written, and not at the round before, with an interval of
      1 second (R20)
    - a round with one of two mapped metrics unavailable, whose report holds only the other pair (R21, D3)
    - a round with every mapped metric unavailable, which writes no report (R21)
    - 21 mapped metrics, `nvme.1.temp` to `nvme.21.temp` on the tags 21 down to 1, giving two reports, the first with
      the tags 1 to 20 in order and the second with tag 21 (R21, D17)
    - a round that takes 2.5 intervals in the fake sources, followed by the next at once and then by the round on the
      next start of the grid, the missed one not made up (R21)
    - a failed write, then two failed reopenings and one that succeeds: `screen lost: fake failure` and then
      `screen back` on stderr and nothing else, one `Reopen` per round, and the clock report followed by the values
      written after the reopening (R22)
    - a stop while waiting for a round and a stop while the screen is lost: no write after the stop, the transport
      closed, `stopped` as the last line of stdout and status 0 (R23)
    - `--trace run` writes a trace line for each write and for each reopening (R24)
- [x] Write `run.go` as the Approach says, so the tests pass (R16, R19 to R24, D11 to D17)
- [x] Run `go mod tidy` and check that `go.mod` keeps `go 1.27` with no `toolchain` line, and has cobra, x/sys and
  `go.yaml.in/yaml/v3` as its only direct requirements (R25)
- [x] Commit: `feature-0004 Add the command run`

### The end-to-end tests

- [ ] In `e2e/main_test.go`, add `runWith`, `start` and `process` as the Approach says, and set
  `SENSORDECK_PROC_ROOT` in the new cases (R34, R35)
- [ ] In `e2e/sensors_test.go`, test `sensors` against the tree of R34, built with `fakesensors` on the sysfs root of
  the fake screen and a `stat` of one line that does not change, with the fake `nvidia-smi` on `PATH`: stdout exactly
  that of R15, nothing on stderr, status 0. Test it against an empty tree with an empty directory as `PATH`: empty
  stdout, `no sensor could be read` on stderr, status 0 (R34)
- [ ] In `e2e/run_test.go`, with the config, the tree and the empty `PATH` of R35 and a screen on `hidraw3`, test:
  the first report a clock report through `expectClockReport` and the second exactly
  `02 02 01 00 2A 02 04 D2` and 56 bytes of `0x00`; SIGTERM once `WaitReports` of 4 is reached, giving status 0,
  stdout `running: 3 metrics every 100ms\nstopped\n` and the end within 3 seconds; `MoveAfter(2, 7)`, after which
  `WaitReports` of 4 within 3 seconds is reached, the third report a clock report and the fourth a value report, and
  after SIGTERM stderr holding a line starting `screen lost: ` and the line `screen back`; a config with the tag 256
  and no screen, giving the error of R18 and status 1; and no `--config` with `XDG_CONFIG_HOME` set to an empty
  temporary directory, giving `sensordeck: cannot read the config file <that directory>/sensordeck/config.yaml: no
  such file or directory` and status 1 (R35)
- [ ] Extend the comment at the top of `e2e/commands_test.go` with what the suite leaves to the unit tests and where:
  the clock after 60 seconds, the GPU metrics in `run`, `cpu.util`, the time limit of `nvidia-smi`, and a reopening
  that fails before it succeeds, in `internal/cli` and `internal/linuxsensors` (R36)
- [ ] Commit: `feature-0004 Add the end-to-end tests of sensors and run`

### The documents

- [ ] In `README.md`, replace the sentence after the marker `**Status: early development.**` that says nothing is
  usable yet with one that says the screen can be driven on Linux with the low-level commands, with
  `sensordeck sensors` and with `sensordeck run`, and that themes, Windows and the service are still to come. The
  sentence that the README describes what sensordeck is meant to become stays, and nothing else of the opening
  changes (R37, D20)
- [ ] Add the section `## Showing sensor values on the screen` to `README.md` after `## Linux: access to the screen`,
  with what R26 lists: `sensordeck sensors` and its format; a table of the twelve metrics of R1 with their units, and
  that the GPU metrics need `nvidia-smi` and come from the first NVIDIA GPU only; that the tags are those of the theme
  on the screen and only the user knows them for a vendor theme; the place of the config file, `--config`, and the
  example of R17 for a vendor theme with tag 1 the CPU temperature and tag 2 the CPU speed; that the values are sent
  as whole numbers, rounded, and an unreadable metric is not sent so the screen keeps its last value; and that
  `sensordeck run` sends until Ctrl+C or SIGTERM, sets the clock, and keeps running when the screen is unplugged and
  plugged in again (R26)
- [ ] In the subsection "Testing" of `AGENTS.md`, name `SENSORDECK_PROC_ROOT` with the two variables named there, and
  add that the tests that need `nvidia-smi` put a fake one on a `PATH` they set, so that the suite never runs a real
  one. The five commands do not change (R27)
- [ ] Commit: `feature-0004 Document the sensors and the run command`

## Verification

- [ ] `go test -count=1 ./...` passes, so that the end-to-end tests really run (R28 to R35)
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` succeed and report
  nothing (R14)
- [ ] `CGO_ENABLED=0 go build ./...` succeeds (R25)
- [ ] `go mod tidy -diff` reports no difference, and `go.mod` holds `go 1.27` and the three direct requirements (R25)
- [ ] `grep -rn 'gopkg.in/yaml' --include='*.go' .` finds nothing (R25)
- [ ] `npm run check-changes` passes on the committed work

## Commit Points

- [x] `feature-0004 Extract the reading of a named file and the check of a clock report`
- [x] `feature-0004 Add the metric names and the config`
- [x] `feature-0004 Add the Linux sources of the sensors`
- [x] `feature-0004 Add the command sensors`
- [x] `feature-0004 Add the command run`
- [x] `feature-0004 Add the end-to-end tests of sensors and run`
- [ ] `feature-0004 Document the sensors and the run command`
