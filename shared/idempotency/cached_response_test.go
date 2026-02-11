package idempotency

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
)

func ptr[T any](v T) *T { return &v }

// --- Success responses ---

func TestUnmarshalCachedResponse_Success(t *testing.T) {
	type Order struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	body := json.RawMessage(`{"id":"or_123","name":"Test Order"}`)
	meta := &appctx.IdempotencyResponseMetadata{}
	ctx := appctx.WithIdempotencyResponseMetadata(context.Background(), meta)

	result, err := UnmarshalCachedResponse[Order](ctx, ptr(http.StatusOK), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasCache {
		t.Fatal("expected HasCache=true")
	}
	if result.Error != nil {
		t.Fatal("expected Error=nil for success response")
	}
	if result.Data == nil {
		t.Fatal("expected Data to be non-nil")
	}
	if result.Data.ID != "or_123" {
		t.Errorf("expected ID %q, got %q", "or_123", result.Data.ID)
	}
	if result.Data.Name != "Test Order" {
		t.Errorf("expected Name %q, got %q", "Test Order", result.Data.Name)
	}
	if !meta.Replayed {
		t.Error("expected Replayed=true after successful unmarshal")
	}
}

func TestUnmarshalCachedResponse_SuccessWithCreatedStatus(t *testing.T) {
	type Item struct {
		Value int `json:"value"`
	}

	body := json.RawMessage(`{"value":42}`)
	ctx := context.Background()

	result, err := UnmarshalCachedResponse[Item](ctx, ptr(http.StatusCreated), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasCache {
		t.Fatal("expected HasCache=true")
	}
	if result.Data == nil || result.Data.Value != 42 {
		t.Errorf("expected Value=42, got %+v", result.Data)
	}
}

// --- Error responses ---

func TestUnmarshalCachedResponse_ErrorResponse(t *testing.T) {
	apiErr := apierror.NewValidationError("Invalid email")
	apiErr.Param = "email"
	errJSON, err := apiErr.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal APIError: %v", err)
	}

	meta := &appctx.IdempotencyResponseMetadata{}
	ctx := appctx.WithIdempotencyResponseMetadata(context.Background(), meta)

	result, err := UnmarshalCachedResponse[any](ctx, ptr(http.StatusBadRequest), json.RawMessage(errJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasCache {
		t.Fatal("expected HasCache=true")
	}
	if result.Data != nil {
		t.Error("expected Data=nil for error response")
	}
	if result.Error == nil {
		t.Fatal("expected Error to be non-nil")
	}
	if result.Error.Code != apierror.ErrorCodeValidationFailed {
		t.Errorf("expected code %q, got %q", apierror.ErrorCodeValidationFailed, result.Error.Code)
	}
	if result.Error.Param != "email" {
		t.Errorf("expected param %q, got %q", "email", result.Error.Param)
	}
	if !meta.Replayed {
		t.Error("expected Replayed=true after error unmarshal")
	}
}

func TestUnmarshalCachedResponse_500ErrorResponse(t *testing.T) {
	apiErr := apierror.NewInternalError(nil, "db connection failed")
	errJSON, err := apiErr.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal APIError: %v", err)
	}

	result, err := UnmarshalCachedResponse[any](context.Background(), ptr(http.StatusInternalServerError), json.RawMessage(errJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasCache {
		t.Fatal("expected HasCache=true")
	}
	if result.Error == nil {
		t.Fatal("expected Error to be non-nil for 500 response")
	}
	if result.Error.Code != apierror.ErrorCodeInternalError {
		t.Errorf("expected code %q, got %q", apierror.ErrorCodeInternalError, result.Error.Code)
	}
}

// --- No cache ---

func TestUnmarshalCachedResponse_NilStatusCode(t *testing.T) {
	result, err := UnmarshalCachedResponse[any](context.Background(), nil, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasCache {
		t.Error("expected HasCache=false when statusCode is nil")
	}
}

// --- Error cases ---

func TestUnmarshalCachedResponse_EmptyBody(t *testing.T) {
	_, err := UnmarshalCachedResponse[any](context.Background(), ptr(http.StatusOK), json.RawMessage{})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestUnmarshalCachedResponse_NilBody(t *testing.T) {
	_, err := UnmarshalCachedResponse[any](context.Background(), ptr(http.StatusOK), nil)
	if err == nil {
		t.Fatal("expected error for nil body")
	}
}

func TestUnmarshalCachedResponse_InvalidSuccessJSON(t *testing.T) {
	type Strict struct {
		ID int `json:"id"`
	}

	_, err := UnmarshalCachedResponse[Strict](context.Background(), ptr(http.StatusOK), json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON in success body")
	}
}

func TestUnmarshalCachedResponse_InvalidErrorJSON(t *testing.T) {
	_, err := UnmarshalCachedResponse[any](context.Background(), ptr(http.StatusBadRequest), json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON in error body")
	}
}

// --- Replayed flag without metadata in context ---

func TestUnmarshalCachedResponse_NoMetadataInContext(t *testing.T) {
	body := json.RawMessage(`{"id":"test"}`)
	// appctx.MarkIdempotencyReplayed is a no-op when no metadata is in context; should not panic.
	result, err := UnmarshalCachedResponse[map[string]string](context.Background(), ptr(http.StatusOK), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasCache {
		t.Error("expected HasCache=true")
	}
}

// --- Boundary: status code at 400 threshold ---

func TestUnmarshalCachedResponse_StatusCode399IsSuccess(t *testing.T) {
	body := json.RawMessage(`{"ok":true}`)
	result, err := UnmarshalCachedResponse[map[string]bool](context.Background(), ptr(399), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != nil {
		t.Error("expected no error for status 399")
	}
	if result.Data == nil || !(*result.Data)["ok"] {
		t.Error("expected data to be parsed for status 399")
	}
}

func TestUnmarshalCachedResponse_StatusCode400IsError(t *testing.T) {
	apiErr := apierror.NewValidationError("bad request")
	errJSON, _ := apiErr.ToJSON()

	result, err := UnmarshalCachedResponse[any](context.Background(), ptr(400), json.RawMessage(errJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil {
		t.Error("expected error for status 400")
	}
}
