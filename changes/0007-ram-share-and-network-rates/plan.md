# The share of RAM in use and the network rates

## Approach

The unit adds three rows to the registry, extends the memory source with `ram.util`, and adds one network source to
`internal/linuxsensors`. All the new code of the sources is in files limited to Linux. No new package, no new
environment variable and no change of the contract of `sensors.Source` is needed.

* `internal/sensors/metrics.go`: three constants, `RAMUtilisation = "ram.util"`, `NetworkUp = "net.up"` and
  `NetworkDown = "net.down"`, and three fixed rows of `metricKinds`: `{prefix: RAMUtilisation, unit: "%"}` right after
  the row of `RAMTotal`, and `{prefix: NetworkUp, unit: "KiB/s"}` and `{prefix: NetworkDown, unit: "KiB/s"}` after
  the row of `nvme.<n>.temp` and before that of `commander.<n>.temp` (R1). The comment above the constants, which
  counts them as eight, says "The metrics of a fixed name" instead, so that it no longer holds a count
* `internal/linuxsensors/meminfo.go`: `memory.Read` reads when any of the three is wanted, reads the file once, and
  gives `ram.used`, `ram.total` and `ram.util` in that order from that one content (R2, R3)
* `internal/linuxsensors/netdev.go`, new and `//go:build linux`: the network source of R4 to R10, laid out below
* `internal/linuxsensors/sources.go`: the network source is the last of `Sources`, after `drives`, with
  `wait: time.Sleep` and `now: time.Now`, so that the commander source that `internal/cli/sources_linux.go` appends
  stays after it (R1, R11, D11). `internal/cli` does not change: `sensors` and `run` read every source by R15 and R21
  of unit 0004 already (R12)
* `internal/fakesensors/fakesensors.go`: two helpers for the trees of the tests of the network

### The network source

```go
// interfaceCounters is what one line of /proc/net/dev gives that matters: the bytes received and transmitted.
type interfaceCounters struct{ received, transmitted uint64 }

// networkSample is the counters of the interfaces that count, and the moment they were read.
type networkSample struct {
	moment     time.Time
	interfaces map[string]interfaceCounters
}

// network reads net.up and net.down as the bytes of the interfaces that belong to a device between two samples.
type network struct {
	sysfsRoot, procRoot string
	wait                func(time.Duration)
	now                 func() time.Time
	previous            *networkSample
}
```

* `parseNetDev(content string) map[string]interfaceCounters` is pure. For each line it cuts at the first colon
  (`strings.Cut`), passes over a line without one, takes the name as `strings.TrimSpace` of the text before, splits
  the rest with `strings.Fields`, passes over a line with fewer than 16 fields, and parses the first and the ninth
  field with the existing `isDigits` and `strconv.ParseUint(text, 10, 64)`, passing over the line when either fails.
  So the two header lines, which have no colon or no 16 numeric fields, are passed over by the same rule (R6). A name
  given twice keeps the last line; the kernel never writes one twice, and the spec says nothing of it
* `hasDevice(sysfsRoot, name string) bool` is true when `os.Lstat(<sysfs root>/class/net/<name>/device)` succeeds,
  so that an entry of any kind counts, a link, a directory or a file (R7)
* `(*network).take() (networkSample, bool)` reads `<proc root>/net/dev`, gives false when the file cannot be read,
  parses it, and keeps the interfaces for which `hasDevice` is true, with `moment: source.now()` taken right after the
  read. It decides `hasDevice` at every call (R6, R7, D4)
* `(*network).Read`:
    1. When neither `net.up` nor `net.down` is wanted, return nil without reading any file (R5)
    2. When `previous` is nil, take a sample; on failure return nil; otherwise keep it as `previous` and call
       `wait(sampleInterval)`, the constant of `procstat.go` (R5, R10)
    3. Take the current sample. On failure return nil and leave `previous` as it is (R5)
    4. Keep the old `previous`, and set `previous` to the current sample, whatever follows, so that the next reading
       measures from this one (R5, R8)
    5. `seconds := current.moment.Sub(old.moment).Seconds()`. For each interface of the current sample that the old
       one also has, add `current - old` of each direction to its sum when the current counter is not below the old
       one, counting that the interface is in both. A counter that went back adds nothing to its direction only (R8,
       D8). Subtract as `uint64` only after that comparison, so that nothing wraps
    6. When no interface is in both samples, or `seconds <= 0`, return nil (R9)
    7. Give `net.up` as `float64(transmitted) / seconds / 1024` and `net.down` as `float64(received) / seconds / 1024`,
       each only when wanted, `net.up` first, as the order of the table (R4, R9)

