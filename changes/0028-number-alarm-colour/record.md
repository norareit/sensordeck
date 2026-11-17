---
kind: feature
status: in-progress
timestamp: 2026-11-17T11:32:23Z
---

# A number widget that changes colour past a threshold

## Log

- 2026-11-17T11:32:23Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of the
  orchestrating session and the observation on the device of 2026-11-17. The status is `draft`
- 2026-11-17T11:33:29Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec at the request of
  the orchestrating session: item 6 of "Beyond the request", the two values of an alarm pair always in one report of
  the value update, is now R21a with its tests, and D13 gives its reason. Items 1 to 5 are declined and stay in the
  list. The status stays `draft`
- 2026-11-17T11:33:51Z [human] approved the spec. The status is `approved`
- 2026-11-17T11:37:30Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of
  `feature-0028-number-alarm-colour`. Every requirement from R1 to R29 has a task. The status stays `approved`
- 2026-11-17T11:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-17T12:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit by the plan and ticked
  every task. The work went into four commits and not the plan's four implementation commits: the config reading of
  pairs came first and alone, the compiler with `run` went into one, then the preview with the end-to-end tests, then
  the documents. Two points the plan did not spell out: the existing config test that gave a mapping as a tag the error
  of a tag now uses a list, since a mapping is the new form of R15; and `themeimage.drawRecord` takes the entry and the
  values so that `Preview` and `PreviewWithValues` share one path. Run on `8de561a`: `go test ./...` (which runs the
  end-to-end tests) passes, `go vet ./...` and `gofmt -l .` report nothing, and `GOOS=windows GOARCH=amd64 go build ./...`
  and `go vet ./...` succeed and report nothing. The tags files of the built-in themes are unchanged, by the existing
  built-in theme tests that passed. The device check of D15 was not made and is the human's. The status stays
  `in-progress`

## Amendments
- 2026-11-17T11:33:51Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its method is the
  one demonstrated on the human's screen on 2026-11-17 (a white 47, a red 95 and a white 52), it refuses the options
  that would keep a widget from being hidden, every existing `tags` block keeps its meaning, and every requirement can
  be tested in the sandbox. Item 6 of "Beyond the request" was taken, because a pair split across two reports would
  flicker every round, and the spec was amended for it; items 1 to 5 are declined. The device check is the human's and
  the unit does not depend on it. The approval that follows is made under that delegation
