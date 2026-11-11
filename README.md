# Breakout Terminal

> A 25-level Breakout campaign for the Linux terminal, ported from a
> desktop-shell plugin by a scripted team of coding agents.

Breakout Terminal is a standalone Go application. It reimplements the behavior
of an MIT-licensed Breakout plugin written for the Omarchy desktop and runs in
an ordinary terminal, with no dependency on Omarchy, Quickshell, Hyprland, Qt,
or the original plugin host.

The port is also a process experiment. A shell script drove three coding agents
through five phases (reverse engineering, implementation, independent review,
correction, final gate), and each phase had to leave a written document behind
before the next one could start. The specifications, the review findings, and
the final verdict are all in [`docs/`](docs/).

## Status

**Preview.** The port passed its own final gate on 2026-09-06
([`docs/FINAL_REVIEW.md`](docs/FINAL_REVIEW.md)). Hands-on play since then has
still turned up bugs that are not triaged in this repository yet, so expect
rough edges. The documented limitations are listed under
[Known limitations](#known-limitations).

## Quick start

Requirements:

- Go 1.25 or newer to build from source. A Go installation that can acquire the
  Go 1.25 toolchain also works.
- A Linux terminal of at least 40 columns by 20 rows. 60x30 or larger is
  recommended.
- UTF-8 and truecolor are recommended, but not required. ASCII and reduced-color
  modes are included.

```sh
git clone https://github.com/ely-id/breakout-terminal.git
cd breakout-terminal
go build -o breakout ./cmd/breakout
./breakout
```

For a static release-style build, which needs neither Go nor a C runtime to
run:

```sh
CGO_ENABLED=0 go build -trimpath -o breakout ./cmd/breakout
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

## How it was built

[`harness/run.sh`](harness/run.sh) opens a [Herdr](https://herdr.dev) workspace with three Codex
agents (mechanics, portability, builder) and sends each one a fixed prompt from
[`harness/prompts/`](harness/prompts/). The script checks that a phase produced
its document before it starts the next phase, and it exits with an error unless
the final review begins with `VERDICT: PASS`.

```text
Phase 1   mechanics    ─┐ in parallel   → docs/GAME_SPEC.md
          portability  ─┘               → docs/PORT_SPEC.md
Phase 2   builder                       → the Go port + docs/BUILD_REPORT.md
Phase 3   portability, as reviewer      → docs/REVIEW.md
Phase 4   builder, corrections          → code + updated BUILD_REPORT.md
Phase 5   portability, as final gate    → docs/FINAL_REVIEW.md (first line: VERDICT)
```

Three rules shaped the result:

- **The stack was a finding, not a starting point.** [`AGENTS.md`](AGENTS.md)
  forbids choosing a language or terminal library before `PORT_SPEC.md` exists.
  Go with Bubble Tea and Lip Gloss was selected against the tick rate, input,
  and rendering requirements found in the original.
- **The reviewer does not build.** The review phase goes to the agent that did
  not write the code. It reads the specifications, the build report, and the
  source, and may not edit production code.
- **Disagreement has to be written down.** The builder must address every valid
  P0 and P1 finding, and any finding it rejects needs a technical reason in the
  build report.

What the record shows:

| Step | Result |
| --- | --- |
| First review (2026-09-02) | 0 P0, 5 P1, and 6 P2 findings |
| Correction phase | All 5 P1 fixed; 5 of the 6 P2 fixed; 1 P2 deferred with a reason |
| First final gate | Reported 1 remaining P1 that the correction had missed: bricks disappeared in small ASCII viewports |
| Targeted fix | Renderer corrected, with a regression test over 700 rendered frames (25 levels, 7 sizes, 4 color profiles) |
| Final gate (2026-09-06) | `VERDICT: PASS`, with 2 P2 items left open |

See [`harness/README.md`](harness/README.md) to run the pipeline yourself.

## Architecture

The game core has no terminal dependency. It imports only `math`, `sort`, and
`strings` from the standard library, so the simulation can be tested without an
interactive terminal.

| Path | Responsibility |
| --- | --- |
| `cmd/breakout` | Entry point: flags, terminal checks, program start |
| `internal/game` | Deterministic campaign simulation: physics, collisions, levels, bosses, power-ups, scoring |
| `internal/render` | Projects game snapshots into terminal cells, in Unicode or ASCII |
| `internal/app` | Bubble Tea adapter: input, frame delivery, resize, lifecycle |
| `internal/audio` | Semantic audio cues, with no playback backend |
| `internal/storage` | High scores and the mute preference, under XDG state |

Physics always runs in the original 800x1000 world at 240 Hz, while rendering
targets 60 Hz. Resizing changes presentation only: the world uses one centered,
uniform scale and is letterboxed where the terminal aspect ratio does not match.
Below 40x20 the adapter pauses the game and shows the required size.

High scores and the mute preference are stored in `$XDG_STATE_HOME/breakout`,
falling back to `$HOME/.local/state/breakout`. Missing, corrupt, or read-only
state is non-fatal. The game performs no runtime network access and launches no
external commands.

## Documents

| Document | Written by | Contents |
| --- | --- | --- |
| [`AGENTS.md`](AGENTS.md) | Author | Standing contract every agent reads: goal, portability invariant, fidelity, licensing, validation |
| [`docs/GAME_SPEC.md`](docs/GAME_SPEC.md) | Mechanics agent | Observable behavior of the original game, with evidence and confidence notation |
| [`docs/PORT_SPEC.md`](docs/PORT_SPEC.md) | Portability agent | Stack decision, package boundaries, and acceptance criteria |
| [`docs/BUILD_REPORT.md`](docs/BUILD_REPORT.md) | Builder agent | What was implemented, intentional deviations, and how each review finding was handled |
| [`docs/REVIEW.md`](docs/REVIEW.md) | Portability agent, as reviewer | Independent review with P1 and P2 findings |
| [`docs/FINAL_REVIEW.md`](docs/FINAL_REVIEW.md) | Portability agent, as reviewer | Final gate verdict and validation evidence |

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/breakout
```

The suite has 39 test functions across 7 files. It covers the game core, the
renderer, the Bubble Tea adapter, audio cue selection, and state persistence.
The two direct frontend dependencies are pinned in `go.mod`: Bubble Tea v2.0.9
and Lip Gloss v2.0.5.

## Known limitations

- **Audio is silent.** Cue selection and the mute preference are preserved, but
  there is no playback backend.
- **Input on legacy terminals is approximate.** Terminals that report
  key-release events preserve held movement. Others use a 140 ms input lease
  refreshed by key repeat, so movement across the initial auto-repeat delay is
  less faithful than the original. One-shot keys use an 800 ms initial guard and
  a 180 ms quiescence guard, which can suppress an intentional repeated press.
- **Effects are simplified.** Trails and impact marks are drawn, but the
  original particle, shake, ring, and flash effects are not reproduced.
- **The renderer allocates more than it needs to.** A cached 120x75 frame
  allocates about 1.26 MB. Frame time is within budget, and the work is deferred.
- **Not every terminal was tested.** Cell aspect ratio, color downsampling, and
  glyph width vary by emulator. `-ascii` is the guaranteed fallback.
- **No release binaries yet.** Build from source with the commands above.

The full list, with the reasoning for each item, is in
[`docs/BUILD_REPORT.md`](docs/BUILD_REPORT.md).

## Provenance and license

The gameplay behavior and campaign data were reimplemented from an MIT-licensed
Breakout plugin, copyright (c) 2026 acrogenesis. The plugin's audio files were
not copied, and `reference/plugin-source/` is a local snapshot used by the
harness that is excluded from the repository.

Released under the [MIT License](LICENSE), which retains the original notice.
Dependency and trademark notices are in
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).
