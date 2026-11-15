# The sensors of the machine on Windows

## Context

Step 7 of the roadmap of the initial plan is Windows support: HID on Windows, and a sensor source on Windows. Unit
`0016-windows-hid-transport` was its first part. Since then the screen and the Commander Core XT work on Windows, and
`sensors` and `run` work there with the Commander as their only source of readings. This unit is the second part: the
sensors of the machine on Windows, so that `sensors` and `run` give there the metric names they give on Linux, with the
same units, and a config written on one system feeds the same tiles on the other.

On Linux the metrics come from `internal/linuxsensors`, a package limited to Linux as a whole, whose `Sources` gives one
source for each group of metrics in the order of the table of metrics of `internal/sensors`. The code of the commands
chooses the sources in `internal/cli/sources_linux.go`, `sources_windows.go` and `sources_other.go`, which do nothing
else. A source reads on demand, reads only the metrics that `wanted` names, and gives nothing for a metric it cannot
read; being unavailable is never an error. `sensors` writes the readings in the order the sources give them, and the
problem of the Commander, when it has one and nothing could be read. `run` reads the mapped metrics in every round.

Windows offers some of the metrics through its own API, with nothing to install: the memory, the times of the CPU and
the counters of the network interfaces. The GPU metrics come from `nvidia-smi`, which the NVIDIA driver installs on
Windows as on Linux. The temperatures and the clock of the CPU, and the temperatures of the drives and the memory
modules, are not offered to a program without a kernel driver. LibreHardwareMonitor, a program under the MPL 2.0 that
has such a driver, reads them and serves them as a JSON tree over HTTP. This unit reads that tree; it does not install
LibreHardwareMonitor and does not depend on it for the native metrics.

The unit is built in a sandbox with no Windows and no LibreHardwareMonitor. The calls of Windows are built and vetted
with `GOOS=windows`, everything around them is tested on Linux, and the JSON tree is tested with documents written in the
tests from the facts below. Every behaviour on a real Windows machine is the human's to check, and the automated loop
does not wait for that check. The section "Unverified until the human runs it" lists what is open until then.

R15 of unit 0004 says that `sensors` reads no config. That stays true on Linux. On Windows `sensors` reads one key of the
config, by R27.

### What is known of the native sources

These are facts of the Windows API as Microsoft documents it, and of `golang.org/x/sys/windows` v0.50.0, which `go.mod`
requires.

- `GlobalMemoryStatusEx` of `kernel32.dll` fills a `MEMORYSTATUSEX`, whose first field `dwLength` the caller sets to the
  size of the structure, 64 bytes. Its fields are, in order: `dwLength` and `dwMemoryLoad` (32 bits each), then
  `ullTotalPhys`, `ullAvailPhys`, `ullTotalPageFile`, `ullAvailPageFile`, `ullTotalVirtual`, `ullAvailVirtual` and
  `ullAvailExtendedVirtual` (64 bits each). `ullTotalPhys` is the physical memory the system can use, in bytes, and
  `ullAvailPhys` the physical memory that can be had now without writing to disk, in bytes, which is what `MemAvailable`
  is on Linux. `dwMemoryLoad` is a whole percentage. The call returns zero on failure
- `GetSystemTimes` of `kernel32.dll` gives three `FILETIME` values, the idle, kernel and user time of all processors
  since boot, in units of 100 nanoseconds, each as a low and a high 32-bit half. The kernel time includes the idle
  time. The call returns zero on failure
- `golang.org/x/sys/windows` binds neither of these two calls, so they are loaded from `kernel32.dll`. It binds
  `GetIfTable2Ex` and `FreeMibTable` of `iphlpapi.dll`, with the types `MibIfTable2` (a count `NumEntries` followed by
  that many `MibIfRow2`) and `MibIfRow2`, and the level `MibIfTableNormal`. The table that `GetIfTable2Ex` gives belongs to
  the system until `FreeMibTable` frees it
- A `MibIfRow2` holds, among others, `InterfaceLuid` (64 bits, unique while the interface exists), `InterfaceIndex`
  (which may change when an adapter is disabled and enabled), `Type` (the IANA type, 24 for loopback, 6 for Ethernet, 71
  for Wi-Fi, 131 for a tunnel), `InterfaceAndOperStatusFlags` (one byte of bits), `InOctets` and `OutOctets` (64 bits,
  the bytes received and sent without errors, unicast, multicast and broadcast). The bits of
  `InterfaceAndOperStatusFlags` are, from the lowest: `HardwareInterface` (0x01, the interface is for hardware),
  `FilterInterface` (0x02, the interface is a filter module), `ConnectorPresent` (0x04, a connector is present, which is
  so when there is a physical network adapter), `NotAuthenticated`, `NotMediaConnected`, `Paused`, `LowPower` and
  `EndPointInterface` (0x80). The table lists, besides the adapters, loopback, tunnels such as Teredo and ISATAP, the
  virtual adapters of Hyper-V, WSL and VPNs, and the filter modules that NDIS stacks on an adapter
- `nvidia-smi.exe` is installed with the NVIDIA driver; current drivers put it in `C:\Windows\System32`, which is on
  `PATH`, and older ones in `C:\Program Files\NVIDIA Corporation\NVSMI`, which is not. Go's `os/exec` finds `nvidia-smi`
  on `PATH` with the extensions of `PATHEXT`. Its output on Windows may end its lines with CR LF

### What is known of LibreHardwareMonitor's web server

These facts are read from the source of LibreHardwareMonitor in its repository,
`https://github.com/LibreHardwareMonitor/LibreHardwareMonitor`, at its release tags v0.8.5 to v0.9.6 (the last release,
of 2026-02-14) and at `master` of 2026-11-15, and from a document that a user published in its issue 2253. The source
was read to learn the format, and no code of it is used. "Confirmed" marks what a cited source shows; "assumed" marks
what is to be checked on the human's machine.

