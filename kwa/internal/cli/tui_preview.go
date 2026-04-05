package cli

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// panelContentWidth returns the approximate inner content width available
// inside the bordered panel after accounting for borders and padding.
func (m model) panelContentWidth() int {
	contentWidth := m.panelWidth() - 8
	if contentWidth < 46 {
		contentWidth = 46
	}

	return contentWidth
}

// newResultPreviewTable builds a non-interactive Bubble Tea table configured
// for the fixed four-column CSV preview and constrained to the panel width.
func newResultPreviewTable(rows []table.Row, contentWidth int) table.Model {
	columns := newResultPreviewColumns(contentWidth)
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithWidth(contentWidth),
		table.WithHeight(len(rows)+2),
	)
	t.Blur()
	t.SetStyles(newResultPreviewTableStyles())
	t.UpdateViewport()

	return t
}

// newResultPreviewColumns returns the fixed preview columns sized for one panel
// width so preview rows remain readable without horizontal overflow.
func newResultPreviewColumns(contentWidth int) []table.Column {
	runIDWidth, createdAtWidth, langWidth, benchmarkWidth := resultPreviewColumnWidths(contentWidth)

	return []table.Column{
		{Title: "Run ID", Width: runIDWidth},
		{Title: "Created At", Width: createdAtWidth},
		{Title: "Lang", Width: langWidth},
		{Title: "Benchmark", Width: benchmarkWidth},
	}
}

// resultPreviewColumnWidths computes fixed-column widths that fill available
// panel space while preserving minimum width for each preview field.
func resultPreviewColumnWidths(contentWidth int) (int, int, int, int) {
	available := contentWidth - 6
	if available < 45 {
		available = 45
	}

	// Keep timestamps fully visible (`YYYY-MM-DD HH:MM:SS`) and shrink run IDs first.
	runIDWidth := 18
	createdAtWidth := 19
	langWidth := 6
	benchmarkWidth := available - (runIDWidth + createdAtWidth + langWidth)

	// Keep benchmark readable; when space is tight, reduce run ID before other columns.
	if benchmarkWidth < 12 {
		deficit := 12 - benchmarkWidth
		benchmarkWidth = 12
		runIDWidth -= deficit
	}

	if runIDWidth < 10 {
		deficit := 10 - runIDWidth
		runIDWidth = 10
		benchmarkWidth -= deficit
	}

	if benchmarkWidth < 8 {
		benchmarkWidth = 8
	}

	return runIDWidth, createdAtWidth, langWidth, benchmarkWidth
}

// newResultPreviewTableStyles returns table styles aligned with the existing
// TUI palette for readable headers/cells in the result panel.
func newResultPreviewTableStyles() table.Styles {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		Bold(true).
		Foreground(lipgloss.Color("#44D17A")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)
	styles.Cell = styles.Cell.
		Foreground(lipgloss.Color("#E8FFE8"))
	styles.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E8FFE8"))

	return styles
}
