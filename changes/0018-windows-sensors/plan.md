# Plan: the sensors of the machine on Windows

## Approach

The work comes in two halves, as R8, R10 and D3 ask. The first half is two refactorings that change no behaviour:
the computation from two samples of the CPU and of the network moves out of `internal/linuxsensors` into a package of
its own, and so does the GPU source. Then come the changes of behaviour: the CR of `nvidia-smi`, the config key, the
mapping of the tree of LibreHardwareMonitor, its reading over HTTP, the native sources over a seam with their fake, the
seam on Windows with its test and the CI step, the commands, and the README.

What the tree on `feature-0018-windows-sensors` holds, and what the plan builds on:

* `internal/linuxsensors` is limited to Linux in every file. `procstat.go` holds `sampleInterval` (200 ms), `sample`
  (`total`, `idle`), `parseStat` and `cpuUtilisation` (`procRoot`, `wait`, `previous`), whose `Read` keeps the
  previous sample, takes two samples with a wait on the first reading, and clamps the share. `netdev.go` holds
  `bytesPerKibibyte`, `interfaceCounters` (`received`, `transmitted`), `networkSample` (`moment`, `interfaces
  map[string]interfaceCounters`), `network` (`sysfsRoot`, `procRoot`, `wait`, `now`, `previous`), `parseNetDev`,
  `hasDevice`, `take`, whose moment is taken by `now` right after the file was read, and `growth`. `nvidiasmi.go`
  holds `gpu{timeLimit}`, `pipeGrace`, `gpuMetrics`, `gpuQueryArguments`, `gpuNumber`, `wantsAny` and `parseGPULine`,
  which cuts the output at the first LF. `sources.go` holds `gpuTimeLimit` (2 s) and `Sources(sysfsRoot, procRoot)`
* The tests of `internal/linuxsensors` are limited to Linux. `hwmon_test.go` holds the helpers `everything`, `read`
  and `newTree`; `nvidiasmi_test.go` holds `fullLine`, `readGPU`, `allFour` and `names`, and `sources_test.go` uses
  `fullLine` and `names`. The GPU tests run shell scripts through `fakesensors.NvidiaSMI`, and `internal/fakesensors` is
  limited to Linux
* `internal/sensors` holds the table `metricKinds`, `IsName`, `Unit`, the name builders, `Source` and `ReadAll`
* `internal/cli/sources.go` holds `sourceSet` (`others`, `controller`, `startSession`, `controllerProblem`) and
  `commanderSources(finder)`. `sources_windows.go` gives `commanderSources(commandercore.Windows())`; `sources_linux.go`
  adds `linuxsensors.Sources`; `sources_other.go` (`!linux && !windows`) fails with `unsupported()`. `newSources` of the
  `environment` takes the three roots, and `harness.run` in `commands_test.go` gives a set made of `harness.sources`,
  `harness.controller` and the other fields of the harness
* `sensors` in `internal/cli/commands.go` reads every source with `ReadAll`, fails with `interrupted` when the stop
  was done, writes the problem of the controller and `no sensor could be read` when nothing was read, and otherwise
  writes each reading in the order the sources gave it. It reads no config
* `run` in `internal/cli/run.go` finds the path with `configPath`, reads it with `readNamedFile("config file", path)`
  (`cannot read the config file <path>: <cause>`), loads it with `loadConfigFile`, whose error is
  `config file <path>: <problem>`, wraps `stderr` in a `lockedWriter` before `newSources`, and gives the `feeder` the
  `others` of the set. `restartKeys` in `runconfig.go` gives `interval`, `commander_interval` and `commander_color`
* `internal/config/config.go` reads the keys `theme`, `tags`, `interval`, `commander_interval` and `commander_color`,
  and names them in `errNotAMapping` and in the error of an unknown key. `config_test.go` compares whole `Config` values
  in two tests
* `layout_test.go` allows `golang.org/x/sys/windows` only in `internal/winhid/seam_windows.go` and its test, and
  `NewLazySystemDLL` only with `"hid.dll"`. `release_test.go` requires the step
  `go test ./internal/winhid ./internal/commandercore` of the job on Windows, and `.github/workflows/ci.yml` has it
