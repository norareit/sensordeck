---
kind: feature
status: in-progress
timestamp: 2026-11-14T14:51:19Z
---

# Device protocol

## Log

- 2026-11-14T14:51:19Z [Claude Code, anthropic/claude-fable-5-2] as the specifier, wrote the spec. The status is `draft`
- 2026-11-14T14:53:55Z [Claude Code, anthropic/claude-fable-5-2] read the spec under the delegation in `AGENTS.md`: every known
  fact of the protocol is in it, the byte examples and CRC values of R10, R13, R14 and R16 were computed again and
  hold, and every requirement can be tested without the device. Nothing was changed. None of the six items of
  "Beyond the request" is taken: item 1 is decided once the device has shown what it answers, and items 2 to 6 have
  no user before there is a command line. The approval that follows is made under that delegation
- 2026-11-14T14:53:55Z [human] approved the spec. The status is `approved`
- 2026-11-14T15:01:56Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  branch. Every requirement R1 to R29 has a task, and the byte examples and CRC values of R13, R14 and R16 were
  computed again and hold. The status stays `approved`
- 2026-11-14T15:02:30Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-14T15:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit in the four commits
  of the plan and ticked every task. On the tree of `9b9f6f6`, `go test ./...` passes, `go vet ./...` reports nothing
  and `gofmt -l .` prints no file name. `go list -deps ./internal/smartmonitor` lists no package outside the standard
  library, there is no `go.sum`, and the grep of the plan for `time.Sleep`, `time.After`, `time.NewTimer` and
  `time.Tick` under `internal/` finds nothing. The tests of the upload were written after `upload.go`, not before it,
  which departs from the order of the plan. Not checked: any behavior on the real device, which this unit does not
  cover
- 2026-11-14T15:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, noted that the commits carry no
  `Co-Authored-By` trailer, because the standards forbid it. The status stays `in-progress`

## Amendments