The server:

- Confirmed (`MainForm.cs`, `HttpServer.cs`): the server is off by default, and is turned on with the menu Options >
  Remote Web Server > Run. Options > Remote Web Server > Interface / Port sets its address and port, and
  Options > Remote Web Server > Authentication turns on HTTP Basic authentication. The default port is 8085. With the
  default address the server listens on every address of the machine, `localhost` among them; with an address chosen
  in the menu it listens on that address only. Issue 1737 reports that before the change of pull request 1612 the
  address `127.0.0.1` could not be chosen
- Confirmed: a `GET` of `/data.json` answers with status 200, the content type `application/json`, the header
  `Cache-Control: no-cache`, and the whole tree as one JSON document in UTF-8, compressed with gzip when the request
  accepts gzip, in every release from v0.8.5. Without valid credentials, while authentication is on, it answers with
  status 401. The page `/metrics`, in the format of Prometheus, exists from v0.9.5 on
- Confirmed: the tree is built anew at each request from the values LibreHardwareMonitor read last. It reads the
  hardware once a second by default
- Assumed: LibreHardwareMonitor runs with the rights of an administrator, which its driver needs, and keeps running for
  the readings to be there

The tree:

- Confirmed: every node is a JSON object with the fields `id` (a whole number, counted through the tree, which changes
  when hardware or sensors appear), `Text` (a string), `Min`, `Value` and `Max` (strings), `ImageURL` (a string) and
  `Children` (an array of nodes, empty for a sensor)
- Confirmed: the root has the `Text` `Sensor`, its fields `Min`, `Value` and `Max` hold the words `Min`, `Value` and
  `Max`, and it has one child, the computer, whose `Text` is the name of the machine. Below the computer come the hardware nodes, a hardware node may hold
  hardware nodes of its own (a board holds its chip of sensors), and a hardware node holds one node for each type of
  sensor it has, whose `Text` is the plural of the type such as `Temperatures`, `Clocks` or `Load`. The sensor nodes are
  the children of these
- Confirmed: in every release from v0.8.5 to v0.9.6, and at `master`, a sensor node also has `SensorId`, a string, and
  `Type`, a string. `Type` is the name of the sensor type: `Temperature`, `Clock`, `Load`, `Voltage`, `Power`, `Data`,
  `Throughput` and others. `SensorId` is the identifier of the sensor: the identifier of its hardware, then `/`, the type
  in lower case, `/`, and the index of the sensor among the sensors of that type of that hardware, as in
  `/amdcpu/0/temperature/2`
- Confirmed: v0.9.5 and v0.9.6 add `RawMin`, `RawValue` and `RawMax` to a sensor node as strings formatted like `Value`
  (issue 2253 reports this), and `master` writes them as JSON numbers or `null`. `master` adds `HardwareId` to a hardware
  node and `Version` to the root. No release has these numbers
- Confirmed: a sensor node is in the tree whether or not the user hid it in the window of LibreHardwareMonitor
- Confirmed: the `Text` of a sensor is its name, which the user can rename in the window; a renamed sensor keeps its
  `SensorId`
- Confirmed: a hardware group switched off in the menu of LibreHardwareMonitor (CPU, RAM, Storage Devices and others,
  all on by default) leaves its hardware out of the tree

A value:

- Confirmed (`SensorNode.cs`): `Value` is the value formatted with the culture of the user: a temperature as a number
  with one decimal, a space and `°C`, as `46.9 °C`, or in Fahrenheit as `116.4 °F` when the user chose Fahrenheit in the
  menu; a clock as a number with one decimal, a space and `MHz`, as `4425.4 MHz`. The decimal separator is that of the
  culture, so `46,9 °C` in a German culture. The format with one decimal writes no group separator. A sensor with no
  value now has the `Value` `-`. A node that is no sensor has the `Value` empty
- Assumed: the minus sign of a negative value is `-` or, in some cultures, `−` (U+2212)

The names, confirmed from the source of `LibreHardwareMonitorLib` at the tags and at `master`, and for an AMD CPU in
v0.9.6 also from the document of issue 2253:

- A CPU is the hardware `/amdcpu/<k>` for AMD and `/intelcpu/<k>` for Intel, with *k* the number of the processor
  package counted from 0
- AMD, Zen and later: the temperature `Core (Tctl/Tdie)` when the CPU has no offset between Tctl and Tdie, which is so
  for Zen 2 and later; otherwise the two temperatures `Core (Tctl)` and `Core (Tdie)`. The temperatures `CCD<n> (Tdie)`,
  one for each CCD present, with *n* the slot of the CCD counted from 1, and with several CCDs also `CCDs Max (Tdie)` and
  `CCDs Average (Tdie)`. The clocks `Core #<n>`, one for each core, `Core #<n> (Effective)`, `Bus Speed`, and from v0.9.6
  `Cores (Average)` and `Cores (Average Effective)`. Issue 2253 shows `Core (Tctl/Tdie)` as `/amdcpu/0/temperature/2` and
  `CCD1 (Tdie)` as `/amdcpu/0/temperature/3`
- Intel: the temperatures `CPU Package`, one for each core named as the core is, `Core Max`, `Core Average`, and
  `<core> Distance to TjMax`. The clocks `Bus Speed` and one for each core, named `Core #<n>`, or `P-Core #<n>` and
  `E-Core #<n>` on a CPU with two kinds of cores from v0.9.6, or `CPU Core` on a CPU of one core