* `golang.org/x/sys/windows` v0.50.0 binds `GetIfTable2Ex(level uint32, table **MibIfTable2) error`, `FreeMibTable
  (unsafe.Pointer)`, `MibIfTableNormal` (0), `MibIfTable2` (`NumEntries uint32`, `Table [1]MibIfRow2`) and `MibIfRow2`
  with `InterfaceLuid uint64`, `InterfaceAndOperStatusFlags uint8`, `InOctets uint64` and `OutOctets uint64`. Its
  `GetIfTable2Ex` calls `procGetIfTable2Ex.Addr()`, which panics with a `*windows.DLLError` when the procedure cannot
  be found. It binds neither `GlobalMemoryStatusEx` nor `GetSystemTimes`
* The end-to-end suite comment at the top of `e2e/commands_test.go` names `internal/linuxsensors` as the place of the
  tests of the GPU source, of `cpu.util` and of the network rates

### Choices the spec leaves to the plan

* **The shared computation from two samples** is a new package `internal/sampling`, with no build constraint and no
  I/O:
    * `const Interval = 200 * time.Millisecond`
    * `CPUSample{Total, Idle uint64}` and `CPU{Take func() (CPUSample, bool); Wait func(time.Duration)}` with the
      unexported field `previous`. `(*CPU).Read` is the `Read` of `cpuUtilisation` today, with `Take` in place of the
      reading of `/proc/stat`
    * `Counters{Received, Transmitted uint64}` and `Network[Key comparable]{Take func() (map[Key]Counters, bool); Wait
      func(time.Duration); Now func() time.Time}` with the unexported field `previous`. `(*Network[Key]).Read` is the
      `Read` of `network` today, with `growth` and the division by 1024; it calls `Now` right after a `Take` that
      succeeded, and never after one that failed, so that the moment follows the read as it does today
    * Linux keys the interfaces by name (`Network[string]`), Windows by LUID (`Network[uint64]`)
* **The Linux sources after the move.** `linuxsensors` keeps `parseStat`, which gives a `sampling.CPUSample`,
  `parseNetDev`, which gives `map[string]sampling.Counters`, and `hasDevice`, and gains `newCPUUtilisation(procRoot
  string, wait func(time.Duration)) *sampling.CPU` and `newNetwork(sysfsRoot, procRoot string, wait
  func(time.Duration), now func() time.Time) *sampling.Network[string]`, whose `Take` reads the file and filters as
  `take` does today. The tests of `procstat_test.go` and `netdev_test.go` keep every input and every assertion; only
  the construction of the source, the type of the field that holds it, and the type in `TestParseStatSums` and
  `TestParseNetDev` change
* **The GPU source** is a new package `internal/nvidiasmi`, with no build constraint: `Source{TimeLimit
  time.Duration}` with the `Read` of `gpu` today, and `const TimeLimit = 2 * time.Second`, which `gpuTimeLimit` of
  `linuxsensors` was. Its tests move from `internal/linuxsensors/nvidiasmi_test.go` to `internal/nvidiasmi/source_test.go`
  and keep `//go:build linux`, because they run shell scripts through `fakesensors`. The helpers `everything` and `read`
  are copied into it, and `fullLine` and `names` stay in `linuxsensors` for `sources_test.go`, in a file
  `helpers_test.go` limited to Linux
* **The config key** is the field `LibreHardwareMonitorURL *url.URL` of `config.Config`, never nil after `Parse`.
  `config.DefaultLibreHardwareMonitorURL = "http://localhost:8085/data.json"` is the text of the default, and
  `config.DefaultMonitorURL() *url.URL` gives a new `&url.URL{Scheme: "http", Host: "localhost:8085", Path:
  "/data.json"}`, whose `String()` a test compares with the constant. A value is valid when the node is a scalar with
  the tag `!!str`, `url.Parse` succeeds, `IsAbs()` holds, the scheme is `http` or `https` (which `url.Parse` gives in
  lower case), and `Hostname()` is not empty, reading "host" of R12 as the host without the port, so that
  `http://:8085/data.json` is refused. The text in the error of R13 is `node.Value`, as `parseInterval` takes it
* **The restart line of the key** compares the two URLs by `String()` through a helper of `runconfig.go` that takes a
  nil URL as the empty text, so that the `config.Config` values of the existing tests, which have none, stay equal
* **The mapping of the tree** is a new package `internal/librehardwaremonitor`, with no build constraint, whose
  `Map(body []byte) ([]sensors.Reading, error)` is R26. It decodes the body with `json.Unmarshal` into `any`, which
  refuses a body that is more than one JSON value, gives `ErrNotJSON` (`the answer is not JSON`) when that fails and
  `ErrNotTree` (`the answer is not the tree of data.json`) when the value is not an object with a `Children` array, and
  walks the nodes depth first in the order of the document. It gives the readings in the order of the table, and within
  a numbered row by *n*, and gives no numbered metric with *n* above `MaxNumber = 64`, so that R18 holds whatever
  `wanted` answers. A value is read by a regular expression on the trimmed `Value`:
  `^([-−]?)([0-9]+)(?:[.,]([0-9]+))? +(°C|°F|MHz)$`, with the unit then checked against the `Type`. The spaces trimmed
  are the character U+0020 only. Drive and module numbers *k* are ordered as numbers by their digits without leading
  zeros, by length and then by text, with the whole text as the tie-break, so that no *k* overflows
