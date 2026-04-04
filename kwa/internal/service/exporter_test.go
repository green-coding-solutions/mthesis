package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/entity"
)

type batchCall struct {
	limit  int
	offset int
	filter entity.TimeRangeFilter
}

type byIDCall struct {
	runID  string
	filter entity.TimeRangeFilter
}

type fakePhaseMetricsProvider struct {
	metricKeys          []string
	metricKeysErr       error
	batches             [][]entity.PhaseMetrics
	batchErrAtCall      map[int]error
	phaseMetricsByID    map[string][]entity.PhaseMetrics
	phaseMetricsByIDErr map[string]error
	getByIDCalls        []byIDCall
	getBatchCalls       []batchCall
	getBatchCallIdx     int
}

func withTempErrorLogPath(t *testing.T, exporterService *ExporterService) *ExporterService {
	t.Helper()
	exporterService.errorLogPath = filepath.Join(t.TempDir(), "logs", "error_logs.txt")
	return exporterService
}

func (f *fakePhaseMetricsProvider) GetMetricKeys(_ context.Context) ([]string, error) {
	if f.metricKeysErr != nil {
		return nil, f.metricKeysErr
	}
	return f.metricKeys, nil
}

func (f *fakePhaseMetricsProvider) GetPhaseMetricsBatch(
	_ context.Context,
	limit, offset int,
	filter entity.TimeRangeFilter,
) ([]entity.PhaseMetrics, error) {
	f.getBatchCalls = append(f.getBatchCalls, batchCall{
		limit:  limit,
		offset: offset,
		filter: filter.Clone(),
	})
	callIndex := f.getBatchCallIdx
	f.getBatchCallIdx++

	if err, ok := f.batchErrAtCall[callIndex]; ok {
		return nil, err
	}
	if callIndex >= len(f.batches) {
		return []entity.PhaseMetrics{}, nil
	}

	return f.batches[callIndex], nil
}

func (f *fakePhaseMetricsProvider) GetPhaseMetricsByID(
	_ context.Context,
	runID string,
	filter entity.TimeRangeFilter,
) ([]entity.PhaseMetrics, error) {
	f.getByIDCalls = append(f.getByIDCalls, byIDCall{
		runID:  runID,
		filter: filter.Clone(),
	})
	if err, ok := f.phaseMetricsByIDErr[runID]; ok {
		return nil, err
	}

	if rows, ok := f.phaseMetricsByID[runID]; ok {
		return rows, nil
	}

	return []entity.PhaseMetrics{}, nil
}

func TestNewExporterService_UsesDefaultParserWhenNil(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"x"},
		batches: [][]entity.PhaseMetrics{
			{
				{
					RunID:   "run-1",
					Phase:   "005_Go-Binary-Trees",
					Metrics: map[string]int64{"x": 1},
				},
			},
			{},
		},
	}

	exporterService := NewExporterService(nil, provider)
	if exporterService.parserService == nil {
		t.Fatalf("parserService should be initialized when nil is passed")
	}
	if exporterService.errorLogPath != constant.DefaultErrorLogPath {
		t.Fatalf("errorLogPath = %q, want %q", exporterService.errorLogPath, constant.DefaultErrorLogPath)
	}
	exporterService = withTempErrorLogPath(t, exporterService)

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 1, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("unexpected error with default parser: %v", err)
	}
}

