# Getting the repository ready to publish

## Context

Step 9 of the roadmap of the initial plan is publishing: a README that carries every identifier people search for when
they own one of these screens, the credit to the project that discovered the protocol, the udev rule, and the
distribution the plan describes, which is GitHub Releases with prebuilt archives built by GoReleaser on a version tag,
and `go install` for people who have Go.

Units 0001 to 0014 are implemented. On Linux sensordeck drives the screen with the low-level commands, reads the
sensors and the probes of a Corsair iCUE Commander Core XT, compiles and previews themes, ships the built-in theme
`minimal-dark`, uploads the theme of the config file only when it is not on the screen, and runs `run` as a systemd user
service. On Windows the executable builds and vets, and the theme commands that need no screen work, but there is no
HID transport and no sensor source: the commands that open a device, and `run`, fail with `windows is not supported
yet`. Windows support comes after this unit.

What the repository holds today, which this unit builds on:

- The README still opens with a status note that ends by saying it describes what sensordeck is meant to become, a
  tagline that names Linux and Windows, a section "What it will do" in the future tense, and a section "Planned usage".
  The sections after it (access on Linux, sensor values, the Commander Core XT, the service, writing a theme, reading
  a theme file) describe what the code does now
- The credits thank the turing-smart-screen-python-HIDdev2 project for discovering the protocol and name
  `golang.org/x/image` and the Go fonts, by the Go authors under a BSD licence. They name no other dependency
- `go.mod` requires `github.com/spf13/cobra`, `go.yaml.in/yaml/v3`, `golang.org/x/image` and `golang.org/x/sys`, and
  indirectly `github.com/spf13/pflag`, `github.com/inconshreveable/mousetrap` and `golang.org/x/text`. Its module path
  is `github.com/norareit/sensordeck` and it names Go 1.27
- The executable has no version: there is no `--version`, and nothing is set at link time
- `sensordeck theme preview <name-or-folder> <out.png>` compiles a theme in memory and writes the preview of the
  compiled file, with every number drawn as 42 and the date and time as 2026-11-15 09:41:07, so its output depends only
  on the theme and the code
- There is no GoReleaser configuration and no GitHub Actions workflow. `dist/` is ignored by git, and on the
  developer's machine it holds the captures of the checks made on the device in earlier units

The human publishes. This unit prepares the repository and pushes, tags and releases nothing.

"The suite" below is the five commands that `AGENTS.md` lists under Testing. "An archive" is one of the archives the
GoReleaser configuration of this unit builds. "The preview image" is the PNG of R3.

## Requirements

### The README

- **R1** The README describes what sensordeck does now. It has no status note, no sentence that says it describes what
  sensordeck is meant to become, no section "What it will do" and no section "Planned usage", and nothing in it states
  as working what the code does not do
- **R2** The README opens, before any other section, with: what sensordeck does today on Linux, in the present tense;
  the sentence that Windows is not supported yet, with what the Windows executable does today (the theme commands that
  need no screen) and that the screen, the sensors and the service are still to come there; and the preview image,
  with an alt text, and a caption or sentence that says it is the preview that `sensordeck theme preview` draws with
  sample values and not a photo of the screen
- **R3** The preview image is `docs/minimal-dark.png`, a PNG written by
  `sensordeck theme preview minimal-dark docs/minimal-dark.png`. A test decodes it and fails when its size or any of
  its pixels differs from the preview that the code draws now from the built-in theme `minimal-dark`, and its message
  names that command as the way to write the image again
- **R4** The tagline under the title names Linux as the system it works on now, and does not claim Windows
- **R5** The README has a section on installing, before the quick start, with three routes in this order:
  1. A release archive from the GitHub Releases page of `github.com/norareit/sensordeck`: which archive to take for
     each platform, how to check it against the checksum file with `sha256sum`, how to extract it and put the
     executable on the `PATH`, and that the udev rule is in the archive, with a link to the section on access on Linux
  2. `go install github.com/norareit/sensordeck@latest`, with the Go version it needs, which is the one `go.mod`
     names, and that the udev rule then comes from the repository
  3. Building from source: cloning the repository and building with `go build`, with CGO disabled as the releases are
- **R6** The quick start comes after the install section and keeps its three steps, its text and its links, with its
  first sentence still saying it is for Linux and why
