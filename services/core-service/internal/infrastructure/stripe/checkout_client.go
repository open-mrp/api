package stripe

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"

	gostripe "github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/customer"
)

var checkoutClientTracer = tracing.GetTracer("core-service.stripe_checkout_client")

type checkoutClientImpl struct {
	apiKey string
}

// NewCheckoutClientFactory creates a factory that produces per-account Stripe checkout clients.
func NewCheckoutClientFactory() domain.StripeCheckoutClientFactory {
	return &checkoutClientFactory{}
}

type checkoutClientFactory struct{}

func (f *checkoutClientFactory) Build(apiKey string) domain.StripeCheckoutClient {
	return &checkoutClientImpl{apiKey: apiKey}
}

func (c *checkoutClientImpl) CreateOneTimeCheckoutSession(ctx context.Context, params domain.CreateCheckoutSessionParams) (*domain.StripeCheckoutSession, *apierror.APIError) {
	_, span := checkoutClientTracer.Start(ctx, "stripe_checkout_client.create_session")
	defer span.End()

	lineItems := make([]*gostripe.CheckoutSessionLineItemParams, len(params.LineItems))
	for i, item := range params.LineItems {
		lineItems[i] = &gostripe.CheckoutSessionLineItemParams{
			PriceData: &gostripe.CheckoutSessionLineItemPriceDataParams{
				Currency: gostripe.String("usd"),
				ProductData: &gostripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        gostripe.String(item.Name),
					Description: gostripe.String(item.Description),
				},
				UnitAmount: new(item.AmountCents),
			},
			Quantity: new(item.Quantity),
		}
	}

	sessParams := &gostripe.CheckoutSessionParams{
		Mode:               gostripe.String(string(gostripe.CheckoutSessionModePayment)),
		LineItems:          lineItems,
		PaymentMethodTypes: gostripe.StringSlice([]string{"card"}),
	}

	// Bill the session to an existing Stripe customer when we have one (and only
	// then offer to save the payment method, matching the legacy dashboard) —
	// otherwise fall back to a bare customer_email. Stripe rejects both together.
	if params.StripeCustomerID != "" {
		sessParams.Customer = gostripe.String(params.StripeCustomerID)
		sessParams.SavedPaymentMethodOptions = &gostripe.CheckoutSessionSavedPaymentMethodOptionsParams{
			PaymentMethodSave: gostripe.String("enabled"),
		}
	} else {
		sessParams.CustomerEmail = gostripe.String(params.CustomerEmail)
	}

	if len(params.PaymentIntentMetadata) > 0 {
		sessParams.PaymentIntentData = &gostripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: params.PaymentIntentMetadata,
		}
	}

	if params.SuccessURL != nil {
		sessParams.SuccessURL = gostripe.String(*params.SuccessURL)
	}
	if params.CancelURL != nil {
		sessParams.CancelURL = gostripe.String(*params.CancelURL)
	}

	// Temporarily swap the global API key to use the per-account key. This is the approach used by the stripe-go SDK for per-request keys.
	savedKey := gostripe.Key
	gostripe.Key = c.apiKey
	sess, err := session.New(sessParams)
	gostripe.Key = savedKey

	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to create Stripe checkout session: %v", err))
	}

	return &domain.StripeCheckoutSession{URL: sess.URL}, nil
}

func (c *checkoutClientImpl) CreateEmbeddedCheckoutSession(ctx context.Context, params domain.CreateEmbeddedCheckoutSessionParams) (*domain.StripeEmbeddedCheckoutSession, *apierror.APIError) {
	_, span := checkoutClientTracer.Start(ctx, "stripe_checkout_client.create_embedded_session")
	defer span.End()

	lineItemName := fmt.Sprintf("SO #%s", params.OrderNumber)
	lineItem := &gostripe.CheckoutSessionLineItemParams{
		PriceData: &gostripe.CheckoutSessionLineItemPriceDataParams{
			Currency: gostripe.String("usd"),
			ProductData: &gostripe.CheckoutSessionLineItemPriceDataProductDataParams{
				Name: gostripe.String(lineItemName),
			},
			UnitAmount: new(params.OrderTotalCents),
		},
		Quantity: gostripe.Int64(1),
	}

	if params.CustomerPO != nil && *params.CustomerPO != "" {
		lineItem.PriceData.ProductData.Description = gostripe.String(fmt.Sprintf("PO #%s", *params.CustomerPO))
	}

	sessParams := &gostripe.CheckoutSessionParams{
		UIMode:             gostripe.String("custom"),
		Customer:           gostripe.String(params.StripeCustomerID),
		Mode:               gostripe.String(string(gostripe.CheckoutSessionModePayment)),
		PaymentMethodTypes: gostripe.StringSlice([]string{"card"}),
		LineItems:          []*gostripe.CheckoutSessionLineItemParams{lineItem},
		ReturnURL:          gostripe.String(params.ReturnURL),
		SavedPaymentMethodOptions: &gostripe.CheckoutSessionSavedPaymentMethodOptionsParams{
			PaymentMethodSave: gostripe.String("enabled"),
		},
		// Session-level metadata so checkout.session.completed webhooks carry the order reference directly (the session object does not echo payment_intent_data.metadata).
		Metadata: map[string]string{
			"orderID":    params.OrderID,
			"customerID": params.CustomerID,
		},
		PaymentIntentData: &gostripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"orderID":    params.OrderID,
				"customerID": params.CustomerID,
			},
		},
	}

	savedKey := gostripe.Key
	gostripe.Key = c.apiKey
	sess, err := session.New(sessParams)
	gostripe.Key = savedKey

	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to create embedded Stripe checkout session: %v", err))
	}

	if sess.ClientSecret == "" {
		return nil, apierror.NewResourceNotFoundError("Stripe checkout session client secret not found.")
	}

	return &domain.StripeEmbeddedCheckoutSession{
		ClientSecret: sess.ClientSecret,
	}, nil
}

