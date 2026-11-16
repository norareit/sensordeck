# A second built-in theme for the Commander Core XT probes: plan

## Approach

The task is a record with no numbered requirements, so this plan numbers what the record says must be true, and each
task below names those numbers:

* R1: the built-in themes are `minimal-dark` and `minimal-dark-probes`, the second a new folder beside the first in the
  embedded tree, and `minimal-dark` is unchanged
* R2: the folder of `minimal-dark-probes` holds `theme.yaml` and `background.png`, both regular files, and nothing else
* R3: its `theme.yaml` is the text of the record, byte for byte, with a line feed after its last line
* R4: its `background.png` has the pixels of the generator, which `go generate ./internal/builtinthemes/` writes into
  both folders; the comments of the background package and of the generator say they draw the background the two
  built-in themes share
* R5: it compiles with no error and no warning when read back, and gives the tags and the tags file of the record
* R6: `theme list` with no themes directory writes the two themes as `built-in` in byte order, and an installed
  `minimal-dark` replaces only `minimal-dark`
* R7: `docs/minimal-dark-probes.png` is committed and is the preview of the theme
* R8: the tests of `internal/builtinthemes` pin R1 to R5, and the background and compilation tests go through one table
  of the two themes
* R9: the test of the preview images in `docs_test.go` covers both images through one table, and its failure message
  gives the command that writes the image of that theme again
* R10: the unit test of `theme list` that runs the real built-in themes and the end-to-end test of `theme list` expect
  both themes in every case, the override case included
* R11: the end-to-end test compiles `minimal-dark-probes` by its name and checks its tags file
* R12: the README has the section "Built-in themes", after "Quick start" and before "Commands", with the content the
  record gives
* R13: the opening paragraph of the README names both themes, the image of `minimal-dark` at the top stays, and the
  quick start is unchanged
* R14: `docs/theme-folder.md` lists both built-in themes
* R15: the five commands of the suite in `AGENTS.md` succeed

The lookup and `theme list` read the embedded tree, so no production code beyond the generator's `go:generate` line and
two comments changes. The generator keeps its one argument: a second `//go:generate` line writes the second folder's
PNG, so that one `go generate ./internal/builtinthemes/` writes both. The function `background.MinimalDark` keeps its
name, since the record asks only for the comments to change.

The record's theme was compiled on the unit's branch before this plan was written, from a copy of the folder outside the
tree, and it gave exactly the tags file of the record.

## Files

* `internal/builtinthemes/builtinthemes_test.go`: the table of the built-in themes, and the tests through it
* `internal/builtinthemes/builtinthemes.go`: the second `//go:generate` line
* `internal/builtinthemes/themes/minimal-dark-probes/theme.yaml` and `background.png`: new
* `internal/builtinthemes/background/background.go` and `internal/builtinthemes/background/generate/main.go`: the
  package and function comments
* `docs_test.go`: the table of the preview images
* `docs/minimal-dark-probes.png`: new
* `internal/cli/themes_test.go`: `TestThemeListShowsWhatTheLookupFinds`
* `e2e/builtin_theme_test.go`: the `theme list` test and the compile test
* `README.md`, `docs/theme-folder.md`

## Tasks

- [x] In `internal/builtinthemes/builtinthemes_test.go`, add a constant `minimalDarkProbesYAML` holding the
  `theme.yaml` of the record verbatim, and a constant `minimalDarkProbesTags` holding its tags file verbatim. Add one
  table of the built-in themes, each row with the name, the expected `theme.yaml`, the metric names in the order of
  their tags, and the expected tags file; the rows are `minimal-dark` with the existing constants and names, and
  `minimal-dark-probes` with the new ones and the names `commander.1.temp`, `commander.2.temp`, `cpu.temp`, `cpu.util`,
  `gpu.mem.used`, `gpu.temp`, `gpu.util`, `net.down`, `net.up`, `nvme.2.temp`, `ram.used`, `ram.util`. Replace the
  helper `minimalDark` by one that takes the name and fails the test when `Folder` does not find it (R8)