- **R7** The section "Planned usage" becomes a section on the commands, with every command of the executable in one
  code block with a short comment each: `devices`, `clock`, `values`, `upload`, `listen`, `raw`, `sensors`, `run`,
  `theme list`, `theme use`, `theme preview`, `theme compile`, `theme dump`, `service install`, `service status` and
  `service uninstall`, plus `--version` and `--help`. The six lines of the old block stay, with the comment of
  `service install` corrected to what it does on Linux, which is to run `run` as a systemd user service that starts
  at login
- **R8** Each of these appears in the README as written here: `0483:0065`, `llhmi`, `llTechCo`, `SmartMonitor`,
  `3.5 inch`, `480x320` (with the letter x, beside the existing `480×320`), `Windows only` in quotes as the screen is
  sold, `Corsair iCUE Commander Core XT` and `1b1c:0c2a`. The section on supported devices names the Commander Core XT
  with its USB ID as the one temperature controller whose probes sensordeck reads
- **R9** The sections "Linux: access to the screen", "Showing sensor values on the screen", "The Corsair iCUE Commander
  Core XT", "Running as a service", "Writing a theme" and "Reading a theme file" are kept with their text, their
  commands, their code blocks and their links unchanged, except where R13 names a change. Their headings stay, so that
  the anchors other sections link to keep working
- **R10** The credits keep the thanks to the turing-smart-screen-python-HIDdev2 project, with its link, for
  discovering the protocol, and keep the sentence on what the theme compiler draws glyphs with. They name each module
  that `go.mod` requires, directly or indirectly, with its licence as the licence file of the module states it:
  `github.com/spf13/cobra`, `github.com/spf13/pflag`, `github.com/inconshreveable/mousetrap`, `go.yaml.in/yaml/v3`,
  `golang.org/x/sys`, `golang.org/x/image` with the Go fonts built into sensordeck, and `golang.org/x/text`
- **R11** The license section says the project is under the MIT licence and links to `LICENSE`
- **R12** The README has a short section for the maintainer on making a release, as the last section before the
  credits: that CI runs the suite on pushes and pull requests, that pushing a tag of the form `v<major>.<minor>.<patch>`
  makes the release workflow run the suite and, only when it succeeds, build the archives and the checksum file into a
  draft release, that the draft is checked and published by hand on GitHub, how to try the configuration locally with
  `goreleaser release --snapshot --clean` and where that writes, and the command that writes the notices file of R27
  again after a dependency changed
- **R13** Every change to the existing content of the README is one of these, and the record of the implementation
  accounts for each:
  1. The title's tagline: corrected by R4
  2. The status note: replaced by the opening of R2. Its list of what works moves into the opening and into the
     commands of R7; its sentence that Windows is still to come becomes the Windows sentence of R2; its last
     sentence, that the README describes what sensordeck is meant to become, is removed, because it is no longer true
  3. "What it will do": merged into the opening of R2, in the present tense. Its introduction on screens sold as
     "Windows only" is kept. Its bullet on live readings is corrected to the readings the metrics table lists, because
     motherboard sensors other than the probes of the Commander Core XT are not read; its bullet on the service says
     systemd on Linux and the startup task on Windows as still to come. Its bullets on themes as folders, the preview
     and the single executable with built-in themes are kept
  4. "Supported devices": the sentence that version 1 targets a single device is corrected to say that sensordeck
     drives this one screen, the table is kept, the identifiers of R8 are added, the Commander Core XT is added as R8
     says, and the sentence that other screens can be added later is kept
  5. "Planned usage": becomes the commands section of R7
  6. "Quick start": moved after the new install section, unchanged otherwise
  7. "Credits": extended by R10
  8. "License": extended by R11
  9. New sections: the install section of R5 and the release section of R12

  The links of the README to `docs/theme-folder.md` and `docs/theme-format.md` stay where the README has them now, and
  every link to an anchor of the README resolves to a heading of it

### Distribution

- **R14** A GoReleaser configuration, of version 2 of its format, builds the executable from the module root with
  `CGO_ENABLED=0` and `-trimpath` for exactly linux/amd64, linux/arm64 and windows/amd64. Its ldflags are
  `-s -w -X main.version=v{{.Version}}`
