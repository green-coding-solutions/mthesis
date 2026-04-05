package measure

// Request contains normalized inputs for one measure workflow execution.
// It captures the selected language/benchmark sets, run iterations, and the
// final CSV output path used by the follow-up export.
type Request struct {
	Languages  []string
	Benchmarks []string
	Iterations int
	OutPath    string
}
