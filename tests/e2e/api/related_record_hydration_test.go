//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The `related` object on a purchasing document names the records it sits between.
//
// Two properties are pinned here. First, each reference is whole: a client reading `related` should
// be able to label the document it points at — number and status — without fetching it. Second, the
// children are independently expandable, so asking for one must not hand back the other; an
// expandable field that arrives unasked is one a client cannot tell was ever requested, and the
// whole point of `?include` is that the caller decides what they pay for.

// assertRecordReference checks a record reference is complete enough to display on its own.
func assertRecordReference(t *testing.T, record map[string]any, where string) {
	t.Helper()

	require.NotNil(t, record, "%s must be present", where)
	assertObjectField(t, record, "record")
	assert.NotEmpty(t, jsonField(record, "id"), "%s.id", where)
	assert.NotEmpty(t, jsonField(record, "type"), "%s.type names what the reference points at", where)
	assert.NotEmpty(t, jsonField(record, "number"),
		"%s.number — a reference nobody can name still costs a fetch to display", where)
	assert.NotEmpty(t, jsonField(record, "status"),
		"%s.status — the reference is there so a screen can label it without following it", where)
}

func relatedWithIncludes(t *testing.T, path string, includes ...string) (map[string]any, []byte) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(path, url.Values{"include": includes})
	require.NoError(t, err)
	require.Less(t, status, 500, "%s must not 5xx: %s", path, string(body))
	requireStatus(t, 200, status, body)

	related := jsonObject(parseJSON(body), "related")
	require.NotNil(t, related, "related must expand when asked for: %s", string(body))
	return related, body
}

// --- Deliveries ---

func TestDeliveries_RelatedReferencesAreCompleteRecords(t *testing.T) {
	t.Parallel()

	related, body := relatedWithIncludes(t, deliveriesPath+"/"+SeedDeliveryID,
		"related", "related.purchase_order", "related.receiving_order")

	assertRecordReference(t, jsonObject(related, "purchase_order"), "delivery related.purchase_order")

	// A delivery is booked against a purchase order, and that order's receiving order is the record it
	// was stocked through. Both are known to the delivery query, so both should be nameable.
	if receiving := jsonObject(related, "receiving_order"); receiving != nil {
		assertRecordReference(t, receiving, "delivery related.receiving_order")
	} else {
		t.Logf("the seeded delivery's order has no receiving order: %s", string(body))
	}
}

// Asking for the purchase order must not also return the receiving order. A single `related` include
// that attaches every reference it holds makes both children unconditional, which is the same as not
// having made them expandable at all.
func TestDeliveries_RelatedChildrenExpandIndependently(t *testing.T) {
	t.Parallel()

	onlyPurchaseOrder, poBody := relatedWithIncludes(t, deliveriesPath+"/"+SeedDeliveryID,
		"related", "related.purchase_order")
	assert.NotNil(t, onlyPurchaseOrder["purchase_order"],
		"the child that was asked for is present: %s", string(poBody))
	assertNilField(t, onlyPurchaseOrder, "receiving_order")

	onlyReceivingOrder, _ := relatedWithIncludes(t, deliveriesPath+"/"+SeedDeliveryID,
		"related", "related.receiving_order")
	assertNilField(t, onlyReceivingOrder, "purchase_order")
}

func TestDeliveries_RelatedIsAbsentWithoutTheInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(deliveriesPath+"/"+SeedDeliveryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assertNilField(t, parseJSON(body), "related")
}

// --- Purchase orders ---

func TestPurchaseOrders_RelatedChildrenExpandIndependently(t *testing.T) {
	t.Parallel()

	purchaseOrderID, _ := issuedPurchaseOrderReceiving(t)
	path := purchaseOrdersPath + "/" + purchaseOrderID

	onlyReceivingOrder, roBody := relatedWithIncludes(t, path, "related", "related.receiving_order")
	assert.NotNil(t, onlyReceivingOrder["receiving_order"],
		"issuing an order creates the receiving order that was asked for: %s", string(roBody))
	assertNilField(t, onlyReceivingOrder, "deliveries")

	onlyDeliveries, _ := relatedWithIncludes(t, path, "related", "related.deliveries")
	assertNilField(t, onlyDeliveries, "receiving_order")
}

// --- Receiving orders ---

func TestReceivingOrders_RelatedChildrenExpandIndependently(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	path := receivingOrdersPath + "/" + receivingOrderID

	onlyPurchaseOrder, poBody := relatedWithIncludes(t, path, "related", "related.purchase_order")
	assert.NotNil(t, onlyPurchaseOrder["purchase_order"],
		"a receiving order always knows the order that created it: %s", string(poBody))
	assertNilField(t, onlyPurchaseOrder, "deliveries")

	onlyDeliveries, _ := relatedWithIncludes(t, path, "related", "related.deliveries")
	assertNilField(t, onlyDeliveries, "purchase_order")
}

func TestReceivingOrders_RelatedPurchaseOrderIsACompleteRecord(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	related, _ := relatedWithIncludes(t, receivingOrdersPath+"/"+receivingOrderID,
		"related", "related.purchase_order")
	assertRecordReference(t, jsonObject(related, "purchase_order"),
		"receiving order related.purchase_order")
}
