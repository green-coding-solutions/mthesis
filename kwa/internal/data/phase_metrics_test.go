package data

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"mthesis/kwa/internal/entity"
)

func TestGetPhaseMetricsByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	runID := "ff37312e-45a2-4f3d-ae3e-6f9680a0f335"
	createdAtOne := time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC)
	createdAtTwo := time.Date(2026, time.April, 1, 9, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"run_id", "measured_at", "phase", "metrics"}).
		AddRow(
			runID,
			createdAtOne,
			"005_Go-Binary-Trees",
			[]byte(`{"cpu_time_powermetrics_vm-docker_vm-ns":47560725453,"gpu_carbon_powermetrics_component-component-ug":13}`),
		).
		AddRow(runID, createdAtTwo, "006_Go-Fasta", nil)

	mock.ExpectQuery(regexp.QuoteMeta(getPhaseMetricsQueryByID)).
		WithArgs(runID).
		WillReturnRows(rows)

	svc := &service{db: db}
	got, err := svc.GetPhaseMetricsByID(context.Background(), runID)
	if err != nil {
		t.Fatalf("GetPhaseMetricsByID() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	if got[0].RunID != runID {
		t.Fatalf("got[0].RunID = %q, want %q", got[0].RunID, runID)
	}
	if got[0].Phase != "005_Go-Binary-Trees" {
		t.Fatalf("got[0].Phase = %q, want %q", got[0].Phase, "005_Go-Binary-Trees")
	}
	if !got[0].MeasuredAt.Equal(createdAtOne) {
		t.Fatalf("got[0].MeasuredAt = %v, want %v", got[0].MeasuredAt, createdAtOne)
	}
	if got[0].Metrics["cpu_time_powermetrics_vm-docker_vm-ns"] != 47560725453 {
		t.Fatalf("unexpected cpu_time value: got %d", got[0].Metrics["cpu_time_powermetrics_vm-docker_vm-ns"])
	}
	if got[0].Metrics["non_existing_metric"] != 0 {
		t.Fatalf("missing key default must be 0; got %d", got[0].Metrics["non_existing_metric"])
	}
	if got[1].Metrics == nil {
		t.Fatalf("got[1].Metrics must not be nil")
	}
	if len(got[1].Metrics) != 0 {
		t.Fatalf("len(got[1].Metrics) = %d, want 0", len(got[1].Metrics))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetPhaseMetricsByID_EmptyRunID(t *testing.T) {
	svc := &service{}
	_, err := svc.GetPhaseMetricsByID(context.Background(), " ")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestGetPhaseMetricsByID_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	runID := "run-id"
	mock.ExpectQuery(regexp.QuoteMeta(getPhaseMetricsQueryByID)).
		WithArgs(runID).
		WillReturnError(errors.New("db down"))

	svc := &service{db: db}
	_, err = svc.GetPhaseMetricsByID(context.Background(), runID)
	if err == nil {
		t.Fatalf("GetPhaseMetricsByID() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "query phase metrics") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPhaseMetricsByID_InvalidMetricsJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	runID := "run-id"
	rows := sqlmock.NewRows([]string{"run_id", "measured_at", "phase", "metrics"}).
		AddRow(runID, time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC), "005_Go-Binary-Trees", []byte(`{"invalid-json"`))

	mock.ExpectQuery(regexp.QuoteMeta(getPhaseMetricsQueryByID)).
		WithArgs(runID).
		WillReturnRows(rows)

	svc := &service{db: db}
	_, err = svc.GetPhaseMetricsByID(context.Background(), runID)
	if err == nil {
		t.Fatalf("GetPhaseMetricsByID() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "decode metrics json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPhaseMetricsBatch_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRows([]string{"run_id", "measured_at", "phase", "metrics"}).
		AddRow("run-1", time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC), "005_Go-Binary-Trees", []byte(`{"cpu_time_powermetrics_vm-docker_vm-ns":47560725453}`)).
		AddRow("run-2", time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC), "006_Go-Fasta", []byte(`{"gpu_carbon_powermetrics_component-component-ug":13}`))

	mock.ExpectQuery(regexp.QuoteMeta(getPhaseMetricsBatchQuery)).
		WithArgs(2, 1, nil, nil).
		WillReturnRows(rows)

	svc := &service{db: db}
	got, err := svc.GetPhaseMetricsBatch(context.Background(), 2, 1, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("GetPhaseMetricsBatch() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].RunID != "run-1" || got[0].Phase != "005_Go-Binary-Trees" {
		t.Fatalf("unexpected first row: %#v", got[0])
	}
	if got[1].RunID != "run-2" || got[1].Phase != "006_Go-Fasta" {
		t.Fatalf("unexpected second row: %#v", got[1])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetPhaseMetricsBatch_InvalidPagination(t *testing.T) {
	svc := &service{}

	_, err := svc.GetPhaseMetricsBatch(context.Background(), 0, 0, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error for invalid limit, got nil")
	}

	_, err = svc.GetPhaseMetricsBatch(context.Background(), 1, -1, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error for invalid offset, got nil")
	}
}

func TestGetPhaseMetricsBatch_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	mock.ExpectQuery(regexp.QuoteMeta(getPhaseMetricsBatchQuery)).
		WithArgs(50, 100, nil, nil).
		WillReturnError(errors.New("db down"))

	svc := &service{db: db}
	_, err = svc.GetPhaseMetricsBatch(context.Background(), 50, 100, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("GetPhaseMetricsBatch() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "query phase metrics batch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPhaseMetricsBatch_InvalidMetricsJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRows([]string{"run_id", "measured_at", "phase", "metrics"}).
		AddRow("run-1", time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC), "005_Go-Binary-Trees", []byte(`{"invalid-json"`))

	mock.ExpectQuery(regexp.QuoteMeta(getPhaseMetricsBatchQuery)).
		WithArgs(10, 0, nil, nil).
		WillReturnRows(rows)

	svc := &service{db: db}
	_, err = svc.GetPhaseMetricsBatch(context.Background(), 10, 0, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("GetPhaseMetricsBatch() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "decode metrics json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetMetricKeys_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRows([]string{"k"}).
		AddRow("cpu_time_powermetrics_vm-docker_vm-ns").
		AddRow("gpu_carbon_powermetrics_component-component-ug")

	mock.ExpectQuery(regexp.QuoteMeta(getMetricKeysQuery)).
		WillReturnRows(rows)

	svc := &service{db: db}
	got, err := svc.GetMetricKeys(context.Background())
	if err != nil {
		t.Fatalf("GetMetricKeys() error = %v", err)
	}

	want := []string{
		"cpu_time_powermetrics_vm-docker_vm-ns",
		"gpu_carbon_powermetrics_component-component-ug",
	}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetMetricKeys_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	mock.ExpectQuery(regexp.QuoteMeta(getMetricKeysQuery)).
		WillReturnError(errors.New("db down"))

	svc := &service{db: db}
	_, err = svc.GetMetricKeys(context.Background())
	if err == nil {
		t.Fatalf("GetMetricKeys() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "query metric keys") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPhaseMetricsQueryByID_ContainsNormalizationAndDedup(t *testing.T) {
	requiredFragments := []string{
		"SELECT",
		"run_id,",
		"created_at,",
		"concat_ws(",
		"regexp_replace(lower(metric), '[^a-z0-9]+', '_', 'g')",
		"regexp_replace(lower(detail_name), '[^a-z0-9]+', '_', 'g')",
		"regexp_replace(lower(unit), '[^a-z0-9]+', '_', 'g')",
		"phase !~* '\\[(baseline|installation|boot|idle|runtime|remove)\\]'",
		"MAX(value) AS value",
		"MAX(created_at) AS measured_at",
		"jsonb_object_agg(k, value ORDER BY k)",
		"GROUP BY run_id, phase",
		"ORDER BY MAX(created_at) DESC, run_id, phase",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(getPhaseMetricsQueryByID, fragment) {
			t.Fatalf("query is missing required fragment %q", fragment)
		}
	}

	if strings.Contains(getPhaseMetricsQueryByID, "created_at BETWEEN") {
		t.Fatalf("by-id query must not contain timestamp range filtering")
	}
}

func TestGetPhaseMetricsBatchQuery_ContainsPagination(t *testing.T) {
	requiredFragments := []string{
		"GROUP BY run_id, phase",
		"created_at BETWEEN $3::timestamp AND $4::timestamp",
		"ORDER BY MAX(created_at) DESC, run_id, phase",
		"LIMIT $1 OFFSET $2",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(getPhaseMetricsBatchQuery, fragment) {
			t.Fatalf("batch query is missing required fragment %q", fragment)
		}
	}
}

func TestGetPhaseMetricsBatch_InvalidDateRange(t *testing.T) {
	svc := &service{}
	from := time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)

	if _, err := svc.GetPhaseMetricsBatch(context.Background(), 1, 0, entity.TimeRangeFilter{From: &from}); err == nil {
		t.Fatalf("expected partial bounds error")
	}
	if _, err := svc.GetPhaseMetricsBatch(context.Background(), 1, 0, entity.TimeRangeFilter{From: &from, To: &to}); err == nil {
		t.Fatalf("expected inverted bounds error")
	}
}

func TestGetMetricKeysQuery_ContainsNormalizationAndOrdering(t *testing.T) {
	requiredFragments := []string{
		"SELECT DISTINCT",
		"concat_ws(",
		"phase !~* '\\[(baseline|installation|boot|idle|runtime|remove)\\]'",
		"ORDER BY k",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(getMetricKeysQuery, fragment) {
			t.Fatalf("metric keys query is missing required fragment %q", fragment)
		}
	}
}
