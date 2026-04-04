package export

import (
	"testing"
	"time"

	"mthesis/kwa/internal/entity"
)

func TestParseTimestampInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		wantNil   bool
		wantValue string
		wantErr   bool
	}{
		{name: "empty", input: " ", wantNil: true},
		{name: "full timestamp", input: "2026-04-04 13:14:15", wantValue: "2026-04-04 13:14:15"},
		{name: "date only defaults to midnight", input: "2026-04-04", wantValue: "2026-04-04 00:00:00"},
		{name: "invalid value", input: "2026/04/04", wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseTimestampInput(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTimestampInput() error = %v", err)
			}

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil timestamp, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("expected non-nil timestamp")
			}
			if got.Format(TimestampLayout) != tc.wantValue {
				t.Fatalf("timestamp = %q, want %q", got.Format(TimestampLayout), tc.wantValue)
			}
		})
	}
}

func TestParseTimeRange(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		from      string
		to        string
		wantErr   bool
		wantFrom  string
		wantTo    string
		expectNil bool
	}{
		{name: "both empty", from: "", to: "", expectNil: true},
		{
			name:     "full range",
			from:     "2026-04-01 10:00:00",
			to:       "2026-04-02 11:30:00",
			wantFrom: "2026-04-01 10:00:00",
			wantTo:   "2026-04-02 11:30:00",
		},
		{
			name:     "date only uses midnight",
			from:     "2026-04-01",
			to:       "2026-04-02",
			wantFrom: "2026-04-01 00:00:00",
			wantTo:   "2026-04-02 00:00:00",
		},
		{name: "missing to", from: "2026-04-01", to: "", wantErr: true},
		{name: "from after to", from: "2026-04-03", to: "2026-04-02", wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			filter, err := ParseTimeRange(tc.from, tc.to)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTimeRange() error = %v", err)
			}

			if tc.expectNil {
				if filter.From != nil || filter.To != nil {
					t.Fatalf("expected nil range, got from=%v to=%v", filter.From, filter.To)
				}
				return
			}

			if filter.From == nil || filter.To == nil {
				t.Fatalf("expected both timestamps, got from=%v to=%v", filter.From, filter.To)
			}
			if filter.From.Format(TimestampLayout) != tc.wantFrom {
				t.Fatalf("from = %q, want %q", filter.From.Format(TimestampLayout), tc.wantFrom)
			}
			if filter.To.Format(TimestampLayout) != tc.wantTo {
				t.Fatalf("to = %q, want %q", filter.To.Format(TimestampLayout), tc.wantTo)
			}
		})
	}
}

func TestTimeRangeFilterValidate(t *testing.T) {
	t.Parallel()

	now := time.Now()
	later := now.Add(1 * time.Hour)

	if err := (entity.TimeRangeFilter{}).Validate(); err != nil {
		t.Fatalf("unexpected error for nil range: %v", err)
	}
	if err := (entity.TimeRangeFilter{From: &now, To: &later}).Validate(); err != nil {
		t.Fatalf("unexpected error for valid range: %v", err)
	}
	if err := (entity.TimeRangeFilter{From: &later, To: &now}).Validate(); err == nil {
		t.Fatalf("expected from>to error")
	}
	if err := (entity.TimeRangeFilter{From: &now}).Validate(); err == nil {
		t.Fatalf("expected partial bounds error")
	}
}
