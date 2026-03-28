package service

import "testing"

func TestParseMeasurementFromPhase(t *testing.T) {
	parserService := NewParserService()

	tests := []struct {
		name          string
		phase         string
		value         string
		wantLanguage  string
		wantBenchmark string
		wantValue     string
		wantErr       bool
	}{
		{
			name:          "parses go binary trees",
			phase:         "005_Go-Binary-Trees",
			value:         "12.34",
			wantLanguage:  "go",
			wantBenchmark: "binary-trees",
			wantValue:     "12.34",
			wantErr:       false,
		},
		{
			name:          "parses lowercase with multi-word benchmark",
			phase:         "009_python-regex-redux",
			value:         "98",
			wantLanguage:  "python",
			wantBenchmark: "regex-redux",
			wantValue:     "98",
			wantErr:       false,
		},
		{
			name:    "fails on invalid phase format",
			phase:   "005GoBinaryTrees",
			value:   "1",
			wantErr: true,
		},
		{
			name:    "fails on unknown language",
			phase:   "005_Goo-Binary-Trees",
			value:   "1",
			wantErr: true,
		},
		{
			name:    "fails on unknown benchmark",
			phase:   "005_Go-Binary-Treez",
			value:   "1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parserService.ParseMeasurementFromPhase(tt.phase, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Language != tt.wantLanguage {
				t.Fatalf("language mismatch: got %q want %q", got.Language, tt.wantLanguage)
			}
			if got.Benchmark != tt.wantBenchmark {
				t.Fatalf("benchmark mismatch: got %q want %q", got.Benchmark, tt.wantBenchmark)
			}
			if got.Value != tt.wantValue {
				t.Fatalf("value mismatch: got %q want %q", got.Value, tt.wantValue)
			}
		})
	}
}
