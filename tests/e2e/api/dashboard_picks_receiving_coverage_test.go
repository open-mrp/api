//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// What the picking and receiving screens rely on beyond the happy path: who may reach each route
// (another tenant, the customer portal, a read-only role, a merchant targeting a counterparty), replays
// of the same request, and the guards around a finished pick and a completed receiving order. Every
// test builds its own pick or receiving order, so none touches a shared seed row.

const dashPicksReplayedHeader = "Idempotent-Replayed"

// dashPicksCall is one write the dashboard sends, with a body a well-formed client would send.
type dashPicksCall struct {
	name   string
	method string
	path   string
	body   any
}

func (c dashPicksCall) send(t *testing.T, client *Client) (int, []byte) {
	t.Helper()
	status, body, err := client.Do(c.method, c.path, c.body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "%s %s must not 5xx: %s", c.method, c.path, string(body))
	return status, body
}

// Every pick write, each of which would visibly change a pick holding one picked unit of three.
func dashPicksWrites(pickID, lineID string) []dashPicksCall {
	pick := picksPath + "/" + pickID
	line := pick + "/lines/" + lineID
	return []dashPicksCall{
		{name: "update line", method: http.MethodPatch, path: line, body: map[string]any{"quantity_value": "2"}},
		{name: "pick all", method: http.MethodPut, path: pick + "/actions/pick"},
		{name: "void", method: http.MethodPut, path: pick + "/actions/void"},
		{name: "pick line", method: http.MethodPut, path: line + "/actions/pick"},
		{name: "void line", method: http.MethodPut, path: line + "/actions/void"},
		{name: "pack", method: http.MethodPost, path: pick + "/actions/pack", body: map[string]any{"shipment_case_count": 1}},
	}
}

// Issues a fresh single-line order for quantity pairs and returns its pick and that pick's line.
func dashPicksFixture(t *testing.T, quantity string) (pickID, lineID string) {
	t.Helper()
	pickID = pickForOrderBody(t, orderBodyForQuantity(t, quantity))
	lineID = firstUnpackedPickLine(t, pickID)
	require.NotEmpty(t, lineID, "a freshly issued order's pick has a line")
	return pickID, lineID
}

// A pick of three with one unit picked: every write changes it, and it is packable.
func dashPicksPartlyPicked(t *testing.T) (pickID, lineID string) {
	t.Helper()
	pickID, lineID = dashPicksFixture(t, "3")
	setPickedQuantity(t, pickID, lineID, "1")
	return pickID, lineID
}

func dashPicksAssertPartlyPicked(t *testing.T, pickID, lineID string) {
	t.Helper()
	quantities := readPickLineQuantities(t, pickID)
	assert.Len(t, quantities, 1, "no pick line was added or removed")
	assert.InDelta(t, 1.0, quantities[lineID], 0.001, "the picked quantity is still the one unit")
	assert.Empty(t, pickShipmentNumbers(t, pickID), "nothing was packed")
	assert.Nil(t, retrievePick(t, pickID)["finished_at"], "the pick is still open")
}

func dashPicksLineQuantity(t *testing.T, pickID, lineID string) float64 {
	t.Helper()
	quantities := readPickLineQuantities(t, pickID)
	require.Contains(t, quantities, lineID)
	return quantities[lineID]
}

func dashPicksQuantityValue(t *testing.T, body []byte) string {
	t.Helper()
	quantity := jsonObject(parseJSON(body), "quantity")
	require.NotNil(t, quantity, "the line carries its quantity: %s", string(body))
	return jsonField(quantity, "value")
}

// Pages through a list as client and returns every id on it.
func dashPicksAllIDs(t *testing.T, client *Client, path string, params url.Values) []string {
	t.Helper()
	params.Set("limit", "100")
	list, status, err := client.GetList(path, params)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	var ids []string
	for range 50 {
		for _, raw := range list.Data {
			ids = append(ids, jsonField(parseJSON(raw), "id"))
		}
		if !list.PageInfo.HasNextPage {
			return ids
		}
		list, status, err = client.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
		require.Equal(t, 200, status)
	}
	t.Fatalf("%s did not finish paging", path)
	return nil
}

