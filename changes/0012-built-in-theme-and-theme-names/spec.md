# A built-in theme and themes found by name

## Context

Step 6 of the roadmap of the initial plan is sensordeck's own themes: at least one minimal dark theme, required before
the project is published. Unit `0011-theme-compiler` gives the theme folder format, the compiler, which takes the folder
as an `fs.FS`, and the commands `sensordeck theme compile <theme-dir> <out.dat>` and
`sensordeck theme preview <theme-dir> <out.png>`. Unit `0010-theme-file-reader` gives the reader and the renderer that
the preview goes through. The initial plan says where themes live: the built-in ones embedded in the executable with
`go:embed` and read-only, so that one download is enough for a working screen, and the ones a user installs in
`sensordeck/themes/` under the config directory that Go's `os.UserConfigDir` gives (`~/.config` on Linux, or
`$XDG_CONFIG_HOME`, and `%AppData%` on Windows), where a user theme with the name of a built-in one takes its place.

This unit adds:

- the first built-in theme, `minimal-dark`, a dashboard of six tiles and a bar for the clock and the date
- the lookup of a theme by name in `theme compile` and `theme preview`, beside the folder they take today
- `sensordeck theme list`
- a quick start in the README that takes a new user from nothing to the built-in theme on the screen

"R*n* of unit 0011" below names a requirement of `changes/0011-theme-compiler/spec.md`. "The themes directory" is the
directory `sensordeck/themes` in the directory that `os.UserConfigDir` gives, as the configuration path of `run` is
found today.

### What `run` does with a metric it cannot read

The tiles of `minimal-dark` show GPU metrics, which need `nvidia-smi`, and the temperature of the first NVMe drive,
which a host may not have. In the code of `run`, the values of a round are those of the mapped metrics that were read in
that round, and a mapped metric that gave no reading is left out of the value update. Nothing is sent in its place. So
the screen keeps the last value it was sent for that tag, and for a tag that was never sent a value since the theme was
uploaded it shows whatever the device shows for a number it has no value for. What that is has not been seen on the
device: no document of the project and no test records it. The README must say this and nothing more certain.

### How the background of the prototype was checked

The orchestrating session drew a prototype of the background and checked the preview of the theme with the compiler of
unit 0011. For this spec the prototype was decoded pixel by pixel: it holds exactly three colours, `#0b0f14`,
`#141b24` and `#1f2a36`, with no anti-aliasing, and the rule of R4 below gives every one of its 153600 pixels. The
prototype is not in the repository and the sandbox does not have it; R4 is the whole of what the background is.

## Requirements

### The built-in theme

- **R1** The executable holds a built-in theme named `minimal-dark`, embedded with `go:embed` from a theme folder in the
  repository in the format of unit 0011, so that it is found and compiled with no file of the host. The built-in themes
  are read-only: no command writes to them
- **R2** The `theme.yaml` of `minimal-dark` is exactly this file, byte for byte, with a line feed after its last line:

  ```yaml
  name: Minimal Dark
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
    - {type: text, font: label, text: TOTAL, x: 20, y: 234}
    - {type: number, font: small, metric: gpu.mem.total, digits: 5, x: 76, y: 233}
    - {type: text, font: label, text: DISK, x: 176, y: 150}
    - {type: text, font: unit, text: "°C", x: 292, y: 150}
    - {type: number, font: big, metric: nvme.1.temp, x: 176, y: 176}
    - {type: text, font: label, text: NETWORK, x: 332, y: 150}
    - {type: text, font: unit, text: KiB/s, x: 424, y: 150}
    - {type: text, font: label, text: UP, x: 332, y: 186}
    - {type: number, font: small, metric: net.up, digits: 5, x: 396, y: 185}
    - {type: text, font: label, text: DOWN, x: 332, y: 222}
    - {type: number, font: small, metric: net.down, digits: 5, x: 396, y: 221}
    - {type: datetime, font: clock, format: "HH:mm:ss", x: 20, y: 279}
    - {type: datetime, font: date, format: "YYYY-MM-DD", x: 364, y: 283}
  ```

