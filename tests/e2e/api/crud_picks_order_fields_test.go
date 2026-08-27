//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A pick carries a copy of the order fields the floor works from — the customer's PO number, the
// order note, who raised the order, and how it ships — so a picking screen renders without a
// second fetch of the order. These pin that denormalization: each value must match the order it
// was copied from, the expandable ones must stay absent until asked for, and a pick whose order
// left a field unset must report null rather than an empty stub.

// Issues an order carrying the given extra create fields and returns both the order and its pick's id.
func issuedOrderAndPick(t *testing.T, customerID string, extra map[string]any) (map[string]any, string) {
	t.Helper()

	order := issueOrderForCustomer(t, customerID, extra)
	pickID := jsonField(jsonObject(jsonObject(order, "related"), "pick"), "id")
	if pickID == "" {
		status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+jsonField(order, "id"),
			url.Values{"include": {"related.pick"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		pickID = jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "pick"), "id")
	}
	require.NotEmpty(t, pickID, "issuing an order creates its pick")
	return order, pickID
}

func retrievePick(t *testing.T, pickID string, includes ...string) map[string]any {
	t.Helper()

	params := url.Values{}
	for _, inc := range includes {
		params.Add("include", inc)
	}
	status, body, err := apiClient.GetListRaw(picksPath+"/"+pickID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// Returns the list row for a customer's single pick, so a field can be checked on the list
// projection as well as on detail — they are built by different queries.
func onlyPickRowForCustomer(t *testing.T, customerID string, includes ...string) map[string]any {
	t.Helper()

	params := url.Values{"customer_ids": {customerID}, "limit": {"10"}}
	for _, inc := range includes {
		params.Add("include", inc)
	}
	list, status, err := apiClient.GetList(picksPath, params)
	require.NoError(t, err)
	require.Equal(t, 200, status, "picks list should return 200")
	require.Len(t, list.Data, 1, "the customer was created for this test and has exactly one pick")

	var row map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &row))
	return row
}

// --- customer PO number and note ------------------------------------------

func TestPicks_CarryTheOrdersPurchaseOrderNumberAndNote(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-po", nil, "")
	po := uniqueName("PO-E2E")
	order, pickID := issuedOrderAndPick(t, customerID, map[string]any{
		"customer_purchase_order_number": po,
		"note":                           "Stage on dock 4",
	})

	// Both are base scalars: the floor reads them off the pick with no include.
	pick := retrievePick(t, pickID)
	assert.Equal(t, po, jsonField(pick, "customer_purchase_order_number"),
		"the pick carries the order's customer PO number")
	assert.Equal(t, jsonField(order, "customer_purchase_order_number"), jsonField(pick, "customer_purchase_order_number"))
	assert.Equal(t, "Stage on dock 4", jsonField(pick, "note"), "the pick carries the order's note")

	row := onlyPickRowForCustomer(t, customerID)
	assert.Equal(t, po, jsonField(row, "customer_purchase_order_number"),
		"the list projection carries them too — the picking index shows the PO in its own column")
	assert.Equal(t, "Stage on dock 4", jsonField(row, "note"))
}

func TestPicks_PurchaseOrderNumberAndNoteAreNullWhenTheOrderHasNone(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-nopo", nil, "")
	_, pickID := issuedOrderAndPick(t, customerID, nil)

	pick := retrievePick(t, pickID)
	assertNilField(t, pick, "customer_purchase_order_number")
	assertNilField(t, pick, "note")

	row := onlyPickRowForCustomer(t, customerID)
	assertNilField(t, row, "customer_purchase_order_number")
	assertNilField(t, row, "note")
}

// --- freight --------------------------------------------------------------

func TestPicks_IncludeFreight(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-freight", nil, "")
	_, pickID := issuedOrderAndPick(t, customerID, map[string]any{
		"carrier_id":                     SeedCarrierID,
		"service_level_id":               SeedServiceLevelID,
		"carrier_billing_type":           "third_party",
		"carrier_billing_account_number": "ACCT-E2E-99",
	})

	assertNilField(t, retrievePick(t, pickID), "freight")
	assertNilField(t, onlyPickRowForCustomer(t, customerID), "freight")

	for label, got := range map[string]map[string]any{
		"detail": retrievePick(t, pickID, "freight"),
		"list":   onlyPickRowForCustomer(t, customerID, "freight"),
	} {
		freight := jsonObject(got, "freight")
		require.NotNil(t, freight, "%s: freight should be present with ?include=freight", label)
		assertObjectField(t, freight, "freight")
		assert.Equal(t, "third_party", jsonField(freight, "billing_type"), "%s", label)
		assert.Equal(t, "ACCT-E2E-99", jsonField(freight, "billing_account_number"), "%s", label)

		// Carrier and service level ride inside the one freight include rather than being
		// separately expandable, so a header renders the whole shipping method in one ask.
		carrier := jsonObject(freight, "carrier")
		require.NotNil(t, carrier, "%s: the carrier rides inside freight", label)
		assertObjectField(t, carrier, "carrier")
		assert.Equal(t, SeedCarrierID, jsonField(carrier, "id"), "%s", label)
		assert.NotEmpty(t, jsonField(carrier, "name"), "%s: the header prints the carrier's name", label)
		// A carrier the account never enabled for the portal must read as hidden, not as an
		// empty string — the field is required and drives a portal-visibility badge.
		assert.Equal(t, "hidden", jsonField(carrier, "customer_portal_visibility"), "%s", label)

		serviceLevel := jsonObject(freight, "service_level")
		require.NotNil(t, serviceLevel, "%s: the service level rides inside freight", label)
		assertObjectField(t, serviceLevel, "service_level")
		assert.Equal(t, SeedServiceLevelID, jsonField(serviceLevel, "id"), "%s", label)
		assert.NotEmpty(t, jsonField(serviceLevel, "name"), "%s", label)
		assert.NotEmpty(t, jsonField(serviceLevel, "service_level_token"), "%s", label)
		assert.Equal(t, "hidden", jsonField(serviceLevel, "customer_portal_visibility"), "%s", label)
	}
}

