package versiontransforms

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/constants"
)

func TestPreview9_OrderDiscountReadsTheFractionFields(t *testing.T) {
	t.Parallel()

	pct := downgrade(t, constants.ObjectTypeOrderDiscount, `{"id":"ords_1","object":"order_discount","status":"active","name":"Ten","code":"SAVE10",
		"discount":{"type":"percent_off","percent_off":"10","amount_off":null},"times_redeemed":3}`)
	assert.Equal(t, "0.1", pct["percentage"])
	assert.Equal(t, "0", pct["amount"])
	assert.Equal(t, "percentage", pct["discount_type"])
	assert.EqualValues(t, 3, pct["order_count"])
	assert.NotContains(t, pct, "discount")
	assert.NotContains(t, pct, "times_redeemed")

	amt := downgrade(t, constants.ObjectTypeOrderDiscount, `{"object":"list","data":[{"id":"ords_2","object":"order_discount",
		"discount":{"type":"amount_off","percent_off":null,"amount_off":{"amount":"25.00","currency":"usd"}},"times_redeemed":0}]}`)
	row := amt["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "amount", row["discount_type"])
	assert.Equal(t, "25.00", row["amount"])
	assert.Equal(t, "0", row["percentage"])
}

func TestPreview9_OrderDiscountWritesBecomeADiscount(t *testing.T) {
	t.Parallel()

	got := upgradeBody(constants.ObjectTypeOrderDiscount, http.MethodPost, orderDiscountsRoute, map[string]any{
		"name": "Ten", "code": "SAVE10", "discount_type": "percentage", "percentage": "0.05",
	})
	assert.Equal(t, map[string]any{"type": "percent_off", "percent_off": "5"}, got["discount"])
	assert.NotContains(t, got, "percentage")
	assert.NotContains(t, got, "discount_type")

	got = upgradeBody(constants.ObjectTypeOrderDiscount, http.MethodPatch, orderDiscountRoute, map[string]any{"amount": "12.5"})
	assert.Equal(t, map[string]any{"type": "amount_off", "amount_off": map[string]any{"amount": "12.5", "currency": "usd"}}, got["discount"])

	got = upgradeBody(constants.ObjectTypeOrderDiscount, http.MethodPatch, orderDiscountRoute, map[string]any{"name": "Renamed"})
	assert.NotContains(t, got, "discount", "a write that touches no amount leaves the discount alone")
}