- An NVMe drive is the hardware `/nvme/<k>`, with *k* a whole number: the number of the physical drive of Windows at
  v0.9.6 and at `master`, and the order in which the drives were found before. Its temperature `Temperature`, the
  composite temperature of the drive, is the sensor `/nvme/<k>/temperature/0`. A drive may also have `Temperature 1`,
  `Temperature 2`, or from v0.9.5 `Temperature #1` and so on, and `Warning Temperature` and `Critical Temperature`, with
  other indexes. The hardware of a drive that gave no temperature when it was found has no sensor
  `/nvme/<k>/temperature/0`
- A memory module is the hardware `/memory/dimm/<k>` from v0.9.5 on, with *k* the index of the module. A module with a
  thermal sensor has the temperature `DIMM #<k>` as the sensor `/memory/dimm/<k>/temperature/0`, and its limits
  `Thermal Sensor Low Limit`, `Thermal Sensor High Limit`, `Thermal Sensor Critical Limit` and
  `Temperature Sensor Resolution` as temperatures of other indexes. A module with no thermal sensor has no temperature.
  Before v0.9.5 no temperature of a module is given

## Requirements

"The table" is the table of metrics of `internal/sensors`. "The native metrics" are `cpu.util`, `ram.used`, `ram.total`,
`ram.util`, `net.up` and `net.down`. "The GPU metrics" are the four of R1 of unit 0004. "The monitor metrics" are
`cpu.temp`, `cpu.ccd<n>.temp`, `cpu.clock`, `ram.<n>.temp` and `nvme.<n>.temp`. "The monitor" is the web server of
LibreHardwareMonitor. "The URL" is the URL of R14. "The document" is what one request to the URL gives. "The seam" is
the narrow set of calls into Windows of R2. "R*n* of unit 000*k*" names a requirement of the spec of that unit. `\t`
stands for one tab character.

### Metrics on Windows

- **R1** On Windows, `sensors` and `run` know every metric of the table and no other, with the names and units of the
  table. The native metrics come from Windows by R3 to R9, the GPU metrics from `nvidia-smi` by R10 and R11, the monitor
  metrics from the monitor by R12 to R26, and `commander.<n>.temp` from the Commander as unit 0016 says. The table, the
  rule of names of R2 of unit 0004 and the conversion of R4 of unit 0004 do not change

### The native sources

- **R2** The calls into Windows are behind a seam that holds nothing else: `GlobalMemoryStatusEx`, `GetSystemTimes`,
  and `GetIfTable2Ex` with `FreeMibTable`, each giving plain values (the two counts of bytes of memory; the three times
  as 64-bit counts; and for each row of the table its LUID, its flags byte, its `InOctets` and its `OutOctets`) or a
  failure. The files of the seam are limited to Windows by a build constraint. `kernel32.dll` is loaded lazily from the
  system directory with `windows.NewLazySystemDLL`, and a procedure that cannot be found is a failure of its call at the
  first call, and never a panic. Everything else of the native sources is in files without a constraint for Windows, and
  is tested on Linux by `go test ./...` against a fake of the seam
- **R3** `ram.total` is `ullTotalPhys` and `ram.used` is `ullTotalPhys` minus `ullAvailPhys`, each divided by 2³⁰.
  `ram.util` is 100 times `ram.used` divided by `ram.total`, not clamped, as R2 of unit 0007 says on Linux.
  `dwMemoryLoad` is not used. A reading of any of the three makes one call, and the three come from it. A failed call,
  an `ullTotalPhys` of 0, and an `ullAvailPhys` above `ullTotalPhys`, make all three unavailable
- **R4** A sample of the CPU is the total, the kernel time plus the user time, and the idle time, of one call of
  `GetSystemTimes`. `cpu.util` is computed from two samples by the rule of R8 of unit 0004: 100 times the growth of the
  total minus the growth of the idle time, divided by the growth of the total, clamped to 0 to 100, unavailable when the
  growth of the total is 0 or less. The source keeps its previous sample, and a reading without one takes a sample,
  waits 200 milliseconds through a wait the source takes, and takes another. A failed call makes no sample
- **R5** A sample of the network is the rows of one call of `GetIfTable2Ex` at the level `MibIfTableNormal` that count by
  R6, each keyed by its `InterfaceLuid` with its `InOctets` as the bytes received and its `OutOctets` as the bytes
  sent, and the moment it was taken, from a clock the source takes. The table is freed with `FreeMibTable` in every
  case. A failed call makes no sample
- **R6** A row counts when its flags have `HardwareInterface` (0x01) and `ConnectorPresent` (0x04) set and
  `FilterInterface` (0x02) clear. Whether a row counts is decided at every sample anew
- **R7** `net.up` and `net.down` are computed from two samples by the rules of R4, R5, R8 and R9 of unit 0007, with the
  LUID in place of the name of the interface: the bytes sent and received, summed over the interfaces in both samples,
  divided by the seconds between the two moments and by 1024; an interface that appeared or vanished adds nothing; a
  counter that went back adds nothing to its direction for one reading; no interface in both samples, or a time of 0 or
  less, makes both unavailable. A first reading waits 200 milliseconds between its two samples, and a reading of both
  metrics takes one sample for both
- **R8** The computation of a share from two samples of the CPU, and of the rates from two samples of the network, is
  one piece of code that Linux and Windows both use, and is not limited to a platform. It is moved out of
  `internal/linuxsensors` in a commit that changes no behaviour, as the standards ask, before the code of Windows uses
  it. On Linux the readings, the files read and the tests of units 0004 and 0007 do not change, beyond the moving of
  tests with the code they test
- **R9** A reading that wants none of the metrics of a native source makes no call of that source

### The GPU

- **R10** The GPU source of R11 and R12 of unit 0004 is one source that Linux and Windows both use, and is not limited to
  a platform. It is moved out of `internal/linuxsensors` in a commit that changes no behaviour, before Windows uses it. On
  Windows it runs `nvidia-smi` found through `PATH` with the arguments, the time limit and the parsing of unit 0004
