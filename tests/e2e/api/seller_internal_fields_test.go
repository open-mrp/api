//go:build e2e

package api_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seller's internal operational data reaches only its own people: what a production run holds and who runs it, how fast an item is consumed, how a product line is planned. Customer and supplier portal actors read those fields as null while the record itself still loads; every internal role that may read the record sees them.

// sellerInternalFields mirrors the fields the gateway tags sensitive:"internal", by the object that carries them.
var sellerInternalFields = map[string][]string{
	"production_run":    {"responsible_user", "batch_count", "batch_summaries"},
	"item":              {"burn_rate"},
	"product_line":      {"default_lot", "fulfillment_policy"},
	"customer_defaults": {"fulfillment_policy"},
}

// internalLeaks lists the JSON paths in v where a seller-internal field holds a value.
func internalLeaks(v any, path string) []string {
	var out []string
	switch x := v.(type) {
	case map[string]any:
		object, _ := x["object"].(string)
		for _, field := range sellerInternalFields[object] {
			if x[field] != nil {
				out = append(out, path+"."+field)
			}
		}
		for k, child := range x {
			out = append(out, internalLeaks(child, path+"."+k)...)
		}
	case []any:
		for _, elem := range x {
			out = append(out, internalLeaks(elem, path+"[]")...)
		}
	}
	return out
}

func requireNoInternal(t *testing.T, who, what string, body []byte) {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal(body, &v), "%s %s: %s", who, what, string(body))
	leaks := internalLeaks(v, "")
	sort.Strings(leaks)
	assert.Empty(t, leaks, "%s received the seller's internal data from %s", who, what)
}

func portalClients(t *testing.T) map[string]*Client {
	t.Helper()
	return map[string]*Client{"customer portal": getCustomerPortalClient(), "supplier portal": getSupplierPortalClient(t)}
}

// --- Production runs ---

// A buyer following its order to the run producing it learns the run's number and whether it is still open, and nothing of the other work the run carries; the seller's staff read all of it.
func TestSellerInternal_PortalBuyerSeesItsOrdersProductionRunAsNumberAndStatus(t *testing.T) {
	t.Parallel()
	include := url.Values{"include": {"related.production_run"}}

	order := parseJSON(mustGetAs(t, getCustomerPortalClient(), salesOrdersPath+"/"+SeedSalesOrderID, include))
	run := jsonObject(jsonObject(order, "related"), "production_run")
	require.NotNil(t, run, "the buyer's order links to its production run: %v", order)
	runID := jsonField(run, "id")
	require.NotEmpty(t, runID)
	assert.Equal(t, "record", jsonField(run, "object"))
	assert.Equal(t, "production_run", jsonField(run, "type"))
	assert.NotEmpty(t, jsonField(run, "number"))
	assert.Contains(t, []string{"open", "closed"}, jsonField(run, "status"))
	for _, field := range sellerInternalFields["production_run"] {
		_, present := run[field]
		assert.False(t, present, "the buyer's view of the run carries no %s", field)
	}

	staff := customRoleClient(t, "sales_orders:read", "production_runs:read")
	assert.Equal(t, run, jsonObject(jsonObject(parseJSON(mustGetAs(t, staff, salesOrdersPath+"/"+SeedSalesOrderID, include)), "related"), "production_run"),
		"the seller's staff see the same reference on the order")

	full := parseJSON(mustGetAs(t, staff, productionRunsPath+"/"+runID, url.Values{"include": {"responsible_user"}}))
	assert.Equal(t, jsonField(run, "number"), jsonField(full, "number"))
	require.NotNil(t, full["batch_count"], "the seller's staff read the run's batch count")
	require.NotNil(t, jsonObject(full, "batch_summaries"), "the seller's staff read the run's batch summaries")
	assert.NotEmpty(t, jsonField(jsonObject(full, "responsible_user"), "id"), "the seller's staff read who is responsible for the run")

	for who, portal := range portalClients(t) {
		status, body, err := portal.GetListRaw(productionRunsPath+"/"+runID, nil)
		require.NoError(t, err)
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, status, "%s may not read the run itself: %s", who, string(body))
	}
}

// --- Items ---

func TestSellerInternal_ItemBurnRateIsWithheldFromPortals(t *testing.T) {
	t.Parallel()
	path := productsPath + "/" + SeedProductID
	include := url.Values{"include": {"item", "item.burn_rate", "item.unit_value"}}

	itemOf := func(client *Client) map[string]any {
		item := jsonObject(parseJSON(mustGetAs(t, client, path, include)), "item")
		require.NotNil(t, item, "the product's item loads")
		return item
	}

	for who, client := range map[string]*Client{"admin": apiClient, "items reader": customRoleClient(t, "items:read")} {
		assert.NotEmpty(t, rateValue(t, itemOf(client), "burn_rate"), "%s reads the item's burn rate", who)
	}
	for who, portal := range portalClients(t) {
		item := itemOf(portal)
		assertNilField(t, item, "burn_rate")
		assert.NotEmpty(t, jsonField(item, "sku"), "%s still reads the item", who)
	}
}

// --- Product lines ---

func TestSellerInternal_ProductLinePlanningIsWithheldFromPortals(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, productLinesPath, map[string]any{
		"name":               uniqueName("e2e-internal-pdln"),
		"unit_group_id":      SeedUnitGroupID,
		"commission_policy":  "commission_applied",
		"freight_policy":     "billed_freight",
		"fulfillment_policy": "make_to_order",
		"default_lot":        map[string]any{"value": "60", "unit_id": SeedPairUnitID},
	})
	path := productLinesPath + "/" + jsonField(created, "id")
	include := url.Values{"include": {"default_lot"}}

	for who, client := range map[string]*Client{"admin": apiClient, "product line reader": customRoleClient(t, "product_lines:read")} {
		line := parseJSON(mustGetAs(t, client, path, include))
		assert.Equal(t, "make_to_order", jsonField(line, "fulfillment_policy"), who)
		assert.Equal(t, "60", jsonField(jsonObject(line, "default_lot"), "value"), who)
	}
	for who, portal := range portalClients(t) {
		line := parseJSON(mustGetAs(t, portal, path, include))
		assertNilField(t, line, "fulfillment_policy")
		assertNilField(t, line, "default_lot")
		assert.Equal(t, jsonField(created, "name"), jsonField(line, "name"), "%s still reads the product line", who)
	}
}

// --- Sweeps ---

func TestSellerInternal_CustomerPortalSweepFindsNoInternalData(t *testing.T) {
	t.Parallel()
	portalSweep(t, "customer portal", getCustomerPortalClient(), requireNoInternal)
}

func TestSellerInternal_SupplierPortalSweepFindsNoInternalData(t *testing.T) {
	t.Parallel()
	portalSweep(t, "supplier portal", getSupplierPortalClient(t), requireNoInternal)
}
