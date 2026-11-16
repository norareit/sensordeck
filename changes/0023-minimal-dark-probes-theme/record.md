---
kind: task
status: draft
timestamp: 2026-11-16T20:16:20Z
---

# A second built-in theme for the Commander Core XT probes

sensordeck has one built-in theme, `minimal-dark`, from unit `0012-built-in-theme-and-theme-names`. This task adds a
second, `minimal-dark-probes`, as an example for a host with a Corsair iCUE Commander Core XT with two temperature
probes and a second NVMe drive. It is `minimal-dark` with two tiles changed: the line under the VRAM shows the
temperature of the second NVMe drive instead of the VRAM total, and the middle tile of the second row shows the two
Commander probes instead of the first NVMe drive. The orchestrating session designed the layout and checked its
preview with the compiler on 2026-11-15; it draws as intended. "R*n* of unit 0012" below names a requirement of
`changes/0012-built-in-theme-and-theme-names/spec.md`, and "R*n* of unit 0011" one of
`changes/0011-theme-compiler/spec.md`.

## What is true when the task is done

* The built-in themes are `minimal-dark` and `minimal-dark-probes`, the second a new folder beside the first in the
  embedded tree of theme folders. By D5 of unit 0012 the lookup and `theme list` take it with no change to their code.
  `minimal-dark` is unchanged, its `theme.yaml`, its `background.png` and its tags included
* The folder of `minimal-dark-probes` holds `theme.yaml` and `background.png`, both regular files, and nothing else
* Its `theme.yaml` is exactly this file, byte for byte, with a line feed after its last line. The time format is
  `"HH:mm:ss"` at x 20, as in `minimal-dark`; a copy on the human's machine with a leading space in the format is an
  experiment and is not shipped

  ```yaml
  name: Minimal Dark with probes
  author: sensordeck
  license: MIT
  background: background.png
  fonts:
    label: {builtin: go-medium, size: 15, color: "#7f8ea3"}
    unit:  {builtin: go-medium, size: 15, color: "#4f5d70"}
    big:   {builtin: go-bold, size: 46, color: "#e6edf3"}
    small: {builtin: go-medium, size: 17, color: "#c9d4df"}
    clock: {builtin: go-bold, size: 24, color: "#e6edf3"}
    date:  {builtin: go-medium, size: 17, color: "#7f8ea3"}
  widgets:
    - {type: text, font: label, text: CPU, x: 20, y: 18}
    - {type: text, font: unit, text: "°C", x: 136, y: 18}
    - {type: number, font: big, metric: cpu.temp, x: 20, y: 44}
    - {type: text, font: label, text: LOAD, x: 20, y: 102}
    - {type: number, font: small, metric: cpu.util, x: 70, y: 101}
    - {type: text, font: unit, text: "%", x: 140, y: 102}
    - {type: text, font: label, text: GPU, x: 176, y: 18}
    - {type: text, font: unit, text: "°C", x: 292, y: 18}
    - {type: number, font: big, metric: gpu.temp, x: 176, y: 44}
    - {type: text, font: label, text: LOAD, x: 176, y: 102}
    - {type: number, font: small, metric: gpu.util, x: 226, y: 101}
    - {type: text, font: unit, text: "%", x: 296, y: 102}
    - {type: text, font: label, text: RAM, x: 332, y: 18}
    - {type: text, font: unit, text: "%", x: 452, y: 18}
    - {type: number, font: big, metric: ram.util, x: 332, y: 44}
    - {type: text, font: label, text: USED, x: 332, y: 102}
    - {type: number, font: small, metric: ram.used, x: 382, y: 101}
    - {type: text, font: unit, text: GiB, x: 436, y: 102}
    - {type: text, font: label, text: VRAM, x: 20, y: 150}
    - {type: text, font: unit, text: MiB, x: 128, y: 150}
    - {type: number, font: big, metric: gpu.mem.used, digits: 5, x: 20, y: 176}
    - {type: text, font: label, text: DISK, x: 20, y: 234}
    - {type: number, font: small, metric: nvme.2.temp, x: 76, y: 233}
    - {type: text, font: unit, text: "°C", x: 136, y: 234}
    - {type: text, font: label, text: GPU CABLE, x: 176, y: 150}
    - {type: text, font: unit, text: "°C", x: 292, y: 150}
    - {type: number, font: big, metric: commander.1.temp, x: 176, y: 176}
    - {type: text, font: label, text: INTAKE, x: 176, y: 234}
    - {type: number, font: small, metric: commander.2.temp, x: 240, y: 233}
    - {type: text, font: unit, text: "°C", x: 292, y: 234}
    - {type: text, font: label, text: NETWORK, x: 332, y: 150}
    - {type: text, font: unit, text: KiB/s, x: 424, y: 150}
    - {type: text, font: label, text: UP, x: 332, y: 186}
    - {type: number, font: small, metric: net.up, digits: 5, x: 396, y: 185}
    - {type: text, font: label, text: DOWN, x: 332, y: 222}
    - {type: number, font: small, metric: net.down, digits: 5, x: 396, y: 221}
    - {type: datetime, font: clock, format: "HH:mm:ss", x: 20, y: 279}
    - {type: datetime, font: date, format: "YYYY-MM-DD", x: 364, y: 283}
  ```

