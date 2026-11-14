---
kind: task
status: approved
timestamp: 2026-11-14T14:50:00Z
---

# Go in the sandbox

sensordeck is written in Go, and the units that build it run in the sandbox, whose image holds no Go toolchain and
whose firewall admits neither the Go module hosts nor the host of the reviewer's model. This task makes the sandbox
able to build and test Go code and to run the orchestrator's default definition:

* `sandbox/Dockerfile` is added, built on the kit's image. It installs Go 1.27.4 from the official archive of
  `go.dev`, for `amd64` and `arm64`, checks the archive against its published SHA-256, and puts `go` and `gofmt` on
  the `PATH` of the agent
* `ALLOWLIST` in `sandbox/compose.project.yml` gains `proxy.golang.org` and `sum.golang.org`, for the download and
  the verification of Go modules, and `api.openai.com`, for the reviewer that the default definition runs in
  OpenCode on an OpenAI model
* A unit run of the sandbox runs `go version` and reports 1.27.4, and `go mod download` of a module with one
  dependency succeeds in it

The task adds no Go code and no `go.mod`. Those come with the first feature.

## Log

- 2026-11-14T14:50:00Z [Claude Code, anthropic/claude-fable-5-2] as the specifier, wrote the record of the task. The
  status is `draft`
- 2026-11-14T14:53:55Z [Claude Code, anthropic/claude-fable-5-2] read the record under the delegation in `AGENTS.md` and found nothing
  to change. The approval that follows is made under that delegation
- 2026-11-14T14:53:55Z [human] approved the record. The status is `approved`

## Amendments