- **R15** Each archive holds the executable, `README.md`, `LICENSE`, `THIRD_PARTY_LICENSES` (the notices file of
  R27) and `udev/70-sensordeck.rules`. The Linux archives
  are `.tar.gz` and the Windows archive is `.zip`. Their names are GoReleaser's default, as
  `sensordeck_<version>_<os>_<arch>`, and the README's install section names them in that form
- **R16** The release has a SHA-256 checksum file of the archives, under GoReleaser's default name
  `sensordeck_<version>_checksums.txt`
- **R17** The release is a draft, and GoReleaser writes no changelog into it
- **R18** GoReleaser writes into a directory other than `dist/`, and git ignores that directory
- **R19** A GitHub Actions workflow runs when a tag that starts with `v` is pushed, and has two jobs. The first calls
  the CI workflow of R20 as a reusable workflow, with the permission `contents: read` and no other. The second needs
  the first, so that it does not run when the suite fails, and runs GoReleaser with `release --clean` after checking
  out the full history and setting up the Go version that `go.mod` names, with the permission `contents: write` and
  no other. The workflow grants no permission at its top level. The version of GoReleaser it runs is an exact
  release, not `latest` or a range
- **R20** A second GitHub Actions workflow runs the five commands of the suite on every push and every pull request,
  and when another workflow calls it, on an Ubuntu runner named by its version, with the permission `contents: read`
  and no other, and with the Go version that `go.mod` names. Its step for `gofmt -l .` fails when the command prints
  a file name
- **R21** Each action that either workflow uses is pinned to an exact release tag, such as `v4.2.2`, and not to a
  branch or a major version tag

### The version

- **R22** `sensordeck --version` prints `sensordeck version <version>` and a newline to standard output and ends with
  exit status 0, where `<version>` is the value that the ldflags of R14 set in the variable `version` of package
  `main`; without one, the version of the main module that the build information of the executable gives; and
  `(devel)` when that is empty or there is no build information. The variable reaches the command line as a parameter
  and is not read from the command line's package
- **R23** No existing command or flag changes because of `--version`

### Tests

- **R24** The tests check, by parsing the files as YAML with the YAML module the project already uses:
  - of the GoReleaser configuration: version 2, the three platforms of R14 and no other, `CGO_ENABLED=0`, the ldflags
    of R14, the files of R15 in the archives, the formats of R15, the draft and the disabled changelog of R17, and an
    output directory other than `dist`
  - of the release workflow: the trigger on pushed tags that start with `v` and no other trigger, no permission at the
    top level, a job that calls the CI workflow by its path with `contents: read`, a job that needs that job, has
    `contents: write` and has a step that runs GoReleaser with `release --clean` at an exact version, and that every
    `uses` of a step names an exact release tag
  - of the CI workflow: the triggers `push`, `pull_request` and `workflow_call`, the permission of R20, that its steps
    run each of the five commands of the suite, each holding the command as `AGENTS.md` writes it, and that every
    `uses` names an exact release tag
- **R25** The unit tests cover R22: the printed line with a version set, with none set and a build information
  version, and with neither. An end-to-end test runs the built executable with `--version` and checks the form of
  the line and the exit status
- **R26** The suite succeeds

### Third-party notices

- **R27** The repository holds a committed plain-text file `THIRD_PARTY_LICENSES` with the licence notices of the code
  linked into the executable. It has one part for the Go standard library, with the text of the `LICENSE` file of the
  Go toolchain, and one part for each module that `go.mod` requires, directly or indirectly, in the order of the
  module paths. Each module's part opens with a line naming its path and the version `go.mod` requires, followed by
  the text, unchanged, of every file at the root of the module whose name starts with `LICENSE`, `LICENCE`, `COPYING`
  or `NOTICE`, in the order of the file names, each under a line with the file's name. It holds no Go toolchain
  version, so that it does not change when the toolchain does
- **R28** A program in the repository writes the file of R27, run with one `go run` command that R12 names. It finds
  the directory of each module with `go list -m` and the toolchain's `LICENSE` under `go env GOROOT`, so it needs
  nothing but the module cache and the module proxy. It fails, naming the module, when a module has no file of those
  names or is not in the module cache, and writes nothing then
- **R29** A test produces the content of R27 with the same code as the program of R28 and fails when it differs from
  the committed file, with a message that names the command of R28. The text of the file is made by a pure function
  of the module list and the licence texts, which is unit tested with plain values for the order, the headers, the
  file names it takes and the failure on a module with no licence file

## Decisions