- **R11** The first line of the output is taken up to the first LF, and a CR right before that LF is not part of the
  line, so that `44, 17, 3256, 32607\r\n` gives the four metrics on both systems. Nothing else of the parsing changes

### The URL

- **R12** The config file has one more key, `librehardwaremonitor_url`, a YAML string holding an absolute URL whose scheme
  is `http` or `https` and whose host is not empty, with or without user information, a port, a path and a query. Without
  the key the URL is `http://localhost:8085/data.json`
- **R13** A value that is no string, or a string that Go's `net/url.Parse` cannot parse or whose scheme or host does not
  meet R12, is the error `librehardwaremonitor_url: "<text>" is not an http URL such as http://localhost:8085/data.json`
  after the prefix of R18 of unit 0004, with the text as written in the file
- **R14** The URL is the value of the key, or the default of R12. The key is accepted on every system, and on Linux it
  changes nothing. The lists of keys in the error of an unknown key and in the error of a file that is not a mapping name
  it in their alphabetical order: `commander_color, commander_interval, interval, librehardwaremonitor_url, tags and
  theme`
- **R15** A change of the key while `run` runs is not applied: `run` writes the line of a key that a restart applies, as
  it does for `interval`, in the order `interval`, `commander_interval`, `commander_color`, `librehardwaremonitor_url`

### Reading the monitor

- **R16** A reading of the monitor is one `GET` of the URL, with user information of the URL sent as HTTP Basic
  authentication, through no proxy, whatever the environment says. It accepts gzip, follows redirects as Go's HTTP
  client does, and reuses its connection from one reading to the next. The whole exchange, from the connection to the
  end of the body, takes at most 500 milliseconds, a value the source takes, so that a test gives a shorter one; at the
  time limit the request is cancelled and the reading fails. It also ends at once when the stop of the reading is done
- **R17** A reading fails when the request fails, the status is other than 200, the body is not one JSON value, or that
  value is not an object with a field `Children` that is an array. The error of a failure is, in that order of cases:
  the text of the error of the HTTP client without the method and the URL it names, `status <code> <text>` with the code
  and the text of Go's `http.StatusText`, `the answer is not JSON`, and `the answer is not the tree of data.json`
- **R18** A reading of the metrics makes at most one request, for all the monitor metrics it wants together, and makes
  none when it wants none of them. It wants one when `wanted` gives true for `cpu.temp`, for `cpu.clock`, or for a name
  of `cpu.ccd<n>.temp`, `ram.<n>.temp` or `nvme.<n>.temp` with *n* from 1 to 64. A monitor metric with *n* above 64 is
  never given on Windows

### The tree

- **R19** A sensor of the document is a node, at any depth below the root, that is an object with `SensorId`, `Type`,
  `Text` and `Value` all strings, and a `SensorId` of at least three segments separated by `/` after a leading `/`, the
  last of which is decimal digits. Its hardware is its `SensorId` without its last two segments. A node with a field of
  another type than these, or without one of them, is no sensor, and its `Children` are still searched when they are an
  array; a `Children` that is no array is searched as none. Every other field of a node is ignored: `id`, `Min`, `Max`,
  `ImageURL`, `RawMin`, `RawValue`, `RawMax`, `HardwareId`, `Version` and any other
- **R20** The value of a sensor is read from its `Value` with the spaces around it removed. It is a number followed by
  one or more spaces and a unit. The number is an optional sign, `-` or `−` (U+2212), one or more decimal digits, and
  optionally one decimal separator, `.` or `,`, followed by one or more decimal digits. For a sensor whose `Type` is
  `Temperature` the unit is `°C`, or `°F`, in which case the value is the number minus 32, divided by 1.8. For a sensor
  whose `Type` is `Clock` the unit is `MHz`. Any other `Value`, `-` and the empty string among them, gives no value, and
  the metric of that sensor is unavailable
- **R21** The CPU is the hardware `/amdcpu/0` when the document has a sensor of it, and otherwise the hardware
  `/intelcpu/0`. Among the sensors of the CPU whose `Type` is `Temperature`:
    - `cpu.temp` is the first, in the order of the document, whose `Text` is `Core (Tctl/Tdie)`; without one, the first
      whose `Text` is `Core (Tctl)`; without one, the first whose `Text` is `CPU Package`
    - `cpu.ccd<n>.temp` is the first whose `Text` is `CCD<n> (Tdie)`, with *n* a decimal number of 1 or more written
      without leading zeros, as written in the name. `CCDs Max (Tdie)` and `CCDs Average (Tdie)` give nothing
- **R22** `cpu.clock` is the highest value among the sensors of the CPU whose `Type` is `Clock` and whose `Text` is
  `CPU Core`, or `Core #<n>`, `P-Core #<n>` or `E-Core #<n>` with *n* decimal digits. A sensor of those that has no value
  is passed over, and with none left the metric is unavailable. `Bus Speed`, `Core #<n> (Effective)`,
  `Cores (Average)` and `Cores (Average Effective)` give nothing
- **R23** The drives are the hardware of the form `/nvme/<k>`, with *k* decimal digits, of every sensor of the document,
  ordered by *k* as numbers. `nvme.<n>.temp` is the value of the sensor `/nvme/<k>/temperature/0` of the *n*-th drive,
  counted from 1, whether or not it has that sensor or a value now, so that a drive whose temperature is missing does not
  renumber the others. The `Text` of the sensor is not looked at
- **R24** The modules are the hardware of the form `/memory/dimm/<k>`, with *k* decimal digits, that have a sensor
  `/memory/dimm/<k>/temperature/0`, ordered by *k* as numbers. `ram.<n>.temp` is the value of that sensor of the *n*-th
  module, counted from 1, whether or not it has a value now. A module without that sensor takes no number. The `Text` of
  the sensor is not looked at
