package cli

import (
	"fmt"
	"strings"
)

const batLogo = `
    /\                 /\
   / '._   (\_/)   _.' \
  /_.''._'--(.)--'_.''._\
  | \_ /  _   _  \ _/ |
   \/   \__\___/__\/
        /_/   \_\
`

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

	return tuiStyles.doc.Render(content)
}

func (m model) viewMenu() string {
	panelWidth := m.panelWidth()
	logo := m.renderMenuLogo(panelWidth)
	items := []string{}
	labels := []string{"batch export", "byID export"}
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
		tuiStyles.hint.Render("↑/↓ move • Enter select • q quit"),
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
	message := fmt.Sprintf("%s Export in progress", m.spinner.View())

	body := strings.Join([]string{
		tuiStyles.sectionTitle.Render("Running"),
		message,
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
