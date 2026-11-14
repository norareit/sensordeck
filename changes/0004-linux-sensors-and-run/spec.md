# Linux sensors and the run command

## Context

Unit `0003-linux-transport-and-command-line` gave sensordeck an executable that reaches the screen over hidraw, with
low-level commands. After its merge the human checked it on the real device: the clock report sets the clock of the
screen, and the value update works. With the vendor theme now on the screen, tag 1 is shown as the CPU temperature
and tag 2 as the CPU speed: `values 1=42 2=1234` showed 42 and 1234. `listen` for 10 seconds received nothing
unasked, and the time limit of a read works on a real hidraw node. The upload is not yet checked on the device, and
nothing in this unit depends on it.

This unit makes the screen useful: a sensor layer that reads hardware readings on Linux, a command that lists them,
a config file that maps readings to the tags of the theme on the screen, and the command `sensordeck run` that sends
them to the screen until it is stopped. Until sensordeck has its own theme compiler, the tags are those of a vendor
theme, and only the user can say which tag shows what.

The unit is built in a sandbox that holds only what is committed: no screen, no sensors of the host (its `/sys` is
that of the container), no `nvidia-smi`, and no network but the Go module proxy. Every requirement below is checked
by automated tests with fakes: a sysfs and proc tree in a temporary directory, a fake `nvidia-smi` on a `PATH` the
test sets, and the fake screen of unit 0003.

### The machine this is for

The host of the human is an MSI MAG X870 TOMAHAWK WIFI with a Ryzen 9 9950X3D, an RTX 5090, two DDR5 modules, three
NVMe drives and 96 GB of RAM. On it, `/sys/class/hwmon` holds:

- `k10temp` with the labels `Tctl`, `Tccd1` and `Tccd2`
- two `spd5118` entries, one for each memory module, with `temp1_input` and no label
- three `nvme` entries, each with `temp1_label` `Composite` and two more sensors
- `amdgpu`, the integrated GPU, with the label `edge`
- `r8169_0_900:00`, the network chip
- no driver for the sensors of the board

### What is known of the sources on Linux

- Every hwmon device has a directory `/sys/class/hwmon/hwmonN`, with *N* a decimal number that the kernel hands out
  in the order the drivers load, so it can differ after a reboot. On a real system the entry is a symbolic link to a
  directory under `/sys/devices`
- An entry holds the file `name`, the name of the driver, and for each temperature sensor *K* the file
  `tempK_input`, the temperature in millidegrees Celsius as a decimal integer, and often `tempK_label`. Both end in a
  newline
- `<entry>/device` is a symbolic link to the device the sensor belongs to. Its target does not hold the hwmon
  number. On the host it resolves for instance to `/sys/devices/pci0000:00/0000:00:18.3` for `k10temp`, to
  `/sys/devices/pci0000:00/0000:00:14.0/i2c-1/1-0051` and `.../1-0053` for the two modules, and to
  `/sys/devices/pci0000:00/0000:00:01.2/0000:02:00.0/nvme/nvme2` for one drive
- The first line of `/proc/stat` is `cpu` followed by counters of time since boot: user, nice, system, idle, iowait,
  irq, softirq, steal, guest and guest_nice. guest and guest_nice are already counted in user and nice
- `/proc/meminfo` holds lines such as `MemTotal:       96187828 kB` and `MemAvailable:   73300372 kB`, in KiB
- `/sys/devices/system/cpu/cpuN/cpufreq/scaling_cur_freq` holds the current clock of CPU *N* in kHz, for instance
  `4425361`
- `nvidia-smi --query-gpu=temperature.gpu,utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits`
  writes one line for each GPU, for instance `44, 17, 3256, 32607`: °C, %, MiB and MiB. A field that the GPU does not
  report is written as `[N/A]` or `[Not Supported]`. On the host the command takes about 20 milliseconds

## Requirements

