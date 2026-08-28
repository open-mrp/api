package db

import "testing"

func TestTrimDecimal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "no fraction is untouched", value: "1", want: "1"},
		// TrimRight over the whole string would eat these, turning 100 into 1.
		{name: "integer zeros survive", value: "1000", want: "1000"},
		{name: "integer zeros survive an all-zero fraction", value: "100.000", want: "100"},
		{name: "all-zero fraction", value: "1.000000000000000000000000000000", want: "1"},
		{name: "zero", value: "0.000", want: "0"},
		{name: "negative zero", value: "-0.000", want: "-0"},
		{name: "trailing zeros only", value: "10.500000", want: "10.5"},
		{name: "interior zero survives", value: "1.010", want: "1.01"},
		{name: "no trailing zeros", value: "10.55", want: "10.55"},
		{name: "negative", value: "-2.500", want: "-2.5"},
		{name: "empty", value: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := TrimDecimal(tt.value); got != tt.want {
				t.Fatalf("TrimDecimal(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
