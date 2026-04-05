package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const batLogo = `
    /\                 /\
   / '._   (\_/)   _.' \
  /_.''._'--(.)--'_.''._\
  | \_ /  _   _  \ _/ |
   \/   \__\___/__\/
        /_/   \_\
`

// View renders the screen-specific content wrapped by the shared document style.
func (m model) View() string {
	var content string
	switch m.state {
	case screenMenu:
		content = m.viewMenu()
	case screenForm:
		content = m.viewForm()
	case screenMeasureBenchmarks, screenMeasureLanguages:
		content = m.viewMeasureSelection()
	case screenMeasureConfig:
		content = m.viewMeasureConfig()
	case screenRunning:
		content = m.viewRunning()
	case screenResult:
		content = m.viewResult()
	default:
		content = ""
	}

	return tuiStyles.doc.Render(content)
}

// viewMenu renders the top-level workflow picker.
func (m model) viewMenu() string {
	panelWidth := m.panelWidth()
	logo := m.renderMenuLogo(panelWidth)
	items := []string{}
	labels := []string{"Export (Batch mode)", "Export (by Run ID)", "Measure"}
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
		tuiStyles.subtitle.Render("Pick a workflow to continue"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("↑/↓ move • Enter select • Esc quit"),
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

// viewForm renders batch/by-id field input screens.
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
		tuiStyles.hint.Render("↑/↓ focus • Enter next/submit • ctrl+u clear • Esc quit"),
	}, "\n")
}

// viewMeasureSelection renders benchmark/language multi-select screens.
func (m model) viewMeasureSelection() string {
	panelWidth := m.panelWidth()
	options := m.measureBenchmarks
	sectionTitle := "Measure // Benchmarks"
	subtitle := "Select one or more benchmarks"
	if m.state == screenMeasureLanguages {
		options = m.measureLanguages
		sectionTitle = "Measure // Languages"
		subtitle = "Select one or more programming languages"
	}

	lines := make([]string, 0, len(options))
	for i, option := range options {
		checkmark := "[ ]"
		if option.selected {
			checkmark = "[x]"
		}

		line := fmt.Sprintf("  %s %s", checkmark, option.label)
		if i == m.measureCursor {
			line = fmt.Sprintf("> %s %s", checkmark, option.label)
			lines = append(lines, tuiStyles.menuItemSelected.Render(truncateText(line, panelWidth-8)))
			continue
		}

		lines = append(lines, tuiStyles.menuItem.Render(truncateText(line, panelWidth-6)))
	}

	if len(lines) == 0 {
		lines = append(lines, tuiStyles.error.Render("no options available"))
	}

	if m.validationErr != "" {
		lines = append(lines, "", tuiStyles.error.Render(m.validationErr))
	}

	body := strings.Join([]string{
		tuiStyles.sectionTitle.Render(sectionTitle),
		tuiStyles.label.Render(subtitle),
		"",
		strings.Join(lines, "\n"),
	}, "\n")

	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Build your measure scope"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("↑/↓ move • Space toggle • A toggle all • Enter continue • Esc quit"),
	}, "\n")
}

// viewMeasureConfig renders iterations and fileName inputs after selections are complete.
func (m model) viewMeasureConfig() string {
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
		tuiStyles.sectionTitle.Render("Measure // Config"),
		"",
		strings.Join(sections, "\n\n"),
	}, "\n")

	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Set iterations and output filename"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("↑/↓ focus • Enter next/submit • ctrl+u clear • Esc quit"),
	}, "\n")
}

// renderFieldValue renders one input field value or placeholder with focus styling.
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

