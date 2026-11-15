# The share of RAM in use and the network rates

## Context

The vendor theme now on the human's screen has a tile that shows the RAM in use in percent, on tag 12, and two tiles
that show the upload and the download rate in KB/s, on tags 18 and 19. The human found these tags on the device.
sensordeck reads `ram.used` and `ram.total` in GiB, by unit `0004-linux-sensors-and-run`, but has no share of RAM and
no network metric. Item 2 of "Beyond the request" of unit 0004 already named a `ram.util` in %, and was not taken.

For now the tiles of tags 18 and 19 are used for the probes of the Commander Core XT, so the network rates are not
mapped on today's screen. The project's own theme will want them later, so the three metrics of this unit are general
metrics of the registry, and nothing of them is tied to the vendor theme.

The unit is built in the sandbox, which has no network of the host: every requirement is checked by automated tests
with fake trees under the sysfs root and the proc root, as unit 0004 does. The check on the real machine is the
human's.

### What is known of the sources on Linux

- `/proc/meminfo` is the source of `ram.used` and `ram.total`, as R10 of unit 0004 reads it
- `/proc/net/dev` starts with two header lines, followed by one line for each network interface of the network
  namespace of the process: the name of the interface, a colon, and 16 decimal counters separated by spaces. The first
  eight are of what was received: bytes, packets, errs, drop, fifo, frame, compressed and multicast. The last eight
  are of what was transmitted: bytes, packets, errs, drop, fifo, colls, carrier and compressed. The name is padded with
  spaces on the left, and on some kernels the first counter follows the colon with no space. The counters are 64 bits
  wide on the kernels of today, and count from the moment the interface was made, so they go back to 0 when the
  interface is made anew, for instance when its driver is reloaded. A name of an interface holds no colon and no space
- `/sys/class/net` has one entry for each interface of the same namespace. The entry of an interface that belongs to
  a device, a network card on PCI or a USB adapter, holds a symbolic link `device` to it. The entry of a virtual
  interface, which lives under `/sys/devices/virtual/net`, has none: loopback `lo`, a bridge such as `docker0`,
  `virbr0` or `br-<id>`, a `veth` pair, `tun` and `tap`, WireGuard, bonds and VLANs
- On the host, the network chip is driven by `r8169`, as the hwmon entry `r8169_0_900:00` that unit 0004 lists shows

## Requirements

"R*n* of unit 000*k*" names a requirement of the spec of that unit. "The sysfs root" and "the proc root" are those of
R13 of unit 0004. "A sample" of the network is what one read of `<proc root>/net/dev` gives, R6, together with the
moment it was read. A KiB is 1024 bytes.

### Metrics

- **R1** The table of metrics of R1 of unit 0004, with the row of R1 of unit 0005, gets three more rows, so that the
  table is, in this order:

  | Metric | Unit | Source |
  |---|---|---|
  | `cpu.temp` | `°C` | as before |
  | `cpu.ccd<n>.temp` | `°C` | as before |
  | `cpu.util` | `%` | as before |
  | `cpu.clock` | `MHz` | as before |
  | `gpu.temp` | `°C` | as before |
  | `gpu.util` | `%` | as before |
  | `gpu.mem.used` | `MiB` | as before |
  | `gpu.mem.total` | `MiB` | as before |
  | `ram.used` | `GiB` | as before |
  | `ram.total` | `GiB` | as before |
  | `ram.util` | `%` | `/proc/meminfo`, the share of `MemTotal` that `ram.used` is |
  | `ram.<n>.temp` | `°C` | as before |
  | `nvme.<n>.temp` | `°C` | as before |
  | `net.up` | `KiB/s` | `/proc/net/dev`, the bytes transmitted by the interfaces of R7 |
  | `net.down` | `KiB/s` | `/proc/net/dev`, the bytes received by the interfaces of R7 |
  | `commander.<n>.temp` | `°C` | as before |

  The three new names are names of a fixed name in the sense of R2 of unit 0004, so the config of `run` accepts them
  in `tags` with no other change, and `ram.util.x`, `net.upload` and `NET.UP` are no metric names. The row
  `commander.<n>.temp` is no longer the last, and keeps its place after every row of the Linux sources

### The share of RAM

- **R2** `ram.util` is 100 times `MemTotal` minus `MemAvailable`, divided by `MemTotal`, with the two lines of
  `<proc root>/meminfo` read as R10 of unit 0004 reads them. A missing or malformed line, and a `MemTotal` of 0, make
  the metric unavailable. The value is not clamped, so that it is exactly the share that `ram.used` is of `ram.total`
