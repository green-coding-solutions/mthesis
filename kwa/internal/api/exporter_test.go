package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"mthesis/kwa/internal/entity"
)

type batchCall struct {
	batchSize int
	filter    entity.TimeRangeFilter
}

type byIDCall struct {
	runID string
}

type fakeExporter struct {
	batchErr error
	byIDErr  error

	batchCalls []batchCall
	byIDCalls  []byIDCall
}

func (f *fakeExporter) ExportMeasurementsCSV(_ context.Context, _ io.Writer, batchSize int, filter entity.TimeRangeFilter) error {
	f.batchCalls = append(f.batchCalls, batchCall{
		batchSize: batchSize,
		filter:    filter.Clone(),
	})
	return f.batchErr
}

func (f *fakeExporter) ExportMeasurementsCSVByID(_ context.Context, _ io.Writer, runID string) error {
	f.byIDCalls = append(f.byIDCalls, byIDCall{
		runID: runID,
	})
	return f.byIDErr
}

func TestExportBatch_Success(t *testing.T) {
	exp := &fakeExporter{}
	handler := NewCLIHandler(exp)

	err := handler.ExportBatch(context.Background(), &bytes.Buffer{}, 50, entity.TimeRangeFilter{})
	if err != nil {
		t.Fatalf("ExportBatch() error = %v", err)
	}

	if !reflect.DeepEqual(exp.batchCalls, []batchCall{{batchSize: 50, filter: entity.TimeRangeFilter{}}}) {
		t.Fatalf("batch calls mismatch: got=%#v want=%#v", exp.batchCalls, []batchCall{{batchSize: 50, filter: entity.TimeRangeFilter{}}})
	}
}

func TestExportBatch_Error(t *testing.T) {
	exp := &fakeExporter{batchErr: errors.New("boom")}
	handler := NewCLIHandler(exp)

	err := handler.ExportBatch(context.Background(), &bytes.Buffer{}, 10, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportBatch_NilExporter(t *testing.T) {
	handler := NewCLIHandler(nil)

	err := handler.ExportBatch(context.Background(), &bytes.Buffer{}, 10, entity.TimeRangeFilter{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportByID_Success(t *testing.T) {
	exp := &fakeExporter{}
	handler := NewCLIHandler(exp)

	err := handler.ExportByID(context.Background(), &bytes.Buffer{}, "run-1")
	if err != nil {
		t.Fatalf("ExportByID() error = %v", err)
	}

	if !reflect.DeepEqual(exp.byIDCalls, []byIDCall{{runID: "run-1"}}) {
		t.Fatalf("byID calls mismatch: got=%#v want=%#v", exp.byIDCalls, []byIDCall{{runID: "run-1"}})
	}
}

func TestExportByID_Error(t *testing.T) {
	exp := &fakeExporter{byIDErr: errors.New("boom")}
	handler := NewCLIHandler(exp)

	err := handler.ExportByID(context.Background(), &bytes.Buffer{}, "run-1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExportByID_NilExporter(t *testing.T) {
	handler := NewCLIHandler(nil)

	err := handler.ExportByID(context.Background(), &bytes.Buffer{}, "run-1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