- **R3** The folder of `minimal-dark` holds `theme.yaml` and `background.png` and nothing else. No font file is in it or
  anywhere in the repository for it: its fonts are Go fonts
- **R4** `background.png` is a PNG of 480 x 320 pixels whose every pixel is opaque and is the colour this rule gives:
    - "the rounded rectangle from (x0, y0) to (x1, y1) of radius r", with the corners inclusive, holds the pixel (x, y)
      when x0 <= x <= x1, y0 <= y <= y1 and (x - cx)² + (y - cy)² <= r x (r + 1), where cx is x clamped to the range
      x0 + r to x1 - r and cy is y clamped to the range y0 + r to y1 - r
    - the seven panels are the rounded rectangles of radius 10 with the top left corners (8, 8), (164, 8), (320, 8),
      (8, 140), (164, 140) and (320, 140), each to the corner 151 pixels to the right and 123 down of it, and the bar
      from (8, 272) to (471, 311)
    - a pixel in the rounded rectangle from (x0 + 1, y0 + 1) to (x1 - 1, y1 - 1) of radius 9 of a panel is `#141b24`,
      the fill; a pixel in the panel of radius 10 but not in that one is `#1f2a36`, the outline of 1 pixel; every pixel
      in no panel is `#0b0f14`
- **R5** `background.png` is reproducible from the repository: a generator program in it, run with `go generate`, draws
  the picture of R4 and writes it as `background.png`, and a test decodes the committed `background.png` and fails
  unless each of its pixels equals the pixel of the generator's drawing
- **R6** `minimal-dark` compiles with no error and no warning of the reader of unit 0010 when read back, and its tags
  file, as R20 of unit 0011 writes it, holds this mapping:

  ```yaml
  tags:
    cpu.temp: 1
    cpu.util: 2
    gpu.mem.total: 3
    gpu.mem.used: 4
    gpu.temp: 5
    gpu.util: 6
    net.down: 7
    net.up: 8
    nvme.1.temp: 9
    ram.used: 10
    ram.util: 11
  ```

- **R7** Every metric of `minimal-dark` is one of the table of metrics in the README, which the Linux sensors of the
  project produce: `cpu.temp`, `cpu.util`, `gpu.temp`, `gpu.util`, `gpu.mem.used`, `gpu.mem.total`, `ram.used`,
  `ram.util`, `nvme.1.temp`, `net.up` and `net.down`

### A theme named on the command line

