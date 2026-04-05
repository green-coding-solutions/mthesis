package measure

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appexport "mthesis/kwa/internal/app/export"
	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/entity"
	"mthesis/kwa/internal/service"
)

type fakeMeasureRunner struct {
	lastReq service.MeasureRunRequest
	filter  entity.TimeRangeFilter
	err     error
}

// Run captures the request and returns the configured response for test assertions.
func (f *fakeMeasureRunner) Run(_ context.Context, req service.MeasureRunRequest) (entity.TimeRangeFilter, error) {
	f.lastReq = service.MeasureRunRequest{
		Languages:  append([]string(nil), req.Languages...),
		Benchmarks: append([]string(nil), req.Benchmarks...),
		Iterations: req.Iterations,
	}

	if f.err != nil {
		return entity.TimeRangeFilter{}, f.err
	}

	return f.filter.Clone(), nil
}

func TestExecuteSuccessRunsMeasureRunnerThenExport(t *testing.T) {
	t.Parallel()

	var gotExportReq appexport.Request
	from := time.Date(2026, time.April, 4, 22, 26, 40, 0, time.UTC)
	to := time.Date(2026, time.April, 4, 22, 29, 15, 0, time.UTC)
	runner := &fakeMeasureRunner{
		filter: entity.TimeRangeFilter{From: &from, To: &to},
	}

	executor := NewExecutorWithDeps(Dependencies{
		MeasureRunner: runner,
		ExecuteExport: func(_ context.Context, req appexport.Request) error {
			gotExportReq = req
			return nil
		},
	})

	err := executor.Execute(context.Background(), Request{
		Languages:  []string{"go", "c"},
		Benchmarks: []string{"binary-trees", "mandelbrot"},
		Iterations: 3,
		OutPath:    "results/v2.csv",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if runner.lastReq.Iterations != 3 {
		t.Fatalf("measure iterations = %d, want 3", runner.lastReq.Iterations)
	}
	if strings.Join(runner.lastReq.Languages, ",") != "go,c" {
		t.Fatalf("measure languages = %#v, want %#v", runner.lastReq.Languages, []string{"go", "c"})
	}
	if strings.Join(runner.lastReq.Benchmarks, ",") != "binary-trees,mandelbrot" {
		t.Fatalf("measure benchmarks = %#v, want %#v", runner.lastReq.Benchmarks, []string{"binary-trees", "mandelbrot"})
	}

	if gotExportReq.Mode != constant.ExportModeBatch {
		t.Fatalf("mode = %q, want %q", gotExportReq.Mode, constant.ExportModeBatch)
	}
	if gotExportReq.BatchSize != constant.DefaultBatchSize {
		t.Fatalf("batch size = %d, want %d", gotExportReq.BatchSize, constant.DefaultBatchSize)
	}
	if gotExportReq.OutPath != "results/v2.csv" {
		t.Fatalf("out path = %q, want %q", gotExportReq.OutPath, "results/v2.csv")
	}
	if gotExportReq.TimeRange.From == nil || gotExportReq.TimeRange.To == nil {
		t.Fatalf("expected non-nil interval")
	}
	if !gotExportReq.TimeRange.From.Equal(from) || !gotExportReq.TimeRange.To.Equal(to) {
		t.Fatalf("unexpected interval: from=%v to=%v", gotExportReq.TimeRange.From, gotExportReq.TimeRange.To)
	}
}

func TestExecuteStopsWhenMeasureRunnerFails(t *testing.T) {
	t.Parallel()

	exportCalled := false
	runner := &fakeMeasureRunner{err: errors.New("run measure script: boom")}

	executor := NewExecutorWithDeps(Dependencies{
		MeasureRunner: runner,
		ExecuteExport: func(context.Context, appexport.Request) error {
			exportCalled = true
			return nil
		},
	})

	err := executor.Execute(context.Background(), Request{
		Languages:  []string{"go"},
		Benchmarks: []string{"binary-trees"},
		Iterations: 1,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "run measure script") {
		t.Fatalf("unexpected error: %v", err)
	}
	if exportCalled {
		t.Fatalf("export should not be called when measure runner fails")
	}
}

func TestExecuteReturnsExportFailure(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, time.April, 4, 22, 26, 40, 0, time.UTC)
	to := time.Date(2026, time.April, 4, 22, 29, 15, 0, time.UTC)
	runner := &fakeMeasureRunner{
		filter: entity.TimeRangeFilter{From: &from, To: &to},
	}

	executor := NewExecutorWithDeps(Dependencies{
		MeasureRunner: runner,
		ExecuteExport: func(context.Context, appexport.Request) error {
			return errors.New("export failed")
		},
	})

	err := executor.Execute(context.Background(), Request{
		Languages:  []string{"go"},
		Benchmarks: []string{"binary-trees"},
		Iterations: 1,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "export measurements after measure run") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteFailsOnInvalidFilterFromMeasureRunner(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, time.April, 4, 22, 29, 15, 0, time.UTC)
	to := time.Date(2026, time.April, 4, 22, 26, 40, 0, time.UTC)
	runner := &fakeMeasureRunner{
		filter: entity.TimeRangeFilter{From: &from, To: &to},
	}

	executor := NewExecutorWithDeps(Dependencies{
		MeasureRunner: runner,
		ExecuteExport: func(context.Context, appexport.Request) error { return nil },
	})

	err := executor.Execute(context.Background(), Request{
		Languages:  []string{"go"},
		Benchmarks: []string{"binary-trees"},
		Iterations: 1,
	})
	if err == nil {
		t.Fatalf("expected invalid interval error")
	}
	if !strings.Contains(err.Error(), "invalid measure interval") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAndNormalizeRequestRejectsUnsupportedValues(t *testing.T) {
	t.Parallel()

	_, err := validateAndNormalizeRequest(Request{
		Languages:  []string{"pascal"},
		Benchmarks: []string{"binary-trees"},
		Iterations: 1,
	})
	if err == nil {
		t.Fatalf("expected unsupported language error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported measure language") {
		t.Fatalf("unexpected error: %v", err)
	}
}
