---
kind: feature
status: draft
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

## Amendments