- **D1 A feature.** The work makes decisions on the shape of the release, the version and the README, so it is not a
  task.
- **D2 The README describes the present and keeps its reference content.** The project's standards treat existing
  documentation as intentional, so the reference sections are kept as they are and only what is no longer true is
  corrected. R13 lists every change so the reviewer can account for each. Rewriting the reference sections for tone
  was rejected: they were written and checked unit by unit against the code, and a rewrite would lose that.
- **D3 The image is the preview, checked by pixels.** A photo of the screen cannot be made in the sandbox and would
  go stale with every theme change. The preview is drawn by the code itself, so a test can say whether the committed
  image is still what the code draws. Comparing the decoded pixels was chosen over comparing the bytes of the file,
  because the PNG encoder of a later Go release may compress the same pixels into other bytes, and the test would then
  fail with nothing wrong. The README says it is a preview, because the preview draws numbers left-aligned and the
  device's own drawing has not been compared with it.
- **D4 The version is in this unit.** A published executable that cannot say its version makes every bug report
  harder to answer, and the GoReleaser configuration has to set it at the time it is written. The variable is
  `main.version`, GoReleaser's convention, passed to the command line by `main`, so that the command line does not
  depend on a link-time global. The value is `v{{.Version}}` rather than GoReleaser's `{{.Version}}`, which drops the
  `v`, so that a release and `go install @v0.1.0` print the same `v0.1.0`; `{{.Tag}}` was rejected because a snapshot
  build would print the last tag as if it were that release. The fallback to the build information covers
  `go install` and `go build`, for which Go records the module version from the module proxy or from the VCS. Cobra's
  `Version` field gives `--version`, and its default line `sensordeck version <version>` is kept. A `version`
  subcommand was rejected as a second way to say the same thing.
- **D5 A draft release with no changelog.** The workflow cannot be run before the first tag, so the first time it
  runs is on the human's release. A draft lets the human look at the archives and write the notes before anyone sees
  the release, and publishing it is the human's step. GoReleaser's changelog lists commits, which here are hundreds of
  unit commits that say nothing to a user, so it is disabled and the notes are written by hand.
- **D6 GoReleaser's output is not `dist/`.** `--clean` deletes the output directory, and on the developer's machine
  `dist/` holds the captures of the checks on the device. A local snapshot run would delete them. The new directory is
  ignored by git.
- **D7 Exact versions, not commit hashes.** R21 pins the actions to exact release tags, and R19 GoReleaser to an
  exact release. A major tag such as `v4` moves with each release of the action and would change the build without a
  change to the repository. Pinning to commit hashes is stronger, because a tag can be moved, but it needs the hash of
  each tag to be looked up and checked, which the sandbox cannot do reliably; it is item 1 of "Beyond the request".
- **D8 The threat model.** The release workflow holds a token that can write to the repository's releases. The
  adversary considered is a third-party action or tool that changes under the same name, which R19 and R21 limit by
  pinning and by granting only the permission each workflow needs. Not defended against: a compromise of GitHub, of
  the Go toolchain or of the module proxy, a moved release tag of an action (item 1), and anyone with push access to
  the repository, who can change the workflows. The supported usage is the human pushing a version tag to the
  repository on GitHub.
- **D9 CI on every push and pull request.** The end-to-end tests need Linux with `/dev/ptmx`, which GitHub's Ubuntu
  runners have, so the whole suite runs there, including the Windows build and vet, which need no Windows runner. The
  runner is named by its version, as `ubuntu-24.04`, so that a change of `ubuntu-latest` does not change the build
  without a change to the repository. A push to a branch with an open pull request runs twice, which is accepted for
  a project of this size.
- **D10 What the tests check of the workflows.** Neither workflow can run in the sandbox, and GoReleaser is not
  installed there. The tests check what can be read from the files: that they parse and name the platforms, files,
  triggers, permissions, versions and commands this spec requires. Whether GitHub accepts the workflows, whether
  GoReleaser accepts the configuration, and whether the archives come out as specified are seen on the human's first
  CI run and first release, and this unit does not wait for them or depend on them.
- **D11 A Windows archive although Windows is not supported.** The plan names windows/amd64, the theme commands work
  there now, and building it in each release keeps the build from breaking unseen before Windows support arrives. The
  README says plainly what it does and does not do.
