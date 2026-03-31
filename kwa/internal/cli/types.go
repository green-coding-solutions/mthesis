package cli

import "context"

const (
	// DefaultBatchSize is the fallback batch size used by CLI and TUI flows.
	DefaultBatchSize = 100
	// DefaultCSVFilename is the fallback filename for interactive exports.
	DefaultCSVFilename = "measurements.csv"
	// DefaultOutPath is the default output path for non-interactive commands.
	DefaultOutPath = "results/measurements.csv"
)

// ExportMode describes which exporter operation should run.
type ExportMode string

const (
	// ExportModeBatch runs paginated CSV export for all available runs.
	ExportModeBatch ExportMode = "batch"
	// ExportModeByID runs CSV export for a single run ID.
	ExportModeByID ExportMode = "by-id"
)

// ExportRequest carries all execution inputs needed by the exporter pipeline.
type ExportRequest struct {
	Mode      ExportMode
	BatchSize int
	RunID     string
	OutPath   string
}

// ExportExecutor executes a prepared export request.
type ExportExecutor func(ctx context.Context, req ExportRequest) error
