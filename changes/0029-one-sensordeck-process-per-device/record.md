---
kind: bug
status: in-progress
timestamp: 2026-11-17T12:05:47Z
---

# One sensordeck process at a time on the controller and on the screen

## Log

- 2026-11-17T12:05:47Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of the
  orchestrating session and the observations on the human's machine of 2026-11-16 and 2026-11-17. The sequence was not
  reproduced in the sandbox, which has no device; the root cause is stated from the reported log lines and the code.
  The lock on Linux is `flock` on the device node rather than the lock file under `$XDG_RUNTIME_DIR` the request
  suggested, for the reasons of D2. The status is `draft`
- 2026-11-17T12:06:09Z [human] approved the spec. The status is `approved`
- 2026-11-17T12:11:19Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the branch. It
  adds the package `internal/devicelock`, takes the locks where the screen and the controller are opened, and words the
  refusals in the command line; the format of `/proc/locks` it parses was checked in the sandbox. The status stays
  `approved`
- 2026-11-17T12:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-17T13:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built tasks 1 to 7 of the plan on the
  branch. Verified on this tree: `go test ./...` (the e2e package took about 50 seconds), `go vet ./...`, `gofmt -l .`,
  `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` all succeed. The code for
  Windows (`lockfile_windows.go`, `device_windows.go`, its tests) is compiled and vetted only, not run: that, and the
  check of D10 on the device, are the human's. The README sections of the plan are written in the same commit as the
  e2e tests, not in a commit of their own
- 2026-11-17T13:41:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, logged these departures from the
  plan, none of which changes a requirement. `layout_test.go` lists the files that may import `golang.org/x/sys/windows`,
  and the two files of `internal/devicelock` are added to it. `sensors` after a `run` killed with SIGKILL fails to read
  the probes in the e2e test while the fake controller has the LED endpoint of channel 0 left open by the killed
  session, as the driver does not close channel 0 after a failed reading (unit 0005), so the test closes it in the fake
  before the reading, and checks that the lock is free (`values` succeeds and `sensors` writes no line of a controller
  in use). The first reading of a real controller after a killed session may fail the same way: not checked on a device.
  The test of the proc root `Dev` field uses the major and minor of `Stat_t`, checked in the e2e test against a fake
  `locks` and in the unit test against `/proc/locks`
- 2026-11-17T13:23:12Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, completed a full review of unit
  `0029-one-sensordeck-process-per-device` at `3df96a145202d8f77c7fa7a61bb879a776b18333` against merge base
  `b349c99c17ed0553beb318339fbf69897f22477f`. Returned S1 and P1; the complete review is saved in
  `.reviews/0029-one-sensordeck-process-per-device/review-round-1-20261117T132203Z.md`. Independently ran the five
  suite commands, an uncached `go test -count=1 ./...`, `npm run check-changes` and the diff whitespace check;
  all succeeded. Windows runtime behavior, the physical-device checks of D10 and the test-first red phase could
  not be verified. The status remains `in-progress`
- 2026-11-17T14:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accepted finding P1 of
  `review-round-1-20261117T132203Z.md` by the definition of the orchestrator, which accepts the findings the implementer
  agrees with, and not as a decision of the human. Added `TestTheLockEndsWithTheTeardownOfAFailedReadingAndOfAFailedSetup`
  to `internal/commandercore/session_test.go`: the lock of the controller is free after a setup that failed and after a
  keepalive that failed, and is held while the session is held. `go test ./internal/commandercore`, `go vet ./...` and
  `gofmt -l .` report nothing on the tree with this change
- 2026-11-17T14:11:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, proposes declining finding S1
  of the same review, as a proposal and not as a decision. What it says holds: the SIGKILL test closes the LED
  endpoint of channel 0 in the fake before `sensors`, which R19 does not say. The session leaves that endpoint open
  between keepalives, so after any kill of a session the next reading of `sensors` fails on the fake, whatever the lock
  does. Meeting R19 as written needs a change to the protocol of the reading (closing channel 0 first), which is beyond
  this unit and which the spec rules out of its scope, or a correction of R19 by the specifier. The lock itself is
  shown free by the test (`values` succeeds and `sensors` writes no line of a controller in use). The human decides
  between declining S1 with R19 read as it is tested, a correction of R19, or a new unit for the reading after a kill

## Amendments
- 2026-11-17T12:06:08Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its cause is the
  conflict the human and the orchestrating session saw on 2026-11-16 and 2026-11-17, the lock on the node itself (D2)
  is accepted over the lock file of the request because every process that opens the device sees it and the kernel
  releases it at any end, and every requirement can be tested in the sandbox with two processes against the fakes.
  Items 1 to 3 of "Beyond the request" are declined. The device check and Windows are the human's and the unit does
  not depend on them. The approval that follows is made under that delegation
