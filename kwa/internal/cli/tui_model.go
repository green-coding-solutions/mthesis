package cli

import (
	"context"
	"fmt"
	"io"

	appexport "mthesis/kwa/internal/app/export"
	"mthesis/kwa/internal/constant"

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

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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

type spinnerTickMsg struct{}

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
	spinnerIndex  int

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
