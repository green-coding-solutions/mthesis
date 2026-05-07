package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appexport "mthesis/kwa/internal/app/export"
	appmeasure "mthesis/kwa/internal/app/measure"
	"mthesis/kwa/internal/constant"

	tea "github.com/charmbracelet/bubbletea"
)

// updateMenu handles menu-screen keyboard events.
func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.KeyMsg:
		return m.handleMenuKey(event)
	}

	return m, nil
}

// handleMenuKey applies menu navigation and selection keybindings.
func (m model) handleMenuKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyUp:
		m.moveMenu(-1)
		return m, nil
	case tea.KeyDown:
		m.moveMenu(1)
		return m, nil
	case tea.KeyEnter:
		m.startFormForSelected()
		// Clear stale lines when moving from menu to next workflow step.
		return m, tea.ClearScreen
	}

	return m, nil
}

// moveMenu updates the selected menu item using circular navigation.
func (m *model) moveMenu(delta int) {
	menuCount := 3
	next := (int(m.selected) + delta + menuCount) % menuCount
	m.selected = menuOption(next)
}

// startFormForSelected transitions from menu to the selected workflow entrypoint.
func (m *model) startFormForSelected() {
	switch m.selected {
	case menuBatch:
		m.initForm(constant.ExportModeBatch)
	case menuByID:
		m.initForm(constant.ExportModeByID)
	case menuMeasure:
		m.initMeasureBenchmarkSelection()
	}
}

// initForm builds the field list and defaults for batch/by-id export forms.
// Batch forms include optional timestamps, while by-id forms collect only run
// ID and filename; it mutates model form state and produces no command.
func (m *model) initForm(mode constant.ExportMode) {
	m.state = screenForm
	m.formMode = mode
	m.focus = 0
	m.validationErr = ""

	switch mode {
	case constant.ExportModeBatch:
		m.formTitle = "Batch Export"
		m.fields = []fieldState{
			{
				spec:  fieldSpec{label: "Rows per batch (optional)", placeholder: "100"},
				value: strconv.Itoa(constant.DefaultBatchSize),
			},
			{
				spec:  fieldSpec{label: "From timestamp (optional)", placeholder: "YYYY-MM-DD or YYYY-MM-DD HH:MM:SS"},
				value: "",
			},
			{
				spec:  fieldSpec{label: "To timestamp (optional)", placeholder: "YYYY-MM-DD or YYYY-MM-DD HH:MM:SS"},
				value: "",
			},
			{
				spec:  fieldSpec{label: "Filename", placeholder: constant.DefaultCSVFilename},
				value: constant.DefaultCSVFilename,
			},
		}
	case constant.ExportModeByID:
		m.formTitle = "byID Export"
		m.fields = []fieldState{
			{
				spec:  fieldSpec{label: "Run ID", placeholder: "paste run ID"},
				value: "",
			},
			{
				spec:  fieldSpec{label: "Filename", placeholder: constant.DefaultCSVFilename},
				value: constant.DefaultCSVFilename,
			},
		}
	}
}

// initMeasureBenchmarkSelection resets measure selections and opens the
// benchmark multi-select step.
func (m *model) initMeasureBenchmarkSelection() {
	m.state = screenMeasureBenchmarks
	m.validationErr = ""
	m.measureCursor = 0
	m.measureBenchmarks = newMeasureOptions(constant.MeasureBenchmarks())
	m.measureLanguages = newMeasureOptions(constant.MeasureLanguages())
}

// initMeasureLanguageSelection opens the language multi-select step.
func (m *model) initMeasureLanguageSelection() {
	m.state = screenMeasureLanguages
	m.validationErr = ""
	m.measureCursor = 0
}

// initMeasureConfigForm builds the final measure config form with iterations
// and output filename fields.
func (m *model) initMeasureConfigForm() {
	m.state = screenMeasureConfig
	m.validationErr = ""
	m.focus = 0
	m.formTitle = "Measure Config"
	m.fields = []fieldState{
		{
			spec:  fieldSpec{label: "Iterations", placeholder: "1"},
			value: "1",
		},
		{
			spec:  fieldSpec{label: "Filename", placeholder: constant.DefaultCSVFilename},
			value: constant.DefaultCSVFilename,
		},
	}
}

