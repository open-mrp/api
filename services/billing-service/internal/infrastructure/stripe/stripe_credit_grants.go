package stripe

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/billing/creditbalancesummary"
	"github.com/stripe/stripe-go/v84/billing/creditgrant"
)

// CreateCreditGrant grants prepaid billing credits to a customer for a purchased token pack. The grant is monetary (amountCents in USD), scoped to metered prices so it draws down against the plan's LLM-token rate card, and has no expiry so packs roll over indefinitely (per our pricing). category=paid marks it as purchased (not promotional). idempotencyKey guards against duplicate grants when a webhook is redelivered. Returns the Stripe credit grant id.
func (c *stripeClientImpl) CreateCreditGrant(ctx context.Context, customerID string, amountCents int64, name, idempotencyKey string) (string, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.create_credit_grant")
	defer span.End()

	params := &stripe.BillingCreditGrantParams{
		Customer: stripe.String(customerID),
		Name:     stripe.String(name),
		Category: stripe.String(string(stripe.BillingCreditGrantCategoryPaid)),
		Amount: &stripe.BillingCreditGrantAmountParams{
			Type: stripe.String(string(stripe.BillingCreditGrantAmountTypeMonetary)),
			Monetary: &stripe.BillingCreditGrantAmountMonetaryParams{
				Currency: stripe.String(string(stripe.CurrencyUSD)),
				Value:    stripe.Int64(amountCents),
			},
		},
		ApplicabilityConfig: &stripe.BillingCreditGrantApplicabilityConfigParams{
			Scope: &stripe.BillingCreditGrantApplicabilityConfigScopeParams{
				PriceType: stripe.String(string(stripe.BillingCreditGrantApplicabilityConfigScopePriceTypeMetered)),
			},
		},
	}
	params.Context = ctx
	if idempotencyKey != "" {
		params.IdempotencyKey = stripe.String(idempotencyKey)
	}

	grant, err := creditgrant.New(params)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to create credit grant: %w", err)
	}

	return grant.ID, nil
}

// GetCreditGrantBalanceCents returns the customer's available prepaid credit balance in cents, summed across all monetary credit grants scoped to metered usage. This is the balance the dashboard displays and the agent runner gates prepaid runs on; the authoritative burndown happens inside Stripe as metered token usage is reported. Returns 0 when the customer has no credit grants.
func (c *stripeClientImpl) GetCreditGrantBalanceCents(ctx context.Context, customerID string) (int64, error) {
	_, span := stripeClientTracer.Start(ctx, "stripe_client.get_credit_grant_balance")
	defer span.End()

	params := &stripe.BillingCreditBalanceSummaryParams{
		Customer: stripe.String(customerID),
		Filter: &stripe.BillingCreditBalanceSummaryFilterParams{
			Type: stripe.String("applicability_scope"),
			ApplicabilityScope: &stripe.BillingCreditBalanceSummaryFilterApplicabilityScopeParams{
				PriceType: stripe.String(string(stripe.BillingCreditGrantApplicabilityConfigScopePriceTypeMetered)),
			},
		},
	}
	params.Context = ctx

	summary, err := creditbalancesummary.Get(params)
	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get credit balance summary: %w", err)
	}

	var availableCents int64
	for _, b := range summary.Balances {
		if b.AvailableBalance != nil && b.AvailableBalance.Monetary != nil {
			availableCents += b.AvailableBalance.Monetary.Value
		}
	}

	return availableCents, nil
}
