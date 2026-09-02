You are performing a targeted correction after final review.

Read completely:

    docs/FINAL_REVIEW.md

Address every remaining P1 finding in that document.

Do not reopen already resolved findings from docs/REVIEW.md unless the final
review demonstrates a regression related to them.

For the current ASCII rendering P1:

- preserve brick semantics when projected half-cells collapse into a terminal
  ASCII cell;
- a collision-active brick must never render as an ordinary playfield cell
  merely because its projection occupies only one vertical half-cell;
- preserve the correct semantic brick identity;
- do not change Game Core geometry or collision behavior.

Add regression coverage using real campaign layouts at the documented minimum
40x20 viewport and other supported sizes where useful.

The regression must detect collision-active/occupied brick regions that would
otherwise be represented only as ordinary playfield dots.

Run focused renderer tests first, then the full applicable validation suite.

Update docs/BUILD_REPORT.md with the correction and validation evidence.

Do not modify:

    reference/plugin-source/
    docs/GAME_SPEC.md
    docs/PORT_SPEC.md
    docs/FINAL_REVIEW.md

Do not address remaining P2 performance work unless required by this P1 fix.
