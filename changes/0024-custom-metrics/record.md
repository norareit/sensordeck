---
kind: feature
status: in-progress
timestamp: 2026-11-16T20:28:31Z
---

# Custom metrics defined in the config file

## Log

- 2026-11-16T20:28:31Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session, against the README, `docs/theme-folder.md`, `internal/sensors`, `internal/config`, the
  sources of Linux and Windows, `nvidia-smi`, the commands `sensors` and `run` with the reload of `run`, and the specs
  of units 0004, 0013 and 0018. Decided: no prefix for custom names and no shadowing of a built-in one; the compiler
  checks the form and `run` refuses a theme metric that is neither built in nor defined; the `hwmon` form and `offset`
  are in the unit; commands run through `/bin/sh` or `cmd.exe` in the background, `every` 10 seconds by default and at
  least 1 second, a time limit of 10 seconds or `every`, a process group killed on Linux; a reload applies `metrics`;
  `sensors` reads the config on every system and takes `--config`; values go through `sensors.ToValue`; the flag
  `CREATE_NO_WINDOW` for every child that reads a metric, `nvidia-smi` included, is part of the unit. Six items are
  left under "Beyond the request". The status is `draft`
- 2026-11-16T20:30:46Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec at the decision of
  the orchestrating session under the delegation in `AGENTS.md`, which took item 1 of "Beyond the request" and
  declined items 2 to 6: the `hwmon` form has the key `index`, a whole number of 1 or more and 1 without the key, that
  takes the chip at that place in the order of the device paths. Fewer chips than `index` gives no value and the line
  of R23 with the number of chips found, not an error of the config. The README gives a command that lists the chips
  with their device paths in that order and tells a card from a processor's graphics by its PCI address, and the tests
  cover the key. The status stays `draft`
- 2026-11-16T20:31:31Z [human] approved the spec. The status is `approved`
- 2026-11-16T20:37:14Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the branch:
  ten commits, first the lookup of hwmon chips shared in a package that every system builds and the flag that keeps
  a console window from opening for `nvidia-smi` on Windows, then the name rule, the theme compiler, the key
  `metrics`, the package that reads custom metrics, `run`, `sensors`, the end-to-end tests and the documents. Every
  requirement traces to a task. The status stays `approved`
- 2026-11-16T20:45:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-16T22:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built tasks 1 to 10 of the plan in ten commits and
  ran the five commands of the suite on the final tree `8f15b26` plus this record: `go test ./...` (the unit tests
  and the end-to-end tests), `go vet ./...`, `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=windows GOARCH=amd64 go vet ./...` all reported nothing and passed. The tests of `custommetrics` also passed under
  `-race`. Deviations from the plan, all small: `runCommand` returns the number unscaled and the scale is applied by
  `scaled` in `outcome.go`, which the plan left open; the value of a number that overflows after the scale and the offset
  has no value with a cause of its own, a case the spec does not cover; `ReadOnce` runs the commands concurrently, so that
  several slow commands do not add up. Not checked: Windows, so the `shell_windows_test.go` and `nowindow_windows_test.go`
  tests are built by `go vet` and not run, and the three items under "Unverified until the human runs it". No review was
  run by this step, so the status stays `in-progress`
- 2026-11-16T20:50:21Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, reviewed `changes/0024-custom-metrics` at
  `8663d2ed51a9569df228267380b9e42edf785258` in a full review against merge base
  `f0f08d2f42be45ca1da9391d857ef1fd2315d6f6`. Returned S1 in
  `.reviews/0024-custom-metrics/review-round-1.md`. The five required commands, the change check, an uncached full
  test run and the custom-metrics race tests passed. Windows runtime behavior and the physical screen were not
  verified. The status stays `in-progress`
- 2026-11-16T22:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, applied finding S1 of the review
  round 1, accepted by the definition of the orchestrator and not as a decision of the human: the first line of
  standard error of a failed command is now the text up to the first line feed, with only a trailing carriage return
  removed, so an empty first line gives no line and an indented one keeps its indentation. The regression test failed
  before the fix and passes after it. The five commands of the suite passed on `da6cd3b`, the commit of the fix. The status
  stays `in-progress`

## Amendments
- 2026-11-16T20:31:31Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the three sources
  are defined with their failures and limits, a custom name can never change the meaning of a built-in one, the
  console flag for child processes on Windows covers `nvidia-smi` too, and every requirement can be tested in the
  sandbox with temporary files and fake commands. Item 1 of "Beyond the request" was taken, because every recent Ryzen
  desktop CPU has an `amdgpu` chip of its own beside an AMD card, and the spec was amended for it; items 2 to 6 are
  declined. What is open on Windows is the human's to check and the unit does not depend on it. The approval that
  follows is made under that delegation
