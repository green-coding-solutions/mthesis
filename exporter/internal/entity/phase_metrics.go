package entity

// PhaseMetrics is the raw aggregated record fetched from the data layer.
// Phase is later parsed into Language and Benchmark.
type PhaseMetrics struct {
	RunID   string
	Phase   string
	Metrics map[string]int64
}