- [x] Rewrite the tests of that file to go through the table, each row a subtest named by the theme: `Names()` is
  exactly the names of the table in their order, which is `minimal-dark`, `minimal-dark-probes`; each folder holds
  exactly `background.png` and `theme.yaml`, both regular; each `theme.yaml` equals the row's text; each
  `background.png` is 480 x 320 and equals `background.MinimalDark()` at every pixel, with the failure message naming
  the theme, the pixel and `run go generate ./internal/builtinthemes/` as today; each theme compiles, reads back with
  `themefile.Parse` with no error and no warning, and gives the row's tags and tags file. Rename the test of the names
  from `TestTheBuiltInThemesAreMinimalDarkAlone` to one that says the built-in themes are the rows of the table. Keep
  `TestFolderIsFoundForADirectoryOfTheThemesAlone` as it is. Run `go test ./internal/builtinthemes/` and see the tests
  of `minimal-dark-probes` fail because the theme is missing (R1, R2, R3, R4, R5, R8)
- [x] Add a second line `//go:generate go run ./background/generate themes/minimal-dark-probes/background.png` below
  the existing one in `internal/builtinthemes/builtinthemes.go`. Create the folder
  `internal/builtinthemes/themes/minimal-dark-probes/`, write its `theme.yaml` as the text of the record, and run
  `go generate ./internal/builtinthemes/`, which writes both PNGs. Check with `git status` that the PNG of
  `minimal-dark` is unchanged or, if its bytes differ, that `go test ./internal/builtinthemes/` still passes and restore
  it with `git checkout` so that `minimal-dark` stays byte for byte as it was (R1, R2, R3, R4)
- [x] Change the package comment of `internal/builtinthemes/background/background.go` to say that it draws the
  background the two built-in themes share, and the comment of `MinimalDark` to say that it draws that background, by
  the rule of R4 of unit 0012. Change the command comment of `internal/builtinthemes/background/generate/main.go` to say
  that it writes the background the two built-in themes share as a PNG to the path of its one argument. Run
  `go test ./internal/builtinthemes/...`, which passes (R4, R8)
- [x] Commit point: `task-0023 Add the built-in theme minimal-dark-probes`
- [ ] In `docs_test.go`, replace the constants `previewImagePath` and `previewCommand` by a table of the preview images
  with the rows `minimal-dark` with `docs/minimal-dark.png` and `minimal-dark-probes` with
  `docs/minimal-dark-probes.png`. The test runs one subtest per row, compiles the built-in theme of the row, and
  compares the image of the row with `themeimage.Preview` of the read-back theme by decoded pixels; `compareWithImage`
  takes the path as a parameter. The failure message is `<path> <mismatch>: write the image again with
  "sensordeck theme preview <name> <path>"`, with the command built from the row. Run `go test .` and see the row of
  `minimal-dark-probes` fail because the image is missing (R7, R9)
- [ ] Write the image with `go run . theme preview minimal-dark-probes docs/minimal-dark-probes.png` and run
  `go test .`, which passes. Look at the image and check that the second row shows `VRAM` with `DISK` and a value under it, and the
  tile `GPU CABLE` with `INTAKE` under it (R7)
- [ ] Commit point: `task-0023 Add the preview image of minimal-dark-probes`
- [ ] In `internal/cli/themes_test.go`, change `TestThemeListShowsWhatTheLookupFinds` so that with no themes directory
  it expects `minimal-dark\tbuilt-in\nminimal-dark-probes\tbuilt-in\n`, and after the installs it expects the lines of
  `linked`, of `minimal-dark` as `installed, overrides built-in`, of `minimal-dark-probes\tbuilt-in` and of `night`, in
  that order; the existing removal of the first line when the symbolic link cannot be made still applies. Leave
  `TestThemeListLinesAreSortedByName`, which takes its built-in names as a value, as it is. Run
  `go test ./internal/cli/`, which passes (R6, R10)
