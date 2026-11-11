# Harness

`run.sh` is the script that produced this port. It starts three coding agents,
sends each one a fixed prompt, and refuses to move to the next phase until the
previous phase has written its document.

## Requirements

- [Herdr](https://herdr.dev), which manages the terminal
  panes and the agents running in them
- Codex CLI, the agent used in all three panes
- `jq`
- A local copy of the original Breakout plugin

## Usage

```sh
harness/run.sh /path/to/original-breakout-plugin
```

The script copies the plugin into `reference/plugin-source/`, replacing any
earlier snapshot. That directory is listed in `.gitignore` and agents are told
never to modify it. It then creates a Herdr workspace named `breakout-port`
with three panes: `mechanics`, `portability`, and `builder`.

## Phases

| Phase | Agent | Prompt | Must produce |
| --- | --- | --- | --- |
| 1. Reverse engineering | `mechanics` | `prompts/mechanics.md` | `docs/GAME_SPEC.md` |
| 1. Reverse engineering | `portability` | `prompts/portability.md` | `docs/PORT_SPEC.md` |
| 2. Implementation | `builder` | `prompts/builder.md` | The port and `docs/BUILD_REPORT.md` |
| 3. Independent review | `portability` | `prompts/reviewer.md` | `docs/REVIEW.md` |
| 4. Corrections | `builder` | `prompts/fix.md` | Fixes, recorded in `docs/BUILD_REPORT.md` |
| 5. Final gate | `portability` | `prompts/final-review.md` | `docs/FINAL_REVIEW.md` |

The two phase 1 agents run in parallel and are read-only with respect to
production code. The script stops if either one ends blocked, or if a required
document is missing or empty.

The final gate decides the exit status. `run.sh` reads the first line of
`docs/FINAL_REVIEW.md` and succeeds only when it is exactly `VERDICT: PASS`.

## When the gate does not pass

`prompts/final-fix.md` is not called by `run.sh`. It is the prompt for a manual
extra round: the builder addresses the P1 findings left in
`docs/FINAL_REVIEW.md`, and the final review is run again. This port needed one
such round, for bricks that disappeared in small ASCII viewports.

## Design notes

- Every prompt names the documents the agent has to read first, so no phase
  depends on what an earlier agent happened to say in its pane.
- The agent that reviews is never the agent that built.
- Findings carry a severity. The builder must address every valid P0 and P1,
  may address a P2 only when the change is small and low-risk, and has to write
  down the technical reason for any finding it rejects.
- [`AGENTS.md`](../AGENTS.md) holds the rules that apply to every phase.
