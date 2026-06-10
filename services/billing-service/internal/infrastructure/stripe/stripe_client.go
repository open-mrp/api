package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/tracing"
	"github.com/stripe/stripe-go/v84"
	portalsession "github.com/stripe/stripe-go/v84/billingportal/session"
	"github.com/stripe/stripe-go/v84/customer"
	"github.com/stripe/stripe-go/v84/paymentintent"
	"github.com/stripe/stripe-go/v84/paymentmethod"
	"github.com/stripe/stripe-go/v84/setupintent"
	"github.com/stripe/stripe-go/v84/webhook"
)

type versionOverrideTransport struct {
	wrapped http.RoundTripper
}

func (t *versionOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Stripe-Version", constants.StripeAPIVersion)
	slog.Info("stripe outgoing request", // #nosec G706 -- values sanitized via sanitizeLogValue
		"method", sanitizeLogValue(req.Method),
		"url", sanitizeLogValue(req.URL.String()),
		"stripe_version", sanitizeLogValue(req.Header.Get("Stripe-Version")),
		"has_auth", req.Header.Get("Authorization") != "",
		"content_type", sanitizeLogValue(req.Header.Get("Content-Type")),
		"stripe_account", sanitizeLogValue(req.Header.Get("Stripe-Account")),
		"stripe_context", sanitizeLogValue(req.Header.Get("Stripe-Context")),
	)
	return t.wrapped.RoundTrip(req)
}

