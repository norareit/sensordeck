---
kind: feature
status: in-progress
timestamp: 2026-11-15T13:06:35Z
---

# Getting the repository ready to publish

## Log

- 2026-11-15T13:06:35Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for step 9 of the roadmap, against the README, `go.mod`, the command line, the preview of
  `internal/themeimage` and the ignored `dist/`. Decided: the README keeps its reference sections and R13 lists each
  change to its existing content; the preview image is checked by its decoded pixels, not its bytes; the version is in
  this unit, set as `v{{.Version}}` into `main.version` with a fallback to the build information, and printed by
  `--version`; the release is a draft with no changelog; GoReleaser writes outside `dist/`, which holds the device
  captures on the developer's machine; the actions and GoReleaser are pinned to exact release tags, with commit
  hashes left to "Beyond the request"; the credits also name the indirect modules of `go.mod`. The tests check what
  can be read from the configuration and the workflows, and running them is the human's first release, which the unit
  does not wait for. Nothing is pushed, tagged or released by this unit. The status is `draft`
- 2026-11-15T13:09:09Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, amended the spec on the decision of
  the orchestrating session under the delegation in `AGENTS.md`, which took items 2 and 5 of "Beyond the request" and
  declined items 1, 3, 4, 6, 7 and 8, which stay in the list marked as declined. Item 2 became R27 to R29 and D13: a
  committed `THIRD_PARTY_LICENSES` with the licence and notice files of the Go standard library and of each module of
  `go.mod`, written by a `go run` program from the module cache and `go env GOROOT`, compared with the committed file
  by a test, and put into each archive by R15. Item 5 became R19, R20 and D14: the release workflow calls the CI
  workflow as a reusable workflow, and the job that runs GoReleaser needs it, with the permissions granted per job.
  R12 and R24 follow both. The status stays `draft`
- 2026-11-15T13:10:23Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: it keeps the
  README's reference sections and lists each change to them, pins the release to the three platforms of the plan, and
  every requirement can be tested in the sandbox. Items 2 and 5 of "Beyond the request" were taken, because the
  licences of the linked modules require their notices in the archives and a release must not be built from a tag
  whose suite fails, and the spec was amended for them; items 1, 3, 4, 6, 7 and 8 are declined. The unit pushes, tags
  and releases nothing; the first runs of the workflows are the human's. The approval that follows is made under that
  delegation
- 2026-11-15T13:10:23Z [human] approved the spec. The status is `approved`
- 2026-11-15T13:15:04Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the unit's
  branch, in five commits: the version, the notices file, the preview image, the release configuration with the two
  workflows, and the README. Found in the tree: a Linux build leaves `github.com/inconshreveable/mousetrap` out of
  the module cache, so the CI workflow runs `go mod download` before the suite; `go list -m all` lists modules that
  are not in `go.mod`, so the module list is taken from `go.mod`; and cobra gives `--version` the new shorthand
  `-v`, which changes no existing flag. The sandbox cannot reach GitHub, so the pinned versions of the actions and of
  GoReleaser are releases known to exist and were not checked against GitHub. The status stays `approved`
- 2026-11-15T14:00:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-15T14:01:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the five pieces of the plan
  in five commits: the version (`--version`, `resolveVersion`, the variable `version` of `main`), the notices package
  and the committed `THIRD_PARTY_LICENSES` (eight parts, no toolchain version), `docs/minimal-dark.png` with its pixel
  test, `.goreleaser.yaml` with `ci.yml` and `release.yml` and their test, and the README. Work not in the plan: none.
  One departure in the code: `go list` in the generator runs with `-e`, so that a module that is not in the module cache
  is named in the generator's own error instead of failing `go list` with its message
- 2026-11-15T14:02:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accounted for R13: (1) the tagline
  now says Linux only; (2) the status note is replaced by the opening, whose Windows paragraph is the Windows sentence of
  R2 and whose last sentence on what the README is meant to become is gone; (3) "What it will do" is merged into the
  opening, keeping the "Windows only" introduction and the bullets on themes, the preview and the single executable, with
  the live readings bullet corrected to the metrics table (the Commander Core XT probes, no other motherboard sensors)
  and the service bullet saying the startup task on Windows is still to come; (4) "Supported devices" says the one
  screen is driven, keeps the table with `480x320` added in its Screen row, and names the Commander Core XT with
  `1b1c:0c2a`; (5) "Planned usage" became "Commands" with the sixteen commands, `--version` and `--help`, the six old
  lines kept, and the `service install` comment corrected; (6) the quick start moved after the new install section
  unchanged; (7) the credits name the seven modules with the licences their own licence files state: Apache 2.0 for
  cobra and mousetrap, BSD 3-clause for pflag and the three golang.org/x modules (image, sys, text; the plan said four,
  go.mod has three), MIT and Apache 2.0 for the YAML module; (8) the license section links `LICENSE`; (9) the install
  and release sections are new. The Windows sentence says that the theme commands that need no screen work there and
  names none, because `theme use` has no system gate in the code and was not checked on Windows. Checked: the nine
  identifiers of R8 each found with `grep -F`; every `](#...)` link of the README resolves to a heading; the diff
  against the merge base removes only lines of the sections named in R13. Not checked: the render of the README on
  GitHub
- 2026-11-15T14:03:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, noted that the pinned versions
  (`actions/checkout@v5.0.0`, `actions/setup-go@v6.0.0`, `goreleaser/goreleaser-action@v6.3.0`, GoReleaser `v2.8.0`) were
  not checked against GitHub from the sandbox, and that neither workflow nor GoReleaser was run. The tests read the
  files only (D10)
