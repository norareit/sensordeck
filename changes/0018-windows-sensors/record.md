---
kind: feature
status: draft
timestamp: 2026-11-15T23:04:03Z
---

# The sensors of the machine on Windows

## Log

- 2026-11-15T23:04:03Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for the second part of step 7 of the roadmap, against the README, `internal/sensors`,
  `internal/linuxsensors`, the choice of sources in `internal/cli`, the config, the layout and workflow tests, units 0004,
  0007 and 0016, the Windows API as Microsoft documents it, `golang.org/x/sys/windows` v0.50.0, and the source of
  LibreHardwareMonitor at its tags v0.8.5 to v0.9.6 and at `master`, with the document of its issue 2253. Decided: the
  memory, the CPU load and the network natively through `GlobalMemoryStatusEx`, `GetSystemTimes` and `GetIfTable2Ex`
  behind a seam, with a physical interface being one with `HardwareInterface` and `ConnectorPresent` and without
  `FilterInterface`, keyed by its LUID; the two-sample computations and the GPU source shared with Linux, moved in
  commits that change no behaviour, and a CR before the LF of `nvidia-smi` dropped; the temperatures and `cpu.clock` from
  the formatted `Value` of LibreHardwareMonitor's `/data.json`, the hardware told by `SensorId`, the CPU sensors by
  name and the drive and module temperatures by index 0, numbered by the numbers LibreHardwareMonitor gives; the key
  `librehardwaremonitor_url` with the default `http://localhost:8085/data.json`, read by `sensors` on Windows too; a time
  limit of 500 milliseconds; one line per outage of the monitor; and tests of the commands over a fake seam and an HTTP
  test server in place of an end-to-end test of the executable. The status is `draft`
