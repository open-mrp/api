package llm

import "strings"

// Per-model pricing in cents per million tokens.
type modelPricing struct {
	inputCentsPerMillion  int64
	outputCentsPerMillion int64
}

var modelPricingMap = map[string]modelPricing{
	"claude-sonnet-4":  {inputCentsPerMillion: 300, outputCentsPerMillion: 1500},
	"claude-haiku-4.5": {inputCentsPerMillion: 80, outputCentsPerMillion: 400},
	"gpt-4o":           {inputCentsPerMillion: 250, outputCentsPerMillion: 1000},
	"gpt-4o-mini":      {inputCentsPerMillion: 15, outputCentsPerMillion: 60},
}

// defaultPricing uses the most expensive rate as a conservative fallback.
var defaultPricing = modelPricing{inputCentsPerMillion: 300, outputCentsPerMillion: 1500}

func pricingForModel(model string) modelPricing {
	if p, ok := modelPricingMap[model]; ok {
		return p
	}
	// Try prefix matching for versioned model names (e.g. "claude-sonnet-4-20260301").
	for key, p := range modelPricingMap {
		if strings.HasPrefix(model, key) {
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
