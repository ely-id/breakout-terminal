Perform a final independent validation of the Breakout terminal port.

Re-read:

    docs/GAME_SPEC.md
    docs/PORT_SPEC.md
    docs/REVIEW.md
    docs/BUILD_REPORT.md

Inspect the final source and run the relevant tests/build checks.

The first line of:

    docs/FINAL_REVIEW.md

must be exactly one of:

VERDICT: PASS
VERDICT: FAIL

PASS means there are no remaining P0 or P1 problems preventing this from being
a credible standalone port of the original Breakout plugin.

After the verdict, briefly document the evidence and any remaining P2 issues.

Do not modify production code.
