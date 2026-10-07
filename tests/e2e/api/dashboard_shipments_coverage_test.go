//go:build e2e

package api_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shipment and shipping-case routes the dashboard's shipping pages call, beyond what the other
// shipment suites cover: idempotent replays, tenant isolation on every route, the portal actor and a
// read-only role against writes, the admin-only overrides, validation, and rate-shop scoping and terms.
//
// Tests build their own orders and carriers, so the suite runs again on the same stack.

// --- fixtures ---

// A packed shipment of two pairs for the seed customer on a carrier without labels.
func dashShipmentsPacked(t *testing.T) (orderID, pickID, shipmentID string) {
	t.Helper()
	return packedOrder(t, parityOrderBody(t, SeedCustomerAccountID))
}

// A packed, weighed shipment on the Shippo stub's carrier, so shipping it buys labels.
func dashShipmentsLabelled(t *testing.T) (orderID, shipmentID string) {
	t.Helper()
	body := parityOrderBody(t, SeedCustomerAccountID)
	body["carrier_id"] = SeedTransitCarrierID
	body["service_level_id"] = SeedTransitGroundServiceLevelID
	body["ship_to_address_id"] = transitAddress(t, zipStubLabels)
	orderID, _, shipmentID = packedOrder(t, body)
	weighShipmentCases(t, shipmentID, "5")
	return orderID, shipmentID
}

type dashShipmentsCall struct {
	name   string
	method string
	path   string
	body   any
}

func dashShipmentsDo(t *testing.T, c *Client, call dashShipmentsCall) (int, []byte) {
	t.Helper()
	key := ""
	if call.method == http.MethodPost || call.method == http.MethodPatch {
		key = newIdempotencyKey()
	}
	status, body, err := c.Do(call.method, call.path, call.body, key)
	require.NoError(t, err)
	return status, body
}

// Every write the shipment detail page can make against one shipment and its case.
func dashShipmentsWrites(shipmentID, caseID string) []dashShipmentsCall {
	sp := shipmentsPath + "/" + shipmentID
	cp := shippingCasesPath + "/" + caseID
	return []dashShipmentsCall{
		{"update shipment", http.MethodPatch, sp, map[string]any{"note": "must not land"}},
		{"ship", http.MethodPost, sp + "/actions/ship", map[string]any{"email_customer": false}},
		{"void", http.MethodPost, sp + "/actions/void", nil},
		{"delete shipment", http.MethodDelete, sp, nil},
		{"admin shipment tracking", http.MethodPost, sp + adminUpdateTrackingAction, map[string]any{"master_tracking_number": "1Z-MUST-NOT-LAND"}},
		{"update case", http.MethodPatch, cp, map[string]any{"tracking_number": "1Z-MUST-NOT-LAND"}},
		{"admin case tracking", http.MethodPost, cp + adminUpdateTrackingAction, map[string]any{"tracking_number": "1Z-MUST-NOT-LAND"}},
	}
}

// Fails unless the shipment is still packed and untouched by any of dashShipmentsWrites.
func dashShipmentsAssertUntouched(t *testing.T, shipmentID string) {
	t.Helper()
	shipment := readShipment(t, shipmentID, "shipping_cases")
	assert.Equal(t, "packed", jsonField(shipment, "status"))
	assert.Nil(t, shipment["note"])
	assert.Nil(t, shipment["master_tracking_number"])
	for _, raw := range jsonListData(shipment, "shipping_cases") {
		assert.Nil(t, raw.(map[string]any)["tracking_number"], "case tracking: %v", raw)
	}
}

func dashShipmentsAction(t *testing.T, shipmentID, action string, body any, key string) *Response {
	t.Helper()
	resp, err := apiClient.PostFull(shipmentsPath+"/"+shipmentID+"/actions/"+action, body, key)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "%s must not 5xx: %s", action, string(resp.Body))
	return resp
}

// A replay is served from the stored response: flagged, same status, same document.
func dashShipmentsAssertReplay(t *testing.T, first, replay *Response) {
	t.Helper()
	assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))
	assert.Equal(t, first.StatusCode, replay.StatusCode)
	var want, got any
	require.NoError(t, json.Unmarshal(first.Body, &want))
	require.NoError(t, json.Unmarshal(replay.Body, &got))
	assert.Equal(t, want, got, "the replay returns the first response")
}

func dashShipmentsFreightIDs(t *testing.T, shipmentID string) (carrierID, serviceLevelID string) {
	t.Helper()
	freight := jsonObject(readShipment(t, shipmentID, "freight"), "freight")
	require.NotNil(t, freight)
	return jsonField(jsonObject(freight, "carrier"), "id"), jsonField(jsonObject(freight, "service_level"), "id")
}

// Checks how many of the order's two pairs have been issued, the rest staying reserved. Issued goods
// move from open to closed as allocation catches up, so only the reserved/issued split is stable.
func dashShipmentsAssertIssued(t *testing.T, orderID string, issued float64, msg string) {
	t.Helper()
	var gotIssued, gotReserved float64
	for status, value := range orderIssueTotals(t, orderID) {
		if status == "reserved" {
			gotReserved += value
		} else {
			gotIssued += value
		}
	}
	assert.InDelta(t, issued, gotIssued, 0.0001, "issued: %s", msg)
	assert.InDelta(t, 2-issued, gotReserved, 0.0001, "reserved: %s", msg)
}

