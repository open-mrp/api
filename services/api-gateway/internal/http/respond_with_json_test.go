package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/shared/appctx"
)

func TestRespondWithJSON_WithoutRequestLog(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"message": "success",
		"data":    "test",
	}
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json", w.Header().Get("Content-Type"))
	}

	if w.Header().Get("Request-ID") != "" {
		t.Fatalf("expected Request-ID to be empty, got %q", w.Header().Get("Request-ID"))
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "success" {
		t.Fatalf("expected message %q, got %q", "success", response["message"])
	}

	if response["data"] != "test" {
		t.Fatalf("expected data %q, got %q", "test", response["data"])
	}
}

func TestRespondWithJSON_WithRequestLog(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"message": "success",
	}
	ua := "Mozilla/5.0"
	rl := &appctx.RequestLog{
		ID:        "test-request-id-123",
		UserAgent: &ua,
	}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json", w.Header().Get("Content-Type"))
	}

	if w.Header().Get("Request-ID") != "test-request-id-123" {
		t.Fatalf("expected Request-ID %q, got %q", "test-request-id-123", w.Header().Get("Request-ID"))
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "success" {
		t.Fatalf("expected message %q, got %q", "success", response["message"])
	}
}

func TestRespondWithJSON_WithCurlUserAgent_PrettyPrint(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"message": "success",
		"data":    "test",
	}
	ua := "curl/7.68.0"
	rl := &appctx.RequestLog{
		ID:        "test-request-id",
		UserAgent: &ua,
	}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	// Pretty printed JSON should have newlines and indentation
	if !json.Valid([]byte(body)) {
		t.Fatal("response body is not valid JSON")
	}

	// Check that it's pretty printed (contains newlines)
	if !containsNewlines(body) {
		t.Fatal("expected pretty printed JSON with newlines, got compact JSON")
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "success" {
		t.Fatalf("expected message %q, got %q", "success", response["message"])
	}
}

func TestRespondWithJSON_WithNonCurlUserAgent_CompactJSON(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"message": "success",
		"data":    "test",
	}
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	rl := &appctx.RequestLog{
		ID:        "test-request-id",
		UserAgent: &ua,
	}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	// Compact JSON should not have newlines (except possibly at the end)
	bodyWithoutTrailingNewline := body
	if len(bodyWithoutTrailingNewline) > 0 && bodyWithoutTrailingNewline[len(bodyWithoutTrailingNewline)-1] == '\n' {
		bodyWithoutTrailingNewline = bodyWithoutTrailingNewline[:len(bodyWithoutTrailingNewline)-1]
	}

	if !json.Valid([]byte(bodyWithoutTrailingNewline)) {
		t.Fatal("response body is not valid JSON")
	}

	// Check that it's compact (no newlines in the middle)
	if containsNewlines(bodyWithoutTrailingNewline) {
		t.Fatal("expected compact JSON without newlines, got pretty printed JSON")
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "success" {
		t.Fatalf("expected message %q, got %q", "success", response["message"])
	}
}

func TestRespondWithJSON_UnauthorizedStatus_SetsWWWAuthenticate(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"error": "unauthorized",
	}
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusUnauthorized, payload)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if w.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("expected WWW-Authenticate %q, got %q", "Bearer", w.Header().Get("WWW-Authenticate"))
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json", w.Header().Get("Content-Type"))
	}
}

func TestRespondWithJSON_DifferentStatusCodes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		statusCode     int
		payload        any
		expectedStatus int
	}{
		{
			name:           "Created",
			statusCode:     http.StatusCreated,
			payload:        map[string]any{"id": "123"},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "BadRequest",
			statusCode:     http.StatusBadRequest,
			payload:        map[string]any{"error": "bad request"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "NotFound",
			statusCode:     http.StatusNotFound,
			payload:        map[string]any{"error": "not found"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "InternalServerError",
			statusCode:     http.StatusInternalServerError,
			payload:        map[string]any{"error": "internal error"},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			w := httptest.NewRecorder()

			RespondWithJSON(ctx, w, tc.statusCode, tc.payload)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("expected Content-Type %q, got %q", "application/json", w.Header().Get("Content-Type"))
			}

			var response map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
		})
	}
}

