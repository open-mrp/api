// Package httpgateway provides an HTTP client that invokes api-gateway endpoints over the gateway's trusted internal listener on behalf of an agent identity. It backs the generated endpoint-tools.
package httpgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/shared/version"
)

const (
	defaultTimeout = 30 * time.Second
	// maxResponseBytes caps how much of a response we read from the gateway.
	maxResponseBytes = 1 << 20 // 1 MiB
	// maxOutputBytes caps what we hand back to the LLM, to avoid bloating context.
	maxOutputBytes = 24000

	versionHeader              = "Augno-Version"
	internalIdentityHeader     = "X-Augno-Internal-Identity"
	internalServiceTokenHeader = "X-Augno-Service-Token" // #nosec G101 -- header name, not a credential
	idempotencyKeyHeader       = "Idempotency-Key"
)

// Client calls api-gateway endpoints via the internal listener. It implements domain.GatewayClient.
type Client struct {
	baseURL      string
	serviceToken string
	apiVersion   string
	httpClient   *http.Client
}

// NewClient builds a gateway client pointed at the internal listener base URL (e.g. http://api-gateway-internal:8091), authenticating with the shared service token.
func NewClient(baseURL, serviceToken string) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		serviceToken: serviceToken,
		apiVersion:   version.Latest.Version,
		httpClient:   &http.Client{Timeout: defaultTimeout},
	}
}

// Do issues the request, forwarding the agent identity and service token on the internal headers. HTTP error responses (4xx/5xx) are returned as the result string so the LLM can self-correct; only transport/encoding failures return a Go error.
func (c *Client) Do(ctx context.Context, gr domain.GatewayRequest) (string, error) {
	u := c.baseURL + gr.Path
	if len(gr.Query) > 0 {
		u += "?" + gr.Query.Encode()
	}

	var bodyReader io.Reader
	if len(gr.Body) > 0 {
		bodyReader = bytes.NewReader(gr.Body)
	}

	req, err := http.NewRequestWithContext(ctx, gr.Method, u, bodyReader)
	if err != nil {
		return "", fmt.Errorf("gateway: build request: %w", err)
	}
	if len(gr.Body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(versionHeader, c.apiVersion)
	req.Header.Set(internalServiceTokenHeader, c.serviceToken)

	idJSON, err := json.Marshal(gr.Identity)
	if err != nil {
		return "", fmt.Errorf("gateway: marshal identity: %w", err)
	}
	req.Header.Set(internalIdentityHeader, string(idJSON))

	if gr.IdempotencyKey != "" {
		req.Header.Set(idempotencyKeyHeader, gr.IdempotencyKey)
	}

	resp, err := c.httpClient.Do(req) // #nosec G704 -- URL from server-configured api-gateway internal base and generated endpoint paths
	if err != nil {
		return "", fmt.Errorf("gateway: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("gateway: read response: %w", err)
	}

	out := string(body)
	if resp.StatusCode >= http.StatusBadRequest {
		out = fmt.Sprintf("Request failed with HTTP %d: %s", resp.StatusCode, out)
	}
	if len(out) > maxOutputBytes {
		out = out[:maxOutputBytes] + "\n...[response truncated]"
	}
	return out, nil
}
