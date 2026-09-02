# Standalone terminal port specification

Status: **approved architecture for the initial port**

Decision date: 2026-09-02

## Decision

Use Go with Bubble Tea v2 and Lip Gloss v2 for the standalone terminal
frontend, subject to the acceptance checks in this document.

The selection is based on terminal lifecycle, resize, color-profile, renderer,
and modern keyboard-event support. It is not based only on appearance. The
architecture is deliberately split as follows:

```text
terminal events / clock / resize
              |
              v
  Bubble Tea application adapter
      |                    |
      | input intent       | immutable snapshot
      v                    v
framework-independent   terminal renderer
     Game Core          + Lip Gloss styles
      |
      v
semantic events (score, impact, transition, audio cue)
```

Bubble Tea is an application and event-loop adapter. Lip Gloss is a styling
and layout dependency. Neither owns gameplay state, advances physics, or
decides collisions. The Game Core must compile and be testable without a TTY
and without importing either package.

Use the stable v2 module paths and pin exact patch versions in `go.mod` and
`go.sum`. The versions evaluated on the decision date were:

- `charm.land/bubbletea/v2` v2.0.9;
- `charm.land/lipgloss/v2` v2.0.5.

These versions require Go 1.25 according to their published module files.
Builders therefore need Go 1.25 or a Go installation capable of acquiring
that toolchain. End users of release binaries do not need Go.

The evaluation machine did not have Go installed, so this phase could not run
a Bubble Tea render prototype. Performance and PTY smoke checks below are
mandatory implementation gates, not assumed facts.

## Evidence from the original

The reference directory is read-only input to the port. Source locations in
this section refer to `reference/plugin-source/`.

### Intrinsic game requirements

| Requirement | Evidence and consequence for the port |
| --- | --- |
| Deterministic fixed-step simulation | `engine.js:12` defines `STEP = 1 / 240`; `engine.js:925-931` accumulates elapsed time and calls `tick(STEP)`. Preserve a 240 Hz logical step. |
| Render rate is separate from simulation rate | `Panel.qml:621-643` advances on `FrameAnimation`, clamps frame time to 50 ms, and paints once per display frame. The terminal must not redraw 240 times per second. |
| Fast late-game motion | Level launch speeds rise from 560 to 1,040 world units/s (`levels.js:4-35`), then increase by up to 22% as bricks clear (`engine.js:582-599`). At the maximum target speed, one 1/60-second move is taller than a normal 20-unit brick; retaining the 1/240 step is important to collision fidelity. |
| Held directional input | `keyDown` and `keyUp` maintain left/right intent (`engine.js:498-519`); the paddle moves at 660 world units/s (`engine.js:529-551`). Responsive press and release handling is preferable to discrete menu-like key handling. |
| Collision work belongs in the core | Ball, paddle, wall, shield, brick, boss-shot, and falling-drop rules live in `engine.js`, including a 64-unit brick-band index. Terminal cell rounding must never feed back into collision geometry. |
| Stable world coordinates | Physics and Canvas rendering use an 800 x 1,000 portrait world (`engine.js:10-13`, `renderer.js:3-5`). Resize must alter only the world-to-cell transform. |
| Bounded ordinary scene, dense secret scene | The largest generated level has 116 brick objects. Normal multiball is capped at 9 (`engine.js:17`), while the exact hidden code deliberately creates 100 balls (`engine.js:451-486`). The presentation must remain usable in both cases. |
| Recognizable level art | Levels 1-24 are 19-column symbolic brick mosaics (level 23 is an 18-column sampled Omarchy mark); level 25 is a large Apple-shaped boss arena. A terminal renderer must preserve occupied-cell silhouettes, even when it cannot preserve pixel curves. |
| Game status and overlays | Score, level/name, lives, combo, active powers, messages, high scores, pause, game-over, and victory states are visible in `renderer.js:373-803`. These are required information, not decoration. |
| Canonical palette | `Panel.qml:52-61` contains a complete Tokyo Night-like fallback palette. It is usable without Omarchy and becomes the standalone canonical palette. |
| Small Unicode set | The Canvas text uses `●`, `↔`, `×`, `▰`, arrows, a middle dot, and em dashes. None is needed by collision logic. There must be an ASCII-safe rendering path. |

