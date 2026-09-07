# Build report

Initial validation: 2026-09-02. Final-review correction: 2026-09-06.

## Final-review correction and validation — 2026-09-06

The sole remaining P1 in `docs/FINAL_REVIEW.md` was confirmed and corrected.
The old ASCII brick loop started at `(y0 + 1) / 2`, skipping a brick whose
entire projection occupied one odd half-cell. The new campaign regression
failed on the original implementation; the isolated real `L1-3-2` rectangle
rendered as a playfield dot for normal, reinforced, damaged, and steel states.
This was a gap in the earlier P1-5 resolution, whose tests missed small ASCII
projections.

ASCII composition now derives the material glyph from both stored half-cell
semantics. Either occupied half produces `=`, `%`, `+`, or `#` for normal,
reinforced, damaged, or steel respectively. If two materials share a terminal
cell, the lower occupied half takes precedence, consistent with campaign row
draw order. Explicit special labels keep precedence over material glyphs.
The separate ASCII row loop was removed. Game Core geometry, collision rules,
simulation, and Unicode rendering were not changed. Remaining P2 performance
and legacy-input work was not undertaken.

Regression coverage in `internal/render/ascii_test.go` includes:

- Real snapshots of all 25 campaign layouts at 40x20, 41x20, 60x24, 80x24,
  60x30, 80x50, and 120x75, with ASCII explicitly enabled in all four color
  profiles (700 rendered frames). The test independently projects live
  non-boss collision rectangles, including steel, and rejects any occupied
  terminal cell represented only by a dot or space. It also checks ASCII
  purity and exact frame dimensions. Boss art continues to have its separate
  existing tests.
- The reported `L1-3-2` rectangle at its actual campaign coordinates and the
  minimum viewport, verifying the exact material, special, and destroyed
  glyphs without enlarging the brick.
- Top-only, bottom-only, and both-half occupancy for each material, including
  identical color roles, mixed-material precedence, and special-label priority.

Focused renderer tests ran first, reproduced the bug before the correction,
then passed after it. Final validation on Linux 7.2.3 x86-64, Go 1.27.1,
AMD Ryzen 7 5800H:

- `go test ./internal/render -count=1`: pass, including the unchanged Unicode
  golden hashes and new ASCII properties.
- `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`,
  `go build ./...`, and `go mod verify`: pass.
- `node --test reference/plugin-source/tests/*.js`: all four reference suites
  pass.
- Native executable and `CGO_ENABLED=0` Linux amd64/arm64 builds: pass.
  `file` confirms static release binaries; `readelf -d` reports no dynamic
  section for either. Native and static amd64 binaries were actually launched;
  arm64 was cross-built and inspected, not executed on this x86-64 machine.
- Real PTY smoke: ASCII play at 40x20, movement/lease expiry, launch,
  pause/resume, shrink to 39x19, recovery through 120x40 and back to 40x20,
  and Escape exit pass. Separate Ctrl+C, SIGINT, and SIGTERM cases restore
  alternate screen, cursor, and termios state. External SIGINT exits one after
  cleanup; the other exits are zero. Non-TTY launch returns the documented
  diagnostic and status two.
- `BREAKOUT_SOAK=1 go test ./internal/render -run TestTenMinuteSimulatedSoak
  -count=1 -v`: 36,000 frames pass in 29.69 seconds with retained-heap and
  goroutine checks.
- Existing renderer benchmarks (`-bench . -benchmem -benchtime=1s -count=3`):
  medians 1.85 ms cached one-ball, 1.85 ms cached 100-ball, and 2.15 ms static
  rebuild, below 16.7 ms. Cached allocation cost remains about 1.26 MB and
  17,000 allocations per frame; no P2 optimization is claimed.
- `git diff --check`: pass. Only the renderer, its new regression tests, and
  this report changed. Protected specifications and reference sources remain
  untouched. Dependency versions did not change; the vulnerability-scan
  evidence below belongs to the original validation and was not rerun here.

## Implemented

- A standalone Go 1.25 application using Bubble Tea v2.0.9 and Lip Gloss
  v2.0.5. It uses terminal input/output only and has no Omarchy, plugin-host,
  compositor, Qt, or shell-command launch dependency.
- A standard-library-only `internal/game` core with the 800x1000 world, 240 Hz
  fixed-step accumulator, 50 ms delivery clamp, deterministic serve and speed
  progression, paddle and collision rules, combo/scoring/lives, high scores,
  all state transitions, and event generation.
- All 25 campaign definitions and exact occupied-cell matrices, deterministic
  reinforced/steel/power assignment, special bricks, the four campaign bosses,
  THE BITE arena, boss movement/projectiles/stages, capsule rain, all eight
  power-ups, typed secrets, and the exact Konami ball storm.
