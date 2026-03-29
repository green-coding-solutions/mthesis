package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"mthesis/exporter/internal/entity"
)

type ExporterService struct {
	parserService      *ParserService
	phaseMetricsSource PhaseMetricsBatchProvider
}

// PhaseMetricsBatchProvider is the data dependency needed for batch CSV export.
type PhaseMetricsBatchProvider interface {
	GetMetricKeys(ctx context.Context) ([]string, error)
	GetPhaseMetricsBatch(ctx context.Context, limit, offset int) ([]entity.PhaseMetrics, error)
}

func NewExporterService(parserService *ParserService, phaseMetricsSource PhaseMetricsBatchProvider) *ExporterService {
	if parserService == nil {
		parserService = NewParserService()
	}

	return &ExporterService{
		parserService:      parserService,
		phaseMetricsSource: phaseMetricsSource,
	}
}

// ExportMeasurementsCSV streams measurements to CSV in paginated batches.
// Header layout is fixed: run_id,language,benchmark,<metric keys...>.
func (s *ExporterService) ExportMeasurementsCSV(ctx context.Context, w io.Writer, batchSize int) error {
	if w == nil {
		return fmt.Errorf("csv writer target must not be nil")
	}
	if s.phaseMetricsSource == nil {
		return fmt.Errorf("phase metrics provider must not be nil")
	}
	if batchSize <= 0 {
		return fmt.Errorf("batch size must be greater than zero")
	}

	metricKeys, err := s.phaseMetricsSource.GetMetricKeys(ctx)
	if err != nil {
		return fmt.Errorf("load metric keys: %w", err)
	}

	csvWriter := csv.NewWriter(w)
	header := append([]string{"run_id", "language", "benchmark"}, metricKeys...)
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush csv header: %w", err)
	}

	offset := 0
	for {
		phaseMetricsBatch, err := s.phaseMetricsSource.GetPhaseMetricsBatch(ctx, batchSize, offset)
		if err != nil {
			return fmt.Errorf("load phase metrics batch at offset %d: %w", offset, err)
		}
		if len(phaseMetricsBatch) == 0 {
			break
		}

		for _, pm := range phaseMetricsBatch {
			measurement, err := s.parserService.ParseMeasurementFromPhase(pm)
			if err != nil {
				return fmt.Errorf("parse measurement for run %q phase %q: %w", pm.RunID, pm.Phase, err)
			}

			row := measurementToCSVRow(measurement, metricKeys)
			if err := csvWriter.Write(row); err != nil {
				return fmt.Errorf("write csv row for run %q: %w", measurement.RunID, err)
			}
		}

		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("flush csv batch at offset %d: %w", offset, err)
		}

		offset += len(phaseMetricsBatch)
	}

	return nil
}

// measurementToCSVRow writes known scalar columns first, then one cell per metric key.
func measurementToCSVRow(m entity.Measurement, metricKeys []string) []string {
	row := make([]string, 0, 3+len(metricKeys))
	row = append(row, m.RunID, m.Language, m.Benchmark)
	for _, key := range metricKeys {
		if value, ok := m.Metrics[key]; ok {
			row = append(row, strconv.FormatInt(value, 10))
			continue
		}
		row = append(row, "")
	}

	return row
}