The reference mechanics tests passed during this evaluation: 44 engine checks,
the effects and audio suites, and the 25-level campaign integration suite. A
read-only Node timing probe also showed that the existing fixed-step engine is
small compared with a display-frame budget, including its 100-ball mode. That
probe does not establish the speed of the future Go renderer.

### Rendering and host details that are not game rules

- JavaScript is the original engine implementation language; QML Canvas is
  its pixel renderer. Neither is part of the gameplay contract.
- The two Canvas layers cache mostly-static bricks separately from balls and
  effects (`Panel.qml:511-532`). The terminal port should retain the idea of a
  static cache, not the Canvas API.
- QML `FrameAnimation`, Qt key events, Qt Multimedia, `FloatingWindow`, and
  `IpcHandler` are framework or plugin lifecycle choices.
- Reading `.../omarchy/current/theme/colors.toml`, invoking `hyprctl`, shell
  summon/toggle IPC, the bar widget, monitor selection, and the 80%-of-screen
  floating-window policy are Omarchy/desktop integrations. None may appear in
  the standalone launch path.
- The original can distinguish physical key release and ignores Qt
  auto-repeat. A traditional terminal byte stream often cannot. This is an
  environment limitation that the input adapter must expose and mitigate.

## Criteria evaluation

### Update frequency and game-loop architecture: suitable

Bubble Tea's serialized `Update` method can own an adapter clock without
owning game rules. Schedule presentation messages at a target of 60 Hz. Each
message supplies monotonic elapsed time to `Game.Advance`, which:

1. ignores non-positive elapsed values and paused gameplay;
2. clamps a single elapsed value to 50 ms, as the reference does;
3. adds it to a private accumulator;
4. executes as many exact 1/240-second simulation steps as fit; and
5. retains only the sub-step remainder.

Use the measured elapsed time rather than assuming every Bubble Tea timer
fires exactly on schedule. Schedule exactly one successor timer per animation
message so delayed messages cannot multiply timer chains. No gameplay
goroutine is required: keyboard messages and animation messages can update the
same core serially, avoiding races. Visual-only effects may have their own
presentation clock but must consume semantic core events rather than alter the
simulation clock.

A nominal 60 Hz presentation gives the core about four fixed updates per
render and bounds normal input-to-visible-response delay to one display frame.
If validation shows that a terminal cannot sustain it, a 30 Hz presentation
mode is acceptable; the Game Core remains at 240 Hz in either mode.

### Rendering performance: suitable with a bounded cell renderer

Bubble Tea v2's renderer is cell-based and diffs successive full views. This
matches a full-screen terminal game better than manually repainting a QML-like
pixel canvas. It does not remove the need for an efficient game renderer.

The renderer must:

- maintain a cell framebuffer and compose one complete frame string;
- cache the background, frame, level bricks, header labels, and footer until
  the level, theme, score/status text, or viewport invalidates them;
- overlay dynamic balls, paddle, drops, boss, shots, messages, and effects on
  a copy or sparse overlay of that cache;
- prebuild the small palette of Lip Gloss styles and render same-style runs,
  rather than constructing a style for every world pixel every frame;
- bound its play area (a recommended maximum is 120 columns by 75 rows) so an
  unusually large terminal does not cause unbounded work;
- coalesce multiple objects landing in the same cell and reduce trails or
  particles during the 100-ball secret, following the reference's own dense
  rendering safeguards.

Lip Gloss is appropriate for palette styles, borders, labels, overlays, and
joining/placing terminal blocks. It should not be used as a scene graph or
called once for every virtual 800 x 1,000 pixel. A custom world-to-cell
rasterizer is still required.