"A reading" is a metric name, a value and a unit. "Unavailable" means that a metric has no value now. "The sysfs
root" and "the proc root" are `/sys` and `/proc`, or what R13 puts in their place. `\t` stands for one tab character.
"R*n* of unit 0003" names a requirement of `changes/0003-linux-transport-and-command-line/spec.md`.

### Metrics

- **R1** The sensor layer knows exactly these metrics, with these units, in this order:

  | Metric | Unit | Source |
  |---|---|---|
  | `cpu.temp` | `°C` | hwmon `k10temp`, label `Tctl` |
  | `cpu.ccd<n>.temp` | `°C` | hwmon `k10temp`, label `Tccd<n>` |
  | `cpu.util` | `%` | `/proc/stat` |
  | `cpu.clock` | `MHz` | cpufreq `scaling_cur_freq`, the highest of all CPUs |
  | `gpu.temp` | `°C` | `nvidia-smi`, `temperature.gpu` |
  | `gpu.util` | `%` | `nvidia-smi`, `utilization.gpu` |
  | `gpu.mem.used` | `MiB` | `nvidia-smi`, `memory.used` |
  | `gpu.mem.total` | `MiB` | `nvidia-smi`, `memory.total` |
  | `ram.used` | `GiB` | `/proc/meminfo`, `MemTotal` minus `MemAvailable` |
  | `ram.total` | `GiB` | `/proc/meminfo`, `MemTotal` |
  | `ram.<n>.temp` | `°C` | hwmon `spd5118`, `temp1_input` |
  | `nvme.<n>.temp` | `°C` | hwmon `nvme`, label `Composite` |

  Where a row has `<n>`, the metrics of that row come in the order of *n*, ascending as numbers
- **R2** A metric name is one of the eight names of R1 without `<n>`, or one of the three patterns of R1 with `<n>`
  replaced by a decimal number of 1 or more written without leading zeros. Nothing else is a metric name: `ram.0.temp`,
  `ram.01.temp`, `CPU.temp` and `cpu.tmp` are not. A well-formed name for which this machine has no sensor, such as
  `nvme.9.temp` on a machine with three drives, is a metric name whose metric is unavailable
- **R3** Metrics are read on demand: a reading reads its source at that moment, and nothing is read in the background.
  A metric whose source is missing, cannot be read, or holds something that is not as R5 to R12 describe is
  unavailable, and that is never an error of a command. The value of a reading is a real number, not rounded
- **R4** A value becomes the 16-bit value of a value update by rounding to the nearest whole number, halves away from
  zero, and then clamping to 0 to 65535. So 51.5 gives 52, 51.49 gives 51, 0.5 gives 1, -5 gives 0, and 70000 gives
  65535

### The sources on Linux

- **R5** The hwmon sources read the directory `<sysfs root>/class/hwmon`. They take every entry whose name is `hwmon`
  followed by decimal digits, read its `name` without the trailing newline, and resolve the symbolic link
  `<entry>/device` to its final path. An entry whose `name` cannot be read or whose `device` cannot be resolved is
  passed over. Among the entries with the same `name`, the order is that of the resolved paths compared as strings,
  and the hwmon number decides only between equal paths. A missing `class/hwmon` directory gives no hwmon metrics
- **R6** From the first entry named `k10temp` in the order of R5: `cpu.temp` is the input of the sensor whose label is
  `Tctl`, and `cpu.ccd<n>.temp` is the input of the sensor whose label is `Tccd<n>`, with *n* the number in the label.
  An input is read as a decimal integer, possibly negative, without the trailing newline, and divided by 1000. A
  label is matched exactly, without the trailing newline
- **R7** The entries named `spd5118` give `ram.<n>.temp` from `temp1_input`, and the entries named `nvme` give
  `nvme.<n>.temp` from the input of the sensor whose label is `Composite`, both read as R6 reads an input. *n* is the
  position of the entry among all entries of that name in the order of R5, counted from 1, whether or not its
  temperature can be read now, so that a drive whose sensor fails does not renumber the others