- **R25** Every other sensor and hardware of the document gives nothing: the GPUs, the board, the network, the
  controllers, other CPUs than that of R21, and drives that are not NVMe
- **R26** The mapping of R19 to R25 is a function of plain values: it takes the bytes of the body and gives the readings
  of the monitor metrics it found, or the failure of R17 for a body that is no tree. It does no input and output

### The commands

- **R27** On Windows, `sensors` takes the URL from the config file at the place where `run` reads it without `--config`.
  When that place cannot be found or no file is there, the URL is the default of R12. When the file cannot be read or is
  not valid by R18 of unit 0004 and this unit, `sensors` writes the error that `run` would give, as one line to standard
  error, takes the default URL and goes on. A theme named in the file is not compiled. On Linux `sensors` reads no config,
  as today
- **R28** On Windows, `sensordeck sensors` writes the readings of every metric of R1 that is not unavailable, in the order
  of the table and, within a numbered row, in the order of *n*, in the format of R15 of unit 0004. The exit status is 0
- **R29** When the reading of the monitor fails, `sensors` writes to standard error the line
  `LibreHardwareMonitor cannot be read at <URL>: <error>. The temperatures and cpu.clock need it running, with Options > Remote Web Server > Run checked`,
  with the URL without its user information and the error of R17, and lists the other metrics as R28 says. When the
  reading succeeds and gives none of the monitor metrics, it writes the line
  `LibreHardwareMonitor at <URL> gives no temperature and no clock that sensordeck knows`. When no metric at all can be
  read, the line of the monitor comes before the lines that unit 0016 writes then
- **R30** On Windows, `run` sends the native metrics, the GPU metrics, the monitor metrics and the probes of the
  Commander that the config maps, as it does on Linux. A round reads the monitor only when a monitor metric is mapped,
  by R18. A failed reading of the monitor writes the line of R29 to standard error, once: no further line is written for
  the failures that follow, until a reading succeeds. A reading that succeeds after a line of R29 writes
  `LibreHardwareMonitor read again at <URL>`. The monitor metrics of a failed reading are unavailable, so their tiles keep
  their last value, as R21 of unit 0004 says
- **R31** No round of `run`, and no `sensors`, waits for the monitor longer than the time limit of R16. A stop of `run`
  while a reading of the monitor is waiting ends that reading, and `run` ends within the 3 seconds of R23 of unit 0004

### Platforms and layout

- **R32** No new module is required: the monitor is read with Go's `net/http` and `encoding/json`, and the seam uses
  `golang.org/x/sys/windows`. `go.mod`, `go.sum` and `THIRD_PARTY_LICENSES` do not change, and no Go file imports `C`
- **R33** The code of the commands chooses the sources of Windows only in `internal/cli/sources_windows.go`, which does
  nothing but choose them. The sources of Windows, the reading of the monitor and its mapping are in packages that are
  not limited to Linux; only the files of the seam are limited to Windows
- **R34** The layout tests allow `golang.org/x/sys/windows` to be imported by the files of the seam of R2 and its test
  of R43, besides those of unit 0016, and allow `kernel32.dll` to be loaded by name with `NewLazySystemDLL`, besides
  `hid.dll`. Every other check of the layout tests stays. On a system that is neither Linux nor Windows nothing changes
- **R35** The suite of `AGENTS.md`, all five commands, succeeds

### The documents

- **R36** `README.md` has a section with the heading `## Windows: the sensors of the machine`, right after the section
  "Showing sensor values on the screen". It says:
    - which metrics Windows gives natively, with nothing to install: `cpu.util`, `ram.used`, `ram.total`, `ram.util`,
      `net.up` and `net.down`, and the GPU metrics through `nvidia-smi`, which the NVIDIA driver installs, when it is
      on `PATH`
    - that `net.up` and `net.down` on Windows are the sum over the network adapters with a connector, so that loopback,
      tunnels, filter modules and the virtual adapters of Hyper-V, WSL and VPNs are left out
    - which metrics need LibreHardwareMonitor: `cpu.temp`, `cpu.ccd<n>.temp`, `cpu.clock`, `ram.<n>.temp` and
      `nvme.<n>.temp`, and that sensordeck neither installs nor starts it
    - how to turn on its web server: start LibreHardwareMonitor as an administrator, check Options > Remote Web Server >
      Run, and keep it running; that its menu Options > Run On Windows Startup starts it with Windows; and that the CPU,
      RAM and Storage Devices groups of its menu must be on
    - the key `librehardwaremonitor_url` with its default, that it is to be set when the port, the address or the
      authentication of the web server was changed, with an example with a user and a password in the URL, and that the
      key does nothing on Linux
    - which sensors of LibreHardwareMonitor give which metric, by R21 to R24; that a renamed CPU sensor is not found;
      that the temperatures of memory modules need LibreHardwareMonitor 0.9.5 or newer; and that the numbering of
      `nvme.<n>.temp` and `ram.<n>.temp` on Windows follows the numbers LibreHardwareMonitor gives the drives and modules,
      and may differ from that on Linux on the same machine
    - what `sensors` and `run` do when the web server cannot be read, by R29 and R30, and that the other metrics still
      work
    - that the sensors of Windows have not yet been checked on a real Windows machine
- **R37** The sentences of the README that say the sensors of the machine are not read on Windows yet, in its opening,
  in the line of the Windows archive under "A release archive" and in the quick start, say instead that they are read,
  natively and through LibreHardwareMonitor, with a link to the section of R36. The section "Showing sensor values on the
  screen" says that its table names the sources of Linux, and that the section of R36 names those of Windows, and its
  sentence on the keys that a restart applies names `librehardwaremonitor_url`

### Tests

