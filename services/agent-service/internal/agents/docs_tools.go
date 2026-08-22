package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

var docsHTTPClient = &http.Client{Timeout: 15 * time.Second}

// docsURLPrefixes are the documentation origins read_doc will fetch from.
var docsURLPrefixes = []string{"https://docs.openmrp.ai/", "https://docs.augno.com/"}

func HandleReadDoc(_ context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid read_doc input: %w", err)
	}

	// docs.openmrp.ai is the live docs host since the DNS cutover; docs.augno.com is still
	// accepted because it keeps serving, and an agent may be working from a URL it read
	// before the move or from a page that still links the old host.
	if !slices.ContainsFunc(docsURLPrefixes, func(prefix string) bool {
		return strings.HasPrefix(params.URL, prefix)
	}) {
		return "", fmt.Errorf("read_doc: URL must be from docs.openmrp.ai or docs.augno.com")
	}

	resp, err := docsHTTPClient.Get(params.URL)
	if err != nil {
		return "", fmt.Errorf("read_doc: failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("read_doc: page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100_000))
	if err != nil {
		return "", fmt.Errorf("read_doc: failed to read response: %w", err)
	}

	content := string(body)
	if len(content) == 0 {
		return "The page returned empty content.", nil
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
