package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screenState int

const (
	screenMenu screenState = iota
	screenForm
	screenRunning
	screenResult
)

type menuOption int

const (
	menuBatch menuOption = iota
	menuByID
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type fieldSpec struct {
	label       string
	placeholder string
	required    bool
	defaultText string
}

type fieldState struct {
	spec  fieldSpec
	value string
}

type exportDoneMsg struct {
	path string
	err  error
}

type spinnerTickMsg struct{}

type model struct {
	ctx      context.Context
	executor ExportExecutor

	state     screenState
	selected  menuOption
	formMode  ExportMode
	formTitle string
	fields    []fieldState
	focus     int

	validationErr string
	runningReq    ExportRequest
	spinnerIndex  int

	resultPath string
	resultErr  error
	finalErr   error

	width  int
	height int
}

// runInteractive starts the Bubble Tea application used by the root `kwa` command.
func runInteractive(ctx context.Context, execute ExportExecutor, _ io.Writer, _ io.Writer) error {
	if execute == nil {
		return fmt.Errorf("export executor must not be nil")
	}

	program := tea.NewProgram(
		newModel(ctx, execute),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("run interactive app: %w", err)
	}

	appModel, ok := finalModel.(model)
	if !ok {
		return nil
	}

	return appModel.finalErr
}

func newModel(ctx context.Context, execute ExportExecutor) model {
	return model{
		ctx:      ctx,
		executor: execute,
		state:    screenMenu,
		selected: menuBatch,
		formMode: ExportModeBatch,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = event.Width
		m.height = event.Height
		return m, nil
	case exportDoneMsg:
		m.state = screenResult
		m.resultPath = event.path
		m.resultErr = event.err
		m.validationErr = ""
		if event.err != nil {
			m.finalErr = event.err
		}
		return m, tea.ClearScreen
	case spinnerTickMsg:
		if m.state != screenRunning {
			return m, nil
		}
		m.spinnerIndex = (m.spinnerIndex + 1) % len(spinnerFrames)
		return m, spinnerTickCmd()
	}

	switch m.state {
	case screenMenu:
		return m.updateMenu(msg)
	case screenForm:
		return m.updateForm(msg)
	case screenRunning:
		return m.updateRunning(msg)
	case screenResult:
		return m.updateResult(msg)
	default:
		return m, nil
	}
}

func (m model) View() string {
	var content string
	switch m.state {
	case screenMenu:
		content = m.viewMenu()
	case screenForm:
		content = m.viewForm()
	case screenRunning:
		content = m.viewRunning()
	case screenResult:
		content = m.viewResult()
	default:
		content = ""
	}

	rendered := tuiStyles.doc.Render(content)
	if m.width <= 0 || m.height <= 0 {
		return rendered
	}

	// Place content in a full terminal-sized frame so old characters are erased
	// when switching between screens with different line lengths.
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, rendered)
}

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
		m.initForm(ExportModeBatch)
		return
	}

	m.initForm(ExportModeByID)
}