- **R38** Unit tests that run on Linux in `go test ./...` cover, against a fake of the seam where a call is needed:
    - R3, with a value checked for each metric, a failed call, a total of 0, and an available above the total
    - R4, with a wait the test gives: the first reading and its two samples, a second that does not wait, a growth of
      0, a value clamped to 100, and a failed call
    - R5 to R7, with a wait and a clock the test gives: the first reading, a second reading, the time between samples
      deciding the value, two rows that count summed, a row of each of the flags that make it not count (no
      `HardwareInterface`, no `ConnectorPresent`, `FilterInterface` set), an interface that appears and one that
      vanishes, a counter that went back, a failed call, no row that counts, and a reading of `net.down` alone
    - R9, as a fake that counts its calls
    - R11, with a line ending in CR LF and one ending in LF
    - R12 to R15, with each error and the default
- **R39** Unit tests of the mapping of R26, with documents written in the tests in the shape of "What is known", cover:
    - an AMD CPU with `Core (Tctl/Tdie)`, `CCD1 (Tdie)`, `CCD2 (Tdie)`, `CCDs Max (Tdie)`, `CCDs Average (Tdie)`, clocks
      `Core #1` to `Core #4` of which one is `-`, `Core #1 (Effective)` with a higher value, `Bus Speed` and
      `Cores (Average)`, giving `cpu.temp`, `cpu.ccd1.temp`, `cpu.ccd2.temp` and `cpu.clock` exactly
    - an AMD CPU with `Core (Tctl)` and `Core (Tdie)`, giving `Core (Tctl)`; an Intel CPU with `CPU Package`, `Core Max`
      and clocks `P-Core #1` and `E-Core #1`; a CPU of one core with the clock `CPU Core`; and both an AMD and an Intel
      CPU, giving the AMD one
    - values with `,` as the decimal separator, with `°F`, with `−` and `-`, with two spaces, `-`, the empty string,
      `NaN °C`, a unit that does not fit the type, and a group separator, each giving the value or no value as R20 says
    - three drives `/nvme/0`, `/nvme/2` and `/nvme/10`, the middle one without `temperature/0` but with another sensor,
      giving `nvme.1.temp` and `nvme.3.temp` and keeping the number 2; `Warning Temperature`, `Temperature 1` and a drive
      `/scsi/1` giving nothing; and `/nvme/id_error` passed over
    - two modules `/memory/dimm/1` and `/memory/dimm/3` with `temperature/0` and their limit sensors, and a module
      `/memory/dimm/2` with only timings, giving `ram.1.temp` and `ram.2.temp`
    - a sensor with a field of another type, a node whose `Children` is no array, nodes at several depths with
      hardware below hardware, fields of `master` (`RawValue` as a number, `HardwareId`, `Version`) and of v0.9.6
      (`RawValue` as a string), all of which change nothing of the rest
    - each failure of R17 that the body decides, and a document that gives none of the monitor metrics
- **R40** Unit tests of the reading of R16 to R18, against `net/http/httptest` servers, cover: a document read; a status
  of 401 and of 404; a body that is no JSON; a server that answers after the time limit, a time limit of 100
  milliseconds or less, where the reading fails within the time limit and 100 milliseconds; a server that holds the body
  open; a stop while waiting; a URL with a user and a password, which the server sees as Basic authentication; a proxy
  named in `HTTP_PROXY` that is not used; a reading that wants no monitor metric, which makes no request; and that one
  reading of several monitor metrics makes one request
- **R41** Tests of the commands that run on Linux, with the sources of Windows built over a fake of the seam, a fake
  `nvidia-smi` on a `PATH` they set and an `httptest` server, cover:
    - `sensors` with all sources readable, whose output lists the native, GPU, monitor and Commander metrics in the order
      of R28, checked line by line
    - `sensors` with the server stopped, whose output lists the other metrics and whose standard error is the line of
      R29, and with a document that gives no monitor metric
    - `sensors` with a config file whose `librehardwaremonitor_url` names the server, with none, and with one that is not
      valid, as R27 says
    - `run` with a config that maps a native metric and a monitor metric: the values sent, the line of R29 once over
      several rounds while the server fails, the line of R30 when it answers again, and no request in rounds when no
      monitor metric is mapped
- **R42** The executable is not run against the monitor in the end-to-end tests, which build it for Linux, where the
  sources of Windows are not chosen. The comment of the end-to-end suite says that the sources of Windows are tested by
  the tests of R38 to R41
- **R43** A test limited to Windows by a build constraint calls each call of the seam on the real system and passes when
  each succeeds, `ullTotalPhys` is above 0 and the total of the CPU is above 0, whatever the network adapters, none
  included. The job of R36 of unit 0016 runs it: its step of tests names the package of the native sources besides the
  two it names, and the test of the workflows checks that step

## Unverified until the human runs it

Each of these is open until the human checks it on Windows. The unit does not depend on it, and a wrong one is corrected
by a later unit.

- That `sensors` lists `cpu.util`, `ram.used`, `ram.total` and `ram.util` with values that agree with the Task Manager
- That the rule of R6 counts the Ethernet adapter of the host, and a Wi-Fi adapter, and leaves out loopback, Teredo, the
  filter modules and the `vEthernet` adapters of Hyper-V and WSL; and that `net.down` moves with a download
- That `nvidia-smi` is on `PATH` on the host, and writes the line of unit 0004 there, with CR LF or LF
- That `http://localhost:8085/data.json` reaches LibreHardwareMonitor with its web server in its default setting, on the
  version the human runs
- That on the 9950X3D the CPU is `/amdcpu/0` with `Core (Tctl/Tdie)`, `CCD1 (Tdie)` and `CCD2 (Tdie)` and clocks
  `Core #1` to `Core #16`, and that `cpu.temp` and `cpu.clock` are close to what Linux shows on the same machine
