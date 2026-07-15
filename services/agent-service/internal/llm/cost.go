package llm

import (
	"math"
	"strings"

	"github.com/augno/api/shared/constants"
)

// Per-model pricing in cents per million tokens.
type modelPricing struct {
	inputCentsPerMillion  int64
	outputCentsPerMillion int64
}

var modelPricingMap = map[constants.Model]modelPricing{
	constants.ModelClaudeOpus48:   {inputCentsPerMillion: 500, outputCentsPerMillion: 2500},
	constants.ModelClaudeSonnet46: {inputCentsPerMillion: 300, outputCentsPerMillion: 1500},
	constants.ModelClaudeSonnet4:  {inputCentsPerMillion: 300, outputCentsPerMillion: 1500},
	constants.ModelClaudeHaiku45:  {inputCentsPerMillion: 80, outputCentsPerMillion: 400},
	constants.ModelGPT55:          {inputCentsPerMillion: 500, outputCentsPerMillion: 3000},
	constants.ModelGPT4o:          {inputCentsPerMillion: 250, outputCentsPerMillion: 1000},
	constants.ModelGPT4oMini:      {inputCentsPerMillion: 15, outputCentsPerMillion: 60},
}

// defaultPricing uses the most expensive rate as a conservative fallback.
var defaultPricing = modelPricing{inputCentsPerMillion: 300, outputCentsPerMillion: 1500}

func pricingForModel(model string) modelPricing {
	if p, ok := modelPricingMap[constants.Model(model)]; ok {
		return p
	}
	// Try prefix matching for versioned model names (e.g. "claude-sonnet-4-20260301").
	for key, p := range modelPricingMap {
		if strings.HasPrefix(model, string(key)) {
			return p
		}
	}
	return defaultPricing
}

// EstimateTokenCostCents returns the estimated cost in cents for the given token counts and model. Uses conservative (most expensive) pricing as fallback for unknown models.
func EstimateTokenCostCents(inputTokens, outputTokens int, model string) int64 {
	p := pricingForModel(model)
	inputCost := int64(inputTokens) * p.inputCentsPerMillion / 1_000_000
	outputCost := int64(outputTokens) * p.outputCentsPerMillion / 1_000_000
	return inputCost + outputCost
}

// TokenRateKey identifies a marked-up rate by the gateway model name and token type it applies to.
type TokenRateKey struct {
	Model     string
	TokenType string
}

// MarkedUpTokenCostCents prices a turn's tokens at the plan's marked-up rate card rates — the same figures Stripe bills — so the in-run spending-cap gate matches the customer's actual bill. gatewayModel is the provider-prefixed name the rate card is keyed on (see GatewayModelName). Returns (cost, true) when both input and output rates are present; (0, false) when a rate is missing so the caller can fall back to EstimateTokenCostCents. Cached input/output are billed cheaper by Stripe but aren't tracked per turn, so charging them at the (higher) input/output rate keeps the gate conservative.
func MarkedUpTokenCostCents(inputTokens, outputTokens int, gatewayModel string, rates map[TokenRateKey]float64) (int64, bool) {
	inputRate, okIn := rates[TokenRateKey{Model: gatewayModel, TokenType: "input"}]
	outputRate, okOut := rates[TokenRateKey{Model: gatewayModel, TokenType: "output"}]
	if !okIn || !okOut {
		return 0, false
	}
	cents := float64(inputTokens)*inputRate + float64(outputTokens)*outputRate
	return int64(math.Round(cents)), true
}