* **The reading of the monitor** is in the same package. `NewReader(address *url.URL, timeLimit time.Duration)
  *Reader` holds one `http.Client` whose transport is a clone of `http.DefaultTransport` with `Proxy` set to nil, so
  that the connection is reused and no proxy is used, with the default handling of redirects and of gzip.
  `(*Reader).Read(stop context.Context) ([]sensors.Reading, error)` makes one `GET` under `context.WithTimeout(stop,
  timeLimit)`, reads the whole body under it, closes the body, and gives `Map` of the body. Go's client sends the user
  information of the URL as Basic authentication. The error of a failed request is the `Err` of the `*url.Error`, so
  that the method and the URL are not named, and any other error of the exchange as it is. `const TimeLimit = 500 *
  time.Millisecond`
* **The monitor as a source.** `Wants(wanted func(string) bool) bool` asks `wanted` for `cpu.temp`, `cpu.clock` and
  the three numbered rows with *n* from 1 to 64. `Source{Reader *Reader; Report func(found []sensors.Reading, err
  error)}` implements `sensors.Source`: without `Wants` it makes no request and calls nothing; otherwise it calls
  `Reader.Read`, calls `Report` with every reading of the document, or with the error, and gives the readings that
  `wanted` names. The lines of the commands are made from `Report` in `internal/cli`
* **The native sources** are a new package `internal/windowssensors`, with no build constraint except for its seam
  files. `seam.go` holds `Seam` with `Memory() (total, available uint64, err error)`, `SystemTimes() (idle, kernel,
  user uint64, err error)` and `Interfaces() ([]Interface, error)`, where `Interface` is `{LUID uint64; Flags uint8;
  InOctets, OutOctets uint64}`. `NewMemory(seam) sensors.Source`, `NewCPU(seam, wait) sensors.Source`, built on
  `sampling.CPU` with `Total` the kernel plus the user time, and `NewNetwork(seam, wait, now) sensors.Source`, built on
  `sampling.Network[uint64]` with the rows of R6, are exported because the tests are in the package
  `windowssensors_test`, which imports the fake. `Sources(seam Seam) []sensors.Source` gives the CPU, the GPU of
  `nvidiasmi` with its `TimeLimit`, the memory and the network, with `time.Sleep` and `time.Now`
* **The fake of the seam** is a new package `internal/fakewinsensors`, with no build constraint, on the pattern of
  `internal/fakehid`: a `Seam` with the memory and its error, a list of the times of the CPU and a list of the tables of
  interfaces, each taken one per call with the last one repeated, an error for each call, and a count of the calls of
  each, behind a mutex
* **The seam on Windows** is `internal/windowssensors/seam_windows.go`, `System() Seam`. `kernel32.dll` comes from
  `windows.NewLazySystemDLL`, and `GlobalMemoryStatusEx` and `GetSystemTimes` are called only after their `Find`
  succeeded, through a helper on the pattern of `call` in `internal/winhid/seam_windows.go`. That helper is repeated
  rather than shared, because R34 lets only the seam files import `golang.org/x/sys/windows`. `Interfaces` calls
  `windows.GetIfTable2Ex(windows.MibIfTableNormal, &table)`, frees the table with `windows.FreeMibTable` in a `defer`
  once it was given, and reads the rows with `unsafe.Slice(&table.Table[0], table.NumEntries)`. Since x/sys panics with
  a `*windows.DLLError` when `GetIfTable2Ex` cannot be found, `Interfaces` recovers that panic, and only that one, as
  the failure of its call, so that no call of the seam panics as R2 asks. `MEMORYSTATUSEX` is laid out as the spec
  lists it, with `dwLength` set to its size, 64
