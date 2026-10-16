//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Changes OpenMRP makes outside any request — here, linking a payment Stripe reported to the
// order it paid — are audited with the system as their actor.

// postCheckoutCompleted delivers a signed checkout.session.completed for orderID to the platform
// Stripe webhook, as Stripe does when a portal customer pays an order.
func postCheckoutCompleted(t *testing.T, orderID, paymentIntentID string) {
	t.Helper()
	event := map[string]any{
		"id":   "evt_e2e_" + paymentIntentID,
		"type": "checkout.session.completed",
		"data": map[string]any{"object": map[string]any{
			"id":             "cs_e2e_" + paymentIntentID,
			"payment_intent": paymentIntentID,
			"payment_status": "paid",
			"amount_total":   1000,
			"currency":       "usd",
			"customer":       "cus_e2e_system_actor",
			"metadata": map[string]any{
				"orderID":             orderID,
				"expectedAmountCents": "1000",
				"expectedCurrency":    "usd",
				"stripeCustomerID":    "cus_e2e_system_actor",
			},
		}},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	status, body, err := apiClient.PostSigned(stripeWebhookPath, StubStripeSignature, payload)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
}

// findPaymentLinkAuditEvent polls the order's audit history for the payment link, read with the actor.
func findPaymentLinkAuditEvent(t *testing.T, since, orderID string) map[string]any {
	t.Helper()
	var found map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
			"resource_ids":   {orderID},
			"resource_types": {"sales_order"},
			"actions":        {"update"},
			"occurred_after": {since},
			"include":        {"changes"},
		})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("audit events: %d %s", status, body)
		}
		for _, item := range jsonArray(parseJSON(body), "data") {
			event, _ := item.(map[string]any)
			if _, ok := changeForField(jsonListData(event, "changes"), "payment_intent_ids"); ok {
				found = event
				return nil
			}
		}
		return fmt.Errorf("no payment link audit event yet for %s", orderID)
	})
	return found
}

func TestSystemActor_StripePaymentLinkIsAuditedAsTheSystem(t *testing.T) {
	t.Parallel()
	since := time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339)
	order := createAndCleanup(t, salesOrdersPath, minimalSalesOrderCreateBody(t, SeedCustomerAccountID))
	orderID := jsonField(order, "id")
	paymentIntentID := "pi_e2e_" + uniqueName("sysactor")

	postCheckoutCompleted(t, orderID, paymentIntentID)

	event := findPaymentLinkAuditEvent(t, since, orderID)
	actor := jsonObject(event, "actor")
	require.NotNil(t, actor, "actor is always present: %v", event)
	assert.Equal(t, map[string]any{"type": "system", "id": nil, "name": "Stripe", "client": nil}, actor,
		"a Stripe webhook's change is the system's, named for the process, with no id")
	assert.Nil(t, event["source_ip"], "no request caused the change")
	assert.Nil(t, event["idempotency_key"], "no request caused the change")

	change, _ := changeForField(jsonListData(event, "changes"), "payment_intent_ids")
	assert.Contains(t, fmt.Sprint(change["new_value"]), paymentIntentID)

	// The system is an actor type the list filters by.
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
		"resource_ids":   {orderID},
		"actor_types":    {"system"},
		"occurred_after": {since},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var ids []string
	for _, item := range jsonArray(parseJSON(body), "data") {
		ids = append(ids, jsonField(item.(map[string]any), "id"))
	}
	assert.Contains(t, ids, jsonField(event, "id"))
}
