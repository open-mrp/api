package llm

import (
	"context"
	"errors"
)

// ErrNoAPIKey means neither the run nor the deployment supplied an OpenRouter key.
var ErrNoAPIKey = errors.New("no OpenRouter API key: connect the OpenRouter integration in Settings > Integrations")

type apiKeyContextKey struct{}

// WithAPIKey makes the LLM calls made with ctx bill to apiKey instead of the provider's own key.
func WithAPIKey(ctx context.Context, apiKey string) context.Context {
	return context.WithValue(ctx, apiKeyContextKey{}, apiKey)
}

// resolveAPIKey returns the key ctx carries, else fallback.
func resolveAPIKey(ctx context.Context, fallback string) (string, error) {
	if key, _ := ctx.Value(apiKeyContextKey{}).(string); key != "" {
		return key, nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", ErrNoAPIKey
}