* **The sources of the commands on Windows.** `sourceSet` gains `monitor func(address *url.URL, report
  func(found []sensors.Reading, err error)) sensors.Source`, nil on Linux and on other systems. `internal/cli/sources.go`
  gains `withWindowsSensors(set sourceSet, seam windowssensors.Seam) sourceSet`, which sets `others` to
  `windowssensors.Sources(seam)` and `monitor` to a function that makes a `librehardwaremonitor.Source` over
  `NewReader(address, librehardwaremonitor.TimeLimit)`, beside `commanderSources`. `sources_windows.go` does nothing but
  give `withWindowsSensors(commanderSources(commandercore.Windows()), windowssensors.System())`, so that it alone chooses
  the sources of Windows (R33), and the tests of the commands build the same set over the fake
* **The order of `sensors`** is kept by `sensors.InTableOrder(readings []Reading) []Reading`, which sorts a copy
  stably by the row of the table and, within a numbered row, by *n*. `sensors` writes its result. The sources of Linux
  give that order already, so the output on Linux does not change, which the end-to-end tests of `sensors` show
* **The URL of `sensors` on Windows** is read only when the set has a `monitor`, which is how `sensors` keeps reading
  no config on Linux. It is found with `configPath("")`. A failure to find the place and a file that does not exist
  (`fs.ErrNotExist`) give the default in silence. A file that cannot be read gives the line `cannot read the config
  file <path>: <cause>`, and a file that `config.Parse` refuses the line `config file <path>: <problem>`, which are the
  errors of `run` without the prefix `sensordeck: `, since `sensors` goes on. The theme is not compiled. The wording of
  `config file %s: %w` moves into a helper that `loadConfigFile` and `sensors` both use
* **The lines of the monitor** are in a new file `internal/cli/monitor.go`: the line of R29 of a failure, the line of
  a document with no monitor metric, the line of return of R30, the URL without its user information (a copy with
  `User` nil, `String()`), and `monitorOutage{stderr, address, out}` whose `observe(found, err)` writes the line of
  R29 on the first failure, nothing on the next ones, and the line of return on the first success after a failure.
  `sensors` keeps the outcome of the one reading and writes its line after the reading, so that an interrupted
  `sensors` writes none and the line comes before the problem lines of unit 0016

## Files

- `internal/sampling/sampling.go`: new, the CPU and the network from two samples
- `internal/linuxsensors/procstat.go`, `netdev.go`, `sources.go`, `procstat_test.go`, `netdev_test.go`: use
  `internal/sampling`
- `internal/nvidiasmi/source.go`: new, moved from `internal/linuxsensors/nvidiasmi.go`, with the CR of R11
- `internal/nvidiasmi/source_test.go`: new, moved from `internal/linuxsensors/nvidiasmi_test.go`, limited to Linux
- `internal/linuxsensors/helpers_test.go`: new, `fullLine` and `names`
- `internal/linuxsensors/hwmon.go`: the package comment no longer names nvidia-smi
- `internal/sensors/metrics.go`, `metrics_test.go`: `InTableOrder`
- `internal/config/config.go`, `config_test.go`: the key `librehardwaremonitor_url`
- `internal/librehardwaremonitor/tree.go`, `tree_test.go`: new, `Map`
- `internal/librehardwaremonitor/reader.go`, `reader_test.go`: new, `Reader`, `Wants` and `Source`
- `internal/windowssensors/seam.go`, `memory.go`, `cpu.go`, `network.go`, `sources.go`: new, no build constraint
- `internal/windowssensors/memory_test.go`, `cpu_test.go`, `network_test.go`: new, package `windowssensors_test`
- `internal/windowssensors/seam_windows.go`, `seam_windows_test.go`: new, limited to Windows by their names
- `internal/fakewinsensors/fakewinsensors.go`, `fakewinsensors_test.go`: new, the fake of the seam
- `internal/cli/sources.go`, `sources_windows.go`, `commands.go`, `run.go`, `runconfig.go`, `runconfig_test.go`,
  `runreload_test.go`, `commands_test.go`, `sensors_test.go`: the monitor in the set, the order and the URL of
  `sensors`, the monitor in `run`, the restart key
- `internal/cli/monitor.go`, `monitor_test.go`: new, the lines of the monitor
- `internal/cli/windowssensors_linux_test.go`: new, the commands over the sources of Windows, limited to Linux because
  it uses `fakesensors.NvidiaSMI`
- `e2e/commands_test.go`: the comment of the suite
- `layout_test.go`: the seam of the native sources and `kernel32.dll`
- `.github/workflows/ci.yml`, `release_test.go`: the package of the native sources in the test step on Windows
- `README.md`: the section of R36 and the sentences of R37

## Tasks

### 1. The computation from two samples, shared (refactoring)

- [x] Add `internal/sampling/sampling.go` with `Interval`, `CPUSample`, `CPU`, `Counters` and `Network[Key]`, as the
  choices above say, carrying over the comments of `procstat.go` and `netdev.go` that say why (R8)
