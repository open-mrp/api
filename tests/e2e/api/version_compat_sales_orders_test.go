//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A preview.9 caller (the dashboard) writes a sales order with its flat fields and reads the old shape; forge reads
// the same order its own way.
func TestVersionCompat_SalesOrderRoundTrip(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	contact := createContact(t, apiClient.WithAccountID(customerID), uniqueName("p9-so"), uniqueName("p9-so")+"@e2e-test.openmrp.ai", nil)
	contactID := jsonField(contact, "id")

	body := minimalSalesOrderCreateBody(t, customerID)
	body["note"] = "Leave at dock 4."
	body["promised_at"] = "2026-11-02T00:00:00Z"
	body["carrier_billing_type"] = "third_party"
	body["carrier_billing_account_number"] = "TP-9"
	body["acknowledgement_email_contacts"] = []map[string]any{{"account_user_id": contactID}}
	body["invoice_email_contacts"] = []map[string]any{}
	status, resp, err := p9Client().Post(salesOrdersPath+"?include[]=freight&include[]=contacts&include[]=ship_to_address", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	order := parseJSON(resp)
	orderID := jsonField(order, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(salesOrdersPath + "/" + orderID) })

	assert.Equal(t, "estimate", jsonField(order, "status"))
	assert.Equal(t, "Leave at dock 4.", jsonField(order, "note"))
	assertNilField(t, order, "priority")
	assert.Equal(t, []any{}, order["payment_intent_ids"])
	for _, forge := range []string{"instructions", "currency", "bill_to", "ship_to", "site", "email_recipients", "fulfilled_at", "expires_on"} {
		assert.NotContains(t, order, forge)
	}
	freight := jsonObject(order, "freight")
	assert.Equal(t, "freight", jsonField(freight, "object"))
	assert.Equal(t, "third_party", jsonField(freight, "billing_type"))
	assert.Equal(t, "TP-9", jsonField(freight, "billing_account_number"))
	assert.Equal(t, SeedCarrierID, jsonField(jsonObject(freight, "carrier"), "id"))
	assert.Equal(t, "address", jsonField(jsonObject(order, "ship_to_address"), "object"))
	contacts := jsonObject(order, "contacts")
	assert.Len(t, contacts["acknowledgement"], 1)
	assert.Empty(t, contacts["invoice"])
	commitment := jsonObject(order, "commitment")
	assert.Equal(t, "commitment", jsonField(commitment, "object"))
	assert.Equal(t, "2026-11-02T00:00:00Z", jsonField(commitment, "promised_at"))

	// The same order as forge reads it.
	forge := forgeReadOrder(t, orderID, "email_recipients")
	assert.Equal(t, "draft", jsonField(forge, "status"))
	assert.Equal(t, "Leave at dock 4.", jsonField(forge, "instructions"))
	assert.Equal(t, "2026-11-02", jsonField(jsonObject(forge, "commitment"), "promised_on"))
	recipients := jsonListData(forge, "email_recipients")
	require.Len(t, recipients, 1)
	assert.Equal(t, []any{"acknowledgment"}, recipients[0].(map[string]any)["documents"])

	// preview.9 replaces one kind of recipient alone, and its PUT actions still move the order.
	status, resp, err = p9Client().Patch(salesOrdersPath+"/"+orderID+"?include[]=contacts", map[string]any{
		"invoice_email_contacts": []map[string]any{{"account_user_id": contactID}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	contacts = jsonObject(parseJSON(resp), "contacts")
	assert.Len(t, contacts["acknowledgement"], 1, "the acknowledgment recipients are kept")
	assert.Len(t, contacts["invoice"], 1)

	status, resp, err = p9Client().Put(salesOrdersPath+"/"+orderID+"/actions/issue", map[string]any{"notify_customer": false})
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	assert.Equal(t, "issued", jsonField(parseJSON(resp), "status"))

	status, resp, err = p9Client().Put(salesOrdersPath+"/"+orderID+"/actions/close", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	assert.Equal(t, "fulfilled", jsonField(parseJSON(resp), "status"), "preview.9's close fulfills the order")

	list, listStatus, err := p9Client().GetList(salesOrdersPath, url.Values{"status_codes[]": {"fulfilled"}, "customer_ids[]": {customerID}})
	require.NoError(t, err)
	require.Equal(t, 200, listStatus)
	require.Len(t, list.Data, 1)
}

// A forge cancel of an order nothing shipped from reads as fulfilled to preview.9, which has no canceled status.
func TestVersionCompat_CanceledSalesOrderReadsFulfilled(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	orderID := jsonField(forgeCreateOrder(t, forgeOrderBody(customerID)), "id")
	status, resp := forgeOrderAction(t, orderID, "issue", nil)
	requireStatus(t, 200, status, resp)
	status, resp = forgeOrderAction(t, orderID, "cancel", nil)
	requireStatus(t, 200, status, resp)

	status, resp, err := p9Client().GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	assert.Equal(t, "fulfilled", jsonField(parseJSON(resp), "status"))
}
