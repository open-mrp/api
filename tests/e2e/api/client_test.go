//go:build e2e

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Response wraps an HTTP response with status code, body, and headers.
type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

const defaultAPIVersion = "1.0.forge-preview.1"

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
		retries:    3,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithAccountID returns a new Client targeting a different account.
func (c *Client) WithAccountID(accountID string) *Client {
	return NewClient(c.baseURL, c.apiKey, accountID)
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

// PageInfo contains cursor-based pagination metadata.
type PageInfo struct {
	NextCursor  *string `json:"next_cursor"`
	PrevCursor  *string `json:"prev_cursor"`
	HasNextPage bool    `json:"has_next_page"`
	HasPrevPage bool    `json:"has_prev_page"`
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