- **R8** `cpu.util` is read from the first line of `<proc root>/stat` that starts with `cpu` followed by a space. A
  sample is the sum of the first eight counters, the total, and the sum of the fourth and fifth, the idle time. A line
  with fewer than eight counters, or a counter that is not a decimal whole number, makes no sample. The value is
  100 times the growth of the total minus the growth of the idle time, divided by the growth of the total, between the
  previous sample and this one, clamped to 0 to 100. The source keeps its previous sample from one reading to the
  next. A reading without a previous sample takes a sample, waits 200 milliseconds, and takes another. A growth of
  the total of 0 or less makes the metric unavailable
- **R9** `cpu.clock` is the highest value among the files
  `<sysfs root>/devices/system/cpu/cpu<N>/cpufreq/scaling_cur_freq`, with *N* decimal digits, each read as a
  decimal whole number in kHz and divided by 1000. A file that is missing or malformed is passed over, and with none
  left the metric is unavailable
- **R10** `ram.total` is the value of the line `MemTotal:` of `<proc root>/meminfo`, and `ram.used` that value minus
  the value of the line `MemAvailable:`, each in KiB and divided by 1048576. A missing or malformed line makes the
  metrics that need it unavailable
- **R11** The GPU metrics come from one run of the program `nvidia-smi`, found through `PATH`, with exactly the
  arguments `--query-gpu=temperature.gpu,utilization.gpu,memory.used,memory.total`,
  `--format=csv,noheader,nounits`. Its standard output is read, and its first line alone is used. The line is split
  at commas into fields, and the spaces around a field are removed. A line of other than four fields makes all four
  metrics unavailable. Otherwise each field gives its metric of R1 when it is a number, one or more decimal digits,
  optionally followed by `.` and one or more decimal digits, and leaves its metric unavailable otherwise, which
  covers `[N/A]` and `[Not Supported]`
- **R12** A missing program, a run that ends with a status other than 0, or a run that is not over after the time
  limit makes all four GPU metrics unavailable. The time limit is 2 seconds and is a value the source takes, so that
  a test gives a shorter one. A run over its time limit is killed, and the reading returns no later than 1 second
  after the time limit, also when the program started a child that keeps its standard output open. A reading of
  several GPU metrics at the same moment runs the program once. Every reading runs it anew, so that a program
  installed while `run` is running is used from then on
- **R13** The executable reads one more environment variable, `SENSORDECK_PROC_ROOT`, which replaces the proc root.
  It is read in the one place where R24 of unit 0003 reads the other two, an empty or unset variable means `/proc`,
  and it appears in no help text. The hwmon and cpufreq sources use the sysfs root of R24 of unit 0003. Everything
  past the start of the program takes the roots as values
- **R14** The Linux sources are limited to Linux by build constraints, and the code of the commands names them only
  in files limited to Linux, which do nothing but choose the sources, as R4 of unit 0003 holds for the transport. The
  test of the layout of unit 0003 checks this as well. The metric names, the conversion of R4 and the config are not
  limited to a platform. In a build for Windows, `sensors`, and `run` with a valid config, fail as R16 of unit 0003
  says with the message `windows is not supported yet`

### The command `sensors`

- **R15** `sensordeck sensors` takes no arguments. It reads every metric of R1 once, and writes one line to standard
  output for each metric that is not unavailable, in the order of R1: `<name>\t<value>\t<unit>`, the value with
  exactly one decimal, rounded as Go's `strconv.FormatFloat` with the format `f` and the precision 1 does. When no
  metric can be read it writes nothing to standard output and the line `no sensor could be read` to standard error.
  The exit status is 0 in both cases. It takes no account of `--device`, opens no device node, and reads no config.
  For the tree of R34 the standard output is exactly:

  ```
  cpu.temp	51.4	°C
  cpu.ccd1.temp	40.0	°C
  cpu.ccd2.temp	38.1	°C
  cpu.clock	4425.4	MHz
  gpu.temp	44.0	°C
  gpu.util	17.0	%
  gpu.mem.used	3256.0	MiB
  gpu.mem.total	32607.0	MiB
  ram.used	21.8	GiB
  ram.total	91.7	GiB
  ram.1.temp	40.0	°C
  ram.2.temp	37.9	°C
  nvme.1.temp	33.9	°C
  nvme.2.temp	38.8	°C
  ```

