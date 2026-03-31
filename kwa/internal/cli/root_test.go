package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	appexport "mthesis/kwa/internal/app/export"
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
		runTUI: func(context.Context, executeRequestFunc, io.Writer, io.Writer) error { return nil },
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
		runTUI: func(context.Context, executeRequestFunc, io.Writer, io.Writer) error { return nil },
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
}

func TestByIDRequiresRunID(t *testing.T) {
	t.Parallel()

	deps := rootDependencies{
		execute: func(_ context.Context, _ appexport.Request) error { return nil },
		runTUI:  func(context.Context, executeRequestFunc, io.Writer, io.Writer) error { return nil },
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
		runTUI: func(context.Context, executeRequestFunc, io.Writer, io.Writer) error { return nil },
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
}

func TestRootLaunchesTUI(t *testing.T) {
	t.Parallel()

	var launched bool

	deps := rootDependencies{
		execute: func(_ context.Context, _ appexport.Request) error { return nil },
		runTUI: func(_ context.Context, _ executeRequestFunc, _ io.Writer, _ io.Writer) error {
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
