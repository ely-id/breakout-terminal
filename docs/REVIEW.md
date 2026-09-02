# Independent port review

Date: 2026-09-02

## Verdict

Go with Bubble Tea and Lip Gloss remains an appropriate presentation-layer
choice for this game. The implementation keeps `internal/game` independent of
the terminal framework, preserves the 240 Hz fixed-step simulation, renders at
a separate 60 Hz target, and produces self-contained Linux binaries. Direct
comparison with the reference found the principal collision, scoring, boss,
power-up, and transition rules to be close behavior-level ports rather than
framework copies.

There are no P0 findings. The port is playable and standalone, but it is not yet
fully compliant with its own input and presentation requirements. The P1 items
below should be corrected before calling the initial port faithful across its
documented terminal modes.

## Presentation-layer assessment

| Requirement | Review result |
| --- | --- |
| Update/tick frequency | Suitable. `internal/game.Game.Advance` retains the reference 1/240-second fixed step and 50 ms delivery clamp; the Bubble Tea adapter requests 60 Hz frames and supplies measured elapsed time. |
| Rendering performance | Time budget passes on the review machine, including a 100-ball frame, but allocation volume is unnecessarily high; see P2-4. |
| Keyboard responsiveness | Bubble Tea v2 exposes the needed enhanced press/repeat/release events, but the adapter's held-key and legacy-repeat bookkeeping has behavioral bugs; see P1-1 and P1-2. |
| Collision/game-loop architecture | Suitable. Collision and simulation live in the standard-library-only game core and do not depend on Bubble Tea, Lip Gloss, terminal size, or wall-clock reads. No collision-rule discrepancy was found in the reviewed paths. |
| Terminal resize handling | The lifecycle pause and immutable core coordinates are correct, and PTY resize recovery worked. The renderer does not implement the specified uniform letterboxed projection; see P1-3. |
| Color fidelity | Truecolor/256/16 palettes are explicit and Bubble Tea handles final terminal downsampling. The monochrome encoding loses gameplay distinctions; see P1-5. |
| Unicode/glyphs | An automatic/explicit ASCII fallback exists. Width is nevertheless calculated by rune count for potentially ambiguous-width glyphs; see P2-3. |
| Portability | Passes the main invariant: no runtime Omarchy, plugin-host, compositor, GUI, shell-command, or network dependency was found. The `OMARCHY` level name/art is campaign content, not an integration dependency. |
| Binary distribution | `CGO_ENABLED=0` Linux amd64 and arm64 builds were statically linked and had no dynamic section. |
| Dependency complexity | Acceptable. Production has only the selected Bubble Tea and Lip Gloss direct dependencies; no Bubbles/widget framework or unrelated runtime dependency was introduced. |

## Findings

### P1-1 — Direction repeats can change action and leave stale movement/aim intent

Affected code: `internal/app/model.go`, especially `press`, `remember`,
`directionAction`, `release`, and `expireLegacy` (lines 129-169 and 227-269).

The reference chooses an arrow's action on the initial press and ignores every
auto-repeat (`reference/plugin-source/Panel.qml`, `Keys.onPressed`, lines
540-561). The port instead runs `directionAction` on every repeat. If sticky
state changes while an arrow is held, `remember` overwrites the held entry with
the new action without releasing the previous one. For example, a repeated Left
press can change from `MoveLeft` to `AimLeft`, leaving both core booleans set. In
legacy mode, lease expiry releases only the newest action, so the earlier
movement intent can remain set indefinitely. Enhanced release eventually clears
both arrow actions, but behavior remains wrong until release.

There is a second stuck-key edge during keyboard capability negotiation:
`expireLegacy` returns immediately once `enhancedEvents` becomes true. A leased
key received before the enhancement response can therefore stop expiring even
if no corresponding enhanced release exists.

Expected correction: choose and store a direction action only on its initial
press. Repeats should refresh the existing lease and must not issue or replace a
different action. If an action ever is replaced, send `KeyUp` for the old action
first. Expire entries according to whether that entry was leased, rather than a
global current capability flag, and safely clear/migrate leased entries when
enhanced reporting activates. Add tests for sticky catch/release during a held
arrow and for legacy-to-enhanced negotiation with an active lease.

