package entity

import (
	"fmt"
	"time"
)

// TimeRangeFilter represents an optional inclusive time window used by export queries.
type TimeRangeFilter struct {
	From *time.Time
	To   *time.Time
}

// Validate enforces all-or-nothing bounds and ascending inclusive range semantics.
func (f TimeRangeFilter) Validate() error {
	switch {
	case f.From == nil && f.To == nil:
		return nil
	case f.From == nil || f.To == nil:
		return fmt.Errorf("from and to must both be provided or both be empty")
	case f.From.After(*f.To):
		return fmt.Errorf("from must be less than or equal to to")
	default:
		return nil
	}
}

// Clone returns a deep-copied filter so callers can safely retain immutable inputs.
func (f TimeRangeFilter) Clone() TimeRangeFilter {
	return TimeRangeFilter{
		From: cloneOptionalTime(f.From),
		To:   cloneOptionalTime(f.To),
	}
}

// cloneOptionalTime creates a copy of an optional time pointer.
func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
