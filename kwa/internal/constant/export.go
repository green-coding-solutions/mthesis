package constant

// DefaultBatchSize is the default row count per page for batch exports.
const DefaultBatchSize = 100

// DefaultOutPath is the default CSV output path for non-interactive commands.
const DefaultOutPath = "results/measurements.csv"

// DefaultCSVFilename is the default output filename in interactive prompts.
const DefaultCSVFilename = "measurements.csv"

// DefaultErrorLogPath is the default path for exporter parse warning logs.
const DefaultErrorLogPath = "logs/error_logs.txt"

// ExportMode selects which export operation should run.
type ExportMode string

const (
	// ExportModeBatch exports all runs using paginated reads.
	ExportModeBatch ExportMode = "batch"
	// ExportModeByID exports one run selected by ID.
	ExportModeByID ExportMode = "by-id"
)
