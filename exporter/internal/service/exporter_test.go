package service

import "testing"

func TestExporterServiceParseMeasurementFromPhase_DelegatesToParser(t *testing.T) {
	exporterService := NewExporterService(NewParserService())

	got, err := exporterService.ParseMeasurementFromPhase("005_Go-Binary-Trees", "12.34")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Language != "go" {
		t.Fatalf("language mismatch: got %q want %q", got.Language, "go")
	}
	if got.Benchmark != "binary-trees" {
		t.Fatalf("benchmark mismatch: got %q want %q", got.Benchmark, "binary-trees")
	}
	if got.Value != "12.34" {
		t.Fatalf("value mismatch: got %q want %q", got.Value, "12.34")
	}
}

func TestNewExporterService_UsesDefaultParserWhenNil(t *testing.T) {
	exporterService := NewExporterService(nil)

	_, err := exporterService.ParseMeasurementFromPhase("005_Go-Binary-Trees", "1")
	if err != nil {
		t.Fatalf("unexpected error with default parser: %v", err)
	}
}
