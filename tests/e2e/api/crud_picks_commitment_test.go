//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pickIDForOrder returns the id of the pick an issued order created, reading it off
// related.pick and re-fetching the order with the include if the issue response did
// not already carry it.
func pickIDForOrder(t *testing.T, order map[string]any) string {
	t.Helper()

	pickID := jsonField(jsonObject(jsonObject(order, "related"), "pick"), "id")
	if pickID == "" {
		status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+jsonField(order, "id"),
			url.Values{"include": {"related.pick"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		pickID = jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "pick"), "id")
	}
	require.NotEmpty(t, pickID, "issuing an order creates its pick")
	return pickID
}

// Pins the commitment a pick inherits: the pick page shows when the order must ship without
// fetching the order. Issuing an order is the only way a pick comes into existence.
func TestPicks_CarryTheOrdersCommitment(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-commit", ptrInt(12), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderShipBy := shipByDate(t, order)
	require.NotEmpty(t, orderShipBy, "the order must carry a commitment for the pick to inherit one")

	pickID := pickIDForOrder(t, order)

	status, body, err := apiClient.GetListRaw(picksPath+"/"+pickID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	pick := parseJSON(body)

	// The commitment and the rules behind it come from the order, with no include required.
	assert.Equal(t, orderShipBy, shipByDate(t, pick), "the pick's ship-by matches its order's")
	// Compared against the order rather than the customer's configured 12: lead_time_days is the effective span to the ship-by date, which the ship calendar snaps back off a closed day, so the literal would only hold on issue dates whose twelfth day is an operating one.
	require.NotNil(t, commitmentOf(order)["lead_time_days"], "the order must report the lead time the pick inherits")
	assert.EqualValues(t, commitmentOf(order)["lead_time_days"], commitmentOf(pick)["lead_time_days"], "lead time rides along from the order")
	assert.Equal(t, "customer", commitmentOf(pick)["lead_time_source"],
		"a customer-level lead time reports the customer as its source")
}

// The carried subset of the commitment — every field a pick denormalizes off its order.
// The authoring history behind the date (calendar adjustment, overrides, arrival) stays on
// the order, so it is checked for absence separately.
var pickCommitmentFields = []string{
	"promised_at", "ship_by_date", "lead_time_days", "lead_time_source", "transit_days", "transit_source",
}

// A commitment fully hydrates onto the pick only when every field it can carry is set: a
// promised delivery date, a resolved ship-by, a lead-time source, and a carrier lane that
// priced. The plain ship-by test above never exercises the transit half, so a list or detail
// query that selected only the ship-by date would still pass it. This drives the full lane and
// asserts the whole subset rides across on both read paths, which are built by different queries.
func TestPicks_CarryTheOrdersFullCommitment(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-pick-full-commit", ptrInt(30), "")
	orderID := createTransitOrder(t, customerID, SeedTransitGroundServiceLevelID, zipStubNormal, promised)
	order := issueOnceWarm(t, orderID)
	oc := commitmentOf(order)

	// Precondition: the order itself carries a fully-populated commitment, so a pick that
	// merely omits transit cannot pass by matching an order that never had it either.
	require.Equal(t, "3", jsonField(oc, "transit_days"), "precondition: the lane priced")
	require.Equal(t, "carrier_lane", jsonField(oc, "transit_source"))
	require.Equal(t, "manual", jsonField(oc, "lead_time_source"), "a promised date beats the standing rule")
	for _, f := range pickCommitmentFields {
		require.NotEmpty(t, jsonField(oc, f), "precondition: the order carries %q", f)
	}

	pickID := pickIDForOrder(t, order)

	// Detail: the floor reads the full deadline and its derivation with no include.
	assertPickCarriesCommitment(t, retrievePick(t, pickID), order)

	// List: a separate query hydrates the same row, so the picking index shows it too.
	assertPickCarriesCommitment(t, pickRowByID(t, customerID, pickID), order)
}

// assertPickCarriesCommitment checks that a pick — detail or list row — carries the whole
// denormalized commitment its order does, and none of the authoring history that stays behind.
func assertPickCarriesCommitment(t *testing.T, pick, order map[string]any) {
	t.Helper()

	pc := commitmentOf(pick)
	require.NotNil(t, pc, "the pick must carry a commitment")
	oc := commitmentOf(order)

	// ship_by_date folds in a pickup cutoff and serializes as a timestamp, so compare the
	// normalized date rather than the raw string.
	assert.Equal(t, shipByDate(t, order), shipByDate(t, pick), "the pick's ship-by matches its order's")
	for _, f := range pickCommitmentFields {
		if f == "ship_by_date" {
			continue
		}
		assert.NotEmpty(t, jsonField(pc, f), "%q should be populated on a fully-committed pick", f)
		assert.Equal(t, jsonField(oc, f), jsonField(pc, f), "the pick's %q must match the order's", f)
	}

	// A pick carries the date, not how it was reached: the calendar adjustment and the arrival
	// estimate stay on the order (see commitmentFromPickProto).
	assert.Empty(t, jsonField(pc, "calendar_adjustment_days"), "the calendar adjustment stays on the order")
	assert.Empty(t, jsonField(pc, "estimated_delivery_date"), "a stamped record carries no arrival projection")
}

// pickRowByID returns the list row for one pick, paging the customer-scoped picks list until it
// is found. It looks up by id rather than assuming a single pick, because warming the lane re-issues
// the order, and every page is walked because a parallel suite churns the list (see
// [[project_list_hydration_race]]).
func pickRowByID(t *testing.T, customerID, pickID string) map[string]any {
	t.Helper()

	params := url.Values{"customer_ids": {customerID}, "limit": {"100"}}
	for range 20 {
		status, body, err := apiClient.GetListRaw(picksPath, params)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		parsed := parseJSON(body)
		data, _ := parsed["data"].([]any)
		for _, raw := range data {
			row, ok := raw.(map[string]any)
			if ok && jsonField(row, "id") == pickID {
				return row
			}
		}

		next := jsonField(jsonObject(parsed, "page_info"), "next_cursor")
		if next == "" {
			break
		}
		params.Set("cursor", next)
	}
	t.Fatalf("pick %s did not appear in the customer's picks list", pickID)
	return nil
}
