---
kind: feature
status: in-progress
timestamp: 2026-11-15T09:21:42Z
---

# A built-in theme and themes found by name

## Log

- 2026-11-15T09:21:42Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for step 6 of the roadmap, against the specs of `0010-theme-file-reader` and
  `0011-theme-compiler`, `docs/theme-folder.md`, the README, the theme commands and the code of `run`. The `theme.yaml`
  of `minimal-dark` is carried verbatim from the file the orchestrating session designed. Its background prototype,
  which is not committed, was decoded pixel by pixel: it holds three colours and no anti-aliasing, and the rule of R4
  gives every one of its pixels. In the code of `run` a metric that cannot be read is left out of the value update, so
  the README is asked to say that and not that the tile shows blank or 0, which nothing in the project establishes. The
  status is `draft`
- 2026-11-15T09:24:28Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the theme and its
  background are carried in full and match the prototype whose preview the orchestrating session checked, the rule of
  folder or name is decided from the argument alone for a stated reason, the README states truthfully what `run` does
  with a metric it cannot read, and every requirement can be tested in the sandbox. The device check is the human's and
  the unit does not depend on it. Items 1 to 4 of "Beyond the request" are declined: none is needed for the theme or
  the lookup to be correct. The approval that follows is made under that delegation
- 2026-11-15T09:24:28Z [human] approved the spec. The status is `approved`
- 2026-11-15T09:27:59Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote `plan.md` against the tree of the unit's
  branch. Before writing it, the `theme.yaml` of R2 was compiled in the sandbox with the current `theme compile` against
  a background drawn by the rule of R4: it compiles, its dump gives no warning, and its tags file holds the mapping of
  R6, so the spec fits the tree. Every requirement has a task and every task names the requirements it delivers. None
  of the items of "Beyond the request" is planned, as decided before the approval. The status stays `approved`
- 2026-11-15T09:35:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-15T10:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built every task of `plan.md`. Two
  deviations from the plan: `theme compile`, `theme preview` and `theme list` were committed together as one commit
  and not two, because they share `themes.go` and `theme.go`; and the end-to-end tests add `runIn` to
  `e2e/main_test.go`, which `runWith` now calls, as the plan allowed. A commit trailer that the harness asks for was
  left out, as the standards forbid it. On the final tree, `go test ./...` (which runs `e2e/`), `go vet ./...`,
  `gofmt -l .` (no file), and the Windows build and vet all succeed, and `go generate ./internal/builtinthemes/` leaves
  `background.png` unchanged. Not checked: the screen showing the theme, and anything on Windows, which are the human's
- 2026-11-15T12:03:07Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, completed a full review of unit
  `0012-built-in-theme-and-theme-names` at `60c92cb`, against merge base `debe4e9` with `origin/master`.
  Returned P1, C1 and C2 in `.reviews/0012-built-in-theme-and-theme-names/round-1-20261115T120307Z.md`.
  The five required suite commands, an uncached `go test -count=1 ./...` including end-to-end tests,
  `npm run check-changes` and the diff whitespace check pass. Not verified: generation by rerunning the writing
  command, the physical screen, or Windows runtime behavior. The status stays `in-progress`