### P1-2 — Legacy auto-repeat retriggers one-shot actions after 140 ms

Affected code: `internal/app/model.go`, `press` and `expireLegacy` (lines
129-179 and the one-shot branches at 199-225).

The same 140 ms lease is used for directions and for Space, Enter, P, F10, and
secret/code keys. When a legacy entry expires, the next terminal auto-repeat is
reported as an ordinary press and is treated as a new initial press. Typical
terminal repeat delay is longer than 140 ms, so holding P toggles pause once on
the real press and again when repeat begins. Holding Space from attract can
start a run and then launch it on the later repeat. F10 can similarly toggle
back. The reference explicitly ignores auto-repeat, and the behavior contradicts
`BUILD_REPORT.md`'s “repeat-safe one-shot actions” claim. The documented
limitation only explains directional delay and suppression of very fast double
presses, not long-held one-shot retriggering.

Expected correction: separate directional leases from one-shot repeat
suppression. Enhanced terminals should continue using the repeat/release event
metadata. The legacy path needs a separately tuned and tested quiescence/guard
policy that covers the terminal's initial repeat delay, with its unavoidable
trade-off documented. At minimum, do not claim repeat-safe legacy one-shots
until a PTY test holds P, Space, and F10 through initial auto-repeat.

### P1-3 — Common terminal sizes stretch the world instead of letterboxing it

Affected code: `internal/render/render.go`, `dimensions`, `transform`, and
`Render` (lines 118-125, 166-172, and 518-536).

`transform.x` always consumes the selected width while `transform.hy` always
consumes the selected height. These are independent scales, so a 40x20, 60x30,
80x30, or 120x20 viewport changes the apparent aspect ratio of bricks,
trajectories, paddle motion, and boss shapes. Centering occurs only when a
dimension exceeds the independent 120x75 caps. This differs from the reference's
uniform centered fit and from `PORT_SPEC.md` lines 176-179, which require a
centered, letterboxed transform for the immutable 800x1000 world.

