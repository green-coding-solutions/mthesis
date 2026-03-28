package service

import (
	"fmt"
	"strings"

	"mthesis/exporter/internal/constant"
	"mthesis/exporter/internal/entity"
)

type ParserService struct {
}

func NewParserService() *ParserService {
	return &ParserService{}
}

// ParseMeasurementFromPhase parses a phase value like "005_Go-Binary-Trees"
// and writes normalized values into a Measurement entity.
func (s *ParserService) ParseMeasurementFromPhase(phase, value string) (entity.Measurement, error) {
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
		Language:  string(language),
		Benchmark: string(benchmark),
		Value:     value,
	}, nil
}