// initForm builds the field list and defaults for the selected export mode.
func (m *model) initForm(mode ExportMode) {
	m.state = screenForm
	m.formMode = mode
	m.focus = 0
	m.validationErr = ""

	switch mode {
	case ExportModeBatch:
		m.formTitle = "Batch Export"
		m.fields = []fieldState{
			{
				spec: fieldSpec{
					label:       "Rows per batch (optional)",
					placeholder: "100",
					required:    false,
					defaultText: "100",
				},
				value: "100",
			},
			{
				spec: fieldSpec{
					label:       "fileName",
					placeholder: "measurements.csv",
					required:    false,
					defaultText: "measurements.csv",
				},
				value: "measurements.csv",
			},
		}
	case ExportModeByID:
		m.formTitle = "Export by ID"
		m.fields = []fieldState{
			{
				spec: fieldSpec{
					label:       "Run ID",
					placeholder: "paste run ID",
					required:    true,
				},
				value: "",
			},
			{
				spec: fieldSpec{
					label:       "fileName",
					placeholder: "measurements.csv",
					required:    false,
					defaultText: "measurements.csv",
				},
				value: "measurements.csv",
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
func (m model) buildRequestFromForm() (ExportRequest, error) {
	switch m.formMode {
	case ExportModeBatch:
		batchInput := strings.TrimSpace(m.fields[0].value)
		batchSize := DefaultBatchSize
		if batchInput != "" {
			parsed, err := strconv.Atoi(batchInput)
			if err != nil || parsed <= 0 {
				return ExportRequest{}, fmt.Errorf("batch size must be a positive number")
			}
			batchSize = parsed
		}

		outPath := buildOutputPath(m.fields[1].value)
		return ExportRequest{
			Mode:      ExportModeBatch,
			BatchSize: batchSize,
			OutPath:   outPath,
		}, nil
	case ExportModeByID:
		runID := strings.TrimSpace(m.fields[0].value)
		if runID == "" {
			return ExportRequest{}, fmt.Errorf("run ID is required")
		}

		outPath := buildOutputPath(m.fields[1].value)
		return ExportRequest{
			Mode:    ExportModeByID,
			RunID:   runID,
			OutPath: outPath,
		}, nil
	default:
		return ExportRequest{}, fmt.Errorf("unsupported form mode %q", m.formMode)
	}
}

func runExportCmd(ctx context.Context, execute ExportExecutor, req ExportRequest) tea.Cmd {
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

func (m model) viewMenu() string {
	panelWidth := m.panelWidth()
	logo := m.renderMenuLogo(panelWidth)
	items := []string{}
	labels := []string{"batch export", "byID"}
	for i, label := range labels {
		line := "  " + truncateText(label, panelWidth-8)
		if menuOption(i) == m.selected {
			line = "> " + truncateText(label, panelWidth-10)
			items = append(items, tuiStyles.menuItemSelected.Render(line))
			continue
		}

		items = append(items, tuiStyles.menuItem.Render(line))
	}

	body := strings.Join([]string{
		logo,
		"",
		tuiStyles.sectionTitle.Render("Main Menu"),
		"",
		strings.Join(items, "\n"),
	}, "\n")

	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Pick an export flow to continue"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("↑/↓ move • Enter select • mouse wheel • q quit"),
	}, "\n")
}

// renderMenuLogo renders the bat logo inside the menu panel and truncates long
// lines to avoid wrapping artifacts on narrow terminals.
func (m model) renderMenuLogo(panelWidth int) string {
	maxWidth := panelWidth - 6 // panel border and horizontal padding
	if maxWidth < 10 {
		maxWidth = 10
	}

	lines := strings.Split(strings.Trim(batLogo, "\n"), "\n")
	for i := range lines {
		lines[i] = truncateText(strings.TrimRight(lines[i], " "), maxWidth)
	}

	return tuiStyles.logo.Render(strings.Join(lines, "\n"))
}

func (m model) viewForm() string {
	panelWidth := m.panelWidth()
	sections := make([]string, 0, len(m.fields)*2)

	for i, field := range m.fields {
		labelStyle := tuiStyles.label
		if i == m.focus {
			labelStyle = tuiStyles.labelFocused
		}

		sections = append(sections, labelStyle.Render(field.spec.label))
		sections = append(sections, m.renderFieldValue(i, panelWidth))
	}

	if m.validationErr != "" {
		sections = append(sections, tuiStyles.error.Render(m.validationErr))
	}

	body := strings.Join([]string{
		tuiStyles.sectionTitle.Render(m.formTitle),
		"",
		strings.Join(sections, "\n\n"),
	}, "\n")

	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Fill in the fields and press Enter on the last input"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("↑/↓ focus • Enter next/submit • ctrl+u clear • q quit"),
	}, "\n")
}

func (m model) renderFieldValue(index int, panelWidth int) string {
	field := m.fields[index]
	value := field.value
	focused := index == m.focus

	maxTextWidth := panelWidth - 12
	if maxTextWidth < 10 {
		maxTextWidth = 10
	}

	if strings.TrimSpace(value) == "" {
		value = tuiStyles.inputPlaceholder.Render(truncateText(field.spec.placeholder, maxTextWidth))
	} else {
		value = tuiStyles.inputText.Render(truncateText(value, maxTextWidth))
	}

	if focused {
		value = value + tuiStyles.inputCursor.Render("▌")
		return tuiStyles.inputFocused.Render(value)
	}

	return tuiStyles.input.Render(value)
}

func (m model) viewRunning() string {
	panelWidth := m.panelWidth()
	spinner := spinnerFrames[m.spinnerIndex]
	message := fmt.Sprintf("%s Export in progress", spinner)

	body := strings.Join([]string{
		tuiStyles.sectionTitle.Render("Running"),
		"",
		tuiStyles.spinner.Render(message),
		"",
		tuiStyles.label.Render(fmt.Sprintf("mode: %s", m.runningReq.Mode)),
		tuiStyles.label.Render(fmt.Sprintf("output: %s", truncateText(m.runningReq.OutPath, panelWidth-12))),
	}, "\n")

	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Working on your export request"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("q quit"),
	}, "\n")
}

func (m model) viewResult() string {
	panelWidth := m.panelWidth()

	statusLine := tuiStyles.success.Render("Export finished")
	if m.resultErr != nil {
		statusLine = tuiStyles.error.Render("Export failed")
	}

	lines := []string{
		tuiStyles.sectionTitle.Render("Result"),
		"",
		statusLine,
		tuiStyles.label.Render(fmt.Sprintf("path: %s", truncateText(m.resultPath, panelWidth-10))),
	}

	if m.resultErr != nil {
		lines = append(lines, tuiStyles.error.Render(truncateText(m.resultErr.Error(), panelWidth-8)))
	}

	body := strings.Join(lines, "\n")
	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Press q to leave the app"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("q exit"),
	}, "\n")
}

func (m model) panelWidth() int {
	if m.width <= 0 {
		return 72
	}

	width := m.width - 8
	if width < 46 {
		width = 46
	}
	if width > 90 {
		width = 90
	}

	return width
}

func truncateText(value string, max int) string {
	if max <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max == 1 {
		return "…"
	}

	return string(runes[:max-1]) + "…"
}