- That the three NVMe drives are `/nvme/<k>` with a sensor `temperature/0`, and not under another transport such as
  `/scsi/<k>`; and that their numbers stay from one boot to the next
- That the two DDR5 modules give `/memory/dimm/<k>/temperature/0`, which needs LibreHardwareMonitor 0.9.5 or newer, and
  whether another program that reads the SMBus keeps them from it
- That the values are written as "What is known" says in the culture of the human, `.` or `,`
- That 500 milliseconds is ample for the monitor on the host, and how long a reading takes there
- That the job of R43 passes on GitHub. The loop does not push, so the job first runs when the human pushes

## Decisions

- **D1 LibreHardwareMonitor for what Windows does not give.** The temperatures of the CPU, the drives and the memory
  modules, and the clock of the cores, need a kernel driver on Windows, which a program that installs nothing cannot
  have. WMI's `MSAcpi_ThermalZoneTemperature` gives a temperature of an ACPI zone on some boards, not of the CPU, and needs
  administrator rights; the clock that WMI gives, `CurrentClockSpeed`, does not follow the boost on many machines.
  HWiNFO is closed source, and its free version limits the sharing of its readings in time. These two are assumptions
  from common knowledge and were not checked for this spec. The
  initial plan names LibreHardwareMonitor, which is free, open source under the MPL 2.0, widely used for this, and serves
  its readings over HTTP, which Go reads with its standard library. The `/metrics` page of v0.9.5 and later, in the
  format of Prometheus, was rejected because no release before v0.9.5 has it and issue 2146 reports wrong values in it.
  Reading LibreHardwareMonitor's library from Go was rejected: it is a .NET library
- **D2 The native calls for what Windows does give.** `GlobalMemoryStatusEx`, `GetSystemTimes` and `GetIfTable2Ex` give
  the memory, the times of the CPU and the bytes of each interface with no rights and nothing installed, so these
  metrics work without LibreHardwareMonitor, as the request asks. They are the same quantities as on Linux:
  `ullAvailPhys` is what `MemAvailable` is, the kernel time includes the idle time as the total of `/proc/stat` does, and
  `InOctets` and `OutOctets` are the counters of `/proc/net/dev`. Reading them from LibreHardwareMonitor too was rejected,
  because it would make the native metrics depend on it. `dwMemoryLoad` was rejected for `ram.util` because it is a
  whole percentage and would break the agreement of `ram.util` with `ram.used` that D1 of unit 0007 gives
- **D3 One computation for both systems.** The share of busy time and the rates between two samples are the same
  decision on both systems, and the standards ask that such knowledge is held once. Moving it is a refactoring, so it
  is a commit of its own before the code of Windows. The same holds for the GPU source, whose code needs no change for
  Windows beyond R11, which also holds on Linux
- **D4 What a physical interface is on Windows.** On Linux an interface counts when the kernel ties it to a device,
  D4 of unit 0007. On Windows the nearest facts are `HardwareInterface`, which NDIS sets for an interface of a miniport,
  and `ConnectorPresent`, which Microsoft documents as set when there is a physical adapter. `HardwareInterface` alone
  was rejected, because virtual miniports such as those of VPNs and of the virtual switch of Hyper-V may set it.
  `FilterInterface` is excluded because a filter module that NDIS stacks on an adapter appears in the table with the
  counters of that adapter, and would count its bytes twice. A list of types or names was rejected for the reason of D4
  of unit 0007. The interface is keyed by its LUID, which does not change while it exists, and not by its index, which
  may change when it is disabled and enabled. The rule is a documented fact applied to virtual adapters whose flags no
  document lists, so it is unverified until the human runs it
- **D5 The fields the mapping relies on.** The mapping uses `SensorId`, `Type`, `Text`, `Value` and `Children`, which
  every release from v0.8.5 to v0.9.6 and `master` write in the same way. It ignores `id`, which changes when a sensor
  appears; `Min` and `Max`; and the `Raw*` fields, which no release before v0.9.5 has, which v0.9.5 and v0.9.6 write as
  formatted strings and `master` as numbers, three shapes for one field. Reading the formatted `Value` is the one way
  that works with every release; its format is known from the source, and R20 reads both decimal separators and both
  temperature units it can have. A node of an unexpected shape is passed over rather than failing the document, so that
  a field a later version changes costs that field only. A document from before v0.8.5, without `SensorId`, gives no
  metric, and the line of R29 says so
- **D6 The hardware by identifier, the sensor by name or by index.** The hardware is told by its identifier, which is
  stable, does not depend on the language of the window, and is the same in every version. Within the CPU the
  sensors are told by their names, because the index of a temperature of the CPU depends on the order in which sensors
  were created (the CCDs are created as they are found) while the names are fixed in the source; a sensor the user
  renamed is then not found, and the README says so. For the drives and the modules the index 0 is the composite
  temperature and the temperature of the module in every version, while their names changed between versions
  (`Temperature 1` and `Temperature #1`) and hold the index of a module, so the index is used
- **D7 The names of the metrics.** `cpu.temp` is the control temperature, Tctl, as on Linux, D5 of unit 0004:
  `Core (Tctl/Tdie)` is Tctl on a CPU without an offset, and `Core (Tctl)` on one with. Intel's `CPU Package` is read
  too, because its name is known and it costs one line; on Linux the Intel CPU stays item 5 of "Beyond the request" of
  unit 0004. `cpu.ccd<n>.temp` keeps the number of the slot that both LibreHardwareMonitor and `k10temp` give. `cpu.clock`
  is the highest clock of a core, as D6 of unit 0004 decided on Linux; the effective clocks, the averages and the bus are
  not clocks of a core
