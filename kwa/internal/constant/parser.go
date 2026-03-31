package constant

import (
	"fmt"
	"strings"
)

// ParseBenchmark normalizes free-form benchmark tokens and maps them to the
// canonical benchmark enum value.
func ParseBenchmark(value string) (Benchmark, error) {
	normalized := normalizeEnumToken(value)
	benchmark, ok := benchmarkByNormalized[normalized]
	if !ok {
		return "", fmt.Errorf("unknown benchmark: %q", value)
	}
	return benchmark, nil
}

// ParseProgrammingLanguage normalizes free-form language tokens and maps them
// to the canonical programming language enum value.
func ParseProgrammingLanguage(value string) (ProgrammingLanguage, error) {
	normalized := normalizeEnumToken(value)
	language, ok := languageByNormalized[normalized]
	if !ok {
		return "", fmt.Errorf("unknown programming language: %q", value)
	}
	return language, nil
}

// normalizeEnumToken standardizes parser inputs by trimming whitespace,
// lowercasing, and removing separators so aliases can be matched reliably.
func normalizeEnumToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}
