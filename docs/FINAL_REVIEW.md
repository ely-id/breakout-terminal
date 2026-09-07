VERDICT: PASS

Independent validation: 2026-09-06. No remaining P0 or P1 issue was found that prevents this from being a credible standalone port. The four requested specifications/reports were re-read, and the current implementation was inspected against the reference engine, level generator, and QML input/frame handling.

The previous ASCII failure is resolved in `internal/render/render.go`: `Renderer.compose` now derives brick glyphs from both half-cell materials. The former minimum-size invisible bricks remain visible. The new `internal/render/ascii_test.go` checks all 25 layouts at seven sizes and four color profiles (700 ASCII frames), plus the actual `L1-3-2` rectangle in normal, reinforced, damaged, steel, special, and destroyed states. These checks pass on the current source. As a negative control, the same regression was run against the old renderer through a temporary Go overlay; it reproduced the four material failures. Production files were not altered for that check.

This closes the prior P1-5 gap and supports the corrected build report. The change affects presentation only; world coordinates, collision rules, scoring, and timers remain in the framework-independent Game Core. The other prior P1 corrections—stable held-direction actions, capability migration, separate legacy one-shot guards, uniform letterboxing, and distinct capsule identities—are present and tested.

## Evidence

Validation used Go 1.27.1 on Linux 7.2.3 x86-64, AMD Ryzen 7 5800H.

- Fresh `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go build ./...`, and `go mod verify` passed.
- All four reference test programs passed with `node --test reference/plugin-source/tests/*.js`.
- Native and `CGO_ENABLED=0` Linux amd64/arm64 builds passed. `file` and `readelf -d` confirmed static release binaries without dynamic sections. Native and static amd64 binaries were executed; arm64 was inspected but not executed.
- Real PTY checks passed for attract/start, 40x20 ASCII brick visibility, paddle movement and lease expiry, launch, pause/resume, shrink to 39x19, recovery through 120x40 and back to 40x20, and Escape exit. Separate Ctrl+C, SIGINT, and SIGTERM checks restored alternate screen, cursor, and terminal attributes without panic. External SIGINT returned 1 after cleanup; the other exits returned 0. Non-TTY launch returned the expected diagnostic and status 2.
- PTY runs used an isolated state directory, no desktop-session variables, and a PATH containing no helper programs. Source/import inspection confirmed no required Omarchy, plugin-host, compositor, hardcoded user path, shell-command, or network integration. The launcher filters the transitive color detector's `TMUX` helper trigger. Game Core imports remain only `math`, `sort`, and `strings`; the two selected presentation dependencies remain pinned.
- The 36,000-frame simulated ten-minute soak passed its retained-heap and goroutine checks in 31.18 seconds. Three-sample renderer medians were 1.80 ms for a cached one-ball frame, 1.81 ms for 100 balls, and 2.09 ms for a static rebuild, below the 16.7 ms composition budget.

No additional gameplay, collision, timing, resize, or portability discrepancy warranting P0/P1 was found. The documented silent audio, simplified effects, standalone exit behavior, and terminal input approximations remain declared deviations. This validation does not establish behavior in every terminal emulator; no fresh vulnerability scan or syscall trace was performed.

## Remaining P2

- `internal/render/render.go`, `clone`/`Renderer.compose`: cached 120x75 frames still allocate approximately 1.26 MB across 17,000 allocations. Buffer reuse and less allocation during style composition would reduce GC pressure; current measured timing passes.
- `internal/app/model.go`, legacy input: the documented 140 ms direction lease cannot reproduce physical releases or uninterrupted movement across initial auto-repeat delay. The 800 ms initial/180 ms quiescence guards can suppress intentional identical presses, including repeated secret tokens. Enhanced event reporting avoids this ambiguity; broader terminal testing and clearer tuning guidance remain useful.

Only `docs/FINAL_REVIEW.md` was changed by this validation. Existing implementation changes and the reference tree were preserved.
