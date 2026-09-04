//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

// Deliveries: what actually arrived against a purchase order, recorded when a receiving order is
// stocked. They are never created directly, so the only writes exercised here are the stocking
// runs that produce them.
//
// Everything a delivery line is worth reading for — the item, what it cost, where it went, which
// lot it joined — is a separately expandable sub-object. A caller that asks only for `lines` gets
// rows of nulls rather than an error, so the include contract is the substance of this file.

const deliveriesPath = "/v1/operations/deliveries"

// --- Retrieve ---

func TestDeliveries_RetrieveResponseShape(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "delivery retrieve must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	delivery := parseJSON(body)
	assertIDFormat(t, jsonField(delivery, "id"), id.DeliveryIDPrefix)
	assertObjectField(t, delivery, "delivery")
	assert.NotEmpty(t, jsonField(delivery, "number"), "a delivery carries a human-readable number")
	assert.Contains(t, []string{"accepted", "rejected"}, jsonField(delivery, "status"))
	assertValidTimestamp(t, jsonField(delivery, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(delivery, "updated_at"), "updated_at")
}

func TestDeliveries_RetrieveUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/dv_doesnotexist00000", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown delivery must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown delivery is a 404: %s", string(body))
}

// --- Expandable fields ---

func TestDeliveries_ExpandablesAreNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	delivery := parseJSON(body)
	assertNilField(t, delivery, "lines")
}

func TestDeliveries_LinesExpandWithInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines, "the seeded delivery has lines: %s", string(body))

	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	assertIDFormat(t, jsonField(line, "id"), id.DeliveryLineIDPrefix)
	assertObjectField(t, line, "delivery_line")

	quantity := jsonObject(line, "quantity")
	require.NotNil(t, quantity, "quantity is required on a line and never expandable: %v", line)
	assert.NotEmpty(t, jsonField(quantity, "value"))
}

// Asking for `lines` alone leaves every sub-object on the line null. The dashboard's delivery
// table renders all four, so this documents the trap: a missing include is silent, not an error.
func TestDeliveries_LineSubObjectsAreNullWhenOnlyLinesRequested(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)

	for _, field := range []string{"item", "unit_cost", "location", "lot"} {
		assertNilField(t, line, field)
	}
}

func TestDeliveries_LineItemAndUnitCostExpandWithInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{
		"include": {"lines", "lines.item", "lines.unit_cost"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)

	item := jsonObject(line, "item")
	require.NotNil(t, item, "lines.item must expand when asked for: %s", string(body))
	assert.NotEmpty(t, jsonField(item, "id"))
	assertObjectField(t, item, "item")

	unitCost := jsonObject(line, "unit_cost")
	require.NotNil(t, unitCost, "lines.unit_cost must expand when asked for: %s", string(body))
	assert.NotEmpty(t, jsonField(unitCost, "value"))
}

