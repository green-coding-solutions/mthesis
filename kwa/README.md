# KWA CLI (v1)

KWA is a Go CLI for exporting Green Metrics measurements into CSV files.

## Current Scope

Version 1 supports:
- Interactive mode (`kwa`) powered by Bubble Tea + Lipgloss.
- Non-interactive exports with Cobra subcommands.
- Two export modes: `batch` and `by-id`.

## Prerequisites

- Go 1.26+
- Database settings configured in `.env` (see `.env.example`)

The CLI uses `internal/config` and `internal/data` to connect to your database, so the `.env` values must be valid before running exports.

## Build and Run

From `/Users/brandao/mthesis/kwa`:

```bash
go build -o kwa ./cmd
./kwa
```

## Interactive Mode (`kwa`)

Running `kwa` with no subcommand:
1. Opens a TUI menu with the bat logo and two options:
   - `batch export`
   - `byID`
2. Prompts for mode-specific fields.
3. Executes the export.
4. Shows a result screen with the output path and waits for `q` to exit.

### Interactive Inputs

- `batch export`
  - `Rows per batch` (optional, default `100`)
  - `fileName` (default `measurements.csv`)
- `byID`
  - `Run ID` (required)
  - `fileName` (default `measurements.csv`)

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
./kwa batch --batch-size 100 --out results/measurements.csv
```

Flags:
- `--batch-size` (default: `100`)
- `--out` (default: `results/measurements.csv`)

### Export by run ID

```bash
./kwa by-id --run-id <RUN_ID> --out results/measurements.csv
```

Alias:

```bash
./kwa byID --run-id <RUN_ID> --out results/measurements.csv
```

Flags:
- `--run-id` (required)
- `--out` (default: `results/measurements.csv`)

## Troubleshooting

- If you see DB connection errors, confirm `.env` values for host, port, user, password, database, and schema.
- If output creation fails, check directory permissions for the target path.
- If `go test ./...` fails due local Go toolchain issues, verify your installed Go stdlib/toolchain path and re-run.
