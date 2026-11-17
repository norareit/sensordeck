# Alarm colours in the built-in theme minimal-dark-probes

The record of the task states what is true when it is done. "The record" below is
`changes/0030-probes-theme-alarm-colours/record.md`, and its bullets are traced as these requirements:

- **R1** The five lines of `theme.yaml` of `minimal-dark-probes` carry `alarm: {above: <n>, color: "#ff3030"}` as their
  last key, at the thresholds of the record, and every other line is as it was
- **R2** The layout, the background, the fonts and the folder are unchanged, and no key `align` is added
- **R3** The theme compiles with no error and no warning when read back, with 43 widgets, and gives the tags file of
  the record byte for byte
- **R4** `docs/minimal-dark-probes.png` is the preview drawn after the change
- **R5** `minimal-dark` is unchanged: its `theme.yaml`, its `background.png`, its tags and `docs/minimal-dark.png`
- **R6** The tests pin the text of each theme, its plain tags and alarm pairs and its tags file through the one table
  of the built-in themes, the preview images through `docs_test.go`, and the tags file of `minimal-dark-probes` in
  `e2e/builtin_theme_test.go`
- **R7** The section "Built-in themes" of `README.md` gives the alarms and their thresholds, links to the part of the
  README on the key `alarm` and says that a copy changes the thresholds in its `theme.yaml`. Nothing else of the README
  or of `docs/` changes
- **R8** The five commands of the suite in `AGENTS.md` succeed

## Approach

The change is data: five lines of one theme file, a preview image and a paragraph of the README. No code of the
compiler, the preview or `run` changes. The tests are changed first, so that they fail against the theme of today and
pass once the five lines are in.

The table of the built-in themes in `internal/builtinthemes/builtinthemes_test.go` lists the plain tags of a theme as
`metricNames` numbered from 1, which cannot state an alarm pair. Its column becomes `tags []config.Mapping`, the plain
entries with their tags as `themecompile.Result.Tags` gives them, beside a new column `alarms []config.AlarmPair`, as
`themecompile.Result.Alarms` gives them, and the test compares both with `reflect.DeepEqual`. The row of
`minimal-dark` states the same tags as today and no alarm pair, a nil slice, which is what `assignTags` in
`internal/themecompile/tags.go` returns for a theme with no alarm number.

The preview is written with the executable of the branch, with `XDG_CONFIG_HOME` set to an empty temporary
directory so that no installed theme of the same name on the machine replaces the built-in one.

## Files

- `internal/builtinthemes/builtinthemes_test.go`: the constants `minimalDarkProbesYAML` and `minimalDarkProbesTags`
  take the text and the tags file of the record; the table states plain tags and alarm pairs for each row; the test of
  the compilation compares both
- `e2e/builtin_theme_test.go`: the constant `minimalDarkProbesTagsFile` takes the tags file of the record
- `internal/builtinthemes/themes/minimal-dark-probes/theme.yaml`: the five lines of the record
- `docs/minimal-dark-probes.png`: written again by `sensordeck theme preview`
- `README.md`: the section "Built-in themes"

No other file changes. `internal/builtinthemes/themes/minimal-dark/`, `docs/minimal-dark.png`, `docs_test.go`,
`docs/theme-folder.md` and `internal/builtinthemes/themes/minimal-dark-probes/background.png` stay as they are.

## Tasks

- [ ] In `internal/builtinthemes/builtinthemes_test.go`, replace the column `metricNames []string` of `builtInThemes`
  with `tags []config.Mapping` and add the column `alarms []config.AlarmPair`. The row of `minimal-dark` states the
  eleven entries of today, `cpu.temp` 1 to `ram.util` 11, as `config.Mapping` values, and no alarm pair. In
  `TestEachThemeCompilesWithItsTags`, compare `result.Tags` with `theme.tags` and `result.Alarms` with `theme.alarms`,
  each with `reflect.DeepEqual` and a failure message that prints the value got, and drop the loop that numbered the
  names from 1. Leave the other tests of the file as they are (R5, R6)