func dashShipmentsFloat(t *testing.T, s string) float64 {
	t.Helper()
	var f float64
	require.NoError(t, json.Unmarshal([]byte(s), &f), "decimal %q", s)
	return f
}

// --- list ---

// The search box matches more than the shipment number: tracking, note, the order's number and PO.
func TestDashShipments_ListSearchMatchesTrackingNoteOrderAndPO(t *testing.T) {
	t.Parallel()

	po := searchToken("DASHPO")
	body := parityOrderBody(t, SeedCustomerAccountID)
	body["customer_purchase_order_number"] = po
	_, _, shipmentID := packedOrder(t, body)

	tracking, note := searchToken("DASHTRK"), searchToken("DASHNOTE")
	status, resp, err := apiClient.Patch(shipmentsPath+"/"+shipmentID,
		map[string]any{"master_tracking_number": tracking, "note": note}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)

	only := []string{shipmentID}
	assert.Equal(t, only, shipmentIDsFiltered(t, url.Values{"q": {tracking}}), "master tracking number")
	assert.Equal(t, only, shipmentIDsFiltered(t, url.Values{"q": {note}}), "note")
	assert.Equal(t, only, shipmentIDsFiltered(t, url.Values{"q": {po}}), "customer PO")

	order := jsonObject(jsonObject(readShipment(t, shipmentID, "related.sales_order"), "related"), "sales_order")
	assert.Contains(t, shipmentIDsFiltered(t, url.Values{"q": {jsonField(order, "number")}}), shipmentID, "order number")
}

