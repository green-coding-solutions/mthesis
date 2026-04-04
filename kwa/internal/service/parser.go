package service

import (
	"fmt"
	"strings"

	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/entity"
)

// ParserService converts raw phase identifiers into canonical measurement dimensions.
type ParserService struct{}

// NewParserService creates a parser that maps raw phase tokens into canonical dimensions.
func NewParserService() *ParserService {
	return &ParserService{}
}

// ParseMeasurementFromPhase parses a phase value like "005_Go-Binary-Trees"
// and writes normalized values into a Measurement entity.
func (s *ParserService) ParseMeasurementFromPhase(pm entity.PhaseMetrics) (entity.Measurement, error) {
	phase := pm.Phase
	parts := strings.SplitN(strings.TrimSpace(phase), "_", 2)
	if len(parts) != 2 || parts[1] == "" {
		return entity.Measurement{}, fmt.Errorf("invalid phase format: %q", phase)
	}

	langAndBenchmark := strings.TrimSpace(parts[1])
	idx := strings.Index(langAndBenchmark, "-")
	if idx <= 0 || idx >= len(langAndBenchmark)-1 {
		return entity.Measurement{}, fmt.Errorf("invalid language/benchmark segment: %q", langAndBenchmark)
	}

	rawLanguage := langAndBenchmark[:idx]
	rawBenchmark := langAndBenchmark[idx+1:]

	language, err := constant.ParseProgrammingLanguage(rawLanguage)
	if err != nil {
		return entity.Measurement{}, err
	}

	benchmark, err := constant.ParseBenchmark(rawBenchmark)
	if err != nil {
		return entity.Measurement{}, err
	}

	return entity.Measurement{
		RunID:      pm.RunID,
		MeasuredAt: pm.MeasuredAt,
		Language:   string(language),
		Benchmark:  string(benchmark),
		Metrics:    cloneMetrics(pm.Metrics),
	}, nil
}

// cloneMetrics prevents downstream mutation of the original map owned by data DTOs.
func cloneMetrics(metrics map[string]int64) map[string]int64 {
	if len(metrics) == 0 {
		return make(map[string]int64)
	}

	cloned := make(map[string]int64, len(metrics))
	for k, v := range metrics {
		cloned[k] = v
	}

	return cloned
}
