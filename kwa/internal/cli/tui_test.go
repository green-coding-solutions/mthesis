package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuEnterStartsBatchForm(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, ExportRequest) error { return nil })
	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := nextModel.(model)

	if updated.state != screenForm {
		t.Fatalf("state = %v, want %v", updated.state, screenForm)
	}
	if updated.formMode != ExportModeBatch {
		t.Fatalf("form mode = %q, want %q", updated.formMode, ExportModeBatch)
	}
}

func TestMenuDownEnterStartsByIDForm(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, ExportRequest) error { return nil })
	moved, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	nextModel, _ := moved.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := nextModel.(model)

	if updated.formMode != ExportModeByID {
		t.Fatalf("form mode = %q, want %q", updated.formMode, ExportModeByID)
	}
	if updated.state != screenForm {
		t.Fatalf("state = %v, want %v", updated.state, screenForm)
	}
}

func TestMenuViewContainsBatLogo(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, ExportRequest) error { return nil })
	view := m.View()
	if !strings.Contains(view, "_.''._'--(.)--'_.''._") {
		t.Fatalf("expected menu view to contain bat logo")
	}
}

func TestBuildRequestFromFormValidationAndDefaults(t *testing.T) {
	t.Parallel()

	batchModel := newModel(context.Background(), func(context.Context, ExportRequest) error { return nil })
	batchModel.initForm(ExportModeBatch)
	batchModel.fields[0].value = ""
	batchModel.fields[1].value = "custom-file"

	req, err := batchModel.buildRequestFromForm()
	if err != nil {
		t.Fatalf("buildRequestFromForm() batch: %v", err)
	}
	if req.BatchSize != DefaultBatchSize {
		t.Fatalf("batch size = %d, want %d", req.BatchSize, DefaultBatchSize)
	}
	if req.OutPath != "results/custom-file.csv" {
		t.Fatalf("out path = %q, want %q", req.OutPath, "results/custom-file.csv")
	}

	batchModel.fields[0].value = "invalid"
	if _, err := batchModel.buildRequestFromForm(); err == nil {
		t.Fatalf("expected invalid batch size error, got nil")
	}

	byIDModel := newModel(context.Background(), func(context.Context, ExportRequest) error { return nil })
	byIDModel.initForm(ExportModeByID)
	byIDModel.fields[0].value = ""
	if _, err := byIDModel.buildRequestFromForm(); err == nil {
		t.Fatalf("expected missing run ID error, got nil")
	}

	byIDModel.fields[0].value = "run-99"
	byIDModel.fields[1].value = "exports/by-id"
	byIDReq, err := byIDModel.buildRequestFromForm()
	if err != nil {
		t.Fatalf("buildRequestFromForm() by-id: %v", err)
	}
	if byIDReq.OutPath != "exports/by-id.csv" {
		t.Fatalf("out path = %q, want %q", byIDReq.OutPath, "exports/by-id.csv")
	}
}

func TestResultViewContainsOutputPath(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, ExportRequest) error { return nil })
	m.state = screenRunning

	nextModel, _ := m.Update(exportDoneMsg{path: "results/measurements.csv"})
	updated := nextModel.(model)

	if updated.state != screenResult {
		t.Fatalf("state = %v, want %v", updated.state, screenResult)
	}
	if !strings.Contains(updated.View(), "results/measurements.csv") {
		t.Fatalf("expected result view to contain output path")
	}
}

func TestInteractiveReturnsFinalError(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, ExportRequest) error { return nil })
	m.state = screenResult
	expectedErr := errors.New("export failed")
	m.finalErr = expectedErr

	if m.finalErr == nil || m.finalErr.Error() != expectedErr.Error() {
		t.Fatalf("final error mismatch")
	}
}