### Keyboard responsiveness: suitable in enhanced terminals, degraded fallback required

Bubble Tea v2 can request progressive keyboard event types and reports
whether they are supported. Set
`View.KeyboardEnhancements.ReportEventTypes = true`. When supported:

- `KeyPressMsg` sets core left/right or sticky-aim intent;
- repeat messages do not retrigger one-shot actions or secret characters;
- `KeyReleaseMsg` clears the matching intent;
- launch, new game, pause, sound, secret sequences, and quit are handled on
  the initial press.

This can preserve the original held-key model in supporting terminals.

When event-type reporting is unavailable, no TUI framework can infer a key
release that the terminal did not send. The adapter must use a legacy input
lease: a left/right press sets intent for a short, fixed duration; subsequent
terminal repeats refresh it; expiry synthesizes release. The duration must be
covered by adapter tests and tuned in the PTY/manual test, not placed in the
Game Core. This prevents a tap from moving forever. Movement after a legacy
terminal's initial auto-repeat delay will be less faithful and must be stated
in the README.

Arrow keys retain their special sticky-ball meaning: arrows aim a held ball,
while `A`/`D` continue moving the paddle. Bubble Tea input handling must also
preserve the letter and Konami buffers without counting key repeats.

### Terminal resize handling: suitable

Handle every Bubble Tea `WindowSizeMsg`. Store terminal columns/rows only in
the application/render model. Recompute a centered, letterboxed transform for
the immutable 800 x 1,000 world and invalidate the static cell cache. Never
rescale or rebuild physics objects in response to a resize.

Use a vertical half-block (`▀`) in the normal Unicode renderer so one terminal
cell can carry separate top foreground and bottom background colors. With the
usual tall terminal cells this gives useful vertical resolution for the
portrait board. Do not assume exact cell pixel dimensions, because the normal
resize event exposes terminal cells as the portable unit.

The complete information layout should be considered enhanced at 60 x 30 or
larger and usable, with reduced labels/effects, at 40 x 20 or larger. Below
that, display a clear size requirement and pause through an adapter-owned
`pausedByResize` reason; restore the prior pause state when adequate space
returns. This is a lifecycle safeguard and is listed as an intentional
behavioral change below.

### Color fidelity: suitable with terminal-dependent limits

Use the exact opaque RGB fallback values from `Panel.qml` as the canonical
dark palette. On a truecolor terminal, Lip Gloss can emit those RGB values. On
256-color, 16-color, monochrome, or `NO_COLOR` output, colors must degrade
while object shape and status text remain legible. Tests must force each color
profile rather than depending on the developer's terminal.

Canvas-only effects cannot be exact in a character grid: gradients, alpha,
rounded corners, arbitrary arcs, rotation, sub-cell motion, and smooth scale
transforms will be approximated. Preblend a small number of alpha shades
against the known board background where useful; do not emit unsupported
alpha. Color degradation is a presentation limitation and must not change
brick type, hit points, power-up identity, or collision state.

Do not read the Omarchy theme path. A portable, optional theme-file format can
be considered after the faithful initial port, but is not part of this scope.

### Unicode and glyphs: suitable with an explicit fallback

UTF-8 mode may use half blocks, bullets, arrows, multiplication signs, and box
drawing. Validate display width before adding a glyph: the renderer's cell
coordinates, not byte/rune counts, are authoritative. Avoid Nerd Font icons,
emoji, and ambiguous-width glyphs. The bar icon `` is host chrome and is not
part of the standalone game.

Provide an ASCII mode, selected by a command-line flag and automatically when
the locale is clearly non-UTF-8. It should use plain borders, `o` for balls,
`=` for the paddle, `*`/letters for power-ups, `x` for multipliers, and ASCII
control hints. ASCII mode may have lower vertical resolution but must remain
playable. `NO_COLOR` affects color only and must not itself imply ASCII.

### Portability and distribution: suitable

