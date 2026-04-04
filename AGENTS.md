# AGENTS

Repository-level orientation for future agent runs.

## Purpose

This repository contains:

- Benchmark definitions and orchestration scripts for Green Metrics runs.
- A Go CLI exporter (`kwa`) that reads measurements from Postgres and writes CSV outputs.

## High-Level Layout

- `benchmarks/`: benchmark YAML inputs grouped by language.
- `scripts/`: setup, uninstall, and measurement automation scripts.
- `green-metrics-tool/`: local GMT checkout used during measurements.
- `kwa/`: exporter CLI implementation (Cobra + Bubble Tea + service/data layers).
- `docs/`: additional project docs/assets.

## Core Commands (from repo root)

```bash
make setup
make uninstall
make measure
make kwa-build
make kwa-run
```

## Conventions for Future Agents

- Default to non-destructive operations; do not reset/revert unrelated user changes.
- Validate behavior with focused tests first, then broader suites when touching shared paths.
- Keep docs and README pointers aligned when contracts or UX behavior change.
- Commenting standard:
  - Every new or modified function must include a leading comment that explains what it does.
  - The comment should cover behavior, key inputs, outputs/return value, and notable side effects/errors.
  - Keep comments concise and factual; avoid redundant line-by-line narration.

## Where To Go Next

For KWA architecture, runtime flows, invariants, and change-impact guidance, use:

- [`kwa/AGENTS.md`](kwa/AGENTS.md)
