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

// Changes OpenMRP makes on its own behalf — here, linking a payment Stripe reported to the
// order it paid — are audited with no public actor.

// postCheckoutCompleted delivers a checkout.session.completed for orderID to the seed account's Stripe webhook, as Stripe does when a portal customer pays an order.
func postCheckoutCompleted(t *testing.T, orderID, paymentIntentID string) {
	t.Helper()
	connectSeedAccountStripe(t)
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

	status, body, err := apiClient.PostSigned(stripeWebhookPath+"/accounts/"+SeedAccountID, StubStripeSignature, payload)
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
			"starts_at":      {since},
			"include":        {"actor", "changes"},
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

func TestSystemActor_StripePaymentLinkIsAuditedWithNoActor(t *testing.T) {
	t.Parallel()
	since := time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339)
	order := createAndCleanup(t, salesOrdersPath, minimalSalesOrderCreateBody(t, SeedCustomerAccountID))
	orderID := jsonField(order, "id")
	paymentIntentID := "pi_e2e_" + uniqueName("sysactor")

	postCheckoutCompleted(t, orderID, paymentIntentID)

	event := findPaymentLinkAuditEvent(t, since, orderID)
	actorValue, present := event["actor"]
	assert.True(t, present, "actor is always present when included: %v", event)
	assert.Nil(t, actorValue, "a background change has no public actor")
	assert.NotEmpty(t, event["source_ip"], "the webhook delivery's IP is recorded")
	assert.Nil(t, event["idempotency_key"], "Stripe sends no idempotency key")

	change, _ := changeForField(jsonListData(event, "changes"), "payment_intent_ids")
	assert.Contains(t, fmt.Sprint(change["new_value"]), paymentIntentID)

	// The public actor types are unchanged.
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
		"resource_ids": {orderID},
		"actor_types":  {"system"},
		"starts_at":    {since},
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}