### The config file

- **R16** `run` reads its config from `sensordeck/config.yaml` in the directory that Go's `os.UserConfigDir` gives,
  which on Linux is `$XDG_CONFIG_HOME`, or `$HOME/.config` when that is unset. The option `--config <path>` of `run`
  reads the file at that path instead. When `os.UserConfigDir` fails and no `--config` is given, the error is
  `cannot find the config directory, name the file with --config: ` followed by its error. A file that cannot be read
  is the error `cannot read the config file <path>: ` followed by the error of the system, with the path written
  once. The config is read with `gopkg.in/yaml.v3`
- **R17** The file is a YAML mapping with two keys:
    - `tags`, a mapping from metric name to tag, with at least one entry. A tag is a YAML integer from 0 to 255
    - `interval`, the time between two rounds of `run`, in the syntax of Go's `time.ParseDuration`, such as `1s` or
      `500ms`, of at least 100 milliseconds. Without the key it is 1 second

  For example:

  ```yaml
  interval: 1s
  tags:
    cpu.temp: 1
    cpu.clock: 2
  ```

- **R18** The config is checked completely before `run` opens the device. Every problem is an error that starts with
  `config file <path>: `, and when a file has several problems one of them is reported. The texts after that prefix:
    - a file that YAML cannot read, including a key given twice: the text of the error of the YAML parser
    - a file whose content is not a mapping, including an empty file: `want a mapping with the keys interval and tags`
    - a key other than the two: `unknown key "<key>", the keys are interval and tags`
    - `tags` missing, empty, or no mapping: `tags: want a mapping from metric name to tag, with at least one entry`
    - a key of `tags` that is no metric name by R2: `tags: "<key>" is not a metric name`
    - a tag that is no YAML integer: `tags: <name>: the tag must be a whole number from 0 to 255`
    - a tag outside the range: `tags: <name>: the tag <tag> is outside 0 to 255`
    - two metrics with one tag: `tags: <first name> and <second name> both have the tag <tag>`, the two names in the
      order of Go's string comparison
    - an `interval` that is no duration: `interval: "<text>" is not a duration such as 1s or 500ms`
    - an `interval` below 100 milliseconds: `interval: <text> is shorter than 100ms`, with the text as written in the
      file

### The command `run`

- **R19** `sensordeck run [--config <path>]` takes no arguments. It reads and checks the config, then opens the
  transport as the other commands do: by discovery, or on the node of `--device`. A failure of either is a failure as
  R16 of unit 0003 says, and nothing is written to the device before the config is known to be valid. After the
  opening it writes `running: <count> metrics every <interval>` to standard output, with the count of entries of
  `tags` and the interval as Go's `time.Duration` writes it, such as `1s` or `100ms`
- **R20** `run` writes the clock report of `EncodeClock` for the current local time right after the opening, then
  before the values of the first round that starts at least 60 seconds after the clock was last written, and right
  after every reopening that succeeds
- **R21** `run` works in rounds, the first right after the first clock report and then one every interval, counted
  from the start of the first. A round that ends after the start of the next one is followed by the next at once, and
  rounds that were missed are not made up. A round reads the mapped metrics at that moment, and only those: the
  program `nvidia-smi` runs only when a GPU metric is mapped, and once in the round. Each metric that is not
  unavailable gives one pair of its tag and its value converted by R4. A metric that is unavailable is left out of
  the round, and no value is sent for it. The pairs are ordered by tag, ascending, and written as the reports that
  `EncodeValues` gives. A round in which every mapped metric is unavailable writes no report. The source of
  `cpu.util` keeps its sample for the whole run, so only the first round waits the 200 milliseconds of R8
