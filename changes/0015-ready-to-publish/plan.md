# Plan: getting the repository ready to publish

## Approach

The work comes in five pieces, each committed on its own: the version, the notices file, the preview image, the
release configuration with its two workflows, and the README. The README comes last, because it names the files,
directories and commands that the other pieces create.

What the tree on `feature-0015-ready-to-publish` holds, and what the plan builds on:

* `main.go` calls `cli.Main()`, which builds an `environment` and calls `run(args, env)`. `newRootCommand` makes the
  cobra root, and `harness.run` in `internal/cli/commands_test.go` fills the same `environment` with fakes
* `internal/builtinthemes/background/generate/main.go` is the pattern for a `go run` program: `main` calls
  `run(arguments, stderr)`, which takes the output path as its one argument
* `e2e/builtin_theme_test.go` shows how the preview of the built-in theme is drawn: `themecompile.Compile` of
  `builtinthemes.Folder("minimal-dark")`, `themefile.Parse` of the result, then `themeimage.Preview`
* `layout_test.go` is a test of package `main` at the module root, and `go.yaml.in/yaml/v3` is already required
* The root files of the modules of `go.mod` in the module cache are: `LICENSE.txt` for cobra; `LICENSE` and `NOTICE`
  for `go.yaml.in/yaml/v3`; `LICENSE` for each of the other five. The Go toolchain's is `$(go env GOROOT)/LICENSE`
* `go list -m all` also lists modules that are not in `go.mod` (`go-md2man`, `blackfriday`, `x/mod`, `x/sync`,
  `x/tools`, `check.v1`), with no directory. The list of modules is therefore taken from `go.mod` with
  `go mod edit -json`, and only the directories are looked up with `go list -m -json`
* A Linux build does not download `github.com/inconshreveable/mousetrap`, which only the Windows build of cobra
  imports, and `go list -m` gives it no directory until `go mod download` has run. This was checked with an empty
  module cache. The CI workflow therefore runs `go mod download` before the suite, and the generator's error for a
  module not in the module cache names `go mod download` as the remedy
* Cobra, when the root command's `Version` is set, adds the flag `--version` with the shorthand `-v` to the root
  only, and prints its default line `sensordeck version <version>` and a newline to the command's standard output. No
  existing flag has the shorthand `-v`, so the shorthand is new and changes no existing flag (R23)

### Choices the spec leaves to the plan

* **The version reaches the command line** as the parameter of `cli.Main(linked string)`. `cli.Main` resolves it into
  a new field `version` of `environment`, and `newRootCommand` sets `root.Version` from it. The resolution is a pure
  function `resolveVersion(linked string, info *debug.BuildInfo, ok bool) string`, and `cli.Main` passes it what
  `debug.ReadBuildInfo()` gives
* **The notices code** is the package `internal/notices`, with the program in `internal/notices/generate/main.go`. The
  command of R28 is `go run ./internal/notices/generate THIRD_PARTY_LICENSES`, run at the module root. The package
  holds:
    * the pure `LicenceFileNames(names []string) []string`, which keeps the names that start with `LICENSE`,
      `LICENCE`, `COPYING` or `NOTICE`, case-sensitively, sorted by byte order, without changing its argument
    * the pure `Render(standardLibrary string, modules []Module) (string, error)`, where `Module` has `Path`,
      `Version` and `Files []File`, and `File` has `Name` and `Text`. It sorts a copy of the modules by path, and fails
      with an error that names the module when one has no file
    * `Generate(moduleRoot string) (string, error)`, in a file of its own, which runs `go mod edit -json`,
      `go list -m -json <paths>` and `go env GOROOT` with the module root as their directory, reads the root
      directory of each module, takes its files through `LicenceFileNames`, reads them and calls `Render`. It fails
      naming the module when its directory is empty. The program and the test of R29 both call it
* **The format of `THIRD_PARTY_LICENSES`**: the parts follow one another, separated by a blank line, a line of 80 `=`
  and a blank line. A part opens with its header line, which is `Go standard library` for the first part and
  `<path> <version>` for a module, as `github.com/spf13/cobra v1.10.2`. Each file follows as a blank line, the file's
  name, a line of `-` as long as the name, a blank line, and the text unchanged. When a text does not end with a
  newline, one is written after it. The file ends with one newline after the last text
* **The output directory of GoReleaser** is `release/`, ignored by git. Nothing in the tree uses that name now
* **The pinned versions** are `actions/checkout@v5.0.0`, `actions/setup-go@v6.0.0`,
  `goreleaser/goreleaser-action@v6.3.0` and GoReleaser `v2.8.0`. The sandbox cannot reach GitHub, so these were
  taken as releases known to exist and were not checked against the latest release. The setup of Go uses
  `go-version-file: go.mod`, so that the version is the one `go.mod` names and is written in one place