Expected correction: derive one uniform scale from both available dimensions
(accounting for the renderer's half-cell vertical coordinate system), calculate
x/y offsets for the unused cells, and use that same transform for every static
and dynamic entity. Add geometric tests at non-matching aspect ratios that
assert landmark spacing and offsets; current golden hashes only freeze the
stretched output and cannot establish the required geometry.

### P1-4 — Power-up capsules do not preserve distinct identities

Affected code: `internal/render/render.go`, drop rendering in `overlayDynamic`
(lines 274-292).

All UTF-8 drops use the same cyan color. Slow and sticky both become `S`; life
becomes `L` instead of the reference `+`; multi loses the `3`; and magnet becomes
`M` instead of the reference `U`. More seriously, ASCII mode renders every
power-up as `*`. Players therefore cannot reliably tell which effect they are
about to catch. This violates `PORT_SPEC.md`'s requirement that color
degradation not change power-up identity and is not merely an aesthetic
difference. The renderer tests do not exercise the eight capsule kinds.

Expected correction: define a stable per-kind glyph and color-role table based
on the reference (`↔`, `S`, `+`, `×3`, `F`, `G`, `▰`, `U`) and provide distinct,
one-cell-safe ASCII equivalents or short labels for all eight. Add forced
truecolor, monochrome, and ASCII tests proving that every kind remains
distinguishable.

### P1-5 — Monochrome and ASCII encodings lose required board semantics

Affected code: `internal/render/render.go`, `style`, `buildStatic`, and `compose`
(lines 91-107, 174-227, and 466-515).

In monochrome UTF-8 mode every role is `NoColor`. A solid region whose top and
bottom halves share a role is emitted as a space, so filled bricks and the frame
largely disappear except where a role transition happens to produce `▀`.
Normal, damaged/reinforced, and steel bricks then have the same geometric
encoding. Special labels are also omitted below 80 columns. In ASCII mode the
brick glyphs are better, but the frame still consists only of color roles and
therefore renders as spaces. This fails the non-color distinction and plain
border requirements in `PORT_SPEC.md` lines 194-200 and 297-310. It also makes
`NO_COLOR` less usable than README.md claims.

Expected correction: make semantic shape independent of color. Use explicit
glyph/pattern choices for normal, damaged, steel, special, and boss cells in
monochrome, and write real ASCII border characters in ASCII mode. Tests should
assert the visible symbols and pairwise distinction of these object types, not
only absence of ANSI escapes and whole-frame hashes.

### P2-1 — The documented 40-column layout clips critical status

Affected code: `internal/render/render.go`, `writeText`, the status-header
format, and `Render` (lines 230-246, 331-367, and 518-536).

At 40 columns, the normal header is about 42 cells even after level-name
truncation. Centering begins at a negative x coordinate, so `writeText` drops
both ends; the current lives value is among the clipped characters. The footer
is truncated generically rather than using a compact control layout. The
below-minimum message is also returned as one over-width line rather than a
viewport-sized view. Thus “compact layouts” in `BUILD_REPORT.md` is overstated
and the minimum is only partially usable.

Expected correction: introduce explicit compact header/footer variants that
prioritize score, level, lives, launch/pause, and quit within 40 cells. Fit and
center the too-small notice within the actual viewport. Add field-presence and
dimension assertions at widths 40 and 41.

### P2-2 — Boss rendering discards most boss-specific visual state

Affected code: `internal/render/render.go`, `drawBoss` (lines 388-410).

All machine boss kinds are the same filled rectangle; `BossKind` changes only
the apple's color. `lookLeft` and `lookRight`, which the game core correctly
produces during dodge behavior, fall through to the default face. THE BITE is a
red rectangle rather than a recognizable striped bitten-apple silhouette. The
name and numeric health preserve basic playability, but the result does not
support `BUILD_REPORT.md`'s plural “compact chassis/face glyphs” claim or the
recognizable boss identity described in `GAME_SPEC.md`.

Expected correction: add small, distinct terminal silhouettes for creep,
reload, maze, monolith, and apple, and map every emitted face state. This is a
presentation-only enhancement; `BossKind`, `BossFace`, position, stage, and HP
already cross the snapshot boundary, so no simulation change is needed.

### P2-3 — Text placement assumes rune count equals terminal cell width

Affected code: `internal/render/render.go`, `writeText`, `centered`, `truncate`,
special-label placement, and boss placement (lines 230-246, 322-328, 388-410,
and 420-463).

Text is centered and truncated with `len([]rune(...))`. The selected glyph set
contains characters such as `↔`, `×`, `●`, and `·` whose displayed width can
vary with terminal width policy. A two-cell rendering of one of these glyphs
shifts the cell framebuffer and can corrupt the following column. This is the
exact assumption `PORT_SPEC.md` lines 212-218 cautions against.

Expected correction: use terminal display-width measurement for placement and
truncation, constrain dynamic entity marks to verified one-cell glyphs, and add
width tests for every non-ASCII glyph. Keep `-ascii` as the guaranteed fallback.

### P2-4 — Cached frames still allocate and compose the full framebuffer

Affected code: `internal/render/render.go`, `clone`, `overlayDynamic`, and
`compose` (lines 139-145, 248-386, and 466-515).

The static brick raster is cached, but every frame clones all 9,000 cells and
recomposes every style run. Independent benchmarks on the report's stated CPU
measured cached 120x75 frames at roughly 1.55-2.30 ms, so the frame-time target
passes. They also measured about 872 KB and 15,518 allocations per frame,
independent of whether one or 100 balls were present. At 60 Hz that is roughly
52 MB/s and 930,000 allocations/s before Bubble Tea parses/diffs the view. The
soak showed no retained leak, but this is avoidable GC/CPU pressure and the
implementation is not yet a genuinely sparse dynamic renderer.

Expected correction: reuse framebuffer and output buffers, restore only the
previous dynamic dirty cells (or use a low-allocation overlay representation),
and avoid per-run Lip Gloss object/string churn. Retain both the compose
benchmark and an end-to-end Bubble Tea render benchmark after optimization.

### P2-5 — Trails and semantic impact feedback are produced but never presented

Affected code: `internal/game/game.go`, `Snapshot`; `internal/app/model.go`,
`refresh`; and all of `internal/render/render.go`.

The core copies six trail points into each `BallView` and drains semantic events
into each snapshot. The app uses events only to save scores and select an unused
audio cue; the renderer reads neither `BallView.Trail` nor `Snapshot.Events`.
There is therefore no trail, impact flash, shake, ring, or other bounded visual
impact response. `BUILD_REPORT.md`'s statement that the renderer “conveys
deterministic semantic events” is unsupported by the implementation, and the
terminal-specific effects promised in `PORT_SPEC.md` lines 312-323 are absent.

Expected correction: consume trails and selected semantic events in bounded,
density-limited presentation state owned by the app/renderer, leaving the game
core unchanged. If this remains intentionally deferred, amend the build report
to say that events are exposed but not visually conveyed. Record these as visual
enhancements, not behavioral changes.

### P2-6 — Persisted high-score normalization is narrower than the reference

Affected code: `internal/storage/storage.go`, `LoadHighScores` (lines 80-83),
and `internal/game/game.go`, `LoadHighScoreState` (lines 1125-1143).

The reference accepts finite numeric/coercible values, floors scores and levels,
clamps levels, rejects bad rows individually, and clamps a finite negative
`lastScore` to zero. Go unmarshals directly into integer fields, so a fractional
or numeric-string value rejects the entire file. A negative `lastScore` is
ignored rather than normalized to zero. Files written by this port contain
integers, so this does not block normal play, but it is a behavioral difference
and makes the build report's unqualified “high-score normalization” claim
broader than the actual compatibility surface.

Expected correction: decode persisted rows through an input DTO using
`json.Number`/raw values, normalize valid rows independently with the reference
rules, and default a valid negative last score to zero. Add mixed valid/invalid,
fractional, numeric-string, and negative-last-score fixtures.

## Validation performed

The review ran and observed the following:

- `go test ./...`: pass.
- `go test -race ./...`: pass.
- `go vet ./...`: pass.
- Native `go build -trimpath -o /tmp/breakout-review-native ./cmd/breakout`: pass.
- `CGO_ENABLED=0` Linux amd64 and arm64 release-style builds: pass; `file`
  reported both statically linked and `readelf -d` reported no dynamic section.
- `node --test reference/plugin-source/tests/*.js`: 4/4 reference suites pass.
- Renderer benchmarks, three samples: cached one-ball 1.73-2.30 ms, cached
  100-ball 1.55-1.69 ms, static rebuild 1.79-1.83 ms; allocation observations
  are recorded in P2-4.
- `BREAKOUT_SOAK=1` 36,000-frame soak: pass in 34.37 seconds with its retained
  heap and goroutine checks.
- Independent real-PTY smoke: entered/left alternate screen, started play,
  delivered legacy arrow input, paused/resumed, shrank below 40x20, displayed
  the size notice, grew again, and exited with Escape and Ctrl+C. Terminal
  attributes and cursor state were restored. Separate SIGINT and SIGTERM checks
  also restored terminal state; externally delivered SIGINT returns exit 1 with
  Bubble Tea's interruption error, while Ctrl+C input and SIGTERM exit cleanly.
- Source/import audit: no production network import, process execution,
  hardcoded user path, Omarchy command, plugin API, compositor API, or desktop
  runtime dependency was found.

`govulncheck` was not installed in the review environment, so the historical
vulnerability-scan result stated in `BUILD_REPORT.md` was not independently
reproduced. No finding is assigned solely for that unavailable tool.

## Behavioral changes versus visual enhancements

Behavioral/input corrections are P1-1 and P1-2. P2-6 is a persisted-data
compatibility correction. They should not be described as visual polish.

Presentation corrections are P1-3 through P1-5 and P2-1 through P2-5. The
letterbox transform, semantic glyph encodings, compact layouts, boss art,
width-safe text, renderer allocation work, and bounded effects can all be
implemented without changing world coordinates, collision rules, scoring,
timers, or state transitions. Any such improvements should be recorded under
visual/terminal enhancements, while the gameplay core remains framework
independent.
