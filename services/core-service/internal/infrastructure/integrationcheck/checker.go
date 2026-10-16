// Package integrationcheck asks an integration's provider whether it accepts a set of credentials, with one cheap
// authenticated read per provider.
package integrationcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

var checkerTracer = tracing.GetTracer("core-service.integration_check")

// Config configures a Checker.
type Config struct {
	// StripeBaseURL (optional; default https://api.stripe.com) is the Stripe API root.
	StripeBaseURL string
	// ShippoBaseURL (optional; default https://api.goshippo.com) is the Shippo API root.
	ShippoBaseURL string
	// HubspotBaseURL (optional; default https://api.hubapi.com) is the HubSpot API root.
	HubspotBaseURL string
	// Timeout (optional; default 10s) bounds one check.
	Timeout time.Duration
}

// WithDefaults fills zero-value fields with production defaults.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}
	out := *c
	if out.StripeBaseURL == "" {
		out.StripeBaseURL = "https://api.stripe.com"
	}
	if out.ShippoBaseURL == "" {
		out.ShippoBaseURL = "https://api.goshippo.com"
	}
	if out.HubspotBaseURL == "" {
		out.HubspotBaseURL = "https://api.hubapi.com"
	}
	if out.Timeout == 0 {
		out.Timeout = 10 * time.Second
	}
	return &out
}

// Checker implements domain.IntegrationCredentialChecker against the live provider APIs.
type Checker struct {
	config     Config
	httpClient *http.Client
}

// New builds a Checker.
func New(config *Config) *Checker {
	config = config.WithDefaults()
	return &Checker{config: *config, httpClient: &http.Client{Timeout: config.Timeout}}
}

var _ domain.IntegrationCredentialChecker = (*Checker)(nil)

// Check reads one resource the credentials must be able to read: Stripe's balance with the secret key, Shippo's
// carrier accounts with the API key, HubSpot companies with the access token (a scope the sync needs anyway). The
// Stripe publishable key and webhook secret cannot be checked without a payment or an event.
func (c *Checker) Check(ctx context.Context, provider constants.IntegrationCode, credentials string) (constants.IntegrationConnectionStatus, *apierror.APIError) {
	ctx, span := checkerTracer.Start(ctx, "integration_check.check")
	defer span.End()

	var url, auth string
	switch provider {
	case constants.IntegrationCodeStripe:
		var creds domain.StripeCredentials
		if err := json.Unmarshal([]byte(credentials), &creds); err != nil {
			return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Stripe credentials."))
		}
		url, auth = c.config.StripeBaseURL+"/v1/balance", "Bearer "+creds.PrivateKey
	case constants.IntegrationCodeShippo:
		var creds domain.ShippoCredentials
		if err := json.Unmarshal([]byte(credentials), &creds); err != nil {
			return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo credentials."))
		}
		url, auth = c.config.ShippoBaseURL+"/carrier_accounts/?results=1", "ShippoToken "+creds.APIKey
	case constants.IntegrationCodeHubspot:
		var creds domain.HubspotCredentials
		if err := json.Unmarshal([]byte(credentials), &creds); err != nil {
			return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse HubSpot credentials."))
		}
		url, auth = c.config.HubspotBaseURL+"/crm/v3/objects/companies?limit=1", "Bearer "+creds.AccessToken
	default:
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("No credential check for provider "+string(provider)+"."))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to build the credential check request."))
	}
	req.Header.Set("Authorization", auth)
	resp, err := c.httpClient.Do(req) // #nosec G107 G704 -- the URL is a fixed provider endpoint from config
	if err != nil {
		return "", tracing.Trace(span, apierror.NewExternalServiceError(err, "Credential check could not reach "+string(provider)+"."))
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	status, apiErr := statusFromResponse(provider, resp.StatusCode)
	return status, tracing.Trace(span, apiErr)
}

// statusFromResponse reads a provider's answer: 2xx accepts the credentials, 401 and 403 reject them, and anything
// else leaves the question open.
func statusFromResponse(provider constants.IntegrationCode, code int) (constants.IntegrationConnectionStatus, *apierror.APIError) {
	switch {
	case code >= 200 && code < 300:
		return constants.IntegrationConnectionStatusConnected, nil
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return constants.IntegrationConnectionStatusInvalidCredentials, nil
	default:
		return "", apierror.NewExternalServiceError(fmt.Errorf("%s answered %d", provider, code), "Credential check got no answer from "+string(provider)+".")
	}
}
