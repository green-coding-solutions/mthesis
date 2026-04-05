package cli

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
)

const csvPreviewRowLimit = 10

// csvPreviewColumnIndexes stores source CSV indexes for preview fields.
type csvPreviewColumnIndexes struct {
	runID     int
	createdAt int
	lang      int
	benchmark int
}

// readCSVPreviewRows reads the first `limit` data rows from one CSV file and
// returns only the fixed preview columns (run ID, created at, lang, benchmark).
// It returns an error for file/header/row parsing failures and nil rows when no
// data rows exist.
func readCSVPreviewRows(path string, limit int) ([]table.Row, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("preview unavailable: empty csv path")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("preview unavailable: invalid row limit %d", limit)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("preview unavailable: open csv %q: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("preview unavailable: csv %q is empty", path)
		}
		return nil, fmt.Errorf("preview unavailable: read csv header from %q: %w", path, err)
	}

	indexes, err := resolveCSVPreviewColumnIndexes(header)
	if err != nil {
		return nil, fmt.Errorf("preview unavailable: %w", err)
	}

	rows := make([]table.Row, 0, limit)
	for len(rows) < limit {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("preview unavailable: read csv row from %q: %w", path, readErr)
		}

		rows = append(rows, table.Row{
			csvFieldAt(record, indexes.runID),
			csvFieldAt(record, indexes.createdAt),
			csvFieldAt(record, indexes.lang),
			csvFieldAt(record, indexes.benchmark),
		})
	}

	return rows, nil
}

// resolveCSVPreviewColumnIndexes maps required preview columns to header
// positions using case-insensitive lookup and fallback aliases.
func resolveCSVPreviewColumnIndexes(header []string) (csvPreviewColumnIndexes, error) {
	headerIndex := buildCSVHeaderIndex(header)

	runID, ok := firstCSVHeaderIndex(headerIndex, "run_id")
	if !ok {
		return csvPreviewColumnIndexes{}, fmt.Errorf("missing run_id column")
	}

	createdAt, ok := firstCSVHeaderIndex(headerIndex, "created_at", "measured_at")
	if !ok {
		return csvPreviewColumnIndexes{}, fmt.Errorf("missing created_at/measured_at column")
	}

	lang, ok := firstCSVHeaderIndex(headerIndex, "lang", "language")
	if !ok {
		return csvPreviewColumnIndexes{}, fmt.Errorf("missing lang/language column")
	}

	benchmark, ok := firstCSVHeaderIndex(headerIndex, "benchmark")
	if !ok {
		return csvPreviewColumnIndexes{}, fmt.Errorf("missing benchmark column")
	}

	return csvPreviewColumnIndexes{
		runID:     runID,
		createdAt: createdAt,
		lang:      lang,
		benchmark: benchmark,
	}, nil
}

// buildCSVHeaderIndex creates a normalized header lookup map where keys are
// lowercased, trimmed header names and values are their first column indexes.
func buildCSVHeaderIndex(header []string) map[string]int {
	index := make(map[string]int, len(header))
	for i, cell := range header {
		normalized := strings.ToLower(strings.TrimSpace(cell))
		if normalized == "" {
			continue
		}
		if _, exists := index[normalized]; exists {
			continue
		}
		index[normalized] = i
	}

	return index
}

// firstCSVHeaderIndex returns the first present header index from one alias list.
func firstCSVHeaderIndex(index map[string]int, aliases ...string) (int, bool) {
	for _, alias := range aliases {
		if value, ok := index[alias]; ok {
			return value, true
		}
	}

	return 0, false
}

// csvFieldAt returns one CSV cell by index and falls back to an empty value for
// short/invalid records to keep preview extraction resilient.
func csvFieldAt(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}

	return strings.TrimSpace(record[index])
}