- [ ] In the same file, change the row of `minimal-dark-probes` to what the record states. Its `tags` are the seven
  plain entries of the tags file of the record: `cpu.util` 7, `gpu.mem.used` 8, `gpu.util` 11, `net.down` 12,
  `net.up` 13, `ram.used` 16, `ram.util` 17. Its `alarms` are the five pairs, in ascending order of the normal tag,
  each with `config.Threshold{Above: <n>, HasAbove: true}`: `commander.1.temp` above 60 normal 1 alarm 2,
  `commander.2.temp` above 35 normal 3 alarm 4, `cpu.temp` above 90 normal 5 alarm 6, `gpu.temp` above 85 normal 9
  alarm 10, `nvme.2.temp` above 70 normal 14 alarm 15 (R3, R6)
- [ ] In the same file, change `minimalDarkProbesYAML` to the text of today with the five widget lines of the record
  put in verbatim, and change `minimalDarkProbesTags` to the tags file of the record, byte for byte. No other line of
  either constant changes, and `minimalDarkYAML` and `minimalDarkTags` do not change (R1, R2, R3, R5, R6)
- [ ] In `e2e/builtin_theme_test.go`, change `minimalDarkProbesTagsFile` to the tags file of the record, byte for
  byte, and leave `minimalDarkTagsFile` as it is (R3, R6)
- [ ] Run `go test ./internal/builtinthemes/ ./e2e/` and see the tests of `minimal-dark-probes` fail on its
  `theme.yaml`, its tags and its tags file, and those of `minimal-dark` pass (R6)
- [ ] In `internal/builtinthemes/themes/minimal-dark-probes/theme.yaml`, append
  `, alarm: {above: <n>, color: "#ff3030"}` before the closing brace of the five number widgets of `cpu.temp` (90),
  `gpu.temp` (85), `nvme.2.temp` (70), `commander.1.temp` (60) and `commander.2.temp` (35), so that the five lines
  read as the record gives them. Change no other byte of the file, and check with `git diff --stat` and `git diff`
  that the file has five lines changed and nothing else (R1, R2)
- [ ] Run `go test ./internal/builtinthemes/ ./e2e/` and see every test pass except, in `go test ./...`, the test of
  the preview images in `docs_test.go` for `minimal-dark-probes` (R3, R6)
- [ ] Check the count of widgets once: compile the folder with the executable of the branch
  (`go run . theme compile minimal-dark-probes <temporary>/img.dat` with `XDG_CONFIG_HOME` set to an empty temporary
  directory) and read the file back with `go run . theme dump <temporary>/img.dat`, or with `themefile.Parse` in a
  scratch test that is not committed, and confirm that it holds 43 widgets and no warning. Log the count in the
  record (R3)
- [ ] Write the preview again with
  `XDG_CONFIG_HOME=$(mktemp -d) go run . theme preview minimal-dark-probes docs/minimal-dark-probes.png`, and check
  that `git status` shows `docs/minimal-dark-probes.png` changed and `docs/minimal-dark.png` not. Look at the image:
  the five numbers show 42 in the colour of their font, CPU, GPU and GPU CABLE about half a digit further left than
  before and INTAKE next to its label, and nothing else moved (R4, R5)
- [ ] In the section "Built-in themes" of `README.md`, add to the bullet of `minimal-dark-probes`, after its first
  sentence, that the theme shows its five temperatures in red past their thresholds, in °C: CPU above 90, GPU above 85,
  DISK above 70, GPU CABLE above 60 and INTAKE above 35, with a link to [Writing a theme](#writing-a-theme), the part of
  the README that describes the key `alarm`. Add to the paragraph that begins "`minimal-dark-probes` is an example to
  copy and adapt" that a copy changes the thresholds in the key `alarm` of its `theme.yaml`. Leave the alt texts of the
  two images, the rest of the section and the rest of the README as they are (R7)
- [ ] Run the five commands of the suite in `AGENTS.md` and see each succeed: `go test ./...`, `go vet ./...`,
  `gofmt -l .`, `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` (R8)

## Verification

- [ ] `git diff master --stat` lists only the plan, the record and the five files under "Files"
- [ ] `git diff master -- internal/builtinthemes/themes/` shows the five lines of the record changed and nothing else
- [ ] The five commands of the suite in `AGENTS.md` succeed
- [ ] `npm run check-changes` passes on the committed branch

## Commit Points

- [ ] `task-0030 Plan the alarm colours of minimal-dark-probes`: this plan and its log entry in the record
- [ ] `task-0030 Give the temperatures of minimal-dark-probes an alarm colour`: the tests, the theme file, the preview
  and the README, together, since the tests fail without the theme and the preview test fails without the image
- [ ] `task-0030 Record the implementation`: the ticked plan and the log entries of the record
