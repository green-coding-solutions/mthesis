package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/entity"
)

// ExporterService streams normalized measurements to CSV files.
type ExporterService struct {
	parserService      *ParserService
	phaseMetricsSource PhaseMetricsBatchProvider
	errorLogPath       string
}

type batchFetchResult struct {
	phaseMetrics []entity.PhaseMetrics
	err          error
}

// PhaseMetricsBatchProvider is the data dependency needed for CSV export.
type PhaseMetricsBatchProvider interface {
	GetMetricKeys(ctx context.Context) ([]string, error)
	GetPhaseMetricsBatch(ctx context.Context, limit, offset int) ([]entity.PhaseMetrics, error)
	GetPhaseMetricsByID(ctx context.Context, runID string) ([]entity.PhaseMetrics, error)
}

// NewExporterService builds an exporter service with parser defaults and data source dependencies.
func NewExporterService(parserService *ParserService, phaseMetricsSource PhaseMetricsBatchProvider) *ExporterService {
	if parserService == nil {
		parserService = NewParserService()
	}

	return &ExporterService{
		parserService:      parserService,
		phaseMetricsSource: phaseMetricsSource,
		errorLogPath:       "logs/error_logs.txt",
	}
}

// ExportMeasurementsCSV streams measurements to CSV in paginated batches.
// Header layout is fixed: run_id,language,benchmark,<metric keys...>.
func (s *ExporterService) ExportMeasurementsCSV(ctx context.Context, w io.Writer, batchSize int) error {
	if err := s.validateWriterAndProvider(w); err != nil {
		return err
	}
	if batchSize <= 0 {
		return fmt.Errorf("batch size must be greater than zero")
	}

	csvWriter, metricKeys, err := s.newCSVWriterWithHeader(ctx, w)
	if err != nil {
		return err
	}
	errorLog, err := s.openErrorLogWriter()
	if err != nil {
		return err
	}
	defer errorLog.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	offset := 0
	// Prime the pipeline with the first batch fetch.
	nextBatchResult := s.fetchPhaseMetricsBatchAsync(ctx, batchSize, offset)
	for {
		// Wait for the batch that was prefetched in the previous iteration.
		result := <-nextBatchResult
		if result.err != nil {
			return result.err
		}

		phaseMetricsBatch := result.phaseMetrics
		if len(phaseMetricsBatch) == 0 {
			break
		}

		nextOffset := offset + len(phaseMetricsBatch)
		// Start fetching the next batch before writing the current one so DB I/O
		// can overlap with CSV parsing/writing while preserving row order.
		nextBatchResult = s.fetchPhaseMetricsBatchAsync(ctx, batchSize, nextOffset)

		// Keep all writes on this goroutine; csv.Writer and error log are not shared.
		if err := s.writePhaseMetricsRows(csvWriter, phaseMetricsBatch, metricKeys, errorLog); err != nil {
			return err
		}

		if err := flushCSV(csvWriter); err != nil {
			return fmt.Errorf("flush csv batch at offset %d: %w", offset, err)
		}

		offset = nextOffset
	}

	return nil
}

// ExportMeasurementsCSVByID writes measurements to CSV for a single run ID.
// Header layout is fixed: run_id,language,benchmark,<metric keys...>.
func (s *ExporterService) ExportMeasurementsCSVByID(ctx context.Context, w io.Writer, runID string) error {
	if err := s.validateWriterAndProvider(w); err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("runID must not be empty")
	}

	csvWriter, metricKeys, err := s.newCSVWriterWithHeader(ctx, w)
	if err != nil {
		return err
	}
	errorLog, err := s.openErrorLogWriter()
	if err != nil {
		return err
	}
	defer errorLog.Close()

	phaseMetricsByID, err := s.phaseMetricsSource.GetPhaseMetricsByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("load phase metrics for run %q: %w", runID, err)
	}

	if err := s.writePhaseMetricsRows(csvWriter, phaseMetricsByID, metricKeys, errorLog); err != nil {
		return err
	}

	if err := flushCSV(csvWriter); err != nil {
		return fmt.Errorf("flush csv rows for run %q: %w", runID, err)
	}

	return nil
}

