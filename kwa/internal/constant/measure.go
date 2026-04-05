package constant

import "strings"

var measureLanguages = programmingLanguagesToStrings(allProgrammingLanguages)

var measureBenchmarks = benchmarksToStrings(allMeasureBenchmarks)

var measureLanguageLookup = buildMeasureValueLookup(measureLanguages)
var measureBenchmarkLookup = buildMeasureValueLookup(measureBenchmarks)

// MeasureLanguages returns the ordered list of languages supported by the
// measure workflow and scripts/measure.sh integration.
func MeasureLanguages() []string {
	return append([]string(nil), measureLanguages...)
}

// MeasureBenchmarks returns the ordered list of benchmarks supported by the
// measure workflow and scripts/measure.sh integration.
func MeasureBenchmarks() []string {
	return append([]string(nil), measureBenchmarks...)
}

// IsSupportedMeasureLanguage reports whether value matches one of the supported
// measure languages after trimming whitespace.
func IsSupportedMeasureLanguage(value string) bool {
	trimmed := strings.TrimSpace(value)
	_, ok := measureLanguageLookup[trimmed]
	return ok
}

// IsSupportedMeasureBenchmark reports whether value matches one of the
// supported measure benchmarks after trimming whitespace.
func IsSupportedMeasureBenchmark(value string) bool {
	trimmed := strings.TrimSpace(value)
	_, ok := measureBenchmarkLookup[trimmed]
	return ok
}

// programmingLanguagesToStrings converts canonical language enums into ordered string values.
func programmingLanguagesToStrings(values []ProgrammingLanguage) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}

	return result
}

// benchmarksToStrings converts canonical benchmark enums into ordered string values.
func benchmarksToStrings(values []Benchmark) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}

	return result
}

// buildMeasureValueLookup creates O(1) membership lookup tables for measure catalogs.
func buildMeasureValueLookup(values []string) map[string]struct{} {
	lookup := make(map[string]struct{}, len(values))
	for _, value := range values {
		lookup[value] = struct{}{}
	}

	return lookup
}
