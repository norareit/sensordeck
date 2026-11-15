---
kind: feature
status: in-progress
timestamp: 2026-11-15T08:25:25Z
---

# A reader of the theme file and the document of its format

## Log

- 2026-11-15T08:25:25Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for step 4 of the roadmap, from the facts of the format it learned by inspecting the bytes
  of three vendor theme files and checked by composing previews of them, carried in full in the spec so that the
  sandbox needs no vendor file, and from the README, `docs/protocol.md` and the command line in the tree. No code of
  any other project was read. The status is `draft`
- 2026-11-15T08:27:40Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: the
  facts of its context match the layout and the formats with which the orchestrating session composed correct previews
  of the three vendor samples, the guesses and unknowns are marked as such, and every requirement can be tested in the
  sandbox with synthetic files. No device check is needed. Items 1 to 6 of "Beyond the request" are declined: R8 and
  the truncation test of R22 cover the reader's robustness for this unit, and the others are small enough to come with
  the compiler if it needs them. The approval that follows is made under that delegation
- 2026-11-15T08:27:47Z [human] approved the spec. The status is `approved`
- 2026-11-15T08:32:01Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the unit's
  branch: a fake theme builder, the reader, the renderer, the dump and the command `theme dump`, each in its own
  package, then the end-to-end tests and the documents, with every requirement R1 to R24 traced to a task. The status
  stays `approved`
- 2026-11-15T08:33:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work on the unit's branch. The status is
  `in-progress`
- 2026-11-15T09:10:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the unit on its branch in
  seven commits: the reader and the fake theme, the images and the preview, the dump, a refactor that moves the
  unwrapping of the cause of a file error into `innerError` so that `theme dump` and `readNamedFile` share it (not in
  the plan, behavior preserving, and committed apart from the feature), the command, the end-to-end tests and the
  documents. Tests were written before the code for the reader, the images and the command; for `faketheme` and for
  the dump the code was written first and its tests right after, and the `theme.json` expected in the dump test was
  taken from the first output after checking it field by field against the bytes of `faketheme.Sample`. No new
  dependency. None of the items of "Beyond the request" was built. All tasks of `plan.md` are ticked
- 2026-11-15T09:12:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran the five commands of the suite
  on the final tree of the code and documents (`go test ./...`, `go vet ./...`, `gofmt -l .`, and the Windows build and
  vet): all succeeded with no output other than the passing packages. Not checked: a run on Windows, which `AGENTS.md`
  leaves to the human, and a dump of a real vendor theme file, which the sandbox does not have. The status stays
  `in-progress` until the review

## Amendments
