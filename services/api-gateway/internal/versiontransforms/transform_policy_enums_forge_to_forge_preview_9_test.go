package versiontransforms

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

func TestPreview9_PolicyValuesDowngrade(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCustomer, `{"id":"acre_1","object":"customer","commission_policy":"exempt","freight_policy":"free",
		"freight_preferences":{"object":"customer_freight_preferences","status":"billed"},
		"type":{"id":"acgp_1","object":"account_group","commission_policy":"applied","freight_policy":"billed"}}`)
	assert.Equal(t, "commission_exempt", got["commission_policy"])
	assert.Equal(t, "free_freight", got["freight_policy"])
	assert.Equal(t, "billed_freight", got["freight_preferences"].(map[string]any)["status"])
	group := got["type"].(map[string]any)
	assert.Equal(t, "commission_applied", group["commission_policy"])
	assert.Equal(t, "billed_freight", group["freight_policy"])

	order := downgrade(t, constants.ObjectTypeSalesOrder, `{"id":"so_1","object":"sales_order","freight":{"object":"freight","policy":"free"}}`)
	assert.Equal(t, "free_freight", order["freight"].(map[string]any)["policy"])

	portal := downgrade(t, constants.ObjectTypeProductLine, `{"id":"pdln_1","object":"product_line","commission_policy":null,"freight_policy":"billed"}`)
	assert.Nil(t, portal["commission_policy"])
	assert.Equal(t, "billed_freight", portal["freight_policy"])
}

func TestPreview9_PolicyValuesUpgradeInWrites(t *testing.T) {
	t.Parallel()

	got := upgradeBody(constants.ObjectTypeAccountGroup, http.MethodPost, "/v1/sales/account-groups", map[string]any{
		"name": "Distributors", "commission_policy": "commission_exempt", "freight_policy": "free_freight",
	})
	assert.Equal(t, map[string]any{"name": "Distributors", "commission_policy": "exempt", "freight_policy": "free"}, got)

	unchanged := upgradeBody(constants.ObjectTypeCustomer, http.MethodPatch, "/v1/sales/customers/{id}", map[string]any{"commission_policy": "applied"})
	assert.Equal(t, "applied", unchanged["commission_policy"])

	export := upgradeBody(constants.ObjectTypeAsyncJob, http.MethodPost, customersExportRoute, map[string]any{
		"commission_status_codes": []any{"commission_applied"}, "freight_status_codes": []any{"billed_freight", "free_freight"},
	})
	assert.Equal(t, []any{"applied"}, export["commission_status_codes"])
	assert.Equal(t, []any{"billed", "free"}, export["freight_status_codes"])
}

func TestPreview9_PolicyValuesUpgradeInCustomerFilters(t *testing.T) {
	t.Parallel()

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeCustomer, customersRoute, url.Values{
		"commission_status_codes[]": {"commission_exempt"}, "freight_status_codes": {"free_freight"},
	})
	assert.Equal(t, url.Values{"commission_status_codes[]": {"exempt"}, "freight_status_codes": {"free"}}, q)
}
