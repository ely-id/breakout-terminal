## Presentation-layer evaluation

Evaluate whether Go with Bubble Tea and Lip Gloss is appropriate for the
standalone terminal frontend.

Do not select it merely for aesthetics.

Evaluate it against the actual requirements discovered in the original game:

- required update/tick frequency;
- rendering performance;
- keyboard responsiveness;
- collision/game-loop architecture;
- terminal resize handling;
- color fidelity;
- Unicode/glyph requirements;
- portability;
- binary distribution;
- dependency complexity.

If selected, keep gameplay simulation independent from Bubble Tea and
Lip Gloss.

Treat:

Bubble Tea = terminal application/event-loop adapter
Lip Gloss  = presentation/styling layer
Game Core  = framework-independent simulation

The visual implementation may improve terminal presentation beyond a literal
copy of the original plugin, provided that gameplay behavior and recognizable
visual identity remain faithful.

Document visual enhancements separately from behavioral changes.
