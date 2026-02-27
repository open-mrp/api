package service

import (
	"context"
	"fmt"
	"math"

	"github.com/augno/api/services/billing-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var checkoutSvcTracer = tracing.GetTracer("billing-service.checkout_service")

// CheckoutSvcConfig holds the dependencies for the checkout service.
type CheckoutSvcConfig struct {
	StripeClient   domain.StripeClient
	BillingSvc     domain.BillingSvc
	PublishableKey string
}

type checkoutSvcImpl struct {
	stripeClient   domain.StripeClient
	billingSvc     domain.BillingSvc
	publishableKey string
}

func (c *CheckoutSvcConfig) validate() error {
	if c.StripeClient == nil {
		return fmt.Errorf("checkout service: stripe client is required")
	}
	if c.BillingSvc == nil {
		return fmt.Errorf("checkout service: billing svc is required")
	}
	if c.PublishableKey == "" {
		return fmt.Errorf("checkout service: publishable key is required")
	}
	return nil
}

// NewCheckoutSvc creates a new CheckoutSvc that orchestrates Stripe customer
// and checkout session creation.
func NewCheckoutSvc(config *CheckoutSvcConfig) domain.CheckoutSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &checkoutSvcImpl{
		stripeClient:   config.StripeClient,
		billingSvc:     config.BillingSvc,
		publishableKey: config.PublishableKey,
	}
}

func (s *checkoutSvcImpl) CreateCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*domain.StripeCustomer, *apierror.APIError) {
	ctx, span := checkoutSvcTracer.Start(ctx, "service.checkout.create_customer")
	defer span.End()

	cust, err := s.stripeClient.CreateCustomer(ctx, email, name, idempotencyKey, metadata)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to create Stripe customer."))
	}

	return cust, nil
}

func (s *checkoutSvcImpl) CreateCheckoutSession(ctx context.Context, input domain.CreateCheckoutSessionInput) (*domain.CreateCheckoutSessionResult, *apierror.APIError) {
	ctx, span := checkoutSvcTracer.Start(ctx, "service.checkout.create_checkout_session")
	defer span.End()

	// Fetch plan from billing service
	plan, apiErr := s.billingSvc.GetPlanByCode(ctx, input.PlanCode)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Calculate unit amount in cents
	var unitAmount int64
	if plan.PricePerMonth != nil && *plan.PricePerMonth > 0 {
		unitAmount = int64(math.Round(*plan.PricePerMonth * 100))
	} else {
		unitAmount = int64(math.Round(plan.PricePerSeat * 100))
	}

	// Get or create Stripe product
	productIdempotencyKey := fmt.Sprintf("reg_product_%s", input.PlanCode)
	prod, err := s.stripeClient.GetOrCreateProduct(ctx, input.PlanCode, plan.Name, productIdempotencyKey)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to get or create Stripe product."))
	}

	// Get or create Stripe price
	priceIdempotencyKey := fmt.Sprintf("reg_price_%s", input.PlanCode)
	stripePrice, err := s.stripeClient.GetOrCreatePrice(ctx, prod.ID, unitAmount, input.PlanCode, priceIdempotencyKey)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to get or create Stripe price."))
	}

	// Determine quantity from seat minimum
	var quantity int64 = 1
	if plan.SeatMinimum != nil && *plan.SeatMinimum > 1 {
		quantity = int64(*plan.SeatMinimum)
	}

	// Create checkout session
	sess, err := s.stripeClient.CreateCheckoutSession(ctx, domain.StripeCheckoutSessionInput{
		CustomerID:     input.CustomerID,
		PriceID:        stripePrice.ID,
		Quantity:       quantity,
		TrialDays:      30,
		IdempotencyKey: input.IdempotencyKey,
		ReturnURL:      input.ReturnURL,
	})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to create Stripe checkout session."))
	}

	return &domain.CreateCheckoutSessionResult{
		SessionID:      sess.ID,
		ClientSecret:   sess.ClientSecret,
		PublishableKey: s.publishableKey,
	}, nil
}

func (s *checkoutSvcImpl) GetCheckoutSessionStatus(ctx context.Context, input domain.GetCheckoutSessionStatusInput) (*domain.GetCheckoutSessionStatusResult, *apierror.APIError) {
	ctx, span := checkoutSvcTracer.Start(ctx, "service.checkout.get_checkout_session_status")
	defer span.End()

	status, err := s.stripeClient.GetCheckoutSession(ctx, input.CheckoutSessionID)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to get checkout session status."))
	}

	return &domain.GetCheckoutSessionStatusResult{
		Status:         status.Status,
		SubscriptionID: status.SubscriptionID,
		CustomerID:     status.CustomerID,
	}, nil
}
