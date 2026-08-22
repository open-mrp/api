package stub

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// StripeCheckoutClientFactory is a no-op factory for use in test mode.
type StripeCheckoutClientFactory struct{}

func (f *StripeCheckoutClientFactory) Build(_ string) domain.StripeCheckoutClient {
	return &stripeCheckoutClient{}
}

type stripeCheckoutClient struct{}

func (s *stripeCheckoutClient) CreateOneTimeCheckoutSession(_ context.Context, _ domain.CreateCheckoutSessionParams) (*domain.StripeCheckoutSession, *apierror.APIError) {
	return &domain.StripeCheckoutSession{URL: "https://stub.local/checkout"}, nil
}

func (s *stripeCheckoutClient) CreateEmbeddedCheckoutSession(_ context.Context, _ domain.CreateEmbeddedCheckoutSessionParams) (*domain.StripeEmbeddedCheckoutSession, *apierror.APIError) {
	return &domain.StripeEmbeddedCheckoutSession{ClientSecret: "stub_secret"}, nil
}

func (s *stripeCheckoutClient) CreateStripeCustomer(_ context.Context, _ domain.CreateStripeCustomerParams) (*domain.StripeCustomer, *apierror.APIError) {
	return &domain.StripeCustomer{ID: "cus_stub"}, nil
}

func (s *stripeCheckoutClient) UpdateStripeCustomer(_ context.Context, _ domain.UpdateStripeCustomerParams) *apierror.APIError {
	return nil
}

func (s *stripeCheckoutClient) ConstructWebhookEvent(_ []byte, _, _ string) (*domain.StripeWebhookEvent, *domain.StripePaymentIntent, *apierror.APIError) {
	return &domain.StripeWebhookEvent{}, &domain.StripePaymentIntent{}, nil
}

func (s *stripeCheckoutClient) ListPayoutPaymentIntentIDs(_ context.Context, _ string) ([]string, *apierror.APIError) {
	return nil, nil
}
