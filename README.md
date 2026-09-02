# Breakout Terminal

A standalone terminal port of the 25-level Breakout game in
`reference/plugin-source/`. It runs as a normal Linux terminal application and
does not require Omarchy, Quickshell, Hyprland, Qt, or the original plugin host.

## Dependencies

- Go 1.25 or newer to build from source. A Go installation that can acquire the
  Go 1.25 toolchain also works.
- A Linux terminal of at least 40 columns by 20 rows. 60x30 or larger is
  recommended.
- UTF-8 and truecolor are recommended, but not required. ASCII and reduced-color
  modes are included.

Release binaries are self-contained and do not require Go or a C runtime.

## Build and install

Build in the repository root:

```sh
go build -o breakout ./cmd/breakout
```

For a static release-style build:

```sh
CGO_ENABLED=0 go build -trimpath -o breakout ./cmd/breakout
```

Install with the Go toolchain:

```sh
go install ./cmd/breakout
```

The two direct frontend dependencies are pinned in `go.mod`: Bubble Tea v2.0.9
and Lip Gloss v2.0.5.

## Run

```sh
./breakout
```

Options:

```text
-ascii                    force ASCII-only glyphs
-color auto|truecolor|256|16|mono
-level 1..25              begin a fresh run at a campaign level
```

`NO_COLOR` selects monochrome output when `-color` is left at `auto`. A clearly
non-UTF-8 locale selects ASCII automatically. The application uses the
alternate screen and restores the terminal when it exits.

## Controls

| Key | Action |
| --- | --- |
| Left/Right or A/D | Move the paddle |
| Left/Right while a sticky ball is held | Aim the held ball |
| A/D while a sticky ball is held | Continue moving the paddle |
| Space | Start/restart, launch, or release sticky balls |
| Enter | Start/restart from attract, game-over, or victory |
| P | Pause/resume |
| F10 | Toggle the persisted mute preference |
| Escape or Ctrl+C | Save state and quit |

Modern terminals that report key-release events preserve held movement. On
legacy terminals, the game uses a 140 ms input lease refreshed by terminal key
repeat; a tap cannot move forever, but movement across the terminal's initial
auto-repeat delay is necessarily less faithful than the original Qt input.
Legacy one-shot keys use a separate 800 ms initial guard followed by a 180 ms
quiescence guard, so holding Space, P, or F10 does not repeatedly activate it.
The trade-off is that an intentional identical press within that guard can be
suppressed on terminals that cannot report releases.

## State and portability

High scores and the mute preference are stored in
`$XDG_STATE_HOME/breakout`, falling back to
`$HOME/.local/state/breakout`. Missing, corrupt, or read-only state is
non-fatal. The game performs no runtime network access and launches no external
commands.

When the viewport becomes smaller than 40x20, the adapter pauses gameplay and
shows the required size. Restoring the viewport removes only that resize-owned
pause. Physics always remains in the original 800x1000 world at 240 Hz; resize
changes presentation only. The world uses one centered, uniform scale and is
letterboxed where the terminal aspect ratio does not match. Monochrome and
ASCII modes retain explicit frame, brick, boss, and power-up symbols.

The initial port preserves semantic audio cues and mute persistence but has no
audio playback backend, so it plays silently. See [docs/BUILD_REPORT.md](docs/BUILD_REPORT.md)
for validation results and documented presentation deviations.