func dashPicksRequireForbidden(t *testing.T, status int, body []byte, what string) {
	t.Helper()
	require.Equal(t, 403, status, "%s: %s", what, string(body))
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// ---------------------------------------------------------------------------
// Picks — who may reach them
// ---------------------------------------------------------------------------

// Another tenant learns nothing of the pick and changes none of it: each route is a 404, not a 403.
func TestDashPicks_AnotherTenantCannotSeeOrWorkThePick(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksPartlyPicked(t)
	number := jsonField(retrievePick(t, pickID), "number")
	clientB := getTenantBClient()

	status, body, err := clientB.GetListRaw(picksPath+"/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	require.Equal(t, 404, status, "retrieve: %s", string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")

	assert.NotContains(t, dashPicksAllIDs(t, clientB, picksPath, url.Values{"q": {number}}), pickID,
		"searching the pick's number from another tenant does not find it")
	assert.NotContains(t, dashPicksAllIDs(t, clientB, picksPath, url.Values{"customer_ids": {SeedCustomerAccountID}, "status": {"open"}}), pickID,
		"filtering by the pick's customer from another tenant does not find it")

	for _, call := range dashPicksWrites(pickID, lineID) {
		status, body := call.send(t, clientB)
		assert.Equal(t, 404, status, "%s from another tenant: %s", call.name, string(body))
	}

	dashPicksAssertPartlyPicked(t, pickID, lineID)
}

// The pick is the seller's warehouse record: even the customer whose order it is can neither read nor
// work it.
func TestDashPicks_CustomerPortalCannotSeeOrWorkTheSellersPick(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksPartlyPicked(t)
	portal := getCustomerPortalClient()

	status, body, err := portal.GetListRaw(picksPath, url.Values{"customer_ids": {SeedCustomerAccountID}})
	require.NoError(t, err)
	dashPicksRequireForbidden(t, status, body, "list")

	status, body, err = portal.GetListRaw(picksPath+"/"+pickID, nil)
	require.NoError(t, err)
	dashPicksRequireForbidden(t, status, body, "retrieve")

	for _, call := range dashPicksWrites(pickID, lineID) {
		status, body := call.send(t, portal)
		dashPicksRequireForbidden(t, status, body, call.name)
	}

	dashPicksAssertPartlyPicked(t, pickID, lineID)
}

// Targeting the customer's account lists that account's own picks, never the seller's, and the
// single-pick routes refuse a cross-account caller outright.
func TestDashPicks_MerchantTargetingTheCustomerAccountCannotReachTheSellersPick(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksPartlyPicked(t)
	number := jsonField(retrievePick(t, pickID), "number")
	asCustomer := apiClient.WithAccountID(SeedCustomerAccountID)

	assert.NotContains(t, dashPicksAllIDs(t, asCustomer, picksPath, url.Values{"q": {number}}), pickID,
		"the seller's pick is not one of the customer account's picks")

	status, body, err := asCustomer.GetListRaw(picksPath+"/"+pickID, nil)
	require.NoError(t, err)
	dashPicksRequireForbidden(t, status, body, "retrieve")

	for _, call := range dashPicksWrites(pickID, lineID) {
		status, body := call.send(t, asCustomer)
		dashPicksRequireForbidden(t, status, body, call.name)
	}

	dashPicksAssertPartlyPicked(t, pickID, lineID)
}

// picks:read is enough to watch the floor, not to change it.
func TestDashPicks_ReadOnlyRoleCannotWorkAPick(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksPartlyPicked(t)
	reader := customRoleClient(t, "picks:read")

	status, body, err := reader.GetListRaw(picksPath+"/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, call := range dashPicksWrites(pickID, lineID) {
		status, body := call.send(t, reader)
		dashPicksRequireForbidden(t, status, body, call.name)
	}

	dashPicksAssertPartlyPicked(t, pickID, lineID)
}

// ---------------------------------------------------------------------------
// Picks — replays
// ---------------------------------------------------------------------------

// A retried update answers with the first result and does not overwrite a later change; the same key
// with another quantity is refused.
func TestDashPicks_UpdateLineReplayDoesNotReapply(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksFixture(t, "3")
	path := picksPath + "/" + pickID + "/lines/" + lineID
	key := newIdempotencyKey()

	first, err := apiClient.PatchFull(path, map[string]any{"quantity_value": "1"}, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	assert.NotEqual(t, "true", first.Header.Get(dashPicksReplayedHeader))

	setPickedQuantity(t, pickID, lineID, "2")

	replay, err := apiClient.PatchFull(path, map[string]any{"quantity_value": "1"}, key)
	require.NoError(t, err)
	requireStatus(t, 200, replay.StatusCode, replay.Body)
	assert.Equal(t, "true", replay.Header.Get(dashPicksReplayedHeader))
	assert.JSONEq(t, string(first.Body), string(replay.Body), "a replay answers with the first response")
	assert.InDelta(t, 2.0, dashPicksLineQuantity(t, pickID, lineID), 0.001, "the replay did not write the line again")

	status, body, err := apiClient.Patch(path, map[string]any{"quantity_value": "3"}, key)
	require.NoError(t, err)
	require.Equal(t, 400, status, "%s", string(body))
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")
	assert.InDelta(t, 2.0, dashPicksLineQuantity(t, pickID, lineID), 0.001)
}

// The line actions are PUTs, but a retry carrying the same key still answers from the first call
// rather than picking (or voiding) the line over again after it was changed in between.
func TestDashPicks_LineActionReplayDoesNotReapply(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksFixture(t, "3")
	linePath := picksPath + "/" + pickID + "/lines/" + lineID

	pickKey := newIdempotencyKey()
	status, picked, err := apiClient.Do(http.MethodPut, linePath+"/actions/pick", nil, pickKey)
	require.NoError(t, err)
	requireStatus(t, 200, status, picked)
	assertDecimalEqual(t, "3", dashPicksQuantityValue(t, picked))

	status, body, err := apiClient.Put(linePath+"/actions/void", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, replay, err := apiClient.Do(http.MethodPut, linePath+"/actions/pick", nil, pickKey)
	require.NoError(t, err)
	requireStatus(t, 200, status, replay)
	assert.JSONEq(t, string(picked), string(replay), "a replayed pick answers with the first response")
	assert.InDelta(t, 0.0, dashPicksLineQuantity(t, pickID, lineID), 0.001, "the replay did not pick the voided line again")

	voidKey := newIdempotencyKey()
	status, voided, err := apiClient.Do(http.MethodPut, linePath+"/actions/void", nil, voidKey)
	require.NoError(t, err)
	requireStatus(t, 200, status, voided)

	status, body, err = apiClient.Put(linePath+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, replay, err = apiClient.Do(http.MethodPut, linePath+"/actions/void", nil, voidKey)
	require.NoError(t, err)
	requireStatus(t, 200, status, replay)
	assert.JSONEq(t, string(voided), string(replay), "a replayed void answers with the first response")
	assert.InDelta(t, 3.0, dashPicksLineQuantity(t, pickID, lineID), 0.001, "the replay did not void the re-picked line")
}

// A retried pack names the same job and ships the goods once; the same key with another case count
// is refused rather than packing again.
func TestDashPicks_PackReplayShipsOnce(t *testing.T) {
	t.Parallel()
	pickID, _ := dashPicksFixture(t, "3")
	pickAllLines(t, pickID)
	path := picksPath + "/" + pickID + "/actions/pack"
	key := newIdempotencyKey()

	first, err := apiClient.PostFull(path, map[string]any{"shipment_case_count": 1}, key)
	require.NoError(t, err)
	requireStatus(t, 202, first.StatusCode, first.Body)
	jobID := jsonField(parseJSON(first.Body), "id")
	require.NotEmpty(t, jobID)

	replay, err := apiClient.PostFull(path, map[string]any{"shipment_case_count": 1}, key)
	require.NoError(t, err)
	requireStatus(t, 202, replay.StatusCode, replay.Body)
	assert.Equal(t, "true", replay.Header.Get(dashPicksReplayedHeader))
	assert.Equal(t, jobID, jsonField(parseJSON(replay.Body), "id"), "the replay names the first job")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "%v", job)

	status, body, err := apiClient.Post(path, map[string]any{"shipment_case_count": 2}, key)
	require.NoError(t, err)
	require.Equal(t, 400, status, "%s", string(body))
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")

	assert.Len(t, pickShipmentNumbers(t, pickID), 1, "the pick was packed into exactly one shipment")
	assert.NotNil(t, retrievePick(t, pickID)["finished_at"], "the one pack finished the pick")
}

// ---------------------------------------------------------------------------
// Picks — validation and state
// ---------------------------------------------------------------------------

// quantity_value may be left out of a non-empty body, but never sent as null, blank, negative, or a
// bare number.
func TestDashPicks_UpdateLineRejectsMalformedQuantities(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksFixture(t, "3")
	setPickedQuantity(t, pickID, lineID, "1")
	path := picksPath + "/" + pickID + "/lines/" + lineID

	for name, value := range map[string]any{"null": nil, "blank": "", "negative": "-1", "number": 2} {
		status, body, err := apiClient.Patch(path, map[string]any{"quantity_value": value}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "%s: %s", name, string(body))
		assert.Equal(t, 400, status, "quantity_value %s is refused: %s", name, string(body))
	}

	status, body, err := apiClient.Patch(path, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an empty body names nothing to change: %s", string(body))

	assert.InDelta(t, 1.0, dashPicksLineQuantity(t, pickID, lineID), 0.001, "the line kept its picked quantity")
}

// A pick with nothing picked has nothing to ship, so the pack is refused when it is asked for rather
// than accepted as a job that fails later.
func TestDashPicks_PackRefusesAnUnpickedPick(t *testing.T) {
	t.Parallel()
	pickID, _ := dashPicksFixture(t, "3")
	path := picksPath + "/" + pickID + "/actions/pack"

	status, body, err := apiClient.Post(path, map[string]any{"shipment_case_count": 1}, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, 400, status, "%s", string(body))
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")

	for name, count := range map[string]any{"negative": -1, "string": "one", "fraction": 1.5} {
		status, body, err := apiClient.Post(path, map[string]any{"shipment_case_count": count}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "%s: %s", name, string(body))
		assert.Equal(t, 400, status, "a %s case count is refused: %s", name, string(body))
	}

	assert.Empty(t, pickShipmentNumbers(t, pickID), "nothing was packed")
}

// The date window takes calendar dates only; anything else is refused rather than ignored.
func TestDashPicks_ListRejectsAMalformedDate(t *testing.T) {
	t.Parallel()

	for _, param := range []string{"starts_at", "ends_at"} {
		for _, value := range []string{"06/10/2026", "2026-13-01", "yesterday"} {
			status, body, err := apiClient.GetListRaw(picksPath, url.Values{param: {value}})
			require.NoError(t, err)
			require.Less(t, status, 500, "%s=%s: %s", param, value, string(body))
			assert.Equal(t, 400, status, "%s=%s is refused: %s", param, value, string(body))
		}
	}
}

// Voiding resets a pick rather than retiring it, so it can be picked again from the start.
func TestDashPicks_AVoidedPickCanBePickedAgain(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksFixture(t, "3")

	for _, action := range []string{"pick", "void", "pick"} {
		status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/"+action, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}

	assert.InDelta(t, 3.0, dashPicksLineQuantity(t, pickID, lineID), 0.001, "the voided pick is picked in full again")
}

// Once every line is packed the pick is finished, and pick-all leaves it exactly as it was.
func TestDashPicks_PickAllLeavesAFinishedPickAlone(t *testing.T) {
	t.Parallel()
	pickID, lineID := dashPicksFixture(t, "3")
	setPickedQuantity(t, pickID, lineID, "3")
	packPick(t, pickID)
	require.NotNil(t, retrievePick(t, pickID)["finished_at"])
	before := readPickLineQuantities(t, pickID)

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assert.Equal(t, before, readPickLineQuantities(t, pickID), "no line of a finished pick was touched")
	assert.NotNil(t, parseJSON(body)["finished_at"], "the pick stays finished")
	assert.Len(t, pickShipmentNumbers(t, pickID), 1)
}

// ---------------------------------------------------------------------------
// Receiving orders — fixtures
// ---------------------------------------------------------------------------

// Issues and receives a purchase order for four pairs of productID, returning its receiving order and line.
func dashPicksReceivingOrderFor(t *testing.T, productID string) (receivingOrderID, lineID string) {
	t.Helper()
	line := purchaseOrderLineBody("E2E-DASH-RCV")
	line["product_id"] = productID
	_, receivingOrderID = receivedPurchaseOrderReceivingOf(t, func(b map[string]any) {
		b["lines"] = []map[string]any{line}
	})
	lineID = jsonField(firstLine(t, receivingOrderID), "id")
	require.NotEmpty(t, lineID)
	return receivingOrderID, lineID
}

// A received order for four pairs of a fresh product, so the product's stock starts at zero.
func dashPicksReceivingFixture(t *testing.T) (receivingOrderID, lineID, itemID string) {
	t.Helper()
	productID, itemID := newProductItemIDs(t, "e2e-dash-rcv")
	receivingOrderID, lineID = dashPicksReceivingOrderFor(t, productID)
	return receivingOrderID, lineID, itemID
}

// Puts quantity pairs of one line away at the seeded location.
func dashPicksStockItems(lineID, quantity string) []map[string]any {
	return []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs(quantity), "location_id": SeedLocationID}},
	}}
}

func dashPicksStockBody(lineID, quantity string) map[string]any {
	return map[string]any{"line_items": dashPicksStockItems(lineID, quantity)}
}

// Every receiving write, each of which would visibly change an order with two of four pairs counted.
func dashPicksReceivingWrites(receivingOrderID, lineID string) []dashPicksCall {
	order := receivingOrdersPath + "/" + receivingOrderID
	line := order + "/lines/" + lineID
	return []dashPicksCall{
		{name: "update line", method: http.MethodPatch, path: line, body: map[string]any{"quantity": pairs("1")}},
		{name: "receive", method: http.MethodPut, path: order + "/actions/receive"},
		{name: "void", method: http.MethodPut, path: order + "/actions/void"},
		{name: "receive line", method: http.MethodPut, path: line + "/actions/receive"},
		{name: "void line", method: http.MethodPut, path: line + "/actions/void"},
		{name: "stock", method: http.MethodPost, path: order + "/actions/stock", body: dashPicksStockBody(lineID, "2")},
	}
}

// Two of four pairs counted and nothing stocked: every write changes it.
func dashPicksPartlyCounted(t *testing.T) (receivingOrderID, lineID, itemID string) {
	t.Helper()
	receivingOrderID, lineID, itemID = dashPicksReceivingFixture(t)
	setReceivingLineQuantity(t, receivingOrderID, lineID, "2")
	return receivingOrderID, lineID, itemID
}

func dashPicksAssertPartlyCounted(t *testing.T, receivingOrderID, itemID string) {
	t.Helper()
	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 1, "no receiving line was added or removed")
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	assertDecimalEqual(t, "2", lineQuantityValue(t, line), "the counted quantity is unchanged")
	assert.Empty(t, jsonField(line, "stocked_at"), "nothing was stocked")
	assert.Zero(t, dashPicksDeliveryCount(t, receivingOrderID), "no delivery was recorded")
	assertDecimalEqual(t, "0", readInventory(t, itemID).onHand.String(), "nothing reached inventory")
}

