//go:build e2e

package api_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shipping, voiding and editing a shipment the way the dashboard does, against the Shippo stub.
//
// A shipment shipped to a stub label zip buys labels: the stub numbers each purchase per shipment
// and prints the number into the master tracking (STUB-<n>-<shipment id>), so counting purchases is
// reading that number. Each test builds its own order, so the suite runs again on the same stack.

const (
	// The stub buys labels for these destinations (stub.ZipStubLabels, stub.ZipStubLabelsNoRefund).
	zipStubLabels         = "99920"
	zipStubLabelsNoRefund = "99921"
)

// --- helpers ---

// An order for two of the seed product, to the given customer.
func parityOrderBody(t *testing.T, customerID string) map[string]any {
	t.Helper()
	body := minimalSalesOrderCreateBody(t, customerID)
	body["lines"] = []map[string]any{{
		"product_id": SeedProductID,
		"quantity":   map[string]any{"value": "2", "unit_id": SeedUnitID},
	}}
	return body
}

// A packed shipment on the Shippo-backed carrier, bound for a stub zip, with every case weighed.
func labelShipment(t *testing.T, zip string) string {
	t.Helper()
	body := parityOrderBody(t, SeedCustomerAccountID)
	body["carrier_id"] = SeedTransitCarrierID
	body["service_level_id"] = SeedTransitGroundServiceLevelID
	body["ship_to_address_id"] = transitAddress(t, zip)
	_, _, shipmentID := packedOrder(t, body)
	weighShipmentCases(t, shipmentID, "5")
	return shipmentID
}

func weighShipmentCases(t *testing.T, shipmentID, weight string) {
	t.Helper()
	for _, caseID := range shipmentCaseIDs(t, shipmentID) {
		status, body, err := apiClient.Patch(shippingCasesPath+"/"+caseID, map[string]any{"freight_weight_value": weight}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}
}

func shipParity(t *testing.T, shipmentID string) (int, []byte) {
	t.Helper()
	status, body, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/ship", map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "ship must not 5xx: %s", string(body))
	return status, body
}

func voidParity(t *testing.T, shipmentID string) (int, []byte) {
	t.Helper()
	status, body, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/void", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "void must not 5xx: %s", string(body))
	return status, body
}

// The purchase the stub numbered into the shipment's master tracking.
func stubPurchaseTracking(n int, shipmentID string) string {
	return fmt.Sprintf("STUB-%d-%s", n, shipmentID)
}

func shipmentCases(t *testing.T, shipmentID string) []map[string]any {
	t.Helper()
	data := jsonListData(readShipment(t, shipmentID, "shipping_cases"), "shipping_cases")
	cases := make([]map[string]any, 0, len(data))
	for _, raw := range data {
		sc, ok := raw.(map[string]any)
		require.True(t, ok)
		cases = append(cases, sc)
	}
	require.NotEmpty(t, cases, "a packed shipment has cases")
	return cases
}

func countInvoicesNumbered(t *testing.T, number string) int {
	t.Helper()
	var count int
	require.NoError(t, authDB(t).QueryRow(
		"SELECT COUNT(*) FROM invoice WHERE account_id = ? AND number = ?", SeedAccountID, number,
	).Scan(&count))
	return count
}

func linkedInvoiceID(t *testing.T, shipmentID string) string {
	t.Helper()
	invoice := jsonObject(jsonObject(readShipment(t, shipmentID, "related.invoice"), "related"), "invoice")
	require.NotNil(t, invoice, "a shipped shipment links its invoice")
	return jsonField(invoice, "id")
}

// --- label purchases ---

// Two ships of one shipment at once: one ships it, the other conflicts, and only one label purchase
// and one invoice come of it.
func TestShipmentParity_ConcurrentShipsBuyOnceAndInvoiceOnce(t *testing.T) {
	t.Parallel()

	shipmentID := labelShipment(t, zipStubLabels)
	number := jsonField(readShipment(t, shipmentID), "number")

	statuses := make([]int, 2)
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Go(func() {
			status, _, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/ship", map[string]any{"email_customer": false}, newIdempotencyKey())
			assert.NoError(t, err)
			statuses[i] = status
		})
	}
	wg.Wait()
	assert.ElementsMatch(t, []int{200, 409}, statuses, "exactly one ship wins; the other conflicts")

	shipped := readShipment(t, shipmentID)
	assert.Equal(t, "shipped", jsonField(shipped, "status"))
	assert.Equal(t, stubPurchaseTracking(1, shipmentID), jsonField(shipped, "master_tracking_number"),
		"the winner's labels came from the shipment's first purchase")
	for _, sc := range shipmentCases(t, shipmentID) {
		assert.True(t, strings.HasPrefix(jsonField(sc, "shippo_transaction_id"), "stub_txn_1_"+shipmentID),
			"every case records the one purchase: %v", sc)
	}
	assert.Equal(t, 1, countInvoicesNumbered(t, number), "the shipment is invoiced once")

	// A second purchase is the next number, so the loser never reached the carrier.
	status, body := voidParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	status, body = shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	assert.Equal(t, stubPurchaseTracking(2, shipmentID), jsonField(parseJSON(body), "master_tracking_number"),
		"only one purchase happened before this one")
}