func TestExportMeasurementsCSV_Success(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{
			"cpu_time_powermetrics_vm-docker_vm-ns",
			"gpu_carbon_powermetrics_component-component-ug",
		},
		batches: [][]entity.PhaseMetrics{
			{
				{
					RunID:      "run-1",
					MeasuredAt: time.Date(2026, time.April, 2, 10, 0, 0, 0, time.Local),
					Phase:      "005_Go-Binary-Trees",
					Metrics:    map[string]int64{"cpu_time_powermetrics_vm-docker_vm-ns": 47560725453},
				},
				{
					RunID:      "run-1",
					MeasuredAt: time.Date(2026, time.April, 2, 10, 0, 0, 0, time.Local),
					Phase:      "009_python-regex-redux",
					Metrics:    map[string]int64{"gpu_carbon_powermetrics_component-component-ug": 13},
				},
			},
			{
				{
					RunID:      "run-2",
					MeasuredAt: time.Date(2026, time.April, 1, 9, 0, 0, 0, time.Local),
					Phase:      "006_Go-Fasta",
					Metrics:    map[string]int64{},
				},
			},
			{},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("ExportMeasurementsCSV() error = %v", err)
	}

	records, err := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()
	if err != nil {
		t.Fatalf("csv read error = %v", err)
	}

	want := [][]string{
		{
			"run_id",
			"measured_at",
			"language",
			"benchmark",
			"cpu_time_powermetrics_vm-docker_vm-ns",
			"gpu_carbon_powermetrics_component-component-ug",
		},
		{
			"run-1",
			"2026-04-02 10:00:00",
			"go",
			"binary-trees",
			"47560725453",
			"",
		},
		{
			"run-1",
			"2026-04-02 10:00:00",
			"python",
			"regex-redux",
			"",
			"13",
		},
		{
			"run-2",
			"2026-04-01 09:00:00",
			"go",
			"fasta",
			"",
			"",
		},
	}

	if !reflect.DeepEqual(records, want) {
		t.Fatalf("csv records mismatch:\n got=%#v\nwant=%#v", records, want)
	}

	wantBatchCalls := []batchCall{
		{limit: 2, offset: 0},
		{limit: 2, offset: 2},
		{limit: 2, offset: 3},
	}
	if !reflect.DeepEqual(provider.getBatchCalls, wantBatchCalls) {
		t.Fatalf("batch calls mismatch:\n got=%#v\nwant=%#v", provider.getBatchCalls, wantBatchCalls)
	}
}

