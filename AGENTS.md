# Breakout Terminal Port

## Goal

Port the existing Breakout plugin located under:

    reference/plugin-source/

into a standalone terminal game that can run independently of Omarchy and
the plugin runtime used by the original implementation.

The goal is behavioral and visual preservation where appropriate, not
preservation of the plugin architecture.

## Source reference

`reference/plugin-source/` is reference material.

Never modify files under this directory.

Do not assume its framework, language, APIs, lifecycle, or architecture.
Inspect the actual source before making claims.

Clearly distinguish:

- behavior intrinsic to the game;
- rendering/input implementation details;
- framework/plugin lifecycle;
- Omarchy-specific integration;
- environment-specific assumptions.

## Portability invariant

The final game must not require Omarchy.

It must not require the original plugin host/runtime merely to launch the game.

Do not introduce:

- hardcoded user paths;
- Omarchy commands;
- shell-specific assumptions;
- compositor-specific dependencies;
- desktop-environment-specific dependencies unless technically unavoidable
  and explicitly justified.

The target is a normal Linux terminal.

## Architecture

Do not choose the implementation language or terminal library before
`docs/PORT_SPEC.md` exists.

The standalone implementation should preferably separate:

- game state;
- game simulation / update loop;
- collision rules;
- input;
- terminal rendering;
- application lifecycle.

Core gameplay should be testable without requiring an interactive terminal.

## Fidelity

Preserve the original game's observable behavior wherever practical:

- controls;
- paddle behavior;
- ball movement;
- collision semantics;
- brick layout;
- score/lives/state transitions;
- timing;
- restart/game-over behavior;
- important visual characteristics.

When exact behavior cannot be determined, document the uncertainty instead of
inventing behavior.

## Licensing

Inspect the original project's license and provenance before copying code or
assets.

If redistribution rights are unclear, prefer a behavior-level reimplementation
rather than copying substantial source verbatim.

Document relevant license conclusions.

## Scope

Do not add unrelated features during the initial port.

First achieve a faithful, standalone Breakout.

## Validation

Every implementation must have:

- automated tests for deterministic gameplay logic where practical;
- a build/run smoke test;
- documentation for building and running the game;
- no dependency on the original plugin runtime at execution time.