func TestDeliveries_RelatedPurchaseOrderExpandsWithInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{
		"include": {"related", "related.purchase_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	related := jsonObject(parseJSON(body), "related")
	require.NotNil(t, related, "related must expand when asked for: %s", string(body))
	assertObjectField(t, related, "delivery_related")

	purchaseOrder := jsonObject(related, "purchase_order")
	require.NotNil(t, purchaseOrder, "a delivery is always received against a purchase order: %s", string(body))
	assert.NotEmpty(t, jsonField(purchaseOrder, "id"))
}

func TestDeliveries_RejectsAnUnknownInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{
		"include": {"lines.not_a_real_field"},
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown include is a client error: %s", string(body))
	assert.Equal(t, 400, status, "include only accepts the documented keys: %s", string(body))
}

// --- List ---

func TestDeliveries_ListResponseShape(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(deliveriesPath, url.Values{"limit": {"3"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "deliveries list must not 5xx")
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data, "the seeded account has deliveries")

	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertObjectField(t, row, "delivery")
		assertIDFormat(t, jsonField(row, "id"), id.DeliveryIDPrefix)
		assert.NotEmpty(t, jsonField(row, "number"))
	}
}

// Deliveries where nothing was accepted are hidden unless asked for, so the default page is not
// simply "everything".
func TestDeliveries_ListDefaultsToAccepted(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(deliveriesPath, url.Values{"limit": {"20"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	for _, raw := range list.Data {
		assert.Equal(t, "accepted", jsonField(parseJSON(raw), "status"),
			"the default page shows accepted deliveries only: %s", string(raw))
	}
}

func TestDeliveries_ListStatusRejectedNarrowsThePage(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(deliveriesPath, url.Values{"status": {"rejected"}, "limit": {"20"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "the rejected filter must not 5xx")
	require.Equal(t, 200, status)

	for _, raw := range list.Data {
		assert.Equal(t, "rejected", jsonField(parseJSON(raw), "status"),
			"asking for rejected returns only rejected: %s", string(raw))
	}
}

func TestDeliveries_ListStatusAllIsAtLeastAsBroadAsTheDefault(t *testing.T) {
	t.Parallel()

	accepted, status, err := apiClient.GetList(deliveriesPath, url.Values{"status": {"accepted"}, "limit": {"50"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	all, status, err := apiClient.GetList(deliveriesPath, url.Values{"status": {"all"}, "limit": {"50"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	assert.GreaterOrEqual(t, len(all.Data), len(accepted.Data),
		"`all` cannot return fewer deliveries than `accepted`")
}

func TestDeliveries_ListRejectsAnUnknownStatus(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath, url.Values{"status": {"bogus_e2e_status"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown status is a client error: %s", string(body))
	require.Equal(t, 400, status, "status only accepts the documented values: %s", string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// --- Date filters ---

// The window is a calendar day, not an instant, and it covers the whole of the end day.
func TestDeliveries_DateWindowInThePastReturnsNothing(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(deliveriesPath, url.Values{
		"starts_at": {"2000-01-01"},
		"ends_at":   {"2000-01-02"},
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "a historical window must not 5xx")
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data, "nothing was delivered in 2000")
}

func TestDeliveries_DateWindowCoveringTodayReturnsTheSeededDeliveries(t *testing.T) {
	t.Parallel()

	today := time.Now().UTC().Format("2006-01-02")
	list, status, err := apiClient.GetList(deliveriesPath, url.Values{
		"starts_at": {"2000-01-01"},
		"ends_at":   {today},
		"limit":     {"20"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.NotEmpty(t, list.Data,
		"a window ending today must include deliveries created today, which requires the end day to be inclusive")
}

// A date the endpoint cannot parse is dropped rather than refused: the filter silently widens to
// everything instead of erroring. Pinned because it is a trap for callers — a client sending an
// ISO instant where a calendar day is wanted gets a full page back and no indication why.
func TestDeliveries_IgnoresAMalformedDate(t *testing.T) {
	t.Parallel()

	for _, param := range []string{"starts_at", "ends_at"} {
		t.Run(param, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(deliveriesPath, url.Values{param: {"not-a-date"}})
			require.NoError(t, err)
			require.Less(t, status, 500, "a malformed date must not 5xx: %s", string(body))
			assert.Equal(t, 200, status,
				"%s is currently ignored when unparseable rather than rejected: %s", param, string(body))
		})
	}
}

// --- Filters ---

func TestDeliveries_ItemFilterWithNoMatchesReturnsEmpty(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(deliveriesPath, url.Values{
		"item_ids": {"it_00000000000000000000000000"},
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unmatched item filter must not 5xx")
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data, "no delivery has a line for an item that does not exist")
}

func TestDeliveries_SupplierFilterWithNoMatchesReturnsEmpty(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(deliveriesPath, url.Values{
		"supplier_ids": {"ac_00000000000000000000000000"},
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unmatched supplier filter must not 5xx")
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data, "no delivery belongs to an account that does not exist")
}

// --- Pagination and validation ---

func TestDeliveries_PaginationAdvances(t *testing.T) {
	t.Parallel()

	assertCursorPaginationAdvances(t, deliveriesPath, url.Values{"status": {"all"}, "limit": {"1"}})
}

func TestDeliveries_RejectsUnknownQueryParam(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, deliveriesPath, status, body)
}

// The location and the lot expand independently of the item and of each other. The seeded delivery
// has one line carrying both and one carrying neither, so each include has to resolve the line that
// has it without failing on the line that does not.
func TestDeliveries_LineLocationAndLotExpandIndependently(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ include, key, field string }{
		{"lines.location", "location", "name"},
		{"lines.lot", "lot", "lot_number"},
	} {
		t.Run(tc.include, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{
				"include": {"lines", tc.include},
			})
			require.NoError(t, err)
			require.Less(t, status, 500, "%s must not 5xx: %s", tc.include, string(body))
			requireStatus(t, 200, status, body)

			lines := jsonListData(parseJSON(body), "lines")
			require.Len(t, lines, 2, "the seeded delivery has two lines: %s", string(body))

			var populated int
			for _, raw := range lines {
				line, ok := raw.(map[string]any)
				require.True(t, ok)
				if sub := jsonObject(line, tc.key); sub != nil {
					populated++
					assert.NotEmpty(t, jsonField(sub, "id"))
					assert.NotEmpty(t, jsonField(sub, tc.field), "%s must carry its %s: %v", tc.key, tc.field, sub)
				}
			}
			assert.Equal(t, 1, populated,
				"exactly the one seeded line carries a %s; the other resolves to null rather than failing", tc.key)
		})
	}
}

// A lot is the batch goods were received under, and its number is what an operator traces. It is
// carried inline off the delivery query rather than fetched, so it has to survive the round trip.
func TestDeliveries_LotCarriesItsNumber(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{
		"include": {"lines", "lines.lot"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var found bool
	for _, raw := range jsonListData(parseJSON(body), "lines") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		if lot := jsonObject(line, "lot"); lot != nil {
			assertObjectField(t, lot, "lot")
			assert.Equal(t, "LOT-DLV-001", jsonField(lot, "lot_number"))
			found = true
		}
	}
	assert.True(t, found, "the seeded delivery has a line under a lot: %s", string(body))
}

// A delivery names the receiving order it was stocked through, not just the purchase order. The
// first page is the common case and was the one that returned null, so the list is checked here
// rather than only the retrieve.
func TestDeliveries_ListRelatedNamesTheReceivingOrder(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(deliveriesPath, url.Values{
		"include": {"related", "related.receiving_order"},
		"limit":   {"5"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	var named int
	for _, raw := range list.Data {
		related := jsonObject(parseJSON(raw), "related")
		require.NotNil(t, related, "related must expand on the list: %s", string(raw))
		if ro := jsonObject(related, "receiving_order"); ro != nil {
			assert.NotEmpty(t, jsonField(ro, "id"))
			assert.NotEmpty(t, jsonField(ro, "number"))
			named++
		}
	}
	assert.Positive(t, named, "a seeded delivery is stocked through a receiving order: %v", list.Data)
}

func TestDeliveries_RetrieveRelatedNamesTheReceivingOrder(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, url.Values{
		"include": {"related", "related.receiving_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	related := jsonObject(parseJSON(body), "related")
	require.NotNil(t, related)

	receivingOrder := jsonObject(related, "receiving_order")
	require.NotNil(t, receivingOrder, "the seeded delivery has a receiving order: %s", string(body))
	assert.NotEmpty(t, jsonField(receivingOrder, "id"))
}