// An order needs a carrier but not a service level or a billing arrangement, so freight has to
// report the parts that were chosen and null for the rest rather than dropping out entirely.
func TestPicks_FreightReportsNullsForWhatTheOrderLeftUnset(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-freight-part", nil, "")
	body := minimalSalesOrderCreateBody(t, customerID)
	delete(body, "service_level_id")
	pickID := pickForOrderBody(t, body)

	freight := jsonObject(retrievePick(t, pickID, "freight"), "freight")
	require.NotNil(t, freight, "freight is present even when only the carrier was chosen")
	require.NotNil(t, jsonObject(freight, "carrier"), "the carrier is mandatory on an order")
	assertNilField(t, freight, "service_level")
	assertNilField(t, freight, "billing_type")
	assertNilField(t, freight, "billing_account_number")
}

// freight is one include, not a tree — the carrier is already inside it, so a caller asking for
// freight.carrier is asking for something the endpoint does not offer.
func TestPicks_RejectsANestedFreightInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, url.Values{"include": {"freight.carrier"}})
	require.NoError(t, err)
	require.Equal(t, 400, status, "freight has no sub-includes: %s", string(body))
}

// --- created_by -----------------------------------------------------------

// A pick has no creator of its own — issuing the order creates it — so its created_by is resolved
// from the order's create audit event, keyed by the order's id rather than the pick's.
//
// Run against the seeded pick: the create event an order raises through the API is published
// through the outbox, so a freshly issued order reports the `system` fallback until it lands.
func TestPicks_IncludeCreatedBy(t *testing.T) {
	t.Parallel()

	assertNilField(t, retrievePick(t, SeedPickID), "created_by")

	orderCreatedBy := jsonObject(retrieveSalesOrder(t, SeedSalesOrderID, "created_by"), "created_by")
	require.NotNil(t, orderCreatedBy, "the order the pick borrows its creator from must report one")

	for label, got := range map[string]map[string]any{
		"detail": retrievePick(t, SeedPickID, "created_by"),
		"list":   seedPickListRow(t, "created_by"),
	} {
		createdBy := jsonObject(got, "created_by")
		require.NotNil(t, createdBy, "%s: created_by should be present with ?include=created_by", label)
		assertObjectField(t, createdBy, "created_by")
		assert.Equal(t, jsonField(orderCreatedBy, "relation"), jsonField(createdBy, "relation"),
			"%s: the pick reports the relation its order does", label)

		actor := jsonObject(createdBy, "actor")
		require.NotNil(t, actor, "%s: an internal creator names its actor", label)
		assert.Equal(t, jsonField(jsonObject(orderCreatedBy, "actor"), "id"), jsonField(actor, "id"),
			"%s: the pick's creator is the order's creator", label)
	}
}

// The seeded pick's row off the list endpoint, so an include can be checked on the list projection
// as well as on detail.
func seedPickListRow(t *testing.T, includes ...string) map[string]any {
	t.Helper()

	params := url.Values{"customer_ids": {SeedCustomerAccountID}, "limit": {"100"}}
	for _, inc := range includes {
		params.Add("include", inc)
	}
	list, status, err := apiClient.GetList(picksPath, params)
	require.NoError(t, err)
	require.Equal(t, 200, status, "picks list should return 200")

	for _, raw := range list.Data {
		var row map[string]any
		require.NoError(t, json.Unmarshal(raw, &row))
		if jsonField(row, "id") == SeedPickID {
			return row
		}
	}
	require.FailNow(t, "the seeded pick must appear in its customer's pick list")
	return nil
}

func retrieveSalesOrder(t *testing.T, orderID string, includes ...string) map[string]any {
	t.Helper()

	params := url.Values{}
	for _, inc := range includes {
		params.Add("include", inc)
	}
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// --- search ---------------------------------------------------------------

// The picking index has one search box for four different things a picker might have in hand:
// the pick's own number, the order it fulfills, the customer's PO number, and the customer.
func TestPicksList_SearchMatchesTheOrderAndCustomerItCameFrom(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-search", nil, "")
	po := uniqueName("PO-SEARCH")
	order, pickID := issuedOrderAndPick(t, customerID, map[string]any{"customer_purchase_order_number": po})

	customer := retrieveCustomerForPickSearch(t, customerID)

	for _, tc := range []struct {
		name string
		q    string
	}{
		{"pick number", jsonField(retrievePick(t, pickID), "number")},
		{"sales order number", jsonField(order, "number")},
		{"customer PO number", po},
		{"customer name", jsonField(customer, "name")},
		{"customer number", jsonField(customer, "number")},
	} {
		require.NotEmpty(t, tc.q, "%s must be set for the search to prove anything", tc.name)
		assert.Contains(t, listIDs(t, picksPath, url.Values{"q": {tc.q}}), pickID,
			"searching by %s should surface the pick", tc.name)
	}
}

func retrieveCustomerForPickSearch(t *testing.T, customerID string) map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}
