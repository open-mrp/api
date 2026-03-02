package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/tracing"
	"github.com/stripe/stripe-go/v84"
	portalsession "github.com/stripe/stripe-go/v84/billingportal/session"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/customer"
	"github.com/stripe/stripe-go/v84/invoice"
	"github.com/stripe/stripe-go/v84/paymentmethod"
	"github.com/stripe/stripe-go/v84/price"
	"github.com/stripe/stripe-go/v84/product"
	"github.com/stripe/stripe-go/v84/subscription"
	"github.com/stripe/stripe-go/v84/webhook"
)

var stripeClientTracer = tracing.GetTracer("billing-service.stripe_client")

type ClientConfig struct {
	WebhookSecret string
	APIKey        string // #nosec G117 - Config field populated from env var at startup
}

type stripeClientImpl struct {
	webhookSecret string
}

func NewStripeClient(cfg *ClientConfig) domain.StripeClient {
	stripe.Key = cfg.APIKey
	return &stripeClientImpl{
		webhookSecret: cfg.WebhookSecret,
	}
}

func (c *stripeClientImpl) CreateSubscription(ctx context.Context, customerID, priceID string, quantity int64, defaultPaymentMethodID string) (*domain.StripeSubscription, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_subscription")
	defer span.End()

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(quantity),
			},
		},
		PaymentBehavior: stripe.String("error_if_incomplete"),
	}
	if defaultPaymentMethodID != "" {
		params.DefaultPaymentMethod = stripe.String(defaultPaymentMethodID)
	}
	params.AddExpand("items.data.price.product")
	params.AddExpand("latest_invoice.payment_intent")

	sub, err := subscription.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	var planCode string
	var currentPeriodStart time.Time
	var currentPeriodEnd time.Time
	if len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.Price != nil && item.Price.Product != nil {
			planCode = item.Price.Product.Metadata["plan_code"]
		}
		if item.CurrentPeriodStart > 0 {
			currentPeriodStart = time.Unix(item.CurrentPeriodStart, 0)
		}
		if item.CurrentPeriodEnd > 0 {
			currentPeriodEnd = time.Unix(item.CurrentPeriodEnd, 0)
		}
	}

	result := &domain.StripeSubscription{
		ID:                 sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             string(sub.Status),
		PlanCode:           planCode,
		CurrentPeriodStart: currentPeriodStart,
		CurrentPeriodEnd:   currentPeriodEnd,
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
	}

	if sub.TrialEnd > 0 {
		t := time.Unix(sub.TrialEnd, 0)
		result.TrialEnd = &t
	}
	if sub.CancelAt > 0 {
		t := time.Unix(sub.CancelAt, 0)
		result.CancelAt = &t
	}

	return result, nil
}

func (c *stripeClientImpl) GetSubscription(ctx context.Context, subscriptionID string) (*domain.StripeSubscription, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_subscription")
	defer span.End()

	params := &stripe.SubscriptionParams{}
	params.AddExpand("items.data.price.product")

	sub, err := subscription.Get(subscriptionID, params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get subscription %s: %w", subscriptionID, err)
	}

	var planCode string
	var currentPeriodStart time.Time
	var currentPeriodEnd time.Time
	if len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.Price != nil && item.Price.Product != nil {
			planCode = item.Price.Product.Metadata["plan_code"]
		}
		if item.CurrentPeriodStart > 0 {
			currentPeriodStart = time.Unix(item.CurrentPeriodStart, 0)
		}
		if item.CurrentPeriodEnd > 0 {
			currentPeriodEnd = time.Unix(item.CurrentPeriodEnd, 0)
		}
	}

	result := &domain.StripeSubscription{
		ID:                 sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             string(sub.Status),
		PlanCode:           planCode,
		CurrentPeriodStart: currentPeriodStart,
		CurrentPeriodEnd:   currentPeriodEnd,
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
	}

	if sub.TrialEnd > 0 {
		t := time.Unix(sub.TrialEnd, 0)
		result.TrialEnd = &t
	}
	if sub.CancelAt > 0 {
		t := time.Unix(sub.CancelAt, 0)
		result.CancelAt = &t
	}

	return result, nil
}