- [x] In `internal/linuxsensors`, remove `sampleInterval`, `sample`, `cpuUtilisation`, `interfaceCounters`,
  `networkSample`, `network`, `take` and `growth`; make `parseStat` give a `sampling.CPUSample` and `parseNetDev` a
  `map[string]sampling.Counters`; add `newCPUUtilisation` and `newNetwork`, and use them in `Sources` (R8)
- [x] In `procstat_test.go` and `netdev_test.go`, build the sources with `newCPUUtilisation` and `newNetwork` and
  change the types that name the moved ones, keeping every input and assertion (R8)
- [x] Run the suite, check that the end-to-end tests pass unchanged, and commit:
  `feature-0018 Share the computation from two samples`

### 2. The GPU source, shared (refactoring)

- [x] Move `internal/linuxsensors/nvidiasmi.go` to `internal/nvidiasmi/source.go` with no build constraint, `gpu`
  becoming `Source{TimeLimit}` and `gpuTimeLimit` becoming `TimeLimit`; make `linuxsensors.Sources` use
  `nvidiasmi.Source{TimeLimit: nvidiasmi.TimeLimit}`, and drop nvidia-smi from the package comment of `hwmon.go` (R10)
- [x] Move `internal/linuxsensors/nvidiasmi_test.go` to `internal/nvidiasmi/source_test.go`, keeping `//go:build linux`
  and every test, with copies of `everything` and `read`; move `fullLine` and `names` into
  `internal/linuxsensors/helpers_test.go` (R10)
- [x] In the comment at the top of `e2e/commands_test.go`, name `internal/nvidiasmi` as the place of the tests of the
  GPU source and of the time limit of nvidia-smi (R10)
- [x] Run the suite, check that the end-to-end tests pass unchanged, and commit:
  `feature-0018 Share the GPU source`

### 3. The CR before the LF of nvidia-smi

- [x] In `internal/nvidiasmi/source_test.go`, add to `TestParseGPULine` the line `44, 17, 3256, 32607\r\n`, giving the
  four fields, and keep the line ending in LF; add a test of `Read` whose script writes the line with `printf` ending in
  `\r\n`, giving the four metrics (R11, R38)
- [x] In `parseGPULine`, drop one CR right before the first LF, and nothing when the output has no LF (R11)
- [x] Run the suite and commit: `feature-0018 Read the line of nvidia-smi that ends in CR LF`

### 4. The key librehardwaremonitor_url

- [ ] In `internal/config/config_test.go`, add: no key gives the default, whose `String()` is
  `http://localhost:8085/data.json`, and `DefaultMonitorURL().String()` is the constant; valid values with `https`, a
  user and a password, a port, a path and a query; each refused value with the exact error of R13: a number, a mapping,
  null, an unparsable string, a relative URL, `ftp://host/`, `http:///data.json` and `http://:8085/data.json`; and the
  new lists of keys of R14 in the two errors. Add the default URL to the two tests that compare a whole `Config`
  (R12, R13, R14, R38)
- [ ] In `internal/config/config.go`, read the key into `LibreHardwareMonitorURL` as the choices above say, add
  `DefaultLibreHardwareMonitorURL` and `DefaultMonitorURL`, and name the key in `errNotAMapping` and in the error of an
  unknown key in the order of R14 (R12, R13, R14)
- [ ] In `internal/cli/runconfig_test.go`, add to `TestRestartKeys` a changed URL and a URL in the case `all` after
  `commander_color`; in `internal/cli/runreload_test.go`, add a reload that changes only the key, writing
  `config file <path>: librehardwaremonitor_url changed, restart run to apply it` (R15, R38)
- [ ] In `restartKeys`, add `librehardwaremonitor_url` last, compared as the choices above say, and update its
  comment (R15)
- [ ] Run the suite and commit: `feature-0018 Add the key librehardwaremonitor_url`

### 5. The mapping of the tree of LibreHardwareMonitor

