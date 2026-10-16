package repository

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

// A unit group member's discount came off the base price before converting it to the member's unit; the derived order
// unit's discount comes off after, and must leave the same price.
func TestDerivedDiscount_KeepsTodaysPrice(t *testing.T) {
	t.Parallel()
	price := decimal.NewFromInt(10) // per pair
	pair, dozen := decimal.NewFromInt(2), decimal.NewFromInt(12)
	k := dozen.Div(pair)

	cases := []struct {
		name, pct, fixed string
		wantType         constants.DiscountType
	}{
		{"fixed only", "0", "1", constants.DiscountTypeAmountOff},
		{"percent only", "0.1", "0", constants.DiscountTypePercentOff},
		{"both", "0.1", "1", constants.DiscountTypeAmountOff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := derivedDiscount(tc.pct, tc.fixed, price, dozen, pair)
			require.NotNil(t, d)
			assert.Equal(t, tc.wantType, d.Type)

			pct := decimal.RequireFromString(tc.pct)
			fixed := decimal.RequireFromString(tc.fixed)
			old := price.Sub(fixed).Mul(decimal.NewFromInt(1).Sub(pct)).Mul(k)
			ou := &domain.ItemOrderUnit{Discount: d}
			assert.True(t, ou.ApplyDiscount(price.Mul(k)).Equal(old), "derived %s gives %s, old price %s", tc.name, ou.ApplyDiscount(price.Mul(k)), old)
		})
	}

	assert.Nil(t, derivedDiscount("0", "0", price, dozen, pair), "no discount stays none")
}