func (c *stripeClientImpl) CreateCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*domain.StripeCustomer, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_customer")
	defer span.End()

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	params.IdempotencyKey = stripe.String(idempotencyKey)
	for k, v := range metadata {
		params.AddMetadata(k, v)
	}

	cust, err := customer.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return &domain.StripeCustomer{ID: cust.ID}, nil
}

func (c *stripeClientImpl) GetOrCreateProduct(ctx context.Context, planCode, planName, idempotencyKey string) (*stripe.Product, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_or_create_product")
	defer span.End()

	searchParams := &stripe.ProductSearchParams{}
	searchParams.Query = fmt.Sprintf("metadata['plan_code']:'%s'", planCode)
	iter := product.Search(searchParams)
	if iter.Next() {
		return iter.Product(), nil
	}

	params := &stripe.ProductParams{
		Name: stripe.String(planName),
	}
	params.IdempotencyKey = stripe.String(idempotencyKey)
	params.AddMetadata("plan_code", planCode)

	prod, err := product.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create Stripe product: %w", err)
	}

	return prod, nil
}

func (c *stripeClientImpl) GetOrCreatePrice(ctx context.Context, productID string, unitAmount int64, planCode, idempotencyKey string) (*stripe.Price, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_or_create_price")
	defer span.End()

	lookupKey := fmt.Sprintf("reg_price_%s", planCode)
	listParams := &stripe.PriceListParams{
		LookupKeys: []*string{stripe.String(lookupKey)},
	}
	iter := price.List(listParams)
	if iter.Next() {
		return iter.Price(), nil
	}

	params := &stripe.PriceParams{
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(unitAmount),
		Currency:   stripe.String(string(stripe.CurrencyUSD)),
		Recurring: &stripe.PriceRecurringParams{
			Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
		},
		LookupKey: stripe.String(lookupKey),
	}
	params.IdempotencyKey = stripe.String(idempotencyKey)

	p, err := price.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create Stripe price: %w", err)
	}

	return p, nil
}

func (c *stripeClientImpl) CreateCheckoutSession(ctx context.Context, input domain.StripeCheckoutSessionInput) (*domain.StripeCheckoutSession, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_checkout_session")
	defer span.End()

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(input.CustomerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		UIMode:   stripe.String(string(stripe.CheckoutSessionUIModeCustom)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(input.PriceID),
				Quantity: stripe.Int64(input.Quantity),
			},
		},
		ReturnURL: stripe.String(input.ReturnURL),
	}
	params.IdempotencyKey = stripe.String(input.IdempotencyKey)

	if input.TrialDays > 0 {
		params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(input.TrialDays),
		}
	}

	sess, err := session.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create Stripe checkout session: %w", err)
	}

	return &domain.StripeCheckoutSession{
		ID:           sess.ID,
		ClientSecret: sess.ClientSecret,
	}, nil
}

func (c *stripeClientImpl) GetCheckoutSession(ctx context.Context, sessionID string) (*domain.StripeCheckoutSessionStatus, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_checkout_session")
	defer span.End()

	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("subscription")

	sess, err := session.Get(sessionID, params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get checkout session %s: %w", sessionID, err)
	}

	result := &domain.StripeCheckoutSessionStatus{
		Status: string(sess.Status),
	}
	if sess.Subscription != nil {
		result.SubscriptionID = sess.Subscription.ID
	}
	if sess.Customer != nil {
		result.CustomerID = sess.Customer.ID
	}

	return result, nil
}

func (c *stripeClientImpl) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (*domain.StripeBillingPortalSession, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_billing_portal_session")
	defer span.End()

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create billing portal session: %w", err)
	}

	return &domain.StripeBillingPortalSession{URL: sess.URL}, nil
}

