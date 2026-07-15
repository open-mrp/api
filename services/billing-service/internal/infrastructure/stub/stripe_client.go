package stub

import (
	"context"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
)

// StripeClient is a no-op StripeClient implementation for use in test mode.
type StripeClient struct{}

func (s *StripeClient) VerifyWebhookSignature(_ []byte, _ string) (*domain.StripeEvent, error) {
	return &domain.StripeEvent{}, nil
}

func (s *StripeClient) CreateCustomer(_ context.Context, _, _, _ string, _ map[string]string) (*domain.StripeCustomer, error) {
	return &domain.StripeCustomer{ID: "cus_stub"}, nil
}

func (s *StripeClient) CreateBillingPortalSession(_ context.Context, _, _ string) (*domain.StripeBillingPortalSession, error) {
	return &domain.StripeBillingPortalSession{URL: "https://stub.local/portal"}, nil
}

func (s *StripeClient) GetPricingPlan(_ context.Context, _ string) (*domain.StripePricingPlan, error) {
	return &domain.StripePricingPlan{}, nil
}

func (s *StripeClient) CreateBillingProfile(_ context.Context, _, _ string) (string, error) {
	return "bp_stub", nil
}

func (s *StripeClient) CreateBillingCadence(_ context.Context, _, _ string) (string, error) {
	return "bc_stub", nil
}

func (s *StripeClient) CreateBillingIntent(_ context.Context, _ string, _ []domain.BillingIntentAction, _ string) (string, error) {
	return "bi_stub", nil
}

func (s *StripeClient) ReserveBillingIntent(_ context.Context, _ string) (*domain.BillingIntentReservation, error) {
	return &domain.BillingIntentReservation{}, nil
}

func (s *StripeClient) CreatePaymentIntent(_ context.Context, _ int64, _, _, _ string) (string, error) {
	return "pi_stub", nil
}

func (s *StripeClient) CommitBillingIntent(_ context.Context, _ string, _ *string, _ string) (*domain.BillingIntentCommitResult, error) {
	return &domain.BillingIntentCommitResult{}, nil
}

func (s *StripeClient) VoidBillingIntent(_ context.Context, _ string) error {
	return nil
}

func (s *StripeClient) FetchObject(_ context.Context, _ string) ([]byte, error) {
	return []byte("{}"), nil
}

func (s *StripeClient) CreateSetupIntent(_ context.Context, _, _ string) (*domain.StripeSetupIntent, error) {
	return &domain.StripeSetupIntent{ID: "seti_stub", ClientSecret: "stub_secret", Status: "requires_payment_method"}, nil
}

func (s *StripeClient) GetSetupIntent(_ context.Context, _ string) (*domain.StripeSetupIntent, error) {
	return &domain.StripeSetupIntent{ID: "seti_stub", ClientSecret: "stub_secret", Status: "requires_payment_method"}, nil
}

func (s *StripeClient) ReportMeterEvent(_ context.Context, _, _ string, _ int, _ string) error {
	return nil
}

func (s *StripeClient) GetAgentTokenSpendCents(_ context.Context, _, _ string, _ time.Time) (int64, error) {
	return 0, nil
}

func (s *StripeClient) GetRateCardTokenRates(_ context.Context, _ string) ([]domain.TokenRate, error) {
	return nil, nil
}

func (s *StripeClient) CreateCreditGrant(_ context.Context, _ string, _ int64, _, _ string) (string, error) {
	return "", nil
}

func (s *StripeClient) GetCreditGrantBalanceCents(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