* **The three platforms** are written as `goos: [linux, windows]`, `goarch: [amd64, arm64]` and an `ignore` of
  windows/arm64. The test computes the product less the ignored pairs and compares it with the three of R14
* **The archives** use the key `formats` of GoReleaser 2, `[tar.gz]` by default and `[zip]` in a `format_overrides`
  entry for windows, and set no `name_template`. The checksum section sets `algorithm: sha256` and no
  `name_template`
* **The gofmt step** of CI runs `files=$(gofmt -l .)`, prints the files and exits with 1 when the list is not empty
* **The tests of R24** are in `release_test.go` at the module root, in package `main`, and parse the three files into
  typed structs with `go.yaml.in/yaml/v3`. The test of R3 is in `docs_test.go` at the module root

## Files

* `main.go`, `internal/cli/main.go`, `internal/cli/version.go` (new), `internal/cli/version_test.go` (new),
  `internal/cli/commands_test.go`, `e2e/version_test.go` (new)
* `internal/notices/notices.go`, `internal/notices/generate.go`, `internal/notices/notices_test.go`,
  `internal/notices/generate/main.go` (all new), `THIRD_PARTY_LICENSES` (new)
* `docs_test.go` (new), `docs/minimal-dark.png` (new)
* `.goreleaser.yaml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `release_test.go` (all new),
  `.gitignore`
* `README.md`
* `changes/0015-ready-to-publish/record.md`

## Tasks

### 1. The version

- [x] In `internal/cli/version_test.go`, write the failing unit tests of `resolveVersion`: a linked `v1.2.3` is given
  as it is whatever the build information says; with no linked version, the build information's main module version,
  such as `v0.1.0`, is given; with neither, `(devel)` is given, both for build information whose main module version
  is empty and for `ok` false with nil information (R22, R25)
- [x] In `internal/cli/commands_test.go`, give `harness` a field `version`, pass it into the `environment` of
  `harness.run`, and add a test that `--version` with the version `v1.2.3` writes exactly
  `sensordeck version v1.2.3\n` to standard output, nothing to standard error, and gives status 0 (R22, R25)
- [x] Add `internal/cli/version.go` with `resolveVersion`, add the field `version` to `environment`, change `Main` to
  `Main(linked string) int` and fill the field with `resolveVersion(linked, debug.ReadBuildInfo())`, and set
  `root.Version` in `newRootCommand` from `env.version` (R22)
- [x] In `main.go`, declare `var version string`, with a comment that the release build sets it with
  `-X main.version`, and call `cli.Main(version)` (R22)
- [x] Add `e2e/version_test.go`, with the `linux` build line of the other end-to-end tests, which runs the built
  executable with `--version` and checks that standard output matches `^sensordeck version \S+\n$`, that standard
  error is empty and that the status is 0 (R25)
- [x] Check that every existing test passes unchanged, so that no existing command or flag changed (R23)
- [x] Run the suite and commit: `feature-0015 Print the version with --version`

### 2. The notices file

- [x] In `internal/notices/notices_test.go`, write the failing unit tests with plain values (R29):
    * `LicenceFileNames` keeps `LICENSE`, `LICENSE.txt`, `LICENCE`, `COPYING` and `NOTICE`, drops `license`,
      `README.md`, `go.mod` and `MyLICENSE`, gives them sorted, and leaves its argument unchanged
    * `Render` puts the standard library part first and the modules in the order of their paths whatever the order of
      the input, opens each part with its header line, writes each file under its name and its line of `-`, keeps a
      text byte for byte, adds a newline only after a text that has none, and separates the parts as the format above
      says. One test compares a whole small output with a literal string
    * `Render` fails for a module with no file, with an error that names the module's path, and gives no text then
- [x] Add `internal/notices/notices.go` with `Module`, `File`, `LicenceFileNames` and `Render`, and no I/O (R27, R29)
- [x] Add `internal/notices/generate.go` with `Generate(moduleRoot string)`, as the choice above says. A module with
  an empty directory fails with `<path> <version> is not in the module cache; run go mod download`, and one with no
  licence file fails through `Render` (R27, R28)
- [x] Add `internal/notices/generate/main.go`, on the pattern of the background generator: its one argument is the
  output path, it calls `Generate` with the working directory as the module root, and it writes the file only after
  `Generate` succeeded, so that a failure writes nothing and leaves an existing file as it was (R28)
- [x] Add to `internal/notices/notices_test.go` the test that calls `Generate("../..")` and fails when its result
  differs from `../../THIRD_PARTY_LICENSES`, or when the file cannot be read, with a message that says to run
  `go run ./internal/notices/generate THIRD_PARTY_LICENSES` at the module root (R29)
- [x] Run `go run ./internal/notices/generate THIRD_PARTY_LICENSES` and read the result: eight parts, the standard
  library first, then the seven modules of `go.mod` in the order of their paths, with `LICENSE` and `NOTICE` for
  `go.yaml.in/yaml/v3` and `LICENSE.txt` for cobra, and no Go toolchain version anywhere (R27)
- [x] Run the suite and commit: `feature-0015 Generate and check the third-party notices`

### 3. The preview image

- [x] Add `docs_test.go` with a test that draws the preview of the built-in theme `minimal-dark` as
  `e2e/builtin_theme_test.go` does, decodes `docs/minimal-dark.png`, and fails when the bounds differ or when any
  pixel differs once both are converted with `color.RGBAModel`, naming the first pixel that differs. Every failure,
  the file missing included, says to write the image again with
  `sensordeck theme preview minimal-dark docs/minimal-dark.png` (R3)
- [x] Write the image with `XDG_CONFIG_HOME=$(mktemp -d) go run . theme preview minimal-dark docs/minimal-dark.png`,
  so that no installed theme of the same name is taken instead of the built-in one, and see the test pass (R3)
- [x] Run the suite and commit: `feature-0015 Add the preview of minimal-dark for the README`

### 4. The release configuration and the workflows

- [x] Write `release_test.go`, failing because the three files are missing, with one test per file (R24):
    * `.goreleaser.yaml`: `version: 2`; one build, with `main: .`, whose platforms, computed as the choice above
      says, are exactly linux/amd64, linux/arm64 and windows/amd64; `CGO_ENABLED=0` in its `env`; `-trimpath` in its
      `flags`; its ldflags exactly `-s -w -X main.version=v{{.Version}}`; one archive whose `files` are exactly
      `README.md`, `LICENSE`, `THIRD_PARTY_LICENSES` and `udev/70-sensordeck.rules`, whose `formats` is `[tar.gz]`,
      whose only `format_overrides` entry gives windows `[zip]`, and which sets no `name_template`; a checksum
      section with `algorithm: sha256` and no `name_template`; `release.draft: true`; `changelog.disable: true`; and
      a `dist` that is set and is neither `dist` nor `dist/` (R14 to R18)
    * `.github/workflows/release.yml`: the only trigger is `push` with `tags: ["v*"]` and no other key; the top-level
      `permissions` is present and empty; a job whose `uses` is `./.github/workflows/ci.yml`, whose permissions are
      exactly `contents: read`, and that file exists; a job that `needs` it, has exactly `contents: write`, has a
      checkout step with `fetch-depth: 0`, a setup-go step with `go-version-file: go.mod`, and a step that uses
      `goreleaser/goreleaser-action` with `args: release --clean` and a `version` that matches
      `^v\d+\.\d+\.\d+$`; and every `uses` of a step names a tag that matches `@v\d+\.\d+\.\d+$` (R19, R21)
    * `.github/workflows/ci.yml`: the triggers are exactly `push`, `pull_request` and `workflow_call`; the top-level
      permissions are exactly `contents: read`; its job runs on `ubuntu-24.04`; a setup-go step with
      `go-version-file: go.mod`; for each of `go test ./...`, `go vet ./...`, `gofmt -l .`,
      `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` a step whose `run`
      holds it; and every `uses` of a step names an exact release tag (R20, R21)
- [x] Add `.goreleaser.yaml` as the choices above say, with `project_name: sensordeck`, the build `id` and `binary`
  `sensordeck`, and `dist: release` (R14 to R18)
- [x] Add `release/` to `.gitignore`, under a comment that it is the output of GoReleaser, and check with
  `git check-ignore release/x` (R18)
- [x] Add `.github/workflows/ci.yml` with one job on `ubuntu-24.04`: checkout, setup-go, a `go mod download` step with
  a comment that the notices test needs the Windows-only module in the module cache, then the five commands of the
  suite as five steps, in the order `AGENTS.md` lists them, the gofmt step as the choice above says (R20, R21)
- [x] Add `.github/workflows/release.yml` with `permissions: {}` at the top, the job `suite` that calls the CI
  workflow, and the job `release` on `ubuntu-24.04` that needs it, with the three steps named in the test and
  `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` in the environment of the GoReleaser step (R19, R21)
- [x] Run the suite and commit: `feature-0015 Add the GoReleaser configuration and the CI and release workflows`

### 5. The README

Each of these tasks keeps the reference sections of R9 unchanged, and every change belongs to one item of R13.

- [x] Change the tagline to name Linux as the system it works on now, with no claim of Windows (R4, R13.1)
- [x] Replace the status note and "What it will do" with the opening of R2, before any other section: what
  sensordeck does on Linux, in the present tense; the introduction on screens sold as "Windows only", kept; the
  bullets of "What it will do" in the present tense, the live readings corrected to those the metrics table lists
  and the service bullet saying systemd on Linux and the startup task on Windows as still to come; the Windows
  sentence of R2; and `docs/minimal-dark.png` with an alt text and a sentence that it is the preview
  `sensordeck theme preview` draws with sample values and not a photo of the screen. The sentence that the README
  describes what sensordeck is meant to become goes (R1, R2, R13.2, R13.3)
- [x] In "Supported devices", replace the sentence on version 1 with one saying that sensordeck drives this one
  screen, keep the table and the last sentence, and add `llhmi`, `llTechCo`, `SmartMonitor`, `3.5 inch` and
  `480x320` beside `480×320` where the table does not hold them as written, and the Commander Core XT with
  `1b1c:0c2a` as the one temperature controller whose probes sensordeck reads (R8, R13.4)
- [x] Add the install section before the quick start, with the three routes of R5 in their order. The archives are
  named `sensordeck_<version>_linux_amd64.tar.gz`, `sensordeck_<version>_linux_arm64.tar.gz` and
  `sensordeck_<version>_windows_amd64.zip`, with `<version>` the tag without its `v`; the check is
  `sha256sum --check --ignore-missing sensordeck_<version>_checksums.txt`; the udev rule is
  `udev/70-sensordeck.rules` in the archive, with a link to `#linux-access-to-the-screen`; `go install` names Go
  1.27; the build from source is `CGO_ENABLED=0 go build -trimpath -o sensordeck .` after the clone (R5, R13.9)