- [ ] Add `internal/librehardwaremonitor/tree_test.go`, with documents written in the tests in the shape of "What is
  known", each a root `Sensor` with a computer and hardware below it, covering every case of R39: the AMD CPU with
  `Core (Tctl/Tdie)`, `CCD1 (Tdie)`, `CCD2 (Tdie)`, `CCDs Max (Tdie)`, `CCDs Average (Tdie)`, clocks `Core #1` to
  `Core #4` with one `-`, `Core #1 (Effective)` higher, `Bus Speed` and `Cores (Average)`, giving `cpu.temp`,
  `cpu.ccd1.temp`, `cpu.ccd2.temp` and `cpu.clock` exactly; `Core (Tctl)` with `Core (Tdie)`; Intel with
  `CPU Package`, `Core Max`, `P-Core #1` and `E-Core #1`; `CPU Core`; AMD beside Intel, giving AMD; the values of R20
  (`,`, `°F` converted, `−` and `-`, two spaces, `-`, empty, `NaN °C`, `46.9 MHz` for a temperature, `1,234.5 MHz`);
  the drives `/nvme/0`, `/nvme/2` without `temperature/0` and `/nvme/10`, giving `nvme.1.temp` and `nvme.3.temp`;
  `Warning Temperature`, `Temperature 1` and `/scsi/1` giving nothing, and `/nvme/id_error` passed over; the modules
  `/memory/dimm/1` and `/memory/dimm/3` with their limit sensors and `/memory/dimm/2` with timings only, giving
  `ram.1.temp` and `ram.2.temp`; a sensor with a field of another type, a `Children` that is no array, hardware below
  hardware, the fields of `master` and of v0.9.6; a `CCD65 (Tdie)` and a 65th drive giving nothing (R18); the readings
  in the order of the table; each failure of R17 that the body decides (no JSON, two JSON values, an array, an object
  without `Children`, a `Children` that is no array) with its exact error; and a document with none of the metrics,
  which gives no reading and no error (R19 to R26, R39)
- [ ] Add `internal/librehardwaremonitor/tree.go` with `Map`, `ErrNotJSON`, `ErrNotTree` and `MaxNumber`, as the
  choices above say, importing nothing that does input or output (R19 to R26)
- [ ] Run the suite and commit: `feature-0018 Map the tree of LibreHardwareMonitor to the metrics`

### 6. The reading of LibreHardwareMonitor

- [ ] Add `internal/librehardwaremonitor/reader_test.go`, against `net/http/httptest` servers whose handlers that wait
  end when the request ends or the test releases them, covering R40: a document read, with the request a `GET` of the
  path and query of the URL; a status 401 and 404, `status 401 Unauthorized` and `status 404 Not Found`; a body that is
  no JSON; a server that answers after the time limit, with a limit of 50 ms, failing within the limit and 100 ms; a
  server that writes the headers and holds the body open, failing the same way; a stop done while waiting, failing at
  once; a URL with a user and a password, which the server reads with `BasicAuth`; a gzip body when the request
  accepts it; one connection for two readings, counted with the `ConnState` of the server; a refused connection whose
  error names neither `GET` nor the URL; `HTTP_PROXY` set to a second server that records requests, which sees none,
  together with a check that the transport of the reader has no `Proxy`, because Go never proxies a loopback address and
  the first half alone cannot fail; and through `Source`, a `wanted` with no monitor metric that makes no request and
  no call of `Report`, a reading of `cpu.temp`, `nvme.1.temp` and `ram.1.temp` that makes one request and gives only
  those, `Report` given every reading of the document, and `Report` given the error of a failure (R16, R17, R18, R40)
- [ ] Add `internal/librehardwaremonitor/reader.go` with `TimeLimit`, `NewReader`, `Reader.Read`, `Wants` and
  `Source`, as the choices above say (R16, R17, R18)
- [ ] Run the suite and commit: `feature-0018 Read the web server of LibreHardwareMonitor`

### 7. The native sources of Windows over a seam

- [ ] Add `internal/windowssensors/seam.go` with the package comment, `Seam` and `Interface` (R2)
- [ ] Add `internal/fakewinsensors/fakewinsensors.go` and `fakewinsensors_test.go` for what the fake gives and counts
  (R2, R38)
- [ ] Add `internal/windowssensors/memory_test.go`: the three metrics from one call, each checked for a total of 32
  GiB and 12 GiB available; `ram.util` alone; a failed call, a total of 0 and an available above the total giving
  nothing; and no call when none of the three is wanted (R3, R9, R38)
- [ ] Add `internal/windowssensors/memory.go` with `NewMemory` (R3, R9)
- [ ] Add `internal/windowssensors/cpu_test.go`, with a wait the test gives: the first reading makes two calls and
  waits 200 ms once, with kernel time that includes idle time giving the expected share; a second reading makes one
  call and does not wait; a growth of 0 gives nothing; a share above 100 is clamped to 100; a failed call gives
  nothing and does not wait; no call when `cpu.util` is not wanted (R4, R9, R38)