- **R3** A reading of any of `ram.used`, `ram.total` and `ram.util` reads `<proc root>/meminfo` once, and the three
  readings it gives come from that one content, so that the three always agree with each other. What R10 of unit 0004
  says of `ram.used` and `ram.total` does not change

### The network rates

- **R4** `net.up` is the bytes transmitted and `net.down` the bytes received, summed over the interfaces of R7,
  between the previous sample and this one, divided by the seconds between the moments of the two samples and by 1024.
  The moments are taken from a clock that the source takes, so that a test gives its own, and the time between them
  is that measured, not the interval of `run` nor the wait of R5
- **R5** The source keeps its previous sample from one reading to the next, as R8 of unit 0004 says of `cpu.util`. A
  reading without a previous sample takes a sample, waits 200 milliseconds through a wait that the source takes, so
  that a test replaces it, and takes another. A reading of `net.up` and `net.down` at the same moment, as `sensors`
  makes and as a round of `run` makes when both are mapped, takes one sample for both, and a reading that wants
  neither reads no file. A reading that cannot take a sample keeps the previous one, so that the next reading measures
  from it
- **R6** A sample is read from `<proc root>/net/dev`. A line counts when it has a colon and, after the first colon, 16
  or more fields separated by spaces, of which the first and the ninth are decimal whole numbers that fit in 64 bits.
  The name of the interface is the text before the first colon without the spaces around it, the received bytes are
  the first field and the transmitted bytes the ninth. Every other line, the two header lines among them, is passed
  over. A missing or unreadable file makes no sample, and both metrics unavailable
- **R7** An interface counts when `<sysfs root>/class/net/<name>` holds an entry named `device`, of any kind. So
  `lo`, and every virtual interface of "What is known", do not count. Whether an interface counts is decided at every
  sample anew, and a sample keeps the counters of the interfaces that count only
- **R8** An interface adds to a reading only when it is in both samples. An interface that appeared since the previous
  sample, and one that vanished, add nothing to that reading. An interface whose counter in this sample is below that
  in the previous sample adds nothing to that direction in that reading, and from the next reading on adds again,
  measured from this sample
- **R9** Both metrics are unavailable when no interface is in both samples, and when the time between the moments of
  the two samples is 0 or less. Otherwise both have a value, which is 0 when nothing was sent or received
- **R10** The first round of `run` that reads a network metric waits the 200 milliseconds of R5, and every later round
  measures from the sample of the round before. The source keeps its sample for the whole run, as R21 of unit 0004
  says of `cpu.util`

### Platforms and layout

- **R11** The sources of R2 and R4 are limited to Linux by build constraints, as R14 of unit 0004 says of the Linux
  sources, and the code of the commands chooses them only in the files limited to Linux that choose the sources. The
  metric names are not limited to a platform. In a build for Windows nothing changes, and the five commands of the
  suite of `AGENTS.md` succeed

### The commands

- **R12** `sensordeck sensors` and `sensordeck run` read and send the three metrics as every other metric, by R15 and
  R21 of unit 0004. `sensors` takes about 200 milliseconds longer, for the first sample of the network

### The documents

- **R13** The table of metrics of the section "Showing sensor values on the screen" of `README.md` has the three rows
  of R1, with their units and sources, in the order of R1. Below the table it says that `net.up` and `net.down` are the
  sum over the network interfaces that belong to a device, so that loopback and virtual interfaces such as those of
  Docker, bridges and VPNs are left out, that a KiB is 1024 bytes, and that the rates are measured between two
  readings, the first of which waits a moment for the second

### Tests

- **R14** Unit tests of `ram.util` against trees in a temporary directory cover: both lines, with the value checked;
  `MemAvailable` missing; a malformed `MemTotal`; a `MemTotal` of 0; and, for one content, that `ram.util` equals 100
  times `ram.used` divided by `ram.total`
- **R15** Unit tests of the network source, against trees under a sysfs root and a proc root in a temporary directory,
  with a wait and a clock that the test gives, and with the wait changing the file between the two samples, cover:
  the first reading, its two samples and its one wait of 200 milliseconds; a second reading that uses the kept sample
  and does not wait; the time between the samples as the clock gives it, other than 200 milliseconds, deciding the
  value; two interfaces that count, summed; `lo`, `docker0`, a `veth`, a bridge and a `tun` interface without a
  `device` entry, whose bytes add nothing; a first counter with no space after the colon; a line with fewer than 16
  fields and a line with a field that is no number, passed over while the other lines count; an interface that
  appears, and one that vanishes, between two samples; a received counter that went back, which adds nothing to
  `net.down` while its transmitted counter adds to `net.up`, and adds again at the next reading; no file; no interface
  that counts; a time between the samples of 0; a reading that wants only `net.down`; and a reading that wants
  neither, which reads no file
