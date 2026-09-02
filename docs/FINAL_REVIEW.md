VERDICT: FAIL

The port is close, and no P0 issue was found, but one P1 presentation bug remains in a required operating mode.

## Remaining P1

### P1 — Minimum-size ASCII rendering can hide live bricks

`internal/render/render.go`, `Renderer.buildStatic` (the ASCII loop near line 259), emits a brick glyph only for terminal rows in `(y0 + 1) / 2 ... y1 / 2`. When projection at a short viewport places a brick wholly in an odd half-cell, that range is empty. `Renderer.compose` then renders the same cell as an ordinary `.` playfield cell even though the Game Core still includes the brick in collision detection.

A read-only overlay probe against the final source reproduced this at the documented playable minimum of 40x20. Four live level-1 bricks (`L1-3-2`, `L1-3-4`, `L1-3-7`, and `L1-3-16`) had no ASCII brick glyph. At 60x24 and 80x24, eight live level-1 bricks had no glyph. This makes collision-active geometry invisible and changes the recognizable level mosaic, so the ASCII fallback is not consistently playable as required by `docs/PORT_SPEC.md`.

Expected correction: collapse both projected half-cells into the ASCII cell and emit the appropriate brick symbol whenever either half has brick semantics. Add a regression over real campaign layouts at 40x20 (and other supported sizes) that detects live bricks/occupied regions represented only as playfield dots.

This also means the P1-5 resolution claim in `docs/BUILD_REPORT.md`—that brick cells have explicit ASCII shapes—is not fully supported. Existing golden tests cover four color profiles without enabling `Options.ASCII`; the ASCII semantic test uses oversized synthetic bricks at 120x75, so it does not exercise the failing projection case.

## Validation evidence

- `go test ./...`, `go test -race ./...`, and `go vet ./...` passed.
- The reference JavaScript suite passed: 4/4 tests with `node --test reference/plugin-source/tests/*.js`.
- Native build and `CGO_ENABLED=0` Linux amd64/arm64 cross-builds passed; the cross-built binaries were confirmed static ELF executables.
- A 10-minute simulated renderer soak passed. Cached-frame benchmarks remained within the 16.7 ms frame budget (about 2.5–3.5 ms per 120x75 frame on this machine).
- Real PTY smoke checks covered launch, input, shrink below 40x20, restoration to 120x40, pause, Escape, Ctrl+C, SIGINT, and SIGTERM. Alternate-screen, cursor, and termios state were restored in every case, with no panic.
- A non-TTY launch failed cleanly with the documented diagnostic and exit status 2.
- Source inspection found no production dependency on Omarchy, the plugin host, a compositor, hardcoded user paths, shell commands, external audio players, or runtime network access. The framework-independent Game Core and adapter/presentation separation remain intact, and no additional collision, scoring, timing, or state-transition discrepancy was found.

## Remaining P2 issues

- `Renderer.Render` still clones and composes the complete framebuffer each frame. Measurements remain around 1.26 MB and 17,000 allocations for a cached 120x75 frame. This passes the timing budget but is avoidable GC pressure.
- Legacy terminals cannot report releases. The documented one-shot guard can suppress an intentional identical action within 800 ms (or before 180 ms of post-repeat quiescence), and directional movement remains less faithful during the terminal's initial repeat delay.
- After fixing the P1, add true ASCII golden/property coverage at every required viewport; the current multi-size golden test exercises color profiles but not ASCII mode.
