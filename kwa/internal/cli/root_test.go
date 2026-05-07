package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	appexport "mthesis/kwa/internal/app/export"
	appmeasure "mthesis/kwa/internal/app/measure"
	"mthesis/kwa/internal/constant"
)

func TestBatchCommandDispatchesRequest(t *testing.T) {
	t.Parallel()

	var got appexport.Request
	deps := rootDependencies{
		execute: func(_ context.Context, req appexport.Request) error {
			got = req
			return nil
		},
		executeMeasure: func(context.Context, appmeasure.Request) error { return nil },
		runTUI:         func(context.Context, executeRequestFunc, executeMeasureFunc, io.Writer, io.Writer) error { return nil },
	}

	cmd := newRootCmd(deps)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"batch", "--batch-size", "250", "--out", "tmp/export.csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute batch command: %v", err)
	}

	if got.Mode != constant.ExportModeBatch {
		t.Fatalf("mode = %q, want %q", got.Mode, constant.ExportModeBatch)
	}
	if got.BatchSize != 250 {
		t.Fatalf("batch size = %d, want 250", got.BatchSize)
	}
	if got.OutPath != "tmp/export.csv" {
		t.Fatalf("out path = %q, want %q", got.OutPath, "tmp/export.csv")
	}
	if got.TimeRange.From != nil || got.TimeRange.To != nil {
		t.Fatalf("unexpected date range: from=%v to=%v", got.TimeRange.From, got.TimeRange.To)
	}
	if !strings.Contains(out.String(), "export finished: tmp/export.csv") {
		t.Fatalf("expected success output, got %q", out.String())
	}
}

func TestBatchCommandUsesDefaults(t *testing.T) {
	t.Parallel()

	var got appexport.Request
	deps := rootDependencies{
		execute: func(_ context.Context, req appexport.Request) error {
			got = req
			return nil
		},
		executeMeasure: func(context.Context, appmeasure.Request) error { return nil },
		runTUI:         func(context.Context, executeRequestFunc, executeMeasureFunc, io.Writer, io.Writer) error { return nil },
	}

	cmd := newRootCmd(deps)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"batch"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute batch command with defaults: %v", err)
	}

	if got.BatchSize != constant.DefaultBatchSize {
		t.Fatalf("batch size default = %d, want %d", got.BatchSize, constant.DefaultBatchSize)
	}
	if got.OutPath != constant.DefaultOutPath {
		t.Fatalf("out path default = %q, want %q", got.OutPath, constant.DefaultOutPath)
	}
	if got.TimeRange.From != nil || got.TimeRange.To != nil {
		t.Fatalf("unexpected date range defaults: from=%v to=%v", got.TimeRange.From, got.TimeRange.To)
	}
}

func TestByIDRequiresRunID(t *testing.T) {
	t.Parallel()

	deps := rootDependencies{
		execute:        func(_ context.Context, _ appexport.Request) error { return nil },
		executeMeasure: func(context.Context, appmeasure.Request) error { return nil },
		runTUI:         func(context.Context, executeRequestFunc, executeMeasureFunc, io.Writer, io.Writer) error { return nil },
	}

	cmd := newRootCmd(deps)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"by-id"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected required run-id error, got nil")
	}
	if !strings.Contains(err.Error(), "required flag(s) \"run-id\" not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestByIDAliasDispatchesRequest(t *testing.T) {
	t.Parallel()

	var got appexport.Request
	deps := rootDependencies{
		execute: func(_ context.Context, req appexport.Request) error {
			got = req
			return nil
		},
		executeMeasure: func(context.Context, appmeasure.Request) error { return nil },
		runTUI:         func(context.Context, executeRequestFunc, executeMeasureFunc, io.Writer, io.Writer) error { return nil },
	}

	cmd := newRootCmd(deps)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"byID", "--run-id", "run-42"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute byID alias: %v", err)
	}

	if got.Mode != constant.ExportModeByID {
		t.Fatalf("mode = %q, want %q", got.Mode, constant.ExportModeByID)
	}
	if got.RunID != "run-42" {
		t.Fatalf("run ID = %q, want %q", got.RunID, "run-42")
	}
	if got.OutPath != constant.DefaultOutPath {
		t.Fatalf("out path default = %q, want %q", got.OutPath, constant.DefaultOutPath)
	}
	if got.TimeRange.From != nil || got.TimeRange.To != nil {
		t.Fatalf("unexpected date range defaults: from=%v to=%v", got.TimeRange.From, got.TimeRange.To)
	}
}

func TestRootLaunchesTUI(t *testing.T) {
	t.Parallel()

	var launched bool

	deps := rootDependencies{
		execute:        func(_ context.Context, _ appexport.Request) error { return nil },
		executeMeasure: func(context.Context, appmeasure.Request) error { return nil },
		runTUI: func(_ context.Context, _ executeRequestFunc, _ executeMeasureFunc, _ io.Writer, _ io.Writer) error {
			launched = true
			return nil
		},
	}

	cmd := newRootCmd(deps)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute root command: %v", err)
	}

	if !launched {
		t.Fatalf("expected TUI launcher to run")
	}
}

func TestBatchCommandParsesDateRange(t *testing.T) {
	t.Parallel()

	var got appexport.Request
	deps := rootDependencies{
		execute: func(_ context.Context, req appexport.Request) error {
			got = req
			return nil
		},
		executeMeasure: func(context.Context, appmeasure.Request) error { return nil },
		runTUI:         func(context.Context, executeRequestFunc, executeMeasureFunc, io.Writer, io.Writer) error { return nil },
	}

	cmd := newRootCmd(deps)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"batch", "--from", "2026-04-01", "--to", "2026-04-02 12:30:45"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute batch command with date range: %v", err)
	}

	if got.TimeRange.From == nil || got.TimeRange.To == nil {
		t.Fatalf("expected parsed date range, got from=%v to=%v", got.TimeRange.From, got.TimeRange.To)
	}
	if got.TimeRange.From.Format(appexport.TimestampLayout) != "2026-04-01 00:00:00" {
		t.Fatalf("from = %q, want %q", got.TimeRange.From.Format(appexport.TimestampLayout), "2026-04-01 00:00:00")
	}
	if got.TimeRange.To.Format(appexport.TimestampLayout) != "2026-04-02 12:30:45" {
		t.Fatalf("to = %q, want %q", got.TimeRange.To.Format(appexport.TimestampLayout), "2026-04-02 12:30:45")
	}
}

func TestByIDCommandRejectsTimestampFlags(t *testing.T) {
	t.Parallel()

	deps := rootDependencies{
		execute:        func(_ context.Context, _ appexport.Request) error { return nil },
		executeMeasure: func(context.Context, appmeasure.Request) error { return nil },
		runTUI:         func(context.Context, executeRequestFunc, executeMeasureFunc, io.Writer, io.Writer) error { return nil },
	}

	cmd := newRootCmd(deps)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"by-id", "--run-id", "run-1", "--from", "2026-04-01"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown timestamp flag error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --from") {
		t.Fatalf("unexpected error: %v", err)
	}
}