// updateForm handles key input for batch/by-id interactive form fields.
func (m model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyUp:
		m.moveFocus(-1)
		return m, nil
	case tea.KeyDown:
		m.moveFocus(1)
		return m, nil
	case tea.KeyEnter:
		if m.focus < len(m.fields)-1 {
			m.moveFocus(1)
			return m, nil
		}
		return m.startExportFromForm()
	case tea.KeyBackspace, tea.KeyDelete:
		m.removeLastRune()
		m.validationErr = ""
		return m, nil
	case tea.KeySpace:
		m.appendRunes(" ")
		m.validationErr = ""
		return m, nil
	case tea.KeyRunes:
		m.appendRunes(string(key.Runes))
		m.validationErr = ""
		return m, nil
	}

	switch key.String() {
	case "ctrl+u":
		m.clearFocusedField()
		m.validationErr = ""
		return m, nil
	default:
		return m, nil
	}
}

// updateMeasureConfig handles key input for the measure iterations/file form.
func (m model) updateMeasureConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyUp:
		m.moveFocus(-1)
		return m, nil
	case tea.KeyDown:
		m.moveFocus(1)
		return m, nil
	case tea.KeyEnter:
		if m.focus < len(m.fields)-1 {
			m.moveFocus(1)
			return m, nil
		}
		return m.startMeasureFromForm()
	case tea.KeyBackspace, tea.KeyDelete:
		m.removeLastRune()
		m.validationErr = ""
		return m, nil
	case tea.KeySpace:
		m.appendRunes(" ")
		m.validationErr = ""
		return m, nil
	case tea.KeyRunes:
		m.appendRunes(string(key.Runes))
		m.validationErr = ""
		return m, nil
	}

	switch key.String() {
	case "ctrl+u":
		m.clearFocusedField()
		m.validationErr = ""
		return m, nil
	default:
		return m, nil
	}
}

// updateMeasureSelection handles keyboard events for benchmark/language
// multi-select screens.
func (m model) updateMeasureSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyUp:
		m.moveMeasureCursor(-1)
		return m, nil
	case tea.KeyDown:
		m.moveMeasureCursor(1)
		return m, nil
	case tea.KeySpace:
		m.toggleCurrentMeasureOption()
		m.validationErr = ""
		return m, nil
	case tea.KeyEnter:
		return m.handleMeasureSelectionSubmit()
	}

	switch key.String() {
	case "a", "A":
		m.toggleAllMeasureOptions()
		m.validationErr = ""
		return m, nil
	default:
		return m, nil
	}
}

// handleMeasureSelectionSubmit validates one selection step and advances to the
// next measure step when at least one item is selected.
func (m model) handleMeasureSelectionSubmit() (tea.Model, tea.Cmd) {
	switch m.state {
	case screenMeasureBenchmarks:
		if !hasSelectedMeasureOption(m.measureBenchmarks) {
			m.validationErr = "select at least one benchmark"
			return m, nil
		}
		m.initMeasureLanguageSelection()
		return m, tea.ClearScreen
	case screenMeasureLanguages:
		if !hasSelectedMeasureOption(m.measureLanguages) {
			m.validationErr = "select at least one language"
			return m, nil
		}
		m.initMeasureConfigForm()
		return m, tea.ClearScreen
	default:
		return m, nil
	}
}

// currentMeasureOptions returns the option slice for the active selection step.
func (m *model) currentMeasureOptions() *[]measureOption {
	switch m.state {
	case screenMeasureBenchmarks:
		return &m.measureBenchmarks
	case screenMeasureLanguages:
		return &m.measureLanguages
	default:
		return nil
	}
}

// moveMeasureCursor updates the focused selection row using circular navigation.
func (m *model) moveMeasureCursor(delta int) {
	options := m.currentMeasureOptions()
	if options == nil || len(*options) == 0 {
		return
	}

	next := (m.measureCursor + delta + len(*options)) % len(*options)
	m.measureCursor = next
}

// toggleCurrentMeasureOption flips selection state for the focused option.
func (m *model) toggleCurrentMeasureOption() {
	options := m.currentMeasureOptions()
	if options == nil || len(*options) == 0 {
		return
	}

	(*options)[m.measureCursor].selected = !(*options)[m.measureCursor].selected
}

// toggleAllMeasureOptions selects every option unless all are already selected,
// in which case it clears all selections.
func (m *model) toggleAllMeasureOptions() {
	options := m.currentMeasureOptions()
	if options == nil || len(*options) == 0 {
		return
	}

	selectAll := !allMeasureOptionsSelected(*options)
	for i := range *options {
		(*options)[i].selected = selectAll
	}
}

// allMeasureOptionsSelected reports whether every option in the given list is selected.
func allMeasureOptionsSelected(options []measureOption) bool {
	if len(options) == 0 {
		return false
	}

	for _, option := range options {
		if !option.selected {
			return false
		}
	}

	return true
}

// hasSelectedMeasureOption reports whether at least one option is selected.
func hasSelectedMeasureOption(options []measureOption) bool {
	for _, option := range options {
		if option.selected {
			return true
		}
	}

	return false
}