The source is a pointer in `Sources`, as `cpuUtilisation` is, so that it keeps `previous` for the whole run (R10).

### The test helpers

* `(*Tree).NetDev(content string)` writes `<ProcRoot>/net/dev`, making `net`, by writing a temporary file in the same
  directory and renaming it into place, so that a reader of the executable never sees a file half written. R18 needs
  the rename, and the other tests lose nothing by it
* `(*Tree).NetInterface(name, device string)` makes `<SysfsRoot>/class/net/<name>`, and when `device` is not empty, a
  link `device` in it to `<SysfsRoot>/devices/<device>`, which it makes, as `Hwmon` does

The network lines of the tests are built by a helper of the test files, `netDevLine(name string, received,
transmitted uint64) string`, which writes the name padded on the left, a colon, and 16 fields with the two given in
the first and the ninth place and 0 in the others, so that a test states only the bytes it means. The header lines are
those of the kernel:

```
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
```

### The end-to-end values

With the `meminfo` of R34 of unit 0004, `ram.util` is 100 × (96187828 − 73300372) / 96187828 = 23.79…, which
`sensors` prints as `23.8` and `run` sends as 24 (R17, R18). The `net/dev` of R17 does not change, so both rates are
0.0 in `sensors` and 0 in `run`.

The value report of R18 is checked by decoding its pairs, byte 1 being the count and each pair three bytes from byte 2,
tag and a big-endian value, as `smartmonitor.EncodeValues` writes them, rather than by comparing bytes, so that the
test does not depend on the order in which the three pairs are sent.

## Files

- `internal/sensors/metrics.go`: three constants and three rows
- `internal/sensors/metrics_test.go`: the new names, the refused examples and the units of the new rows
- `internal/linuxsensors/meminfo.go`: `ram.util` from the one content
- `internal/linuxsensors/meminfo_test.go`: the tests of R14, and the existing tests extended by `ram.util`
- `internal/linuxsensors/netdev.go`: new, the network source
- `internal/linuxsensors/netdev_test.go`: new, the tests of R15 and of `parseNetDev`
- `internal/linuxsensors/sources.go`: the network source in `Sources`
- `internal/linuxsensors/sources_test.go`: the order of the table with `ram.util` and the network
- `internal/fakesensors/fakesensors.go`: `NetDev` and `NetInterface`
- `e2e/sensors_test.go`: the network of R17 in `fullTree`, and the output of R17
- `e2e/commander_test.go`: the output of R20 of unit 0005 with the three lines of R17
- `e2e/run_test.go`: the test of R18
- `e2e/commands_test.go`: the comment of the suite, R19
- `README.md`: the three rows and the text of R13

## Tasks

- [ ] In `internal/sensors/metrics_test.go`, add `ram.util`, `net.up` and `net.down` to the accepted names of
  `TestIsName`, and `ram.util.x`, `net.upload` and `NET.UP` to the refused ones; add `"ram.util": "%"`,
  `"net.up": "KiB/s"` and `"net.down": "KiB/s"` to `TestUnitOfEachRow`, whose count of rows then fails until the rows
  exist; and add a test that the readings order of the table puts `ram.util` right after `ram.total` and `net.up`,
  `net.down` after `nvme.<n>.temp` and before `commander.<n>.temp`, by checking the positions of their rows in
  `metricKinds` (R1, R16)
- [ ] In `internal/sensors/metrics.go`, add the three constants and the three rows of "Approach", and change the
  comment above the constants as it says, until the tests of the first task pass (R1)
