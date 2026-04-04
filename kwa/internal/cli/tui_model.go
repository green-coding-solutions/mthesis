package cli

import (
	"context"
	"fmt"
	"io"

	appexport "mthesis/kwa/internal/app/export"
	"mthesis/kwa/internal/constant"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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

type fieldSpec struct {
	label       string
	placeholder string
}

type fieldState struct {
	spec  fieldSpec
	value string
}

type exportDoneMsg struct {
	path string
	err  error
}

type model struct {
	ctx      context.Context
	executor executeRequestFunc

	state     screenState
	selected  menuOption
	formMode  constant.ExportMode
	formTitle string
	fields    []fieldState
	focus     int

	validationErr string
	runningReq    appexport.Request
	spinner       spinner.Model

	resultPath string
	resultErr  error
	finalErr   error

	width  int
	height int
}

// runInteractive starts the Bubble Tea application used by the root `kwa` command.
func runInteractive(ctx context.Context, execute executeRequestFunc, _ io.Writer, _ io.Writer) error {
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

func newModel(ctx context.Context, execute executeRequestFunc) model {
	return model{
		ctx:      ctx,
		executor: execute,
		state:    screenMenu,
		selected: menuBatch,
		formMode: constant.ExportModeBatch,
		spinner:  newSpinnerModel(),
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

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = event.Width
		m.height = event.Height
		return m, nil
	case tea.MouseMsg:
		// Ignore all mouse input globally (wheel, click, movement).
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
	}

	switch m.state {
	case screenMenu:
		return m.updateMenu(msg)
	case screenForm:
		return m.updateForm(msg)
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
