package cli

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	appexport "mthesis/kwa/internal/app/export"
	"mthesis/kwa/internal/constant"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuEnterStartsBatchForm(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := nextModel.(model)

	if updated.state != screenForm {
		t.Fatalf("state = %v, want %v", updated.state, screenForm)
	}
	if updated.formMode != constant.ExportModeBatch {
		t.Fatalf("form mode = %q, want %q", updated.formMode, constant.ExportModeBatch)
	}
}

func TestMenuDownEnterStartsByIDForm(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	moved, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	nextModel, _ := moved.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := nextModel.(model)

	if updated.formMode != constant.ExportModeByID {
		t.Fatalf("form mode = %q, want %q", updated.formMode, constant.ExportModeByID)
	}
	if updated.state != screenForm {
		t.Fatalf("state = %v, want %v", updated.state, screenForm)
	}
}

func TestMenuViewContainsBatLogo(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	view := m.View()
	if !strings.Contains(view, "_.''._'--(.)--'_.''._") {
		t.Fatalf("expected menu view to contain bat logo")
	}
	if !strings.Contains(view, "byID export") {
		t.Fatalf("expected menu view to contain byID export label")
	}
}

func TestBuildRequestFromFormValidationAndDefaults(t *testing.T) {
	t.Parallel()

	batchModel := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	batchModel.initForm(constant.ExportModeBatch)
	batchModel.fields[0].value = ""
	batchModel.fields[1].value = ""
	batchModel.fields[2].value = ""
	batchModel.fields[3].value = "custom-file"

	req, err := batchModel.buildRequestFromForm()
	if err != nil {
		t.Fatalf("buildRequestFromForm() batch: %v", err)
	}
	if req.BatchSize != constant.DefaultBatchSize {
		t.Fatalf("batch size = %d, want %d", req.BatchSize, constant.DefaultBatchSize)
	}
	if req.OutPath != "results/custom-file.csv" {
		t.Fatalf("out path = %q, want %q", req.OutPath, "results/custom-file.csv")
	}
	if req.TimeRange.From != nil || req.TimeRange.To != nil {
		t.Fatalf("expected nil date range defaults, got from=%v to=%v", req.TimeRange.From, req.TimeRange.To)
	}

	batchModel.fields[0].value = "invalid"
	if _, err := batchModel.buildRequestFromForm(); err == nil {
		t.Fatalf("expected invalid batch size error, got nil")
	}
	batchModel.fields[0].value = "100"
	batchModel.fields[1].value = "2026-04-01"
	batchModel.fields[2].value = ""
	if _, err := batchModel.buildRequestFromForm(); err == nil {
		t.Fatalf("expected partial date range error, got nil")
	}

	byIDModel := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	byIDModel.initForm(constant.ExportModeByID)
	byIDModel.fields[0].value = ""
	if _, err := byIDModel.buildRequestFromForm(); err == nil {
		t.Fatalf("expected missing run ID error, got nil")
	}

	byIDModel.fields[0].value = "run-99"
	byIDModel.fields[1].value = "2026-04-01"
	byIDModel.fields[2].value = "2026-04-02 10:00:00"
	byIDModel.fields[3].value = "exports/by-id"
	byIDReq, err := byIDModel.buildRequestFromForm()
	if err != nil {
		t.Fatalf("buildRequestFromForm() by-id: %v", err)
	}
	if byIDReq.OutPath != "exports/by-id.csv" {
		t.Fatalf("out path = %q, want %q", byIDReq.OutPath, "exports/by-id.csv")
	}
	if byIDReq.TimeRange.From == nil || byIDReq.TimeRange.To == nil {
		t.Fatalf("expected parsed date range, got from=%v to=%v", byIDReq.TimeRange.From, byIDReq.TimeRange.To)
	}
	if byIDReq.TimeRange.From.Format(appexport.TimestampLayout) != "2026-04-01 00:00:00" {
		t.Fatalf("from = %q, want %q", byIDReq.TimeRange.From.Format(appexport.TimestampLayout), "2026-04-01 00:00:00")
	}
}

func TestResultViewContainsOutputPath(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
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

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	m.state = screenResult
	expectedErr := errors.New("export failed")
	m.finalErr = expectedErr

	if m.finalErr == nil || m.finalErr.Error() != expectedErr.Error() {
		t.Fatalf("final error mismatch")
	}
}

func TestFormAllowsSpaceInput(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	m.initForm(constant.ExportModeBatch)
	m.focus = 1 // From timestamp field
	m.fields[1].value = "2026-04-01"

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated := nextModel.(model)

	if updated.fields[1].value != "2026-04-01 " {
		t.Fatalf("field value = %q, want %q", updated.fields[1].value, "2026-04-01 ")
	}
}

func TestFormQTypesOnFileNameField(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	m.initForm(constant.ExportModeBatch)
	m.focus = 3
	m.fields[3].value = "measurements"

	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := nextModel.(model)

	if updated.fields[3].value != "measurementsq" {
		t.Fatalf("field value = %q, want %q", updated.fields[3].value, "measurementsq")
	}
	if cmd != nil {
		t.Fatalf("expected nil command for text input, got %T", cmd)
	}
}

func TestFormQTypesOnRunIDField(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	m.initForm(constant.ExportModeByID)
	m.focus = 0
	m.fields[0].value = "abc-123"

	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := nextModel.(model)

	if updated.fields[0].value != "abc-123q" {
		t.Fatalf("field value = %q, want %q", updated.fields[0].value, "abc-123q")
	}
	if cmd != nil {
		t.Fatalf("expected nil command for text input, got %T", cmd)
	}
}

func TestFormQQuitsInNonFileNameField(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	m.initForm(constant.ExportModeBatch)
	m.focus = 1
	m.fields[1].value = "2026-04-01"

	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := nextModel.(model)

	if updated.fields[1].value != "2026-04-01" {
		t.Fatalf("field value = %q, want unchanged", updated.fields[1].value)
	}
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	msg := cmd()
	if !reflect.TypeOf(msg).AssignableTo(reflect.TypeOf(tea.QuitMsg{})) {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestRunningViewShowsSingleProgressLine(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	m.state = screenRunning
	m.runningReq = appexport.Request{
		Mode:    constant.ExportModeBatch,
		OutPath: "results/measurements.csv",
	}

	view := m.viewRunning()
	if got := strings.Count(view, "Export in progress"); got != 1 {
		t.Fatalf("progress line count = %d, want 1\nview:\n%s", got, view)
	}
}

func TestRunningUpdateEmitsSpinnerFollowUpCommand(t *testing.T) {
	t.Parallel()

	m := newModel(context.Background(), func(context.Context, appexport.Request) error { return nil })
	m.state = screenRunning

	// Use the spinner's own tick message to validate update-loop scheduling.
	msg := m.spinner.Tick()
	if msg == nil {
		t.Fatalf("expected spinner tick message, got nil")
	}

	nextModel, cmd := m.Update(msg)
	updated := nextModel.(model)

	if updated.state != screenRunning {
		t.Fatalf("state = %v, want %v", updated.state, screenRunning)
	}
	if cmd == nil {
		t.Fatalf("expected follow-up command for spinner animation, got nil")
	}
}
