package llm

import (
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
