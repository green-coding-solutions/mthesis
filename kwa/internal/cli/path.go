package cli

import (
	"path/filepath"
	"strings"

	"mthesis/kwa/internal/constant"
)

// buildOutputPath applies the interactive filename rules and returns the final path.
// Rule order:
// 1) Empty input falls back to measurements.csv.
// 2) Missing .csv extension is auto-appended.
// 3) Any value containing "/" is treated as a path.
// 4) Plain filenames are written under results/.
func buildOutputPath(input string) string {
	filenameOrPath := normalizeCSVFilename(input)
	if strings.Contains(filenameOrPath, "/") {
		return filenameOrPath
	}

	return filepath.Join("results", filenameOrPath)
}

// normalizeCSVFilename normalizes user input into a CSV filename or path.
func normalizeCSVFilename(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return constant.DefaultCSVFilename
	}
	if strings.HasSuffix(strings.ToLower(trimmed), ".csv") {
		return trimmed
	}

	return trimmed + ".csv"
}