- **R22** A failed write is the screen lost. `run` writes `screen lost: ` and the text of the error to standard error,
  writes no further report of that round, and from then on, at the time of every round, calls `Reopen` once instead
  of the round, until one succeeds. A failed reopening writes no line. After a reopening that succeeds it writes
  `screen back` to standard error, writes the clock as R20 says, and goes on with the round. The reopening is that
  of R12 of unit 0003: by discovery anew, so that a screen that came back under another hidraw number is found, or
  on the same node when `--device` was given
- **R23** On SIGINT or SIGTERM, `run` lets a write in progress end, writes no further report, closes the transport,
  writes `stopped` to standard output and ends with the exit status 0. A wait for the next round ends at once, and the
  process ends no later than 3 seconds after the signal. This holds while the screen is lost as well
- **R24** `--trace` works for `run` as R23 of unit 0003 says, so every write and every reopening of `run` is a line on
  standard error. `run` reads no input reports

### The executable and the module

- **R25** The help of R16 of unit 0003 names the eight commands, `sensors` and `run` among them, with a short
  description each, and the help of `run` names `--config`. `go.mod` has `gopkg.in/yaml.v3` at v3.0.1 or newer as a
  third direct requirement besides those of R2 of unit 0003, and no others. `go mod tidy -diff` reports no difference,
  and `CGO_ENABLED=0 go build ./...` still succeeds

### The documents

- **R26** `README.md` has a section with the heading `## Showing sensor values on the screen`. It says:
    - that `sensordeck sensors` lists the metrics that can be read now, in the format of R15
    - every metric of R1 with its unit, and that the GPU metrics need `nvidia-smi`, for the first NVIDIA GPU only
    - that the tags are those of the theme on the screen, which only the user knows for a vendor theme
    - where the config file is, with the option `--config`, and the example of R17 for a vendor theme in which tag 1
      shows the CPU temperature and tag 2 the CPU speed
    - that values are sent as whole numbers, rounded, and that a metric that cannot be read is not sent, so the screen
      keeps its last value
    - that `sensordeck run` sends the values until Ctrl+C or SIGTERM, sets the clock, and keeps running when the
      screen is unplugged and plugged in again
- **R27** The section "Testing" of `AGENTS.md` names the variable of R13 with the two it names, and says that the
  tests that need `nvidia-smi` put a fake one on a `PATH` they set, so that the suite never runs a real one. The five
  commands of the suite do not change

### Tests

- **R28** Unit tests of the hwmon sources, against trees in a temporary directory, cover: no `class/hwmon` directory;
  `k10temp` with `Tctl`, `Tccd1` and `Tccd2` in the sensors 1, 3 and 4; a negative input; a malformed input; a
  missing label; two `spd5118` and two `nvme` entries whose hwmon numbers run in the opposite order of their device
  paths, to show the order of R5; an `nvme` entry without a `Composite` label, which keeps its number; an entry
  without a `device` link; and an entry of another driver, which gives nothing
- **R29** Unit tests cover `cpu.util` with a wait that the test replaces and that changes the file between the two
  samples: the first reading and its two samples, a second reading that uses the kept sample and does not wait, a
  growth of the total of 0, a line with fewer than eight counters, and a value clamped to 100. They cover `cpu.clock`
  with several CPUs, one malformed file and none, and the RAM metrics with both lines, without `MemAvailable`, and
  with a malformed value