- A Bubble Tea adapter with a measured 60 Hz presentation clock, enhanced key
  press/repeat/release handling, stable initial direction actions, safe
  capability migration, a tested legacy 140 ms movement lease, and separate
  800 ms initial/180 ms quiescence guards for one-shot keys. It also owns
  resize pause, alternate-screen lifecycle, and Escape/Ctrl+C termination.
- A bounded 120x75 cell renderer with a cached static brick raster, dynamic
  overlays, a centered uniform half-cell projection, ASCII fallback, canonical
  Tokyo Night palette, explicit truecolor/256/16/monochrome profiles, compact
  layouts, distinct power-up and boss identities, semantic non-color brick
  patterns, trails, bounded impact markers, active-power status, shield,
  sticky aim, actual dense-ball count, and control text.
- Best-effort XDG high-score and mute persistence with atomic replacement.
  Reference-compatible loading normalizes valid rows independently, including
  finite numeric strings/fractions and negative last scores. Corrupt, missing,
  and unwritable state does not prevent play.
- The launcher prevents Bubble Tea's transitive color detector from invoking
  `tmux info`; TERM, COLORTERM, and terminfo capability detection remain, and
  the application invokes no helper process.
- Semantic audio cue priority, impact classification, and pitch-tier logic.
  This initial port deliberately has no playback backend.

The implementation is a behavior-level Go reimplementation. The original
JavaScript/QML architecture and WAV/PNG assets were not copied. The exact MIT
notice covering the reference and copied campaign data is retained in
`LICENSE`; dependency and sampled-mark notices are in
`THIRD_PARTY_NOTICES.md`.

## Intentional deviations

- Escape quits and restores the terminal rather than hiding a host-owned panel;
  Ctrl+C is also accepted.
- Terminal graphics quantize the Canvas scene to cells. Half blocks, flat
  colors, semantic chassis/face glyphs, short impact marks, and terminal status
  overlays replace pixel curves, gradients, rotation, alpha, shake, rings, and
  the full Canvas particle system.
- A viewport below 40x20 temporarily pauses through the application adapter.
- Terminals without release-event support use the documented direction lease
  and one-shot guard heuristic.
- Audio is silent. Cue selection and mute persistence remain explicit adapter
  boundaries; no external audio process is invoked.
- Boss presentation hit-freeze is not applied to simulation. It was a QML
  presentation effect rather than a core collision rule.

## Review disposition

`docs/REVIEW.md` reported no P0 findings. All five P1 findings were valid and
were corrected:

- P1-1: a direction action is selected only on initial press; repeats renew the
  stored action, legacy entries expire individually, and enabling enhanced key
  events safely releases pre-negotiation intent.
- P1-2: legacy one-shot input now has a long initial guard and a renewable
  quiescence guard whose deadline never moves backward. Unit and real-PTY tests
  cover held Space, P, and F10 through initial auto-repeat.
- P1-3: one scale now projects both world axes into half-cell coordinates and
  centers the unused space. Non-matching-aspect geometric tests verify equal
  landmark scale and balanced letterboxing.
- P1-4: all eight capsules have stable distinct UTF-8 and one-cell ASCII
  identities, with color used only as an additional cue.
- P1-5: the frame and normal, reinforced, damaged, steel, special, and boss
  cells have explicit monochrome/ASCII shapes independent of color.

The small, presentation-local P2 corrections were also applied: P2-1 compact
40/41-column layouts and fitted undersize notices; P2-2 distinct boss
silhouettes and every emitted face direction; P2-3 display-width-aware text and
verified one-cell glyphs; P2-5 density-limited trails and event-driven impact
marks; and P2-6 reference-compatible persisted-number normalization.

P2-4 is a valid performance observation, not a rejected finding. It was not
implemented because replacing full-buffer cloning and Lip Gloss style-run
composition with dirty-cell restoration/output-buffer reuse is an architectural
renderer change, not a small low-risk P2 fix. Fresh benchmarks still pass the
frame budget and the soak still shows no retained leak, but the allocation cost
is recorded under Known limitations. No review finding was otherwise rejected.

## Original validation — 2026-09-02

Validation passed on Linux 7.2.2 x86-64 with Go 1.27.0 (building the Go 1.25
module) and an AMD Ryzen 7 5800H:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- `CGO_ENABLED=0 go build -trimpath` for Linux amd64 and Linux arm64
- `file` identified both release builds as statically linked; `readelf -d`
  reported no dynamic section for either architecture.
- `node --test reference/plugin-source/tests/*.js` passed all four original
  JavaScript suites without modifying the reference tree.
