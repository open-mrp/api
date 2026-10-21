// Package openrouter reads an OpenRouter API key's spend from OpenRouter's key endpoint.
package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"

	"github.com/shopspring/decimal"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

var openRouterTracer = tracing.GetTracer("core-service.openrouter_client")

// ClientConfig configures the OpenRouter key-usage client.
type ClientConfig struct {
	// BaseURL (optional; default: "https://openrouter.ai/api/v1") is OpenRouter's API root.
	BaseURL string

	// Timeout (optional; default: 10s) bounds each request to OpenRouter.
	Timeout time.Duration
}

func (c *ClientConfig) WithDefaults() *ClientConfig {
	if c == nil {
		c = &ClientConfig{}
	}
	if c.BaseURL == "" {
		c.BaseURL = defaultBaseURL
	}
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
	return c
}

func (c *ClientConfig) validate() error {
	if c.Timeout < 0 {
		return fmt.Errorf("openrouter client: timeout must not be negative")
	}
	return nil
}

type client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds the client agent-run spend checks and the usage endpoint read OpenRouter through.
func NewClient(config *ClientConfig) (domain.OpenRouterClient, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &client{baseURL: config.BaseURL, httpClient: &http.Client{Timeout: config.Timeout}}, nil
}

type keyResponse struct {
	Data struct {
		Usage          float64  `json:"usage"`
		UsageDaily     float64  `json:"usage_daily"`
		UsageWeekly    float64  `json:"usage_weekly"`
		UsageMonthly   float64  `json:"usage_monthly"`
		Limit          *float64 `json:"limit"`
		LimitRemaining *float64 `json:"limit_remaining"`
	} `json:"data"`
}

// GetKeyUsage returns what OpenRouter has charged apiKey, in US dollars. Its periods are UTC calendar day, week and month.
func (c *client) GetKeyUsage(ctx context.Context, apiKey string) (*domain.OpenRouterKeyUsage, *apierror.APIError) {
	ctx, span := openRouterTracer.Start(ctx, "openrouter_client.get_key_usage")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/key", nil)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to create OpenRouter request."))
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req) // #nosec G704 -- URL from server-configured OpenRouter API base
	if err != nil {
		return nil, tracing.Trace(span, externalError(err, "OpenRouter could not be reached."))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, tracing.Trace(span, externalError(err, "Failed to read the OpenRouter response."))
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("OpenRouter rejected the integration's API key. Update the OpenRouter integration with a valid key.", "credentials"))
	case resp.StatusCode != http.StatusOK:
		return nil, tracing.Trace(span, externalError(fmt.Errorf("openrouter key endpoint returned %d: %s", resp.StatusCode, body), "OpenRouter returned an error."))
	}

	var parsed keyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, tracing.Trace(span, externalError(err, "Failed to parse the OpenRouter response."))
	}

	return &domain.OpenRouterKeyUsage{
		SpentToday:        dollars(parsed.Data.UsageDaily),
		SpentThisWeek:     dollars(parsed.Data.UsageWeekly),
		SpentThisMonth:    dollars(parsed.Data.UsageMonthly),
		SpentTotal:        dollars(parsed.Data.Usage),
		KeyLimit:          dollarsPtr(parsed.Data.Limit),
		KeyLimitRemaining: dollarsPtr(parsed.Data.LimitRemaining),
	}, nil
}

// dollars rounds OpenRouter's float credit figures to a hundredth of a cent, which is finer than any single request bills.
func dollars(v float64) string {
	return decimal.NewFromFloat(v).Round(4).String()
}

func dollarsPtr(v *float64) *string {
	if v == nil {
		return nil
	}
	s := dollars(*v)
	return &s
}

func externalError(err error, publicMessage string) *apierror.APIError {
	return apierror.NewAPIError(apierror.ErrorCodeExternalSvcError, apierror.ErrorTypeAPI, publicMessage, err.Error(), apierror.WithInternal(err))
}
