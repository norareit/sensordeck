---
kind: feature
status: approved
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
- 2026-11-15T23:08:11Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the native
  metrics use the calls of the system, the facts of the LibreHardwareMonitor document are cited and marked confirmed or
  assumed, the shared computations move in a commit that keeps behaviour, and every requirement can be tested on Linux
  with a fake seam and documents written in the tests. LibreHardwareMonitor is MPL-2.0, so reading it was allowed, and
  no code of it is used. `CPU Package` for Intel stays in R21. Items 1 to 5 of "Beyond the request" are declined.
  Everything on Windows is the human's to check and the unit does not depend on it. The approval that follows is made
  under that delegation
- 2026-11-15T23:08:12Z [human] approved the spec. The status is `approved`
- 2026-11-15T23:15:29Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  `feature-0018-windows-sensors`, with every requirement traced to a task. Read the spec as follows where it leaves a
  detail to the plan: the host of R12 is the host without the port, so that `http://:8085/data.json` is refused; the
  line of R27 for a config that cannot be read or is not valid is the error of `run` without the prefix
  `sensordeck: `; the sentence of R2 on a procedure that cannot be found is kept for `GetIfTable2Ex` too, which
  `golang.org/x/sys/windows` binds with a panic, by recovering that panic alone as the failure of the call; and the
  proxy test of R40 also checks that the transport has no proxy, because Go never proxies a loopback address. The
  status stays `approved`
