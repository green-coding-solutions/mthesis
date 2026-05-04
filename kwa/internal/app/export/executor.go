package export

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mthesis/kwa/internal/api"
	"mthesis/kwa/internal/config"
	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/data"
	"mthesis/kwa/internal/entity"
	"mthesis/kwa/internal/service"
)

// DataService captures only the data-layer capabilities needed for export execution.
type DataService interface {
	service.PhaseMetricsBatchProvider
	Close() error
}

// CLIHandler defines the API operations used by orchestration.
type CLIHandler interface {
	ExportBatch(ctx context.Context, w io.Writer, batchSize int, filter entity.TimeRangeFilter) error
	ExportByID(ctx context.Context, w io.Writer, runID string) error
}

// Dependencies defines overridable constructors used to build runtime dependencies.
// It exists to keep orchestration testable without touching external systems.
type Dependencies struct {
	LoadDatabaseConfig func() (config.DatabaseConfig, error)
	NewDataService     func(cfg config.DatabaseConfig) (DataService, error)
	NewParserService   func() *service.ParserService
	NewExporterService func(parserService *service.ParserService, source service.PhaseMetricsBatchProvider) api.MeasurementsExporter
	NewCLIHandler      func(exporter api.MeasurementsExporter) CLIHandler
	Stderr             io.Writer
}

// Executor orchestrates one export request from config loading to CSV generation.
type Executor struct {
	deps Dependencies
}

// NewExecutor returns the production export executor.
func NewExecutor() *Executor {
	return NewExecutorWithDeps(Dependencies{})
}

// NewExecutorWithDeps returns an executor that uses provided dependencies and
// falls back to production constructors for missing values.
func NewExecutorWithDeps(deps Dependencies) *Executor {
	defaultDeps := defaultDependencies()

	if deps.LoadDatabaseConfig == nil {
		deps.LoadDatabaseConfig = defaultDeps.LoadDatabaseConfig
	}
	if deps.NewDataService == nil {
		deps.NewDataService = defaultDeps.NewDataService
	}
	if deps.NewParserService == nil {
		deps.NewParserService = defaultDeps.NewParserService
	}
	if deps.NewExporterService == nil {
		deps.NewExporterService = defaultDeps.NewExporterService
	}
	if deps.NewCLIHandler == nil {
		deps.NewCLIHandler = defaultDeps.NewCLIHandler
	}
	if deps.Stderr == nil {
		deps.Stderr = defaultDeps.Stderr
	}

	return &Executor{deps: deps}
}

// defaultDependencies returns production constructors used by export orchestration.
func defaultDependencies() Dependencies {
	return Dependencies{
		LoadDatabaseConfig: config.LoadDatabaseConfig,
		NewDataService: func(cfg config.DatabaseConfig) (DataService, error) {
			return data.New(cfg)
		},
		NewParserService: service.NewParserService,
		NewExporterService: func(parserService *service.ParserService, source service.PhaseMetricsBatchProvider) api.MeasurementsExporter {
			return service.NewExporterService(parserService, source)
		},
		NewCLIHandler: func(exporter api.MeasurementsExporter) CLIHandler { return api.NewCLIHandler(exporter) },
		Stderr:        os.Stderr,
	}
}

// Execute runs one export request by wiring runtime dependencies and delegating
// mode-specific work to the CLI handler.
// It validates the request, creates the output file, closes resources, and
// returns wrapped config, data, output, or export errors.
func (e *Executor) Execute(ctx context.Context, req Request) error {
	normalizedReq, err := validateAndNormalizeRequest(req)
	if err != nil {
		return err
	}

	cfg, err := e.deps.LoadDatabaseConfig()
	if err != nil {
		return fmt.Errorf("load database config: %w", err)
	}

	dataService, err := e.deps.NewDataService(cfg)
	if err != nil {
		return fmt.Errorf("init data service: %w", err)
	}
	defer func() {
		if closeErr := dataService.Close(); closeErr != nil {
			e.writeWarning("close data service: %v\n", closeErr)
		}
	}()

	exporterService := e.deps.NewExporterService(e.deps.NewParserService(), dataService)
	handler := e.deps.NewCLIHandler(exporterService)

	outFile, err := createOutputFile(normalizedReq.OutPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			e.writeWarning("close output file %q: %v\n", normalizedReq.OutPath, closeErr)
		}
	}()

	switch normalizedReq.Mode {
	case constant.ExportModeBatch:
		if err := handler.ExportBatch(ctx, outFile, normalizedReq.BatchSize, normalizedReq.TimeRange); err != nil {
			return fmt.Errorf("batch export failed: %w", err)
		}
	case constant.ExportModeByID:
		if err := handler.ExportByID(ctx, outFile, normalizedReq.RunID); err != nil {
			return fmt.Errorf("single-run export failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported export mode %q", normalizedReq.Mode)
	}

	return nil
}

// writeWarning prints non-fatal cleanup warnings to stderr when available.
func (e *Executor) writeWarning(format string, args ...any) {
	if e.deps.Stderr == nil {
		return
	}

	_, _ = fmt.Fprintf(e.deps.Stderr, format, args...)
}

// validateAndNormalizeRequest applies output defaults and validates one request.
// Batch requests must include a positive batch size and valid optional time
// range; by-id requests require a run ID and discard any hidden time range.
func validateAndNormalizeRequest(req Request) (Request, error) {
	req.OutPath = strings.TrimSpace(req.OutPath)
	if req.OutPath == "" {
		req.OutPath = constant.DefaultOutPath
	}

	switch req.Mode {
	case constant.ExportModeBatch:
		if req.BatchSize <= 0 {
			return Request{}, fmt.Errorf("batch size must be greater than zero")
		}
		if err := req.TimeRange.Validate(); err != nil {
			return Request{}, err
		}
		req.TimeRange = req.TimeRange.Clone()
	case constant.ExportModeByID:
		req.RunID = strings.TrimSpace(req.RunID)
		if req.RunID == "" {
			return Request{}, fmt.Errorf("run ID must not be empty")
		}
		req.TimeRange = entity.TimeRangeFilter{}
	default:
		return Request{}, fmt.Errorf("invalid mode %q: use batch or by-id", req.Mode)
	}

	return req, nil
}

// createOutputFile ensures parent directories exist and truncates/creates target file.
func createOutputFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create output directory for %q: %w", path, err)
	}

	outFile, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create output file %q: %w", path, err)
	}

	return outFile, nil
}
