# KWA CLI (v1)

KWA is a Go CLI for exporting Green Metrics measurements into CSV files.
Agent docs: [AGENTS.md](AGENTS.md) for deep KWA context and [../AGENTS.md](../AGENTS.md) for repo-level orientation.

## Current Scope

Version 1 supports:
- Interactive mode (`kwa`) powered by Bubble Tea + Lipgloss.
- Non-interactive exports with Cobra subcommands.
- Two export modes: `batch` and `by-id`.
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
1. Opens a TUI menu with the bat logo and two options:
   - `batch export`
   - `byID export`
2. Prompts for mode-specific fields.
3. Executes the export.
4. Shows a result screen with the output path and waits for `q` to exit.

### Interactive Inputs

- `batch export`
  - `Rows per batch` (optional, default `100`)
  - `From timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `To timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `fileName` (default `measurements.csv`)
- `byID export`
  - `Run ID` (required)
  - `From timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `To timestamp` (optional, `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
  - `fileName` (default `measurements.csv`)

Date input behavior:
- If only date is provided, time defaults to `00:00:00`.
- Both `from` and `to` must be provided together, or both left empty.
- If both are empty, no date filter is applied.

### Interactive Output Path Rules

When `fileName` is entered:
- Empty value defaults to `measurements.csv`
- Missing `.csv` suffix is auto-appended
- If value contains `/`, it is treated as a path
- Otherwise output is written to `results/<fileName>`

Examples:
- `metrics` -> `results/metrics.csv`
- `exports/custom` -> `exports/custom.csv`

### Interactive Keybindings

- `Up` / `Down`: navigate menu and form focus
- `Enter`: confirm selection / submit on last field
- `q`: quit
- `Ctrl+U`: clear focused form field
- Mouse wheel / left click are enabled in the menu

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