- **R16** The unit tests of R2 of unit 0004 cover the three new names and the three examples of R1 that are no
  metric names, and the test of the units of each row covers the new rows
- **R17** The tree of the end-to-end tests of `sensors`, of R34 of unit 0004, also holds a `net/dev` that does not
  change, with the lines of `lo`, of `enp6s0` and of `docker0`, and entries for the three under `class/net` of the
  sysfs root, of which only `enp6s0` holds a `device`. The output of R15 of unit 0004, and that of R20 of unit 0005,
  then gains the line `ram.util\t23.8\t%` after `ram.total`, and the lines `net.up\t0.0\tKiB/s` and
  `net.down\t0.0\tKiB/s` after `nvme.2.temp`, with every other line unchanged. The test with an empty tree keeps its
  output
- **R18** An end-to-end test runs `run` against the fake screen with a config of `interval: 100ms` and `tags` of
  `ram.util: 12`, `net.up: 18` and `net.down: 19`, against a tree with the `meminfo` of R34 of unit 0004 and the
  network of R17. A value report holds the pairs (12, 24), (18, 0) and (19, 0). Once that report has arrived, the test
  replaces `net/dev` at once, through a new file renamed into place, by one in which the received bytes of `enp6s0`
  are 1048576 higher, and a value report holding (19, *v*) with *v* of 1 or more arrives within 3 seconds. After
  SIGTERM the process ends with status 0
- **R19** The comment of the end-to-end suite says what it leaves of the network to the unit tests: the exact value of
  a rate other than 0, which depends on the time between two rounds, counters that go back, and interfaces that appear
  or vanish

## Decisions

- **D1** The share of RAM is `ram.util`. The registry names a share in % `.util`, as `cpu.util` and `gpu.util` are,
  and item 2 of "Beyond the request" of unit 0004 already used this name. It is read from the same content of
  `/proc/meminfo` as `ram.used`, R3, so that a tile in GiB and a tile in % on one screen never disagree within one
  round. It is a reading of its own and not something the config computes from two metrics, because the value update
  carries whole numbers: `ram.used` and `ram.total` arrive at the screen rounded to whole GiB, and a share computed
  from them would be off by several percent
- **D2** `ram.util` is not clamped to 0 to 100, unlike `cpu.util`. A `MemAvailable` above `MemTotal` does not happen
  on a sound kernel, and clamping would break the identity with `ram.used` that R2 promises for no reading anyone
  would see. R4 of unit 0004 clamps a negative value to 0 for the screen anyway
- **D3** The network metrics are `net.up` and `net.down`, each one name of a fixed name for the whole machine, as
  the tiles of the theme show one rate each. `up` and `down` say what a user reads on a tile, upload and download,
  where `tx` and `rx` would say it in the kernel's terms. `net` is short as `cpu`, `gpu` and `ram` are. Rates of each
  interface, numbered or named, are item 1 of "Beyond the request"
- **D4** Only interfaces that belong to a device count, R7, told by the `device` entry under `/sys/class/net`. Every
  byte the machine exchanges with the outside crosses such an interface, while a virtual interface carries bytes that
  are counted again on a physical one: a packet of a Docker container is counted on its `veth`, on `docker0` and on
  the network card, and a packet of a VPN on `tun` or `wg0` and again, encrypted, on the card. Summing every interface
  would count such traffic two or three times, and loopback would add traffic that never leaves the machine. The
  `device` entry is how the kernel itself tells a device from a virtual interface, so one rule covers every kind of
  virtual interface, also those that a list of names would miss, and a test needs only a directory with or without
  the entry. A list of names to exclude, such as `lo`, `docker*`, `br-*`, `veth*`, `virbr*`, `tun*`, `tap*` and
  `wg*`, was rejected: it is never complete, and a user may give a physical interface any name. The cost is that the
  traffic between containers or virtual machines on the same host, which leaves no physical interface, is not counted;
  it is not upload or download in the sense of the tiles. Wi-Fi cards and USB adapters belong to a device and count.
  The rule is decided at every sample, so that a USB adapter plugged in while `run` runs is counted from the next
  reading on
- **D5** The unit is KiB/s, 1024 bytes a second, and is written `KiB/s`. Every unit of memory in the registry is
  binary and written with its binary symbol, `MiB` and `GiB`, and one convention keeps a reader from having to ask
  which one a unit follows. The theme writes `KB/s`, which says nothing of 1000 or 1024; the two differ by 2.4 %,
  which a tile of whole numbers hardly shows. 1000 bytes, written `kB/s`, was rejected because it would be the only
  decimal unit of the table. Bits a second, as link speeds are given, were rejected because the theme shows bytes
