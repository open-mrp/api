package stub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
)

// StripeClient is a no-op StripeClient implementation for use in test mode.
type StripeClient struct{}

// StubWebhookSignature is the signature the stub accepts. It stands in for a real HMAC so a
// test can exercise both the accepted and the rejected branch; anything else is refused.
const StubWebhookSignature = "t=0,v1=e2e-stub-signature"

// VerifyWebhookSignature stands in for Stripe's HMAC check. It refuses anything but the known
// stub signature, then parses the envelope exactly as the real client does — so the routing and
// enqueue decisions downstream are driven by the payload's real event type rather than by an
// empty event that never matches anything.
func (s *StripeClient) VerifyWebhookSignature(payload []byte, signature string) (*domain.StripeEvent, error) {
	if signature != StubWebhookSignature {
		return nil, fmt.Errorf("failed to verify webhook signature: no signatures found matching the expected signature for payload")
	}

	var envelope struct {
		ID            string          `json:"id"`
		Type          string          `json:"type"`
		Data          json.RawMessage `json:"data"`
		RelatedObject *struct {
			ID string `json:"id"`
		} `json:"related_object,omitempty"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event envelope: %w", err)
	}

	var objectID string
	if envelope.RelatedObject != nil {
		objectID = envelope.RelatedObject.ID
	} else if len(envelope.Data) > 0 {
		var data struct {
			Object struct {
				ID string `json:"id"`
			} `json:"object"`
		}
		if json.Unmarshal(envelope.Data, &data) == nil {
			objectID = data.Object.ID
		}
	}

	return &domain.StripeEvent{
		ID:       envelope.ID,
		Type:     envelope.Type,
		ObjectID: objectID,
		Data:     payload,
	}, nil
}

func (s *StripeClient) CreateCustomer(_ context.Context, _, _, _ string, _ map[string]string) (*domain.StripeCustomer, error) {
	return &domain.StripeCustomer{ID: fmt.Sprintf("cus_stub_%d", stubSequence.Add(1))}, nil
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

// stubSequence gives each stubbed Stripe object a distinct ID. A constant ID would make two
// registrations indistinguishable, so a session confirming against another session's setup
// intent would look identical to one confirming against its own.
var stubSequence atomic.Uint64

// CreateSetupIntent mints an intent in the state Stripe creates one in: awaiting a card. The
// client secret follows Stripe's `<id>_secret_<key>` shape, because that is what the browser is
// given and what the intent ID is read back out of.
func (s *StripeClient) CreateSetupIntent(_ context.Context, _, _ string) (*domain.StripeSetupIntent, error) {
	id := fmt.Sprintf("seti_stub_%d", stubSequence.Add(1))
	return &domain.StripeSetupIntent{
		ID:           id,
		ClientSecret: id + "_secret_stub",
		Status:       "requires_payment_method",
	}, nil
}

// GetSetupIntent reports the intent as succeeded. The server only ever reads an intent after
// the browser has confirmed the card against it, so that is the state this call observes; an
// intent still awaiting payment would leave the confirm path permanently unreachable in tests.
func (s *StripeClient) GetSetupIntent(_ context.Context, setupIntentID string) (*domain.StripeSetupIntent, error) {
	paymentMethodID := "pm_stub"
	return &domain.StripeSetupIntent{
		ID:              setupIntentID,
		ClientSecret:    setupIntentID + "_secret_stub",
		Status:          "succeeded",
		PaymentMethodID: &paymentMethodID,
	}, nil
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