Go is suitable for release as versioned Linux `amd64` and `arm64` binaries.
Build releases with `CGO_ENABLED=0` and verify that they have no dynamic
runtime dependencies. Embed required immutable data with `go:embed`; keep
writable state under `$XDG_STATE_HOME/breakout` with the standard
`$HOME/.local/state/breakout` fallback. Failure to read or write state must not
prevent play.

The executable must use terminal input/output only. It must make no runtime
network requests and invoke no shell, Omarchy, Quickshell, Qt, Hyprland,
Wayland, X11, compositor, or desktop-session command. It should fail clearly
when standard input/output are not usable as a terminal. Bubble Tea must
restore terminal modes and leave the alternate screen on normal exit, error,
SIGINT, and SIGTERM.

`Escape` exits rather than hiding a reusable plugin panel, because the
standalone process has no host to hide into. `Ctrl+C` is an additional terminal
lifecycle exit. Both must preserve any high-score update already earned and
restore the terminal.

### Dependency complexity: acceptable, but not minimal

The production code has two direct presentation dependencies. Their evaluated
module files collectively list roughly eighteen unique transitive modules
(color/profile, terminal, ANSI, Unicode-width, diff, and platform support).
This is materially more than a direct ANSI implementation. Most are shared by
both libraries and are compiled into the release binary, so end users do not
install them separately.

This cost is accepted because Bubble Tea supplies raw-mode lifecycle, signals,
resize delivery, terminal capability negotiation, key-event enhancements,
alternate-screen cleanup, and a diffing renderer. Reimplementing and testing
all of those is higher risk for this port. Do not add Charm's Bubbles component
library: this game does not need form widgets. Do not use Lip Gloss subpackages
such as tables or trees unless a demonstrated requirement appears.

Pin versions, commit `go.sum`, run license/vulnerability review in CI, and
review patch upgrades deliberately. Bubble Tea v2 is relatively new, and a
reported v2.0.9 resize issue can show a one-render stale frame in resize-driven
views; the resize stress test below is required. A transient stale frame is
acceptable, but a wrong persistent viewport or changed core state is not.

## Package boundaries

The implementation should use boundaries equivalent to these; exact names may
vary without weakening dependency direction:

```text
cmd/breakout/          parse flags, load/save state, run Bubble Tea
internal/game/         state, levels, fixed-step simulation, collision, events
internal/app/          Bubble Tea model, clock messages, lifecycle, input adapter
internal/render/       snapshots -> cell framebuffer; Lip Gloss styles/layout
internal/storage/      XDG high-score/settings persistence
internal/audio/        optional semantic-event consumer; no gameplay authority
```

`internal/game` must depend only on the Go standard library. It exposes
commands/input intent, `Advance(elapsed)`, snapshots, and semantic events. It
must not expose terminal dimensions, ANSI strings, Lip Gloss styles, Bubble
Tea messages, wall-clock reads, filesystem access, or audio handles.

The application model is allowed to contain the Game Core; Bubble Tea's Elm
model is not the Game Core. The renderer receives read-only snapshots. Storage
and audio consume explicit state/events at application boundaries.

Audio is not provided by Bubble Tea or Lip Gloss and is not selected by this
presentation decision. Preserve semantic audio events and the mute preference
boundary so an audio adapter can be evaluated independently. Do not invoke an
external audio player merely to reproduce Qt Multimedia.

## Visual plan

### Faithful visual identity

- Retain the portrait frame, dark surface/playfield, centered `BREAKOUT`
  header, score/level/lives row, game board, status area, and control footer.
- Reproduce every level's occupied brick mosaic and alternating level palette.
- Distinguish normal, damaged/reinforced, steel, special, and boss bricks in
  both color and non-color modes.
- Keep the paddle/ball, boss name/phase/health, drops, shield, sticky aim,
  active-power timers, combo, multiball count, transient message, attract,
  pause, game-over, victory, and high-score presentations recognizable.