* Its `background.png` has the same pixels as that of `minimal-dark`: the picture of R4 of unit 0012, drawn by the
  same generator. See "How the two themes share the background" below
* It compiles with no error and no warning of the reader of `internal/themefile` when read back, and its tags file, as
  R20 of unit 0011 writes it, is exactly this, with the tags that R18 of unit 0011 gives the metric names sorted by
  their bytes:

  ```yaml
  # Paste this block into the config file of sensordeck run.
  tags:
    commander.1.temp: 1
    commander.2.temp: 2
    cpu.temp: 3
    cpu.util: 4
    gpu.mem.used: 5
    gpu.temp: 6
    gpu.util: 7
    net.down: 8
    net.up: 9
    nvme.2.temp: 10
    ram.used: 11
    ram.util: 12
  ```

* `sensordeck theme list` with no themes directory writes `minimal-dark\tbuilt-in` and `minimal-dark-probes\tbuilt-in`,
  in that order, as R15 of unit 0012 sorts by bytes. An installed `minimal-dark` replaces only `minimal-dark`, and
  `minimal-dark-probes` stays `built-in`
* `docs/minimal-dark-probes.png` is committed beside `docs/minimal-dark.png`, and is the preview that
  `sensordeck theme preview minimal-dark-probes docs/minimal-dark-probes.png` draws

### How the two themes share the background

The picture is described once, in the generator of `internal/builtinthemes/background`, and each theme folder holds a
`background.png` written by it. `go generate ./internal/builtinthemes/` writes both files, and the test of R5 of unit
0012 is run for each of the two folders, so a PNG that drifts from the generator fails at the first differing pixel,
with the message that names `go generate` as it does today. The comments of the background package and of the
generator, which say they draw the background of `minimal-dark`, say that they draw the background the two built-in
themes share.

The second PNG is derived output, as a compiled file is, and not a second statement of the picture, so it repeats no
knowledge in the sense of the standards. The alternatives are rejected:

* A symbolic link from one folder to the other's PNG: `go:embed` does not embed a symbolic link, and the test of R3 of
  unit 0012 requires regular files
* A `background:` path that leads out of the folder, such as `../minimal-dark/background.png`: the confinement of the
  compiler of unit 0011 refuses it, and a built-in theme would then be one a user cannot copy as a folder
* Drawing the background when the theme is compiled, which D6 of unit 0012 rejected

### What the tests pin

* In `internal/builtinthemes`: the names of the built-in themes are exactly the two; each folder holds exactly
  `background.png` and `theme.yaml` as regular files; the `theme.yaml` of `minimal-dark-probes` is the text above,
  verbatim, and that of `minimal-dark` is still the text of R2 of unit 0012; the `background.png` of each folder equals
  the generator's drawing at every pixel; each theme compiles, reads back with no error and no warning, and gives its
  tags and its tags file exactly, `minimal-dark-probes` those above. The test of the two themes' backgrounds and the
  test of their compilation go through one table of the two themes, so that a third built-in theme is a row and not a
  copy of a test
