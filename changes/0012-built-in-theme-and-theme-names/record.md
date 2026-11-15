---
kind: feature
status: approved
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
