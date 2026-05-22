# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Repo Is

Master's thesis research project measuring energy efficiency across 18 programming languages using the [Green Metrics Tool (GMT)](https://github.com/green-coding-solutions/green-metrics-tool). The 8 core benchmarks come from the [Computer Language Benchmarks Game](https://benchmarksgame-team.pages.debian.net/benchmarksgame/index.html).

## Key Commands

```bash
# Environment (Linux only)
make setup          # bootstrap full local env (Docker, GMT, Go, Python 3.12)
make uninstall      # teardown local env

# Running benchmarks
make measure                                        # all languages, all 8 benchmarks
make measure lang=go                                # single language
make measure lang=go,c bench=binary-trees,mandelbrot iterations=10
make measure lang=go profile=test                   # fast test run (--dev-no-sleeps, _test.yml files)

# KWA exporter
make kwa-build      # compile to kwa/build/kwa
make kwa-run        # run from source

# KWA tests
cd kwa && GOCACHE=../.gocache_local go test ./...

# Notebooks (analysis/visualization)
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
jupyter lab
```

## Architecture

### Benchmark Layer (`benchmarks/<lang>/`)

Each language has one `.yml` file per benchmark for GMT to run. Naming conventions:
- `<benchmark>.yml` — canonical measurement run (`profile=measure`)
- `<benchmark>_test.yml` — fast smoke-test run (`profile=test`, uses smaller inputs)
- `gmt-cluster-scenario.yml` — all 8 benchmarks combined in separate flows (stdout suppressed to `/dev/null`)

Each YAML defines a `services` block (Docker container + optional `setup-commands` for compilation) and a `flow` block (sequence of timed commands). The C/C++/compiled language YMLs compile in `setup-commands`; interpreted languages just run directly in the flow.

`inputs/` holds shared input files (`fasta-*.txt`) mounted into containers at `/tmp/repo/inputs/`.

### `scripts/measure.sh`

Orchestrates GMT runs. Resolves which `benchmarks/<lang>/<bench>[_test].yml` files to pass to GMT's `runner.py` based on `lang=`, `bench=`, `profile=`, and `iterations=` arguments. Calls `green-metrics-tool/runner.py` via the GMT venv Python directly.

### KWA (`kwa/`)

Go CLI that reads measurements out of GMT's Postgres DB and exports them to CSV. Layer dependency chain:

```
cli → app/export + app/measure → api → service → data
```

- **`cmd/main.go`** — entrypoint
- **`internal/cli/`** — Cobra commands + Bubble Tea TUI (model/update/view split across `tui_model.go`, `tui_update.go`, `tui_view.go`)
- **`internal/app/export/`** — request contract, timestamp parsing/validation, executor orchestration
- **`internal/app/measure/`** — measure workflow executor (runs `scripts/measure.sh`, captures timestamps, then auto-exports)
- **`internal/service/`** — CSV serialization pipeline; parser maps metric keys to columns
- **`internal/data/`** — SQL queries against GMT's `phase_stats` table
- **`internal/constant/catalog.go`** — canonical lists of languages and benchmarks (used by both TUI multi-select and validation)

The CSV schema has fixed columns `run_id, measured_at, language, benchmark` followed by dynamic metric columns discovered from the data.

### `notebooks/`

Jupyter notebooks for data analysis. `01_data_cleaning.ipynb` processes raw GMT exports into `results/results_clean.csv`. Notebooks `03–10` are per-benchmark deep dives; `02` and `11` cover cross-language visualization and disk I/O.

### `green-metrics-tool/`

GMT subproject cloned locally by `make setup`. Not committed to this repo — generated on setup. The GMT venv at `green-metrics-tool/venv/` is used directly by `scripts/measure.sh`.

## Benchmark YAML Structure

```yaml
services:
  <container-name>:
    image: <docker-image>
    setup-commands:           # optional — for compilation
      - command: gcc ... -o /tmp/<binary>
    command: sleep infinity   # keep container alive

flow:
  - name: <Flow-Name>
    container: <container-name>
    commands:
      - type: console
        shell: sh             # required when using shell redirects (< or >)
        command: <cmd>
```

In `gmt-cluster-scenario.yml` files, each of the 8 benchmarks is a separate flow entry within the same service, all with `> /dev/null` to suppress stdout.

## Conventions

- Default to non-destructive operations; do not reset/revert unrelated changes.
- Validate behavior with focused tests first, then broader suites when touching shared paths.
- Keep docs and README pointers aligned when contracts or UX behavior change.
- Do not add `Co-Authored-By: Claude` or any Claude co-author line to commit messages.

## Commenting Standard

Every new or modified function must have a leading comment covering: behavior, key inputs, outputs/return value, notable side effects and errors. Keep comments concise and factual.