func dashPicksDeliveryCount(t *testing.T, receivingOrderID string) int {
	t.Helper()
	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID,
		url.Values{"include": {"related", "related.deliveries"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return len(jsonListData(jsonObject(parseJSON(body), "related"), "deliveries"))
}

func dashPicksReceivingLine(t *testing.T, receivingOrderID string) map[string]any {
	t.Helper()
	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 1)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	return line
}

func dashPicksStockChangeLogs(t *testing.T, itemID string) int {
	t.Helper()
	logs, status, err := apiClient.GetList(inventoryChangeLogsPath, url.Values{
		"item_ids": {itemID}, "action_types": {"system_action"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	return len(logs.Data)
}

func dashPicksReceivingIDs(t *testing.T, params url.Values) []string {
	t.Helper()
	return dashPicksAllIDs(t, apiClient, receivingOrdersPath, params)
}

// ---------------------------------------------------------------------------
// Receiving orders — who may reach them
// ---------------------------------------------------------------------------

// Another tenant learns nothing of the receiving order and changes none of it, stocking included.
func TestDashReceiving_AnotherTenantCannotSeeOrWorkTheOrder(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksPartlyCounted(t)
	clientB := getTenantBClient()

	status, body, err := clientB.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	require.Equal(t, 404, status, "retrieve: %s", string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")

	for _, listStatus := range []string{"open", "all"} {
		assert.NotContains(t, dashPicksAllIDs(t, clientB, receivingOrdersPath, url.Values{"status": {listStatus}, "item_ids": {itemID}}), receivingOrderID,
			"another tenant's %s list does not carry the order", listStatus)
		assert.NotContains(t, dashPicksAllIDs(t, clientB, receivingOrdersPath, url.Values{"status": {listStatus}, "supplier_ids": {SeedSupplierAccountID}}), receivingOrderID,
			"another tenant's %s list filtered by the order's supplier does not carry it", listStatus)
	}

	for _, call := range dashPicksReceivingWrites(receivingOrderID, lineID) {
		status, body := call.send(t, clientB)
		assert.Equal(t, 404, status, "%s from another tenant: %s", call.name, string(body))
	}

	dashPicksAssertPartlyCounted(t, receivingOrderID, itemID)
}

// Receiving is the seller's back office; the customer portal reaches none of it.
func TestDashReceiving_CustomerPortalCannotSeeOrWorkReceivingOrders(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksPartlyCounted(t)
	portal := getCustomerPortalClient()

	status, body, err := portal.GetListRaw(receivingOrdersPath, nil)
	require.NoError(t, err)
	dashPicksRequireForbidden(t, status, body, "list")

	status, body, err = portal.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, nil)
	require.NoError(t, err)
	dashPicksRequireForbidden(t, status, body, "retrieve")

	for _, call := range dashPicksReceivingWrites(receivingOrderID, lineID) {
		status, body := call.send(t, portal)
		dashPicksRequireForbidden(t, status, body, call.name)
	}

	dashPicksAssertPartlyCounted(t, receivingOrderID, itemID)
}

// Receiving orders are only the account's own: a merchant targeting its supplier's account is refused
// on every route rather than shown that account's orders.
func TestDashReceiving_MerchantTargetingTheSupplierAccountIsRefused(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksPartlyCounted(t)
	asSupplier := apiClient.WithAccountID(SeedSupplierAccountID)

	status, body, err := asSupplier.GetListRaw(receivingOrdersPath, nil)
	require.NoError(t, err)
	dashPicksRequireForbidden(t, status, body, "list")

	status, body, err = asSupplier.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, nil)
	require.NoError(t, err)
	dashPicksRequireForbidden(t, status, body, "retrieve")

	for _, call := range dashPicksReceivingWrites(receivingOrderID, lineID) {
		status, body := call.send(t, asSupplier)
		dashPicksRequireForbidden(t, status, body, call.name)
	}

	dashPicksAssertPartlyCounted(t, receivingOrderID, itemID)
}

// receiving_orders:read is enough to see what is due in, not to receive or stock it.
func TestDashReceiving_ReadOnlyRoleCannotWorkAnOrder(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksPartlyCounted(t)
	reader := customRoleClient(t, "receiving_orders:read")

	status, body, err := reader.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, call := range dashPicksReceivingWrites(receivingOrderID, lineID) {
		status, body := call.send(t, reader)
		dashPicksRequireForbidden(t, status, body, call.name)
	}

	dashPicksAssertPartlyCounted(t, receivingOrderID, itemID)
}

// ---------------------------------------------------------------------------
// Receiving orders — replays
// ---------------------------------------------------------------------------

// A retried stocking answers with the first result and puts the goods away once: one delivery, one
// inventory change, four pairs on hand. The same key with another split is refused.
func TestDashReceiving_StockReplayPutsStockAwayOnce(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksReceivingFixture(t)
	path := receivingOrdersPath + "/" + receivingOrderID + "/actions/stock"
	key := newIdempotencyKey()

	first, err := apiClient.PostFull(path, dashPicksStockBody(lineID, "4"), key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	require.NotEmpty(t, jsonField(parseJSON(first.Body), "completed_at"))

	replay, err := apiClient.PostFull(path, dashPicksStockBody(lineID, "4"), key)
	require.NoError(t, err)
	requireStatus(t, 200, replay.StatusCode, replay.Body)
	assert.Equal(t, "true", replay.Header.Get(dashPicksReplayedHeader))
	assert.JSONEq(t, string(first.Body), string(replay.Body), "a replay answers with the first response")

	status, body, err := apiClient.Post(path, dashPicksStockBody(lineID, "3"), key)
	require.NoError(t, err)
	require.Equal(t, 400, status, "%s", string(body))
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")

	assertDecimalEqual(t, "4", readInventory(t, itemID).onHand.String(), "four pairs on hand, not eight")
	assert.Equal(t, 1, dashPicksDeliveryCount(t, receivingOrderID), "one delivery")
	assert.Equal(t, 1, dashPicksStockChangeLogs(t, itemID), "one inventory change")
}

// A retried line update answers with the first result and does not overwrite a later count; the same
// key with another quantity is refused.
func TestDashReceiving_UpdateLineReplayDoesNotReapply(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, _ := dashPicksReceivingFixture(t)
	path := receivingOrdersPath + "/" + receivingOrderID + "/lines/" + lineID
	key := newIdempotencyKey()

	first, err := apiClient.PatchFull(path, map[string]any{"quantity": pairs("1")}, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)

	setReceivingLineQuantity(t, receivingOrderID, lineID, "2")

	replay, err := apiClient.PatchFull(path, map[string]any{"quantity": pairs("1")}, key)
	require.NoError(t, err)
	requireStatus(t, 200, replay.StatusCode, replay.Body)
	assert.Equal(t, "true", replay.Header.Get(dashPicksReplayedHeader))
	assert.JSONEq(t, string(first.Body), string(replay.Body), "a replay answers with the first response")
	assertDecimalEqual(t, "2", lineQuantityValue(t, dashPicksReceivingLine(t, receivingOrderID)), "the replay did not write the line again")

	status, body, err := apiClient.Patch(path, map[string]any{"quantity": pairs("3")}, key)
	require.NoError(t, err)
	require.Equal(t, 400, status, "%s", string(body))
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")
	assertDecimalEqual(t, "2", lineQuantityValue(t, dashPicksReceivingLine(t, receivingOrderID)))
}

// Receive and void are PUTs, but a retry carrying the same key answers from the first call instead of
// receiving (or voiding) again after the order moved on. A void does not retire the order: a fresh
// receive after it records the outstanding quantity again.
func TestDashReceiving_ActionReplayDoesNotReapply(t *testing.T) {
	t.Parallel()
	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")
	order := receivingOrdersPath + "/" + receivingOrderID
	line := order + "/lines/" + lineID
	counted := func() string { return lineQuantityValue(t, dashPicksReceivingLine(t, receivingOrderID)) }

	put := func(path, key string) []byte {
		t.Helper()
		status, body, err := apiClient.Do(http.MethodPut, path, nil, key)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		return body
	}

	receiveKey := newIdempotencyKey()
	received := put(order+"/actions/receive", receiveKey)
	assertDecimalEqual(t, "4", counted())
	put(order+"/actions/void", newIdempotencyKey())
	assert.JSONEq(t, string(received), string(put(order+"/actions/receive", receiveKey)), "a replayed receive answers with the first response")
	assertDecimalEqual(t, "0", counted(), "the replayed receive did not count the voided order again")

	put(order+"/actions/receive", newIdempotencyKey())
	assertDecimalEqual(t, "4", counted(), "a fresh receive on the voided order counts it again")

	voidKey := newIdempotencyKey()
	voided := put(order+"/actions/void", voidKey)
	put(order+"/actions/receive", newIdempotencyKey())
	assert.JSONEq(t, string(voided), string(put(order+"/actions/void", voidKey)), "a replayed void answers with the first response")
	assertDecimalEqual(t, "4", counted(), "the replayed void did not reset the re-received order")

	lineVoidKey := newIdempotencyKey()
	lineVoided := put(line+"/actions/void", lineVoidKey)
	assertDecimalEqual(t, "0", counted())
	put(line+"/actions/receive", newIdempotencyKey())
	assert.JSONEq(t, string(lineVoided), string(put(line+"/actions/void", lineVoidKey)), "a replayed line void answers with the first response")
	assertDecimalEqual(t, "4", counted(), "the replayed line void did not reset the line")

	lineReceiveKey := newIdempotencyKey()
	put(line+"/actions/void", newIdempotencyKey())
	lineReceived := put(line+"/actions/receive", lineReceiveKey)
	put(line+"/actions/void", newIdempotencyKey())
	assert.JSONEq(t, string(lineReceived), string(put(line+"/actions/receive", lineReceiveKey)), "a replayed line receive answers with the first response")
	assertDecimalEqual(t, "0", counted(), "the replayed line receive did not count the voided line again")
}

// ---------------------------------------------------------------------------
// Receiving orders — validation and state
// ---------------------------------------------------------------------------

// The quantity travels with its unit and may be left out, but not sent as null or half a quantity.
func TestDashReceiving_UpdateLineRejectsNullAndIncompleteQuantities(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, _ := dashPicksReceivingFixture(t)

	for name, quantity := range map[string]any{
		"null":            nil,
		"missing unit":    map[string]any{"value": "1"},
		"missing value":   map[string]any{"unit_id": SeedUnitID},
		"non-numeric":     pairs("two"),
		"unknown unit id": map[string]any{"value": "1", "unit_id": "un_01doesnotexist00000"},
	} {
		status, body := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": quantity})
		assert.Equal(t, 400, status, "a %s quantity is refused: %s", name, string(body))
	}

	assertDecimalEqual(t, "4", lineQuantityValue(t, dashPicksReceivingLine(t, receivingOrderID)), "the line kept its count")
}

// Once complete, the order's lines are what went into inventory. Receive leaves them alone, the line
// writes are refused, and stocking again books nothing.
func TestDashReceiving_ACompletedOrderCannotBeReceivedOrStockedAgain(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksReceivingFixture(t)
	status, body := stockReceivingOrder(t, receivingOrderID, dashPicksStockItems(lineID, "4"))
	requireStatus(t, 200, status, body)
	stockedAt := jsonField(dashPicksReceivingLine(t, receivingOrderID), "stocked_at")
	require.NotEmpty(t, stockedAt)

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/receive", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"), "receive leaves the order complete")

	line := receivingOrdersPath + "/" + receivingOrderID + "/lines/" + lineID
	for _, call := range []dashPicksCall{
		{name: "update line", method: http.MethodPatch, path: line, body: map[string]any{"quantity": pairs("5")}},
		{name: "receive line", method: http.MethodPut, path: line + "/actions/receive"},
		{name: "void line", method: http.MethodPut, path: line + "/actions/void"},
		{name: "stock the stocked line", method: http.MethodPost, path: receivingOrdersPath + "/" + receivingOrderID + "/actions/stock", body: dashPicksStockBody(lineID, "1")},
	} {
		status, body := call.send(t, apiClient)
		require.Equal(t, 400, status, "%s on a completed order: %s", call.name, string(body))
		requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	}

	status, body = stockReceivingOrder(t, receivingOrderID, nil)
	requireStatus(t, 200, status, body)

	after := dashPicksReceivingLine(t, receivingOrderID)
	assertDecimalEqual(t, "4", lineQuantityValue(t, after))
	assert.Equal(t, stockedAt, jsonField(after, "stocked_at"))
	assertDecimalEqual(t, "4", readInventory(t, itemID).onHand.String(), "nothing more reached inventory")
	assert.Equal(t, 1, dashPicksDeliveryCount(t, receivingOrderID))
}

// Voiding a completed order only reopens it: the stocked line keeps its quantity and stays stocked, and
// the inventory it booked is not reversed, so stocking the reopened order books nothing twice.
func TestDashReceiving_VoidingACompletedOrderReopensItWithoutUnstocking(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksReceivingFixture(t)
	status, body := stockReceivingOrder(t, receivingOrderID, dashPicksStockItems(lineID, "4"))
	requireStatus(t, 200, status, body)

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/void", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["completed_at"], "the order is open again")

	line := dashPicksReceivingLine(t, receivingOrderID)
	assertDecimalEqual(t, "4", lineQuantityValue(t, line), "the stocked line keeps its quantity")
	assert.NotEmpty(t, jsonField(line, "stocked_at"), "and stays stocked")

	status, body = stockReceivingOrder(t, receivingOrderID, dashPicksStockItems(lineID, "4"))
	require.Equal(t, 400, status, "the still-stocked line cannot be stocked again: %s", string(body))
	status, body = stockReceivingOrder(t, receivingOrderID, nil)
	requireStatus(t, 200, status, body)

	assertDecimalEqual(t, "4", readInventory(t, itemID).onHand.String(), "the void neither reversed nor repeated the put-away")
	assert.Equal(t, 1, dashPicksDeliveryCount(t, receivingOrderID))
}

// An order line stocked in two rounds has two stocked lines, each with its own delivery. Voiding the
// completed order (or refusing to) must not lose either: both deliveries keep their lines, and the
// order still accounts for all four pairs put away.
func TestDashReceiving_VoidingAnOrderStockedInTwoRoundsKeepsBothDeliveries(t *testing.T) {
	t.Parallel()
	receivingOrderID, firstLineID, itemID := dashPicksReceivingFixture(t)
	setReceivingLineQuantity(t, receivingOrderID, firstLineID, "2")
	status, body := stockReceivingOrder(t, receivingOrderID, dashPicksStockItems(firstLineID, "2"))
	requireStatus(t, 200, status, body)

	var remainderID string
	for _, raw := range receivingOrderLines(t, receivingOrderID) {
		if id := jsonField(raw.(map[string]any), "id"); id != firstLineID {
			remainderID = id
		}
	}
	require.NotEmpty(t, remainderID, "stocking short opens a remainder line")
	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/lines/"+remainderID+"/actions/receive", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body = stockReceivingOrder(t, receivingOrderID, dashPicksStockItems(remainderID, "2"))
	requireStatus(t, 200, status, body)
	require.NotEmpty(t, jsonField(parseJSON(body), "completed_at"))

	status, body, err = apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/void", nil)
	require.NoError(t, err)
	require.Contains(t, []int{200, 400}, status, "%s", string(body))

	assertDecimalEqual(t, "4", readInventory(t, itemID).onHand.String())
	assert.InDelta(t, 1.0, stageCompletion(t, receivingOrderTotals(t, receivingOrderID), "stocked"), 0.0001,
		"the order still accounts for every pair put away")

	deliveries, status, err := apiClient.GetList(deliveriesPath, url.Values{"item_ids": {itemID}, "include": {"lines"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Len(t, deliveries.Data, 2, "both deliveries still carry the item")
	for _, raw := range deliveries.Data {
		delivery := parseJSON(raw)
		assert.Len(t, jsonListData(delivery, "lines"), 1, "delivery %s still lists its line", jsonField(delivery, "number"))
	}
}

// A line with nothing counted on it has nothing to put away: naming it is refused, and leaving it out
// stocks nothing and leaves the order open.
func TestDashReceiving_StockRefusesALineWithNothingReceived(t *testing.T) {
	t.Parallel()
	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, dashPicksStockItems(lineID, "1"))
	require.Equal(t, 400, status, "%s", string(body))
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Equal(t, "line_items.receiving_order_line_id", errObj["param"])

	status, body = stockReceivingOrder(t, receivingOrderID, nil)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["completed_at"], "nothing was stocked, so the order stays open")
	assert.Empty(t, jsonField(dashPicksReceivingLine(t, receivingOrderID), "stocked_at"))
	assert.Zero(t, dashPicksDeliveryCount(t, receivingOrderID))
}

// Each line item is checked before anything is written: a zero allocation, a negative refusal, and a
// missing line id are all refused, and nothing is stocked.
func TestDashReceiving_StockRefusesMalformedLineItems(t *testing.T) {
	t.Parallel()
	receivingOrderID, lineID, itemID := dashPicksReceivingFixture(t)

	for name, item := range map[string]map[string]any{
		"zero allocation": {
			"receiving_order_line_id": lineID,
			"allocations":             []map[string]any{{"quantity": pairs("0"), "location_id": SeedLocationID}},
		},
		"negative refusal": {
			"receiving_order_line_id": lineID,
			"rejected_quantity":       pairs("-1"),
		},
		"missing line id": {
			"allocations": []map[string]any{{"quantity": pairs("1"), "location_id": SeedLocationID}},
		},
	} {
		status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{item})
		assert.Equal(t, 400, status, "%s is refused: %s", name, string(body))
	}

	assert.Empty(t, jsonField(dashPicksReceivingLine(t, receivingOrderID), "stocked_at"), "nothing was stocked")
	assertDecimalEqual(t, "0", readInventory(t, itemID).onHand.String())
	assert.Zero(t, dashPicksDeliveryCount(t, receivingOrderID))
}

// Goods can only be put away at one of the account's own locations. A location that does not exist,
// or that another tenant owns, would leave a receipt nobody can find on the floor.
func TestDashReceiving_StockRefusesALocationTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()

	status, body, err := getTenantBClient().Post(locationsPath, map[string]any{
		"name": uniqueName("e2e-dash-rcv-foreign-loc"),
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	foreignLocationID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = getTenantBClient().Delete(locationsPath + "/" + foreignLocationID) })

	for name, locationID := range map[string]string{
		"unknown location":          "sglc_01doesnotexist0000",
		"another tenant's location": foreignLocationID,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			receivingOrderID, lineID, itemID := dashPicksReceivingFixture(t)

			status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
				"receiving_order_line_id": lineID,
				"allocations":             []map[string]any{{"quantity": pairs("4"), "location_id": locationID}},
			}})
			assert.Contains(t, []int{400, 404}, status, "the location is refused: %s", string(body))

			assert.Empty(t, jsonField(dashPicksReceivingLine(t, receivingOrderID), "stocked_at"), "nothing was stocked")
			assertDecimalEqual(t, "0", readInventory(t, itemID).onHand.String(), "nothing reached inventory")
			assert.Zero(t, dashPicksDeliveryCount(t, receivingOrderID), "no delivery was recorded")
		})
	}
}