// sanitizeLogValue strips control characters (newlines, tabs, etc.) from a
// string before it is written to structured logs, preventing log injection.
func sanitizeLogValue(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

var stripeClientTracer = tracing.GetTracer("billing-service.stripe_client")

type ClientConfig struct {
	// WebhookSecret (required) is the Stripe webhook signing secret used to
	// verify incoming webhook payloads.
	WebhookSecret string

	// APIKey (required) is the Stripe secret API key.
	APIKey string // #nosec G117 - Config field populated from env var at startup
}

func (c *ClientConfig) validate() error {
	if c.WebhookSecret == "" {
		return fmt.Errorf("stripe client: webhook secret is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("stripe client: api key is required")
	}
	return nil
}

type stripeClientImpl struct {
	webhookSecret string
}

func NewStripeClient(cfg *ClientConfig) domain.StripeClient {
	if err := cfg.validate(); err != nil {
		panic(err)
	}

	stripe.Key = cfg.APIKey

	httpClient := &http.Client{
		Transport: &versionOverrideTransport{wrapped: http.DefaultTransport},
	}
	stripe.SetBackend(stripe.APIBackend, stripe.GetBackendWithConfig(
		stripe.APIBackend,
		&stripe.BackendConfig{
			HTTPClient: httpClient,
		},
	))

	return &stripeClientImpl{
		webhookSecret: cfg.WebhookSecret,
	}
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

func (c *stripeClientImpl) GetPricingPlan(ctx context.Context, pricingPlanID string) (*domain.StripePricingPlan, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_pricing_plan")
	defer span.End()

	resp, err := stripe.RawRequest(http.MethodGet,
		fmt.Sprintf("/v2/billing/pricing_plans/%s", pricingPlanID), "", nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to fetch pricing plan: %w", err)
	}
	var result struct {
		ID          string `json:"id"`
		LiveVersion string `json:"live_version"`
	}
	if err := json.Unmarshal(resp.RawJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pricing plan response: %w", err)
	}
	if result.LiveVersion == "" {
		return nil, fmt.Errorf("pricing plan %s has no live version", pricingPlanID)
	}

	plan := &domain.StripePricingPlan{
		ID:          result.ID,
		LiveVersion: result.LiveVersion,
	}

	// Fetch the version to get the license fee component ID
	versionResp, err := stripe.RawRequest(http.MethodGet,
		fmt.Sprintf("/v2/billing/pricing_plan_versions/%s", result.LiveVersion), "", nil)
	if err == nil {
		var version struct {
			Components []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"components"`
		}
		if parseErr := json.Unmarshal(versionResp.RawJSON, &version); parseErr == nil {
			for _, comp := range version.Components {
				if comp.Type == "license_fee" {
					plan.LicenseFeeComponentID = comp.ID
					break
				}
			}
		}
	}

	return plan, nil
}

func (c *stripeClientImpl) CreateBillingProfile(ctx context.Context, customerID, idempotencyKey string) (string, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_billing_profile")
	defer span.End()

	body := fmt.Sprintf(`{"customer":"%s"}`, customerID)
	resp, err := stripe.RawRequest(http.MethodPost, "/v2/billing/profiles", body, &stripe.RawParams{
		Params: stripe.Params{
			IdempotencyKey: stripe.String(idempotencyKey),
		},
	})
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to create billing profile: %w", err)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.RawJSON, &result); err != nil {
		return "", fmt.Errorf("failed to parse billing profile response: %w", err)
	}

	return result.ID, nil
}

func (c *stripeClientImpl) CreateBillingCadence(ctx context.Context, billingProfileID, idempotencyKey string) (string, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_billing_cadence")
	defer span.End()

	body := fmt.Sprintf(`{
		"payer":{"billing_profile":"%s"},
		"billing_cycle":{"type":"month","interval_count":1}
	}`, billingProfileID)
	resp, err := stripe.RawRequest(http.MethodPost, "/v2/billing/cadences", body, &stripe.RawParams{
		Params: stripe.Params{
			IdempotencyKey: stripe.String(idempotencyKey),
		},
	})
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to create billing cadence: %w", err)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.RawJSON, &result); err != nil {
		return "", fmt.Errorf("failed to parse billing cadence response: %w", err)
	}

	return result.ID, nil
}

func (c *stripeClientImpl) CreateBillingIntent(ctx context.Context, cadenceID string, actions []domain.BillingIntentAction, idempotencyKey string) (string, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_billing_intent")
	defer span.End()

	actionsJSON := buildActionsJSON(actions)
	body := fmt.Sprintf(`{
		"currency":"usd",
		"cadence":"%s",
		"actions":%s
	}`, cadenceID, actionsJSON)

	var rawParams *stripe.RawParams
	if idempotencyKey != "" {
		rawParams = &stripe.RawParams{
			Params: stripe.Params{
				IdempotencyKey: stripe.String(idempotencyKey),
			},
		}
	}

	resp, err := stripe.RawRequest(http.MethodPost, "/v2/billing/intents", body, rawParams)
	if err != nil {
		span.RecordError(err)
		if conflictID, ok := parseBillingIntentConflict(err); ok {
			return "", &domain.ErrBillingIntentConflict{ConflictingIntentID: conflictID, Err: err}
		}
		return "", fmt.Errorf("failed to create billing intent: %w", err)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.RawJSON, &result); err != nil {
		return "", fmt.Errorf("failed to parse billing intent response: %w", err)
	}

	return result.ID, nil
}

func buildComponentConfigurations(configs []domain.ComponentConfiguration) []any {
	if len(configs) == 0 {
		return []any{}
	}
	result := make([]any, len(configs))
	for i, c := range configs {
		result[i] = map[string]any{
			"pricing_plan_component": c.PricingPlanComponentID,
			"quantity":               c.Quantity,
		}
	}
	return result
}

func buildActionsJSON(actions []domain.BillingIntentAction) string {
	type subscribeDetails struct {
		PricingPlan             string `json:"pricing_plan"`
		PricingPlanVersion      string `json:"pricing_plan_version"`
		ComponentConfigurations []any  `json:"component_configurations"`
	}
	type subscribeAction struct {
		Type                           string           `json:"type"`
		PricingPlanSubscriptionDetails subscribeDetails `json:"pricing_plan_subscription_details"`
	}

	type modifyDetails struct {
		PricingPlanSubscription string `json:"pricing_plan_subscription"`
		NewPricingPlan          string `json:"new_pricing_plan"`
		NewPricingPlanVersion   string `json:"new_pricing_plan_version"`
		ComponentConfigurations []any  `json:"component_configurations"`
	}
	type modifyAction struct {
		Type                           string        `json:"type"`
		PricingPlanSubscriptionDetails modifyDetails `json:"pricing_plan_subscription_details"`
	}

	type deactivateDetails struct {
		PricingPlanSubscription string `json:"pricing_plan_subscription"`
	}
	type deactivateAction struct {
		Type                           string            `json:"type"`
		PricingPlanSubscriptionDetails deactivateDetails `json:"pricing_plan_subscription_details"`
	}

	type actionJSON struct {
		Type       string            `json:"type"`
		Subscribe  *subscribeAction  `json:"subscribe,omitempty"`
		Modify     *modifyAction     `json:"modify,omitempty"`
		Deactivate *deactivateAction `json:"deactivate,omitempty"`
	}

	var jsonActions []actionJSON
	for _, a := range actions {
		aj := actionJSON{Type: a.Type}
		compConfigs := buildComponentConfigurations(a.ComponentConfigurations)
		switch a.Type {
		case "subscribe":
			aj.Subscribe = &subscribeAction{
				Type: "pricing_plan_subscription_details",
				PricingPlanSubscriptionDetails: subscribeDetails{
					PricingPlan:             a.PricingPlanID,
					PricingPlanVersion:      a.PricingPlanVersion,
					ComponentConfigurations: compConfigs,
				},
			}
		case "modify":
			aj.Modify = &modifyAction{
				Type: "pricing_plan_subscription_details",
				PricingPlanSubscriptionDetails: modifyDetails{
					PricingPlanSubscription: a.SubscriptionID,
					NewPricingPlan:          a.PricingPlanID,
					NewPricingPlanVersion:   a.PricingPlanVersion,
					ComponentConfigurations: compConfigs,
				},
			}
		case "deactivate":
			aj.Deactivate = &deactivateAction{
				Type: "pricing_plan_subscription_details",
				PricingPlanSubscriptionDetails: deactivateDetails{
					PricingPlanSubscription: a.SubscriptionID,
				},
			}
		}
		jsonActions = append(jsonActions, aj)
	}

	b, _ := json.Marshal(jsonActions)
	return string(b)
}

func (c *stripeClientImpl) ReserveBillingIntent(ctx context.Context, intentID string) (*domain.BillingIntentReservation, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.reserve_billing_intent")
	defer span.End()

	_, err := stripe.RawRequest(http.MethodPost,
		fmt.Sprintf("/v2/billing/intents/%s/reserve", intentID), "", nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to reserve billing intent: %w", err)
	}

	// Fetch the reserved intent to get amount_details.total — the reserve
	// response itself does not include the total at the top level.
	intentResp, err := stripe.RawRequest(http.MethodGet,
		fmt.Sprintf("/v2/billing/intents/%s", intentID), "", nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to fetch reserved billing intent: %w", err)
	}

	var result struct {
		AmountDetails struct {
			Total json.Number `json:"total"`
		} `json:"amount_details"`
	}
	if err := json.Unmarshal(intentResp.RawJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse reserved billing intent: %w", err)
	}

	total, err := result.AmountDetails.Total.Int64()
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount_details.total %q: %w", result.AmountDetails.Total, err)
	}

	reservation := &domain.BillingIntentReservation{
		IntentID:  intentID,
		NetAmount: total,
	}

	return reservation, nil
}

func (c *stripeClientImpl) CreatePaymentIntent(ctx context.Context, amountCents int64, currency, customerID, returnURL string) (string, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_payment_intent")
	defer span.End()

	// Look up the customer's default payment method. If none is set on
	// invoice_settings, fall back to the first attached payment method.
	cust, err := customer.Get(customerID, nil)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to fetch customer for payment method: %w", err)
	}

	var paymentMethodID string
	if cust.InvoiceSettings != nil && cust.InvoiceSettings.DefaultPaymentMethod != nil {
		paymentMethodID = cust.InvoiceSettings.DefaultPaymentMethod.ID
	}
	if paymentMethodID == "" {
		pmList := paymentmethod.List(&stripe.PaymentMethodListParams{
			Customer: stripe.String(customerID),
			Type:     stripe.String("card"),
		})
		if pmList.Next() {
			paymentMethodID = pmList.PaymentMethod().ID
		}
	}
	if paymentMethodID == "" {
		return "", fmt.Errorf("customer %s has no payment method attached", customerID)
	}

	params := &stripe.PaymentIntentParams{
		Amount:        new(amountCents),
		Currency:      stripe.String(currency),
		Customer:      stripe.String(customerID),
		PaymentMethod: stripe.String(paymentMethodID),
		Confirm:       new(true),
		ReturnURL:     stripe.String(returnURL),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to create payment intent: %w", err)
	}

	return pi.ID, nil
}

func (c *stripeClientImpl) CommitBillingIntent(ctx context.Context, intentID string, paymentIntentID *string, cadenceID string) (*domain.BillingIntentCommitResult, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.commit_billing_intent")
	defer span.End()

	body := "{}"
	if paymentIntentID != nil {
		body = fmt.Sprintf(`{"payment_intent":"%s"}`, *paymentIntentID)
	}

	resp, err := stripe.RawRequest(http.MethodPost,
		fmt.Sprintf("/v2/billing/intents/%s/commit", intentID), body, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to commit billing intent: %w", err)
	}

	// Try to extract subscription IDs from the commit response.
	ids := extractSubscriptionIDs(resp.RawJSON)

	// Fallback: fetch the committed intent if commit response didn't include them.
	if len(ids) == 0 {
		intentResp, getErr := stripe.RawRequest(http.MethodGet,
			fmt.Sprintf("/v2/billing/intents/%s", intentID), "", nil)
		if getErr == nil {
			ids = extractSubscriptionIDs(intentResp.RawJSON)
		}
	}

	// Fallback: list subscriptions by cadence.
	if len(ids) == 0 && cadenceID != "" {
		listResp, listErr := stripe.RawRequest(http.MethodGet,
			fmt.Sprintf("/v2/billing/pricing_plan_subscriptions?billing_cadence=%s", cadenceID), "", nil)
		if listErr == nil {
			var listResult struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if json.Unmarshal(listResp.RawJSON, &listResult) == nil {
				for _, sub := range listResult.Data {
					if sub.ID != "" {
						ids = append(ids, sub.ID)
					}
				}
			}
		}
	}

	return &domain.BillingIntentCommitResult{PricingPlanSubscriptionIDs: ids}, nil
}

// extractSubscriptionIDs parses subscription IDs from a billing intent response.
func extractSubscriptionIDs(raw []byte) []string {
	var resp struct {
		Actions []struct {
			Subscribe *struct {
				PricingPlanSubscriptionDetails struct {
					PricingPlanSubscription string `json:"pricing_plan_subscription"`
				} `json:"pricing_plan_subscription_details"`
			} `json:"subscribe"`
			Modify *struct {
				PricingPlanSubscriptionDetails struct {
					PricingPlanSubscription string `json:"pricing_plan_subscription"`
				} `json:"pricing_plan_subscription_details"`
			} `json:"modify"`
		} `json:"actions"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return nil
	}
	var ids []string
	for _, action := range resp.Actions {
		if action.Subscribe != nil && action.Subscribe.PricingPlanSubscriptionDetails.PricingPlanSubscription != "" {
			ids = append(ids, action.Subscribe.PricingPlanSubscriptionDetails.PricingPlanSubscription)
		}
		if action.Modify != nil && action.Modify.PricingPlanSubscriptionDetails.PricingPlanSubscription != "" {
			ids = append(ids, action.Modify.PricingPlanSubscriptionDetails.PricingPlanSubscription)
		}
	}
	return ids
}

func (c *stripeClientImpl) VoidBillingIntent(ctx context.Context, intentID string) error {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.void_billing_intent")
	defer span.End()

	_, err := stripe.RawRequest(http.MethodPost,
		fmt.Sprintf("/v2/billing/intents/%s/cancel", intentID), "", nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to void billing intent: %w", err)
	}

	return nil
}

// parseBillingIntentConflict checks if a Stripe error indicates a billing intent
// conflict and extracts the conflicting intent ID from the message.
func parseBillingIntentConflict(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	msg := err.Error()
	const marker = "reserved by billing intent "
	_, after, ok := strings.Cut(msg, marker)
	if !ok {
		return "", false
	}
	rest := after
	// The intent ID ends at the first character that isn't alphanumeric or underscore.
	end := strings.IndexFunc(rest, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	if end < 0 {
		end = len(rest)
	}
	intentID := rest[:end]
	if !strings.HasPrefix(intentID, "bilint_") {
		return "", false
	}
	return intentID, true
}

func (c *stripeClientImpl) CreateSetupIntent(ctx context.Context, customerID, idempotencyKey string) (*domain.StripeSetupIntent, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_setup_intent")
	defer span.End()

	params := &stripe.SetupIntentParams{
		Customer:           stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	params.IdempotencyKey = stripe.String(idempotencyKey)

	si, err := setupintent.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create Setup Intent: %w", err)
	}

	return &domain.StripeSetupIntent{
		ID:           si.ID,
		ClientSecret: si.ClientSecret,
		Status:       string(si.Status),
	}, nil
}

func (c *stripeClientImpl) GetSetupIntent(ctx context.Context, setupIntentID string) (*domain.StripeSetupIntent, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_setup_intent")
	defer span.End()

	si, err := setupintent.Get(setupIntentID, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get Setup Intent: %w", err)
	}

	result := &domain.StripeSetupIntent{
		ID:           si.ID,
		ClientSecret: si.ClientSecret,
		Status:       string(si.Status),
	}
	if si.PaymentMethod != nil {
		result.PaymentMethodID = &si.PaymentMethod.ID
	}

	return result, nil
}

func (c *stripeClientImpl) VerifyWebhookSignature(payload []byte, signature string) (*domain.StripeEvent, error) {
	slog.Info("verifying webhook signature",
		"payload_size", len(payload),
		"signature_preview", truncate(signature, 30),
	)

	// Use ValidatePayload instead of ConstructEvent so that both v1 events
	// ("object":"event") and v2 event notifications ("object":"v2.core.event")
	// pass signature verification. ConstructEvent rejects v2 payloads.
	if err := webhook.ValidatePayload(payload, signature, c.webhookSecret); err != nil {
		slog.Error("stripe webhook signature validation failed",
			"error", err.Error(),
			"payload_size", len(payload),
			"signature_preview", truncate(signature, 30),
		)
		return nil, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	// Parse the event envelope to extract ID, type, and object data.
	// This works for both v1 and v2 event payloads.
	var envelope struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		// v1 events embed data.object inline
		Data json.RawMessage `json:"data"`
		// v2 thin events reference a related_object instead
		RelatedObject *struct {
			ID string `json:"id"`
		} `json:"related_object,omitempty"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event envelope: %w", err)
	}

	// Extract object ID from the payload.
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
		Data:     payload, // pass full raw payload; consumer re-parses envelope
	}, nil
}

func (c *stripeClientImpl) FetchObject(ctx context.Context, objectURL string) ([]byte, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.fetch_object")
	defer span.End()

	resp, err := stripe.RawRequest(http.MethodGet, objectURL, "", nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to fetch Stripe object at %s: %w", objectURL, err)
	}

	return resp.RawJSON, nil
}

func (c *stripeClientImpl) ReportMeterEvent(ctx context.Context, eventName, stripeCustomerID string, value int, idempotencyKey string) error {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.report_meter_event")
	defer span.End()

	body := fmt.Sprintf(`{"event_name":%q,"payload":{"stripe_customer_id":%q,"value":"%d"}}`,
		eventName, stripeCustomerID, value)

	_, err := stripe.RawRequest(http.MethodPost, "/v2/billing/meter_events", body, &stripe.RawParams{
		Params: stripe.Params{
			IdempotencyKey: stripe.String(idempotencyKey),
		},
	})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to report meter event: %w", err)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