- Preserve semantic visibility before decorative effects when space or output
  bandwidth is limited.

### Terminal-specific visual enhancements

These are presentation improvements, not gameplay changes:

- half-block vertical sampling for smoother ball motion and more recognizable
  brick silhouettes;
- crisp box-drawn frame and palette-colored status chips in UTF-8/color mode;
- responsive centering and compact header/footer variants;
- cached static level art plus sparse dynamic overlays;
- terminal-appropriate trails, impact flashes, and screen shake with density
  limits instead of literal Canvas blur/rotation;
- automatic palette downsampling and a deliberate monochrome encoding.

### Intentional behavioral or lifecycle changes

Keep these separate from the enhancements above:

- `Escape` terminates the standalone application instead of hiding and
  pausing a host-owned panel; `Ctrl+C` is also accepted for terminal hygiene.
- Terminals without release-event support use the documented input lease, so
  directional hold timing cannot be exactly identical to Qt key-down/up.
- A viewport below the playable minimum temporarily pauses gameplay and
  resumes only the pause reason introduced by the resize adapter.
- Rendering is quantized to cells and may run at 30 rather than 60 Hz on a
  slow terminal; the Game Core still advances in exact 1/240-second steps.

No other gameplay changes are authorized by this presentation choice.

## Options considered

| Option | Result |
| --- | --- |
| Go + Bubble Tea v2 + Lip Gloss v2 | **Selected.** Best overall fit for event-loop ownership, resize, terminal cleanup, color downsampling, optimized cell diffs, and enhanced press/release reporting. Requires a custom game framebuffer and legacy-input fallback. |
| Bubble Tea v1 | Rejected for a new port. It lacks v2's explicit key-release messages/capability report and would start the port on an older API solely to lower toolchain/dependency churn. |
| Go + direct ANSI/raw TTY | Viable and potentially smaller, but would require implementing terminal mode restoration, signals, resize, escape parsing, capability detection, output diffs, and cross-terminal testing already supplied by Bubble Tea. |
| Go + tcell | Viable low-level screen API, but it does not eliminate the terminal protocol's legacy key-release limitation and provides less benefit for the desired styled overlays/layout. Switching does not justify another abstraction layer. |
| QML/Qt or the existing JavaScript host | Rejected. It preserves the environment architecture rather than producing a normal terminal binary and carries Qt/plugin-host dependencies. |
| Node.js terminal frontend | Rejected for the initial port. It would ease reuse of JavaScript but requires a runtime at execution and encourages copying framework-shaped code instead of isolating a testable game core. |
| Rust + a terminal crate | Technically capable, but offers no requirement-level advantage sufficient to offset a slower behavior port and a second unfamiliar implementation surface. |

## Licensing and provenance

The reference plugin declares MIT in `manifest.json` and contains an MIT
license with `Copyright (c) 2026 acrogenesis`. MIT permits copying and
modification provided the copyright and permission notice accompany copied or
substantial portions. The README calls the plugin locally maintained by
`ely`; that does not replace the copyright notice in the license.

Prefer a behavior-level Go reimplementation rather than translating
`engine.js` or `renderer.js` line by line. If level matrices, sound files, or
other assets are copied, distribute the reference MIT notice with them and
record their origin. The changelog calls the WAV cues original, but the files
contain no individual provenance metadata; retain the notice and do not claim
new authorship. The preview PNG is unnecessary for a terminal release and
should not be copied.

The level-23 matrix says it samples Omarchy's `quattro/icon.txt`. The upstream
Omarchy repository and that icon are also published in an MIT-licensed
repository. Preserve the relevant upstream MIT attribution if the sampled
mark is retained. Names and marks may also carry trademark significance even
when code is MIT-licensed; use them descriptively as part of the faithful game
campaign and do not imply endorsement.

Bubble Tea and Lip Gloss are MIT-licensed. A release must include the notices
required for distributed third-party code/assets, normally in a
`THIRD_PARTY_NOTICES` file or equivalent release documentation.

