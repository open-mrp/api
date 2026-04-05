package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/augno/api/services/agent-service/internal/domain"
)

var docsHTTPClient = &http.Client{Timeout: 15 * time.Second}

func HandleReadDoc(_ context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid read_doc input: %w", err)
	}

	if !strings.HasPrefix(params.URL, "https://docs.augno.com/") {
		return "", fmt.Errorf("read_doc: URL must be from docs.augno.com")
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