- [x] Move the quick start after the install section, unchanged (R6, R13.6)
- [x] Replace "Planned usage" with the commands section of R7: one code block with the sixteen commands, `--version`
  and `--help`, each with a short comment, keeping the six old lines, with the comment of `service install` saying it
  runs `run` as a systemd user service that starts at login (R7, R13.5)
- [x] Add the release section as the last section before the credits, with the points of R12: CI on pushes and pull
  requests; a tag `v<major>.<minor>.<patch>` runs the suite and, only when it succeeds, builds the archives and
  the checksum file into a draft release; the draft is checked and published by hand on GitHub;
  `goreleaser release --snapshot --clean` writes into `release/`; and
  `go run ./internal/notices/generate THIRD_PARTY_LICENSES` writes the notices file again after a dependency changed
  (R12, R13.9)
- [x] Extend the credits: keep the thanks with its link and the sentence on the glyphs, and name the seven modules of
  R10, each with the licence its own licence file states, read from the module cache: Apache 2.0 for cobra and
  mousetrap, BSD 3-clause for pflag and the four `golang.org/x` modules, with the Go fonts under that of
  `golang.org/x/image`, and MIT and Apache 2.0 for `go.yaml.in/yaml/v3` as its `LICENSE` says. Add that
  `THIRD_PARTY_LICENSES` holds their texts (R10, R13.7)
