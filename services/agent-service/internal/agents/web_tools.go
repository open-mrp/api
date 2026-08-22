package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

const maxFetchOutputBytes = 30_000 // 30 KB post-conversion cap

var webHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

const maxResponseBytes = 200_000 // 200 KB

func HandleFetchURL(_ context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid fetch_url input: %w", err)
	}

	parsed, err := url.Parse(params.URL)
	if err != nil {
		return "", fmt.Errorf("fetch_url: invalid URL: %w", err)
	}

	if parsed.Scheme != "https" {
		return "", fmt.Errorf("fetch_url: only HTTPS URLs are allowed")
	}

	// Block private/internal network ranges
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "172.") || host == "0.0.0.0" ||
		strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return "", fmt.Errorf("fetch_url: internal/private URLs are not allowed")
	}

	req, err := http.NewRequest(http.MethodGet, params.URL, nil)
	if err != nil {
		return "", fmt.Errorf("fetch_url: failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "OpenMRPAgent/1.0")
	req.Header.Set("Accept", "text/html, text/plain, application/json, text/markdown")

	resp, err := webHTTPClient.Do(req) // #nosec G704 -- URL from agent-provided tool input, validated upstream
	if err != nil {
		return "", fmt.Errorf("fetch_url: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("The URL returned HTTP status %d.", resp.StatusCode), nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("fetch_url: failed to read response: %w", err)
	}

	content := string(body)
	if len(content) == 0 {
		return "The URL returned empty content.", nil
	}

	// Convert HTML responses to markdown for compact LLM consumption.
	ct := resp.Header.Get("Content-Type")
	if isHTMLContent(ct) {
		content = htmlToMarkdown(content)
	}

	// Cap output to avoid bloating context.
	if len(content) > maxFetchOutputBytes {
		content = content[:maxFetchOutputBytes] + "\n...[content truncated]"
	}

	return content, nil
}
