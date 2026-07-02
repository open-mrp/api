package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v84"
)

// tokenRateKey identifies a rate card rate and a usage bucket by the (model, token_type) dimensions the token meter is segmented on.
type tokenRateKey struct {
	model     string
	tokenType string
}

// GetAgentTokenSpendCents reconstructs the marked-up cost the customer will be billed for metered LLM tokens since the given time. Stripe does not expose a preview/upcoming invoice for v2 pricing plan subscriptions, so we compute the same figure Stripe bills: cost = Σ over (model, token_type) of usage_quantity × rate_card_unit_amount. Rate card unit_amount is denominated in cents per token and already includes the plan's markup, so no local price table is involved.
func (c *stripeClientImpl) GetAgentTokenSpendCents(ctx context.Context, customerID, rateCardID string, since time.Time) (int64, error) {
	ctx, span := stripeClientTracer.Start(ctx, "stripe_client.get_agent_token_spend_cents")
	defer span.End()

	rates, meterID, err := c.fetchRateCardTokenRates(rateCardID)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	if meterID == "" || len(rates) == 0 {
		return 0, nil
	}

	usage, err := c.fetchMeterTokenUsage(customerID, meterID, since)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	var totalCents float64
	for key, quantity := range usage {
		unitAmountCents, ok := rates[key]
		if !ok {
			// A model/token_type the customer used but the rate card has no rate for. Excluding it undercounts, so surface it — it usually means the rate card is missing a newly enabled model.
			slog.Warn("no rate card rate for metered token usage; excluding from agent spend",
				"model", sanitizeLogValue(key.model),
				"token_type", sanitizeLogValue(key.tokenType),
				"rate_card", sanitizeLogValue(rateCardID),
			)
			continue
		}
		totalCents += quantity * unitAmountCents
	}

	return int64(math.Round(totalCents)), nil
}

// fetchRateCardTokenRates loads every rate on the rate card, keyed by (model, token_type), and returns the token meter id the rates are attached to. unit_amount is returned verbatim (cents per token).
func (c *stripeClientImpl) fetchRateCardTokenRates(rateCardID string) (map[tokenRateKey]float64, string, error) {
	rates := make(map[tokenRateKey]float64)
	var meterID string

	path := fmt.Sprintf("/v2/billing/rate_cards/%s/rates?limit=100", rateCardID)
	for range 50 {
		resp, err := stripe.RawRequest(http.MethodGet, path, "", nil)
		if err != nil {
			return nil, "", fmt.Errorf("failed to fetch rate card rates: %w", err)
		}

		var parsed struct {
			Data []struct {
				UnitAmount  json.Number `json:"unit_amount"`
				MeteredItem struct {
					Meter                  string `json:"meter"`
					MeterSegmentConditions []struct {
						Dimension string `json:"dimension"`
						Value     string `json:"value"`
					} `json:"meter_segment_conditions"`
				} `json:"metered_item"`
			} `json:"data"`
			NextPageURL *string `json:"next_page_url"`
		}
		if err := json.Unmarshal(resp.RawJSON, &parsed); err != nil {
			return nil, "", fmt.Errorf("failed to parse rate card rates: %w", err)
		}

		for _, rate := range parsed.Data {
			if rate.MeteredItem.Meter != "" {
				meterID = rate.MeteredItem.Meter
			}
			var model, tokenType string
			for _, cond := range rate.MeteredItem.MeterSegmentConditions {
				switch cond.Dimension {
				case "model":
					model = cond.Value
				case "token_type":
					tokenType = cond.Value
				}
			}
			if model == "" || tokenType == "" {
				continue
			}
			amount, err := rate.UnitAmount.Float64()
			if err != nil {
				continue
			}
			rates[tokenRateKey{model: model, tokenType: tokenType}] = amount
		}

		if parsed.NextPageURL == nil || *parsed.NextPageURL == "" {
			break
		}
		path = *parsed.NextPageURL
	}

	return rates, meterID, nil
}

// fetchMeterTokenUsage returns the customer's token usage quantities since the given time, aggregated by (model, token_type). The Meter Usage Analytics API requires minute-aligned bounds and a customer; results have a short freshness lag.
func (c *stripeClientImpl) fetchMeterTokenUsage(customerID, meterID string, since time.Time) (map[tokenRateKey]float64, error) {
	startsAt := since.Truncate(time.Minute).Unix()
	endsAt := time.Now().UTC().Truncate(time.Minute).Unix()
	if endsAt <= startsAt {
		return map[tokenRateKey]float64{}, nil
	}

	q := url.Values{}
	q.Set("starts_at", strconv.FormatInt(startsAt, 10))
	q.Set("ends_at", strconv.FormatInt(endsAt, 10))
	q.Set("customer", customerID)
	q.Set("meters[0][meter]", meterID)
	q.Set("meters[0][dimension_group_by_keys][0]", "model")
	q.Set("meters[0][dimension_group_by_keys][1]", "token_type")
	q.Set("value_grouping_window", "month")

	resp, err := stripe.RawRequest(http.MethodGet, "/v1/billing/analytics/meter_usage?"+q.Encode(), "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meter usage: %w", err)
	}

	var parsed struct {
		Rows struct {
			Data []struct {
				Value      json.Number `json:"value"`
				Dimensions struct {
					Model     string `json:"model"`
					TokenType string `json:"token_type"`
				} `json:"dimensions"`
			} `json:"data"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(resp.RawJSON, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse meter usage: %w", err)
	}

	usage := make(map[tokenRateKey]float64)
	for _, row := range parsed.Rows.Data {
		value, err := row.Value.Float64()
		if err != nil {
			continue
		}
		key := tokenRateKey{model: row.Dimensions.Model, tokenType: row.Dimensions.TokenType}
		usage[key] += value
	}

	return usage, nil
}