// validateWriterAndProvider ensures required export dependencies are available.
func (s *ExporterService) validateWriterAndProvider(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("csv writer target must not be nil")
	}
	if s.phaseMetricsSource == nil {
		return fmt.Errorf("phase metrics provider must not be nil")
	}
	return nil
}

// newCSVWriterWithHeader creates a CSV writer and emits the canonical header row.
func (s *ExporterService) newCSVWriterWithHeader(ctx context.Context, w io.Writer) (*csv.Writer, []string, error) {
	metricKeys, err := s.phaseMetricsSource.GetMetricKeys(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("load metric keys: %w", err)
	}

	csvWriter := csv.NewWriter(w)
	header := append([]string{"run_id", "language", "benchmark"}, metricKeys...)
	if err := csvWriter.Write(header); err != nil {
		return nil, nil, fmt.Errorf("write csv header: %w", err)
	}
	if err := flushCSV(csvWriter); err != nil {
		return nil, nil, fmt.Errorf("flush csv header: %w", err)
	}

	return csvWriter, metricKeys, nil
}

// openErrorLogWriter opens the append-only log file used for skipped phase warnings.
func (s *ExporterService) openErrorLogWriter() (io.WriteCloser, error) {
	errorLogPath := strings.TrimSpace(s.errorLogPath)
	if errorLogPath == "" {
		errorLogPath = "logs/error_logs.txt"
	}

	errorLogDir := filepath.Dir(errorLogPath)
	if err := os.MkdirAll(errorLogDir, 0o755); err != nil {
		return nil, fmt.Errorf("create error log directory %q: %w", errorLogDir, err)
	}

	errorLog, err := os.OpenFile(errorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open error log file %q: %w", errorLogPath, err)
	}

	return errorLog, nil
}

// writePhaseMetricsRows parses each phase and appends corresponding CSV rows.
func (s *ExporterService) writePhaseMetricsRows(csvWriter *csv.Writer, phaseMetrics []entity.PhaseMetrics, metricKeys []string, errorLog io.Writer) error {
	for _, pm := range phaseMetrics {
		measurement, err := s.parserService.ParseMeasurementFromPhase(pm)
		if err != nil {
			if isUnknownDimensionError(err) {
				if logErr := logSkippedPhase(errorLog, pm, err); logErr != nil {
					return fmt.Errorf("write parse warning log for run %q phase %q: %w", pm.RunID, pm.Phase, logErr)
				}
				continue
			}
			return fmt.Errorf("parse measurement for run %q phase %q: %w", pm.RunID, pm.Phase, err)
		}

		row := measurementToCSVRow(measurement, metricKeys)
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write csv row for run %q: %w", measurement.RunID, err)
		}
	}

	return nil
}

// fetchPhaseMetricsBatchAsync fetches a batch in a goroutine and returns a one-shot result channel.
func (s *ExporterService) fetchPhaseMetricsBatchAsync(ctx context.Context, batchSize, offset int) <-chan batchFetchResult {
	resultCh := make(chan batchFetchResult, 1)
	go func() {
		phaseMetrics, err := s.phaseMetricsSource.GetPhaseMetricsBatch(ctx, batchSize, offset)
		if err != nil {
			err = fmt.Errorf("load phase metrics batch at offset %d: %w", offset, err)
		}
		resultCh <- batchFetchResult{phaseMetrics: phaseMetrics, err: err}
		close(resultCh)
	}()

	return resultCh
}

// flushCSV flushes buffered CSV writes and returns any writer error.
func flushCSV(csvWriter *csv.Writer) error {
	csvWriter.Flush()
	return csvWriter.Error()
}

// isUnknownDimensionError reports whether parsing failed due to unknown lookup dimensions.
func isUnknownDimensionError(err error) bool {
	return errors.Is(err, constant.ErrUnknownProgrammingLanguage) || errors.Is(err, constant.ErrUnknownBenchmark)
}

// logSkippedPhase records skipped phase context when parsing cannot map dimensions.
func logSkippedPhase(w io.Writer, phaseMetrics entity.PhaseMetrics, parseErr error) error {
	_, err := fmt.Fprintf(
		w,
		"run_id=%q phase=%q skipped=true reason=%q\n",
		phaseMetrics.RunID,
		phaseMetrics.Phase,
		parseErr.Error(),
	)
	return err
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