- **D8 A stable numbering of drives and modules.** On Linux the numbers follow the places of the hardware, D4 of unit
  0004. LibreHardwareMonitor gives a drive the number of the physical drive of Windows, which does not change from boot
  to boot unless drives are added or removed, and a module the index of its slot, so the numbers follow those. A drive
  keeps its number when its temperature is missing, as R7 of unit 0004 says on Linux. A module without a thermal sensor
  takes no number, because on Linux only modules with a thermal sensor have an `spd5118` entry, and a DDR4 machine whose
  modules have none would otherwise show numbers with nothing behind them. The numbering may differ from that of Linux
  on the same machine, and the README says so
- **D9 The limit of 64.** `wanted` answers for a name, and cannot list the names it wants. To make no request when no
  monitor metric is wanted, R18 asks it for the names a machine can have, as the Commander asks for its probes. 64 is
  above the 16 CCDs that LibreHardwareMonitor reads at most and above any count of drives or modules of a desktop
- **D10 The time limit of 500 milliseconds.** The monitor runs on the same machine and builds its document from values
  it holds, so it answers in a few milliseconds. 500 milliseconds leaves room for a busy machine, while a round of the
  default interval of 1 second stays on time, and a hung server delays the round and the clock ticks after it by at most
  half a second. A longer limit, such as the 2 seconds of `nvidia-smi`, was rejected because `nvidia-smi` must start a
  process and wake the driver, while the monitor is already running
- **D11 Lines that say what to do.** A user who sees no temperature must learn why from `sensors` and from the log of
  `run`. R29 names the URL, the error and the menu item to check, in one line. `run` writes it once per outage, as it
  writes `screen lost` once, so that a service whose monitor is not running does not fill its log; the line of return
  shows that it works again. The user information is left out of the line so that a password does not reach a log. A
  document with no metric that sensordeck knows gets its own line, because it is the case that a change of names in a
  later version would cause, and the human needs to see it
- **D12 The URL in the config, read by `sensors` too.** The request asks for a config key. `sensors` reads it on Windows
  so that it reads the same monitor as `run`; it reads it from the place `run` reads by default, without a `--config` of
  its own, which is item 1 of "Beyond the request". A config that is not valid does not stop `sensors`, because
  `sensors` is the command a user runs while writing the config, to see the names. On Linux `sensors` keeps reading no
  config, so that nothing changes there. The key is accepted on Linux so that one config works on both systems. The whole
  URL is the value, and not a port, so that an address, a path, `https` and a user with a password are all one key
- **D13 No proxy, and Basic authentication from the URL.** The monitor is on the machine or on its network, and a proxy
  of the environment, which Go's default client would use for an address other than `localhost`, would not reach it. Go's
  HTTP client sends the user information of a URL as Basic authentication, which is what the monitor's authentication
  checks, so it costs nothing
- **D14 The adversary.** The monitor is the user's own program on the user's machine or network, and is trusted to send
  its document. The time limit defends against a slow or hung server, and the passing over of nodes of another shape
  against a change of format. A hostile server that sends a document of many megabytes within the time limit is not
  defended against; a limit on the size is item 3 of "Beyond the request"
- **D15 No end-to-end test of the executable against the monitor.** The end-to-end tests build the executable for Linux,
  where the sources of Windows are not chosen, and the request asks for the test only where it is possible. A hidden
  switch that chooses the sources of Windows on Linux was rejected as a path that only tests would take, beside the three
  roots of the tests that already exist. R41 runs the commands themselves over the sources of Windows, with only the seam
  faked and a real HTTP server, which covers everything but the calls of Windows; R43 covers those on Windows
- **D16 The order of `sensors`.** On Linux the order of the readings is that of the table because each source gives one
  group of the table. The monitor gives metrics of four groups in one reading, so R28 states the order as a requirement,
  and how it is kept is the plan's

## Non-goals

- The startup task on Windows and `sensordeck service` on Windows, which are the next unit
- Installing, starting or configuring LibreHardwareMonitor, and any other program
- Reading HWiNFO, WMI, AIDA64 or the `/metrics` page of LibreHardwareMonitor
- Metrics that are not in the table, such as the fans, the voltages, the power of the CPU, and the temperature of the
  GPU memory that LibreHardwareMonitor may give
- Reading the GPU through LibreHardwareMonitor, and GPUs other than the first NVIDIA GPU
- A second CPU package, and drives that are not NVMe
- Rates of each interface, as on Linux
- End-to-end tests on Windows, and widening the CI job on Windows beyond R43

## Beyond the request

1. A `--config` option for `sensors`, so that it reads the URL from the same file as a `run` started with `--config`. It
   costs an option on every system and its tests. Without it `sensors` reads the monitor at the default place's URL, and
   a user with a config elsewhere sees the line of R29, which names the URL it tried
2. Looking for `nvidia-smi.exe` in `C:\Program Files\NVIDIA Corporation\NVSMI` when it is not on `PATH`, for older
   drivers. It costs a second place and a test. Without it a machine with an old driver has no GPU metrics, which
   `sensors` shows, and the README says to put the program on `PATH`
3. A limit on the size of the document, such as 16 MiB. It costs a check and a test. Without it a hostile server can make
   sensordeck read a large body for 500 milliseconds; the monitor is the user's own program
4. A pause after a failed reading of the monitor, such as 10 seconds without a request, so that a server that does not
   answer delays at most one round in ten. It costs a clock in the source and tests. Without it every round that maps a
   monitor metric waits the time limit while such a server does not answer; a refused connection, as when the monitor is
   not running, fails at once
5. A fourth field in `sensors` on Windows naming the sensor of LibreHardwareMonitor behind a numbered metric, so that the
   user can tell which drive is `nvme.2`. It costs a field and tests. Without it the drives are told apart by their
   temperatures, as on Linux
