---
kind: feature
status: draft
timestamp: 2026-11-15T12:44:19Z
---

# Running `run` as a systemd user service

## Log

- 2026-11-15T12:44:19Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the request of
  the orchestrating session for the second half of step 8 of the roadmap, against the spec of unit 0013 and the code
  of `internal/cli`, the session of the controller and the end-to-end tests of the stop on SIGTERM. Decided: a user
  unit with `Type=exec`, `Restart=on-failure`, `RestartSec=10s`, no start limit, output to the journal and
  `WantedBy=default.target` with no other dependency; the unit always names the config file, because the user
  instance of systemd may not see the shell's `XDG_CONFIG_HOME`, which differs from the request; `install` checks the
  config with the load of `run`, theme compilation included, and then overwrites the unit file and makes the calls
  `daemon-reload`, `enable` and `restart`, rather than `enable --now`, so that running it again applies a moved
  executable; `status` reports three fields from `systemctl --user show`; the commands are present on other systems
  and fail with a message. `run` is not changed. The check on a real systemd is the human's and the unit does not wait
  for it. The status is `draft`
