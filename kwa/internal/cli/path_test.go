package cli

import "testing"

func TestNormalizeCSVFilename(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "measurements.csv"},
		{name: "trim and append", input: "  export  ", want: "export.csv"},
		{name: "already csv", input: "report.csv", want: "report.csv"},
		{name: "uppercase csv suffix", input: "report.CSV", want: "report.CSV"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeCSVFilename(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeCSVFilename() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildOutputPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "default result dir", input: "measurements", want: "results/measurements.csv"},
		{name: "existing extension", input: "metrics.csv", want: "results/metrics.csv"},
		{name: "slash path", input: "exports/metrics", want: "exports/metrics.csv"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildOutputPath(tc.input)
			if got != tc.want {
				t.Fatalf("buildOutputPath() = %q, want %q", got, tc.want)
			}
		})
	}
}