// moveFocus changes focused form input using circular navigation.
func (m *model) moveFocus(delta int) {
	if len(m.fields) == 0 {
		return
	}

	next := (m.focus + delta + len(m.fields)) % len(m.fields)
	m.focus = next
}

// appendRunes appends typed text to the currently focused field.
func (m *model) appendRunes(value string) {
	if len(m.fields) == 0 {
		return
	}
	m.fields[m.focus].value += value
}

// removeLastRune deletes the last rune from the focused field value.
func (m *model) removeLastRune() {
	if len(m.fields) == 0 {
		return
	}

	current := m.fields[m.focus].value
	if current == "" {
		return
	}

	runes := []rune(current)
	m.fields[m.focus].value = string(runes[:len(runes)-1])
}

// clearFocusedField clears the currently focused form input value.
func (m *model) clearFocusedField() {
	if len(m.fields) == 0 {
		return
	}

	m.fields[m.focus].value = ""
}

// startExportFromForm validates export form fields and starts async export execution.
func (m model) startExportFromForm() (tea.Model, tea.Cmd) {
	req, err := m.buildRequestFromForm()
	if err != nil {
		m.validationErr = err.Error()
		return m, nil
	}

	m.state = screenRunning
	m.runningLabel = fmt.Sprintf("%s export", req.Mode)
	m.runningOutPath = req.OutPath
	m.runningMeasureLanguages = nil
	m.runningMeasureBenchmarks = nil
	m.runningMeasureIterations = 0
	m.runningQuitPromptVisible = false
	m.runningQuitInput = ""
	m.runningQuitNoFocused = false
	m.runningQuitReminder = ""
	m.spinner = newSpinnerModel()
	m.validationErr = ""

	return m, tea.Batch(
		runExportCmd(m.ctx, m.executeExport, req),
		m.spinner.Tick,
	)
}

// startMeasureFromForm validates measure config fields, captures a run summary,
// and starts async measure+export execution.
func (m model) startMeasureFromForm() (tea.Model, tea.Cmd) {
	req, err := m.buildMeasureRequestFromForm()
	if err != nil {
		m.validationErr = err.Error()
		return m, nil
	}

	m.state = screenRunning
	m.runningLabel = "measure + export"
	m.runningOutPath = req.OutPath
	m.runningMeasureLanguages = append([]string(nil), req.Languages...)
	m.runningMeasureBenchmarks = append([]string(nil), req.Benchmarks...)
	m.runningMeasureIterations = req.Iterations
	m.runningQuitPromptVisible = false
	m.runningQuitInput = ""
	m.runningQuitNoFocused = false
	m.runningQuitReminder = ""
	m.spinner = newSpinnerModel()
	m.validationErr = ""

	return m, tea.Batch(
		runMeasureCmd(m.ctx, m.executeMeasure, req),
		m.spinner.Tick,
	)
}

// buildRequestFromForm maps batch/by-id form values into a validated export request.
// It parses batch size and optional batch timestamps, requires by-id run ID,
// resolves output paths, and returns validation errors without starting I/O.
func (m model) buildRequestFromForm() (appexport.Request, error) {
	switch m.formMode {
	case constant.ExportModeBatch:
		batchInput := strings.TrimSpace(m.fields[0].value)
		batchSize := constant.DefaultBatchSize
		if batchInput != "" {
			parsed, err := strconv.Atoi(batchInput)
			if err != nil || parsed <= 0 {
				return appexport.Request{}, fmt.Errorf("batch size must be a positive number")
			}
			batchSize = parsed
		}

		timeRange, err := appexport.ParseTimeRange(m.fields[1].value, m.fields[2].value)
		if err != nil {
			return appexport.Request{}, err
		}

		outPath := buildOutputPath(m.fields[3].value)
		return appexport.Request{
			Mode:      constant.ExportModeBatch,
			BatchSize: batchSize,
			OutPath:   outPath,
			TimeRange: timeRange,
		}, nil
	case constant.ExportModeByID:
		runID := strings.TrimSpace(m.fields[0].value)
		if runID == "" {
			return appexport.Request{}, fmt.Errorf("run ID is required")
		}

		outPath := buildOutputPath(m.fields[1].value)
		return appexport.Request{
			Mode:    constant.ExportModeByID,
			RunID:   runID,
			OutPath: outPath,
		}, nil
	default:
		return appexport.Request{}, fmt.Errorf("unsupported form mode %q", m.formMode)
	}
}

