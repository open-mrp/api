//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seller's internal operational data reaches only its own people: what a production run holds and who runs it, how fast an item is consumed, how a product line is planned, its team's notes and commission settings, when its users were last active, and which conversations are under legal hold. Customer and supplier portal actors read those fields as null while the record itself still loads; every internal role that may read the record sees them.

// sellerInternalFields mirrors the fields the gateway tags sensitive:"internal", by the object that carries them.
var sellerInternalFields = map[string][]string{
	"production_run":    {"responsible_user", "batch_count", "batch_summaries"},
	"item":              {"burn_rate", "notes"},
	"item_category":     {"notes"},
	"product_line":      {"default_lot", "fulfillment_policy", "notes", "commission_policy"},
	"customer":          {"note", "commission_policy"},
	"customer_defaults": {"fulfillment_policy"},
	"account_group":     {"commission_policy"},
	"account_user":      {"is_commission_eligible", "last_used_at"},
	"conversation":      {"legal_hold"},
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

// --- Notes and commission settings ---

func TestSellerInternal_CatalogNotesAndCommissionAreWithheldFromPortals(t *testing.T) {
	t.Parallel()
	line := createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-internal-notes-pdln"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	})
	// The API takes no product line notes; the seller's team writes them in the dashboard.
	_, err := authDB(t).Exec("UPDATE product_line SET notes = ? WHERE id = ?", "Renegotiate with the mill before the spring run", jsonField(line, "id"))
	require.NoError(t, err)
	category := createAndCleanup(t, itemCategoriesPath, map[string]any{
		"name":          uniqueName("e2e-internal-notes-itcg"),
		"type":          "product_category",
		"unit_group_id": SeedUnitGroupID,
	})
	categoryPath := itemCategoriesPath + "/" + jsonField(category, "id")
	status, body, err := apiClient.Patch(categoryPath, map[string]any{"notes": "Only the second shift knits these"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	product := createAndCleanup(t, productsPath, map[string]any{
		"sku":               uniqueName("e2e-internal-notes-pd"),
		"type":              "sale",
		"category_id":       SeedItemCategoryID,
		"product_line_id":   SeedProductLineID,
		"portal_visibility": "visible",
		"notes":             "Margin is thin on this one",
		"unit_price":        map[string]any{"value": "4.50", "numerator_unit_id": currencyUnitID, "denominator_unit_id": nonCurrencyUnitID},
	})
	linePath := productLinesPath + "/" + jsonField(line, "id")
	productPath := productsPath + "/" + jsonField(product, "id")
	itemOf := func(client *Client) map[string]any {
		item := jsonObject(parseJSON(mustGetAs(t, client, productPath, url.Values{"include": {"item"}})), "item")
		require.NotNil(t, item, "the product's item loads")
		return item
	}

	for who, client := range map[string]*Client{"admin": apiClient, "catalog reader": customRoleClient(t, "product_lines:read", "items:read")} {
		got := parseJSON(mustGetAs(t, client, linePath, nil))
		assert.Equal(t, "Renegotiate with the mill before the spring run", jsonField(got, "notes"), who)
		assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"), who)
		assert.Equal(t, "Margin is thin on this one", jsonField(itemOf(client), "notes"), who)
	}
	assert.Equal(t, "Only the second shift knits these", jsonField(parseJSON(mustGetAs(t, apiClient, categoryPath, nil)), "notes"))

	for who, portal := range portalClients(t) {
		got := parseJSON(mustGetAs(t, portal, linePath, nil))
		assertNilField(t, got, "notes")
		assertNilField(t, got, "commission_policy")
		assert.Equal(t, jsonField(line, "name"), jsonField(got, "name"), "%s still reads the product line", who)

		item := itemOf(portal)
		assertNilField(t, item, "notes")
		assert.NotEmpty(t, jsonField(item, "sku"), "%s still reads the item", who)

		got = parseJSON(mustGetAs(t, portal, categoryPath, nil))
		assertNilField(t, got, "notes")
		assert.Equal(t, jsonField(category, "name"), jsonField(got, "name"), "%s still reads the category", who)
	}
}

// A customer reads its own record, its groups, its child accounts and its sales rep without the seller's note, commission settings or the rep's activity. A supplier reaches no customer record at all.
func TestSellerInternal_CustomerPortalReadsItsRecordWithoutNotesCommissionOrActivity(t *testing.T) {
	t.Parallel()
	path := customersPath + "/" + SeedCustomerAccountID
	children := jsonListData(parseJSON(mustGetAs(t, apiClient, path, url.Values{"include": {"child_accounts"}})), "child_accounts")
	require.NotEmpty(t, children, "the seed customer has child accounts")
	childID := jsonField(children[0].(map[string]any), "id")
	childPath := customersPath + "/" + childID
	status, body, err := apiClient.Patch(childPath, map[string]any{"note": "Call the buyer, not the plant"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.Patch(childPath, map[string]any{"note": nil}, newIdempotencyKey()) })

	// The sales rep signs in, so the seller sees when it last used the account.
	mustGetAs(t, chatUserClient(t), customersPath+"/"+SeedCustomerAccountID, nil)

	include := url.Values{"include": {"type", "price_groups", "child_accounts", "defaults.sales_rep"}}
	var staff map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		staff = parseJSON(mustGetAs(t, apiClient, path, include))
		if jsonObject(jsonObject(staff, "defaults"), "sales_rep")["last_used_at"] == nil {
			return fmt.Errorf("the sales rep's last use is not recorded yet")
		}
		return nil
	})
	assert.NotEmpty(t, jsonField(staff, "commission_policy"), "the seller reads the customer's commission policy")
	assert.NotEmpty(t, jsonField(jsonObject(staff, "type"), "commission_policy"), "the seller reads the group's commission policy")
	rep := jsonObject(jsonObject(staff, "defaults"), "sales_rep")
	require.NotNil(t, rep, "the seed customer has a sales rep")
	_, isBool := rep["is_commission_eligible"].(bool)
	assert.True(t, isBool, "the seller reads whether its rep earns commission")
	assert.Equal(t, "Call the buyer, not the plant", jsonField(childAccount(t, staff, childID), "note"))

	own := parseJSON(mustGetAs(t, getCustomerPortalClient(), path, include))
	assert.Equal(t, SeedCustomerAccountID, jsonField(own, "id"))
	assertNilField(t, own, "note")
	assertNilField(t, own, "commission_policy")
	assertNilField(t, jsonObject(own, "type"), "commission_policy")
	for _, group := range jsonListData(own, "price_groups") {
		assertNilField(t, group.(map[string]any), "commission_policy")
	}
	child := childAccount(t, own, childID)
	assertNilField(t, child, "note")
	assertNilField(t, child, "commission_policy")
	portalRep := jsonObject(jsonObject(own, "defaults"), "sales_rep")
	require.NotNil(t, portalRep, "the customer still sees who its rep is")
	assert.Equal(t, jsonField(rep, "id"), jsonField(portalRep, "id"))
	assertNilField(t, portalRep, "is_commission_eligible")
	assertNilField(t, portalRep, "last_used_at")

	requireStatusAs(t, http.StatusNotFound, "supplier portal", getSupplierPortalClient(t), path, nil)
}

func childAccount(t *testing.T, customer map[string]any, id string) map[string]any {
	t.Helper()
	for _, child := range jsonListData(customer, "child_accounts") {
		if c := child.(map[string]any); jsonField(c, "id") == id {
			return c
		}
	}
	t.Fatalf("child account %s is not listed on %s", id, jsonField(customer, "id"))
	return nil
}

// --- Legal hold ---

// openSupportCase has a portal contact the seller's support and returns the case it opens.
func openSupportCase(t *testing.T, portal *Client) string {
	t.Helper()
	resp, err := portal.PostFull(supportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, resp.StatusCode, resp.Body)
	id := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, id)
	return id
}

// Not parallel: the support route these cases open through is account-wide.
func TestSellerInternal_LegalHoldIsWithheldFromPortals(t *testing.T) {
	seedSupportRoute(t)
	staff := chatUserClient(t)

	for who, portal := range portalClients(t) {
		convID := openSupportCase(t, portal)

		got := parseJSON(mustGetAs(t, staff, conversationsPath+"/"+convID, nil))
		assert.Equal(t, "released", jsonField(got, "legal_hold"), "the seller reads the case's legal hold")

		got = parseJSON(mustGetAs(t, portal, conversationsPath+"/"+convID, nil))
		assert.Equal(t, convID, jsonField(got, "id"), "%s reads its own case", who)
		assertNilField(t, got, "legal_hold")

		list := parseJSON(mustGetAs(t, portal, conversationsPath, url.Values{"limit": {"50"}}))
		for _, row := range jsonArray(list, "data") {
			assertNilField(t, row.(map[string]any), "legal_hold")
		}
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