- [ ] In `internal/linuxsensors/meminfo_test.go`, extend `TestMemoryGivesUsedAndTotal` with the reading
  `{Name: "ram.util", Value: 100 * float64(96187828-73300372) / 96187828}` after `ram.total`. The existing
  `TestMemoryWithoutAvailableGivesTotalOnly` and `TestMemoryMalformedOrMissingGivesNothing` read with `everything`,
  so they cover `MemAvailable` missing and a malformed `MemTotal` for `ram.util` once it exists, and keep their
  expectations. Add a test that `MemTotal:       0 kB` with `MemAvailable:   0 kB` gives `ram.used` and `ram.total` of 0 and no
  `ram.util`; a test that for `hostMeminfo` the value of `ram.util` equals `100 * ram.used / ram.total` within 1e-9;
  and a test that wanting only `ram.util` gives only `ram.util` (R2, R3, R14)
- [ ] In `internal/linuxsensors/meminfo.go`, make `memory.Read` return nil only when none of the three is wanted, and
  after `ram.total` append `ram.util` as `100 * (float64(total) - float64(available)) / float64(total)` when it is
  wanted, `MemAvailable` was read and `total` is not 0, all from the one `content` read at the start, until the tests
  of the task before pass (R2, R3)
- [ ] Add `NetDev` and `NetInterface` to `internal/fakesensors/fakesensors.go`, as "The test helpers" says, with
  their doc comments, and extend the package comment to name the network
- [ ] Write `internal/linuxsensors/netdev_test.go`, `//go:build linux`, with `netDevLine`, the kernel's two header
  lines and a helper that makes the source of a tree with a wait that records its duration and writes the next
  content of `net/dev`, as `newUtilisation` of `procstat_test.go` does, and a clock that gives a list of moments the
  test sets, one for each sample, and fails the test when asked for more. The interfaces that count are made with
  `NetInterface(name, "pci0000:00/…")`. Cover, each in a test of its own (R15):
    - the first reading: two calls of the clock, one wait of exactly 200 milliseconds, and the values of `net.up` and
      `net.down` for 1 second between the moments
    - a second reading that uses the kept sample: one more call of the clock, no further wait, and the value against
      the sample of the first reading
    - a time between the samples of 2.5 seconds from the clock, while the wait records 200 milliseconds, deciding the
      value: 10240 bytes received give `net.down` of 4
    - two interfaces that count, `enp6s0` and `wlp5s0`, summed in both directions
    - `lo`, `docker0`, `veth1a2b3c`, `br-0123456789ab` and `tun0`, with entries under `class/net` and no `device`,
      whose bytes grow and add nothing next to `enp6s0`, which counts
    - a line of `enp6s0` written as `enp6s0:1234 …`, with no space after the colon, whose received bytes count
    - a line with 15 fields and a line whose ninth field is `12x`, each of an interface with a `device`, passed over,
      while the line of `enp6s0` counts
    - an interface that appears in the second sample and one that vanishes from it, which add nothing while
      `enp6s0` counts
    - a received counter of `enp6s0` that went back while its transmitted counter grew: `net.down` is 0 and `net.up`
      has its value; at the next reading the received bytes grown since the sample that went back give `net.down`
    - no `net/dev`: nothing, no wait and no call of the clock
    - no interface that counts, only `lo`: nothing
    - a time between the samples of 0: nothing; and the next reading, with time between, has a value, measured from
      the sample of the reading that gave nothing
    - a second reading whose `net/dev` was removed: nothing, and a third reading after it is written again measures
      from the sample of the first reading, which shows that the previous sample was kept (R5)
    - a reading that wants only `net.down`: only `net.down`
    - a reading that wants neither, with a `net/dev` in place: nothing, no wait, no call of the clock and `previous`
      still nil, which shows that no file was read, since `take` calls the clock after every read
  Add a test of `parseNetDev` on the two header lines and the lines of `lo` and `enp6s0`, which gives the two
  interfaces with their first and ninth fields, and a test that a counter above 2⁶⁴ − 1 passes the line over (R6)
- [ ] Write `internal/linuxsensors/netdev.go` as "The network source" says, until the tests of the task before pass
  (R4 to R9)