// A taken invoice number fails the ship before any label is bought, rather than after it is paid for.
func TestShipmentParity_DuplicateInvoiceNumberIsRefusedBeforeBuying(t *testing.T) {
	t.Parallel()

	_, _, invoicedID := packedOrder(t, parityOrderBody(t, SeedCustomerAccountID))
	status, body := shipParity(t, invoicedID)
	requireStatus(t, 200, status, body)
	takenNumber := jsonField(readShipment(t, invoicedID), "number")

	shipmentID := labelShipment(t, zipStubLabels)
	status, body, err := apiClient.Patch(shipmentsPath+"/"+shipmentID, map[string]any{"number": takenNumber}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body = shipParity(t, shipmentID)
	assert.Equal(t, 409, status, "an invoice already carries this number: %s", string(body))
	assert.Empty(t, jsonField(readShipment(t, shipmentID), "shipped_at"))

	status, body, err = apiClient.Patch(shipmentsPath+"/"+shipmentID, map[string]any{"number": uniqueName("SHP-PARITY")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body = shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	assert.Equal(t, stubPurchaseTracking(1, shipmentID), jsonField(parseJSON(body), "master_tracking_number"),
		"the refused ship bought nothing, so this is the shipment's first purchase")
}

// A case with no weight cannot be rated, so its label is refused before the carrier is called.
func TestShipmentParity_UnweighedCaseIsRefusedBeforeBuying(t *testing.T) {
	t.Parallel()

	shipmentID := labelShipment(t, zipStubLabels)
	weighShipmentCases(t, shipmentID, "0")

	status, body := shipParity(t, shipmentID)
	assert.Equal(t, 400, status, "an unweighed case refuses the label: %s", string(body))

	weighShipmentCases(t, shipmentID, "5")
	status, body = shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	assert.Equal(t, stubPurchaseTracking(1, shipmentID), jsonField(parseJSON(body), "master_tracking_number"))
}

// A refund the carrier refuses stops the void, so the charged label is never wiped from the case.
func TestShipmentParity_RefusedRefundStopsTheVoid(t *testing.T) {
	t.Parallel()

	shipmentID := labelShipment(t, zipStubLabelsNoRefund)
	status, body := shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)

	status, body = voidParity(t, shipmentID)
	assert.Equal(t, 400, status, "the carrier refused the refund: %s", string(body))

	assert.Equal(t, "shipped", jsonField(readShipment(t, shipmentID), "status"), "nothing is unwound")
	cases := shipmentCases(t, shipmentID)
	for _, sc := range cases {
		assert.NotEmpty(t, jsonField(sc, "shippo_transaction_id"), "the label stays on record to refund later: %v", sc)
	}

	// Nor can the case be deleted out from under its unrefunded label.
	status, body, err := apiClient.Delete(shippingCasesPath + "/" + jsonField(cases[0], "id"))
	require.NoError(t, err)
	assert.Equal(t, 409, status, "a case with a bought label cannot be deleted: %s", string(body))
}

// --- tracking edits ---

func TestShipmentParity_TrackingNumbersClearWithNull(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := packedOrder(t, parityOrderBody(t, SeedCustomerAccountID))
	caseID := shipmentCaseIDs(t, shipmentID)[0]

	patch := func(path string, body map[string]any) map[string]any {
		t.Helper()
		status, resp, err := apiClient.Patch(path, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		return parseJSON(resp)
	}

	got := patch(shipmentsPath+"/"+shipmentID, map[string]any{"master_tracking_number": "1Z-PARITY"})
	assert.Equal(t, "1Z-PARITY", jsonField(got, "master_tracking_number"))
	got = patch(shipmentsPath+"/"+shipmentID, map[string]any{"note": "kept"})
	assert.Equal(t, "1Z-PARITY", jsonField(got, "master_tracking_number"), "an omitted tracking number is left alone")
	got = patch(shipmentsPath+"/"+shipmentID, map[string]any{"master_tracking_number": nil})
	assert.Nil(t, got["master_tracking_number"], "null clears the master tracking number")

	got = patch(shippingCasesPath+"/"+caseID, map[string]any{"tracking_number": "1Z-CASE"})
	assert.Equal(t, "1Z-CASE", jsonField(got, "tracking_number"))
	got = patch(shippingCasesPath+"/"+caseID, map[string]any{"tracking_number": nil})
	assert.Nil(t, got["tracking_number"], "null clears the case tracking number")
}

func TestShipmentParity_AdminTrackingClearsWithNull(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := packedOrder(t, parityOrderBody(t, SeedCustomerAccountID))
	status, body := shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	caseID := shipmentCaseIDs(t, shipmentID)[0]

	post := func(path string, body map[string]any) map[string]any {
		t.Helper()
		status, resp, err := apiClient.Post(path+adminUpdateTrackingAction, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		return parseJSON(resp)
	}

	got := post(shipmentsPath+"/"+shipmentID, map[string]any{"master_tracking_number": "1Z-ADMIN"})
	assert.Equal(t, "1Z-ADMIN", jsonField(got, "master_tracking_number"))
	got = post(shipmentsPath+"/"+shipmentID, map[string]any{"master_tracking_number": nil})
	assert.Nil(t, got["master_tracking_number"], "null clears a shipped shipment's tracking")

	got = post(shippingCasesPath+"/"+caseID, map[string]any{"tracking_number": "1Z-ADMIN-CASE"})
	assert.Equal(t, "1Z-ADMIN-CASE", jsonField(got, "tracking_number"))
	got = post(shippingCasesPath+"/"+caseID, map[string]any{"tracking_number": nil})
	assert.Nil(t, got["tracking_number"], "null clears a shipped case's tracking")
}

// --- shape ---

// Tracking links are built from the carrier's code, so the shipment's freight and its cases carry it.
func TestShipmentParity_CarrierCodeOnFreightAndCases(t *testing.T) {
	t.Parallel()

	shipmentID := labelShipment(t, zipStubLabels)
	shipment := readShipment(t, shipmentID, "freight", "shipping_cases")

	carrier := jsonObject(jsonObject(shipment, "freight"), "carrier")
	require.NotNil(t, carrier, "freight carries its carrier: %v", shipment)
	assert.Equal(t, "fedex", jsonField(carrier, "code"))

	for _, raw := range jsonListData(shipment, "shipping_cases") {
		sc := raw.(map[string]any)
		assert.Equal(t, "fedex", jsonField(jsonObject(sc, "carrier"), "code"), "case carrier: %v", sc)
	}
}

func TestShipmentParity_CustomerPurchaseOrderOnTheRelatedOrder(t *testing.T) {
	t.Parallel()

	po := uniqueName("PO")
	body := parityOrderBody(t, SeedCustomerAccountID)
	body["customer_purchase_order_number"] = po
	_, _, shipmentID := packedOrder(t, body)

	order := jsonObject(jsonObject(readShipment(t, shipmentID, "related.sales_order"), "related"), "sales_order")
	require.NotNil(t, order)
	assert.Equal(t, po, jsonField(jsonObject(order, "metadata"), "customer_purchase_order_number"))

	// Without the include the related order, and so its PO, is not loaded.
	assert.Nil(t, readShipment(t, shipmentID)["related"])
}

// --- delete ---

func TestShipmentParity_DeletingAShippedShipmentConflicts(t *testing.T) {
	t.Parallel()

	_, _, shipmentID := packedOrder(t, parityOrderBody(t, SeedCustomerAccountID))
	status, body := shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)

	status, body, err := apiClient.Delete(shipmentsPath + "/" + shipmentID)
	require.NoError(t, err)
	assert.Equal(t, 409, status, "a shipped shipment is voided, not deleted: %s", string(body))

	status, body = voidParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	status, body, err = apiClient.Delete(shipmentsPath + "/" + shipmentID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}