- [ ] Add `internal/windowssensors/cpu.go` with `NewCPU` (R4)
- [ ] Add `internal/windowssensors/network_test.go`, with a wait and a clock the test gives: the first reading and its
  two samples with one wait of 200 ms; a second reading without a wait; the seconds between the moments deciding the
  value; two rows that count summed; a row without `HardwareInterface`, one without `ConnectorPresent` and one with
  `FilterInterface` adding nothing; a row whose flags change between samples counting only where it counts; an
  interface that appears and one that vanishes; a counter that went back adding nothing to its direction for one
  reading; a failed call; no row that counts; a time of 0; `net.down` alone; and no call when neither is wanted (R5,
  R6, R7, R9, R38)
- [ ] Add `internal/windowssensors/network.go` with `NewNetwork` and the rule of R6 on the flags `0x01`, `0x02` and
  `0x04` as named constants (R5, R6, R7)
- [ ] Add `internal/windowssensors/sources.go` with `Sources` (R1, R10)
- [ ] Run the suite, check `GOOS=windows GOARCH=amd64 go vet ./internal/windowssensors ./internal/fakewinsensors`, and
  commit: `feature-0018 Read the memory, the CPU and the network of Windows over a seam`

### 8. The seam on Windows

- [ ] Add `internal/windowssensors/seam_windows_test.go`, in the package `windowssensors`: both procedures of
  `kernel32.dll` resolve with `Find`; `System().Memory()` succeeds with a total above 0; `System().SystemTimes()`
  succeeds with the kernel plus the user time above 0; `System().Interfaces()` succeeds, whatever rows it gives (R43)
- [ ] Add `internal/windowssensors/seam_windows.go` with `System`, as the choices above say (R2)
- [ ] In `layout_test.go`, allow `golang.org/x/sys/windows` in `internal/windowssensors/seam_windows.go` and its test
  besides those of `winhid`, still excluded from a Linux build, and allow `NewLazySystemDLL` with `"kernel32.dll"`
  besides `"hid.dll"`, renaming the test to say so; every other check stays (R34)
- [ ] In `release_test.go`, require the step `go test ./internal/winhid ./internal/commandercore
  ./internal/windowssensors` of the job on Windows, and say three packages in the comment of `checkWindowsJob`; change
  the step in `.github/workflows/ci.yml` to match (R43)
- [ ] Check `GOOS=windows GOARCH=amd64 go vet ./internal/windowssensors`, `GOOS=windows GOARCH=amd64 go test -c -o
  /dev/null ./internal/windowssensors` and `GOOS=darwin GOARCH=arm64 go build ./...` (R2, R34, R43)
- [ ] Run the suite and commit: `feature-0018 Call the memory, the CPU and the network of Windows behind the seam`

### 9. The commands on Windows

- [ ] In `internal/sensors/metrics_test.go`, test `InTableOrder`: readings of several rows and numbers given out of
  order come back in the order of the table and of *n*, `commander.<n>.temp` last, equal rows keep their order, and the
  input is not changed; add `InTableOrder` to `metrics.go` (R28)
- [ ] Add `internal/cli/monitor_test.go` for the lines of the monitor: the line of R29 with the URL without its user
  and password; the line of a document with no metric; `monitorOutage` writing the line of R29 once over three
  failures, the line of return after them, and nothing for a success with no failure before (R29, R30)
- [ ] Add `internal/cli/monitor.go` as the choices above say (R29, R30)
- [ ] In `internal/cli/sources.go`, add `monitor` to `sourceSet` and add `withWindowsSensors`; make
  `sources_windows.go` give the set of the choices above, and change its comment (R1, R33)
- [ ] Give the harness of `internal/cli/commands_test.go` a field `set *sourceSet` that `newSources` gives when it is
  set, and add `internal/cli/windowssensors_linux_test.go`, which builds `withWindowsSensors` over `fakewinsensors`
  with the controller of the harness, a fake `nvidia-smi` on a `PATH` it sets, and an `httptest` server, covering R41:
  `sensors` with every source readable, whose output is checked line by line, native, GPU, monitor and Commander
  metrics in the order of R28, with network counters that do not change so that the rates are `0.0`; `sensors` with
  the server closed, listing the other metrics with the line of R29 on standard error; a document with no monitor
  metric and its line; `sensors` with a config whose key names the server, with no config file, with a config dir that
  cannot be found, with a file that cannot be read, and with a file that is not valid, writing its line and reading the
  default URL; `sensors` with nothing readable, the line of R29 first and then the lines of unit 0016; `run` with a
  config mapping `ram.used` and `cpu.temp`, the values sent, the line of R29 once over several rounds while the server
  fails, the line of return when it answers again, and the tiles of the failed rounds not sent; `run` with a config
  mapping only `ram.used`, with no request to the server; a stop of `run` while the server holds a request, ending
  within the 3 seconds of R23 of unit 0004 (R27 to R31, R41)