// viewRunning renders a single progress panel while async work is in flight.
func (m model) viewRunning() string {
	panelWidth := m.panelWidth()
	message := fmt.Sprintf("%s Export in progress", m.spinner.View())
	if m.runningLabel == "measure + export" {
		message = fmt.Sprintf("%s Measure and export in progress", m.spinner.View())
	}

	taskLabel := m.runningLabel
	if strings.TrimSpace(taskLabel) == "" {
		taskLabel = "workflow"
	}

	lines := []string{
		tuiStyles.sectionTitle.Render("Running"),
		message,
		"",
		tuiStyles.label.Render(fmt.Sprintf("task: %s", taskLabel)),
		tuiStyles.label.Render(fmt.Sprintf("output: %s", truncateText(m.runningOutPath, panelWidth-12))),
	}

	if m.runningLabel == "measure + export" {
		languages := strings.Join(m.runningMeasureLanguages, ", ")
		if strings.TrimSpace(languages) == "" {
			languages = "-"
		}

		benchmarks := strings.Join(m.runningMeasureBenchmarks, ", ")
		if strings.TrimSpace(benchmarks) == "" {
			benchmarks = "-"
		}

		iterations := "-"
		if m.runningMeasureIterations > 0 {
			iterations = fmt.Sprintf("%d", m.runningMeasureIterations)
		}

		lines = append(lines, "")
		lines = append(lines, tuiStyles.label.Render(fmt.Sprintf("languages: %s", truncateText(languages, panelWidth-14))))
		lines = append(lines, tuiStyles.label.Render(fmt.Sprintf("benchmarks: %s", truncateText(benchmarks, panelWidth-15))))
		lines = append(lines, tuiStyles.label.Render(fmt.Sprintf("iterations: %s", iterations)))
	}

	if m.runningQuitPromptVisible {
		lines = append(lines, "")
		lines = append(lines, tuiStyles.sectionTitle.Render("Confirm Quit"))
		lines = append(lines, tuiStyles.label.Render("Type yes and press Enter to quit."))
		if strings.TrimSpace(m.runningQuitReminder) != "" {
			lines = append(lines, tuiStyles.error.Render(m.runningQuitReminder))
		}
		lines = append(lines, m.renderRunningQuitPromptLine(panelWidth))
	}

	body := strings.Join(lines, "\n")
	hint := "Esc quit"
	if m.runningQuitPromptVisible {
		hint = "Type yes + Enter quit • Tab switch • Enter on No cancel • Esc cancel"
	}

	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Working on your request"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render(hint),
	}, "\n")
}

// renderRunningQuitPromptLine renders the running-screen quit confirmation
// input plus the adjacent "No" button.
func (m model) renderRunningQuitPromptLine(panelWidth int) string {
	inputWidth := panelWidth - 22
	if inputWidth < 14 {
		inputWidth = 14
	}

	visibleInput := strings.TrimSpace(m.runningQuitInput)
	inputText := ""
	if visibleInput == "" {
		inputText = tuiStyles.inputPlaceholder.Render("type yes")
	} else {
		inputText = tuiStyles.inputText.Render(truncateText(m.runningQuitInput, inputWidth-3))
	}

	var inputField string
	if m.runningQuitNoFocused {
		inputField = tuiStyles.input.Width(inputWidth).Render(inputText)
	} else {
		inputField = tuiStyles.inputFocused.Width(inputWidth).Render(inputText + tuiStyles.inputCursor.Render("▌"))
	}

	noButton := tuiStyles.button.Render("No")
	if m.runningQuitNoFocused {
		noButton = tuiStyles.buttonFocused.Render("No")
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, inputField, " ", noButton)
}

// viewResult renders the final success/failure screen and output path.
func (m model) viewResult() string {
	panelWidth := m.panelWidth()
	contentWidth := m.panelContentWidth()

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
		lines = append(lines, tuiStyles.error.Render(wrapText(m.resultErr.Error(), panelWidth-8)))
	}
	if m.resultErr == nil {
		lines = append(lines, "", tuiStyles.sectionTitle.Render("CSV Preview"))
		if m.resultPreviewAvailable {
			previewTable := m.resultPreviewTable
			previewTable.SetColumns(newResultPreviewColumns(contentWidth))
			previewTable.SetWidth(contentWidth)
			previewTable.SetHeight(len(previewTable.Rows()) + 2)
			previewTable.UpdateViewport()
			lines = append(lines, previewTable.View())
		} else {
			previewMessage := m.resultPreviewErr
			if strings.TrimSpace(previewMessage) == "" {
				previewMessage = "preview unavailable"
			}
			lines = append(lines, tuiStyles.label.Render(wrapText(previewMessage, panelWidth-8)))
		}
	}

	body := strings.Join(lines, "\n")
	return strings.Join([]string{
		tuiStyles.title.Render("KWA // Green Metrics CLI"),
		tuiStyles.subtitle.Render("Press Esc to leave the app"),
		tuiStyles.panel.Width(panelWidth).Render(body),
		tuiStyles.hint.Render("Esc exit"),
	}, "\n")
}

// panelWidth clamps the central panel width using terminal size constraints.
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

// truncateText shortens a string to max runes and appends an ellipsis when needed.
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

// wrapText breaks text into fixed-width lines so long errors remain fully visible.
func wrapText(value string, max int) string {
	if max <= 0 {
		return value
	}

	inputLines := strings.Split(value, "\n")
	wrapped := make([]string, 0, len(inputLines))
	for _, line := range inputLines {
		runes := []rune(line)
		if len(runes) == 0 {
			wrapped = append(wrapped, "")
			continue
		}

		for len(runes) > max {
			wrapped = append(wrapped, string(runes[:max]))
			runes = runes[max:]
		}
		wrapped = append(wrapped, string(runes))
	}

	return strings.Join(wrapped, "\n")
}
