package constant

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownBenchmark is returned when a benchmark token cannot be mapped.
var ErrUnknownBenchmark = errors.New("unknown benchmark")

// ErrUnknownProgrammingLanguage is returned when a language token cannot be mapped.
var ErrUnknownProgrammingLanguage = errors.New("unknown programming language")

// ParseBenchmark normalizes free-form benchmark tokens and maps them to the
// canonical benchmark enum value.
func ParseBenchmark(value string) (Benchmark, error) {
	normalized := normalizeEnumToken(value)
	benchmark, ok := benchmarkByNormalized[normalized]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownBenchmark, value)
	}
	return benchmark, nil
}

// ParseProgrammingLanguage normalizes free-form language tokens and maps them
// to the canonical programming language enum value.
func ParseProgrammingLanguage(value string) (ProgrammingLanguage, error) {
	normalized := normalizeEnumToken(value)
	language, ok := languageByNormalized[normalized]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownProgrammingLanguage, value)
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
