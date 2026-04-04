package entity

import "time"

// PhaseMetrics is the raw aggregated record fetched from the data layer.
// Phase is later parsed into Language and Benchmark.
type PhaseMetrics struct {
	RunID      string
	MeasuredAt time.Time
	Phase      string
	Metrics    map[string]int64
}
