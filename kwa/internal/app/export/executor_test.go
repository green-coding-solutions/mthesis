package export

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mthesis/kwa/internal/api"
	"mthesis/kwa/internal/config"
	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/entity"
	"mthesis/kwa/internal/service"
)

type fakeDataService struct {
	closed   bool
	closeErr error
}

func (f *fakeDataService) GetMetricKeys(context.Context) ([]string, error) {
	return []string{"k"}, nil
}

func (f *fakeDataService) GetPhaseMetricsBatch(context.Context, int, int) ([]entity.PhaseMetrics, error) {
	return []entity.PhaseMetrics{}, nil
}

func (f *fakeDataService) GetPhaseMetricsByID(context.Context, string) ([]entity.PhaseMetrics, error) {
	return []entity.PhaseMetrics{}, nil
}

func (f *fakeDataService) Close() error {
	f.closed = true
	return f.closeErr
}

type fakeCLIHandler struct {
	batchCalls []int
	byIDCalls  []string
	batchErr   error
	byIDErr    error
}

func (f *fakeCLIHandler) ExportBatch(_ context.Context, _ io.Writer, batchSize int) error {
	f.batchCalls = append(f.batchCalls, batchSize)
	return f.batchErr
}

func (f *fakeCLIHandler) ExportByID(_ context.Context, _ io.Writer, runID string) error {
	f.byIDCalls = append(f.byIDCalls, runID)
	return f.byIDErr
}

type fakeExporter struct{}

func (fakeExporter) ExportMeasurementsCSV(context.Context, io.Writer, int) error {
	return nil
}

func (fakeExporter) ExportMeasurementsCSVByID(context.Context, io.Writer, string) error {
	return nil
}

func TestExecuteBatch_Success(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "nested", "batch.csv")
	dataService := &fakeDataService{}
	handler := &fakeCLIHandler{}

	parserFactoryCalled := false
	exporterFactoryCalled := false

	deps := Dependencies{
		LoadDatabaseConfig: func() (config.DatabaseConfig, error) { return config.DatabaseConfig{}, nil },
		NewDataService:     func(config.DatabaseConfig) (DataService, error) { return dataService, nil },
		NewParserService: func() *service.ParserService {
			parserFactoryCalled = true
			return &service.ParserService{}
		},
		NewExporterService: func(*service.ParserService, service.PhaseMetricsBatchProvider) api.MeasurementsExporter {
			exporterFactoryCalled = true
			return fakeExporter{}
		},
		NewCLIHandler: func(api.MeasurementsExporter) CLIHandler { return handler },
	}

	executor := NewExecutorWithDeps(deps)
	err := executor.Execute(context.Background(), Request{Mode: constant.ExportModeBatch, BatchSize: 17, OutPath: outPath})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !parserFactoryCalled {
		t.Fatalf("expected parser factory call")
	}
	if !exporterFactoryCalled {
		t.Fatalf("expected exporter factory call")
	}
	if !dataService.closed {
		t.Fatalf("expected data service close")
	}
	if len(handler.batchCalls) != 1 || handler.batchCalls[0] != 17 {
		t.Fatalf("unexpected batch calls: %#v", handler.batchCalls)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file to exist, got: %v", err)
	}
}

func TestExecuteByID_Success(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "single.csv")
	handler := &fakeCLIHandler{}

	deps := Dependencies{
		LoadDatabaseConfig: func() (config.DatabaseConfig, error) { return config.DatabaseConfig{}, nil },
		NewDataService:     func(config.DatabaseConfig) (DataService, error) { return &fakeDataService{}, nil },
		NewParserService:   func() *service.ParserService { return &service.ParserService{} },
		NewExporterService: func(*service.ParserService, service.PhaseMetricsBatchProvider) api.MeasurementsExporter {
			return fakeExporter{}
		},
		NewCLIHandler: func(api.MeasurementsExporter) CLIHandler { return handler },
	}

	executor := NewExecutorWithDeps(deps)
	err := executor.Execute(context.Background(), Request{Mode: constant.ExportModeByID, RunID: "run-42", OutPath: outPath})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(handler.byIDCalls) != 1 || handler.byIDCalls[0] != "run-42" {
		t.Fatalf("unexpected by-id calls: %#v", handler.byIDCalls)
	}
}

func TestExecute_ConfigAndDataInitFailures(t *testing.T) {
	t.Parallel()

	t.Run("config failure", func(t *testing.T) {
		executor := NewExecutorWithDeps(Dependencies{
			LoadDatabaseConfig: func() (config.DatabaseConfig, error) { return config.DatabaseConfig{}, errors.New("boom") },
		})

		err := executor.Execute(context.Background(), Request{Mode: constant.ExportModeBatch, BatchSize: 1, OutPath: filepath.Join(t.TempDir(), "out.csv")})
		if err == nil || !strings.Contains(err.Error(), "load database config") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("data init failure", func(t *testing.T) {
		executor := NewExecutorWithDeps(Dependencies{
			LoadDatabaseConfig: func() (config.DatabaseConfig, error) { return config.DatabaseConfig{}, nil },
			NewDataService:     func(config.DatabaseConfig) (DataService, error) { return nil, errors.New("db init") },
		})

		err := executor.Execute(context.Background(), Request{Mode: constant.ExportModeBatch, BatchSize: 1, OutPath: filepath.Join(t.TempDir(), "out.csv")})
		if err == nil || !strings.Contains(err.Error(), "init data service") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestExecute_ValidationFailures(t *testing.T) {
	t.Parallel()

	loadCalled := false
	executor := NewExecutorWithDeps(Dependencies{
		LoadDatabaseConfig: func() (config.DatabaseConfig, error) {
			loadCalled = true
			return config.DatabaseConfig{}, nil
		},
	})

	cases := []Request{
		{Mode: constant.ExportModeBatch, BatchSize: 0},
		{Mode: constant.ExportModeByID, RunID: "   "},
	}

	for _, req := range cases {
		if err := executor.Execute(context.Background(), req); err == nil {
			t.Fatalf("expected validation error for request %#v", req)
		}
	}

	if loadCalled {
		t.Fatalf("load config should not run when validation fails")
	}
}

func TestExecute_CloseWarningIsWritten(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	dataService := &fakeDataService{closeErr: errors.New("close failed")}

	executor := NewExecutorWithDeps(Dependencies{
		LoadDatabaseConfig: func() (config.DatabaseConfig, error) { return config.DatabaseConfig{}, nil },
		NewDataService:     func(config.DatabaseConfig) (DataService, error) { return dataService, nil },
		NewParserService:   func() *service.ParserService { return &service.ParserService{} },
		NewExporterService: func(*service.ParserService, service.PhaseMetricsBatchProvider) api.MeasurementsExporter {
			return fakeExporter{}
		},
		NewCLIHandler: func(api.MeasurementsExporter) CLIHandler { return &fakeCLIHandler{} },
		Stderr:        stderr,
	})

	err := executor.Execute(context.Background(), Request{Mode: constant.ExportModeBatch, BatchSize: 1, OutPath: filepath.Join(t.TempDir(), "out.csv")})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "close data service") {
		t.Fatalf("expected close warning, got %q", stderr.String())
	}
}