func (c *checkoutClientImpl) CreateStripeCustomer(ctx context.Context, params domain.CreateStripeCustomerParams) (*domain.StripeCustomer, *apierror.APIError) {
	_, span := checkoutClientTracer.Start(ctx, "stripe_checkout_client.create_customer")
	defer span.End()

	custParams := &gostripe.CustomerParams{
		Email: gostripe.String(params.Email),
		Name:  gostripe.String(params.Name),
		Metadata: map[string]string{
			"customerID": params.CustomerID,
			"number":     params.Number,
		},
	}

	savedKey := gostripe.Key
	gostripe.Key = c.apiKey
	cust, err := customer.New(custParams)
	gostripe.Key = savedKey

	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to create Stripe customer: %v", err))
	}

	return &domain.StripeCustomer{
		ID: cust.ID,
	}, nil
}

func (c *checkoutClientImpl) ConstructWebhookEvent(payload []byte, signature, webhookSecret string) (*domain.StripeWebhookEvent, *domain.StripePaymentIntent, *apierror.APIError) {
	// Vendors pin their own Stripe API version, which will rarely match the version this SDK expects, so the default version-equality check would reject nearly every per-account event. Only version-stable fields (id, metadata, amount, payment_method_types) are read from the payload, which makes ignoring the mismatch safe.
	event, err := gostripe.ConstructEvent(payload, signature, webhookSecret, gostripe.WithIgnoreAPIVersionMismatch())
	if err != nil {
		return nil, nil, apierror.NewValidationError(fmt.Sprintf("Failed to verify webhook signature: %v", err))
	}

	webhookEvent := &domain.StripeWebhookEvent{
		ID:   event.ID,
		Type: string(event.Type),
	}

	if event.Data == nil {
		return webhookEvent, nil, nil
	}
	webhookEvent.RawJSON = event.Data.Raw

	var pi gostripe.PaymentIntent
	if err := pi.UnmarshalJSON(event.Data.Raw); err != nil {
		return webhookEvent, nil, nil
	}

	paymentIntent := &domain.StripePaymentIntent{
		ID:     pi.ID,
		Amount: pi.Amount,
	}

	if pi.Metadata != nil {
		paymentIntent.Metadata = pi.Metadata
	}

	if pi.PaymentMethodTypes != nil {
		paymentIntent.PaymentMethodTypes = pi.PaymentMethodTypes
	}

	return webhookEvent, paymentIntent, nil
}

func (c *checkoutClientImpl) ListPayoutPaymentIntentIDs(ctx context.Context, payoutID string) ([]string, *apierror.APIError) {
	_, span := checkoutClientTracer.Start(ctx, "stripe_checkout_client.list_payout_payment_intent_ids")
	defer span.End()

	// A dedicated client keeps the per-account key scoped for the duration of the multi-call walk, unlike the global-key swap used for single calls elsewhere in this file.
	sc := gostripe.NewClient(c.apiKey)

	params := &gostripe.BalanceTransactionListParams{Payout: gostripe.String(payoutID)}
	params.Limit = gostripe.Int64(100)

	seen := make(map[string]struct{})
	ids := make([]string, 0, 8)

	for bt, err := range sc.V1BalanceTransactions.List(ctx, params) {
		if err != nil {
			span.RecordError(err)
			return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to list payout balance transactions: %v", err))
		}
		if bt.Type != gostripe.BalanceTransactionTypeCharge && bt.Type != gostripe.BalanceTransactionTypePayment {
			continue
		}
		if bt.Source == nil || bt.Source.ID == "" {
			continue
		}
		charge, err := sc.V1Charges.Retrieve(ctx, bt.Source.ID, nil)
		if err != nil {
			span.RecordError(err)
			return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to resolve charge for payout balance transaction: %v", err))
		}
		if charge.PaymentIntent == nil || charge.PaymentIntent.ID == "" {
			continue
		}
		if _, ok := seen[charge.PaymentIntent.ID]; ok {
			continue
		}
		seen[charge.PaymentIntent.ID] = struct{}{}
		ids = append(ids, charge.PaymentIntent.ID)
	}

	return ids, nil
}