func TestExportMeasurementsCSV_GetMetricKeysError(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeysErr: errors.New("boom"),
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportMeasurementsCSV_GetPhaseMetricsBatchError(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		batches:    [][]entity.PhaseMetrics{{}},
		batchErrAtCall: map[int]error{
			0: errors.New("batch failed"),
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportMeasurementsCSV_PrefetchNextBatchError(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		batches: [][]entity.PhaseMetrics{
			{
				{
					RunID:   "run-1",
					Phase:   "005_Go-Binary-Trees",
					Metrics: map[string]int64{"k": 1},
				},
			},
		},
		batchErrAtCall: map[int]error{
			1: errors.New("batch failed at second call"),
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 1, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "offset 1") {
		t.Fatalf("expected offset in error message, got %q", err.Error())
	}

	records, readErr := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()
	if readErr != nil {
		t.Fatalf("csv read error = %v", readErr)
	}
	wantRecords := [][]string{
		{"run_id", "measured_at", "language", "benchmark", "k"},
		{"run-1", "", "go", "binary-trees", "1"},
	}
	if !reflect.DeepEqual(records, wantRecords) {
		t.Fatalf("csv records mismatch:\n got=%#v\nwant=%#v", records, wantRecords)
	}

	wantBatchCalls := []batchCall{
		{limit: 1, offset: 0},
		{limit: 1, offset: 1},
	}
	if !reflect.DeepEqual(provider.getBatchCalls, wantBatchCalls) {
		t.Fatalf("batch calls mismatch:\n got=%#v\nwant=%#v", provider.getBatchCalls, wantBatchCalls)
	}
}

func TestExportMeasurementsCSV_StopsAfterTerminalEmptyBatch(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		batches: [][]entity.PhaseMetrics{
			{
				{
					RunID:   "run-1",
					Phase:   "005_Go-Binary-Trees",
					Metrics: map[string]int64{"k": 1},
				},
			},
			{},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 1, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("ExportMeasurementsCSV() error = %v", err)
	}

	wantBatchCalls := []batchCall{
		{limit: 1, offset: 0},
		{limit: 1, offset: 1},
	}
	if !reflect.DeepEqual(provider.getBatchCalls, wantBatchCalls) {
		t.Fatalf("batch calls mismatch:\n got=%#v\nwant=%#v", provider.getBatchCalls, wantBatchCalls)
	}
}

func TestExportMeasurementsCSV_ParseError(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		batches: [][]entity.PhaseMetrics{
			{
				{
					RunID:   "run-1",
					Phase:   "invalid-phase",
					Metrics: map[string]int64{"k": 1},
				},
			},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportMeasurementsCSV_UnknownLanguageOrBenchmark_LogsAndContinues(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "error_logs.txt")
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		batches: [][]entity.PhaseMetrics{
			{
				{
					RunID:   "run-1",
					Phase:   "005_Go-Binary-Trees",
					Metrics: map[string]int64{"k": 1},
				},
				{
					RunID:   "run-2",
					Phase:   "005_Pascal-Binary-Trees",
					Metrics: map[string]int64{"k": 2},
				},
				{
					RunID:   "run-3",
					Phase:   "005_Go-Binary-Treez",
					Metrics: map[string]int64{"k": 3},
				},
			},
			{},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))
	exporterService.errorLogPath = logPath

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 10, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("ExportMeasurementsCSV() error = %v", err)
	}

	records, err := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()
	if err != nil {
		t.Fatalf("csv read error = %v", err)
	}

	want := [][]string{
		{"run_id", "measured_at", "language", "benchmark", "k"},
		{"run-1", "", "go", "binary-trees", "1"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("csv records mismatch:\n got=%#v\nwant=%#v", records, want)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logData)
	if !strings.Contains(logText, `phase="005_Pascal-Binary-Trees"`) {
		t.Fatalf("expected unknown language log entry, got %q", logText)
	}
	if !strings.Contains(logText, `phase="005_Go-Binary-Treez"`) {
		t.Fatalf("expected unknown benchmark log entry, got %q", logText)
	}
}

func TestExportMeasurementsCSV_InvalidArguments(t *testing.T) {
	testCases := []struct {
		name      string
		writer    io.Writer
		service   *ExporterService
		batchSize int
	}{
		{
			name:      "nil writer",
			writer:    nil,
			service:   withTempErrorLogPath(t, NewExporterService(NewParserService(), &fakePhaseMetricsProvider{metricKeys: []string{"k"}})),
			batchSize: 1,
		},
		{
			name:      "nil provider",
			writer:    &bytes.Buffer{},
			service:   withTempErrorLogPath(t, NewExporterService(NewParserService(), nil)),
			batchSize: 1,
		},
		{
			name:      "invalid batch size",
			writer:    &bytes.Buffer{},
			service:   withTempErrorLogPath(t, NewExporterService(NewParserService(), &fakePhaseMetricsProvider{metricKeys: []string{"k"}})),
			batchSize: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.service.ExportMeasurementsCSV(context.Background(), tc.writer, tc.batchSize, entity.TimeRangeFilter{})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestExportMeasurementsCSV_PropagatesDateRangeToProvider(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		batches:    [][]entity.PhaseMetrics{{}},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	from := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, time.April, 2, 0, 0, 0, 0, time.Local)

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 5, entity.TimeRangeFilter{From: &from, To: &to})
	if err != nil {
		t.Fatalf("ExportMeasurementsCSV() error = %v", err)
	}
	if len(provider.getBatchCalls) != 1 {
		t.Fatalf("expected one batch call, got %#v", provider.getBatchCalls)
	}
	if provider.getBatchCalls[0].filter.From == nil || provider.getBatchCalls[0].filter.To == nil {
		t.Fatalf("expected date range in provider call")
	}
	if !provider.getBatchCalls[0].filter.From.Equal(from) || !provider.getBatchCalls[0].filter.To.Equal(to) {
		t.Fatalf("unexpected propagated range: %#v", provider.getBatchCalls[0])
	}
}

func TestExportMeasurementsCSVByID_Success(t *testing.T) {
	const runID = "run-1"
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{
			"cpu_time_powermetrics_vm-docker_vm-ns",
			"gpu_carbon_powermetrics_component-component-ug",
		},
		phaseMetricsByID: map[string][]entity.PhaseMetrics{
			runID: {
				{
					RunID:      runID,
					MeasuredAt: time.Date(2026, time.April, 2, 10, 0, 0, 0, time.Local),
					Phase:      "005_Go-Binary-Trees",
					Metrics:    map[string]int64{"cpu_time_powermetrics_vm-docker_vm-ns": 47560725453},
				},
				{
					RunID:      runID,
					MeasuredAt: time.Date(2026, time.April, 2, 10, 0, 0, 0, time.Local),
					Phase:      "009_python-regex-redux",
					Metrics:    map[string]int64{"gpu_carbon_powermetrics_component-component-ug": 13},
				},
			},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSVByID(context.Background(), &out, runID, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("ExportMeasurementsCSVByID() error = %v", err)
	}

	records, err := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()
	if err != nil {
		t.Fatalf("csv read error = %v", err)
	}

	want := [][]string{
		{
			"run_id",
			"measured_at",
			"language",
			"benchmark",
			"cpu_time_powermetrics_vm-docker_vm-ns",
			"gpu_carbon_powermetrics_component-component-ug",
		},
		{
			"run-1",
			"2026-04-02 10:00:00",
			"go",
			"binary-trees",
			"47560725453",
			"",
		},
		{
			"run-1",
			"2026-04-02 10:00:00",
			"python",
			"regex-redux",
			"",
			"13",
		},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("csv records mismatch:\n got=%#v\nwant=%#v", records, want)
	}

	wantByIDCalls := []byIDCall{
		{runID: runID, filter: entity.TimeRangeFilter{}},
	}
	if !reflect.DeepEqual(provider.getByIDCalls, wantByIDCalls) {
		t.Fatalf("get by ID calls mismatch:\n got=%#v\nwant=%#v", provider.getByIDCalls, wantByIDCalls)
	}
}

func TestExportMeasurementsCSVByID_GetMetricKeysError(t *testing.T) {
	provider := &fakePhaseMetricsProvider{
		metricKeysErr: errors.New("boom"),
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSVByID(context.Background(), &out, "run-1", entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportMeasurementsCSVByID_GetPhaseMetricsByIDError(t *testing.T) {
	const runID = "run-1"
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		phaseMetricsByIDErr: map[string]error{
			runID: errors.New("load failed"),
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSVByID(context.Background(), &out, runID, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportMeasurementsCSVByID_ParseError(t *testing.T) {
	const runID = "run-1"
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		phaseMetricsByID: map[string][]entity.PhaseMetrics{
			runID: {
				{
					RunID:   runID,
					Phase:   "invalid-phase",
					Metrics: map[string]int64{"k": 1},
				},
			},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSVByID(context.Background(), &out, runID, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportMeasurementsCSVByID_UnknownLanguageOrBenchmark_LogsAndContinues(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "error_logs.txt")
	const runID = "run-1"
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		phaseMetricsByID: map[string][]entity.PhaseMetrics{
			runID: {
				{
					RunID:   runID,
					Phase:   "005_Go-Binary-Trees",
					Metrics: map[string]int64{"k": 1},
				},
				{
					RunID:   runID,
					Phase:   "005_Pascal-Binary-Trees",
					Metrics: map[string]int64{"k": 2},
				},
				{
					RunID:   runID,
					Phase:   "005_Go-Binary-Treez",
					Metrics: map[string]int64{"k": 3},
				},
			},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))
	exporterService.errorLogPath = logPath

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSVByID(context.Background(), &out, runID, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("ExportMeasurementsCSVByID() error = %v", err)
	}

	records, err := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()
	if err != nil {
		t.Fatalf("csv read error = %v", err)
	}

	want := [][]string{
		{"run_id", "measured_at", "language", "benchmark", "k"},
		{"run-1", "", "go", "binary-trees", "1"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("csv records mismatch:\n got=%#v\nwant=%#v", records, want)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logData)
	if !strings.Contains(logText, `phase="005_Pascal-Binary-Trees"`) {
		t.Fatalf("expected unknown language log entry, got %q", logText)
	}
	if !strings.Contains(logText, `phase="005_Go-Binary-Treez"`) {
		t.Fatalf("expected unknown benchmark log entry, got %q", logText)
	}
}

func TestExportMeasurementsCSVByID_InvalidArguments(t *testing.T) {
	testCases := []struct {
		name    string
		writer  io.Writer
		service *ExporterService
		runID   string
	}{
		{
			name:    "nil writer",
			writer:  nil,
			service: withTempErrorLogPath(t, NewExporterService(NewParserService(), &fakePhaseMetricsProvider{metricKeys: []string{"k"}})),
			runID:   "run-1",
		},
		{
			name:    "nil provider",
			writer:  &bytes.Buffer{},
			service: withTempErrorLogPath(t, NewExporterService(NewParserService(), nil)),
			runID:   "run-1",
		},
		{
			name:    "empty run id",
			writer:  &bytes.Buffer{},
			service: withTempErrorLogPath(t, NewExporterService(NewParserService(), &fakePhaseMetricsProvider{metricKeys: []string{"k"}})),
			runID:   "",
		},
		{
			name:    "blank run id",
			writer:  &bytes.Buffer{},
			service: withTempErrorLogPath(t, NewExporterService(NewParserService(), &fakePhaseMetricsProvider{metricKeys: []string{"k"}})),
			runID:   "   ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.service.ExportMeasurementsCSVByID(context.Background(), tc.writer, tc.runID, entity.TimeRangeFilter{})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestExportMeasurementsCSVByID_PropagatesDateRangeToProvider(t *testing.T) {
	const runID = "run-1"
	provider := &fakePhaseMetricsProvider{
		metricKeys: []string{"k"},
		phaseMetricsByID: map[string][]entity.PhaseMetrics{
			runID: []entity.PhaseMetrics{},
		},
	}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	from := time.Date(2026, time.April, 1, 10, 0, 0, 0, time.Local)
	to := time.Date(2026, time.April, 1, 11, 0, 0, 0, time.Local)

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSVByID(context.Background(), &out, runID, entity.TimeRangeFilter{From: &from, To: &to})
	if err != nil {
		t.Fatalf("ExportMeasurementsCSVByID() error = %v", err)
	}
	wantCalls := []byIDCall{
		{runID: runID, filter: entity.TimeRangeFilter{From: &from, To: &to}},
	}
	if len(provider.getByIDCalls) != 1 {
		t.Fatalf("expected one by-id provider call, got %#v", provider.getByIDCalls)
	}
	if !reflect.DeepEqual(provider.getByIDCalls, wantCalls) {
		t.Fatalf("unexpected by-id calls: got=%#v want=%#v", provider.getByIDCalls, wantCalls)
	}
}

func TestOpenErrorLogWriter_CreatesParentDirectory(t *testing.T) {
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), &fakePhaseMetricsProvider{}))
	exporterService.errorLogPath = filepath.Join(t.TempDir(), "nested", "logs", "error_logs.txt")

	logWriter, err := exporterService.openErrorLogWriter()
	if err != nil {
		t.Fatalf("openErrorLogWriter() error = %v", err)
	}
	defer logWriter.Close()

	if _, err := os.Stat(exporterService.errorLogPath); err != nil {
		t.Fatalf("expected log file to exist, got stat error: %v", err)
	}
}

func TestOpenErrorLogWriter_UsesDefaultConstantWhenBlank(t *testing.T) {
	exporterService := NewExporterService(NewParserService(), &fakePhaseMetricsProvider{})
	exporterService.errorLogPath = "   "

	logWriter, err := exporterService.openErrorLogWriter()
	if err != nil {
		t.Fatalf("openErrorLogWriter() error = %v", err)
	}
	defer logWriter.Close()

	if _, err := os.Stat(constant.DefaultErrorLogPath); err != nil {
		t.Fatalf("expected default error log file to exist, got stat error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(constant.DefaultErrorLogPath)
		_ = os.Remove(filepath.Dir(constant.DefaultErrorLogPath))
	})
}

func TestIsUnknownDimensionError_UsesSentinelErrors(t *testing.T) {
	t.Parallel()

	if !isUnknownDimensionError(fmt.Errorf("wrapped: %w", constant.ErrUnknownProgrammingLanguage)) {
		t.Fatalf("expected unknown language sentinel to be recognized")
	}
	if !isUnknownDimensionError(fmt.Errorf("wrapped: %w", constant.ErrUnknownBenchmark)) {
		t.Fatalf("expected unknown benchmark sentinel to be recognized")
	}
	if isUnknownDimensionError(errors.New("unknown programming language: \"pascal\"")) {
		t.Fatalf("expected plain string error not to match sentinel-based detection")
	}
}

func TestExportMeasurementsCSV_InvalidDateRange(t *testing.T) {
	provider := &fakePhaseMetricsProvider{metricKeys: []string{"k"}}
	exporterService := withTempErrorLogPath(t, NewExporterService(NewParserService(), provider))

	from := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.Local)

	var out bytes.Buffer
	if err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 1, entity.TimeRangeFilter{From: &from}); err == nil {
		t.Fatalf("expected partial bounds error")
	}
	if err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 1, entity.TimeRangeFilter{From: &from, To: &to}); err == nil {
		t.Fatalf("expected inverted bounds error")
	}
}
