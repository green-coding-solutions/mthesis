package export

import (
	"fmt"
	"strings"
	"time"

	"mthesis/kwa/internal/entity"
)

const (
	// TimestampLayout is the full timestamp format accepted by CLI/TUI inputs.
	TimestampLayout = "2006-01-02 15:04:05"
	dateOnlyLayout  = "2006-01-02"
)

// ParseTimeRange parses optional start/end timestamp strings and validates them
// as an all-or-nothing inclusive interval.
func ParseTimeRange(fromInput, toInput string) (entity.TimeRangeFilter, error) {
	from, err := ParseTimestampInput(fromInput)
	if err != nil {
		return entity.TimeRangeFilter{}, fmt.Errorf("invalid from timestamp: %w", err)
	}

	to, err := ParseTimestampInput(toInput)
	if err != nil {
		return entity.TimeRangeFilter{}, fmt.Errorf("invalid to timestamp: %w", err)
	}

	filter := entity.TimeRangeFilter{From: from, To: to}
	if err := filter.Validate(); err != nil {
		return entity.TimeRangeFilter{}, err
	}

	return filter.Clone(), nil
}

// ParseTimestampInput parses one optional timestamp input using local timezone.
// Accepted formats:
//   - YYYY-MM-DD HH:MM:SS
//   - YYYY-MM-DD (defaults to 00:00:00)
func ParseTimestampInput(input string) (*time.Time, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	if parsed, err := time.ParseInLocation(TimestampLayout, trimmed, time.Local); err == nil {
		return &parsed, nil
	}

	if parsedDate, err := time.ParseInLocation(dateOnlyLayout, trimmed, time.Local); err == nil {
		normalized := time.Date(
			parsedDate.Year(),
			parsedDate.Month(),
			parsedDate.Day(),
			0,
			0,
			0,
			0,
			parsedDate.Location(),
		)
		return &normalized, nil
	}

	return nil, fmt.Errorf("use YYYY-MM-DD or YYYY-MM-DD HH:MM:SS")
}
