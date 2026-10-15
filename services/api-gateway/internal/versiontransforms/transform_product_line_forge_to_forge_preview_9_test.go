package versiontransforms

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/constants"
)

func TestPreview9_ProductLineSupplyBecomesFulfillmentPolicy(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeProductLine, `{"id":"pdln_1","object":"product_line","status":"active","supply":"make","commission_policy":"exempt","freight_policy":"billed"}`)
	assert.Equal(t, "make_to_order", got["fulfillment_policy"])
	assert.NotContains(t, got, "supply")
	assert.Contains(t, got, "default_lot")
	assert.Nil(t, got["default_lot"])
	assert.Contains(t, got, "notes")
	assert.Nil(t, got["notes"])
	assert.Equal(t, "commission_exempt", got["commission_policy"])
	assert.Equal(t, "billed_freight", got["freight_policy"])

	buy := downgrade(t, constants.ObjectTypeProduct, `{"id":"prod_1","object":"product","product_line":{"id":"pdln_2","object":"product_line","supply":"buy"}}`)
	line := buy["product_line"].(map[string]any)
	assert.Nil(t, line["fulfillment_policy"])

	stock := downgrade(t, constants.ObjectTypeProductLine, `{"id":"pdln_3","object":"product_line","supply":null}`)
	assert.Nil(t, stock["fulfillment_policy"])
}

func TestPreview9_ProductLineWrites(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeProductLine, http.MethodPost, productLinesRoute, map[string]any{
		"name": "Socks", "unit_group_id": "ug_1", "commission_policy": "commission_applied", "freight_policy": "free_freight",
		"fulfillment_policy": "make_to_stock", "default_lot": map[string]any{"value": "60", "unit_id": "unit_1"},
	})
	assert.Equal(t, map[string]any{
		"name": "Socks", "unit_group_id": "ug_1", "commission_policy": "applied", "freight_policy": "free", "supply": "stock",
	}, created)

	cleared := upgradeBody(constants.ObjectTypeProductLine, http.MethodPatch, productLineRoute, map[string]any{"fulfillment_policy": nil, "default_lot": nil, "notes": "x"})
	assert.Equal(t, map[string]any{"supply": nil}, cleared)

	bulk := upgradeBody(constants.ObjectTypeAsyncJob, http.MethodPost, productLinesBulkUpsertRoute, map[string]any{"product_lines": []any{
		map[string]any{"name": "Socks", "commission_policy": "commission_exempt", "freight_policy": "billed_freight"},
	}})
	assert.Equal(t, map[string]any{"name": "Socks", "commission_policy": "exempt", "freight_policy": "billed"}, bulk["product_lines"].([]any)[0])
}

func TestPreview9_ItemLotDefaultNamesNoProductLine(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeItemLotDefault, `{"object":"item_lot_default","item":{"id":"itm_1"},"quantity":60,"source":"item_category","item_category":{"id":"ic_1"}}`)
	assert.NotContains(t, got, "item_category")
	assert.Contains(t, got, "product_line")
	assert.Nil(t, got["product_line"])
}