- **D12 Credits name every module of `go.mod`.** The request names the direct dependencies. The indirect ones,
  `pflag`, `mousetrap` and `x/text`, are named too, because `pflag` is linked into every executable and `mousetrap`
  into the Windows one, and a credits list that leaves out linked code is wrong. Each licence is taken from the
  module's own licence file, not from memory.
- **D13 The notices file is generated, committed and checked.** The BSD licences of `pflag`, the `golang.org/x`
  modules and the Go standard library ask that a binary distribution reproduce their notice, and the Apache licence
  of `cobra`, `mousetrap` and the YAML module asks that a `NOTICE` file shipped with the work be passed on. So each
  archive carries the texts, not just the names in the credits. Copying licence texts is not copying code, so the
  publishing rules allow it. The file is generated from the module cache rather than written by hand, because a hand
  copy goes wrong silently; it is committed, rather than generated by GoReleaser at release time, so that its content
  is reviewed in a diff and the release needs no step that the tests have not run; and a test compares it with what
  the generator gives now, so that a new or updated dependency fails the suite until the file is written again.
  Tools that collect licences, such as `go-licenses`, were rejected as a new dependency for a few lines of code.
  Taking the files at the module root only is enough for the modules of `go.mod` today, whose licence files are all at
  the root; the Go fonts are under the licence of `golang.org/x/image`.
- **D14 The release waits for the suite.** A release must not be built from a commit whose suite fails. Calling the CI
  workflow as a reusable workflow, rather than repeating its steps in the release workflow, keeps the five commands in
  one place. The permissions are granted per job, so that the job that runs the suite never holds the write token.

## Non-goals

- Windows support: the HID transport, the sensor source and the startup task
- Packaging beyond the archives: no `.deb`, winget, scoop or Homebrew, which GoReleaser could also make
- macOS builds, which the plan names as optional
- A website or GitHub Pages
- Pushing, tagging, creating the repository's releases or changing its settings on GitHub: the human publishes
- Running the workflows or GoReleaser: the first CI run and the first release are the human's
- Signing the archives or the checksum file, and software bills of materials
- Changing the reference sections of the README beyond what R13 lists, or the documents under `docs/`
- Changing what any existing command does

## Beyond the request

1. Pinning each action to the full commit hash of its release tag, with the tag in a comment. Cost: looking up and
   checking a hash per action, on the host or with network access, and updating hashes rather than tags later.
   Without it, a moved release tag of an action changes the build unseen, which needs a compromise of that action's
   repository. Declined.
2. A file of the licence texts of the modules of `go.mod`, put into each archive. Cost: one committed file of copied
   licence texts, which is allowed because they are licence texts and not code, and a test that it names each module
   of `go.mod`. Without it, the archives carry the names and licences in the README's credits but not the licence
   texts, which the BSD licences of `pflag` and the `golang.org/x` modules ask a binary distribution to reproduce; it
   is fixed in a later release when noticed. Taken: R27 to R29, D13.
3. A test that the credits name each module of `go.mod`, so that a new dependency cannot be added without its credit.
   Cost: one test that reads `go.mod` and the README. Without it, a new dependency goes uncredited until someone
   notices and adds a line. Declined.
4. A test that the README holds each identifier of R8. Cost: a few lines. Without it, a later edit can drop one
   unseen, and searches stop finding the project until it is added back. Declined.
5. The release workflow running the suite before GoReleaser, by calling the CI workflow as a reusable workflow and
   making the release wait for it. Cost: a `workflow_call` trigger on the CI workflow and a second job. Without it, a
   tag on a commit that fails the suite still makes a draft release, which the human sees in CI and deletes.
   Taken: R19, R20, D14.
6. The CI workflow also running `goreleaser check` on the configuration, so that a configuration GoReleaser refuses is
   seen on a push rather than at the first tag. Cost: one step with the pinned GoReleaser action. Without it, such a
   configuration shows at the first release, which fails before it uploads anything and is fixed and tagged
   again. Declined.
7. The CI workflow also running `npm run check-changes`. Cost: Node setup and `npm ci` in CI. Without it, the change
   check runs only where the process runs it today, before work is submitted. Declined.
8. Putting `docs/` into the archives, so that the README's links to the documents and the preview image work
   offline. Cost: one entry in the archive files and a line in R24. Without it, those links in an extracted README
   point to files that are not there, and the documents are one click away on GitHub. Declined.
