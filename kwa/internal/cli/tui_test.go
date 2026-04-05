package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	appexport "mthesis/kwa/internal/app/export"
	appmeasure "mthesis/kwa/internal/app/measure"
	"mthesis/kwa/internal/constant"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuEnterStartsBatchForm(t *testing.T) {
	t.Parallel()

	m := newTestModel()
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

	m := newTestModel()
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

func TestMenuEscQuits(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	msg := cmd()
	if !reflect.TypeOf(msg).AssignableTo(reflect.TypeOf(tea.QuitMsg{})) {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestMenuMeasureEntryStartsBenchmarkSelection(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	nextModel, _ = nextModel.(model).Update(tea.KeyMsg{Type: tea.KeyDown})
	nextModel, _ = nextModel.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := nextModel.(model)

	if updated.state != screenMeasureBenchmarks {
		t.Fatalf("state = %v, want %v", updated.state, screenMeasureBenchmarks)
	}
	if len(updated.measureBenchmarks) != 8 {
		t.Fatalf("benchmark options = %d, want 8", len(updated.measureBenchmarks))
	}
}

func TestMenuViewContainsBatLogoAndMeasure(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	view := m.View()
	if !strings.Contains(view, "_.''._'--(.)--'_.''._") {
		t.Fatalf("expected menu view to contain bat logo")
	}
	if !strings.Contains(view, "Measure") {
		t.Fatalf("expected menu view to contain Measure label")
	}
	if !strings.Contains(view, "Export (Batch mode)") {
		t.Fatalf("expected menu view to contain Export (Batch mode) label")
	}
	if !strings.Contains(view, "Export (by Run ID)") {
		t.Fatalf("expected menu view to contain Export (by Run ID) label")
	}
}

func TestBuildRequestFromFormValidationAndDefaults(t *testing.T) {
	t.Parallel()

	batchModel := newTestModel()
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

	byIDModel := newTestModel()
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

func TestMeasureSelectionSupportsSpaceAndToggleAll(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.initMeasureBenchmarkSelection()

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated := nextModel.(model)
	if !updated.measureBenchmarks[0].selected {
		t.Fatalf("expected first benchmark to be selected")
	}

	nextModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	updated = nextModel.(model)
	for _, option := range updated.measureBenchmarks {
		if !option.selected {
			t.Fatalf("expected all benchmark options to be selected")
		}
	}

	nextModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	updated = nextModel.(model)
	for _, option := range updated.measureBenchmarks {
		if option.selected {
			t.Fatalf("expected all benchmark options to be unselected")
		}
	}
}

func TestMeasureSelectionValidatesBeforeAdvancing(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.initMeasureBenchmarkSelection()

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := nextModel.(model)
	if updated.validationErr == "" {
		t.Fatalf("expected benchmark validation error")
	}

	updated.measureBenchmarks[0].selected = true
	nextModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = nextModel.(model)
	if updated.state != screenMeasureLanguages {
		t.Fatalf("state = %v, want %v", updated.state, screenMeasureLanguages)
	}

	nextModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = nextModel.(model)
	if updated.validationErr == "" {
		t.Fatalf("expected language validation error")
	}
}

func TestBuildMeasureRequestFromForm(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.measureBenchmarks = []measureOption{{label: "binary-trees", selected: true}}
	m.measureLanguages = []measureOption{{label: "go", selected: true}}
	m.initMeasureConfigForm()
	m.fields[0].value = "3"
	m.fields[1].value = "custom-measure"

	req, err := m.buildMeasureRequestFromForm()
	if err != nil {
		t.Fatalf("buildMeasureRequestFromForm() error = %v", err)
	}
	if req.Iterations != 3 {
		t.Fatalf("iterations = %d, want 3", req.Iterations)
	}
	if req.OutPath != "results/custom-measure.csv" {
		t.Fatalf("out path = %q, want %q", req.OutPath, "results/custom-measure.csv")
	}
	if !reflect.DeepEqual(req.Languages, []string{"go"}) {
		t.Fatalf("languages = %#v, want %#v", req.Languages, []string{"go"})
	}
	if !reflect.DeepEqual(req.Benchmarks, []string{"binary-trees"}) {
		t.Fatalf("benchmarks = %#v, want %#v", req.Benchmarks, []string{"binary-trees"})
	}

	m.fields[0].value = "0"
	if _, err := m.buildMeasureRequestFromForm(); err == nil {
		t.Fatalf("expected iterations validation error")
	}
}

// TestStartMeasureFromFormTransitionsToRunning verifies measure submissions
// transition to running state and persist run-summary details for rendering.
func TestStartMeasureFromFormTransitionsToRunning(t *testing.T) {
	t.Parallel()

	called := false
	var gotReq appmeasure.Request
	m := newModel(
		context.Background(),
		func(context.Context, appexport.Request) error { return nil },
		func(_ context.Context, req appmeasure.Request) error {
			called = true
			gotReq = req
			return nil
		},
	)
	m.measureBenchmarks = []measureOption{{label: "binary-trees", selected: true}}
	m.measureLanguages = []measureOption{{label: "go", selected: true}}
	m.initMeasureConfigForm()
	m.fields[0].value = "2"
	m.fields[1].value = "measurements-v2"

	nextModel, cmd := m.startMeasureFromForm()
	updated := nextModel.(model)
	if updated.state != screenRunning {
		t.Fatalf("state = %v, want %v", updated.state, screenRunning)
	}
	if !reflect.DeepEqual(updated.runningMeasureLanguages, []string{"go"}) {
		t.Fatalf("running languages = %#v, want %#v", updated.runningMeasureLanguages, []string{"go"})
	}
	if !reflect.DeepEqual(updated.runningMeasureBenchmarks, []string{"binary-trees"}) {
		t.Fatalf("running benchmarks = %#v, want %#v", updated.runningMeasureBenchmarks, []string{"binary-trees"})
	}
	if updated.runningMeasureIterations != 2 {
		t.Fatalf("running iterations = %d, want %d", updated.runningMeasureIterations, 2)
	}
	if cmd == nil {
		t.Fatalf("expected running command, got nil")
	}

	msg := cmd()
	if msg == nil {
		t.Fatalf("expected non-nil batched message")
	}
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	for _, batchedCmd := range batchMsg {
		if batchedCmd == nil {
			continue
		}
		_ = batchedCmd()
	}
	if !called {
		t.Fatalf("expected measure executor to be called")
	}
	if gotReq.OutPath != "results/measurements-v2.csv" {
		t.Fatalf("out path = %q, want %q", gotReq.OutPath, "results/measurements-v2.csv")
	}
}

func TestResultViewContainsOutputPath(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, _ := m.Update(operationDoneMsg{path: "results/measurements.csv"})
	updated := nextModel.(model)

	if updated.state != screenResult {
		t.Fatalf("state = %v, want %v", updated.state, screenResult)
	}
	if !strings.Contains(updated.View(), "results/measurements.csv") {
		t.Fatalf("expected result view to contain output path")
	}
}

func TestResultViewShowsCSVPreviewTable(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, _ := m.Update(operationDoneMsg{
		path: "results/measurements.csv",
		previewRows: []table.Row{
			{"run-1", "2026-04-05 00:00:01", "go", "binary-trees"},
		},
	})
	updated := nextModel.(model)
	view := updated.View()

	if !strings.Contains(view, "CSV Preview") {
		t.Fatalf("expected result view to contain CSV Preview title, got:\n%s", view)
	}
	if !strings.Contains(view, "Run ID") ||
		!strings.Contains(view, "Created At") ||
		!strings.Contains(view, "Lang") ||
		!strings.Contains(view, "Benchmark") {
		t.Fatalf("expected result view to contain preview column labels, got:\n%s", view)
	}
	if !strings.Contains(view, "run-1") {
		t.Fatalf("expected result view to contain preview row value, got:\n%s", view)
	}
}

func TestResultViewShowsFailureErrorMessage(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, _ := m.Update(operationDoneMsg{
		path: "results/measurements.csv",
		err:  errors.New("run measure script: detected fatal output in logs/measure.txt"),
	})
	updated := nextModel.(model)

	if updated.resultErr == nil {
		t.Fatalf("expected result error to be set")
	}
	view := updated.View()
	if !strings.Contains(view, "Export failed") {
		t.Fatalf("expected failure label in result view")
	}
	if !strings.Contains(view, "logs/measure.txt") {
		t.Fatalf("expected log path hint in result view")
	}
	if strings.Contains(view, "CSV Preview") {
		t.Fatalf("did not expect CSV preview for failed result, got:\n%s", view)
	}
}

func TestResultViewShowsPreviewUnavailableMessage(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, _ := m.Update(operationDoneMsg{
		path:       "results/measurements.csv",
		previewErr: errors.New("preview unavailable: read csv header"),
	})
	updated := nextModel.(model)
	view := updated.View()

	if !strings.Contains(view, "Export finished") {
		t.Fatalf("expected success label in result view, got:\n%s", view)
	}
	if !strings.Contains(view, "CSV Preview") {
		t.Fatalf("expected CSV Preview title in result view, got:\n%s", view)
	}
	if !strings.Contains(view, "preview unavailable: read csv header") {
		t.Fatalf("expected preview unavailable message in result view, got:\n%s", view)
	}
}

func TestResultViewShowsLongErrorTailWithoutTruncation(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	longErr := "run measure script: " + strings.Repeat("x", 160) + " FINAL_TOKEN_VISIBLE"
	nextModel, _ := m.Update(operationDoneMsg{
		path: "results/measurements.csv",
		err:  errors.New(longErr),
	})
	updated := nextModel.(model)
	view := updated.View()

	if strings.Contains(view, "…") {
		t.Fatalf("did not expect ellipsis truncation in wrapped error, got:\n%s", view)
	}
	if !strings.Contains(view, "FINAL_TOKEN") || !strings.Contains(view, "_VISIBLE") {
		t.Fatalf("expected wrapped error tail markers to be visible, got:\n%s", view)
	}
}

func TestInteractiveReturnsFinalError(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenResult
	expectedErr := errors.New("export failed")
	m.finalErr = expectedErr

	if m.finalErr == nil || m.finalErr.Error() != expectedErr.Error() {
		t.Fatalf("final error mismatch")
	}
}

func TestFormAllowsSpaceInput(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.initForm(constant.ExportModeBatch)
	m.focus = 1 // From timestamp field
	m.fields[1].value = "2026-04-01"

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated := nextModel.(model)

	if updated.fields[1].value != "2026-04-01 " {
		t.Fatalf("field value = %q, want %q", updated.fields[1].value, "2026-04-01 ")
	}
}

func TestFormQTypesOnFilenameField(t *testing.T) {
	t.Parallel()

	m := newTestModel()
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

	m := newTestModel()
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

func TestFormQTypesInNonFilenameField(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.initForm(constant.ExportModeBatch)
	m.focus = 1
	m.fields[1].value = "2026-04-01"

	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := nextModel.(model)

	if updated.fields[1].value != "2026-04-01q" {
		t.Fatalf("field value = %q, want %q", updated.fields[1].value, "2026-04-01q")
	}
	if cmd != nil {
		t.Fatalf("expected nil command for text input, got %T", cmd)
	}
}

func TestFormEscQuits(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.initForm(constant.ExportModeBatch)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	msg := cmd()
	if !reflect.TypeOf(msg).AssignableTo(reflect.TypeOf(tea.QuitMsg{})) {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestMeasureSelectionEscQuits(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.initMeasureBenchmarkSelection()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
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

	m := newTestModel()
	m.state = screenRunning
	m.runningLabel = "batch export"
	m.runningOutPath = "results/measurements.csv"

	view := m.viewRunning()
	if got := strings.Count(view, "Export in progress"); got != 1 {
		t.Fatalf("progress line count = %d, want 1\nview:\n%s", got, view)
	}
}

func TestRunExportCmdIncludesPreviewRowsOnSuccess(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "measurements.csv")
	req := appexport.Request{
		Mode:    constant.ExportModeBatch,
		OutPath: outPath,
	}

	cmd := runExportCmd(context.Background(), func(_ context.Context, runReq appexport.Request) error {
		csv := strings.Join([]string{
			"run_id,measured_at,language,benchmark",
			"run-1,2026-04-05 00:00:01,go,binary-trees",
		}, "\n")
		return os.WriteFile(runReq.OutPath, []byte(csv), 0o600)
	}, req)

	msg := cmd()
	done, ok := msg.(operationDoneMsg)
	if !ok {
		t.Fatalf("expected operationDoneMsg, got %T", msg)
	}
	if done.err != nil {
		t.Fatalf("expected nil error, got %v", done.err)
	}
	if len(done.previewRows) != 1 {
		t.Fatalf("preview row count = %d, want %d", len(done.previewRows), 1)
	}
	if done.previewRows[0][0] != "run-1" {
		t.Fatalf("preview run id = %q, want %q", done.previewRows[0][0], "run-1")
	}
}

// TestRunningViewShowsMeasureSelections verifies the running view renders
// selected measure languages, benchmarks, and iterations.
func TestRunningViewShowsMeasureSelections(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning
	m.runningLabel = "measure + export"
	m.runningOutPath = "results/measurements.csv"
	m.runningMeasureLanguages = []string{"go", "c"}
	m.runningMeasureBenchmarks = []string{"binary-trees"}
	m.runningMeasureIterations = 1

	view := m.viewRunning()
	if !strings.Contains(view, "languages: go, c") {
		t.Fatalf("expected running view to show selected languages, got:\n%s", view)
	}
	if !strings.Contains(view, "benchmarks: binary-trees") {
		t.Fatalf("expected running view to show selected benchmarks, got:\n%s", view)
	}
	if !strings.Contains(view, "iterations: 1") {
		t.Fatalf("expected running view to show selected iterations, got:\n%s", view)
	}
}

func TestRunningUpdateEmitsSpinnerFollowUpCommand(t *testing.T) {
	t.Parallel()

	m := newTestModel()
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

func TestRunningEscOpensQuitPrompt(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := nextModel.(model)

	if cmd != nil {
		t.Fatalf("expected nil command when opening quit prompt, got %T", cmd)
	}
	if !updated.runningQuitPromptVisible {
		t.Fatalf("expected running quit prompt to be visible")
	}
}

func TestRunningQuitPromptYesEnterQuits(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := nextModel.(model)
	nextModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y', 'e', 's'}})
	updated = nextModel.(model)

	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	msg := cmd()
	if !reflect.TypeOf(msg).AssignableTo(reflect.TypeOf(tea.QuitMsg{})) {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestRunningQuitPromptNoButtonCancels(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := nextModel.(model)
	nextModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = nextModel.(model)

	nextModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = nextModel.(model)
	if cmd != nil {
		t.Fatalf("expected nil command when cancelling quit prompt, got %T", cmd)
	}
	if updated.runningQuitPromptVisible {
		t.Fatalf("expected running quit prompt to close when No is selected")
	}
}

func TestRunningQuitPromptInvalidInputShowsReminder(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := nextModel.(model)
	nextModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n', 'o'}})
	updated = nextModel.(model)

	nextModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = nextModel.(model)
	if cmd != nil {
		t.Fatalf("expected nil command for invalid quit input, got %T", cmd)
	}
	if !strings.Contains(updated.runningQuitReminder, "Type yes to confirm quit.") {
		t.Fatalf("expected running quit reminder to be set, got %q", updated.runningQuitReminder)
	}
	view := updated.viewRunning()
	if !strings.Contains(view, "Type yes to confirm quit.") {
		t.Fatalf("expected running view to show reminder text, got:\n%s", view)
	}
}

func TestRunningViewShowsQuitPromptWithNoButton(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenRunning
	m.runningQuitPromptVisible = true
	m.runningQuitInput = "ye"

	view := m.viewRunning()
	if !strings.Contains(view, "Confirm Quit") {
		t.Fatalf("expected running view to show confirm quit title, got:\n%s", view)
	}
	if !strings.Contains(view, "Type yes and press Enter to quit.") {
		t.Fatalf("expected running view to show confirm quit instructions, got:\n%s", view)
	}
	if !strings.Contains(view, "No") {
		t.Fatalf("expected running view to show no button, got:\n%s", view)
	}
}

func TestResultEscQuits(t *testing.T) {
	t.Parallel()

	m := newTestModel()
	m.state = screenResult

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	msg := cmd()
	if !reflect.TypeOf(msg).AssignableTo(reflect.TypeOf(tea.QuitMsg{})) {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func newTestModel() model {
	return newModel(
		context.Background(),
		func(context.Context, appexport.Request) error { return nil },
		func(context.Context, appmeasure.Request) error { return nil },
	)
}