func (c *stripeClientImpl) ListSubscriptions(ctx context.Context, customerID string) ([]*stripe.Subscription, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.list_subscriptions")
	defer span.End()

	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}
	params.AddExpand("data.items.data.price")

	var subs []*stripe.Subscription
	iter := subscription.List(params)
	for iter.Next() {
		sub := iter.Subscription()
		// Include active and trialing subscriptions
		if sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing {
			subs = append(subs, sub)
		}
	}
	if err := iter.Err(); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list subscriptions for customer %s: %w", customerID, err)
	}

	return subs, nil
}

func (c *stripeClientImpl) GetCustomerBalance(ctx context.Context, customerID string) (int64, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_customer_balance")
	defer span.End()

	cust, err := customer.Get(customerID, nil)
	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get customer %s: %w", customerID, err)
	}

	return cust.Balance, nil
}

func (c *stripeClientImpl) CreateInvoicePreview(ctx context.Context, customerID string, subscriptionID string, items []*stripe.InvoiceCreatePreviewSubscriptionDetailsItemParams) (*stripe.Invoice, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_invoice_preview")
	defer span.End()

	params := &stripe.InvoiceCreatePreviewParams{
		Customer: stripe.String(customerID),
		SubscriptionDetails: &stripe.InvoiceCreatePreviewSubscriptionDetailsParams{
			Items: items,
		},
	}

	if subscriptionID != "" {
		params.Subscription = stripe.String(subscriptionID)
	}

	inv, err := invoice.CreatePreview(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create invoice preview: %w", err)
	}

	return inv, nil
}

func (c *stripeClientImpl) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.cancel_subscription")
	defer span.End()

	params := &stripe.SubscriptionCancelParams{}
	_, err := subscription.Cancel(subscriptionID, params)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to cancel subscription %s: %w", subscriptionID, err)
	}

	return nil
}

func (c *stripeClientImpl) UpdateSubscription(ctx context.Context, subscriptionID string, items []*stripe.SubscriptionItemsParams) (*stripe.Subscription, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.update_subscription")
	defer span.End()

	params := &stripe.SubscriptionParams{
		Items: items,
	}

	sub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to update subscription %s: %w", subscriptionID, err)
	}

	return sub, nil
}

func (c *stripeClientImpl) CreateHostedCheckoutSession(ctx context.Context, input domain.StripeHostedCheckoutInput) (*domain.StripeHostedCheckoutSession, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_hosted_checkout_session")
	defer span.End()

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(input.CustomerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(input.PriceID),
				Quantity: stripe.Int64(input.Quantity),
			},
		},
		SuccessURL: stripe.String(input.SuccessURL),
		CancelURL:  stripe.String(input.CancelURL),
	}

	sess, err := session.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create hosted checkout session: %w", err)
	}

	return &domain.StripeHostedCheckoutSession{
		ID:  sess.ID,
		URL: sess.URL,
	}, nil
}

func (c *stripeClientImpl) ListPaymentMethods(ctx context.Context, customerID string) ([]string, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.list_payment_methods")
	defer span.End()

	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
	}

	var ids []string
	iter := paymentmethod.List(params)
	for iter.Next() {
		ids = append(ids, iter.PaymentMethod().ID)
	}
	if err := iter.Err(); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list payment methods for customer %s: %w", customerID, err)
	}

	return ids, nil
}

func (c *stripeClientImpl) VerifyWebhookSignature(payload []byte, signature string) (*domain.StripeEvent, error) {
	slog.Info("verifying webhook signature",
		"payload_size", len(payload),
		"signature_preview", truncate(signature, 30),
		"secret_prefix", truncate(c.webhookSecret, 12),
	)

	event, err := webhook.ConstructEvent(payload, signature, c.webhookSecret)
	if err != nil {
		slog.Error("stripe webhook.ConstructEvent failed",
			"error", err.Error(),
			"payload_size", len(payload),
			"signature_preview", truncate(signature, 30),
		)
		return nil, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	var objectID string
	var rawObject map[string]any
	if err := json.Unmarshal(event.Data.Raw, &rawObject); err == nil {
		if id, ok := rawObject["id"].(string); ok {
			objectID = id
		}
	}

	return &domain.StripeEvent{
		ID:       event.ID,
		Type:     string(event.Type),
		ObjectID: objectID,
		Data:     event.Data.Raw,
	}, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
