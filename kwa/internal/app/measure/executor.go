package measure

import (
	"context"
	"fmt"
	"strings"

	appexport "mthesis/kwa/internal/app/export"
	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/service"
)

// Dependencies defines overridable constructors and callbacks used to keep
// app-layer measure orchestration testable.
type Dependencies struct {
	ExecuteExport func(context.Context, appexport.Request) error
	MeasureRunner service.MeasureRunner
}

// Executor validates measure inputs, runs the measure service, and delegates
// CSV extraction for the captured interval to the export executor.
type Executor struct {
	deps Dependencies
}

// NewExecutor returns the production app-layer measure executor and wires the
// provided export callback used after measure execution completes.
func NewExecutor(executeExport func(context.Context, appexport.Request) error) *Executor {
	return NewExecutorWithDeps(Dependencies{ExecuteExport: executeExport})
}

// NewExecutorWithDeps returns an app-layer measure executor with dependency
// overrides and falls back to production dependencies for missing values.
func NewExecutorWithDeps(deps Dependencies) *Executor {
	if deps.ExecuteExport == nil {
		deps.ExecuteExport = appexport.NewExecutor().Execute
	}
	if deps.MeasureRunner == nil {
		deps.MeasureRunner = service.NewMeasureService()
	}

	return &Executor{deps: deps}
}

// Execute validates and normalizes one measure request, runs the service-layer
// measurement workflow, and exports rows for the returned time interval.
func (e *Executor) Execute(ctx context.Context, req Request) error {
	normalizedReq, err := validateAndNormalizeRequest(req)
	if err != nil {
		return err
	}

	filter, err := e.deps.MeasureRunner.Run(ctx, service.MeasureRunRequest{
		Languages:  append([]string(nil), normalizedReq.Languages...),
		Benchmarks: append([]string(nil), normalizedReq.Benchmarks...),
		Iterations: normalizedReq.Iterations,
	})
	if err != nil {
		return err
	}
	if err := filter.Validate(); err != nil {
		return fmt.Errorf("invalid measure interval: %w", err)
	}
	filter = filter.Clone()

	exportReq := appexport.Request{
		Mode:      constant.ExportModeBatch,
		BatchSize: constant.DefaultBatchSize,
		OutPath:   normalizedReq.OutPath,
		TimeRange: filter,
	}

	if err := e.deps.ExecuteExport(ctx, exportReq); err != nil {
		return fmt.Errorf("export measurements after measure run: %w", err)
	}

	return nil
}

// validateAndNormalizeRequest enforces measure request invariants, keeps the
// incoming order, and applies output-path defaults.
func validateAndNormalizeRequest(req Request) (Request, error) {
	normalized := Request{
		Languages:  make([]string, 0, len(req.Languages)),
		Benchmarks: make([]string, 0, len(req.Benchmarks)),
		Iterations: req.Iterations,
		OutPath:    strings.TrimSpace(req.OutPath),
	}

	if normalized.OutPath == "" {
		normalized.OutPath = constant.DefaultOutPath
	}
	if normalized.Iterations <= 0 {
		return Request{}, fmt.Errorf("iterations must be greater than zero")
	}

	seenLanguages := make(map[string]struct{}, len(req.Languages))
	for _, language := range req.Languages {
		trimmed := strings.TrimSpace(language)
		if trimmed == "" {
			return Request{}, fmt.Errorf("languages must not contain empty values")
		}
		if !constant.IsSupportedMeasureLanguage(trimmed) {
			return Request{}, fmt.Errorf("unsupported measure language %q", trimmed)
		}
		if _, seen := seenLanguages[trimmed]; seen {
			continue
		}
		seenLanguages[trimmed] = struct{}{}
		normalized.Languages = append(normalized.Languages, trimmed)
	}

	seenBenchmarks := make(map[string]struct{}, len(req.Benchmarks))
	for _, benchmark := range req.Benchmarks {
		trimmed := strings.TrimSpace(benchmark)
		if trimmed == "" {
			return Request{}, fmt.Errorf("benchmarks must not contain empty values")
		}
		if !constant.IsSupportedMeasureBenchmark(trimmed) {
			return Request{}, fmt.Errorf("unsupported measure benchmark %q", trimmed)
		}
		if _, seen := seenBenchmarks[trimmed]; seen {
			continue
		}
		seenBenchmarks[trimmed] = struct{}{}
		normalized.Benchmarks = append(normalized.Benchmarks, trimmed)
	}

	if len(normalized.Languages) == 0 {
		return Request{}, fmt.Errorf("at least one language must be selected")
	}
	if len(normalized.Benchmarks) == 0 {
		return Request{}, fmt.Errorf("at least one benchmark must be selected")
	}

	return normalized, nil
}
