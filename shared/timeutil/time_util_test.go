package timeutil

import (
	"testing"
	"time"
)

func TestTimestampToTimePtr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantEqual time.Time
	}{
		{
			name:      "valid timestamp",
			input:     "2024-01-15T10:30:00Z",
			wantNil:   false,
			wantEqual: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:      "epoch",
			input:     "1970-01-01T00:00:00Z",
			wantNil:   false,
			wantEqual: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "invalid format",
			input:   "2024-01-15 10:30:00",
			wantNil: true,
		},
		{
			name:    "date only",
			input:   "2024-01-15",
			wantNil: true,
		},
		{
			name:    "missing Z suffix",
			input:   "2024-01-15T10:30:00",
			wantNil: true,
		},
		{
			// The shape agent-service emits for every pgtype.Timestamptz field.
			name:      "millisecond fraction",
			input:     "2024-01-15T10:30:00.000Z",
			wantEqual: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:      "microsecond fraction",
			input:     "2024-01-15T10:30:00.123456Z",
			wantEqual: time.Date(2024, 1, 15, 10, 30, 0, 123456000, time.UTC),
		},
		{
			// The layout's Z is a literal, so RFC 3339 offset forms are rejected rather than converted.
			name:    "rfc 3339 zero offset",
			input:   "2024-01-15T10:30:00+00:00",
			wantNil: true,
		},
		{
			name:    "rfc 3339 negative offset",
			input:   "2024-01-15T10:30:00-05:00",
			wantNil: true,
		},
		{
			name:    "lowercase zone designator",
			input:   "2024-01-15T10:30:00z",
			wantNil: true,
		},
		{
			name:    "trailing whitespace",
			input:   "2024-01-15T10:30:00Z ",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimestampToTimePtr(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("TimestampToTimePtr(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("TimestampToTimePtr(%q) = nil, want %v", tt.input, tt.wantEqual)
			}
			if !got.Equal(tt.wantEqual) {
				t.Errorf("TimestampToTimePtr(%q) = %v, want %v", tt.input, *got, tt.wantEqual)
			}
		})
	}
}

func TestTimestampToTime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantZero  bool
		wantEqual time.Time
	}{
		{
			name:      "valid timestamp",
			input:     "2024-01-15T10:30:00Z",
			wantEqual: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:      "epoch",
			input:     "1970-01-01T00:00:00Z",
			wantEqual: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "empty string",
			input:    "",
			wantZero: true,
		},
		{
			name:     "invalid format",
			input:    "2024-01-15 10:30:00",
			wantZero: true,
		},
		{
			name:     "date only",
			input:    "2024-01-15",
			wantZero: true,
		},
		{
			name:     "missing Z suffix",
			input:    "2024-01-15T10:30:00",
			wantZero: true,
		},
		{
			// The shape agent-service emits for every pgtype.Timestamptz field.
			name:      "millisecond fraction",
			input:     "2024-01-15T10:30:00.000Z",
			wantEqual: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:      "microsecond fraction",
			input:     "2024-01-15T10:30:00.123456Z",
			wantEqual: time.Date(2024, 1, 15, 10, 30, 0, 123456000, time.UTC),
		},
		{
			// The layout's Z is a literal, so RFC 3339 offset forms are rejected rather than converted. Callers get the zero time with no error, which reaches the API as 0001-01-01T00:00:00Z.
			name:     "rfc 3339 zero offset",
			input:    "2024-01-15T10:30:00+00:00",
			wantZero: true,
		},
		{
			name:     "rfc 3339 negative offset",
			input:    "2024-01-15T10:30:00-05:00",
			wantZero: true,
		},
		{
			name:     "lowercase zone designator",
			input:    "2024-01-15T10:30:00z",
			wantZero: true,
		},
		{
			name:     "trailing whitespace",
			input:    "2024-01-15T10:30:00Z ",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimestampToTime(tt.input)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("TimestampToTime(%q) = %v, want zero time", tt.input, got)
				}
				return
			}
			if !got.Equal(tt.wantEqual) {
				t.Errorf("TimestampToTime(%q) = %v, want %v", tt.input, got, tt.wantEqual)
			}
		})
	}
}
