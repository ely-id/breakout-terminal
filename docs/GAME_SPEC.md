# Breakout Gameplay and Presentation Specification

This document reverse engineers the Breakout implementation in
`reference/plugin-source/`. It describes the current source as found, not an
idealized Breakout design and not the architecture of the future terminal
port. No source under `reference/plugin-source/` was modified.

## Evidence and confidence notation

- **OBSERVED** means the behavior is directly implemented or asserted by the
  checked-in tests/artifacts.
- **INFERRED** means it follows from combining observed code paths but is not
  directly exercised by an assertion or live artifact.
- **UNKNOWN** means the reference does not determine the answer.

**OBSERVED:** The gameplay authority is `engine.js` plus the deterministic
campaign data/generator in `levels.js`. `renderer.js`, `effects.js`, and
`audio.js` define presentation. `Panel.qml` supplies the real keyboard mapping,
frame integration, persistence, sound playback, and host/window lifecycle.
The Node tests run the same JavaScript engine used by QML. Sources:
[`engine.js`, `createEngine`](../reference/plugin-source/engine.js#L4),
[`levels.js`, `createLevel`](../reference/plugin-source/levels.js#L413),
[`Panel.qml`, imports and `Component.onCompleted`](../reference/plugin-source/Panel.qml#L1),
[`test_campaign.js`, physics/campaign integration](../reference/plugin-source/tests/test_campaign.js#L13).

**OBSERVED:** The reference tree also contains the plugin manifest and bar
launcher, a Quickshell development harness, 15 WAV assets, a legacy preview,
14 live-test screenshots, 113 captured live state polls, CI configuration, and
four Node test programs plus the live integration script. These are
corroborating evidence; they do not
override the current engine/renderer when they disagree. Sources:
[`manifest.json`](../reference/plugin-source/manifest.json),
[`shell.qml`](../reference/plugin-source/shell.qml),
[`tests/live-test.sh`](../reference/plugin-source/tests/live-test.sh),
[`tests/artifacts/state-polls.jsonl`](../reference/plugin-source/tests/artifacts/state-polls.jsonl).

## What belongs to the game and what belongs to the host

| Classification | Behavior |
|---|---|
| **OBSERVED — intrinsic game** | State machine, 800×1000 logical coordinates, paddle/ball/brick rules, fixed-step simulation, campaign data, bosses, power-ups, secrets, scoring, lives, messages, high-score normalization, event generation. |
| **OBSERVED — presentation implementation** | QML Canvas drawing, two-canvas static/dynamic split, scaling to a window, particles, shake, brief boss hit-freeze, WAV cue routing, font sizes, colors, and glyphs. |
| **OBSERVED — plugin/framework lifecycle** | `manifest.json` entry points, bar button, `open`/`summon`/IPC methods, Quickshell `FloatingWindow`, focus handling, hiding through the shell, and the standalone QML test harness. |
| **OBSERVED — Omarchy/desktop integration** | Reading the active Omarchy theme, `omarchy-shell` commands in the README, and `hyprctl`-based floating/resizing/centering. |
| **OBSERVED — portable OS behavior** | XDG-state high-score and mute files are ordinary local JSON, although their I/O is currently performed with Quickshell types. |

Sources: [`manifest.json`, entry points](../reference/plugin-source/manifest.json#L9),
[`BarWidget.qml`](../reference/plugin-source/BarWidget.qml),
[`Panel.qml`, state files and theme](../reference/plugin-source/Panel.qml#L46),
[`Panel.qml`, window setup](../reference/plugin-source/Panel.qml#L382),
[`README.md`, opening and local data](../reference/plugin-source/README.md#L7).

## Entry points and lifecycle

**OBSERVED:** The installed plugin has two declared entry points:
`Panel.qml` for the game panel and `BarWidget.qml` for a single bar button. The
button displays the Nerd Font glyph `` and asks the host shell to toggle module
`breakout` with payload `{}`. The separate `shell.qml` is only a development
harness: it loads `Panel.qml` and calls `open("{}")`. Sources:
[`manifest.json`, `entryPoints`](../reference/plugin-source/manifest.json#L10),
[`BarWidget.qml`, `WidgetButton`](../reference/plugin-source/BarWidget.qml#L4),
[`shell.qml`, `ShellRoot`](../reference/plugin-source/shell.qml#L7).

**OBSERVED:** Constructing the panel creates the effects state, renderer, and
engine. The engine initially sits in `attract` mode, already holding a generated
level-1 board but no ball; the message is `PRESS ENTER`. This produces the
opening board behind the attract overlay. Sources:
[`Panel.qml`, `Component.onCompleted`](../reference/plugin-source/Panel.qml#L278),
[`engine.js`, initial state](../reference/plugin-source/engine.js#L22),
[`renderer.js`, `drawStatic` and `drawAttract`](../reference/plugin-source/renderer.js#L742).

**OBSERVED:** Opening with a valid level payload starts a completely fresh run
at that 1-based level. Accepted forms in the QML entry point are a number, an
all-digit string, a JSON number, a JSON object with `level`, or, after failed
JSON parsing, text matching `level\D+(\d+)`. The engine itself accepts only
finite integer levels 1–25. Invalid requests leave the current game unchanged
and simply show it. Sources: [`Panel.qml`, `parseRequestedLevel`, `showLevel`,
and `open`](../reference/plugin-source/Panel.qml#L297),
[`engine.js`, `startLevel`](../reference/plugin-source/engine.js#L155).

**OBSERVED:** Closing while in `serve`, `playing`, `levelClear`, or `lifeLost`
automatically pauses an unpaused game. Reopening resumes only a pause that the
close operation itself created. A manually paused game stays paused across a
close/reopen. Host hiding, focus restoration, and IPC are framework behavior,
not game rules. Sources: [`Panel.qml`, `pauseForClose`, `open`, `close`, and
`onVisibleChanged`](../reference/plugin-source/Panel.qml#L288).

## State model and transitions

**OBSERVED:** The primary modes and transitions are:

| Current state/action | Result |
|---|---|
| Initial `attract`; Enter or Space | Fresh level-1 run in `serve`, score 0, 3 lives, one attached ball. Space does **not** also launch on that press. |
| `attract`; explicit start-level 1–25 | Fresh requested level in `serve`. |
| `serve`; Space | Deterministic launch and `playing`. |
| `playing`; at least one ball drains but another remains | Continue `playing`; no life is lost. |
| `playing`; final ball drains | Lose one life and reset combo/drops/boss shots. With lives remaining, enter `lifeLost` for 1.1 s; otherwise enter `gameOver`. |
| `lifeLost`; 1.1 s elapses | Attach one ball and enter `serve`; show `SPACE TO SERVE` for 1.5 s. |
| `playing`; final breakable on levels 1–24 breaks | Award clear bonus, enter `levelClear` for 1.5 s, then load the next level in `serve`. |
| `playing`; level 25 boss breaks | Award clear bonus, record the run, enter `victory`. |
| `gameOver` or `victory`; Enter or Space | Fresh level-1 run in `serve`. |
| Any active non-end mode; P | Toggle an orthogonal `paused` flag. A pause freezes all engine time and transitions. |

Sources: [`engine.js`, `startLevel`, `launch`](../reference/plugin-source/engine.js#L155),
[`engine.js`, `finishRun`, `loseBall`, `completeLevel`](../reference/plugin-source/engine.js#L235),
[`engine.js`, `keyDown` and `togglePause`](../reference/plugin-source/engine.js#L498),
[`engine.js`, `tick`](../reference/plugin-source/engine.js#L877).

**OBSERVED:** `resetRun` resets score, lives, combo, pauses, secrets, input,
transition delays, total-brick telemetry, and boss attacks. Loading each level
also resets all normal timed power-ups, shield, sticky aim, drops, boss shots,
paddle position, and paddle target. It does **not** reset score/lives/combo or
the DHH timer during ordinary campaign progression. Sources:
[`engine.js`, `loadLevel` and `resetRun`](../reference/plugin-source/engine.js#L98).

**INFERRED:** A combo may technically survive the 1.5-second clear transition
because `loadLevel` does not reset it, but its 2.5-second timer continues to run
during `levelClear` and the following serve. DHH mode likewise can cross a
level boundary until its timer expires. Normal powers cannot cross a level
load. Source: [`engine.js`, `loadLevel`, `completeLevel`, and `tick`](../reference/plugin-source/engine.js#L98).

**OBSERVED:** Exact transient messages are part of the visible behavior:

- Level load: `LEVEL NN — NAME`, or `BOSS LEVEL NN — NAME`, for 2.4 s.
- Normal launch: `BREAK OUT` for 1.0 s.
- Life loss: `STAY IN THE FIGHT — N LEFT` for 1.1 s.
- New serve after loss: `SPACE TO SERVE` for 1.5 s.
- Clear: `LEVEL CLEAR — NAME` for 1.5 s.
- End-state engine messages: `GAME OVER` or `OMARCHY SHIPPED`, assigned the
  non-decrementing 9999-second sentinel and therefore held indefinitely.
- Boss phase: `BOSS PHASE S/3 — FIRE RATE UP` for 2.0 s.
- Boss defeat: `BOSS DOWN — NAME` for 2.0 s, normally superseded immediately
  by a level-clear/victory message when it is the last breakable.

Source: [`engine.js`, `setMessage` call sites](../reference/plugin-source/engine.js#L46).

## Logical board, coordinates, and resizing

**OBSERVED:** All gameplay uses a resolution-independent logical board of
width 800 and height 1000. Coordinates use the Canvas convention: origin at the
top-left, +x rightward, +y downward. Positions for balls and the paddle are
centers; brick positions are top-left corners. Sources:
[`engine.js`, constants and entity initialization](../reference/plugin-source/engine.js#L9),
[`renderer.js`, renderer dimensions](../reference/plugin-source/renderer.js#L3).

**OBSERVED:** Collision walls are x=20, x=780, and y=92. There is no bottom
wall. A ball remains alive while `ball.y - ball.r <= 1000` and is removed once
its top passes below the logical bottom. The drawn rounded playfield is
`(20,88)` with size `760×858`, ending at y=946; the slight top-wall/drawing
offset is intentional in the current code. Sources:
[`engine.js`, `updateBall`](../reference/plugin-source/engine.js#L602),
[`renderer.js`, `drawBackdrop`](../reference/plugin-source/renderer.js#L349).

**OBSERVED:** The Canvas renderer uniformly scales the full 800×1000 scene by
`min(actualWidth/800, actualHeight/1000)` and centers it, preserving the 0.8
aspect ratio. It does not adapt gameplay coordinates or layout to size.
Sources: [`renderer.js`, `fit`](../reference/plugin-source/renderer.js#L342).

**OBSERVED — host-specific:** The plugin attempts to make a centered floating
window at 80% of focused-monitor usable height, width 0.8×height, capped to 80%
of usable width. It queries and manipulates the window with `hyprctl`; failures
fall back to a 480×600 window/show path. This sizing is polish rather than a
portable gameplay dependency. Sources: [`Panel.qml`, window constants and
`configureGameWindow`](../reference/plugin-source/Panel.qml#L40).

**UNKNOWN:** The reference defines no terminal-grid resize policy, minimum
terminal dimensions, or behavior when character cells are not square. A port
must choose a projection while keeping the logical geometry/relationships.

## Timing and simulation

**OBSERVED:** Simulation is deterministic fixed-step integration at 240 Hz
(`STEP = 1/240`). `engine.step(dt)` rejects nonpositive/nonfinite time and all
time while paused, caps each submitted delta to 0.05 s, accumulates it, and runs
as many full fixed ticks as fit. Fractional remainder is retained. Source:
[`engine.js`, constants and `step`](../reference/plugin-source/engine.js#L9).

**OBSERVED:** QML drives this from `FrameAnimation`, caps `frameTime` to 0.05 s,
advances presentation effects, conditionally advances the engine, snapshots,
applies events/audio, and repaints the dynamic canvas. Rendering is therefore
display-frame-driven, but gameplay integration is not. Source:
[`Panel.qml`, `FrameAnimation`](../reference/plugin-source/Panel.qml#L621).

**OBSERVED:** Engine time, messages, combo, timed powers, boss face timers, and
transition delays all advance in fixed ticks. Timed effects count down even in
`serve`, `lifeLost`, and `levelClear`; only movement/balls/drops/boss behavior
are gated to `playing`. Pause freezes the entire engine, including these timers.
Source: [`engine.js`, `tick`](../reference/plugin-source/engine.js#L877).

**OBSERVED:** Ordinary brick collisions never freeze simulation. Non-dense boss
phase changes request 0.038 s of hit-freeze and boss defeat requests 0.065 s;
the QML loop realizes this by skipping engine steps while the presentation
freeze timer is active. At more than 12 balls, boss freezes are suppressed.
Sources: [`effects.js`, `applyEvents`](../reference/plugin-source/effects.js#L171),
[`Panel.qml`, `FrameAnimation`](../reference/plugin-source/Panel.qml#L621),
[`CHANGELOG.md`, 1.0.0`](../reference/plugin-source/CHANGELOG.md#L3).

## Paddle

**OBSERVED:** The paddle center is `(400,925)` on level load. Height is 18.
Base width is
`max(98, 150 - 2.25 * zeroBasedLevelIndex)`: 150 on level 1, 125.25 on level
12, 98.25 on level 24, and the 98 minimum on level 25. Sources:
[`engine.js`, constants, `levelPaddleWidth`, and `loadLevel`](../reference/plugin-source/engine.js#L9),
[`test_engine.js`, paddle-width check](../reference/plugin-source/tests/test_engine.js#L335).

**OBSERVED:** Held left/right input produces constant center velocity −660 or
+660 logical units/s; both held cancel. Movement clamps the center to
`[20 + width/2, 780 - width/2]`. A test/autopilot target can synthesize the
same left/right movement when no physical direction is held, stopping within 3
units, but that target is not a normal user control. Source:
[`engine.js`, `updatePaddle` and `setPaddleTarget`](../reference/plugin-source/engine.js#L529).

**OBSERVED:** On a descending-ball collision, the ball is placed 0.1 units
above the paddle. Let `offset = clamp((ball.x-paddle.x)/(paddle.w/2), -1, 1)`.
With no sticky effect, `angle = 1.05*offset` radians, speed is the greater of
current magnitude and target speed, then:

```text
vx = sin(angle) * speed + paddle.vx * 0.13
vy = -abs(cos(angle) * speed)
```

The paddle hit resets combo to 1 and emits a paddle event. Source:
[`engine.js`, paddle branch in `updateBall`](../reference/plugin-source/engine.js#L615).

## Ball model, speed, and world collision

**OBSERVED:** A ball has center `(x,y)`, velocity `(vx,vy)`, radius 9, up to six
trail samples, sticky state/offset, and a set of current piercing contacts. A
serve ball sits at `(paddle.x, 905)`, derived as paddle top minus radius minus
2. Source: [`engine.js`, `makeBall`, `attachBall`, and `updatePaddle`](../reference/plugin-source/engine.js#L87).

**OBSERVED:** A normal serve is deterministic. Speed is the level's launch
speed; horizontal direction is left when `(zeroBasedLevelIndex + lives)` is
even and right otherwise. Initial components are `vx = direction*speed*0.34`,
`vy = -speed*0.94`. A fresh three-life run therefore alternates right on odd
level numbers and left on even level numbers. Source:
[`engine.js`, `launch`](../reference/plugin-source/engine.js#L169).

**OBSERVED:** Target ball speed is:

```text
cleared = 1 - bricksLeft / initialBreakableCount
target  = levelLaunchSpeed * (1 + clamp(cleared,0,1)*0.22)
target *= 0.72 while slow is active
```

After movement/collision, actual speed is clamped to 88%–122% of target rather
than forced exactly to target. If `abs(vy)` is less than 28% of the chosen
speed, vertical velocity is raised to that minimum and horizontal velocity is
recomputed to preserve magnitude. This prevents near-horizontal lockups.
Source: [`engine.js`, `targetBallSpeed` and `normalizeBallSpeed`](../reference/plugin-source/engine.js#L582).

**OBSERVED:** Per fixed tick, collision order is: integrate position; left,
right, and top walls; descending paddle; bricks; speed normalization; trail;
one-use bottom shield; bottom-drain test. Wall response clamps the ball just
inside and forces the corresponding velocity sign. Source:
[`engine.js`, `updateBall`](../reference/plugin-source/engine.js#L602).

**OBSERVED:** Circle-vs-rectangle overlap uses the closest point on the
rectangle. Rectangle bounce first uses the previous center to identify a clean
top/bottom/left/right entry and separates by 0.1 units. If no side is provable,
it computes four penetration distances and flips x for a left/right minimum,
otherwise y. Only the first overlapping non-pierced brick in generated order
is processed in a tick. Sources: [`engine.js`, `circleHitsRect`,
`bounceFromRect`, and brick loop](../reference/plugin-source/engine.js#L554).

**OBSERVED:** Bricks are indexed into 64-unit vertical buckets, expanded by the
ball radius. This changes candidate lookup performance, not collision rules.
Bosses move only horizontally and retain their initial vertical bucket.
Source: [`engine.js`, `rebuildBrickBuckets`](../reference/plugin-source/engine.js#L71).

**OBSERVED:** In multiball, each ball updates independently in reverse array
order. Draining a ball simply removes it; a life is lost only after the final
ball is gone. If a hit clears the level, remaining balls stop updating because
mode immediately leaves `playing`. Sources:
[`engine.js`, `updateBalls`](../reference/plugin-source/engine.js#L669),
[`test_engine.js`, multiball drain check](../reference/plugin-source/tests/test_engine.js#L585).

## Bricks, deterministic generation, and exact layouts

**OBSERVED:** Levels 1–24 derive brick geometry from `#` cells in the matrices
below. For a matrix with `cols` columns:

```text
gap = 4
brickW = floor((734 - (cols-1)*gap) / cols)
brickH = 20
layoutWidth = cols*brickW + (cols-1)*gap
originX = floor((800-layoutWidth)/2)
originY = 154
x = originX + col*(brickW+4)
y = 154 + row*(20+6)
```

Thus 19-column levels use 34×20 bricks and origin x=41; the 20-column Turbo
level uses 32×20/origin x=42; the 18-column Omarchy level uses 37×20/origin
x=33. Sources: [`levels.js`, `createLevel`](../reference/plugin-source/levels.js#L413),
[`test_engine.js`, layout/Omarchy checks](../reference/plugin-source/tests/test_engine.js#L32).

**OBSERVED:** Brick IDs are `L<level>-<row>-<col>`. Palette alternates by
`colors[(row+col) % colors.length]`. Normal HP is 1. Starting with zero-based
level index 5, a non-steel brick has 2 HP when the deterministic hash below is
less than `min(38, 8+index)` percent:

```text
hash(n):
  n = imul(n xor (n unsigned-shift-right 16), 0x45d9f3b)
  n = imul(n xor (n unsigned-shift-right 16), 0x45d9f3b)
  return unsigned(n xor (n unsigned-shift-right 16))

tough when hash(index*2000 + row*71 + col*13) % 100 < min(38,8+index)
```

Source: [`levels.js`, `hash` and `createLevel`](../reference/plugin-source/levels.js#L359).

**OBSERVED:** Steel bricks have HP −1, never break, and do not count in
`bricksLeft`. Actual occupied steel cells occur only on KAMAL (`container`:
occupied first/last-row cells with col divisible by 3, six cells) and RUNNING
IN PRODUCTION (`(row+col)%9==0`, nine cells). Rules also exist for `rails` and
`finale`, but their matching coordinates are unoccupied, so those boards have
zero steel bricks. Source: [`levels.js`, `steelAt` and `createLevel`](../reference/plugin-source/levels.js#L369).

**OBSERVED:** Every non-steel cell independently carries a deterministic 12%
power-up test:

```text
eligible when hash(index*5000 + row*97 + col*17) % 100 < 12
kind = [wide,slow,life,multi,fire,sticky,shield,magnet]
       [hash(index + row*11 + col*7) % 8]
```

The capsule is hidden until the brick breaks. Source:
[`levels.js`, power assignment in `createLevel`](../reference/plugin-source/levels.js#L440).

### Campaign table

`Cells` is the number of occupied regular matrix cells. `1HP/2HP/steel` gives
their actual generated durability. `Clear` is the breakable count including a
boss where present.

| Lv | Name — subtitle | Speed | Colors | Cells (1/2/steel) | Clear | Boss |
|---:|---|---:|---|---:|---:|---|
| 1 | THE BOOT — Everything starts at the prompt | 560 | accent/blue | 77 (77/0/0) | 77 | — |
| 2 | ARCH BTW — A rolling release of bricks | 578 | blue/cyan | 78 (78/0/0) | 78 | — |
| 3 | TOKYO NIGHT — The default after dark | 596 | magenta/blue | 91 (91/0/0) | 91 | — |
| 4 | DHH'S DESK — Ship the whole stack | 614 | red/yellow | 64 (64/0/0) | 64 | — |
| 5 | RYAN'S RUN — Keep the shell moving | 632 | green/cyan | 54 (54/0/0) | 54 | — |
| 6 | BASECAMP HILL — Clarity over chaos | 650 | green/yellow | 68 (61/7/0) | 69 | SCOPE CREEP, 6 HP |
| 7 | SIGNAL 37 — Can you read the signal? | 678 | yellow/red | 54 (44/10/0) | 54 | — |
| 8 | CAMPFIRE — Gather around the glow | 696 | red/yellow | 67 (60/7/0) | 67 | — |
| 9 | HEY — Email deserves better | 714 | yellow/green | 57 (48/9/0) | 57 | — |
| 10 | WRITEBOOK — A book is just a stack | 732 | cyan/blue | 65 (53/12/0) | 65 | — |
| 11 | RAILS — Convention over configuration | 750 | red/bright | 55 (49/6/0) | 55 | — |
| 12 | HOTWIRE — HTML over the wire | 768 | magenta/yellow | 40 (26/14/0) | 41 | FULL RELOAD, 8 HP |
| 13 | TURBO — Navigate without slowing down | 796 | green/cyan | 73 (60/13/0) | 73 | — |
| 14 | STIMULUS — A little JavaScript goes far | 814 | blue/yellow | 45 (37/8/0) | 45 | — |
| 15 | KAMAL — Deploy anywhere | 832 | blue/cyan | 116 (87/23/6) | 110 | — |
| 16 | SOLID CABLE — Realtime, simplified | 850 | green/magenta | 53 (34/19/0) | 53 | — |
| 17 | ONCE — Buy it once, own it forever | 868 | yellow/bright | 44 (35/9/0) | 44 | — |
| 18 | REWORK — Underdo the competition | 886 | red/bright | 72 (56/16/0) | 73 | MEETING MAZE, 10 HP |
| 19 | SHAPE UP — Appetite, pitch, bet, build | 914 | magenta/cyan | 38 (30/8/0) | 38 | — |
| 20 | NO MEETINGS — Protect the maker's schedule | 932 | cyan/green | 77 (57/20/0) | 77 | — |
| 21 | SMALL TEAMS — Constraints make the game | 950 | yellow/blue | 96 (72/24/0) | 96 | — |
| 22 | RUNNING IN PRODUCTION — The test is the real thing | 968 | green/red | 92 (56/27/9) | 83 | — |
| 23 | OMARCHY — Beautiful. Modern. Opinionated. | 986 | accent/magenta | 98 (69/29/0) | 98 | — |
| 24 | ONE PERSON FRAMEWORK — You have everything you need | 1020 | red/yellow | 71 (59/12/0) | 72 | THE MONOLITH, 12 HP |
| 25 | ONE MORE THING — A bite after the framework | 1040 | red/green | no matrix | 1 | THE BITE, 16 HP |

Source: [`levels.js`, `LEVELS`, `BOSS_DEFS`, and `createLevel`](../reference/plugin-source/levels.js#L4).

### Exact occupied-cell matrices

**OBSERVED:** `#` creates a cell and `.` leaves it empty. These are the exact
current symbolic layouts; their motif descriptions are, respectively: terminal
monitor/prompt, Arch mountain, torii/moon/skyline, desk, runner, camp hill,
radio signal 37, campfire, envelope, open book, rails, live wire, turbine,
activation network, cargo ship, cable/sockets, coin, wrench cycle, stepped
shapes, crossed-out calendar, teammates, factory, official Quattro icon, and
one person inside a framework. Level 25 has no matrix. Source:
[`levels.js`, `LEVEL_ART`](../reference/plugin-source/levels.js#L49).

```text
01 THE BOOT
..###############..
..#.............#..
..#..##.........#..
..#.#..#........#..
..#..##...#####.#..
..#.............#..
..#.............#..
..###############..
........#.#........
.....#########.....
...#############...

02 ARCH BTW
.........#.........
........###........
.......#####.......
......###.###......
.....###...###.....
....###.....###....
...###.......###...
..#####.....#####..
.######.....######.
######.......######
....###########....

03 TOKYO NIGHT
..............###..
.............#####.
..###############..
...#############...
......#.....#......
......#.....#......
....###.....###....
....#.........#....
.##.#..##.###..##..
.##.#######.#####..
###################

04 DHH'S DESK
....###########....
....#.........#....
....#..#####..#.##.
....#.........#.##.
....###########....
.........#.........
.......#####.......
...................
.#################.
...#...........#...
...#...........#...

05 RYAN'S RUN
........###........
........###....###.
.........#..###....
......######.......
....###..#..###....
.........#.........
........###........
.......##.##.......
......##...##......
###..##.....##.....
#####.........####.

06 BASECAMP HILL
.........#.........
.........#####.....
.........#.........
........###........
.......#####.......
......###.###......
.....###...###.....
....###.....###....
...###...#...###...
..###...###...###..
###################

07 SIGNAL 37
#.................#
.##.....###.....##.
...##..#####..##...
.....##..#..##.....
........###........
.........#.........
.......#####.......
......#..#..#......
.........#.........
......#######......
....###########....

08 CAMPFIRE
.........#.........
........###........
......##.##........
.......#####.......
.....####.####.....
.....#########.....
......#######......
.......#####.......
...####.....####...
.....#########.....
...####.....####...

09 HEY
.#############..#..
.#...........#..#..
.##.........##..#..
.#.##.....##.#..#..
.#...##.##...#..#..
.#.....#.....#..#..
.#...........#..#..
.#############.....
...................
................#..
...................

10 WRITEBOOK
.........#.........
..######.#.######..
.#......###......#.
.#.......#.......#.
.#..###..#..###..#.
.#.......#.......#.
.#..####.#.####..#.
.#.......#.......#.
.#.......#.......#.
..######.#.######..
.........#.........

11 RAILS
.......#...#.......
.......#...#.......
......#.....#......
.....#########.....
.....#.......#.....
....#.........#....
...#############...
...#...........#...
..#.............#..
.#################.
#.................#

12 HOTWIRE
###................
..#................
..########.........
.........#...#.....
.........#..##.....
.........####......
.......#####.......
.........####......
..........##..#....
..........#...#....
..........#####....

13 TURBO
........#####.......
......###...###.....
.....##.......##....
....##...###...##...
....#...##.##...#...
....#...##.##...#...
....##...###...##...
.....##.......##..#.
......####.####..##.
..........##...####.
###...###......#####

14 STIMULUS
..........##.......
.........##........
........##.........
.......#####.......
.........##........
........##.........
.......##..........
..##...#...##...##.
.####....####.####.
..##.......##...##.
...#........#....#.

15 KAMAL
..###.###.###.###..
..#.#.#.#.#.#.#.#..
..###.###.###.###..
....###.###.###....
....#.#.#.#.#.#....
....###.###.###....
.#################.
..###############..
...#############...
....###########....
##...............##

16 SOLID CABLE
###.............###
#.#.............#.#
###.............###
..#.............#..
..##...........##..
...##.........##...
....###########....
...................
..###.........###..
..#.#.........#.#..
..###.........###..

17 ONCE
.......#####.......
.....##.....##.....
....#....#....#....
...#...###.....#...
...#.....#.....#...
...#.....#.....#...
...#.....#.....#...
...#...#####...#...
....#.........#....
.....##.....##.....
.......#####.......

18 REWORK
.....########......
...###.....#####...
..##.........####..
.##.............###
##......###......##
##.....#####.....##
.##.....###.....##.
..####.........##..
...#####.....###...
......########.....
...................

19 SHAPE UP
..............#....
.............###...
............#####..
...........#######.
......#####........
......#...#........
......#...#........
......#####........
.###...............
#...#..............
.###...............

20 NO MEETINGS
....##.......##....
..###############..
..#.............#..
..#..##.....##..#..
..#...##...##...#..
..#....##.##....#..
..#.....###.....#..
..#....##.##....#..
..#...##...##...#..
..#..##.....##..#..
..###############..

21 SMALL TEAMS
..##.....##.....##.
.####...####...####
..##.....##.....##.
.####...####...####
#.##.#.#.##.#.#.##.
...#.....#.....#...
...#############...
...#.....#.....#...
..###...###...###..
.#####.#####.#####.
..#.#...#.#...#.#..

22 RUNNING IN PRODUCTION
....##.............
...####....##......
...####...####.....
...####...####.....
..###############..
..#..###..###...#..
..#..#.#..#.#...#..
..#..###..###...#..
..#.............#..
..###############..
###.##.##.##.##.###

23 OMARCHY
##################
#.......##.......#
#..#######...##..#
#..#..........#..#
#..#..........#..#
#..#..........#..#
####..........#..#
#..#..........#..#
#..#..........#..#
#..#..........#..#
#..############..#
#.......##.......#
##########.#######

24 ONE PERSON FRAMEWORK
###################
#.................#
#........#........#
#.......###.......#
#........#........#
#......#####......#
#........#........#
#.......#.#.......#
#......#...#......#
#.................#
###################
```

**OBSERVED:** Special printed bricks are DHH at level 4 row 0 col 9, RYAN at
level 5 row 0 col 9, OMARCHY at level 23 row 10 col 9, and DHH at level 24 row
5 col 9. On level 7, every occupied non-steel cell with `(row+col)%5==0` is
marked `37`. Destroying DHH/RYAN/OMARCHY displays flavor text only; destroying
a `37` also adds 270 to that brick's base score and sets the Signal 37 secret
flag. The exact messages are `DHH: SHIP IT`,
`RYAN: KEEP THE SHELL FAST`, `BEAUTIFUL MOTIVATIONS`, and
`SIGNAL RECEIVED: 37`, each for 2.2 s. Sources:
[`levels.js`, level definitions and special assignment](../reference/plugin-source/levels.js#L4),
[`engine.js`, `specialBrickMessage` and `hitBrick`](../reference/plugin-source/engine.js#L356).

## Brick damage, combo, and scoring

**OBSERVED:** Steel emits `steel`, does not lose HP, does not score, and always
bounces a normal or fire ball. A 2-HP reinforced brick's first hit subtracts
one HP and awards `25 × currentMultiplier`; it does not increment combo. Its
second hit uses the normal break path. Source:
[`engine.js`, `hitBrick`](../reference/plugin-source/engine.js#L384).

**OBSERVED:** Breaking any breakable first decrements `bricksLeft`, increments
combo (maximum 7), resets its timeout to 2.5 s, then awards points. This means
the first broken brick of a fresh combo is scored at ×2, not ×1. A normal
regular brick's base is `100 + zeroBasedLevelIndex*12`; a `37` brick adds 270.
A nonfatal boss hit has base `90 + levelNumber*8`; a boss's fatal hit has base
`900 + levelNumber*45`. Source: [`engine.js`, `currentMultiplier`, `addScore`,
and `hitBrick`](../reference/plugin-source/engine.js#L56).

**OBSERVED:** The visible/scoring multiplier is
`min(9, combo + (DHH active ? 2 : 0))`. Combo resets to 1 on an ordinary paddle
bounce, final-ball loss, timeout, or new run. Sticky catches return early and
do not reset combo. Source: [`engine.js`, `currentMultiplier`, paddle collision,
`loseBall`, and `tick`](../reference/plugin-source/engine.js#L56).

**OBSERVED:** Clearing a level awards base
`1000 + levelNumber*250`, multiplied by the current multiplier after the final
brick has already incremented combo. All scoring rounds the multiplied result,
though current bases/multipliers are integers. Source:
[`engine.js`, `completeLevel` and `addScore`](../reference/plugin-source/engine.js#L60).

## Lives, run completion, and high scores

**OBSERVED:** A fresh run has 3 lives. The life capsule adds one up to a maximum
of 7. A life is lost only on the final ball drain or a boss projectile hitting
the paddle. Life loss clears drops and boss shots, but ordinary timed powers
remain active and continue counting down through the 1.1-second delay. Sources:
[`engine.js`, initial/reset state, `loseBall`, and `activatePower`](../reference/plugin-source/engine.js#L22).

**OBSERVED:** Finishing records `{score, level, completed}` where `completed`
is true only for victory. Entries sort descending by score and are capped at
10. Loaded data rejects negative/nonfinite scores and nonfinite levels, floors
numbers, clamps level to 1–25, and preserves only a strict boolean `completed`.
The persisted envelope is `{version:1, highScores:[...], lastScore:N}`;
`lastScore` is floored and clamped to at least zero when finite. The attract
screen renders only the top five. Sources:
[`engine.js`, high-score functions and `finishRun`](../reference/plugin-source/engine.js#L209),
[`renderer.js`, `drawAttract`](../reference/plugin-source/renderer.js#L742).

**OBSERVED:** Persistence uses
`$XDG_STATE_HOME/breakout/highscores.json` or
`$HOME/.local/state/breakout/highscores.json`; mute uses `settings.json` in the
same directory. High scores save on `highScoreUpdated`; corrupt/missing files
are ignored. This is panel I/O, not part of engine physics. Source:
[`Panel.qml`, state path, file views, and `saveHighScores`](../reference/plugin-source/Panel.qml#L46).

## Power-ups

**OBSERVED:** Breaking a powered brick creates a capsule at the brick center,
initial velocity `(0,155)`. It is caught when its y is between 905 and 945 and
its x is within `paddle.w/2 + 12` of paddle center; otherwise it is removed past
y=1020. Capsules can drift horizontally only under magnet behavior. Sources:
[`engine.js`, drop creation and `updateDrops`](../reference/plugin-source/engine.js#L418).

| Kind / glyph | Exact effect |
|---|---|
| **OBSERVED — wide / `↔`** | Set 15 s; paddle width becomes `max(190, baseWidth+62)`. On expiry it returns to base width. |
| **OBSERVED — slow / `S`** | Set 12 s; immediately multiply every ball velocity component by 0.72. Target speed is also ×0.72 while active. Repeated pickups apply the immediate factor again and refresh the timer. |
| **OBSERVED — life / `+`** | Add one life, capped at 7; no timer. |
| **OBSERVED — multi / `×3`** | Add up to two balls, with total capped at 9. Repeated pickups therefore produce 3, 5, 7, then 9 balls. |
| **OBSERVED — fire / `F`** | Set 11 s. Balls pierce non-steel bricks without reversing; a brick is damaged only once per continuous contact. Steel still bounces. |
| **OBSERVED — sticky / `G`** | Set 16 s and reset aim to 0. Descending balls stick to the paddle instead of bouncing. Arrows aim; Space releases; A/D still move. Expiry automatically releases held balls. |
| **OBSERVED — shield / `▰`** | Set exactly one charge (not additive). A missed descending ball crossing y=944 bounces upward and consumes the charge. Boss shots ignore it. |
| **OBSERVED — magnet / `U`** | Set 14 s. Each capsule eases its vx toward `clamp((paddle.x-drop.x)*3.2,-400,400)` with response `min(1,dt*7)`; without magnet vx damps toward zero at 4/s. |

Sources: [`engine.js`, `activatePower`, `splitMultiball`, ball/drop logic](../reference/plugin-source/engine.js#L296),
[`renderer.js`, `drawDrops`](../reference/plugin-source/renderer.js#L671).

**OBSERVED:** Every pickup emits `powerUp` and shows a 2.0-second message, in
table order: `WIDE MODE — LESS, BUT BIGGER`, `TAKE IT SLOW`,
`ONE MORE PERSON`, `THREE-BALL SHIPPING`,
`FIREBALL — PIERCE THE LAYERS`, `STICKY — CATCH, AIM, RELEASE`,
`BOTTOM SHIELD — ONE SAVE`, and `MAGNET — CAPSULES COME TO YOU`. A shield
interception additionally shows `SHIELD SAVED THE RUN` for 1.4 s. Source:
[`engine.js`, `activatePower` and shield branch](../reference/plugin-source/engine.js#L296).

**OBSERVED:** Multiball chooses the first non-stuck ball if possible, otherwise
the first ball. Each new clone starts ±4 x and −2 y from it, at heading ±0.34
radians from the source, with speed `max(targetSpeed, sourceSpeed)`. If the
source is stuck, clones also stick with offsets shifted ±26 and clamped to the
paddle. Source: [`engine.js`, `splitMultiball`](../reference/plugin-source/engine.js#L331).

**OBSERVED:** Sticky release uses current target speed. With `n` held balls,
release number `i` adds angular spread
`(i-(n-1)/2)*0.11` to the shared aim; angle clamps to ±1.12 radians, with
`vx=sin(angle)*speed` and upward `vy=-abs(cos(angle)*speed)`. Manual aim changes
at 1.9 rad/s and clamps to ±1.05. Sources:
[`engine.js`, `releaseStickyBalls` and `updatePaddle`](../reference/plugin-source/engine.js#L189).

## Bosses and level-25 arena

**OBSERVED:** Levels 6, 12, 18, and 24 append one boss below the matrix at
y=464. Level 25 consists solely of a boss at `(290,236)`. Exact chassis:

| Level | ID/name | Kind | Initial x,y | Size | HP |
|---:|---|---|---|---|---:|
| 6 | SCOPE CREEP | creep | 266,464 | 268×64 | 6 |
| 12 | FULL RELOAD | reload | 278,464 | 244×60 | 8 |
| 18 | MEETING MAZE | maze | 272,464 | 256×62 | 10 |
| 24 | THE MONOLITH | monolith | 302,464 | 196×74 | 12 |
| 25 | THE BITE | apple | 290,236 | 220×198 | 16 |

Source: [`levels.js`, `BOSS_DEFS`, `createArenaLevel`, and boss append`](../reference/plugin-source/levels.js#L377).

**OBSERVED:** All bosses have three stages. Stage is 1 while HP is greater than
two-thirds maximum, 2 while greater than one-third, and 3 otherwise. Crossing a
threshold clears current projectiles, sets the next-shot delay to 0.55 s,
displays the phase message, and emits `bossStage`. Damage leaves chassis size
and palette unchanged. Sources: [`engine.js`, `bossStageFor`, `enterBossStage`,
and `hitBrick`](../reference/plugin-source/engine.js#L366),
[`test_engine.js`, boss phase check](../reference/plugin-source/tests/test_engine.js#L145).

**OBSERVED:** A live boss patrols horizontally between x=44 and
`756-boss.w`. Patrol desired speed is
`52 + stage*9 + levelIndex*0.8`; acceleration is 380 units/s². It predicts an
upward ball (`vy < -50`) approaching from below within 1.35 s, including side
wall reflections. If the predicted x falls within 54+radius units of its body,
it accelerates away at desired speed
`158 + stage*20 + levelIndex*1.5`, acceleration 920. Sources:
[`engine.js`, `reflectedBallX` and `updateBossMovement`](../reference/plugin-source/engine.js#L731).

**OBSERVED:** On level load a boss has a 1.25-second shot cooldown; after a life
loss it restarts at 1.0 s. Shot intervals by stage are 2.18, 1.62, and 1.12 s.
The first boss always keeps at most one live orb; later bosses fire 1, 2, or 3
simultaneous shots by stage, with 12 total shots as the cap. Sources:
[`engine.js`, boss cooldown setup, `bossFireInterval`, `bossVolleyCount`, and
`updateBossAttack`](../reference/plugin-source/engine.js#L98).

**OBSERVED:** A volley originates 8 units above the boss bottom. Vertical speed
is `170 + stage*18 + levelIndex*1.2`. Origins are spaced 46 units around boss
center. They aim at the paddle position at firing time, with target spacing 76
in stage 2 and 64 otherwise. Orbs have radius 6, bounce from side walls, pass
through the bottom, and cost exactly one life on paddle overlap. Sources:
[`engine.js`, `fireBossShots` and `updateBossAttack`](../reference/plugin-source/engine.js#L794).

**OBSERVED:** Boss face state is `ouch` for 0.72 s after damage, `shoot` for
0.34 s after a volley, look-left/right while dodging, a short blink during the
last 0.12 s of each 2.6-second cycle, otherwise `grin` in stage 3 and `angry` in
stages 1–2. Source: [`engine.js`, `bossFaceName`](../reference/plugin-source/engine.js#L717).

**OBSERVED:** Level 25's THE BITE is a large rainbow bitten-apple boss and has
deterministic capsule rain. First rain is due after 1.35 playing seconds; later
cooldown is 6.5 s. A new rain capsule waits until no other drop is live. At most
four spawn, in order shield, multi, fire, slow, from x slots 86, 168, 632, 714,
y=118, vy=140. Sources: [`levels.js`, level-25 definition](../reference/plugin-source/levels.js#L38),
[`engine.js`, `updatePowerRain`](../reference/plugin-source/engine.js#L861).

## Secrets

**OBSERVED:** Letter input is uppercased, stripped to A–Z, and appended to a
12-character rolling buffer:

- Suffix `DHH`: 37 s of DHH mode, permanent run flag, +2 to multiplier (still
  capped at 9), yellow paddle/balls, message
  `DHH MODE — RUNNING IN PRODUCTION` for 3.7 s.
- Suffix `BASECAMP`: permanent run flag, 37 s on the shared wide timer, paddle
  width `max(222, baseWidth+72)`, message
  `BASECAMP MODE — CLARITY WINS` for 3.7 s.

Source: [`engine.js`, `triggerSecret` and `inputCharacter`](../reference/plugin-source/engine.js#L431).

**OBSERVED:** The exact Konami token sequence is
Up, Up, Down, Down, Left, Right, Left, Right, B, A, Enter. A successful sequence
replaces all balls with exactly 100, sets mode `playing`, unpauses, and marks the
secret. Their common speed is launch speed×1.06. Angles linearly span
−0.86π..−0.14π; x receives a repeating nine-value ±7.2-unit stagger and y a
five-value 0..−8 stagger. It can start a new game from attract/end or skip a
`lifeLost` delay, but refuses activation in `levelClear`. Source:
[`engine.js`, `KONAMI_CODE`, `spawnBallStorm`, and `inputCodeToken`](../reference/plugin-source/engine.js#L19).

**OBSERVED:** Breaking any marked `37` brick sets `secrets.signal37` and shows
`SIGNAL RECEIVED: 37`. Special DHH text bricks do not activate typed DHH mode.
Source: [`engine.js`, `specialBrickMessage`](../reference/plugin-source/engine.js#L356).

**OBSERVED:** There is no runtime random-number use in gameplay. Campaign HP,
power placement, serves, effects debris, rain, and cheats are all deterministic.
The effects module uses a sine-based deterministic pseudo-random function keyed
by event sequence/brick identity. Sources: [`levels.js`, `hash`](../reference/plugin-source/levels.js#L359),
[`effects.js`, `unit`](../reference/plugin-source/effects.js#L40).

## Keyboard and pointer controls

**OBSERVED:** QML ignores auto-repeat press/release events; held movement works
because the initial press sets a boolean until release.

| Physical input | Current behavior |
|---|---|
| Left Arrow | Feed `left` to Konami; move left, except while any sticky ball is held, when it aims left without moving. |
| Right Arrow | Feed `right` to Konami; move/aim right under the same rule. |
| A / D | Move left/right even while aiming. A also feeds Konami `a`; all letters also feed the typed-secret buffer. |
| Up / Down | Feed only the Konami buffer; no gameplay movement. |
| B | Feed Konami `b` and the typed-secret buffer; no other action. |
| Space | From attract/end, start a fresh run; from serve, launch; with held sticky balls, release them. |
| Enter/Return | Feed Konami `enter`; if it completes the code, do nothing else. Otherwise start/restart only from attract/game-over/victory. |
| P | Toggle pause in active modes. It also enters `P` in the typed-secret buffer. |
| F10 | Toggle persistent sound mute. |
| Escape | Dismiss/close via the host, safely pausing active play. |
| Click bottom-right 230×42 px footer area | Toggle sound mute and restore keyboard focus. |

Sources: [`Panel.qml`, `Keys.onPressed`, `Keys.onReleased`, and `MouseArea`](../reference/plugin-source/Panel.qml#L536),
[`engine.js`, input functions](../reference/plugin-source/engine.js#L476),
[`README.md`, controls](../reference/plugin-source/README.md#L23).

**OBSERVED:** There is no normal mouse/touch paddle control. `setPaddleTarget`
and the `testPower`/`testBall`/`testDrop` IPC calls are test/autopilot surfaces;
the latter are disabled when running under the real plugin host. Sources:
[`engine.js`, test surface](../reference/plugin-source/engine.js#L1014),
[`Panel.qml`, IPC test methods](../reference/plugin-source/Panel.qml#L647).

## Visual design

**OBSERVED:** The fallback palette is Tokyo-Night-like:

| Role | Hex | Role | Hex |
|---|---|---|---|
| background / brick edge | `#1a1b26` | foreground / bright | `#c0caf5` |
| accent / blue | `#7aa2f7` | cyan | `#7dcfff` |
| green | `#9ece6a` | yellow | `#e0af68` |
| red | `#f7768e` | magenta | `#bb9af7` |
| muted | `#414868` | muted text | `#7982a9` |
| surface | `#24283b` | surface 2 | `#16161e` |
| playfield | `#101119` | border | `#3b4261` |
| steel | `#565f89` | overlay background | `#151515` |
| overlay text | `#f5f5f0` | overlay muted | `#b8b8b2` |

Source: [`Panel.qml`, `fallbackTheme`](../reference/plugin-source/Panel.qml#L52).

**OBSERVED — Omarchy-specific:** If available, a small TOML parser reads
`$XDG_STATE_HOME/omarchy/current/theme/colors.toml` and overrides named roles.
Overlay colors remain the fallback values. A standalone implementation must not
depend on this file; the fallback palette is a complete design. Source:
[`Panel.qml`, `parseColorsToml`, `rebuildTheme`, and `colorsFile`](../reference/plugin-source/Panel.qml#L65).

**OBSERVED:** The scene uses monospace text throughout. The outer background
has a 14-unit inset gradient, low-alpha horizontal lines every 24 units, and the
rounded dark playfield. Header: centered `BREAKOUT` at y=44; during play, score
as eight zero-padded digits at left y=72, centered
`LEVEL NN/25  NAME`, and one `● ` per life at right. Sources:
[`renderer.js`, `drawBackdrop` and `drawHeader`](../reference/plugin-source/renderer.js#L349).

**OBSERVED:** Bricks are rounded 5-unit blocks with a top highlight and dark
edge. A damaged 2-HP brick is drawn at 64% opacity. Steel uses the steel color
and a dark horizontal slot. Special text is centered inside its brick. Powered
bricks have no pre-break marker. Source:
[`renderer.js`, `drawBricks`](../reference/plugin-source/renderer.js#L395).

**OBSERVED:** The paddle is a rounded pill, normally accent-colored; sticky,
magnet, and DHH recolor it magenta, cyan, and yellow in that priority. Balls are
foreground circles with a six-sample fading trail, velocity-aligned subtle
stretch, impact squash, and red/yellow fire aura. At more than five balls they
are drawn as a batched path; dense scenes reduce trails/echoes. Source:
[`renderer.js`, `drawPaddleAndBalls`](../reference/plugin-source/renderer.js#L476).

**OBSERVED:** The shield is a pulsing cyan curved line near y=944. Magnet draws
faint curves from paddle to a subset of drops. Sticky-held balls draw a magenta
104-unit aim guide. Boss shots are colored magenta/yellow/red by stage with a
bright core. Source: [`renderer.js`, `drawPaddleAndBalls`](../reference/plugin-source/renderer.js#L476).

**OBSERVED:** Status elements are a combo pill at `(30,895)`, remaining-bricks
text at right y=909, a centered multiball/Konami pill, transient message at
y=116, and active-power pills at y=854. The footer reads exactly:
`← → / A D  MOVE`, `SPACE  LAUNCH     P  PAUSE`, and
`F10  SOUND ON|OFF     ESC  CLOSE`. Sources:
[`renderer.js`, `drawActivePowers`, `drawStatus`, and `drawFooter`](../reference/plugin-source/renderer.js#L687).

**OBSERVED:** Attract mode subdues the level-1 bricks and overlays a 616×520
dark panel with `A WINTERMUTE ARCADE`, a large `BREAKOUT`,
`25 LEVELS · 3 LIVES · ONE OPINIONATED DESKTOP`,
`PRESS ENTER TO SHIP`, and five high-score rows. Game-over/victory overlays say
respectively `GAME OVER` / `BREAKOUT SHIPPED`,
`THE NEXT RELEASE IS YOURS` / `RUNNING IN PRODUCTION`, the padded score, and
`PRESS ENTER TO REWORK`. A pause adds a centered `PAUSED` box. Sources:
[`renderer.js`, `drawAttract`, `drawEnd`, and `drawDynamic`](../reference/plugin-source/renderer.js#L742).

**OBSERVED:** Each machine boss has a distinct chassis, nameplate, segmented HP
meter, and animated pixel LED face. THE BITE uses a six-stripe classic rainbow
apple silhouette and changes stripe/leaf/bite treatment by face state. Sources:
[`renderer.js`, boss drawing functions](../reference/plugin-source/renderer.js#L44),
[`tests/artifacts/04-boss-stage-1.png`](../reference/plugin-source/tests/artifacts/04-boss-stage-1.png),
[`tests/artifacts/11-apple-boss.png`](../reference/plugin-source/tests/artifacts/11-apple-boss.png).

**OBSERVED:** `preview.png` is stale relative to current code: it shows an
`OMARCHY / BREAKOUT` header and `LEVEL 03/24`, whereas the current renderer uses
centered `BREAKOUT` and `/25`. Current live screenshots and tests corroborate
the renderer. It should not be treated as exact current UI authority. Sources:
[`preview.png`](../reference/plugin-source/preview.png),
[`renderer.js`, `drawHeader`](../reference/plugin-source/renderer.js#L373),
[`tests/artifacts/02-first-hit.png`](../reference/plugin-source/tests/artifacts/02-first-hit.png).

## Presentation effects

**OBSERVED:** Effects are deterministic presentation state. The particle cap is
72. A normal break creates 4 fragments and 4 sparks; nonfatal damage/steel
creates 5 sparks and no fragments. Breaks/steel request a 2.15-unit,
0.105-second shake; paddle/wall hits squash the ball but do not shake. Particles
have deterministic velocity, gravity, spin, color, and sub-second lifetimes.
Sources: [`effects.js`, constants, `spawnBrickBurst`, and `applyEvents`](../reference/plugin-source/effects.js#L4).

**OBSERVED:** A normal boss defeat creates 28 fragments, 34 sparks, three
expanding rings, a 0.16-second white flash, a 7.2-unit/0.38-second shake, and
the 0.065-second freeze. Boss projectile hits create red sparks and a
4.2-unit/0.20-second shake. Shield saves and boss stage changes have smaller
dedicated effects. Source: [`effects.js`, `spawnBossExplosion` and
`applyEvents`](../reference/plugin-source/effects.js#L94).

**OBSERVED:** With more than 12 balls, at most two dense brick bursts are
created per event batch, using 3 fragments/2 sparks per break; strong shake is
reduced to 0.65 units for 0.055 s and hit-freeze is suppressed. Dense boss
explosion counts are 12 fragments, 14 sparks, and 3 rings. Source:
[`effects.js`, `applyEvents`](../reference/plugin-source/effects.js#L171).

## Sound

**OBSERVED:** There are 12 logical cue classes backed by 15 mono, 16-bit,
44.1-kHz PCM WAV effects; break has four pitch-tier assets. At most one cue is
selected per rendered physics/event frame using this highest-first priority:

1. victory
2. game over
3. level clear
4. life lost or boss-shot hit
5. Konami, typed secret, or boss defeated
6. power-up, shield save, or boss stage
7. launch, sticky launch, or boss shot fired
8. level started
9. brick broken
10. brick damaged
11. steel
12. paddle, wall, or sticky catch

Sources: [`audio.js`, `PRIORITY` and `selectSound`](../reference/plugin-source/audio.js#L4),
[`Panel.qml`, `playEventSound`](../reference/plugin-source/Panel.qml#L205).

**OBSERVED:** Break pitch tier is 1 below multiplier 3, 2 at 3, 3 at 4–5,
and 4 at 6+. Impact classes break/damage/steel/bounce are additionally
rate-limited to one every 45 ms of wall time. Sources:
[`audio.js`, `IMPACT_SOUNDS` and `breakPitchTier`](../reference/plugin-source/audio.js#L19),
[`Panel.qml`, `playEventSound`](../reference/plugin-source/Panel.qml#L205).

| Asset | Duration | Playback volume |
|---|---:|---:|
| `bounce.wav` | 0.055 s | 0.12 |
| `steel.wav` | 0.090 s | 0.13 |
| `damage.wav` | 0.075 s | 0.15 |
| `break.wav` / `break-2.wav` / `break-3.wav` / `break-4.wav` | 0.140 / 0.130 / 0.121 / 0.110 s | 0.17 |
| `launch.wav` | 0.170 s | 0.18 |
| `ready.wav` | 0.270 s | 0.22 |
| `power.wav` | 0.370 s | 0.22 |
| `secret.wav` | 0.530 s | 0.22 |
| `life.wav` | 0.580 s | 0.22 |
| `level.wav` | 0.550 s | 0.24 |
| `gameover.wav` | 0.840 s | 0.22 |
| `victory.wav` | 1.020 s | 0.25 |

Sources: [`Panel.qml`, `SoundEffect` declarations](../reference/plugin-source/Panel.qml#L262),
[`sounds/`](../reference/plugin-source/sounds/),
[`test_audio.js`, format assertions](../reference/plugin-source/tests/test_audio.js#L32).

**OBSERVED:** Mute defaults off, is toggled by F10/footer click, and persists as
`{"version":1,"muted":bool}`. Invalid preferences fall back to sound on.
Sources: [`Panel.qml`, settings file and mute functions](../reference/plugin-source/Panel.qml#L138).

## Framework-only APIs and assumptions to leave behind

**OBSERVED:** The public Quickshell IPC target `breakout` exposes ping, summon,
new game, start level, launch, abstract key press/release, mute, secret-token
injection, state telemetry, and quit. Test-only ball/drop/power mutation is
disabled under the actual plugin shell. This IPC surface exists for host
integration/testing; it is not required to reproduce normal player behavior.
Source: [`Panel.qml`, `IpcHandler`](../reference/plugin-source/Panel.qml#L647).

**OBSERVED:** `Process`, `FileView`, `FloatingWindow`, `FrameAnimation`,
`SoundEffect`, QML Canvas, and host `hide` calls are implementation mechanisms,
not game mechanics. `hyprctl`, Quickshell, Omarchy theme paths, and
`omarchy-shell` launch commands must not be carried into the standalone
runtime. Sources: [`Panel.qml`](../reference/plugin-source/Panel.qml),
[`README.md`](../reference/plugin-source/README.md).

**INFERRED:** A standalone port may replace the two Canvas layers, monitor
sizing, QML focus, and SoundEffect objects without changing gameplay, provided
the fixed-step engine behavior, control semantics, visible state, and cue
semantics remain equivalent.

## Licensing and provenance

**OBSERVED:** The checked-in license is MIT, copyright © 2026 acrogenesis. It
allows use, modification, distribution, sublicensing, and sale provided the
copyright and permission notice accompany copies or substantial portions.
Source: [`LICENSE`](../reference/plugin-source/LICENSE).

**OBSERVED:** The manifest names `ely` as current author; README calls this a
locally maintained plugin and says LICENSE is the only file preserving original
authorship. The changelog calls version 1.0.0 a fork. The Omarchy matrix comment
cites Basecamp's `quattro/icon.txt`; the changelog calls the sound cues
original. Sources: [`manifest.json`](../reference/plugin-source/manifest.json#L4),
[`README.md`, license section](../reference/plugin-source/README.md#L54),
[`CHANGELOG.md`, 1.0.0 and 0.2.8`](../reference/plugin-source/CHANGELOG.md#L3),
[`levels.js`, Omarchy icon comment](../reference/plugin-source/levels.js#L48).

**UNKNOWN:** The tree does not identify the exact upstream repository/revision
from which the fork was derived, separately license each WAV, explain the
provenance of the Apple Bezier outline, or establish trademark permissions for
the Omarchy/Apple marks. The project-wide MIT file is the only included license
evidence. A behavior-level reimplementation is the safest choice where asset or
mark provenance matters; any substantial copied MIT material must retain its
notice.

## Ambiguities and implementation-sensitive edge cases

**OBSERVED:** Current code, not generic Breakout convention, resolves the
following potentially surprising cases:

- The first brick break scores at combo ×2.
- Space on the attract/end screen starts a serve but needs another press to
  launch.
- Enter during a live serve/game does not restart it.
- Normal timed powers continue counting down outside `playing` until a pause or
  level load; life loss does not clear them.
- Repeated slow pickups multiply current velocities by 0.72 again.
- Repeated shield pickups replace the charge with one rather than stacking.
- Fire pierces reinforced bricks one HP per separate contact, but steel bounces.
- Sticky catches do not reset combo; ordinary paddle returns do.
- Boss shots hit only the paddle, not the shield.
- A final level-25 break can leave an existing rain capsule frozen behind the
  victory overlay because the victory path clears balls but not drops.

Sources: [`engine.js`, relevant update/state functions](../reference/plugin-source/engine.js),
[`Panel.qml`, keyboard mapping](../reference/plugin-source/Panel.qml#L540).

**UNKNOWN:** It is not stated whether the stale preview's 24-level composition
or any behavior from an earlier upstream release should survive. All current
executable code/tests say 25 levels, so the 25-level implementation is the only
defensible port target.

**UNKNOWN:** The intended terminal approximations for curves, Apple/chassis
art, color when fewer than 16/true colors are available, Unicode glyph fallback,
audio-unavailable systems, and very small terminals are not specified by the
reference. These should be documented as port decisions rather than presented
as original behavior.

## Reimplementation Contract

The standalone game must preserve these externally observable behaviors:

- **OBSERVED:** Launch independently into the attract screen; require no
  Omarchy, Quickshell, Hyprland, compositor IPC, plugin host, or hardcoded user
  path.
- **OBSERVED:** Use the 800×1000 logical geometry, 240-Hz fixed simulation,
  deterministic serve/speed normalization, paddle size/speed/clamps, ball
  radius, collision order/response, drain semantics, and pause behavior
  specified above.
- **OBSERVED:** Reproduce all 25 named levels from the exact matrices/generation
  rules, including deterministic reinforced/steel/power cells, special bricks,
  progressive speed/paddle difficulty, four matrix bosses, and THE BITE arena.
- **OBSERVED:** Preserve scoring, combo timing/multiplier, lives, multiball
  final-ball rule, clear/life delays, game-over/victory/restart transitions,
  messages, and top-ten high-score semantics.
- **OBSERVED:** Preserve all eight power-ups with their durations, stacking and
  edge cases; preserve boss movement, prediction, stages, projectiles, faces,
  and deterministic level-25 capsule rain.
- **OBSERVED:** Preserve keyboard semantics for arrows, A/D, Space, Enter, P,
  F10, and Escape, including sticky aiming and safe pause-on-close. Preserve the
  DHH, BASECAMP, Signal 37, and exact Konami behaviors.
- **OBSERVED:** Preserve the important visual hierarchy: dark Tokyo-Night
  fallback palette, symbolic colored brick art, score/level/lives header,
  paddle/balls/capsules/shield, status/messages/footer, attract/pause/end
  overlays, boss identity/health/faces, and bounded deterministic impact
  feedback. Terminal-native approximations may replace pixel curves but must
  convey the same state clearly.
- **OBSERVED:** Preserve logical sound-event priority, break pitch tiers, impact
  throttling, mute control/persistence, and the distinction among bounce,
  steel, damage, break, power, life, level, secret, game-over, and victory cues;
  provide a graceful silent mode where audio is unavailable.
- **OBSERVED:** Keep core simulation deterministic and testable without an
  interactive terminal. Rendering, input, persistence, audio, and lifecycle
  must remain separable from game rules.
- **UNKNOWN:** Terminal resize/glyph/color fallback and exact asset reuse are
  port decisions. Document them explicitly, preserve the MIT notice for copied
  material, and do not claim unsupported asset/trademark provenance.