- Unit coverage includes fixed-step accumulation/clamping, pause, paddle
  intent/clamps, wall/paddle/brick collisions, reinforced and steel bricks,
  scoring, multiball drain semantics, all power categories and sticky release,
  boss stages/volleys/projectile damage, life/level/end transitions, high-score
  normalization, all 25 deterministic silhouettes, and 30/60 Hz replay
  equivalence.
- Adapter coverage includes enhanced press/release, simultaneous directions,
  stable direction action across sticky-state changes, capability migration,
  legacy lease refresh/expiry, long-delay and early-duplicate one-shot repeat
  suppression, F10 persistence, sticky arrow aim versus A/D movement, typed
  and Konami secrets, resize pause, Escape, and Ctrl+C.
- Renderer coverage includes fixed golden hashes at 40x20, 60x30, 80x50, and
  120x75 for truecolor, 256-color, 16-color, and monochrome; uniform transform
  geometry; eight capsule identities; semantic monochrome/ASCII symbols;
  compact fields; undersize dimensions; Unicode display widths; all boss
  kinds/faces; trails/impacts; resize/core isolation; and 100-ball display.
- Storage round-trip/corruption boundaries, reference-compatible mixed-row
  normalization, and semantic audio priority/pitch tiers are tested.
- A real legacy-mode pseudo-terminal smoke entered the alternate screen, held
  Space, P, and F10 through simulated initial auto-repeat without retriggering,
  deliberately launched/unpaused/unmuted after quiescence, delivered movement,
  resized below and above 40x20, exited with Escape, left the alternate screen,
  and restored the original terminal attributes. Exit code was zero.
- Separate real-PTY lifecycle checks confirmed terminal and alternate-screen
  restoration for Ctrl+C input, SIGINT, and SIGTERM. Ctrl+C and SIGTERM exited
  zero; Bubble Tea reports externally delivered SIGINT as an interruption and
  exits one after cleanup.
- A non-TTY launch failed clearly with exit code 2 and the required terminal
  diagnostic.
- The opt-in soak (`BREAKOUT_SOAK=1`) composed 36,000 frames, equivalent to ten
  minutes at 60 Hz, at 120x75. It passed retained-heap and goroutine-growth
  checks in 30.81 seconds.
- A source/import audit found no application network package,
  external-command invocation, original host API, compositor call, or
  desktop-environment path. A launcher unit test verifies that the transitive
  color detector cannot see `TMUX` and therefore cannot invoke its helper.
- Go's official `govulncheck` v1.7.0 reported `No vulnerabilities found` for
  the application and pinned dependency graph. Dependency licenses were
  reviewed from the downloaded module notices and recorded in
  `THIRD_PARTY_NOTICES.md`.

Representative benchmark medians (`-benchmem`, 120x75 render):

| Operation | Result |
| --- | ---: |
| 1-ball `Advance(1/60)` | 5.33 µs |
| 9-ball `Advance(1/60)` | 42.2 µs |
| 100-ball `Advance(1/60)` | 463 µs |
| 100-ball snapshot | 5.54 µs |
| Cached 1-ball frame | 1.86 ms, 1.26 MB, 17,080 allocs |
| Cached 100-ball frame | 1.88 ms, 1.26 MB, 17,053 allocs |
| Static-cache rebuild plus frame | 2.17 ms, 2.08 MB, 17,158 allocs |

The dense frame is comfortably below the 16.7 ms presentation budget on this
machine, so the selected default remains 60 Hz while core simulation remains
240 Hz.

## Known limitations

- A traditional terminal cannot reveal key releases it never sends. Legacy
  auto-repeat therefore moves less faithfully during its initial repeat delay.
  Identical one-shot presses within the 800 ms initial guard, or before 180 ms
  of quiescence after repeats, may be suppressed.
- The renderer still clones and composes the full cell buffer each frame. It
  meets the measured frame budget and does not retain memory in the soak, but
  the roughly 1.26 MB and 17,000 allocations per cached 120x75 frame are
  avoidable GC pressure (review P2-4).
- Cell aspect ratio, color downsampling, ambiguous-width terminal policy, and
  refresh throughput remain terminal-dependent. The selected UTF-8 glyphs are
  one cell under the pinned display-width implementation; `-ascii` is the
  guaranteed fallback and is less vertically detailed.
- Trails and semantic impact marks are presented, but the renderer does not
  reproduce the reference's full particle/shake/ring/flash effect system.
- No release binaries are committed by this implementation step; the documented
  static build command produces one.

## Unresolved uncertainties

- The reference does not establish separate provenance for its WAV files or
  Apple outline. Neither was copied.
- Trademark permissions for campaign names and marks are not established by
  the included software licenses. They are retained descriptively for faithful
  behavior without an endorsement claim.
- Exact appearance and key-release reporting vary by terminal. Automated
  profiles and PTY coverage validate the selected fallbacks, not every terminal
  emulator.