- **R30** Unit tests of the GPU source run a fake `nvidia-smi` from a directory on a `PATH` the test sets, and cover:
  a complete line; the fake checking the arguments of R11; two lines, of which only the first counts; `[N/A]` and
  `[Not Supported]` in one field each, with the other fields read; three fields; a field that is no number; an exit
  status of 1 with a valid line; a missing program; and a program that sleeps beyond a time limit of 100 milliseconds
  or less in a child that keeps its output open, where the reading returns within the bound of R12
- **R31** Unit tests cover R4 with every example named there and 65535.4, R2 with every example named there, and R18
  with every error named there
- **R32** Unit tests of `run`, with a fake transport, a time the test moves and sources the test gives, cover: the
  clock and the first round at the start; the clock again after 60 seconds and not before; a round with an
  unavailable metric, which leaves its pair out; a round with all metrics unavailable, which writes nothing; 21
  mapped metrics, giving two reports; a failed write followed by failed and then successful reopenings, with the
  lines of R22 and the clock after the reopening; and a stop while waiting for a round and while the screen is lost
- **R33** The fake screen of unit 0003 gives a way to wait, with a time limit, until a given number of reports has
  arrived, and to have the reports that arrived so far, because a running `run` never leaves the node quiet
- **R34** The end-to-end tests set `SENSORDECK_PROC_ROOT` and the two variables of unit 0003, and run `sensors`
  against a tree that holds exactly, with `<pci>` standing for `<sysfs root>/devices/pci0000:00`:
    - `hwmon0` and `hwmon1` named `nvme`, whose devices resolve to `<pci>/0000:00:02.1/0000:06:00.0/nvme/nvme0`, with
      `Composite` 38800, and to `<pci>/0000:00:01.2/0000:02:00.0/nvme/nvme1`, with `Composite` 33900
    - `hwmon2` named `k10temp`, with `Tctl` 51375, `Tccd1` 40000 and `Tccd2` 38125
    - `hwmon3` and `hwmon4` named `spd5118`, whose devices resolve to `<pci>/0000:00:14.0/i2c-1/1-0053`, with 37875,
      and to `<pci>/0000:00:14.0/i2c-1/1-0051`, with 40000
    - `scaling_cur_freq` of 4425361 for `cpu0` and 3000000 for `cpu1`
    - a `stat` that does not change, and a `meminfo` with `MemTotal` 96187828 and `MemAvailable` 73300372
    - a fake `nvidia-smi` that writes `44, 17, 3256, 32607`

  The output is that of R15, with exit status 0 and nothing on standard error; `cpu.util` is missing because the
  `stat` does not change. A second test with an empty tree and no `nvidia-smi` gives the line of R15 on standard
  error and exit status 0
- **R35** The end-to-end tests run `run` as a process against the fake screen, with a config of
  `interval: 100ms` and `tags` of `cpu.temp: 1`, `cpu.clock: 2` and `gpu.temp: 3`, a tree with `Tctl` 42000 and one
  CPU at 1234000 kHz, and a `PATH` with no `nvidia-smi`. They cover:
    - the first report is a clock report, checked as R33 of unit 0003 checks the one of `clock`, and the second is
      `02 02 01 00 2A 02 04 D2` followed by 56 bytes of `0x00`, showing tag 3 left out
    - SIGTERM after some rounds: exit status 0, standard output `running: 3 metrics every 100ms` and `stopped` as two
      lines, and the process ended within 3 seconds
    - the screen moved to another hidraw number after the first value report: `screen lost: ` and `screen back` on
      standard error, and a clock report followed by a value report arriving after the move, within 3 seconds
    - a config with a tag of 256, with no screen: the error of R18, exit status 1
    - no config file, with `XDG_CONFIG_HOME` set to an empty temporary directory: the error of R16 naming
      `<that directory>/sensordeck/config.yaml`, exit status 1
- **R36** What the end-to-end tests leave to the unit tests, and the comment of the end-to-end suite says so: the
  clock after 60 seconds, the GPU metrics in `run`, `cpu.util`, the time limit of `nvidia-smi`, and a reopening that
  fails before it succeeds

