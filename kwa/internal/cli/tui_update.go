package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	appexport "mthesis/kwa/internal/app/export"
	"mthesis/kwa/internal/constant"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.KeyMsg:
		return m.handleMenuKey(event)
	case tea.MouseMsg:
		mouse := strings.ToLower(event.String())
		switch {
		case strings.Contains(mouse, "wheelup"):
			m.moveMenu(-1)
		case strings.Contains(mouse, "wheeldown"):
			m.moveMenu(1)
		case strings.Contains(mouse, "left"):
			m.startFormForSelected()
			// Force a clean frame when switching screens via mouse input.
			return m, tea.ClearScreen
		}
	}

	return m, nil
}

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
				spec:  fieldSpec{label: "fileName", placeholder: defaultCSVFilename},
				value: defaultCSVFilename,
			},
		}
	case constant.ExportModeByID:
		m.formTitle = "Export by ID"
		m.fields = []fieldState{
			{
				spec:  fieldSpec{label: "Run ID", placeholder: "paste run ID"},
				value: "",
			},
			{
				spec:  fieldSpec{label: "fileName", placeholder: defaultCSVFilename},
				value: defaultCSVFilename,
			},
		}
	}
}

func (m model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
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
	case tea.KeyRunes:
		m.appendRunes(string(key.Runes))
		m.validationErr = ""
		return m, nil
	}

	switch key.String() {
	case "q":
		return m, tea.Quit
	case "ctrl+u":
		m.clearFocusedField()
		m.validationErr = ""
		return m, nil
	default:
		return m, nil
	}
}

func (m *model) moveFocus(delta int) {
	if len(m.fields) == 0 {
		return
	}

	next := (m.focus + delta + len(m.fields)) % len(m.fields)
	m.focus = next
}

func (m *model) appendRunes(value string) {
	if len(m.fields) == 0 {
		return
	}
	m.fields[m.focus].value += value
}

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
	m.spinnerIndex = 0
	m.validationErr = ""

	return m, tea.Batch(
		tea.ClearScreen,
		runExportCmd(m.ctx, m.executor, req),
		spinnerTickCmd(),
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

		outPath := buildOutputPath(m.fields[1].value)
		return appexport.Request{
			Mode:      constant.ExportModeBatch,
			BatchSize: batchSize,
			OutPath:   outPath,
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

func runExportCmd(ctx context.Context, execute executeRequestFunc, req appexport.Request) tea.Cmd {
	return func() tea.Msg {
		err := execute(ctx, req)
		return exportDoneMsg{path: req.OutPath, err: err}
	}
}

func spinnerTickCmd() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

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

func (m model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.KeyMsg:
		if event.Type == tea.KeyCtrlC || event.Type == tea.KeyEnter || event.String() == "q" {
			return m, tea.Quit
		}
	case tea.MouseMsg:
		if strings.Contains(strings.ToLower(event.String()), "left") {
			return m, tea.Quit
		}
	}

	return m, nil
}
