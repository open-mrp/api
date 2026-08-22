package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// customerObject is the minimal fields parsed from a Stripe customer data.object.
type customerObject struct {
	ID string `json:"id"`
}

// v2PricingPlanSubscriptionObject represents v2 pricing plan subscription data (fetched from the related_object URL in thin events).
type v2PricingPlanSubscriptionObject struct {
	ID               string `json:"id"`
	Customer         string `json:"customer"`
	ServicingStatus  string `json:"servicing_status"`
	CollectionStatus string `json:"collection_status"`
}

// v2CadenceObject represents v2 billing cadence data (fetched from the related_object URL in thin events).
type v2CadenceObject struct {
	ID       string     `json:"id"`
	Customer string     `json:"customer"`
	BilledTo *time.Time `json:"billed_to,omitempty"`
}

func (c *StripeWebhookConsumer) handleCustomerDeleted(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var cust customerObject
	if err := json.Unmarshal(rawObject, &cust); err != nil {
		return fmt.Errorf("failed to parse customer object: %w", err)
	}

	accountID, _, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, cust.ID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			log.Printf("[stripe_webhook] Customer deleted for Stripe customer %s: no account in DB, skipping", cust.ID)
			return nil
		}
		return apiErrToErr(apiErr)
	}

	log.Printf("[stripe_webhook] Customer deleted for account %s, clearing Stripe data", accountID)

	return apiErrToErr(c.coreClient.ClearAccountStripeCustomer(ctx, eventID, accountID))
}

// checkoutSessionObject is the minimal set of fields parsed from a Stripe checkout.session data.object.
type checkoutSessionObject struct {
	ID            string            `json:"id"`
	PaymentIntent string            `json:"payment_intent"`
	PaymentStatus string            `json:"payment_status"`
	Metadata      map[string]string `json:"metadata"`
}

// handleCheckoutSessionCompleted links a completed Stripe checkout to its sales order by recording the payment intent, which is what makes the order read as paid. The orderID is carried on the session metadata set at checkout creation.
func (c *StripeWebhookConsumer) handleCheckoutSessionCompleted(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sess checkoutSessionObject
	if err := json.Unmarshal(rawObject, &sess); err != nil {
		return fmt.Errorf("failed to parse checkout session object: %w", err)
	}

	orderID := sess.Metadata["orderID"]
	if orderID == "" {
		// Not an order checkout (e.g. a subscription session); nothing to link.
		log.Printf("[stripe_webhook] checkout.session.completed %s has no orderID metadata, skipping", sess.ID)
		return nil
	}

	if sess.PaymentStatus != "paid" {
		log.Printf("[stripe_webhook] checkout.session.completed for order %s not paid (status=%s), skipping", orderID, sess.PaymentStatus)
		return nil
	}

	if sess.PaymentIntent == "" {
		log.Printf("[stripe_webhook] checkout.session.completed for order %s has no payment_intent, skipping", orderID)
		return nil
	}

	log.Printf("[stripe_webhook] Recording payment for order %s (payment_intent=%s)", orderID, sess.PaymentIntent)

	return apiErrToErr(c.coreClient.RecordOrderPayment(ctx, eventID, orderID, sess.PaymentIntent))
}

func (c *StripeWebhookConsumer) handleServicingActivated(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub v2PricingPlanSubscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse pricing plan subscription object: %w", err)
	}

	accountID, _, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	servicingStatus := "active"
	log.Printf("[stripe_webhook] Pricing plan subscription servicing activated for account %s", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, "", nil, nil, nil, nil, nil, &sub.ID, &servicingStatus, nil,
	))
}

func (c *StripeWebhookConsumer) handleServicingCanceled(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub v2PricingPlanSubscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse pricing plan subscription object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	// Mark servicing as canceled but keep the current plan active until the paid period ends. The plan will be downgraded to free when the period expires or via a separate cleanup job.
	servicingStatus := "canceled"
	log.Printf("[stripe_webhook] Pricing plan subscription servicing canceled for account %s, marking status (plan retained until period end)", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, planCode, nil, nil, nil, nil, nil, nil, &servicingStatus, nil,
	))
}