## Decisions

- **D1** Temperatures are in °C, utilisation in %, the clock in MHz, GPU memory in MiB and RAM in GiB. RAM is in GiB
  because the value update carries 16 bits: the 96 GB of the host are 93933 MiB, which does not fit, while the
  32607 MiB of the RTX 5090 do. GPU memory stays in MiB, the unit `nvidia-smi` gives exactly; a card of more than
  64 GiB would be clamped. A scale per tag, which would send tenths, is item 1 of "Beyond the request"
- **D2** A reading keeps its real value, and only the value update rounds it, R4. `sensors` shows one decimal so that
  the human sees what the sensor says, and rounding halves away from zero matches what a reader expects of 51.5.
  Clamping, and not an error, is used for values out of range, because a negative temperature or a full counter
  must not stop `run`
- **D3** An unavailable metric is left out of the round and no fixed value is sent. A fixed value such as 0 would be
  shown by the theme as a reading, 0 °C, which is false; leaving it out keeps the last true value on the screen. The
  price is that a sensor that disappears for good leaves a stale value, which `sensors` and `--trace` reveal
- **D4** The order of numbered metrics is that of the resolved `device` paths, R5, and not that of the hwmon numbers,
  which follow the order in which the drivers load and can change after a reboot. The device path depends on where
  the hardware sits: PCI address for a drive, bus and address for a memory module. The number of the I2C bus may
  change between boots, but both modules sit on the same bus, so their order stays. The numbers count from 1, as the
  CPU names its CCDs, and so do not match the kernel's `nvme0` names
- **D5** `Tccd<n>` becomes `cpu.ccd<n>.temp`, numbered as the label numbers it. It costs one pattern, and on a
  9950X3D the two CCDs differ, one holding the extra cache. `cpu.temp` is `Tctl`, the control temperature that the
  fan control of the board uses
- **D6** `cpu.clock` is the highest clock of all CPUs and not the average. On a mostly idle machine the average is
  pulled down by sleeping cores and says little, while the highest clock shows the boost that a load reaches, which
  is what a CPU speed on a dashboard is read as. An average is item 4 of "Beyond the request"
- **D7** `cpu.util` needs two samples. A first reading waits 200 milliseconds between them, so that `sensors` shows a
  real value and the first round of `run` has one. The alternatives were the average since boot, which is near 0 on
  a machine that has been up for a while and is no reading of now, and an unavailable first reading, which would
  leave `sensors` without `cpu.util` for ever. iowait counts as idle, as `top` counts it
- **D8** RAM used is `MemTotal` minus `MemAvailable`, which is what `free` shows as used and what the kernel offers
  as the memory a program can still have. `MemFree` would count the page cache as used and show a machine nearly full
- **D9** The GPU is read through `nvidia-smi` and not through NVML. The Go bindings of NVML need cgo, which R1 of unit
  0003 forbids, so that the executable stays one static file. Loading the library without cgo would need a further
  dependency and code that calls into C by hand. `nvidia-smi` comes with every NVIDIA driver, and one run of about
  20 milliseconds a second is cheap. The time limit of 2 seconds leaves room for a slow first start of the driver
  while bounding a hung program, and a round that waits for it is late rather than stuck. Only the first GPU is read
- **D10** The proc root is a third environment variable, like the two of unit 0003, for the same reason as D17 of
  that unit. The program `nvidia-smi` is found through `PATH`, which the tests already control, so no fourth variable
  is needed
- **D11** `run` reads the config completely and checks it before opening the device, as D23 of unit 0003 does with
  arguments, so that a mistake in the file is reported the same with and without a screen. A config without tags is
  refused, because `run` would then only set the clock, and an empty `tags` is more likely a mistake in indentation
  than a wish. A metric name is checked against the names sensordeck knows and not against the sensors of this
  machine, so that one config works on a machine where a drive is missing for a while
