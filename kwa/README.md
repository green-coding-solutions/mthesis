# KWA CLI (v2)

KWA is a Go CLI for exporting Green Metrics measurements into CSV files.

## Current Scope

Version 2 supports:
- Interactive mode (`kwa`) powered by Bubble Tea + Lipgloss.
- Non-interactive exports with Cobra subcommands.
- Export modes: `batch` and `by-id`.
- Interactive `measure` workflow that runs `scripts/measure.sh` and then auto-exports the captured interval.
- Exports ordered by most recent `created_at`.
- Optional date-range filtering (`created_at BETWEEN from AND to`).

## Prerequisites

- Go 1.26+
- Database settings configured in `.env` (see `.env.example`)

The CLI uses `internal/config` and `internal/data` to connect to your database, so the `.env` values must be valid before running exports.

## Build and Run

From `mthesis/kwa`:

```bash
go build -o kwa ./cmd
./kwa
```

## Interactive Mode (`kwa`)

Running `kwa` with no subcommand:
1. Opens a TUI menu with the bat logo and three options:
   - `Export (Batch mode)`
   - `Export (by Run ID)`
   - `Measure`
2. Prompts for mode-specific fields.
3. Executes the selected workflow.
4. Shows a result screen with the output path and waits for `Esc` to exit.

### Interactive Inputs

- `Export (Batch mode)`
  - `Rows per batch` (optional, default `100`)
  - `From timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `To timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `Filename` (default `measurements.csv`)
- `Export (by Run ID)`
  - `Run ID` (required)
  - `From timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `To timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `Filename` (default `measurements.csv`)
- `Measure`
  - Step 1: benchmark multi-select (8 options)
  - Step 2: language multi-select (18 options)
  - Step 3: `Iterations` (required, positive integer) and `Filename`
  - Execution:
    - runs `scripts/measure.sh` with `profile=measure`
    - writes `measure.sh` stdout/stderr to `logs/measure.txt` (instead of printing in the TUI)
    - fails the workflow in TUI when script exits non-zero or logs GMT fatal markers (for example `Final_exception`)
    - includes selected languages, benchmarks, and iterations in the running screen
    - captures start/end timestamps in memory
    - runs `batch` export for the inclusive interval `[start, end]`

Date input behavior:
- If only date is provided, time defaults to `00:00:00`.
- Both `from` and `to` must be provided together, or both left empty.
- If both are empty, no date filter is applied.

### Interactive Output Path Rules

When `Filename` is entered:
- Empty value defaults to `measurements.csv`
- Missing `.csv` suffix is auto-appended
- If value contains `/`, it is treated as a path
- Otherwise output is written to `results/<Filename>`

Examples:
- `metrics` -> `results/metrics.csv`
- `exports/custom` -> `exports/custom.csv`

### Interactive Keybindings

- `Up` / `Down`: navigate menu and form focus
- `Enter`: confirm selection / continue / submit on last field
- `Space`: toggle focused benchmark/language in `measure` selection screens
- `A` or `a`: select all or clear all in `measure` selection screens
- `Esc`: quit
  - while a workflow is running, `Esc` opens a confirmation prompt
  - type `yes` + `Enter` to quit, or select `No`
- `Ctrl+U`: clear focused form field
- Mouse input is ignored

### Measure Script Resolution

The measure workflow resolves `scripts/measure.sh` with this precedence:
1. `KWA_MEASURE_SCRIPT`
2. `KWA_REPO_ROOT/scripts/measure.sh`
3. Upward search from current working directory to filesystem root (`scripts/measure.sh`)

`logs/measure.txt` is always written relative to the resolved repository root.

## Non-Interactive Commands

### Batch export

```bash
./kwa batch --batch-size 100 --from "2026-04-01" --to "2026-04-02 23:59:59" --out results/measurements.csv
```

Flags:
- `--batch-size` (default: `100`)
- `--from` (optional timestamp: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
- `--to` (optional timestamp: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
- `--out` (default: `results/measurements.csv`)

### Export by run ID

```bash
./kwa by-id --run-id <RUN_ID> --from "2026-04-01" --to "2026-04-02 23:59:59" --out results/measurements.csv
```

Alias:

```bash
./kwa byID --run-id <RUN_ID> --from "2026-04-01" --to "2026-04-02 23:59:59" --out results/measurements.csv
```

Flags:
- `--run-id` (required)
- `--from` (optional timestamp: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
- `--to` (optional timestamp: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
- `--out` (default: `results/measurements.csv`)

## Troubleshooting

- If you see DB connection errors, confirm `.env` values for host, port, user, password, database, and schema.
- If output creation fails, check directory permissions for the target path.
- If `go test ./...` fails due local Go toolchain issues, verify your installed Go stdlib/toolchain path and re-run.