- **R8** The first argument of `theme compile` and of `theme preview` is a theme: a folder path or a theme name. It is a
  folder path when it is `.` or `..` or holds `/`, and on Windows also when it holds `\` or `:`. Any other argument is a
  theme name. The empty argument is an error that says a theme name or folder is needed. The rule looks only at the
  argument and not at the file system
- **R9** A folder path is compiled as unit 0011 compiles a folder, with its errors of R8 and R23 of unit 0011 unchanged
- **R10** A theme name is looked up in this order, and the first that is found is compiled:
    1. an installed theme: the entry of that exact name in the themes directory, when it is a directory or a symbolic
       link that leads to one. It is compiled as a folder path to that entry is, and its errors name the path of its
       `theme.yaml`
    2. a built-in theme of that exact name. An error of its compilation names it as the built-in theme of that name
- **R11** A name that is found in neither place is an error that names it and says that a folder is named by a path,
  giving `./<name>` as the example, with exit status 1 and no file written
- **R12** The themes directory not existing is no error: it holds no installed theme. When the config directory cannot
  be found, or the entry of the name in the themes directory cannot be examined for another reason than that it does
  not exist, the command fails with an error that names the cause, and the path where there is one, and writes no file
- **R13** Compiling `minimal-dark` by its name gives the same theme file and tags file, byte for byte, as compiling a
  folder that holds the same `theme.yaml` and `background.png`
- **R14** The command line help of `theme compile` and `theme preview` names the first argument as a theme name or
  folder

### Listing the themes

- **R15** `sensordeck theme list` takes no argument and writes one line for each theme to standard output, sorted by
  name by its bytes, with the fields separated by tabs:
    - `<name>`, `built-in` for a built-in theme that no installed theme replaces
    - `<name>`, `installed` and the folder for an installed theme whose name no built-in theme has
    - `<name>`, `installed, overrides built-in` and the folder for an installed theme with the name of a built-in one,
      which is then listed once
- **R16** The installed themes of the list are exactly the entries of the themes directory that R10 would find by their
  names. The folder of the list is the path of the entry, the themes directory joined with the name
- **R17** `theme list` follows R12: a themes directory that does not exist gives the built-in themes alone, and a config
  directory that cannot be found, or a themes directory that cannot be read, is an error with exit status 1
- **R18** `theme list`, and `theme compile` and `theme preview` with a name, open no device and read no sensor, with
  `--device` or without it, as R24 of unit 0011 says of the folder

### The documents

- **R19** `README.md` has a section "Quick start", before "Planned usage", that takes a new user on Linux from nothing to
  `minimal-dark` on the screen, in this order: the udev rule of the section on access to the screen, by a link to it;
  `sensordeck theme compile minimal-dark img.dat`; `sensordeck upload img.dat`; a config file that holds the `tags:`
  block of `img.tags.yaml`, shown in full as R6 gives it, at the path the section on sensor values names; and
  `sensordeck run`. It also:
    - names what each tile shows: the CPU temperature and load, the GPU temperature and load, the share of RAM in use
      and the GiB used, the VRAM used and its total in MiB, the temperature of the first NVMe drive, the network rates
      up and down in KiB/s, and the time and the date
    - says that the GPU tiles need `nvidia-smi` and show the first NVIDIA GPU, and that the disk tile is the first NVMe
      drive in the order of the places of the hardware, as the table of metrics says
    - says what happens to a tile whose metric the host cannot read, as the context says and no more: the value is not
      sent, so the tile keeps the last value it was sent, and a tile that was never sent one shows what the screen
      shows for a number with no value, which has not been checked on the device
    - says that `sensordeck theme preview minimal-dark preview.png` draws the theme as a PNG with no screen
    - says that the quick start is for Linux, because `run` reads no sensors on Windows yet
- **R20** The README describes `theme list`, the lookup of R8 and R10, the themes directory on Linux and on Windows, and
  that a theme folder copied there under a name is found by that name and replaces a built-in theme of that name. Every
  command of the README and of `docs/theme-folder.md` that names a theme folder by a bare name, such as
  `sensordeck theme preview my-theme preview.png`, names it by a path, such as `./my-theme`, so that each one still does
  what it says. The status line names the built-in theme `minimal-dark` and `theme list` among what works, and "Planned
  usage" shows the first argument of `theme preview` as a theme name or folder
- **R21** `docs/theme-folder.md` describes how a theme is named on the command line (R8), the order of the lookup (R10),
  the themes directory on Linux and on Windows, the override of a built-in theme, `theme list`, and the built-in themes
  by name

### Tests

- **R22** The unit tests cover at least:
    - R5, the committed `background.png` against the generator's drawing at every pixel, and the rule of R4 at chosen
      pixels: a corner of the background, a pixel of a fill, of a straight outline, of a rounded corner outline, and a
      pixel just outside a rounded corner
    - R6, the compilation of `minimal-dark`, its read back and its tags
    - R13
    - R8 on Linux: `.`, `..`, `./x`, `a/b`, `/abs`, `x`, `x.y` and the empty argument
    - R10 to R12: an installed theme alone, a built-in theme alone, an installed theme that replaces a built-in one, a
      symbolic link to a folder in the themes directory, a file of the name in the themes directory, which is not a
      theme, a name found nowhere, a themes directory that does not exist, and a config directory that cannot be found
    - R15 to R17 for the same cases
- **R23** The end-to-end tests run the built executable with `XDG_CONFIG_HOME` set to a temporary directory, and check
  `theme list` with no themes directory, with an installed theme and with one that replaces `minimal-dark`;
  `theme compile minimal-dark` with the output of R22 of unit 0011, its theme file read back by `internal/themefile` and
  its tags file; `theme preview minimal-dark` against the preview of the compiled file; the compilation of an installed
  theme by its name and of a folder given as `./<name>` from the directory that holds it; a name found nowhere, with
  the error of R11 and no file; and R18 with no fake screen
- **R24** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 A name has no separator.** An argument is told apart by its text alone: one that holds a separator, or is `.` or
  `..`, is a path, and any other is a name. This is the rule a shell applies to `./script` and `script`, so it is
  familiar, and it gives the same answer in every working directory. The cost is that a folder in the working directory
  can no longer be named bare: `my-theme` is a name and the folder is `./my-theme`. The error of R11 says so, and the
  README and the folder document are changed with it. Rejected: a folder first when one of that name exists, which
  makes `minimal-dark` mean another theme in a directory that happens to hold a folder of that name; a name first when
  one is known, which silently compiles the built-in theme for a user who is editing a copy of it in a folder of the
  same name, the likely way to start a theme of one's own; and a flag such as `--name`, which makes the common case the
  longer one. On Windows `\` and `:` also make a path, because both are separators of a Windows path and neither can be
  in the name of a folder. `.` and `..` are paths so that no name reaches the themes directory itself or its parent
- **D2 Installed first, then built-in.** The initial plan says that an installed theme with the name of a built-in one
  overrides it, which lets a user change the built-in theme without a build of their own. A name is compared exactly.
  On a file system that ignores case, as Windows usually does, the installed entry is found as the file system finds
  it, and the built-in theme only by its exact name
- **D3 Any directory is an installed theme.** An entry of the themes directory is a theme when it is a directory, or a
  symbolic link to one, and nothing more is checked, so the list and the lookup agree by construction. A folder with no
  `theme.yaml` is listed and fails to compile with the clear error of unit 0011 that names the path of `theme.yaml`.
  Checking each folder when it is listed is an item of "Beyond the request"
- **D4 No fallback when the config directory is unknown.** When `os.UserConfigDir` fails, as when neither
  `XDG_CONFIG_HOME` nor `HOME` is set, the command fails, as `run` does without `--config`. Falling back to the built-in
  themes alone would compile the built-in theme for a user whose installed theme of that name is in a directory the
  process could not find, which is the wrong theme with no message
- **D5 The built-in themes are a set the commands iterate over.** The lookup and the list read the built-in themes from
  one embedded tree of theme folders and name no theme in their code, so that a second built-in theme is a new folder
  and changes no command. The compiler of unit 0011 takes an `fs.FS`, so the embedded folder is compiled as it is,
  through the same path as a folder on disk, and R13 holds by construction
- **D6 The background is drawn by a generator and checked pixel by pixel.** The standards ask that a repeated piece of
  knowledge is written once. A PNG committed alone would be a picture that the repository could not make again, and a
  test that only checks it at chosen pixels would let the rest drift unnoticed. A generator that draws R4 is the one
  place the picture is described in code, `go generate` writes it, and the test fails as soon as the committed PNG and
  the generator differ, at any pixel. The comparison is of pixels, not of bytes, because the bytes of a PNG depend on the
  encoder and its compression level, which may change with the Go release, while the compiler reads only the pixels.
  The cost over a committed PNG alone is one small program and one test. Rejected: drawing the background when the
  theme is compiled, which would make the built-in theme unlike any theme a user can write, whose background is a file
- **D7 The exact rule of the corners.** R4 is the rule that gives the prototype at every pixel, with no anti-aliasing.
  A pixel's centre is taken at its whole coordinates, and the limit r x (r + 1) is (r + 0.5)² for whole numbers, so a
  corner is round to the eye and the rule needs no floating point. The fill is the panel shrunk by 1 pixel with a radius
  1 smaller, so that the outline is 1 pixel wide along the straight sides and stays closed at the corners
- **D8 The theme is the orchestrating session's design.** The `theme.yaml` of R2 was designed and its preview checked
  with the compiler of unit 0011 by the orchestrating session, and is carried here as it is. Its tiles show the metrics
  the roadmap names for the setup it was made for: CPU, GPU, VRAM, RAM, NVMe and the clock. The roadmap also names
  T_Sensor, a board probe that the hardware of the project no longer has, so it is left out
- **D9 The list shows the names the lookup takes.** A theme is listed by the name that R10 finds it by, the folder name,
  and not by the `name:` of its `theme.yaml`, which is a free text such as `Minimal Dark` and would need each theme to
  be read and to be valid for the list to work. The lines are tab-separated, as those of `sensordeck sensors` are, so
  that they can be read by a script
- **D10 The threat model.** The built-in themes are part of the executable and trusted. An installed theme is a folder
  the user put in their own config directory, and is compiled with the confinement of unit 0011, which keeps the
  compiler inside the folder. A name cannot leave the themes directory, because by R8 it holds no separator and is not
  `.` or `..`. A symbolic link in the themes directory is followed, because it is the user's own way of installing a
  theme that lies elsewhere; the confinement then applies inside the folder it leads to. The unit does not defend
  against a themes directory that another user can write, or against one that changes while a command runs
- **D11 No device check in this unit.** Whether the screen shows `minimal-dark` as its preview shows it is for the human
  to check on the device later. The unit is complete without that check and does not wait for it, and a finding of the
  check is a new unit. Until then the README does not say that the theme has been seen on the device
- **D12 Windows is built, not run.** The lookup and the list use `os.UserConfigDir` and the path functions of Go, so the
  themes directory is `%AppData%\sensordeck\themes` on Windows with no code of its own, and the separators of R8 come
  from the path rules of the platform. The suite cross-builds and vets for Windows. Running the commands on Windows is
  the human's to check, and the unit does not wait for it

## Non-goals

- `theme use`, `theme install`, `theme new`, the active theme in the config file, the compile cache, uploading a theme
  when it changes and reading the tags of the active theme in `run`: they come with the service mode unit. Until then
  the user pastes the tags block by hand, as the quick start shows
- Marking the active theme in `theme list`, which needs `theme use`
- Other built-in themes, such as one without a GPU for a host without `nvidia-smi`
- A command that copies a built-in theme into a folder to edit it
- A preview PNG kept in the theme folder for listings, which the initial plan names as optional
- Changing what `run` sends for a metric it cannot read, such as sending 0 so that a tile does not show a stale value
- The quick start on Windows, which waits for the Windows sensors of step 7
- `theme dump`, which takes a theme file and not a theme, and is unchanged
- Font files: the theme uses only the Go fonts of unit 0011, under their BSD licence, so no font file and no new licence
  is added

## Beyond the request

1. A check of each installed folder in `theme list`, marking one with no `theme.yaml`. Cost: one file check per entry
   and a test. Without it, such a folder is listed and `theme compile` with its name gives the clear error that names
   the missing `theme.yaml`
2. Showing the `name:` of each theme in `theme list` beside the name it is found by. Cost: reading and checking each
   `theme.yaml` when listing, a rule for a folder whose `theme.yaml` is invalid, and tests. Without it, the list shows
   the folder names, which are what the commands take
3. A test that the tags block of the quick start in the README equals the tags of `minimal-dark`. Cost: a test that
   reads the README's code block. Without it, a later change of the theme's metrics that does not update the README
   gives a user who pastes the old block tiles with the wrong values, which they see on the screen; the test of R6
   already fails on such a change and points the implementer at the tags
4. Restricting theme names to a pattern, such as lower-case letters, digits and hyphens. Cost: a check in the lookup and
   the list, and a rule for the entries the list then leaves out. Without it, any folder name is a theme name, which
   works because a folder name never holds a separator