- **D6** The value update carries 16 bits, so a rate above 65535 KiB/s, about 64 MiB/s or 537 Mbit/s, is sent as
  65535 by R4 of unit 0004. A download that fills a gigabit link, or the faster link of the host, shows 65535 on the
  screen while `sensors` shows the true value. A second pair of metrics in MiB/s is item 2 of "Beyond the request"
- **D7** The rates are made from two samples as `cpu.util` is, R5, so that the two behave alike: the source keeps its
  sample, a first reading waits 200 milliseconds between two samples, and every later reading measures from the
  reading before. `sensors` therefore shows a real rate and the first round of `run` has one. The alternatives were
  the same as for `cpu.util` in D7 of unit 0004 and are rejected for the same reasons: the average since the
  interface was made is no reading of now, and an unavailable first reading would leave `sensors` without the
  network for ever. Unlike `cpu.util`, the rate needs the time between the samples, which is measured, R4, so that a
  round that runs late does not inflate the rate. The two sources wait one after the other, so `sensors` takes about
  400 milliseconds where it took about 200, and so does the first round of `run` when both kinds are mapped. One wait
  shared by both sources would need a change of the contract of a source for 200 milliseconds once, and is item 3 of
  "Beyond the request"
- **D8** Each interface is subtracted on its own and only over the interfaces in both samples, R8, and not as a
  difference of two sums, so that an interface that appears does not add all bytes it ever carried to one reading,
  and one that vanishes does not make the sum go back. A counter that went back is taken as the interface made anew,
  for instance by a reload of its driver, and adds nothing for one reading. Treating it as a wrap of a 32-bit counter
  and adding 2³² was rejected: the counters are 64 bits wide on the kernels of today, a 64-bit counter does not wrap
  in the life of a machine, and on a reset the wrap rule would add a huge false rate, while skipping one reading at
  worst shows a rate a little low once. An interface made anew under the same name with counters already above the
  old ones is not told apart, and gives one reading too low at most
- **D9** No interface in both samples makes the metrics unavailable, R9, and not 0: a machine whose `/sys/class/net`
  cannot be read, or that has no network card, has no rate to show, and an unavailable metric keeps the last value on
  the screen, as D3 of unit 0004 decided. An interface that counts and carries nothing gives 0, which is a true rate
- **D10** The network source reads the sysfs root and the proc root that exist already, so no new environment
  variable is needed, as D10 of unit 0004 argued for the program `nvidia-smi`. Both files are read from the namespace
  of the process. sensordeck runs on the host, and run in a container it would see the interfaces of the container
- **D11** The two rows of the network come after `nvme.<n>.temp` and before `commander.<n>.temp`, R1, so that the
  sources of Linux stay together and the controller, which the code of the commands adds after them, stays last.
  `ram.util` comes right after `ram.total`, with the other metrics of memory. `net.up` comes before `net.down`, as the
  tags 18 and 19 of the theme do
- **D12** The end-to-end tests check a rate of 0 exactly and a rate above 0 only as at least 1, R18, because the value
  depends on the time between two rounds of a real process, which a test cannot fix. 1048576 bytes give at least 1 for
  any time between two rounds below about 34 minutes. The exact values are checked in the unit tests with a clock the
  test gives, R15

## Non-goals

- Network metrics on Windows, and the share of RAM on Windows
- Rates of each interface, and choosing in the config which interfaces count
- Packets, errors and drops, and link speed or link state
- Traffic of other network namespaces than that of the process
- Mapping these metrics in the config of the human; which tile shows what is the human's choice
- Telling an interface made anew under the same name apart by its index

## Beyond the request

1. Rates of each interface, as `net.<name>.up` and `net.<name>.down` or numbered in the order of their device paths.
   It costs a new kind of row in the registry, with a name part that is no number, the checks of the config and tests.
   Without it only the sum of the machine is shown, which is what the tiles of the theme show
2. A second pair, such as `net.up.mib` and `net.down.mib` in MiB/s, for links whose traffic goes beyond the
   65535 KiB/s of D6. It costs two rows and tests. Without it a fast download shows 65535 on the screen, which the user
   sees, and `sensors` shows the true value
3. One wait of 200 milliseconds shared by `cpu.util` and the network, so that `sensors` and the first round of `run`
   take 200 milliseconds and not 400. It costs a change of the contract of a source, or a source that holds both, and
   tests. Without it `sensors` is about 200 milliseconds slower, once per run
