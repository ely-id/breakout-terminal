You are the implementation agent for the standalone Breakout port.

Read completely:

    AGENTS.md
    docs/GAME_SPEC.md
    docs/PORT_SPEC.md

Then independently inspect the relevant original source under:

    reference/plugin-source/

Do not blindly trust the analysis documents when the source contradicts them.

Implement the standalone terminal game using the architecture selected in
PORT_SPEC.md.

The application must:

- run without Omarchy;
- run without the original plugin host;
- preserve the gameplay contract from GAME_SPEC.md;
- satisfy the acceptance criteria from PORT_SPEC.md;
- keep game simulation reasonably separated from terminal IO;
- avoid user-specific paths and environment assumptions.

Do not expand scope with unrelated features.

Add automated tests for deterministic game mechanics where practical,
especially collision/state-transition behavior.

Run the available tests and build checks.

Perform an actual launch smoke test if possible.

Create or update README.md with clear instructions for:

- dependencies;
- building/installing;
- running;
- controls.

Write:

    docs/BUILD_REPORT.md

describing:

- what was implemented;
- intentional deviations from the original;
- tests executed;
- known limitations;
- unresolved uncertainties.

You may modify production files, tests, README.md and BUILD_REPORT.md.

Do not modify GAME_SPEC.md or PORT_SPEC.md.
