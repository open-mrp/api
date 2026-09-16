package pricing

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// Units as stored in production: each is the base, a pair is two, a carton of twelve pairs is 24.
var (
	each       = [2]decimal.Decimal{d("1"), d("1")}
	pair       = [2]decimal.Decimal{d("2"), d("1")}
	carton     = [2]decimal.Decimal{d("24"), d("1")}
	ct10pr     = [2]decimal.Decimal{d("20"), d("1")}
	oneSeventh = [2]decimal.Decimal{d("1"), d("7")}
)

func between(qty, price [2]decimal.Decimal) UnitConversion {
	return Between(qty[0], qty[1], price[0], price[1])
}

func TestExtendedPrice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		qty   string
		price string
		conv  UnitConversion
		want  string
	}{
		{"cartons priced per pair", "3", "22.5", between(carton, pair), "810"},
		{"pairs priced per carton", "1200", "270", between(pair, carton), "27000"},
		{"eaches priced per pair", "7", "1", between(each, pair), "3.5"},
		{"same unit", "300", "4.25", Identity, "1275"},
		{"cartons priced per ten-pair carton", "1", "10", between(carton, ct10pr), "12"},
		// Non-terminating divisions are covered against the dashboard in TestMatchesTheDashboard.
		{"a ratio with a denominator", "7", "2", between(oneSeventh, each), "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ExtendedPrice(d(tt.qty), d(tt.price), tt.conv).String())
		})
	}
}

func TestLineTotalRoundsHalfAwayFromZero(t *testing.T) {
	t.Parallel()
	require.Equal(t, "0.01", LineTotal(d("1"), d("0.005"), Identity).StringFixed(2))
	require.Equal(t, "-0.01", LineTotal(d("1"), d("-0.005"), Identity).StringFixed(2))
}

func TestParseUnitConversion(t *testing.T) {
	t.Parallel()

	conv, err := ParseUnitConversion("24.000000000000000000000000000000", "1.000000000000000000000000000000", "2", "1")
	require.NoError(t, err)
	require.Equal(t, "810", ExtendedPrice(d("3"), d("22.5"), conv).String())

	for _, bad := range [][4]string{{"1", "0", "1", "1"}, {"1", "1", "0", "1"}, {"", "", "", ""}, {"1", "x", "1", "1"}} {
		_, err := ParseUnitConversion(bad[0], bad[1], bad[2], bad[3])
		require.Error(t, err, "%v", bad)
	}
}

// ExtendedPrice must agree with the dashboard's QuantityUtils.multiplyRate to the last digit.
// testdata/dashboard_cases.json holds amounts computed by the dashboard's own code (see
// testdata/gen_dashboard_cases.ts), including every generated case where its 40-digit arithmetic
// rounds to a different cent than exact arithmetic would.
func TestMatchesTheDashboard(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/dashboard_cases.json")
	require.NoError(t, err)
	var cases []struct {
		Same          bool
		Qty, Price    string
		Qr, Pr        [2]string
		Amount, Cents string
	}
	require.NoError(t, json.Unmarshal(raw, &cases))
	require.NotEmpty(t, cases)

	for _, c := range cases {
		conv := Identity
		if !c.Same {
			conv, err = ParseUnitConversion(c.Qr[0], c.Qr[1], c.Pr[0], c.Pr[1])
			require.NoError(t, err)
		}
		require.True(t, ExtendedPrice(d(c.Qty), d(c.Price), conv).Equal(d(c.Amount)),
			"%+v: got %s", c, ExtendedPrice(d(c.Qty), d(c.Price), conv))
		require.Equal(t, c.Cents, LineTotal(d(c.Qty), d(c.Price), conv).StringFixed(2), "%+v", c)
	}
}