- **D12** The interval is at least 100 milliseconds, so that a typo such as `1ms` cannot flood the screen, and the
  tests can still run rounds quickly. The interval comes only from the config, which the tests write, so no hidden
  option is needed
- **D13** The clock is sent at the start, once a minute and after a reopening. The screen keeps time itself, as the
  check on the device showed, so the minute only corrects drift. After a reopening the screen may have rebooted and
  lost its time
- **D14** A lost screen is reopened with `Reopen` of unit 0003, once per round, without an end, so that a replug or a
  reboot of the screen does not end a service that runs for days. Only the loss and the return are written, not
  every failed attempt, so that a screen that stays away does not fill a log. At the start, however, a screen that
  cannot be opened is a failure: someone who starts `run` by hand wants to know at once. Waiting at the start is
  item 7 of "Beyond the request"
- **D15** The fake screen's move closes the master of the pseudo terminal, so the next write on the node fails, as a
  write to an unplugged hidraw node does, and the discovery then finds the screen under its new number. That is how
  the end-to-end test of R35 shows the recovery
- **D16** `run` writes few lines: one at the start, the loss and the return, and one at the end. The values of every
  round are not written, because `--trace` shows each report, and a service writing a line a second would fill its
  log
- **D17** The pairs of a round are ordered by tag, so that the bytes of a round are known in advance and a test can
  compare them exactly
- **D18** `gopkg.in/yaml.v3` is the YAML package, as the human chose. It gives the position of a syntax error and
  decodes into a tree of nodes, on which the checks of R18 work with their own texts

## Non-goals

- The Corsair Commander Core XT and the two temperature probes on it, at the front intake and at the power connector
  of the GPU. A later unit adds a native driver for it
- Sensors on Windows
- The theme compiler, theme folders and the `theme` commands
- Installing a service, and an interface between a running `run` and other commands
- Uploading a theme from `run`, or any use of the upload
- Fans, voltages and other sensors of the board. The board of the host has no hwmon driver loaded for them
- GPUs other than the first NVIDIA GPU, the integrated `amdgpu`, and AMD or Intel graphics cards
- CPUs other than AMD's: `coretemp` of Intel is not read
- Memory temperatures of DDR4 modules, whose driver is `jc42`
- Reading input reports of the screen while `run` runs
- Reloading the config while `run` runs

## Beyond the request

1. A scale per mapping entry, so that a value is sent in tenths, such as 51.4 °C as 514 or 21.8 GiB as 218. It costs
   a second form of an entry in `tags`, its checks and tests. Without it every value is a whole number, and RAM shows
   in whole GiB
2. The metrics `ram.util` and `gpu.mem.util` in %. It costs two rows of R1 and their tests. Without them a theme that
   shows a share of memory cannot be fed
3. A fourth field in `sensors` naming the device of a numbered metric, so that the user can tell which drive is
   `nvme.2`. It costs a field and its tests. Without it the drives are told apart by their temperatures or by the
   order of their PCI addresses
4. A metric `cpu.clock.avg` for the average clock. It costs one row and a test. Without it only the highest clock is
   available
5. `cpu.temp` from Intel's `coretemp`, `Package id 0`. It costs a second source for the metric and tests. Without it an
   Intel machine has no `cpu.temp`; the host is AMD
6. The temperature of the integrated GPU, `amdgpu` `edge`, as a metric. It costs one row and tests. Without it that
   sensor is not shown
7. `run` waits for the screen at the start instead of failing, for a service started before the screen enumerates.
   It costs a loop at the start and a test. Without it `run` ends with status 1, and a service manager restarts it
8. A line in `sensors` for each metric showing the 16-bit value `run` would send. It costs a column. Without it the
   rule of R4 in the README tells the user
9. The sentence of the README that says nothing is usable yet is brought up to date. It costs a sentence. Without it
   the README undersells what works, while the new section shows what does
