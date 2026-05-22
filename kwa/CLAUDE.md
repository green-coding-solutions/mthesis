# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Scope

`kwa` is a Go CLI that exports Green Metrics phase measurements to CSV:

- Interactive Bubble Tea TUI (`kwa`)
- Non-interactive Cobra commands (`kwa batch`, `kwa by-id`)
- Interactive measure workflow (`kwa` → Measure): runs `scripts/measure.sh`, captures start/end timestamps, then auto-exports the interval

## Layer Map

Primary dependency direction:

```
cli → app/export + app/measure → api → service → data
```

- `cmd/main.go` — entrypoint, delegates to `internal/cli`
- `internal/cli` — Cobra command wiring + Bubble Tea TUI (`tui_model.go`, `tui_update.go`, `tui_view.go`)
- `internal/app/export` — request contract, batch timestamp parsing/validation, executor orchestration
- `internal/app/measure` — measure workflow executor (script resolution, timestamps, export handoff)
- `internal/api` — thin adapter from CLI/executor calls into service
- `internal/service` — parser + CSV export pipeline and row serialization
- `internal/data` — SQL queries and row scanning against `phase_stats`
- `internal/constant/catalog.go` — canonical language and benchmark lists (shared by TUI multi-select and validation)
- `internal/config` — `.env` loading and DB config validation
- `internal/entity` — DTO/domain structs (`PhaseMetrics`, `Measurement`)

## Execution Flows

### `batch` command

1. `internal/cli/root.go` parses flags (`--batch-size`, `--from`, `--to`, `--out`).
2. `internal/app/export.ParseTimeRange` validates/normalizes optional timestamps.
3. `internal/app/export.Executor.Execute` validates the request and builds runtime deps (DB config → data service → parser + exporter service → API handler).
4. API handler calls `service.ExportMeasurementsCSV(...)`.
5. Service streams CSV in batches from data layer.

### `by-id` command

Same executor wiring as batch. API handler calls `service.ExportMeasurementsCSVByID(...)`, which fetches one run's phase rows and writes CSV.

### Interactive TUI (`kwa`)

1. `internal/cli/runInteractive` starts Bubble Tea model.
2. Menu options: `Export (Batch mode)`, `Export (by Run ID)`, `Measure`.
3. Form submit builds the appropriate `appexport.Request` or `appmeasure.Request` and runs async via the same executor paths.
4. Result screen shows output path and waits for `Esc`.

**Measure workflow specifics:**
- Runs `scripts/measure.sh` with `profile=measure`; stdout/stderr go to `logs/measure.txt` (not the TUI)
- Fails if script exits non-zero or logs GMT fatal markers (e.g. `Final_exception`)
- Captures start/end timestamps in memory and runs batch export for `[start, end]`

**Measure script resolution precedence:**
1. `KWA_MEASURE_SCRIPT` env var
2. `KWA_REPO_ROOT/scripts/measure.sh`
3. Upward search from CWD to filesystem root

## Contracts and Invariants

### DB time source and ordering

- Data is read from `phase_stats.created_at`.
- Query ordering: `ORDER BY MAX(created_at) DESC, run_id, phase` (newest first; tie-breakers keep stable output).

### Batch date-range filtering

- SQL filter is optional, batch-only, and inclusive: `created_at BETWEEN from AND to`.
- `from` and `to` are all-or-nothing — one bound alone is invalid.
- `from > to` is invalid.
- By-id export does not accept timestamp filters.

### Batch timestamp input parsing

Accepted formats: `YYYY-MM-DD HH:MM:SS` or `YYYY-MM-DD` (date-only defaults to `00:00:00`).  
Parsing uses local timezone via `time.ParseInLocation(..., time.Local)`.

### CSV schema

Fixed columns: `run_id, measured_at, language, benchmark`  
Dynamic metric columns follow, discovered from the data.  
`measured_at` is formatted with local `time.DateTime`.

### `q` behavior in TUI form screens

- On non-`fileName` fields: quits the app.
- On `fileName` field: inserts literal `q`.

## Validation Commands

```bash
# Full suite
cd kwa && GOCACHE=../.gocache_local go test ./...

# Targeted
cd kwa && GOCACHE=../.gocache_local go test ./internal/cli ./internal/app/export ./internal/service ./internal/data
```

## Change-Impact Checklist

**CLI flags/commands** → `internal/cli/root.go`, `root_test.go`, `kwa/README.md`

**TUI form behavior** → `tui_model.go`, `tui_update.go`, `tui_view.go`, `tui_test.go`, `path_test.go`

**Date parsing/validation** → `internal/app/export/time_range.go` + `time_range_test.go`; verify propagation in `executor.go` + `executor_test.go` and `internal/cli/root.go`

**SQL row shape/order/filter** → `internal/data/phase_metrics.go` + `phase_metrics_test.go`; confirm scan order still matches query columns (`run_id`, `created_at`, `phase`, `metrics`)

**CSV schema/serialization** → `internal/service/exporter.go` + `exporter_test.go`; keep `measured_at` naming and fixed-column ordering aligned

## Commenting Standard

Every new or modified function must have a leading comment covering: behavior, key inputs, outputs/return value, notable side effects and errors. Update comments whenever behavior changes.
