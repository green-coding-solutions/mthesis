package entity

import "time"

// Measurement is the normalized export-ready record.
// Metrics is a sparse map where keys become CSV columns.
type Measurement struct {
	RunID      string
	MeasuredAt time.Time
	Language   string
	Benchmark  string
	Metrics    map[string]int64
}
