package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// subscriptionObject is the minimal fields parsed from a Stripe subscription data.object.
type subscriptionObject struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Status   string `json:"status"`
}

// customerObject is the minimal fields parsed from a Stripe customer data.object.
type customerObject struct {
	ID string `json:"id"`
}

// invoiceObject is the minimal fields parsed from a Stripe invoice data.object.
type invoiceObject struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
}

func (c *StripeWebhookConsumer) handleSubscriptionUpdated(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub subscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription object: %w", err)
	}

	accountID, _, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	// Fetch full subscription details from Stripe for plan info
	stripeSub, err := c.stripeClient.GetSubscription(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscription from Stripe: %w", err)
	}

	status := stripeSub.Status
	periodEnd := stripeSub.CurrentPeriodEnd
	subID := stripeSub.ID

	log.Printf("[stripe_webhook] Updating subscription for account %s: plan=%s, status=%s", accountID, stripeSub.PlanCode, status)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, &status, stripeSub.PlanCode, &subID, &periodEnd, nil,
	))
}

func (c *StripeWebhookConsumer) handleSubscriptionDeleted(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub subscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription object: %w", err)
	}

	accountID, _, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	log.Printf("[stripe_webhook] Subscription deleted for account %s, reverting to free plan", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, string(constants.PlanCodeFree), nil, nil, nil,
	))
}

func (c *StripeWebhookConsumer) handleCustomerDeleted(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var cust customerObject
	if err := json.Unmarshal(rawObject, &cust); err != nil {
		return fmt.Errorf("failed to parse customer object: %w", err)
	}

	accountID, _, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, cust.ID)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	log.Printf("[stripe_webhook] Customer deleted for account %s, clearing Stripe data", accountID)

	return apiErrToErr(c.coreClient.ClearAccountStripeCustomer(ctx, eventID, accountID))
}

func (c *StripeWebhookConsumer) handleInvoicePaymentFailed(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var inv invoiceObject
	if err := json.Unmarshal(rawObject, &inv); err != nil {
		return fmt.Errorf("failed to parse invoice object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, inv.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	status := constants.SubscriptionStatusPastDue.String()
	log.Printf("[stripe_webhook] Invoice payment failed for account %s, setting status to past_due", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, &status, planCode, nil, nil, nil,
	))
}

func apiErrToErr(apiErr *apierror.APIError) error {
	if apiErr == nil {
		return nil
	}
	return fmt.Errorf("[%s] %s", apiErr.Code, apiErr.PublicMessage)
}
