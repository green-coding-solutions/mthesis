package service

import "mthesis/exporter/internal/entity"

type ExporterService struct {
	parserService *ParserService
}

func NewExporterService(parserService *ParserService) *ExporterService {
	if parserService == nil {
		parserService = NewParserService()
	}

	return &ExporterService{
		parserService: parserService,
	}
}

func (s *ExporterService) ParseMeasurementFromPhase(phase, value string) (entity.Measurement, error) {
	return s.parserService.ParseMeasurementFromPhase(phase, value)
}
