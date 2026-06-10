//go:build e2e

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Response wraps an HTTP response with status code, body, and headers.
type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

const defaultAPIVersion = "1.0.forge-preview.2"

// Client is an HTTP client for the Augno API configured with authentication.
type Client struct {
	baseURL    string
	apiKey     string
	accountID  string
	apiVersion string
	httpClient *http.Client
	retries    int
}

// NewClient creates a new API client.
func NewClient(baseURL, apiKey, accountID string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		accountID:  accountID,
		apiVersion: defaultAPIVersion,
		retries:    5,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithAccountID returns a new Client targeting a different account.
func (c *Client) WithAccountID(accountID string) *Client {
	return NewClient(c.baseURL, c.apiKey, accountID)
}

// WithAPIVersion returns a new Client pinned to a specific API version. Used
// to exercise the version-transform backwards-compatibility path.
func (c *Client) WithAPIVersion(version string) *Client {
	clone := NewClient(c.baseURL, c.apiKey, c.accountID)
	clone.apiVersion = version
	return clone
}

// WithBearerToken returns a new Client that authenticates with the given bearer
// token (e.g. a user access token obtained from login) against accountID. Used
// to exercise endpoints that require a user identity rather than an API key. In
// the non-production e2e environment access tokens are accepted in the
// Authorization header.
func (c *Client) WithBearerToken(token, accountID string) *Client {
	return NewClient(c.baseURL, token, accountID)
}

// Get performs an authenticated GET request with automatic retry on 429.
func (c *Client) Get(path string, params url.Values) (*http.Response, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	for attempt := 0; attempt <= c.retries; attempt++ {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Augno-Account", c.accountID)
		req.Header.Set("Augno-Version", c.apiVersion)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if attempt < c.retries {
			if resp.StatusCode == http.StatusTooManyRequests || isTransientError(resp) {
				resp.Body.Close()
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("exhausted retries for %s", u)
}

// GetFull performs an authenticated GET and returns a Response with headers.
func (c *Client) GetFull(path string, params url.Values) (*Response, error) {
	resp, err := c.Get(path, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Header:     resp.Header,
	}, nil
}

// ListResponse is the standard list envelope returned by all list endpoints.
type ListResponse struct {
	Object   string            `json:"object"`
	PageInfo PageInfo          `json:"page_info"`
	Data     []json.RawMessage `json:"data"`
}

// PageInfo contains URL-based pagination metadata.
type PageInfo struct {
	NextPageURL     *string `json:"next_page_url"`
	PreviousPageURL *string `json:"previous_page_url"`
	HasNextPage     bool    `json:"has_next_page"`
	HasPrevPage     bool    `json:"has_prev_page"`
}

// NextCursor extracts the cursor query parameter from NextPageURL.
func (p PageInfo) NextCursor() *string {
	return extractCursorFromPageURL(p.NextPageURL)
}

// PrevCursor extracts the cursor query parameter from PreviousPageURL.
func (p PageInfo) PrevCursor() *string {
	return extractCursorFromPageURL(p.PreviousPageURL)
}

// extractCursorFromPageURL parses a pagination URL and returns the cursor query param value.
func extractCursorFromPageURL(pageURL *string) *string {
	if pageURL == nil {
		return nil
	}
	// URLs are relative like "/v1/path?cursor=xxx&limit=10"
	idx := strings.Index(*pageURL, "?")
	if idx < 0 {
		return nil
	}
	params, err := url.ParseQuery((*pageURL)[idx+1:])
	if err != nil {
		return nil
	}
	cursor := params.Get("cursor")
	if cursor == "" {
		return nil
	}
	return &cursor
}

// ListURLPathQuery parses a relative list pagination URL (e.g. from page_info
// next_page_url / previous_page_url) into a path and query values for use with
// Get, GetList, or GetListRaw.
func ListURLPathQuery(relativeURL *string) (path string, values url.Values, ok bool) {
	if relativeURL == nil {
		return "", nil, false
	}
	raw := strings.TrimSpace(*relativeURL)
	if raw == "" || raw[0] != '/' {
		return "", nil, false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Path == "" {
		return "", nil, false
	}
	return u.Path, u.Query(), true
}

// GetListFromPageURL performs GET using a relative URL from page_info (next or previous).
func (c *Client) GetListFromPageURL(relativePageURL *string) (*ListResponse, int, error) {
	path, params, ok := ListURLPathQuery(relativePageURL)
	if !ok {
		return nil, 0, fmt.Errorf("invalid list page URL")
	}
	return c.GetList(path, params)
}

// GetListRawFromPageURL is like GetListFromPageURL but returns status and raw body.
func (c *Client) GetListRawFromPageURL(relativePageURL *string) (int, []byte, error) {
	path, params, ok := ListURLPathQuery(relativePageURL)
	if !ok {
		return 0, nil, fmt.Errorf("invalid list page URL")
	}
	return c.GetListRaw(path, params)
}

// GetList performs an authenticated GET and parses the response as a ListResponse.
func (c *Client) GetList(path string, params url.Values) (*ListResponse, int, error) {
	resp, err := c.Get(path, params)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var list ListResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("parsing list response: %w (body: %s)", err, string(body))
	}

	return &list, resp.StatusCode, nil
}

// GetListRaw performs an authenticated GET and returns the status code and raw body.
func (c *Client) GetListRaw(path string, params url.Values) (int, []byte, error) {
	resp, err := c.Get(path, params)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("reading response body: %w", err)
	}

	return resp.StatusCode, body, nil
}

// DoFull performs an authenticated HTTP request with the given method, optional JSON body,
// and optional idempotency key. Returns a Response with status code, body, and headers.
func (c *Client) DoFull(method, path string, body any, idempotencyKey string) (*Response, error) {
	u := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	for attempt := 0; attempt <= c.retries; attempt++ {
		// Reset the reader for retries.
		if body != nil {
			b, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(b)
		}

		req, err := http.NewRequest(method, u, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Augno-Account", c.accountID)
		req.Header.Set("Augno-Version", c.apiVersion)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		if attempt < c.retries {
			if resp.StatusCode == http.StatusTooManyRequests || isTransientServerError(resp.StatusCode, respBody) {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
		}

		return &Response{
			StatusCode: resp.StatusCode,
			Body:       respBody,
			Header:     resp.Header,
		}, nil
	}

	return nil, fmt.Errorf("exhausted retries for %s %s", method, u)
}

// Do performs an authenticated HTTP request and returns status code and raw body.
func (c *Client) Do(method, path string, body any, idempotencyKey string) (int, []byte, error) {
	resp, err := c.DoFull(method, path, body, idempotencyKey)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, resp.Body, nil
}

// Post performs an authenticated POST request with a JSON body.
func (c *Client) Post(path string, body any, idempotencyKey string) (int, []byte, error) {
	return c.Do(http.MethodPost, path, body, idempotencyKey)
}

// Patch performs an authenticated PATCH request with a JSON body.
func (c *Client) Patch(path string, body any, idempotencyKey string) (int, []byte, error) {
	return c.Do(http.MethodPatch, path, body, idempotencyKey)
}

// Put performs an authenticated PUT request with a JSON body.
func (c *Client) Put(path string, body any) (int, []byte, error) {
	return c.Do(http.MethodPut, path, body, "")
}

// PutRaw performs PUT with optional query params (same pattern as GET with query encoding).
func (c *Client) PutRaw(path string, params url.Values, body any) (int, []byte, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	for attempt := 0; attempt <= c.retries; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			b, err := json.Marshal(body)
			if err != nil {
				return 0, nil, fmt.Errorf("marshalling request body: %w", err)
			}
			bodyReader = bytes.NewReader(b)
		}

		req, err := http.NewRequest(http.MethodPut, u, bodyReader)
		if err != nil {
			return 0, nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Augno-Account", c.accountID)
		req.Header.Set("Augno-Version", c.apiVersion)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return resp.StatusCode, nil, fmt.Errorf("reading response body: %w", err)
		}

		if attempt < c.retries {
			if resp.StatusCode == http.StatusTooManyRequests || isTransientServerError(resp.StatusCode, respBody) {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
		}

		return resp.StatusCode, respBody, nil
	}

	return 0, nil, fmt.Errorf("exhausted retries for PUT %s", u)
}

// Delete performs an authenticated DELETE request.
func (c *Client) Delete(path string) (int, []byte, error) {
	return c.Do(http.MethodDelete, path, nil, "")
}

// PostFull performs an authenticated POST and returns a Response with headers.
func (c *Client) PostFull(path string, body any, idempotencyKey string) (*Response, error) {
	return c.DoFull(http.MethodPost, path, body, idempotencyKey)
}

// PatchFull performs an authenticated PATCH and returns a Response with headers.
func (c *Client) PatchFull(path string, body any, idempotencyKey string) (*Response, error) {
	return c.DoFull(http.MethodPatch, path, body, idempotencyKey)
}

// PutFull performs an authenticated PUT and returns a Response with headers.
func (c *Client) PutFull(path string, body any) (*Response, error) {
	return c.DoFull(http.MethodPut, path, body, "")
}

// DeleteFull performs an authenticated DELETE and returns a Response with headers.
func (c *Client) DeleteFull(path string) (*Response, error) {
	return c.DoFull(http.MethodDelete, path, nil, "")
}

// isTransientError checks if an *http.Response contains a transient error by
// peeking at the body. It returns true when the error envelope has is_transient
// set to true (e.g., 5xx server errors, 409 idempotency_in_progress).
// The response body is replaced so it can still be read after this call.
func isTransientError(resp *http.Response) bool {
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return false
	}
	return isTransientServerError(resp.StatusCode, body)
}

// isTransientServerError checks if a response body contains an error with
// is_transient set to true, indicating the request is safe to retry.
func isTransientServerError(_ int, body []byte) bool {
	var envelope struct {
		Error struct {
			IsTransient bool `json:"is_transient"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return false
	}
	return envelope.Error.IsTransient
}

// DataItemField extracts a string field from a data item in a list response.
func DataItemField(item json.RawMessage, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(item, &m); err != nil {
		return ""
	}
	raw, ok := m[field]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}