// ---------------------------------------------------------------------------
// Receiving orders — list filters
// ---------------------------------------------------------------------------

// Completed orders are hidden unless asked for; `open`, `completed` and `all` each return their own.
func TestDashReceiving_ListStatusSplitsOpenFromCompleted(t *testing.T) {
	t.Parallel()
	productID, itemID := newProductItemIDs(t, "e2e-dash-rcv-status")
	openID, _ := dashPicksReceivingOrderFor(t, productID)
	completedID, completedLineID := dashPicksReceivingOrderFor(t, productID)
	status, body := stockReceivingOrder(t, completedID, dashPicksStockItems(completedLineID, "4"))
	requireStatus(t, 200, status, body)

	byItem := func(status string) []string {
		params := url.Values{"item_ids": {itemID}}
		if status != "" {
			params.Set("status", status)
		}
		return dashPicksReceivingIDs(t, params)
	}
	assert.Equal(t, []string{openID}, byItem(""), "the default hides the completed order")
	assert.Equal(t, []string{openID}, byItem("open"))
	assert.Equal(t, []string{completedID}, byItem("completed"))
	assert.ElementsMatch(t, []string{openID, completedID}, byItem("all"))

	status, body, err := apiClient.GetListRaw(receivingOrdersPath, url.Values{"status": {"closed"}})
	require.NoError(t, err)
	require.Equal(t, 400, status, "%s", string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// The window is on creation. A bare date covers that whole UTC day at either end; an RFC 3339 value is
// taken as the exact instant. Anything else is refused.
func TestDashReceiving_ListFiltersByCreationWindow(t *testing.T) {
	t.Parallel()
	receivingOrderID, _, itemID := dashPicksReceivingFixture(t)
	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	created, err := time.Parse(time.RFC3339, jsonField(parseJSON(body), "created_at"))
	require.NoError(t, err)
	created = created.UTC()
	day := func(offset int) string { return created.AddDate(0, 0, offset).Format("2006-01-02") }
	instant := func(offset time.Duration) string { return created.Add(offset).Format(time.RFC3339) }

	found := func(window url.Values) bool {
		window.Set("item_ids", itemID)
		ids := dashPicksReceivingIDs(t, window)
		assert.LessOrEqual(t, len(ids), 1, "only the one order carries the item")
		return len(ids) == 1 && ids[0] == receivingOrderID
	}
	for name, tc := range map[string]struct {
		window url.Values
		want   bool
	}{
		"starts the day it was created":  {url.Values{"starts_at": {day(0)}}, true},
		"starts the next day":            {url.Values{"starts_at": {day(1)}}, false},
		"ends the day it was created":    {url.Values{"ends_at": {day(0)}}, true},
		"ends the day before":            {url.Values{"ends_at": {day(-1)}}, false},
		"window of that day alone":       {url.Values{"starts_at": {day(0)}, "ends_at": {day(0)}}, true},
		"starts at its creation second":  {url.Values{"starts_at": {instant(0)}}, true},
		"starts a second after creation": {url.Values{"starts_at": {instant(time.Second)}}, false},
		"ends a second after creation":   {url.Values{"ends_at": {instant(time.Second)}}, true},
		"ends a second before creation":  {url.Values{"ends_at": {instant(-time.Second)}}, false},
	} {
		assert.Equal(t, tc.want, found(tc.window), name)
	}

	for _, param := range []string{"starts_at", "ends_at"} {
		status, body, err := apiClient.GetListRaw(receivingOrdersPath, url.Values{param: {"06/10/2026"}})
		require.NoError(t, err)
		require.Equal(t, 400, status, "%s: %s", param, string(body))
		errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
		assert.Equal(t, param, errObj["param"])
	}
}