- [x] Change the license section to say the project is under the MIT licence, linking to `LICENSE` (R11, R13.8)
- [x] Check the README: `git diff master -- README.md` shows no change in the six sections of R9; each identifier of
  R8 is found with `grep -F`; the links to `docs/theme-folder.md` and `docs/theme-format.md` are where they were;
  every `](#...)` resolves to a heading; and no sentence states as working what the code does not do (R1, R8, R9,
  R13)
- [x] Run the suite and commit: `feature-0015 Rewrite the README for publishing`

### 6. Closing

- [x] Log in `record.md` how each item of R13 was carried out, and that the pinned versions were not checked against
  GitHub from the sandbox (R13, R21)
- [x] Run the suite and `npm run check-changes`, and commit the record: `feature-0015 Record the implementation`
  (R26)

## Trace

| Requirement | Tasks |
|---|---|
| R1, R2 | 5.2, 5.10 |
| R3 | 3.1, 3.2 |
| R4 | 5.1 |
| R5 | 5.4 |
| R6 | 5.5 |
| R7 | 5.6 |
| R8 | 5.3, 5.10 |
| R9 | 5.10 |
| R10 | 5.8 |
| R11 | 5.9 |
| R12 | 5.7 |
| R13 | 5.1 to 5.10, 6.1 |
| R14 to R18 | 4.1, 4.2, 4.3 |
| R19 | 4.1, 4.5 |
| R20 | 4.1, 4.4 |
| R21 | 4.1, 4.4, 4.5 |
| R22 | 1.1 to 1.4 |
| R23 | 1.6 |
| R24 | 4.1 |
| R25 | 1.1, 1.2, 1.5 |
| R26 | 6.2, and the suite at each commit |
| R27 | 2.2, 2.3, 2.6 |
| R28 | 2.3, 2.4 |
| R29 | 2.1, 2.5 |
