//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const purchaseOrdersPath = "/v1/operations/purchase-orders"

// firstPurchaseOrderID returns the id of the first purchase order in seed data.
// Fails loudly if no PO is seeded so missing fixtures surface rather than skip.
func firstPurchaseOrderID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(purchaseOrdersPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "purchase orders list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one purchase order must be seeded")
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// PurchaseOrder — Include Tests
// ──────────────────────────────────────────────

func TestPurchaseOrders_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)

	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["supplier"], "supplier should be null without ?include=supplier")
	assert.Nil(t, got["bill_to_address"], "bill_to_address should be null without ?include=bill_to_address")
	assert.Nil(t, got["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
	assert.Nil(t, got["carrier"], "carrier should be null without ?include=carrier")
	assert.Nil(t, got["service_level"], "service_level should be null without ?include=service_level")
	assert.Nil(t, got["payment_term"], "payment_term should be null without ?include=payment_term")
	assert.Nil(t, got["shipping_term"], "shipping_term should be null without ?include=shipping_term")
	assert.Nil(t, got["receiving_order"], "receiving_order should be null without ?include=receiving_order")
	assert.Nil(t, got["lines"], "lines should be null without ?include=lines")
	assert.Nil(t, got["contacts"], "contacts should be null without ?include=contacts")
}

func TestPurchaseOrders_IncludeSupplier(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"supplier"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	sup := jsonObject(got, "supplier")
	require.NotNil(t, sup, "supplier should be present with ?include=supplier")
	assert.Equal(t, "supplier", jsonField(sup, "object"))
}

func TestPurchaseOrders_IncludeBillToAddress(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["bill_to_address"]
	assert.True(t, ok, "bill_to_address key should be present with ?include=bill_to_address")
	if addr := jsonObject(got, "bill_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
	}
}

func TestPurchaseOrders_IncludeShipToAddress(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["ship_to_address"]
	assert.True(t, ok, "ship_to_address key should be present with ?include=ship_to_address")
	if addr := jsonObject(got, "ship_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
	}
}

// Carrier and service level are no longer top-level purchase order includes;
// they are nested under the consolidated freight sub-resource (include[]=freight).
func TestPurchaseOrders_IncludeCarrier(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"freight"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	freight := jsonObject(got, "freight")
	require.NotNil(t, freight, "freight key should be present with ?include=freight")
	if c := jsonObject(freight, "carrier"); c != nil {
		assert.Equal(t, "carrier", jsonField(c, "object"))
	}
}

func TestPurchaseOrders_IncludeServiceLevel(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"freight"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	freight := jsonObject(got, "freight")
	require.NotNil(t, freight, "freight key should be present with ?include=freight")
	if sl := jsonObject(freight, "service_level"); sl != nil {
		assert.Equal(t, "service_level", jsonField(sl, "object"))
	}
}

func TestPurchaseOrders_IncludePaymentTerm(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"payment_term"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["payment_term"]
	assert.True(t, ok, "payment_term key should be present with ?include=payment_term")
	if pt := jsonObject(got, "payment_term"); pt != nil {
		assert.Equal(t, "payment_term", jsonField(pt, "object"))
	}
}

func TestPurchaseOrders_IncludeShippingTerm(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"shipping_term"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["shipping_term"]
	assert.True(t, ok, "shipping_term key should be present with ?include=shipping_term")
	if st := jsonObject(got, "shipping_term"); st != nil {
		assert.Equal(t, "shipping_term", jsonField(st, "object"))
	}
}

func TestPurchaseOrders_IncludeReceivingOrder(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"receiving_order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["receiving_order"]
	assert.True(t, ok, "receiving_order key should be present with ?include=receiving_order")
}

func TestPurchaseOrders_IncludeLines(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "lines should be present with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}

func TestPurchaseOrders_IncludeContacts(t *testing.T) {
	t.Parallel()
	id := firstPurchaseOrderID(t)
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+id, url.Values{"include": {"contacts"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	contacts := jsonObject(parseJSON(body), "contacts")
	require.NotNil(t, contacts, "contacts should be present with ?include=contacts")
	assert.Equal(t, "list", jsonField(contacts, "object"))
}
