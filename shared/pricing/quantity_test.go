package pricing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ratio(r [2]string) UnitRatio {
	return UnitRatio{Numerator: d(r[0]), Denominator: d(r[1])}
}

func TestConvertQuantity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		qty      string
		from, to [2]string
		want     string
	}{
		{name: "same unit is exact", qty: "7.125", from: [2]string{"2", "1"}, to: [2]string{"2", "1"}, want: "7.125"},
		{name: "equal ratios stored differently are exact", qty: "3", from: [2]string{"4", "2"}, to: [2]string{"2", "1"}, want: "3"},
		{name: "pairs to each", qty: "5", from: [2]string{"2", "1"}, to: [2]string{"1", "1"}, want: "10"},
		{name: "each to cartons of twelve pairs", qty: "48", from: [2]string{"1", "1"}, to: [2]string{"24", "1"}, want: "2"},
		{name: "grams to pounds", qty: "453.59237", from: [2]string{"1", "1000"}, to: [2]string{"45359237", "100000000"}, want: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ConvertQuantity(d(tt.qty), ratio(tt.from), ratio(tt.to))
			require.True(t, got.Equal(d(tt.want)), "got %s, want %s", got, tt.want)
		})
	}
}

func TestParseUnitRatioRejectsZero(t *testing.T) {
	t.Parallel()

	_, err := ParseUnitRatio("0", "1")
	require.Error(t, err)
	_, err = ParseUnitRatio("1", "")
	require.Error(t, err)

	r, err := ParseUnitRatio("24", "1")
	require.NoError(t, err)
	require.True(t, r.Numerator.Equal(d("24")))
}