func TestRespondWithJSON_ComplexPayload(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"user": map[string]any{
			"id":    "user-123",
			"name":  "John Doe",
			"email": "john@example.com",
			"tags":  []string{"admin", "user"},
		},
		"metadata": map[string]any{
			"created_at": "2024-01-01T00:00:00Z",
			"count":      42,
		},
	}
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	user, ok := response["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user to be a map")
	}

	if user["id"] != "user-123" {
		t.Fatalf("expected user id %q, got %q", "user-123", user["id"])
	}
}

func TestRespondWithJSON_ArrayPayload(t *testing.T) {
	t.Parallel()
	payload := []map[string]any{
		{"id": "1", "name": "Item 1"},
		{"id": "2", "name": "Item 2"},
		{"id": "3", "name": "Item 3"},
	}
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 3 {
		t.Fatalf("expected 3 items, got %d", len(response))
	}

	if response[0]["id"] != "1" {
		t.Fatalf("expected first item id %q, got %q", "1", response[0]["id"])
	}
}

func TestRespondWithJSON_JSONMarshallingError_Returns500(t *testing.T) {
	t.Parallel(
	// Create a type that fails to marshal
	)

	type unmarshalableType struct {
		Value chan int // Channels cannot be marshalled to JSON
	}

	payload := unmarshalableType{
		Value: make(chan int),
	}
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	// Should return 500 when marshalling fails
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json", w.Header().Get("Content-Type"))
	}
}

func TestRespondWithJSON_WriteError_LogsError(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"message": "success",
	}
	ctx := context.Background()
	recorder := httptest.NewRecorder()
	w := &failingResponseWriter{
		ResponseWriter: recorder,
		failOnWrite:    true,
	}

	RespondWithJSON(ctx, w, http.StatusOK, payload)

	// The function should still set the status code even if Write fails
	// (the error is logged but doesn't affect the status code)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
}

// failingResponseWriter is a ResponseWriter that can be configured to fail on Write
type failingResponseWriter struct {
	http.ResponseWriter
	failOnWrite bool
}

func (w *failingResponseWriter) Write(b []byte) (int, error) {
	if w.failOnWrite {
		return 0, &writeError{msg: "write failed"}
	}
	return w.ResponseWriter.Write(b)
}

type writeError struct {
	msg string
}

func (e *writeError) Error() string {
	return e.msg
}

func containsNewlines(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '\n' {
			return true
		}
	}
	return false
}

func TestRespondWithJSON_WithOptions(t *testing.T) {
	t.Parallel()
	payload := map[string]any{"message": "success"}
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusOK, payload, WithHeader(header.IdempotentReplayedHeader, "true"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get(header.IdempotentReplayedHeader) != "true" {
		t.Fatalf("expected %s true, got %q", header.IdempotentReplayedHeader, w.Header().Get(header.IdempotentReplayedHeader))
	}
	if w.Header().Get(header.ContentTypeHeader) != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", w.Header().Get(header.ContentTypeHeader))
	}
}

func TestRespondWithJSON_WithLocation(t *testing.T) {
	t.Parallel()
	payload := map[string]any{"id": "ak_123"}
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithJSON(ctx, w, http.StatusCreated, payload, WithLocation("/v1/auth/api-keys/ak_123"))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	if w.Header().Get(header.LocationHeader) != "/v1/auth/api-keys/ak_123" {
		t.Fatalf("expected Location %q, got %q", "/v1/auth/api-keys/ak_123", w.Header().Get(header.LocationHeader))
	}
}

func TestRespondWithJSONBytes(t *testing.T) {
	t.Parallel()
	body := []byte(`{"replayed":true}`)
	rl := &appctx.RequestLog{ID: "req-123"}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	w := httptest.NewRecorder()

	RespondWithJSONBytes(ctx, w, http.StatusOK, body, WithHeader(header.IdempotentReplayedHeader, "true"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get(header.RequestIDHeader) != "req-123" {
		t.Fatalf("expected Request-ID req-123, got %q", w.Header().Get(header.RequestIDHeader))
	}
	if w.Header().Get(header.IdempotentReplayedHeader) != "true" {
		t.Fatalf("expected Idempotent-Replayed true, got %q", w.Header().Get(header.IdempotentReplayedHeader))
	}
	if w.Body.String() != string(body) {
		t.Fatalf("expected body %s, got %s", body, w.Body.String())
	}
}
