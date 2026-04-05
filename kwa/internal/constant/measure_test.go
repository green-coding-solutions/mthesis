package constant

import "testing"

func TestMeasureCatalogsMatchExpectedSizes(t *testing.T) {
	t.Parallel()

	if got := len(MeasureBenchmarks()); got != 8 {
		t.Fatalf("benchmark catalog size = %d, want 8", got)
	}
	if got := len(MeasureLanguages()); got != 18 {
		t.Fatalf("language catalog size = %d, want 18", got)
	}
}

func TestIsSupportedMeasureValues(t *testing.T) {
	t.Parallel()

	if !IsSupportedMeasureLanguage("go") {
		t.Fatalf("expected go to be supported")
	}
	if IsSupportedMeasureLanguage("pascal") {
		t.Fatalf("expected pascal to be unsupported")
	}
	if !IsSupportedMeasureBenchmark("binary-trees") {
		t.Fatalf("expected binary-trees to be supported")
	}
	if IsSupportedMeasureBenchmark("fib") {
		t.Fatalf("expected fib to be unsupported")
	}
}
