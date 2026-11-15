---
kind: feature
status: in-progress
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
- 2026-11-15T12:46:17Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: it checks what
  `run` does on SIGTERM against the code, keeps the calls of `systemctl` behind a seam the tests replace, and every
  requirement can be tested in the sandbox. D3, the unit always naming the config file, and D5, the sequence of
  `daemon-reload`, `enable` and `restart`, are kept: both make the service read the file `install` checked and
  take effect on a reinstall. Items 1 to 4 of "Beyond the request" are declined. Running the service on the real
  machine is the human's check and the unit does not depend on it. The approval that follows is made under that
  delegation
- 2026-11-15T12:46:17Z [human] approved the spec. The status is `approved`
- 2026-11-15T12:52:10Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote the plan against the tree of the
  branch. The spec fits the tree: `run` gives the errors of R7 through `readNamedFile` and `loadConfig`, and
  `os.UserConfigDir()` gives an absolute directory. The plan moves the prefix that `run` puts on an error of the load
  into one function that `install` shares, in a commit that keeps the behaviour of `run`; puts the calls, the path of
  the executable and the name of the system behind three fields of the environment; and decides the texts of the
  commands. It decides one case the spec leaves unnamed: a check of the presence of the unit file that fails other than
  by its absence fails `status` and `uninstall` with exit status 1 and makes no call. No item of "Beyond the request" is
  built
- 2026-11-15T13:20:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-15T13:21:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the plan in its order: the
  shared `loadConfigFile` first, in a commit that leaves `run` unchanged, then the pure functions of the unit file, the
  seam of `system`, `executable` and `systemctl` with `service install`, `uninstall`, `status` and the check of the
  system, the end-to-end tests with a fake `systemctl`, and the README section and status line. Deviations from the
  plan, none of which changes the spec: the code of `uninstall`, `status` and `supported` was written in the same
  commit as `install`, before the tests of those three, so that their tests were written against existing code and
  were not seen to fail first; the fake `systemctl` of the end-to-end tests checks that its first argument is `--user`
  and logs the arguments after it. No item of "Beyond the request" is built
- 2026-11-15T13:22:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, ran on the tree of `cbf3dfb`,
  whose code the later commits of this unit do not change: `go test ./...`, `go vet ./...`, `gofmt -l .`,
  `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...`. All five succeeded and
  reported nothing but passing packages. Not checked: a real systemd, the unit started on the developer's machine,
  lingering, and Windows, which D18 leaves to the human. The README claims nothing as seen on the device. The status
  stays `in-progress` for the review and the human
- 2026-11-15T12:56:49Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, completed a full review of
  `changes/0014-systemd-user-service` at `68633f8e4e375e9e6cb6d6fa89083aa090131952`, against merge base
  `3af1d14db3a5756e07c0675fb0b5c019d275b767` with `origin/master`. Returned P1 and P2 in
  `.reviews/0014-systemd-user-service/review-round-1.md`. Independently ran all five project suite commands,
  `go test -count=1 ./...` including the end-to-end tests, and `npm run check-changes`, all successfully.
  Could not verify a real systemd session, the physical screen and controller, lingering with udev access, or execution
  on Windows; D18 leaves the real-system checks for later.
- 2026-11-15T13:40:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, accepted finding P1 of the review
  of round 1, as accepted by the definition of the orchestrator and not as a decision of the human. The finding was
  right: the edit of `run.go` of the first commit of the shared load had not applied, so `run` still formatted the
  prefix itself while the plan and the record said it shared `loadConfigFile`. Commit a55253b makes the startup load of
  `run` call `loadConfigFile`; the reload path keeps its own error contract. On that tree `go test -count=1 ./...`
  (the end-to-end tests of the stop on SIGTERM included), `go vet ./...`, `gofmt -l .` and the Windows build and vet
  all succeeded
- 2026-11-15T13:41:00Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, proposes declining P2, which is
  non-blocking and not a requirement of the spec: the tests of uninstall, status and the other systems were written
  after their code, as the record already says. Nothing in the tree can change it without rewriting commits, which
  the review itself advises against, and the tests pass against the final code. The finding stays open until the human
  decides it

- 2026-11-15T12:58:22Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the proposed decline of
  P2 of round 1: declined. The tests of `uninstall`, `status` and the other systems were written after their code,
  against the order of the plan, but the order cannot be restored after the fact, the record states it, and the tests
  check the requirements against the final code, so none is left unchecked. P2 does not block
