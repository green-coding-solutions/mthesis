# KWA AGENTS

Deep operational context for `kwa/`.

Back to repo overview: [`../AGENTS.md`](../AGENTS.md)

## Scope

`kwa` is a Go CLI that exports Green Metrics phase measurements to CSV through:

- Interactive Bubble Tea TUI (`kwa`)
- Non-interactive Cobra commands (`kwa batch`, `kwa by-id`)

## Layer Map

Primary dependency direction:

`cli -> app/export -> api -> service -> data`

Supporting packages:

- `internal/config`: `.env` loading and DB config validation.
- `internal/constant`: export modes, defaults, canonical parser maps/errors.
- `internal/entity`: DTO/domain structs (`PhaseMetrics`, `Measurement`).

Concrete package roles:

- `cmd/main.go`: entrypoint, delegates to `internal/cli`.
- `internal/cli`: Cobra command wiring + Bubble Tea TUI state/update/view.
- `internal/app/export`: request contract, batch timestamp parsing/validation, dependency orchestration, output file creation.
- `internal/api`: thin adapter from CLI/executor calls into exporter service.
- `internal/service`: parser + CSV export pipeline and row serialization.
- `internal/data`: SQL queries, row scanning, metric-key discovery.

## Execution Flows

### 1) `batch` command

1. `internal/cli/root.go` parses flags (`--batch-size`, `--from`, `--to`, `--out`).
2. `internal/app/export.ParseTimeRange` validates/normalizes optional timestamps.
3. `internal/app/export.Executor.Execute` validates request and builds runtime deps:
   - load DB config
   - create data service
   - create parser + exporter service
   - create API handler
4. API handler calls `service.ExportMeasurementsCSV(...)`.
5. Service streams CSV in batches from data layer queries.

### 2) `by-id` command

1. `internal/cli/root.go` parses `--run-id` plus optional `--out`.
2. Same executor wiring path as batch mode.
3. API handler calls `service.ExportMeasurementsCSVByID(...)`.
4. Service fetches one run's phase rows and writes CSV.

### 3) Interactive TUI (`kwa`)

1. `internal/cli/runInteractive` starts Bubble Tea model.
2. Menu options:
   - `batch export`
   - `byID export`
3. Batch form inputs include optional from/to timestamps and `fileName`; byID form inputs include `runID` and `fileName`.
4. Form submit builds `appexport.Request`, then async export command runs via same executor path.
5. Result screen shows output path and final status.

## Contracts and Invariants

### DB time source and ordering

- Data is read from `phase_stats.created_at`.
- Query ordering is newest first:
  - `ORDER BY MAX(created_at) DESC, run_id, phase`
- Tie-breakers (`run_id`, `phase`) keep stable output order for identical timestamps.

### Optional batch date-range filtering

- SQL filter is optional, batch-only, and inclusive:
  - `created_at BETWEEN from AND to`
- No filter is applied when both bounds are empty (`nil`).
- `from` and `to` are all-or-nothing; one bound alone is invalid.
- `from > to` is invalid.
- By-id export does not accept or apply timestamp filters.

### Batch timestamp input parsing

Accepted input formats:

- `YYYY-MM-DD HH:MM:SS`
- `YYYY-MM-DD`

Normalization/semantics:

- Date-only input defaults to `00:00:00` (midnight).
- Parsing uses local timezone via `time.ParseInLocation(..., time.Local)`.

### CSV schema contract

- CSV header starts with:
  - `run_id,measured_at,language,benchmark`
- Dynamic metric columns come after those fixed columns.
- `measured_at` is sourced from entity `CreatedAt` and formatted with local `time.DateTime`.

### Current `q` behavior in TUI form

- In form screens, pressing `q`:
  - Quits app when focus is on non-`fileName` fields.
  - Inserts literal `q` when focus is on `fileName`.
- In menu/running/result screens, `q` quits.

## Skills Map For KWA Work

- `bubbletea`: TUI screens, keybindings, field focus behavior, model/update/view changes.
- `go-cobra`: command/subcommand behavior, flag parsing, CLI UX changes.
- `golang`: general implementation/refactor quality in Go packages.
- `go-testing`: table tests, subtests, assertions for CLI/service/data behavior.
- `software-architecture`: cross-layer changes and dependency-boundary decisions.
- `conventional-commits`: semantic commit message formatting when requested.

## Commenting Standard

- Every new or modified function must have a leading comment that explains what it does.
- For each function comment, include:
  - core behavior/purpose
  - important inputs and validation expectations
  - output/return semantics
  - notable side effects (I/O, DB writes, goroutines, file operations) and key error conditions
- Keep comments short and implementation-accurate; update comments whenever behavior changes.

## Fast Context Reload Checklist

1. Read [`README.md`](README.md) for current user-facing UX/flags.
2. Read this file for architecture and invariants.
3. Inspect current package surface quickly:
   ```bash
   cd kwa
   find internal -maxdepth 3 -type f | sort
   ```
4. Confirm contracts before edits:
   - time parsing: `internal/app/export/time_range.go`
   - CLI/TUI behavior: `internal/cli/root.go`, `internal/cli/tui_*.go`
   - query shape: `internal/data/phase_metrics.go`
   - CSV layout: `internal/service/exporter.go`
5. Run tests after edits.

## Change-Impact Checklist

When changing CLI flags/commands:

- Update `internal/cli/root.go` and `internal/cli/root_test.go`.
- Update `kwa/README.md` command docs if user-facing behavior changed.

When changing interactive form behavior:

- Update `internal/cli/tui_model.go`, `internal/cli/tui_update.go`, `internal/cli/tui_view.go`.
- Update `internal/cli/tui_test.go` and `internal/cli/path_test.go` if output/path rules changed.

When changing date parsing or validation rules:

- Update `internal/app/export/time_range.go` + `time_range_test.go`.
- Verify propagation in `internal/app/export/executor.go` + `executor_test.go`.
- Verify CLI parsing points in `internal/cli/root.go` + tests.

When changing SQL row shape/order/filter:

- Update `internal/data/phase_metrics.go` + `phase_metrics_test.go`.
- Ensure scan order still matches query columns (`run_id`, `created_at`, `phase`, `metrics`).

When changing CSV schema/serialization:

- Update `internal/service/exporter.go` + `exporter_test.go`.
- Keep `measured_at` naming and fixed-column ordering aligned with docs/tests.

## Validation Commands

Primary full-suite check for KWA:

```bash
cd kwa && GOCACHE=../.gocache_local go test ./...
```

Optional targeted checks:

```bash
cd kwa
GOCACHE=../.gocache_local go test ./internal/cli ./internal/app/export ./internal/service ./internal/data
```