- [ ] In `internal/cli/sensors_test.go`, add that `sensors` on a set without a monitor never calls `userConfigDir`,
  and that readings given out of order are written in the order of the table (R27, R28)
- [ ] In `sensors` of `internal/cli/commands.go`, read the URL when the set has a monitor, add the monitor to the
  sources with a report that keeps the outcome, write `InTableOrder` of the readings, and write the line of the monitor
  after a reading that was not interrupted, before the problem lines; move the wording `config file %s: %w` into a
  helper that `loadConfigFile` uses too (R27, R28, R29, R31)
- [ ] In `run` of `internal/cli/run.go`, add the monitor of the URL of the config to the `others` of the feeder, with
  `monitorOutage` on the locked standard error, when the set has one (R30, R31)
- [ ] In the comment at the top of `e2e/commands_test.go`, add that the sources of Windows are not chosen by the
  executable built for Linux, and are covered by the tests of `internal/windowssensors`, `internal/librehardwaremonitor`
  and `internal/nvidiasmi`, and by the tests of the commands in `internal/cli` that build them over a fake seam (R42)
- [ ] Run the suite and commit: `feature-0018 Read the sensors of the machine on Windows`

### 10. The documents

- [ ] In `README.md`, add the section `## Windows: the sensors of the machine` right after "Showing sensor values on
  the screen", saying every point of R36: the native metrics and the GPU metrics with `nvidia-smi` on `PATH`; the
  adapters with a connector for the network; the metrics that need LibreHardwareMonitor, which sensordeck neither
  installs nor starts; turning on its web server, Options > Run On Windows Startup, and the CPU, RAM and Storage Devices
  groups; the key with its default, when to set it, an example with a user and a password, and that it does nothing on
  Linux; the sensors behind each metric by R21 to R24, a renamed CPU sensor not found, version 0.9.5 for the modules,
  and the numbering that may differ from Linux; what `sensors` and `run` do when the server cannot be read; and that
  the sensors of Windows have not been checked on a real Windows machine (R36)
- [ ] In `README.md`, change the opening, the line of the Windows archive under "A release archive" and the quick
  start to say that the sensors of the machine are read on Windows, natively and through LibreHardwareMonitor, with a
  link to the new section; say in "Showing sensor values on the screen" that its table names the sources of Linux and
  that the new section names those of Windows; and name `librehardwaremonitor_url` in its sentence on the keys that a
  restart applies (R37)
- [ ] Run the suite and commit: `feature-0018 Document the sensors of the machine on Windows`

## Verification

- [ ] `go test ./...` passes (R35)
- [ ] `go vet ./...` reports nothing (R35)
- [ ] `gofmt -l .` prints no file name (R35)
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing (R35)
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing (R35)
- [ ] `GOOS=darwin GOARCH=arm64 go build ./...` succeeds, and `sources_other.go` is unchanged (R34)
- [ ] `git diff master -- go.mod go.sum THIRD_PARTY_LICENSES` shows no change, and no Go file imports `C` (R32)
- [ ] Every file of `internal/windowssensors` other than `seam_windows.go` and `seam_windows_test.go` has no build
  constraint, and neither has any non-test file of `internal/sampling`, `internal/nvidiasmi` or
  `internal/librehardwaremonitor` (R2, R33)
- [ ] `npm run check-changes` passes

## Commit Points

- [ ] `feature-0018 Write the plan of the sensors on Windows`
- [ ] `feature-0018 Share the computation from two samples`
- [ ] `feature-0018 Share the GPU source`
- [ ] `feature-0018 Read the line of nvidia-smi that ends in CR LF`
- [ ] `feature-0018 Add the key librehardwaremonitor_url`
- [ ] `feature-0018 Map the tree of LibreHardwareMonitor to the metrics`
- [ ] `feature-0018 Read the web server of LibreHardwareMonitor`
- [ ] `feature-0018 Read the memory, the CPU and the network of Windows over a seam`
- [ ] `feature-0018 Call the memory, the CPU and the network of Windows behind the seam`
- [ ] `feature-0018 Read the sensors of the machine on Windows`
- [ ] `feature-0018 Document the sensors of the machine on Windows`
