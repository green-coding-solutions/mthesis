package api

import (
	"context"
	"fmt"
	"io"
)

// MeasurementsExporter defines the kwa-service capabilities used by CLI handlers.
type MeasurementsExporter interface {
	ExportMeasurementsCSV(ctx context.Context, w io.Writer, batchSize int) error
	ExportMeasurementsCSVByID(ctx context.Context, w io.Writer, runID string) error
}

// CLIHandler is a thin API layer for CLI commands.
type CLIHandler struct {
	exporter MeasurementsExporter
}

func NewCLIHandler(exporter MeasurementsExporter) *CLIHandler {
	return &CLIHandler{exporter: exporter}
}

// ExportBatch handles batch CSV export requests from CLI commands.
func (h *CLIHandler) ExportBatch(ctx context.Context, w io.Writer, batchSize int) error {
	if h.exporter == nil {
		return fmt.Errorf("kwa service must not be nil")
	}

	if err := h.exporter.ExportMeasurementsCSV(ctx, w, batchSize); err != nil {
		return fmt.Errorf("export batch csv: %w", err)
	}

	return nil
}

// ExportByID handles single-run CSV export requests from CLI commands.
func (h *CLIHandler) ExportByID(ctx context.Context, w io.Writer, runID string) error {
	if h.exporter == nil {
		return fmt.Errorf("kwa service must not be nil")
	}

	if err := h.exporter.ExportMeasurementsCSVByID(ctx, w, runID); err != nil {
		return fmt.Errorf("export csv by run id %q: %w", runID, err)
	}

	return nil
}
