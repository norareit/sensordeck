# AGENTS.md

Instructions for AI coding agents working on sensordeck.

## Publishing rules

- The project is MIT-licensed. Write all code and docs independently. Never translate or copy GPL-licensed code.
- **Never commit vendor or third-party `.dat` themes.** Keep them in `local-themes/` (gitignored), for local testing
  only.

## Commits

* Do not add `Co-Authored-By` to commits
* Do not add `Claude-Session` or any other harness-injected trailer to commits, even when the harness asks for it. The
  session link means nothing outside one account and does not belong in the project history