// buildMeasureRequestFromForm maps measure config values plus selected options
// into a validated measure request.
func (m model) buildMeasureRequestFromForm() (appmeasure.Request, error) {
	if !hasSelectedMeasureOption(m.measureBenchmarks) {
		return appmeasure.Request{}, fmt.Errorf("select at least one benchmark")
	}
	if !hasSelectedMeasureOption(m.measureLanguages) {
		return appmeasure.Request{}, fmt.Errorf("select at least one language")
	}

	iterationsInput := strings.TrimSpace(m.fields[0].value)
	if iterationsInput == "" {
		return appmeasure.Request{}, fmt.Errorf("iterations is required")
	}

	iterations, err := strconv.Atoi(iterationsInput)
	if err != nil || iterations <= 0 {
		return appmeasure.Request{}, fmt.Errorf("iterations must be a positive number")
	}

	return appmeasure.Request{
		Languages:  selectedMeasureOptionLabels(m.measureLanguages),
		Benchmarks: selectedMeasureOptionLabels(m.measureBenchmarks),
		Iterations: iterations,
		OutPath:    buildOutputPath(m.fields[1].value),
	}, nil
}

// runExportCmd executes one export request and emits a completion message with
// optional CSV preview rows when export succeeds.
func runExportCmd(ctx context.Context, execute executeRequestFunc, req appexport.Request) tea.Cmd {
	return func() tea.Msg {
		err := execute(ctx, req)
		if err != nil {
			return operationDoneMsg{path: req.OutPath, err: err}
		}

		previewRows, previewErr := readCSVPreviewRows(req.OutPath, csvPreviewRowLimit)
		return operationDoneMsg{
			path:        req.OutPath,
			previewRows: previewRows,
			previewErr:  previewErr,
		}
	}
}

// runMeasureCmd executes one measure request and emits a completion message with
// optional CSV preview rows when measure+export succeeds.
func runMeasureCmd(ctx context.Context, execute executeMeasureFunc, req appmeasure.Request) tea.Cmd {
	return func() tea.Msg {
		err := execute(ctx, req)
		if err != nil {
			return operationDoneMsg{path: req.OutPath, err: err}
		}

		previewRows, previewErr := readCSVPreviewRows(req.OutPath, csvPreviewRowLimit)
		return operationDoneMsg{
			path:        req.OutPath,
			previewRows: previewRows,
			previewErr:  previewErr,
		}
	}
}

// updateRunning handles quit input while export/measure execution is in progress.
func (m model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if key.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	if key.Type == tea.KeyEsc {
		if m.runningQuitPromptVisible {
			m.runningQuitPromptVisible = false
			m.runningQuitInput = ""
			m.runningQuitNoFocused = false
			m.runningQuitReminder = ""
			return m, nil
		}

		m.runningQuitPromptVisible = true
		m.runningQuitInput = ""
		m.runningQuitNoFocused = false
		m.runningQuitReminder = ""
		return m, nil
	}

	if !m.runningQuitPromptVisible {
		return m, nil
	}

	switch key.Type {
	case tea.KeyEnter:
		if m.runningQuitNoFocused {
			m.runningQuitPromptVisible = false
			m.runningQuitInput = ""
			m.runningQuitNoFocused = false
			m.runningQuitReminder = ""
			return m, nil
		}
		if strings.EqualFold(strings.TrimSpace(m.runningQuitInput), "yes") {
			return m, tea.Quit
		}
		m.runningQuitReminder = "? Type yes to confirm quit."
		return m, nil
	case tea.KeyTab, tea.KeyShiftTab, tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
		m.runningQuitNoFocused = !m.runningQuitNoFocused
		m.runningQuitReminder = ""
		return m, nil
	case tea.KeyBackspace, tea.KeyDelete:
		if m.runningQuitNoFocused || m.runningQuitInput == "" {
			return m, nil
		}
		runes := []rune(m.runningQuitInput)
		m.runningQuitInput = string(runes[:len(runes)-1])
		m.runningQuitReminder = ""
		return m, nil
	case tea.KeySpace:
		if m.runningQuitNoFocused {
			return m, nil
		}
		m.runningQuitInput += " "
		m.runningQuitReminder = ""
		return m, nil
	case tea.KeyRunes:
		if m.runningQuitNoFocused {
			return m, nil
		}
		m.runningQuitInput += string(key.Runes)
		m.runningQuitReminder = ""
		return m, nil
	}

	return m, nil
}

// updateResult handles exit actions from the final result screen.
func (m model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.KeyMsg:
		if event.Type == tea.KeyCtrlC || event.Type == tea.KeyEsc || event.Type == tea.KeyEnter {
			return m, tea.Quit
		}
	}

	return m, nil
}
