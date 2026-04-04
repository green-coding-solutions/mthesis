package service

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/entity"
)

func TestParseMeasurementFromPhase(t *testing.T) {
	parserService := NewParserService()

	tests := []struct {
		name           string
		input          entity.PhaseMetrics
		wantRunID      string
		wantMeasuredAt time.Time
		wantLanguage   string
		wantBenchmark  string
		wantMetrics    map[string]int64
		wantErr        bool
	}{
		{
			name:           "parses go binary trees",
			input:          entity.PhaseMetrics{RunID: "run-1", MeasuredAt: time.Date(2026, time.April, 1, 12, 0, 0, 0, time.Local), Phase: "005_Go-Binary-Trees", Metrics: map[string]int64{"cpu_time_powermetrics_vm-docker_vm-ns": 47560725453}},
			wantRunID:      "run-1",
			wantMeasuredAt: time.Date(2026, time.April, 1, 12, 0, 0, 0, time.Local),
			wantLanguage:   "go",
			wantBenchmark:  "binary-trees",
			wantMetrics:    map[string]int64{"cpu_time_powermetrics_vm-docker_vm-ns": 47560725453},
			wantErr:        false,
		},
		{
			name:           "parses lowercase with multi-word benchmark",
			input:          entity.PhaseMetrics{RunID: "run-2", MeasuredAt: time.Date(2026, time.April, 2, 13, 0, 0, 0, time.Local), Phase: "009_python-regex-redux", Metrics: map[string]int64{"gpu_carbon_powermetrics_component-component-ug": 13}},
			wantRunID:      "run-2",
			wantMeasuredAt: time.Date(2026, time.April, 2, 13, 0, 0, 0, time.Local),
			wantLanguage:   "python",
			wantBenchmark:  "regex-redux",
			wantMetrics:    map[string]int64{"gpu_carbon_powermetrics_component-component-ug": 13},
			wantErr:        false,
		},
		{
			name:    "fails on invalid phase format",
			input:   entity.PhaseMetrics{Phase: "005GoBinaryTrees", Metrics: map[string]int64{"x": 1}},
			wantErr: true,
		},
		{
			name:    "fails on unknown language",
			input:   entity.PhaseMetrics{Phase: "005_Goo-Binary-Trees", Metrics: map[string]int64{"x": 1}},
			wantErr: true,
		},
		{
			name:          "defaults nil metrics to empty map",
			input:         entity.PhaseMetrics{RunID: "run-3", Phase: "005_Go-Binary-Trees"},
			wantRunID:     "run-3",
			wantLanguage:  "go",
			wantBenchmark: "binary-trees",
			wantMetrics:   map[string]int64{},
			wantErr:       false,
		},
		{
			name:    "fails on unknown benchmark",
			input:   entity.PhaseMetrics{Phase: "005_Go-Binary-Treez", Metrics: map[string]int64{"x": 1}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parserService.ParseMeasurementFromPhase(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.RunID != tt.wantRunID {
				t.Fatalf("run_id mismatch: got %q want %q", got.RunID, tt.wantRunID)
			}
			if !got.MeasuredAt.Equal(tt.wantMeasuredAt) {
				t.Fatalf("created_at mismatch: got %v want %v", got.MeasuredAt, tt.wantMeasuredAt)
			}
			if got.Language != tt.wantLanguage {
				t.Fatalf("language mismatch: got %q want %q", got.Language, tt.wantLanguage)
			}
			if got.Benchmark != tt.wantBenchmark {
				t.Fatalf("benchmark mismatch: got %q want %q", got.Benchmark, tt.wantBenchmark)
			}
			if got.Metrics == nil {
				t.Fatalf("metrics must not be nil")
			}
			if !reflect.DeepEqual(got.Metrics, tt.wantMetrics) {
				t.Fatalf("metrics mismatch: got %#v want %#v", got.Metrics, tt.wantMetrics)
			}
		})
	}
}

func TestParseMeasurementFromPhase_ClonesMetricsMap(t *testing.T) {
	parserService := NewParserService()

	inputMetrics := map[string]int64{
		"cpu_time_powermetrics_vm-docker_vm-ns": 47560725453,
	}
	input := entity.PhaseMetrics{
		RunID:   "run-clone",
		Phase:   "005_Go-Binary-Trees",
		Metrics: inputMetrics,
	}

	got, err := parserService.ParseMeasurementFromPhase(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputMetrics["cpu_time_powermetrics_vm-docker_vm-ns"] = 1
	inputMetrics["new_metric"] = 99

	if got.Metrics["cpu_time_powermetrics_vm-docker_vm-ns"] != 47560725453 {
		t.Fatalf("metrics map should be cloned; got %d", got.Metrics["cpu_time_powermetrics_vm-docker_vm-ns"])
	}
	if _, ok := got.Metrics["new_metric"]; ok {
		t.Fatalf("metrics map should be cloned; unexpected key found")
	}
	if got.RunID != "run-clone" {
		t.Fatalf("run_id mismatch: got %q want %q", got.RunID, "run-clone")
	}
}

func TestParseMeasurementFromPhase_UnknownDimensionErrorsAreTyped(t *testing.T) {
	parserService := NewParserService()

	_, err := parserService.ParseMeasurementFromPhase(entity.PhaseMetrics{
		Phase: "005_Pascal-Binary-Trees",
	})
	if err == nil {
		t.Fatalf("expected unknown language error, got nil")
	}
	if !errors.Is(err, constant.ErrUnknownProgrammingLanguage) {
		t.Fatalf("expected ErrUnknownProgrammingLanguage, got %v", err)
	}

	_, err = parserService.ParseMeasurementFromPhase(entity.PhaseMetrics{
		Phase: "005_Go-Binary-Treez",
	})
	if err == nil {
		t.Fatalf("expected unknown benchmark error, got nil")
	}
	if !errors.Is(err, constant.ErrUnknownBenchmark) {
		t.Fatalf("expected ErrUnknownBenchmark, got %v", err)
	}
}
