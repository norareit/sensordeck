---
kind: feature
status: draft
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
