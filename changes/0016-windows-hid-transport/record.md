---
kind: feature
status: in-progress
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
- 2026-11-15T22:36:27Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  unit's branch, in eight commits after its own: two of refactoring that change no behaviour, the texts of the screen
  discovery shared in `internal/screenlist` and the Commander package split into its protocol, a `node` and a
  `Finder` with its Linux bytes and texts kept; then the Windows layer `internal/winhid` on a seam with its fake
  `internal/fakehid`, the seam on Windows, the Commander on Windows, the command line on Windows, the CI job, and the
  documents. Found in the tree: `windows.Errno` is an alias of `syscall.Errno`, so the constraint-free code names the
  three Windows codes as `syscall.Errno` constants that a test limited to Windows compares with those of
  `golang.org/x/sys/windows`; `LazyProc.Call` panics on a missing procedure, so each call follows its `Find` (R2);
  `layout_test.go` treats the Commander package as Linux-only as a whole and forbids the command line's Windows files
  to import it, which the split changes; and the keeper already writes a setup error that is not `errNoController`
  once, so R34 needs no change of it. The status stays `approved`
- 2026-11-15T22:45:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on the plan, task 1 first. The status is
  `in-progress`
- 2026-11-15T23:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in the eight
  commits of the plan after its own, `1ae1fa8` to `ce51e0a`, and ticked every box of `plan.md`. Two choices of the plan
  were left open and went as `## Changes while building` of `plan.md` says: the time limit as an argument of `complete`
  with no field for being in time, and the near misses of `Screens` given only when there is no screen. The commit
  trailer that the harness asks for was left out, as the standards say. Run on `ce51e0a`: `go test -count=1 ./...`
  passes, `go vet ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` report nothing, `gofmt -l .` prints nothing,
  `GOOS=windows GOARCH=amd64 go build ./...` and `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` succeed,
  `GOOS=darwin GOARCH=arm64 go build ./...` and `GOOS=freebsd GOARCH=amd64 go vet ./internal/cli` succeed,
  `GOOS=windows go test -c` builds the tests of `internal/winhid` and `internal/commandercore`, and
  `go list -test -deps` of both for Windows names none of `fakecommander`, `internal/hidraw` and `fakepty`. The diff
  against `origin/master` shows no change in `go.mod`, `go.sum`, `THIRD_PARTY_LICENSES` and `internal/smartmonitor`.
  Not checked: any run on Windows, among them `seam_windows_test.go`, the layouts of `HIDD_ATTRIBUTES` and `HIDP_CAPS`
  and the whole of R35 and R36, so every item of "Unverified until the human runs it" is still open. No review has
  run yet. The status stays `in-progress`