## Acceptance criteria

### Architecture and deterministic behavior

- `internal/game` has no Bubble Tea, Lip Gloss, terminal, clock, filesystem,
  audio, or environment imports.
- Unit tests cover fixed-step accumulation and the 50 ms clamp, paddle intent,
  wall/paddle/brick collisions, reinforced/steel behavior, multiball life
  loss, power-ups, boss transitions/attacks, level/life delays, pause,
  scoring/high scores, game over, victory, and deterministic level creation.
- A scripted core replay produces the same gameplay snapshots at 30 Hz and 60
  Hz adapter delivery and at multiple terminal sizes/color modes.
- No world coordinate or collision outcome changes after any resize sequence.

### Frontend and input

- Adapter tests cover initial press, repeat suppression for one-shot actions,
  enhanced release, legacy lease refresh/expiry, simultaneous opposite keys,
  sticky arrow aiming versus `A`/`D` movement, secret input, pause, and quit.
- Renderer golden tests cover at least 40 x 20, 60 x 30, 80 x 50, and 120 x 75
  in truecolor, 256-color, monochrome, and ASCII modes.
- All 25 brick silhouettes are compared with expected normalized occupancy,
  independent of terminal styling.
- Resize stress includes repeated growth/shrink while balls, drops, and a boss
  are active; a subsequent frame must always use the current viewport.
- The 100-ball secret remains responsive and displays its actual ball count,
  even when multiple balls are visually coalesced.

### Performance

- Benchmarks separately measure `Game.Advance(1/60)`, snapshot creation,
  static-cache rebuild, and dynamic frame composition for 1, 9, and 100 balls.
- On the documented release-test machine, a 120 x 75 dense frame must compose
  within a 16.7 ms presentation budget at the median and must not show
  unbounded allocation or goroutine growth during a 10-minute run. Record the
  machine and results in `docs/BUILD_REPORT.md`; do not make a hardware timing
  threshold a flaky unit test.
- If 60 Hz is not sustainable, validate 30 Hz presentation with unchanged 240
  Hz core results and document the selected default.

### Build, smoke, and portability

- `go test ./...`, `go vet ./...`, and `go build ./...` pass from a clean
  checkout with the pinned module graph.
- A PTY smoke test launches into the alternate screen, starts a game, moves
  and releases the paddle, launches a ball, pauses/resumes, resizes, and exits
  while restoring terminal modes.
- A release build with `CGO_ENABLED=0` is verified to have no dynamic runtime
  library dependency and runs on ordinary Linux terminal sessions for each
  published architecture.
- Runtime tracing confirms no network call, external command, original plugin
  host, Omarchy path, compositor, or desktop environment is required.
- State follows XDG rules; missing, read-only, or corrupt state is non-fatal.
- README documentation states terminal/UTF-8/color expectations, the legacy
  release-event input limitation, build/run instructions, controls, and all
  intentional presentation/lifecycle deviations.

## External references used for the frontend decision

- Bubble Tea v2 API and keyboard enhancements:
  <https://github.com/charmbracelet/bubbletea/blob/main/tea.go>
- Bubble Tea v2 upgrade guide and stable module paths:
  <https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md>
- Bubble Tea v2.0.9 module graph:
  <https://github.com/charmbracelet/bubbletea/blob/v2.0.9/go.mod>
- Lip Gloss color profiles/downsampling and layout API:
  <https://github.com/charmbracelet/lipgloss>
- Lip Gloss v2.0.5 module graph:
  <https://github.com/charmbracelet/lipgloss/blob/v2.0.5/go.mod>
- Relevant Bubble Tea v2.0.9 resize report:
  <https://github.com/charmbracelet/bubbletea/issues/1780>
- Upstream Omarchy repository/license and referenced icon:
  <https://github.com/basecamp/omarchy> and
  <https://github.com/basecamp/omarchy/blob/quattro/icon.txt>
