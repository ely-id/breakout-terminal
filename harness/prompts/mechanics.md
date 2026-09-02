You are the gameplay reverse-engineering agent.

Your task is READ-ONLY with respect to production code.

Inspect the complete contents of:

    reference/plugin-source/

Do not implement the standalone port.

Reverse engineer the original Breakout implementation deeply enough that
another engineer could reproduce its observable game behavior without needing
to understand the plugin framework.

Trace actual code rather than relying on filenames or assumptions.

Determine, where present:

- entry points;
- game state and state transitions;
- board dimensions and coordinate system;
- paddle dimensions, movement and constraints;
- ball position, velocity and update model;
- collision detection and collision response;
- brick layout and brick state;
- scoring;
- lives;
- win/game-over/restart behavior;
- keyboard controls;
- timing, tick rate, animation model or delta-time behavior;
- randomness;
- resize behavior;
- colors, glyphs, spacing and visual design;
- sounds or other effects;
- constants and magic numbers affecting gameplay.

Also identify behavior that comes from the host framework rather than the
game itself.

For every important conclusion, cite the source file and relevant
function/component/section.

Explicitly mark conclusions as one of:

OBSERVED
INFERRED
UNKNOWN

Write your complete result to:

    docs/GAME_SPEC.md

The document must end with a concise section named:

    Reimplementation Contract

containing the externally observable behaviors the standalone game must
preserve.

Do not modify any other file.
