---
kind: feature
status: approved
timestamp: 2026-11-15T13:32:18Z
---

# The HID transport on Windows

## Log

- 2026-11-15T13:32:18Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for the first part of step 7 of the roadmap, against the README, `internal/hidraw`, the
  device wiring of the command line, `internal/commandercore`, `docs/protocol.md`, `docs/commander-core-xt.md`, and
  units 0015 and 0017 as merged. Decided: a thin layer of our own, after looking at the hidapi wrappers, which need
  cgo, `usbhid`, whose reads have no time limit, and `telesma-app/hid`, which is before v1 and needs a newer Go; the
  listing through `CM_Get_Device_Interface_List` of `golang.org/x/sys/windows` in place of the SetupDi calls the
  request names, so that no SetupDi structure is laid out by hand; interface 0 of the Commander by the `mi_00` field of
  the path, with its usage page, usage and lengths; the screen by its lengths of 65; the Windows errors carried as they
  are, with only the cancelled read made the timeout; a write limit of 5 seconds; `run` working on Windows with the
  Commander as its only source; and the Commander package split into its protocol and a node per system, its Linux
  bytes unchanged. The behaviours on a real Windows machine are listed as unverified, and the unit does not depend on
  them. The status is `draft`
- 2026-11-15T22:29:05Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec after the
  orchestrating session accepted D2 and the split of the Commander package, took items 1 and 4 of "Beyond the request"
  and declined items 2 and 3. Item 1 is R32 to R34: on Windows, collections whose IDs match but whose lengths or usage
  do not are named with what they gave, in `devices`, the error of no screen, `sensors` and the failed setup of a
  session. Item 4 is R35 to R38 with D15: a second job of the CI workflow on `windows-2025` builds and vets everything
  and tests the package of the Windows transport and the Commander package by name, with a test limited to Windows that
  runs the real listing; the Linux-only tests stay out by their build constraints, and the release gate covers the job
  through its call of the CI workflow. The status is `draft`
- 2026-11-15T22:29:31Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the protocol
  package of the screen and the command line's transport stay as they are, the split of `internal/commandercore` is
  needed for the Commander to run on Windows and is committed apart, and the pure parts are tested on Linux. D2 and
  R24 with D10 are accepted. Items 1 and 4 of "Beyond the request" were taken, because a worded near miss is what a
  failed check on Windows needs and a CI job on Windows is the only automatic check of that code, and the spec was
  amended for them; items 2 and 3 are declined. Everything on Windows is the human's to check and the unit does not
  depend on it. The approval that follows is made under that delegation
- 2026-11-15T22:29:31Z [human] approved the spec. The status is `approved`
