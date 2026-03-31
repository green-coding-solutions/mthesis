package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mthesis/kwa/internal/api"
	"mthesis/kwa/internal/config"
	"mthesis/kwa/internal/data"
	"mthesis/kwa/internal/service"
)

// runExport executes one export request using the shared kwa service pipeline.
func runExport(ctx context.Context, req ExportRequest) error {
	normalizedReq, err := validateAndNormalizeRequest(req)
	if err != nil {
		return err
	}

	handler, closeDataSource, err := newCLIHandler()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeDataSource(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "close data service: %v\n", closeErr)
		}
	}()

	outFile, err := createOutputFile(normalizedReq.OutPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "close output file %q: %v\n", normalizedReq.OutPath, closeErr)
		}
	}()

	switch normalizedReq.Mode {
	case ExportModeBatch:
		if err := handler.ExportBatch(ctx, outFile, normalizedReq.BatchSize); err != nil {
			return fmt.Errorf("batch export failed: %w", err)
		}
	case ExportModeByID:
		if err := handler.ExportByID(ctx, outFile, normalizedReq.RunID); err != nil {
			return fmt.Errorf("single-run export failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported export mode %q", normalizedReq.Mode)
	}

	return nil
}

// validateAndNormalizeRequest validates command/form inputs and applies sane defaults.
func validateAndNormalizeRequest(req ExportRequest) (ExportRequest, error) {
	req.OutPath = strings.TrimSpace(req.OutPath)
	if req.OutPath == "" {
		req.OutPath = DefaultOutPath
	}

	switch req.Mode {
	case ExportModeBatch:
		if req.BatchSize <= 0 {
			return ExportRequest{}, fmt.Errorf("batch size must be greater than zero")
		}
	case ExportModeByID:
		req.RunID = strings.TrimSpace(req.RunID)
		if req.RunID == "" {
			return ExportRequest{}, fmt.Errorf("run ID must not be empty")
		}
	default:
		return ExportRequest{}, fmt.Errorf("invalid mode %q: use batch or by-id", req.Mode)
	}

	return req, nil
}

// newCLIHandler wires configuration, data access, and service dependencies for export.
func newCLIHandler() (*api.CLIHandler, func() error, error) {
	cfg, err := config.LoadDatabaseConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("load database config: %w", err)
	}

	dataService, err := data.New(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("init data service: %w", err)
	}

	exporterService := service.NewExporterService(service.NewParserService(), dataService)
	handler := api.NewCLIHandler(exporterService)

	return handler, dataService.Close, nil
}

// createOutputFile ensures the output directory exists before creating the CSV file.
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