- [ ] Add `&network{sysfsRoot: sysfsRoot, procRoot: procRoot, wait: time.Sleep, now: time.Now}` as the last source of
  `Sources` in `internal/linuxsensors/sources.go`. In `sources_test.go`, add to the tree a `net/dev` with `enp6s0`
  and its `NetInterface`, and expect `ram.util` after `ram.total` and `net.up`, `net.down` after `nvme.1.temp` (R1,
  R10, R11)
- [ ] Run the five commands of the suite, of which the two for Windows show that the build for Windows does not change
  and that no file outside the Linux files names the new source (R11)
- [ ] In `e2e/sensors_test.go`, extend `fullTree` with `tree.NetDev` of the two header lines and the lines of `lo`,
  `enp6s0` and `docker0` with bytes other than 0, and `NetInterface` for the three, of which only `enp6s0` gets a
  device, `pci0000:00/0000:00:02.2/0000:06:00.0`. Add `ram.util\t23.8\t%` after `ram.total` and `net.up\t0.0\tKiB/s`,
  `net.down\t0.0\tKiB/s` after `nvme.2.temp` to the output of `TestSensorsListsTheReadingsOfTheHost`, and the same
  three lines in the same places to the output of `TestSensorsListsTheProbesOfTheController` in
  `e2e/commander_test.go`. `TestSensorsWithAnEmptyTreeSaysSo` keeps its output (R12, R17)
- [ ] In `e2e/run_test.go`, add `TestRunSendsTheShareOfRAMAndTheNetworkRates`: a fake screen as in `runningSetup`, a
  tree with the `meminfo` of `fullTree` and the network of the task before, made by a helper that both share, and the
  config `interval: 100ms` with `ram.util: 12`, `net.up: 18` and `net.down: 19`. A helper `valuePairs(report)
  map[uint8]uint16` decodes a value report as "The end-to-end values" says, and gives nil for a report whose byte 0 is
  not `0x02`. The test polls the reports for up to `reportLimit` until one holds (12, 24), (18, 0) and (19, 0); then
  calls `NetDev` with the received bytes of `enp6s0` 1048576 higher; then polls for up to 3 seconds for a later report
  that holds tag 19 with a value of 1 or more; then sends SIGTERM and expects status 0 (R12, R18)
- [ ] In the comment at the top of `e2e/commands_test.go`, add an item that the exact value of a rate other than 0,
  which depends on the time between two rounds, counters that go back, and interfaces that appear or vanish are
  covered by the tests of the network source in `internal/linuxsensors` (R19)
- [ ] In `README.md`, in the table of "Showing sensor values on the screen", add the row `ram.util` with `%` and the
  source "`/proc/meminfo`, the share of the total that `ram.used` is" after `ram.total`, and the rows `net.up` and
  `net.down` with `KiB/s` and the sources "`/proc/net/dev`, the bytes sent" and "`/proc/net/dev`, the bytes received"
  after `nvme.<n>.temp`. Below the table, after the paragraph of the GPU and numbered metrics, add a paragraph that
  `net.up` and `net.down` are the sum over the network interfaces that belong to a device, so that loopback and
  virtual interfaces such as those of Docker, bridges and VPNs are left out, that a KiB is 1024 bytes, and that the
  rates are measured between two readings, the first of which waits a moment for the second (R13)

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [ ] `go test -count=5 -run TestRunSendsTheShareOfRAMAndTheNetworkRates ./e2e` passes all five times, so that the
  test of R18 does not depend on timing
- [ ] `npm run check-changes` passes

## Commit Points

- [ ] `feature-0007 Write the plan`, with `plan.md` and the entry of `record.md`
- [ ] `feature-0007 Add the share of RAM and the network rates to the metrics`, the first two tasks
- [ ] `feature-0007 Read the share of RAM in use`, the third and fourth tasks
- [ ] `feature-0007 Read the network rates`, the fifth to the ninth tasks
- [ ] `feature-0007 Test the share of RAM and the network rates end to end`, the tenth to the twelfth tasks
- [ ] `feature-0007 Document the share of RAM and the network rates`, the README
