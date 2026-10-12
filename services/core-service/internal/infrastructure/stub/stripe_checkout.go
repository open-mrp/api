package stub

import (
	"context"
	"encoding/json"

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

func (s *stripeCheckoutClient) ConstructWebhookEvent(payload []byte, _, _ string) (*domain.StripeWebhookEvent, *domain.StripePaymentIntent, *apierror.APIError) {
	var event struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return &domain.StripeWebhookEvent{}, &domain.StripePaymentIntent{}, nil
	}
	webhookEvent := &domain.StripeWebhookEvent{ID: event.ID, Type: event.Type, RawJSON: event.Data.Object}
	var pi struct {
		ID                 string            `json:"id"`
		Amount             int64             `json:"amount"`
		PaymentMethodTypes []string          `json:"payment_method_types"`
		Metadata           map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(event.Data.Object, &pi); err != nil {
		return webhookEvent, nil, nil
	}
	return webhookEvent, &domain.StripePaymentIntent{ID: pi.ID, Amount: pi.Amount, PaymentMethodTypes: pi.PaymentMethodTypes, Metadata: pi.Metadata}, nil
}

func (s *stripeCheckoutClient) ListPayoutPaymentIntentIDs(_ context.Context, _ string) ([]string, *apierror.APIError) {
	return nil, nil
}
