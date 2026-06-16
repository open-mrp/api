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
		CustomerEmail:      gostripe.String(params.CustomerEmail),
		LineItems:          lineItems,
		PaymentMethodTypes: gostripe.StringSlice([]string{"card"}),
		SavedPaymentMethodOptions: &gostripe.CheckoutSessionSavedPaymentMethodOptionsParams{
			PaymentMethodSave: gostripe.String("enabled"),
		},
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

	// Temporarily swap the global API key to use the per-account key.
	// This is the approach used by the stripe-go SDK for per-request keys.
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
		// Session-level metadata so checkout.session.completed webhooks carry the
		// order reference directly (the session object does not echo
		// payment_intent_data.metadata).
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
	event, err := gostripe.ConstructEvent(payload, signature, webhookSecret)
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