- [ ] In `e2e/builtin_theme_test.go`, change `TestThemeListShowsTheBuiltInAndTheInstalledThemes` so that each of its
  three cases expects `minimal-dark-probes\tbuilt-in` right after the line of `minimal-dark`, the override case
  included. Add the constant `minimalDarkProbesTagsFile` with the tags file of the record, and make
  `TestThemeCompileAndPreviewTheBuiltInTheme` run, for each row of a table of the two themes and their tags files, the
  `theme compile` by name, the check of its output, the read-back with no warning, the check of the tags file, and the
  `theme preview` by name with the check of its bytes, each row a subtest. Run `go test ./e2e/`, which passes (R5, R6,
  R10, R11)
- [ ] Commit point: `task-0023 Expect both built-in themes in the tests of theme list and compile`
- [ ] In `README.md`, change the last sentence of the opening paragraph so that it names the built-in themes
  `minimal-dark` and `minimal-dark-probes`, and leave the image of `minimal-dark` at the top, its alt text and the
  sentence after it as they are. Leave "Quick start" unchanged (R13)
- [ ] In `README.md`, add the section `## Built-in themes` between "Quick start" and "Commands". For `minimal-dark`: a
  sentence that it shows the tiles the [quick start](#quick-start) lists, by that link and no second list, and the
  image `docs/minimal-dark.png` with an alt text that says what the tiles show. For `minimal-dark-probes`: a sentence
  that it is `minimal-dark` with the VRAM total replaced by the temperature of the second NVMe drive and the disk tile
  replaced by the two probes of a Corsair iCUE Commander Core XT, and the image `docs/minimal-dark-probes.png` with an
  alt text that says what its tiles show. A sentence says that the two images are previews that
  `sensordeck theme preview` draws with sample values, and not photos of the screen (R12)
- [ ] In the same section, say which hardware `minimal-dark-probes` expects: a Commander Core XT with two temperature
  probes, `commander.1.temp` and `commander.2.temp`, labelled `GPU CABLE` and `INTAKE` because on the machine the theme
  was made for probe 1 is at the GPU power cable and probe 2 at the front intake; and a second NVMe drive, whose
  `nvme.2.temp` is the second drive in the order of the places of the hardware, its PCI path, as the table of metrics
  says. Link to [The Corsair iCUE Commander Core XT](#the-corsair-icue-commander-core-xt) for what reading the
  controller does to its LEDs, and say that a tile whose metric the host cannot read behaves as the quick start says.
  Say that the theme is an example to copy and adapt: copy its folder,
  `internal/builtinthemes/themes/minimal-dark-probes` of the repository, into the themes directory under a name of
  one's own, and change the labels and the metrics to those of one's own hardware. Do not say that it has been seen on
  the device (R12)
- [ ] In `docs/theme-folder.md`, change the sentence that says the built-in themes are `minimal-dark` so that it
  names `minimal-dark` and `minimal-dark-probes` (R14)
- [ ] Commit point: `task-0023 Document the built-in theme minimal-dark-probes`

## Verification

- [ ] `go test ./...` passes (R15)
- [ ] `go vet ./...` reports nothing (R15)
- [ ] `gofmt -l .` prints no file name (R15)
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing (R15)
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing (R15)
- [ ] `git diff master -- internal/builtinthemes/themes/minimal-dark` prints nothing (R1)
- [ ] `go run . theme list` with `XDG_CONFIG_HOME` set to an empty temporary directory writes
  `minimal-dark\tbuilt-in` and `minimal-dark-probes\tbuilt-in` (R6)
- [ ] `npm run check-changes` passes after the last commit

## Commit Points

- [ ] `task-0023 Add the built-in theme minimal-dark-probes`
- [ ] `task-0023 Add the preview image of minimal-dark-probes`
- [ ] `task-0023 Expect both built-in themes in the tests of theme list and compile`
- [ ] `task-0023 Document the built-in theme minimal-dark-probes`
