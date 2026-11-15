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

## Amendments
