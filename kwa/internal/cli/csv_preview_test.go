package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCSVPreviewRowsLimit(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "measurements.csv")
	csv := strings.Join([]string{
		"run_id,measured_at,language,benchmark,metric",
		"run-1,2026-04-05 00:00:01,go,binary-trees,1",
		"run-2,2026-04-05 00:00:02,c,binary-trees,2",
		"run-3,2026-04-05 00:00:03,rust,binary-trees,3",
		"run-4,2026-04-05 00:00:04,python,binary-trees,4",
		"run-5,2026-04-05 00:00:05,java,binary-trees,5",
		"run-6,2026-04-05 00:00:06,julia,binary-trees,6",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(csv), 0o600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	rows, err := readCSVPreviewRows(filePath, 5)
	if err != nil {
		t.Fatalf("readCSVPreviewRows() error = %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("row count = %d, want %d", len(rows), 5)
	}
	if rows[0][0] != "run-1" || rows[0][2] != "go" || rows[0][3] != "binary-trees" {
		t.Fatalf("first row = %#v, want run-1/go/binary-trees", rows[0])
	}
	if rows[4][0] != "run-5" {
		t.Fatalf("fifth row run_id = %q, want %q", rows[4][0], "run-5")
	}
}

func TestReadCSVPreviewRowsFallbackHeaders(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "measurements.csv")
	csv := strings.Join([]string{
		"run_id,created_at,lang,benchmark",
		"run-1,2026-04-05 00:00:01,go,binary-trees",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(csv), 0o600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	rows, err := readCSVPreviewRows(filePath, 5)
	if err != nil {
		t.Fatalf("readCSVPreviewRows() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want %d", len(rows), 1)
	}
	if rows[0][1] != "2026-04-05 00:00:01" || rows[0][2] != "go" {
		t.Fatalf("row = %#v, want created_at/lang values", rows[0])
	}
}

func TestReadCSVPreviewRowsMissingHeader(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "measurements.csv")
	csv := strings.Join([]string{
		"run_id,measured_at,language",
		"run-1,2026-04-05 00:00:01,go",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(csv), 0o600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	_, err := readCSVPreviewRows(filePath, 5)
	if err == nil {
		t.Fatalf("expected error for missing benchmark header")
	}
	if !strings.Contains(err.Error(), "missing benchmark column") {
		t.Fatalf("error = %q, want missing benchmark column", err.Error())
	}
}

func TestReadCSVPreviewRowsAllowsEmptyData(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "measurements.csv")
	csv := "run_id,measured_at,language,benchmark\n"
	if err := os.WriteFile(filePath, []byte(csv), 0o600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	rows, err := readCSVPreviewRows(filePath, 5)
	if err != nil {
		t.Fatalf("readCSVPreviewRows() error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("row count = %d, want %d", len(rows), 0)
	}
}
