Act now as an independent port reviewer.

Do not edit production code.

Review the implementation against:

    docs/GAME_SPEC.md
    docs/PORT_SPEC.md
    docs/BUILD_REPORT.md
    reference/plugin-source/

Inspect the actual implementation, not only the reports.

Run the project's tests and relevant build/smoke checks.

Look specifically for:

- gameplay differences from the original;
- incorrect collision behavior;
- timing mistakes;
- rendering assumptions;
- terminal resize bugs;
- input issues;
- remaining Omarchy/plugin-host dependencies;
- hardcoded machine/user paths;
- unnecessary dependencies;
- architectural coupling that harms portability;
- claims in BUILD_REPORT.md that are not supported by the implementation.

Classify findings:

P0 = port fundamentally incorrect or unusable
P1 = meaningful behavioral/portability bug
P2 = improvement that does not block the initial port

Write findings to:

    docs/REVIEW.md

Each finding must identify the relevant file/function or behavior and explain
the expected correction.

Do not modify production code.
