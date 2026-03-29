package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"reflect"
	"testing"

	"mthesis/exporter/internal/entity"
)

type batchCall struct {
	limit  int
	offset int
}

type fakePhaseMetricsProvider struct {
	metricKeys      []string
	metricKeysErr   error
	batches         [][]entity.PhaseMetrics
	batchErrAtCall  map[int]error
	getBatchCalls   []batchCall
	getBatchCallIdx int
}

func (f *fakePhaseMetricsProvider) GetMetricKeys(_ context.Context) ([]string, error) {
	if f.metricKeysErr != nil {
		return nil, f.metricKeysErr
	}
	return f.metricKeys, nil
}

func (f *fakePhaseMetricsProvider) GetPhaseMetricsBatch(_ context.Context, limit, offset int) ([]entity.PhaseMetrics, error) {
	f.getBatchCalls = append(f.getBatchCalls, batchCall{limit: limit, offset: offset})
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

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 1)
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
					RunID:   "run-1",
					Phase:   "005_Go-Binary-Trees",
					Metrics: map[string]int64{"cpu_time_powermetrics_vm-docker_vm-ns": 47560725453},
				},
				{
					RunID:   "run-1",
					Phase:   "009_python-regex-redux",
					Metrics: map[string]int64{"gpu_carbon_powermetrics_component-component-ug": 13},
				},
			},
			{
				{
					RunID:   "run-2",
					Phase:   "006_Go-Fasta",
					Metrics: map[string]int64{},
				},
			},
			{},
		},
	}
	exporterService := NewExporterService(NewParserService(), provider)

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2)
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
			"language",
			"benchmark",
			"cpu_time_powermetrics_vm-docker_vm-ns",
			"gpu_carbon_powermetrics_component-component-ug",
		},
		{
			"run-1",
			"go",
			"binary-trees",
			"47560725453",
			"",
		},
		{
			"run-1",
			"python",
			"regex-redux",
			"",
			"13",
		},
		{
			"run-2",
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
	exporterService := NewExporterService(NewParserService(), provider)

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2)
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
	exporterService := NewExporterService(NewParserService(), provider)

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2)
	if err == nil {
		t.Fatalf("expected error, got nil")
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
	exporterService := NewExporterService(NewParserService(), provider)

	var out bytes.Buffer
	err := exporterService.ExportMeasurementsCSV(context.Background(), &out, 2)
	if err == nil {
		t.Fatalf("expected error, got nil")
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
			service:   NewExporterService(NewParserService(), &fakePhaseMetricsProvider{metricKeys: []string{"k"}}),
			batchSize: 1,
		},
		{
			name:      "nil provider",
			writer:    &bytes.Buffer{},
			service:   NewExporterService(NewParserService(), nil),
			batchSize: 1,
		},
		{
			name:      "invalid batch size",
			writer:    &bytes.Buffer{},
			service:   NewExporterService(NewParserService(), &fakePhaseMetricsProvider{metricKeys: []string{"k"}}),
			batchSize: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.service.ExportMeasurementsCSV(context.Background(), tc.writer, tc.batchSize)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
