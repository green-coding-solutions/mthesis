package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appexport "mthesis/kwa/internal/app/export"
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
	case tea.KeyUp:
		m.moveMenu(-1)
		return m, nil
	case tea.KeyDown:
		m.moveMenu(1)
		return m, nil
	case tea.KeyEnter:
		m.startFormForSelected()
		// Clear stale lines when moving from menu -> form.
		return m, tea.ClearScreen
	}

	if key.String() == "q" {
		return m, tea.Quit
	}

	return m, nil
}

// moveMenu updates the selected menu item using circular navigation.
func (m *model) moveMenu(delta int) {
	menuCount := 2
	next := (int(m.selected) + delta + menuCount) % menuCount
	m.selected = menuOption(next)
}

// startFormForSelected moves from menu state into the matching input form.
func (m *model) startFormForSelected() {
	if m.selected == menuBatch {
		m.initForm(constant.ExportModeBatch)
		return
	}

	m.initForm(constant.ExportModeByID)
}

// initForm builds the field list and defaults for the selected export mode.
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
				spec:  fieldSpec{label: "fileName", placeholder: constant.DefaultCSVFilename},
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
				spec:  fieldSpec{label: "From timestamp (optional)", placeholder: "YYYY-MM-DD or YYYY-MM-DD HH:MM:SS"},
				value: "",
			},
			{
				spec:  fieldSpec{label: "To timestamp (optional)", placeholder: "YYYY-MM-DD or YYYY-MM-DD HH:MM:SS"},
				value: "",
			},
			{
				spec:  fieldSpec{label: "fileName", placeholder: constant.DefaultCSVFilename},
				value: constant.DefaultCSVFilename,
			},
		}
	}
}

// updateForm handles key input for interactive export form fields.
func (m model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if key.String() == "q" && !m.allowsLiteralQInput() {
		return m, tea.Quit
	}

	switch key.Type {
	case tea.KeyCtrlC:
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

// allowsLiteralQInput reports whether the focused field should treat `q` as text.
func (m model) allowsLiteralQInput() bool {
	if m.focus < 0 || m.focus >= len(m.fields) {
		return false
	}

	label := m.fields[m.focus].spec.label
	return label == "fileName" || label == "Run ID"
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

// startExportFromForm validates fields and starts async export execution.
func (m model) startExportFromForm() (tea.Model, tea.Cmd) {
	req, err := m.buildRequestFromForm()
	if err != nil {
		m.validationErr = err.Error()
		return m, nil
	}

	m.state = screenRunning
	m.runningReq = req
	m.spinner = newSpinnerModel()
	m.validationErr = ""

	return m, tea.Batch(
		runExportCmd(m.ctx, m.executor, req),
		m.spinner.Tick,
	)
}

// buildRequestFromForm maps current form values into a validated export request.
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

		timeRange, err := appexport.ParseTimeRange(m.fields[1].value, m.fields[2].value)
		if err != nil {
			return appexport.Request{}, err
		}

		outPath := buildOutputPath(m.fields[3].value)
		return appexport.Request{
			Mode:      constant.ExportModeByID,
			RunID:     runID,
			OutPath:   outPath,
			TimeRange: timeRange,
		}, nil
	default:
		return appexport.Request{}, fmt.Errorf("unsupported form mode %q", m.formMode)
	}
}

// runExportCmd executes one export request and emits a completion message.
func runExportCmd(ctx context.Context, execute executeRequestFunc, req appexport.Request) tea.Cmd {
	return func() tea.Msg {
		err := execute(ctx, req)
		return exportDoneMsg{path: req.OutPath, err: err}
	}
}

// updateRunning handles quit input while export execution is in progress.
func (m model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if key.Type == tea.KeyCtrlC || key.String() == "q" {
		return m, tea.Quit
	}

	return m, nil
}

// updateResult handles exit actions from the final result screen.
func (m model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.KeyMsg:
		if event.Type == tea.KeyCtrlC || event.Type == tea.KeyEnter || event.String() == "q" {
			return m, tea.Quit
		}
	}

	return m, nil
}
