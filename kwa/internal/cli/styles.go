package cli

import "github.com/charmbracelet/lipgloss"

type styleSet struct {
	doc              lipgloss.Style
	title            lipgloss.Style
	subtitle         lipgloss.Style
	logo             lipgloss.Style
	panel            lipgloss.Style
	sectionTitle     lipgloss.Style
	menuItem         lipgloss.Style
	menuItemSelected lipgloss.Style
	label            lipgloss.Style
	labelFocused     lipgloss.Style
	input            lipgloss.Style
	inputFocused     lipgloss.Style
	error            lipgloss.Style
	success          lipgloss.Style
	hint             lipgloss.Style
	spinner          lipgloss.Style
	inputPrompt      lipgloss.Style
	inputText        lipgloss.Style
	inputPlaceholder lipgloss.Style
	inputCursor      lipgloss.Style
	button           lipgloss.Style
	buttonFocused    lipgloss.Style
}

var tuiStyles = newStyleSet()

func newStyleSet() styleSet {
	const (
		bgBase       = "#04160B"
		fgPrimary    = "#E8FFE8"
		fgMuted      = "#8DBA96"
		accent       = "#44D17A"
		accentDeep   = "#1E7A45"
		borderMuted  = "#2A5B3A"
		errorRed     = "#FF6B6B"
		successGreen = "#7CFF9E"
		selectedFG   = "#06120A"
		placeholder  = "#5E7F66"
	)

	return styleSet{
		doc: lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(lipgloss.Color(fgPrimary)),
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(accent)).
			Background(lipgloss.Color(bgBase)).
			Padding(0, 1),
		subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(fgMuted)).
			Padding(0, 1),
		logo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(accent)).
			Bold(true),
		panel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(accentDeep)).
			Padding(1, 2),
		sectionTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(accent)),
		menuItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(fgPrimary)),
		menuItemSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(selectedFG)).
			Background(lipgloss.Color(accent)).
			Padding(0, 1),
		label: lipgloss.NewStyle().
			Foreground(lipgloss.Color(fgMuted)),
		labelFocused: lipgloss.NewStyle().
			Foreground(lipgloss.Color(accent)).
			Bold(true),
		input: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(borderMuted)).
			Padding(0, 1),
		inputFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(accent)).
			Padding(0, 1),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(errorRed)).
			Bold(true),
		success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(successGreen)).
			Bold(true),
		hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(fgMuted)).
			Italic(true),
		spinner: lipgloss.NewStyle().
			Foreground(lipgloss.Color(accent)),
		inputPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color(accent)),
		inputText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(fgPrimary)),
		inputPlaceholder: lipgloss.NewStyle().
			Foreground(lipgloss.Color(placeholder)),
		inputCursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color(accent)),
		button: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(borderMuted)).
			Foreground(lipgloss.Color(fgMuted)).
			Bold(true).
			Padding(0, 2),
		buttonFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(accent)).
			Background(lipgloss.Color(accent)).
			Foreground(lipgloss.Color(selectedFG)).
			Bold(true).
			Padding(0, 2),
	}
}
