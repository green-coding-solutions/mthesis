package cli

import (
	"context"
	"fmt"
	"io"

	"mthesis/kwa/internal/constant"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type screenState int

const (
	screenMenu screenState = iota
	screenForm
	screenMeasureBenchmarks
	screenMeasureLanguages
	screenMeasureConfig
	screenRunning
	screenResult
)

type menuOption int

const (
	menuBatch menuOption = iota
	menuByID
	menuMeasure
)

type fieldSpec struct {
	label       string
	placeholder string
}

type fieldState struct {
	spec  fieldSpec
	value string
}

type measureOption struct {
	label    string
	selected bool
}

// operationDoneMsg carries one async export/measure completion outcome,
// including optional CSV preview rows and a non-fatal preview extraction error.
type operationDoneMsg struct {
	path        string
	err         error
	previewRows []table.Row
	previewErr  error
}

type model struct {
	ctx            context.Context
	executeExport  executeRequestFunc
	executeMeasure executeMeasureFunc

	state    screenState
	selected menuOption

	formMode  constant.ExportMode
	formTitle string
	fields    []fieldState
	focus     int

	measureBenchmarks []measureOption
	measureLanguages  []measureOption
	measureCursor     int

	validationErr string
	spinner       spinner.Model

	runningLabel             string
	runningOutPath           string
	runningMeasureLanguages  []string
	runningMeasureBenchmarks []string
	runningMeasureIterations int
	runningQuitPromptVisible bool
	runningQuitInput         string
	runningQuitNoFocused     bool
	runningQuitReminder      string

	resultPath             string
	resultErr              error
	resultPreviewTable     table.Model
	resultPreviewAvailable bool
	resultPreviewErr       string
	finalErr               error

	width  int
	height int
}

// runInteractive starts the Bubble Tea application used by the root `kwa`
// command and requires both export and measure execution callbacks.
func runInteractive(
	ctx context.Context,
	executeExport executeRequestFunc,
	executeMeasure executeMeasureFunc,
	_ io.Writer,
	_ io.Writer,
) error {
	if executeExport == nil {
		return fmt.Errorf("export executor must not be nil")
	}
	if executeMeasure == nil {
		return fmt.Errorf("measure executor must not be nil")
	}

	program := tea.NewProgram(
		newModel(ctx, executeExport, executeMeasure),
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

// newModel builds the initial TUI model with menu state and runtime dependencies.
func newModel(ctx context.Context, executeExport executeRequestFunc, executeMeasure executeMeasureFunc) model {
	return model{
		ctx:            ctx,
		executeExport:  executeExport,
		executeMeasure: executeMeasure,
		state:          screenMenu,
		selected:       menuBatch,
		formMode:       constant.ExportModeBatch,
		spinner:        newSpinnerModel(),
	}
}

// newSpinnerModel creates the running-screen spinner with shared CLI styling.
func newSpinnerModel() spinner.Model {
	s := spinner.New()
	// MiniDot keeps a single-cell frame width and avoids wrap artifacts.
	s.Spinner = spinner.MiniDot
	s.Style = tuiStyles.spinner
	return s
}

// Init returns no initial command because the model starts in a static menu state.
func (m model) Init() tea.Cmd {
	return nil
}

// Update routes Bubble Tea messages to state-specific handlers and coordinates
// async completion events for export and measure workflows.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = event.Width
		m.height = event.Height
		return m, nil
	case tea.MouseMsg:
		// Ignore all mouse input globally (wheel, click, movement).
		return m, nil
	case operationDoneMsg:
		m.state = screenResult
		m.resultPath = event.path
		m.resultErr = event.err
		m.resultPreviewAvailable = false
		m.resultPreviewErr = ""
		m.resultPreviewTable = table.Model{}
		if event.err == nil {
			if len(event.previewRows) > 0 {
				m.resultPreviewTable = newResultPreviewTable(event.previewRows, m.panelContentWidth())
				m.resultPreviewAvailable = true
			} else if event.previewErr != nil {
				m.resultPreviewErr = event.previewErr.Error()
			} else {
				m.resultPreviewErr = "preview unavailable: CSV contains no data rows"
			}
		}
		m.validationErr = ""
		if event.err != nil {
			m.finalErr = event.err
		}
		return m, tea.ClearScreen
	}

	switch m.state {
	case screenMenu:
		return m.updateMenu(msg)
	case screenForm:
		return m.updateForm(msg)
	case screenMeasureBenchmarks, screenMeasureLanguages:
		return m.updateMeasureSelection(msg)
	case screenMeasureConfig:
		return m.updateMeasureConfig(msg)
	case screenRunning:
		updatedModel, runningCmd := m.updateRunning(msg)
		updated, ok := updatedModel.(model)
		if !ok {
			return updatedModel, runningCmd
		}

		// Only tick messages should advance the spinner animation.
		if _, isTick := msg.(spinner.TickMsg); !isTick {
			return updated, runningCmd
		}

		var spinnerCmd tea.Cmd
		updated.spinner, spinnerCmd = updated.spinner.Update(msg)
		return updated, tea.Batch(runningCmd, spinnerCmd)
	case screenResult:
		return m.updateResult(msg)
	default:
		return m, nil
	}
}

// newMeasureOptions maps plain labels into selectable options initialized as unselected.
func newMeasureOptions(labels []string) []measureOption {
	options := make([]measureOption, 0, len(labels))
	for _, label := range labels {
		options = append(options, measureOption{label: label})
	}

	return options
}

// selectedMeasureOptionLabels returns selected labels preserving option order.
func selectedMeasureOptionLabels(options []measureOption) []string {
	selected := make([]string, 0, len(options))
	for _, option := range options {
		if option.selected {
			selected = append(selected, option.label)
		}
	}

	return selected
}