func (c *StripeWebhookConsumer) handleCollectionPaused(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub v2PricingPlanSubscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse pricing plan subscription object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	collectionStatus := "paused"
	log.Printf("[stripe_webhook] Collection paused for account %s", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, planCode, nil, nil, nil, nil, nil, nil, nil, &collectionStatus,
	))
}

func (c *StripeWebhookConsumer) handleCollectionCurrent(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub v2PricingPlanSubscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse pricing plan subscription object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	collectionStatus := "current"
	log.Printf("[stripe_webhook] Collection current for account %s", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, planCode, nil, nil, nil, nil, nil, nil, nil, &collectionStatus,
	))
}

func (c *StripeWebhookConsumer) handleCadenceErrored(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var cadence v2CadenceObject
	if err := json.Unmarshal(rawObject, &cadence); err != nil {
		return fmt.Errorf("failed to parse cadence object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, cadence.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	status := constants.SubscriptionStatusPastDue.String()
	log.Printf("[stripe_webhook] Billing cadence errored for account %s, setting status to past_due", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, &status, planCode, nil, nil, nil, nil, nil, nil, nil, nil,
	))
}

func (c *StripeWebhookConsumer) handleServicingPaused(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub v2PricingPlanSubscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse pricing plan subscription object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	servicingStatus := "paused"
	log.Printf("[stripe_webhook] Pricing plan subscription servicing paused for account %s", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, planCode, nil, nil, nil, nil, nil, nil, &servicingStatus, nil,
	))
}

func (c *StripeWebhookConsumer) handleCollectionAwaitingCustomerAction(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var sub v2PricingPlanSubscriptionObject
	if err := json.Unmarshal(rawObject, &sub); err != nil {
		return fmt.Errorf("failed to parse pricing plan subscription object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, sub.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	collectionStatus := "awaiting_customer_action"
	log.Printf("[stripe_webhook] Collection awaiting customer action for account %s", accountID)

	if err := apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, planCode, nil, nil, nil, nil, nil, nil, nil, &collectionStatus,
	)); err != nil {
		return err
	}

	adminEmail, emailErr := c.accountUsageRepo.GetAdminEmailByAccountID(ctx, accountID)
	if emailErr != nil {
		log.Printf("[stripe_webhook] Failed to get admin email for account %s, skipping notification: %v", accountID, emailErr)
		return nil
	}

	if notifErr := c.notificationClient.SendPaymentActionRequired(ctx, accountID, adminEmail); notifErr != nil {
		log.Printf("[stripe_webhook] Failed to send payment action required notification for account %s: %v", accountID, notifErr)
	}

	return nil
}

func (c *StripeWebhookConsumer) handleCadenceBilled(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var cadence v2CadenceObject
	if err := json.Unmarshal(rawObject, &cadence); err != nil {
		return fmt.Errorf("failed to parse cadence object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, cadence.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	log.Printf("[stripe_webhook] Billing cadence billed for account %s", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, nil, planCode, nil, cadence.BilledTo, nil, nil, nil, nil, nil, nil,
	))
}

func (c *StripeWebhookConsumer) handleCadenceCanceled(ctx context.Context, eventID string, rawObject json.RawMessage) error {
	var cadence v2CadenceObject
	if err := json.Unmarshal(rawObject, &cadence); err != nil {
		return fmt.Errorf("failed to parse cadence object: %w", err)
	}

	accountID, planCode, apiErr := c.coreClient.GetAccountByStripeCustomerID(ctx, cadence.Customer)
	if apiErr != nil {
		return apiErrToErr(apiErr)
	}

	status := constants.SubscriptionStatusCanceled.String()
	log.Printf("[stripe_webhook] Billing cadence canceled for account %s, setting status to canceled", accountID)

	return apiErrToErr(c.coreClient.UpdateAccountSubscription(
		ctx, eventID, accountID, &status, planCode, nil, nil, nil, nil, nil, nil, nil, nil,
	))
}

func apiErrToErr(apiErr *apierror.APIError) error {
	if apiErr == nil {
		return nil
	}
	return fmt.Errorf("[%s] %s", apiErr.Code, apiErr.PublicMessage)
}