func TestDashShipments_ListRejectsAnUnknownStatus(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(shipmentsPath, url.Values{"status": {"in_transit"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assert.Equal(t, "status", errObj["param"])
}

// A date the server cannot read must be refused: ignoring it returns every shipment as if unfiltered.
func TestDashShipments_ListRejectsMalformedDates(t *testing.T) {
	t.Parallel()

	for _, param := range []string{"starts_at", "ends_at"} {
		for _, value := range []string{"yesterday", "2026-13-45"} {
			status, body, err := apiClient.GetListRaw(shipmentsPath, url.Values{param: {value}, "limit": {"1"}})
			require.NoError(t, err)
			if assert.Equal(t, 400, status, "%s=%s must be refused, not ignored: %s", param, value, string(body)) {
				assert.Equal(t, param, requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")["param"])
			}
		}
	}
}

func TestDashShipments_ListIsRefusedToThePortalAndHidesShipmentsFromAnotherTenant(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	number := jsonField(readShipment(t, shipmentID), "number")

	status, body, err := getCustomerPortalClient().GetListRaw(shipmentsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	for _, params := range []url.Values{{"q": {number}}, {"customer_ids": {SeedCustomerAccountID}}} {
		status, body, err := getTenantBClient().GetListRaw(shipmentsPath, params)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		for _, raw := range jsonArray(parseJSON(body), "data") {
			assert.NotEqual(t, shipmentID, jsonField(raw.(map[string]any), "id"), "tenant B listed tenant A's shipment with %v", params)
		}
	}
}

// --- tenancy and actors ---

func TestDashShipments_AnotherTenantGets404OnEveryRoute(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	caseID := shipmentCaseIDs(t, shipmentID)[0]

	calls := append(dashShipmentsWrites(shipmentID, caseID),
		dashShipmentsCall{"retrieve shipment", http.MethodGet, shipmentsPath + "/" + shipmentID, nil},
		dashShipmentsCall{"case label", http.MethodGet, shippingCasesPath + "/" + caseID + "/label", nil},
	)
	for _, call := range calls {
		status, body := dashShipmentsDo(t, getTenantBClient(), call)
		assert.Equal(t, 404, status, "tenant B %s: %s", call.name, string(body))
	}

	dashShipmentsAssertUntouched(t, shipmentID)
}

// Proves another tenant cannot tell a deleted shipment's id from one that never existed.
func TestDashShipments_AnotherTenantCannotTellADeletedShipmentExisted(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	status, body, err := apiClient.Delete(shipmentsPath + "/" + shipmentID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = getTenantBClient().Delete(shipmentsPath + "/" + shipmentID)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a deleted shipment of another tenant is not found, not gone: %s", string(body))
}

func TestDashShipments_PortalReadsItsOwnShipmentButWritesNothing(t *testing.T) {
	t.Parallel()

	portal := getCustomerPortalClient()
	_, _, shipmentID := dashShipmentsPacked(t)
	caseID := shipmentCaseIDs(t, shipmentID)[0]

	status, body, err := portal.GetListRaw(shipmentsPath+"/"+shipmentID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, shipmentID, jsonField(parseJSON(body), "id"), "the buyer reads its own shipment")

	calls := append(dashShipmentsWrites(shipmentID, caseID),
		dashShipmentsCall{"case label", http.MethodGet, shippingCasesPath + "/" + caseID + "/label", nil})
	for _, call := range calls {
		status, body := dashShipmentsDo(t, portal, call)
		assert.Equal(t, 403, status, "portal %s: %s", call.name, string(body))
	}

	dashShipmentsAssertUntouched(t, shipmentID)
}

func TestDashShipments_PortalCannotReadAnotherCustomersShipment(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := packedOrder(t, parityOrderBody(t, setupOrderCustomer(t)))

	status, body, err := getCustomerPortalClient().GetListRaw(shipmentsPath+"/"+shipmentID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "another buyer's shipment is not found: %s", string(body))
}

// A merchant targeting its customer's account reads inside that account, where its own shipments are
// not found, and cannot ship as that account.
func TestDashShipments_TargetingACustomerAccountDoesNotReachTheSellersShipments(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	targeting := apiClient.WithAccountID(SeedCustomerAccountID)

	status, body, err := targeting.GetListRaw(shipmentsPath+"/"+shipmentID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "%s", string(body))

	status, body = dashShipmentsDo(t, targeting, dashShipmentsCall{"ship", http.MethodPost,
		shipmentsPath + "/" + shipmentID + "/actions/ship", map[string]any{"email_customer": false}})
	assert.Equal(t, 403, status, "%s", string(body))

	dashShipmentsAssertUntouched(t, shipmentID)
}

func TestDashShipments_ReadOnlyRoleReadsButCannotWrite(t *testing.T) {
	t.Parallel()

	readOnly := customRoleClient(t, "shipments:read")
	_, _, shipmentID := dashShipmentsPacked(t)
	caseID := shipmentCaseIDs(t, shipmentID)[0]

	for _, path := range []string{shipmentsPath + "/" + shipmentID, shippingCasesPath + "/" + caseID + "/label"} {
		status, body, err := readOnly.GetListRaw(path, nil)
		require.NoError(t, err)
		assert.Equal(t, 200, status, "GET %s: %s", path, string(body))
	}

	for _, call := range dashShipmentsWrites(shipmentID, caseID) {
		status, body := dashShipmentsDo(t, readOnly, call)
		assert.Equal(t, 403, status, "read-only %s: %s", call.name, string(body))
	}

	dashShipmentsAssertUntouched(t, shipmentID)
}

// The overrides are admin-only: holding shipments:update is not enough.
func TestDashShipments_AdminTrackingOverridesRefuseANonAdminRole(t *testing.T) {
	t.Parallel()

	updater := customRoleClient(t, "shipments:read", "shipments:update")
	_, _, shipmentID := dashShipmentsPacked(t)
	caseID := shipmentCaseIDs(t, shipmentID)[0]
	status, body := shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)

	for _, call := range []dashShipmentsCall{
		{"admin shipment tracking", http.MethodPost, shipmentsPath + "/" + shipmentID + adminUpdateTrackingAction, map[string]any{"master_tracking_number": "1Z-NOT-ADMIN"}},
		{"admin case tracking", http.MethodPost, shippingCasesPath + "/" + caseID + adminUpdateTrackingAction, map[string]any{"tracking_number": "1Z-NOT-ADMIN"}},
	} {
		status, body := dashShipmentsDo(t, updater, call)
		assert.Equal(t, 403, status, "%s: %s", call.name, string(body))
	}

	// The same role may still edit the shipped shipment's tracking the ordinary way.
	status, body = dashShipmentsDo(t, updater, dashShipmentsCall{"update", http.MethodPatch,
		shipmentsPath + "/" + shipmentID, map[string]any{"master_tracking_number": "1Z-ORDINARY"}})
	requireStatus(t, 200, status, body)

	shipment := readShipment(t, shipmentID, "shipping_cases")
	assert.Equal(t, "1Z-ORDINARY", jsonField(shipment, "master_tracking_number"))
	assert.Nil(t, jsonListData(shipment, "shipping_cases")[0].(map[string]any)["tracking_number"])
}

func TestDashShipments_UnknownIDsAre404(t *testing.T) {
	t.Parallel()

	for _, call := range []dashShipmentsCall{
		{"update shipment", http.MethodPatch, shipmentsPath + "/sh_doesnotexist0000", map[string]any{"note": "x"}},
		{"admin shipment tracking", http.MethodPost, shipmentsPath + "/sh_doesnotexist0000" + adminUpdateTrackingAction, map[string]any{"master_tracking_number": "x"}},
		{"update case", http.MethodPatch, shippingCasesPath + "/shcs_doesnotexist000", map[string]any{"tracking_number": "x"}},
		{"admin case tracking", http.MethodPost, shippingCasesPath + "/shcs_doesnotexist000" + adminUpdateTrackingAction, map[string]any{"tracking_number": "x"}},
		{"case label", http.MethodGet, shippingCasesPath + "/shcs_doesnotexist000/label", nil},
	} {
		status, body := dashShipmentsDo(t, apiClient, call)
		if assert.Equal(t, 404, status, "%s: %s", call.name, string(body)) {
			requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
		}
	}
}

// --- update ---

// The detail page's edit form sends the note, tracking and routing in one request.
func TestDashShipments_UpdateSetsEveryDashboardFieldTogetherThenClears(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	number := jsonField(readShipment(t, shipmentID), "number")
	path := shipmentsPath + "/" + shipmentID + "?include=freight"

	status, body, err := apiClient.Patch(path, map[string]any{
		"note":                   "Leave at dock 4",
		"master_tracking_number": "1Z-DASH-ALL",
		"carrier_id":             SeedTransitCarrierID,
		"service_level_id":       SeedTransitGroundServiceLevelID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "Leave at dock 4", jsonField(got, "note"))
	assert.Equal(t, "1Z-DASH-ALL", jsonField(got, "master_tracking_number"))
	assert.Equal(t, number, jsonField(got, "number"), "an unsent number is kept")
	freight := jsonObject(got, "freight")
	assert.Equal(t, SeedTransitCarrierID, jsonField(jsonObject(freight, "carrier"), "id"))
	assert.Equal(t, SeedTransitGroundServiceLevelID, jsonField(jsonObject(freight, "service_level"), "id"))
	for _, caseID := range shipmentCaseIDs(t, shipmentID) {
		assert.Equal(t, SeedTransitCarrierID, shippingCaseCarrierID(t, caseID), "the case follows the carrier")
	}

	status, body, err = apiClient.Patch(path, map[string]any{"master_tracking_number": nil, "service_level_id": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got = parseJSON(body)
	assert.Nil(t, got["master_tracking_number"], "null clears the tracking number")
	assert.Nil(t, jsonObject(got, "freight")["service_level"], "null clears the service level")
	assert.Equal(t, "Leave at dock 4", jsonField(got, "note"), "an unsent note is kept")
	assert.Equal(t, SeedTransitCarrierID, jsonField(jsonObject(jsonObject(got, "freight"), "carrier"), "id"), "an unsent carrier is kept")
}

func TestDashShipments_UpdateValidation(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	path := shipmentsPath + "/" + shipmentID
	long := strings.Repeat("x", 256)

	for _, tc := range []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"blank number", map[string]any{"number": ""}, "number"},
		{"number over 255", map[string]any{"number": long}, "number"},
		{"null note", map[string]any{"note": nil}, "note"},
		{"tracking over 255", map[string]any{"master_tracking_number": long}, "master_tracking_number"},
		{"null carrier", map[string]any{"carrier_id": nil}, "carrier_id"},
		{"non-string note", map[string]any{"note": 7}, "note"},
	} {
		status, body, err := apiClient.Patch(path, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		if assert.Equal(t, 400, status, "%s: %s", tc.name, string(body)) {
			assert.Equal(t, tc.param, requireErrorResponse(t, body, "", "invalid_request_error")["param"], tc.name)
		}
	}

	dashShipmentsAssertUntouched(t, shipmentID)
}

// A carrier or service level the account does not own must be refused, not stored on the shipment.
func TestDashShipments_UpdateRefusesRoutingTheAccountDoesNotOwn(t *testing.T) {
	t.Parallel()

	tenantB := getTenantBClient()
	created := func(path string, body map[string]any) string {
		t.Helper()
		status, resp, err := tenantB.Post(path, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, resp)
		return jsonField(parseJSON(resp), "id")
	}
	foreignCarrier := created(carriersPath, map[string]any{"name": uniqueName("e2e-dash-foreign-carrier"), "code": "will_call"})
	t.Cleanup(func() { _, _, _ = tenantB.Delete(carriersPath + "/" + foreignCarrier) })
	foreignServiceLevel := created(serviceLevelsPath(foreignCarrier), map[string]any{
		"name": uniqueName("e2e-dash-foreign-sl"), "code": uniqueName("e2e-dash-foreign-sl-code"),
	})

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"unknown carrier", map[string]any{"carrier_id": "cr_doesnotexist0000"}},
		{"another tenant's carrier", map[string]any{"carrier_id": foreignCarrier}},
		{"unknown service level", map[string]any{"service_level_id": "crop_doesnotexist000"}},
		{"another tenant's service level", map[string]any{"service_level_id": foreignServiceLevel}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, shipmentID := dashShipmentsPacked(t)
			carrierBefore, serviceLevelBefore := dashShipmentsFreightIDs(t, shipmentID)

			status, body, err := apiClient.Patch(shipmentsPath+"/"+shipmentID, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			assert.Equal(t, 404, status, "%s", string(body))

			carrierAfter, serviceLevelAfter := dashShipmentsFreightIDs(t, shipmentID)
			assert.Equal(t, carrierBefore, carrierAfter, "the carrier is unchanged")
			assert.Equal(t, serviceLevelBefore, serviceLevelAfter, "the service level is unchanged")
		})
	}
}

// A replayed PATCH returns the stored response and does not write again over a later edit.
func TestDashShipments_UpdateReplayDoesNotReapply(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	path := shipmentsPath + "/" + shipmentID
	key := newIdempotencyKey()

	first, err := apiClient.PatchFull(path, map[string]any{"note": "first edit"}, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)

	status, body, err := apiClient.Patch(path, map[string]any{"note": "second edit"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	replay, err := apiClient.PatchFull(path, map[string]any{"note": "first edit"}, key)
	require.NoError(t, err)
	dashShipmentsAssertReplay(t, first, replay)
	assert.Equal(t, "second edit", jsonField(readShipment(t, shipmentID), "note"), "the replay did not write")

	status, body, err = apiClient.Patch(path, map[string]any{"note": "different body"}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")
}

// --- ship ---

// A replayed ship returns the stored response without buying, issuing or invoicing again, even after
// the shipment has been voided back to packed.
func TestDashShipments_ShipReplayShipsOnce(t *testing.T) {
	t.Parallel()

	orderID, shipmentID := dashShipmentsLabelled(t)
	number := jsonField(readShipment(t, shipmentID), "number")
	dashShipmentsAssertIssued(t, orderID, 0, "packing only reserves the order's two pairs")

	key := newIdempotencyKey()
	shipBody := map[string]any{"email_customer": false}
	first := dashShipmentsAction(t, shipmentID, "ship", shipBody, key)
	requireStatus(t, 200, first.StatusCode, first.Body)
	assert.Equal(t, stubPurchaseTracking(1, shipmentID), jsonField(parseJSON(first.Body), "master_tracking_number"))
	dashShipmentsAssertIssued(t, orderID, 2, "shipping issues the reserved pairs")

	replay := dashShipmentsAction(t, shipmentID, "ship", shipBody, key)
	dashShipmentsAssertReplay(t, first, replay)
	dashShipmentsAssertIssued(t, orderID, 2, "the replay issues nothing more")
	assert.Equal(t, 1, countInvoicesNumbered(t, number), "the replay raises no second invoice")

	different := dashShipmentsAction(t, shipmentID, "ship", map[string]any{"email_customer": true}, key)
	requireStatus(t, 400, different.StatusCode, different.Body)
	requireErrorResponse(t, different.Body, "validation_failed", "idempotency_error")

	voided := dashShipmentsAction(t, shipmentID, "void", nil, newIdempotencyKey())
	requireStatus(t, 200, voided.StatusCode, voided.Body)

	// Replaying the ship now must not re-ship what the void undid.
	replay = dashShipmentsAction(t, shipmentID, "ship", shipBody, key)
	dashShipmentsAssertReplay(t, first, replay)
	assert.Equal(t, "packed", jsonField(readShipment(t, shipmentID), "status"))
	assert.Equal(t, 0, countInvoicesNumbered(t, number))
	dashShipmentsAssertIssued(t, orderID, 0, "the replay issues nothing after the void")

	// The next real ship is only the shipment's second purchase, so no replay reached the carrier.
	again := dashShipmentsAction(t, shipmentID, "ship", shipBody, newIdempotencyKey())
	requireStatus(t, 200, again.StatusCode, again.Body)
	assert.Equal(t, stubPurchaseTracking(2, shipmentID), jsonField(parseJSON(again.Body), "master_tracking_number"))
}

func TestDashShipments_ShipRejectsANonBooleanEmailFlag(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	resp := dashShipmentsAction(t, shipmentID, "ship", map[string]any{"email_customer": "yes"}, newIdempotencyKey())
	requireStatus(t, 400, resp.StatusCode, resp.Body)
	assert.Equal(t, "email_customer", requireErrorResponse(t, resp.Body, "invalid_format", "invalid_request_error")["param"])
	assert.Equal(t, "packed", jsonField(readShipment(t, shipmentID), "status"))
}

// --- void ---

// A replayed void returns the stored response without refunding or unwinding the shipment's next
// dispatch; a fresh void of a voided shipment conflicts.
func TestDashShipments_VoidReplayUnwindsOnce(t *testing.T) {
	t.Parallel()

	orderID, shipmentID := dashShipmentsLabelled(t)
	number := jsonField(readShipment(t, shipmentID), "number")
	shipBody := map[string]any{"email_customer": false}

	shipped := dashShipmentsAction(t, shipmentID, "ship", shipBody, newIdempotencyKey())
	requireStatus(t, 200, shipped.StatusCode, shipped.Body)

	voidKey := newIdempotencyKey()
	first := dashShipmentsAction(t, shipmentID, "void", nil, voidKey)
	requireStatus(t, 200, first.StatusCode, first.Body)
	assert.Equal(t, "packed", jsonField(parseJSON(first.Body), "status"))
	for _, sc := range shipmentCases(t, shipmentID) {
		assert.Nil(t, sc["shippo_transaction_id"], "void refunds and drops the label: %v", sc)
	}

	reshipped := dashShipmentsAction(t, shipmentID, "ship", shipBody, newIdempotencyKey())
	requireStatus(t, 200, reshipped.StatusCode, reshipped.Body)
	secondPurchase := stubPurchaseTracking(2, shipmentID)
	require.Equal(t, secondPurchase, jsonField(parseJSON(reshipped.Body), "master_tracking_number"))

	replay := dashShipmentsAction(t, shipmentID, "void", nil, voidKey)
	dashShipmentsAssertReplay(t, first, replay)

	after := readShipment(t, shipmentID)
	assert.Equal(t, "shipped", jsonField(after, "status"), "the replay did not void the second dispatch")
	assert.Equal(t, secondPurchase, jsonField(after, "master_tracking_number"))
	for _, sc := range shipmentCases(t, shipmentID) {
		assert.True(t, strings.HasPrefix(jsonField(sc, "shippo_transaction_id"), "stub_txn_2_"+shipmentID),
			"the second purchase's label was not refunded: %v", sc)
	}
	assert.Equal(t, 1, countInvoicesNumbered(t, number), "the second dispatch's invoice stands")
	dashShipmentsAssertIssued(t, orderID, 2, "the replay put nothing back")

	again := dashShipmentsAction(t, shipmentID, "void", nil, newIdempotencyKey())
	requireStatus(t, 200, again.StatusCode, again.Body)
	twice := dashShipmentsAction(t, shipmentID, "void", nil, newIdempotencyKey())
	requireStatus(t, 409, twice.StatusCode, twice.Body)
	requireErrorResponse(t, twice.Body, "resource_conflict", "invalid_request_error")
}

// --- delete ---

// A repeated delete is gone, and changes nothing the first delete reopened.
func TestDashShipments_DeleteTwiceIsGone(t *testing.T) {
	t.Parallel()

	_, pickID, shipmentID := dashShipmentsPacked(t)
	path := shipmentsPath + "/" + shipmentID

	status, body, err := apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	reopened := readPickLineQuantities(t, pickID)

	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)
	requireErrorResponse(t, body, "resource_gone", "invalid_request_error")

	status, body, err = apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	assert.Equal(t, reopened, readPickLineQuantities(t, pickID), "the second delete left the pick alone")
	assert.Empty(t, jsonField(firstPickLine(t, pickID), "packed_at"))
}

// --- admin tracking overrides ---

func TestDashShipments_AdminTrackingValidatesRouting(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	status, body := shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	carrierBefore, serviceLevelBefore := dashShipmentsFreightIDs(t, shipmentID)
	path := shipmentsPath + "/" + shipmentID + adminUpdateTrackingAction

	for _, tc := range []struct {
		name   string
		body   map[string]any
		status int
	}{
		{"unknown carrier", map[string]any{"carrier_id": "cr_doesnotexist0000"}, 404},
		{"unknown service level", map[string]any{"service_level_id": "crop_doesnotexist000"}, 404},
		{"tracking over 255", map[string]any{"master_tracking_number": strings.Repeat("x", 256)}, 400},
		{"null carrier", map[string]any{"carrier_id": nil}, 400},
	} {
		status, body, err := apiClient.Post(path, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, tc.status, status, "%s: %s", tc.name, string(body))
	}

	carrierAfter, serviceLevelAfter := dashShipmentsFreightIDs(t, shipmentID)
	assert.Equal(t, carrierBefore, carrierAfter)
	assert.Equal(t, serviceLevelBefore, serviceLevelAfter)
	assert.Equal(t, "shipped", jsonField(readShipment(t, shipmentID), "status"))
}

// A replayed override returns the stored response and does not write over a later correction.
func TestDashShipments_AdminTrackingReplayDoesNotReapply(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	status, body := shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)

	for _, tc := range []struct {
		path, field string
	}{
		{shipmentsPath + "/" + shipmentID, "master_tracking_number"},
		{shippingCasesPath + "/" + shipmentCaseIDs(t, shipmentID)[0], "tracking_number"},
	} {
		path := tc.path + adminUpdateTrackingAction
		key := newIdempotencyKey()
		first, err := apiClient.PostFull(path, map[string]any{tc.field: "1Z-ADMIN-FIRST"}, key)
		require.NoError(t, err)
		requireStatus(t, 200, first.StatusCode, first.Body)

		status, resp, err := apiClient.Post(path, map[string]any{tc.field: "1Z-ADMIN-SECOND"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)

		replay, err := apiClient.PostFull(path, map[string]any{tc.field: "1Z-ADMIN-FIRST"}, key)
		require.NoError(t, err)
		dashShipmentsAssertReplay(t, first, replay)

		status, resp, err = apiClient.GetListRaw(tc.path, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		assert.Equal(t, "1Z-ADMIN-SECOND", jsonField(parseJSON(resp), tc.field), "%s: the replay did not write", tc.path)
	}
}

// --- shipping cases ---

// The cases card edits the freight cost and weight; units relabel without converting the number.
func TestDashShipments_CaseUpdateFreightAmountWeightAndUnits(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	path := shippingCasesPath + "/" + shipmentCaseIDs(t, shipmentID)[0] + "?include=freight_amount.unit&include=freight_weight.unit"
	patch := func(body map[string]any) map[string]any {
		t.Helper()
		status, resp, err := apiClient.Patch(path, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		return parseJSON(resp)
	}
	measure := func(got map[string]any, field string) (float64, string) {
		t.Helper()
		q := jsonObject(got, field)
		require.NotNil(t, q, "%s expands: %v", field, got)
		return dashShipmentsFloat(t, jsonField(q, "value")), jsonField(jsonObject(q, "unit"), "id")
	}

	got := patch(map[string]any{"freight_amount_value": "7.25", "freight_weight_value": "3.5", "freight_weight_unit_id": "gram"})
	value, unit := measure(got, "freight_amount")
	assert.InDelta(t, 7.25, value, 0.0001)
	assert.Equal(t, "dollar", unit)
	value, unit = measure(got, "freight_weight")
	assert.InDelta(t, 3.5, value, 0.0001)
	assert.Equal(t, "gram", unit)

	got = patch(map[string]any{"freight_weight_unit_id": "pound", "freight_amount_unit_id": "dollar"})
	value, unit = measure(got, "freight_weight")
	assert.InDelta(t, 3.5, value, 0.0001, "a unit change keeps the number")
	assert.Equal(t, "pound", unit)
	value, _ = measure(got, "freight_amount")
	assert.InDelta(t, 7.25, value, 0.0001)
	assert.Nil(t, got["tracking_number"], "an unsent tracking number stays blank")
}

func TestDashShipments_CaseUpdateValidation(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	caseID := shipmentCaseIDs(t, shipmentID)[0]
	path := shippingCasesPath + "/" + caseID

	for _, tc := range []struct {
		name   string
		body   map[string]any
		status int
	}{
		{"non-numeric weight", map[string]any{"freight_weight_value": "heavy"}, 400},
		{"non-numeric amount", map[string]any{"freight_amount_value": "lots"}, 400},
		{"null weight", map[string]any{"freight_weight_value": nil}, 400},
		{"tracking over 255", map[string]any{"tracking_number": strings.Repeat("x", 256)}, 400},
		{"unknown weight unit", map[string]any{"freight_weight_unit_id": "un_doesnotexist0000"}, 404},
	} {
		status, body, err := apiClient.Patch(path, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, tc.status, status, "%s: %s", tc.name, string(body))
	}

	value, unit := caseFreightWeight(t, caseID)
	assert.InDelta(t, 0.0, value, 0.0001, "the case keeps its weight")
	assert.Equal(t, "pound", unit)
}

func TestDashShipments_CaseUpdateReplayDoesNotReapply(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	path := shippingCasesPath + "/" + shipmentCaseIDs(t, shipmentID)[0]
	key := newIdempotencyKey()
	body := map[string]any{"tracking_number": "1Z-CASE-FIRST", "freight_weight_value": "4"}

	first, err := apiClient.PatchFull(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)

	status, resp, err := apiClient.Patch(path, map[string]any{"tracking_number": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)

	replay, err := apiClient.PatchFull(path, body, key)
	require.NoError(t, err)
	dashShipmentsAssertReplay(t, first, replay)

	status, resp, err = apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	assert.Nil(t, parseJSON(resp)["tracking_number"], "the replay did not restore the cleared tracking number")
}

// The page opens the label in a new tab; a case with no stored label answers with a null link.
func TestDashShipments_LabelOfACaseWithoutOneIsNull(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := dashShipmentsPacked(t)
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+shipmentCaseIDs(t, shipmentID)[0]+"/label", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "shipping_case_label_url", jsonField(got, "object"))
	assert.Contains(t, got, "url")
	assert.Nil(t, got["url"])
}

// --- rate shop ---

func TestDashShipments_RateShopValidation(t *testing.T) {
	t.Parallel()

	to := rateShopRequestBody()["to_address"]
	parcel := map[string]any{"weight": 5.0, "length": 12.0, "width": 8.0, "height": 6.0}
	// A missing destination is reported against its first required field.
	for _, tc := range []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"no destination", map[string]any{"parcels": []any{parcel}}, "validation_failed", "to_address.name"},
		{"no parcels", map[string]any{"to_address": to}, "missing_field", "parcels"},
		{"empty parcels", map[string]any{"to_address": to, "parcels": []any{}}, "invalid_format", "parcels"},
	} {
		status, body, err := apiClient.Post(rateShopPath, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		if assert.Equal(t, 400, status, "%s: %s", tc.name, string(body)) {
			errObj := requireErrorResponse(t, body, tc.code, "invalid_request_error")
			assert.Equal(t, tc.param, errObj["param"], tc.name)
		}
	}

	unknown := rateShopRequestBody()
	unknown["customer_id"] = "ac_doesnotexist00000"
	status, body, err := apiClient.Post(rateShopPath, unknown, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "an unknown customer: %s", string(body))
}

// A carrier on the seed account with one service level per visibility, for portal rate shopping.
func dashShipmentsPortalCarrier(t *testing.T, visibility string) (visibleLevelID, hiddenLevelID string) {
	t.Helper()
	carrier := createAndCleanup(t, carriersPath, map[string]any{
		"name": uniqueName("e2e-dash-portal-carrier"), "code": "will_call", "customer_portal_visibility": visibility,
	})
	level := func(levelVisibility string) string {
		t.Helper()
		status, body, err := apiClient.Post(serviceLevelsPath(jsonField(carrier, "id")), map[string]any{
			"name": uniqueName("e2e-dash-portal-sl"), "code": uniqueName("e2e-dash-portal-sl-code"),
			"customer_portal_visibility": levelVisibility,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		return jsonField(parseJSON(body), "id")
	}
	return level("visible"), level("hidden")
}

func dashShipmentsRateShop(t *testing.T, c *Client, customerID string) (int, map[string]any) {
	t.Helper()
	body := rateShopRequestBody()
	if customerID != "" {
		body["customer_id"] = customerID
	}
	status, resp, err := c.Post(rateShopPath, body, newIdempotencyKey())
	require.NoError(t, err)
	got := parseJSON(resp)
	if status != 200 {
		t.Logf("rate shop %d: %s", status, string(resp))
	}
	return status, got
}

func dashShipmentsRatedLevels(result map[string]any) []string {
	var ids []string
	for _, raw := range jsonListData(result, "options") {
		ids = append(ids, jsonField(jsonObject(raw.(map[string]any), "service_level"), "id"))
	}
	return ids
}

// The portal rates only for its own account and sees only what is enabled for the portal; another
// tenant cannot rate for this account's customer.
func TestDashShipments_RateShopCustomerScoping(t *testing.T) {
	t.Parallel()

	portalLevel, hiddenLevel := dashShipmentsPortalCarrier(t, "visible")
	hiddenCarrierLevel, _ := dashShipmentsPortalCarrier(t, "hidden")

	status, merchant := dashShipmentsRateShop(t, apiClient, SeedCustomerAccountID)
	requireStatus(t, 200, status, nil)
	assert.Subset(t, dashShipmentsRatedLevels(merchant), []string{portalLevel, hiddenLevel, hiddenCarrierLevel},
		"the merchant rates every service level")

	portal := getCustomerPortalClient()
	status, rated := dashShipmentsRateShop(t, portal, SeedCustomerAccountID)
	requireStatus(t, 200, status, nil)
	levels := dashShipmentsRatedLevels(rated)
	assert.Contains(t, levels, portalLevel, "a portal service level on a portal carrier is offered")
	assert.NotContains(t, levels, hiddenLevel, "a hidden service level is not offered to the portal")
	assert.NotContains(t, levels, hiddenCarrierLevel, "a hidden carrier is not offered to the portal")

	status, _ = dashShipmentsRateShop(t, portal, "")
	assert.Equal(t, 403, status, "the portal must name its own account")
	status, _ = dashShipmentsRateShop(t, portal, setupOrderCustomer(t))
	assert.Equal(t, 403, status, "the portal cannot rate for another buyer")

	status, _ = dashShipmentsRateShop(t, getTenantBClient(), SeedCustomerAccountID)
	assert.Equal(t, 404, status, "another tenant's customer is not found")
}

// customers:read covers rating inside a customer's account, not in the merchant's own.
func TestDashShipments_RateShopPermissionFollowsTheTargetAccount(t *testing.T) {
	t.Parallel()

	customersOnly := customRoleClient(t, "customers:read")
	status, _ := dashShipmentsRateShop(t, customersOnly, "")
	assert.Equal(t, 403, status, "customers:read alone does not rate in the merchant's own account")
	status, _ = dashShipmentsRateShop(t, customersOnly.WithAccountID(SeedCustomerAccountID), "")
	assert.Equal(t, 200, status, "customers:read rates inside the customer's account")
}

// A customer defaulting to a new shipping term built from term, in a new group that sets no freight
// policy of its own.
func dashShipmentsTermCustomer(t *testing.T, term map[string]any) string {
	t.Helper()
	term["name"] = uniqueName("e2e-dash-term")
	created := createAndCleanup(t, shippingTermsPath, term)
	body := validCustomerBody(uniqueName("e2e-dash-term-cust"))
	body["default_shipping_term_id"] = jsonField(created, "id")
	body["customer_type_group_id"] = leadTimeAccountGroup(t, "e2e-dash-term-grp", nil)
	return jsonField(createAndCleanup(t, customersPath, body), "id")
}

func dashShipmentsDollars(value string) map[string]any {
	return map[string]any{"value": value, "unit_id": "dollar"}
}

// The checkout sends the customer, order total and product lines, and the customer's freight terms
// decide what the options cost.
func TestDashShipments_RateShopAppliesFreightTerms(t *testing.T) {
	t.Parallel()

	rateShop := func(t *testing.T, customerID string, orderTotal float64, productLineIDs ...string) map[string]any {
		t.Helper()
		body := rateShopRequestBody()
		body["order_total"] = orderTotal
		if customerID != "" {
			body["customer_id"] = customerID
		}
		if len(productLineIDs) > 0 {
			body["product_line_ids"] = productLineIDs
		}
		status, resp, err := apiClient.Post(rateShopPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		return parseJSON(resp)
	}
	ratesByLevel := func(t *testing.T, result map[string]any) map[string]float64 {
		t.Helper()
		rates := map[string]float64{}
		for _, raw := range jsonListData(result, "options") {
			opt := raw.(map[string]any)
			rates[jsonField(jsonObject(opt, "service_level"), "id")] = opt["rate"].(float64)
		}
		require.Contains(t, rates, SeedServiceLevelID, "the seeded ground option is rated")
		return rates
	}
	assertExempt := func(t *testing.T, result map[string]any) {
		t.Helper()
		assert.Equal(t, "freight_exempt", jsonField(result, "exemption_type"))
		assert.Empty(t, jsonListData(result, "options"), "an exempt order is offered no freight")
	}

	t.Run("free-freight product line", func(t *testing.T) {
		t.Parallel()
		line := createAndCleanup(t, productLinesPath, map[string]any{
			"name": uniqueName("e2e-dash-free-pdln"), "unit_group_id": SeedUnitGroupID,
			"commission_policy": "commission_applied", "freight_policy": "free_freight",
		})
		assert.Equal(t, "none", jsonField(rateShop(t, "", 10, SeedProductLineID), "exemption_type"))
		assertExempt(t, rateShop(t, "", 10, SeedProductLineID, jsonField(line, "id")))
	})

	t.Run("free-freight term", func(t *testing.T) {
		t.Parallel()
		assertExempt(t, rateShop(t, dashShipmentsTermCustomer(t, map[string]any{"type": "free_freight"}), 10))
	})

	t.Run("flat-rate term", func(t *testing.T) {
		t.Parallel()
		got := rateShop(t, dashShipmentsTermCustomer(t, map[string]any{
			"type": "flat_rate_freight", "flat_rate": dashShipmentsDollars("15"),
		}), 10)
		assert.Equal(t, "flat_rate", jsonField(got, "exemption_type"))
		assert.Equal(t, 15.0, got["flat_rate"])
		for level, rate := range ratesByLevel(t, got) {
			assert.Equal(t, 15.0, rate, "every option costs the flat rate: %s", level)
		}
	})

	t.Run("minimum order frees the listed service levels", func(t *testing.T) {
		t.Parallel()
		customerID := dashShipmentsTermCustomer(t, map[string]any{
			"type": "flat_rate_freight", "flat_rate": dashShipmentsDollars("15"),
			"minimum_order_value":             dashShipmentsDollars("100"),
			"free_shipping_service_level_ids": []string{SeedServiceLevelID},
		})

		atThreshold := rateShop(t, customerID, 100)
		assert.Equal(t, "flat_rate", jsonField(atThreshold, "exemption_type"), "the minimum must be exceeded, not met")
		assert.Equal(t, 15.0, ratesByLevel(t, atThreshold)[SeedServiceLevelID])

		above := rateShop(t, customerID, 100.01)
		assert.Equal(t, "minimum_order_met", jsonField(above, "exemption_type"))
		for level, rate := range ratesByLevel(t, above) {
			want := 15.0
			if level == SeedServiceLevelID {
				want = 0
			}
			assert.Equal(t, want, rate, "service level %s", level)
		}
	})
}