* The test of the preview images, today in `docs_test.go` for `docs/minimal-dark.png`, covers `docs/minimal-dark-probes.png`
  too, by decoded pixels against `themeimage.Preview` of the compiled theme, and its failure message gives the command
  that writes the image of that theme again. The two images go through one table, as above
* The unit test of `theme list` in `internal/cli` that runs the real built-in themes, and the end-to-end test of
  `theme list` in `e2e/builtin_theme_test.go`, expect both built-in themes in every case they check, including the one
  where an installed `minimal-dark` overrides the built-in one and `minimal-dark-probes` stays `built-in`
* The end-to-end test compiles `minimal-dark-probes` by its name and checks its tags file, as it does `minimal-dark`

### The documents

* `README.md` has a section "Built-in themes", after "Quick start" and before "Commands", that lists the two themes.
  For each it gives a sentence on what it shows and its preview image, with an alt text that says what the tiles show,
  and says, as the README says today of `docs/minimal-dark.png`, that the images are previews and not photos of the
  screen
    * `minimal-dark`: the tiles of the quick start, by a link to it rather than a second list
    * `minimal-dark-probes`: `minimal-dark` with the VRAM total replaced by the temperature of the second NVMe drive,
      and the disk tile replaced by the two probes of a Corsair iCUE Commander Core XT
* The section says which hardware `minimal-dark-probes` expects: a Commander Core XT with two temperature probes,
  `commander.1.temp` and `commander.2.temp`, which the theme labels `GPU CABLE` and `INTAKE` because on the machine it
  was made for probe 1 is at the GPU power cable and probe 2 at the front intake; and a second NVMe drive, whose
  `nvme.2.temp` is the second drive in the order of the places of the hardware, its PCI path, as the table of metrics
  says. It links to the section on the Commander Core XT for what reading the controller does to its LEDs. It says that
  a tile whose metric the host cannot read behaves as the quick start says
* The section says that `minimal-dark-probes` is an example to copy and adapt: copy its folder,
  `internal/builtinthemes/themes/minimal-dark-probes` of the repository, into the themes directory under a name of one's
  own, and change the labels and metrics to those of one's own hardware
* The opening paragraph of the README, which says the built-in theme `minimal-dark` is there, names both built-in
  themes. The image of `minimal-dark` at the top stays
* `docs/theme-folder.md`, which lists the built-in themes as `minimal-dark`, lists both
* The quick start still uses `minimal-dark` and is otherwise unchanged

### Outside the task

* No change to the compiler, the lookup, `theme list`, `run` or the sensors. `commander.<n>.temp` and `nvme.<n>.temp` are
  read today
* No check on the device. Whether the screen shows `minimal-dark-probes` as its preview does is the human's to check,
  and the README does not say it has been seen on the device
* No theme without a GPU, no other example themes

The five commands of the suite in `AGENTS.md` succeed.

## Log

- 2026-11-16T20:16:20Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the record of the task from
  the request of the orchestrating session, against the README, `docs/theme-folder.md`, the spec of unit 0012,
  `internal/builtinthemes` and its tests, `docs_test.go` and the tests of `theme list`. Decided: the two themes share
  the background through the one generator, which writes a `background.png` into each folder and against which each
  is tested, rather than a symbolic link or a path out of the folder, which `go:embed` and the confinement of the
  compiler refuse; the tests of the backgrounds, the compilations and the preview images go through one table of the
  built-in themes. The tags file follows from R18 of unit 0011; the Go toolchain of the project could not be run on
  the host, so it was derived and not compiled here. The status is `draft`

## Amendments
- 2026-11-16T20:17:25Z [Claude Code, anthropic/claude-opus-5-5] read the record under the delegation in `AGENTS.md`: the theme is the
  layout whose preview the orchestrating session checked on 2026-11-15, without the leading space of the experiment,
  the background stays described once in its generator, and the tags file of the record equals the one the compiler
  wrote on the host for the human's copy of the theme, whose differing time format does not change the tags. The
  approval that follows is made under that delegation
