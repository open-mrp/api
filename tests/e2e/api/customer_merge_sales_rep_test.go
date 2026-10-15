//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A merge moves the source customers' orders onto the target. Open, uninvoiced orders take the target's
// default rep, as if they had been placed under it; invoiced orders keep theirs, so commission already
// reported does not move.

func orderSalesRepID(t *testing.T, orderID string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	rep := jsonObject(parseJSON(body), "sales_rep")
	if rep == nil {
		return ""
	}
	return jsonField(rep, "id")
}

func createOrderForCustomer(t *testing.T, customerID string, extra map[string]any) string {
	t.Helper()
	body := minimalSalesOrderCreateBody(t, customerID)
	for k, v := range extra {
		body[k] = v
	}
	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)
	return orderID
}

// shipOrderWithRep issues an order to the customer under the given rep and ships it, which invoices it.
func shipOrderWithRep(t *testing.T, customerID, salesRepID string) string {
	t.Helper()
	order := issueOrderForCustomer(t, customerID, map[string]any{"sales_rep_id": salesRepID})
	orderID := jsonField(order, "id")

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"related.pick"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	pickID := jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "pick"), "id")
	require.NotEmpty(t, pickID, "issuing must create a pick: %s", string(body))

	pickAllLines(t, pickID)
	packPick(t, pickID)
	numbers := pickShipmentNumbers(t, pickID)
	require.NotEmpty(t, numbers, "packing must produce a shipment")
	listStatus, listBody, err := apiClient.GetListRaw(shipmentsPath, url.Values{"q": {numbers[0]}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	shipments := jsonArray(parseJSON(listBody), "data")
	require.NotEmpty(t, shipments)
	shipmentID := jsonField(shipments[0].(map[string]any), "id")

	shipStatus, shipBody := acceptShipAndAwait(t, shipmentID, false)
	requireStatus(t, 200, shipStatus, shipBody)
	t.Cleanup(func() {
		_, _, _ = apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/void", map[string]any{}, newIdempotencyKey())
	})
	return orderID
}

// mergeCustomers merges sourceIDs into targetID and waits for the merge job to complete.
func mergeCustomers(t *testing.T, targetID string, sourceIDs ...string) {
	t.Helper()
	status, body, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{
		"source_customer_ids": sourceIDs,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)
	job := pollJobUntilTerminal(t, jsonField(parseJSON(body), "id"))
	require.Equal(t, "completed", jsonField(job, "status"), "the merge job completes: %v", job)
}

func TestCustomerMerge_OpenOrdersTakeTheTargetsDefaultRep(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, map[string]any{
		"default_sales_rep_id": SeedAccountUserID,
		"commission_policy":    "applied",
	})
	targetID := jsonField(target, "id")
	sourceID := setupOrderCustomer(t)

	estimateID := createOrderForCustomer(t, sourceID, map[string]any{"sales_rep_id": SeedAdmin2AccountUserID})
	issuedID := jsonField(issueOrderForCustomer(t, sourceID, map[string]any{"sales_rep_id": SeedAdmin2AccountUserID}), "id")
	invoicedID := shipOrderWithRep(t, sourceID, SeedAdmin2AccountUserID)

	targetRep := orderSalesRepID(t, createOrderForCustomer(t, targetID, nil))
	require.NotEmpty(t, targetRep, "an order placed under the target takes its default rep")
	sourceRep := orderSalesRepID(t, estimateID)
	require.NotEqual(t, targetRep, sourceRep)

	since := auditWindowStart()
	mergeCustomers(t, targetID, sourceID)

	assert.Equal(t, targetRep, orderSalesRepID(t, estimateID), "an open estimate takes the target's rep")
	assert.Equal(t, targetRep, orderSalesRepID(t, issuedID), "an issued, uninvoiced order takes the target's rep")
	assert.Equal(t, sourceRep, orderSalesRepID(t, invoicedID), "an invoiced order keeps its rep")

	event := findAuditEvent(t, apiClient, since, estimateID, "sales_order", "update", anyEvent)
	assert.Equal(t, "customer_merge", jsonField(jsonObject(event, "metadata"), "source"))
}

func TestCustomerMerge_TargetWithoutDefaultRepLeavesOrdersAlone(t *testing.T) {
	t.Parallel()

	targetID := setupOrderCustomer(t)
	sourceID := setupOrderCustomer(t)
	orderID := createOrderForCustomer(t, sourceID, map[string]any{"sales_rep_id": SeedAdmin2AccountUserID})
	before := orderSalesRepID(t, orderID)
	require.NotEmpty(t, before)

	mergeCustomers(t, targetID, sourceID)

	assert.Equal(t, before, orderSalesRepID(t, orderID))
}

func TestCustomerMerge_CommissionExemptTargetLeavesOrdersAlone(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, map[string]any{
		"default_sales_rep_id": SeedAccountUserID,
		"commission_policy":    "exempt",
	})
	targetID := jsonField(target, "id")
	sourceID := setupOrderCustomer(t)
	orderID := createOrderForCustomer(t, sourceID, map[string]any{"sales_rep_id": SeedAdmin2AccountUserID})
	before := orderSalesRepID(t, orderID)
	require.NotEmpty(t, before)

	mergeCustomers(t, targetID, sourceID)

	assert.Equal(t, before, orderSalesRepID(t, orderID))
}
