package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type testRequest struct {
	ID    int    `json:"id" query:"id" path:"id" validate:"required"`
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type enumTestRequest struct {
	Mode constants.AccountMode `json:"mode" validate:"required"`
	Name string                `json:"name" validate:"required"`
}

func TestValidateEnumFields(t *testing.T) {
	t.Run("Valid enum value", func(t *testing.T) {
		req := &enumTestRequest{
			Mode: constants.AccountModeProduction,
			Name: "Test",
		}
		err := ValidateEnumFields(req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Invalid enum value", func(t *testing.T) {
		req := &enumTestRequest{
			Mode: constants.AccountMode("invalid"),
			Name: "Test",
		}
		err := ValidateEnumFields(req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Param != "mode" {
			t.Errorf("expected param 'mode', got %s", err.Param)
		}
	})

	t.Run("Empty enum value", func(t *testing.T) {
		req := &enumTestRequest{
			Mode: constants.AccountMode(""),
			Name: "Test",
		}
		err := ValidateEnumFields(req)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Non-enum struct field is ignored", func(t *testing.T) {
		type nonEnumRequest struct {
			Name string `json:"name"`
		}
		req := &nonEnumRequest{Name: "anything"}
		err := ValidateEnumFields(req)
		if err != nil {
			t.Errorf("expected no error for non-enum field, got %v", err)
		}
	})
}

type rawBodyTestRequest struct {
	RawBody   []byte `rawbody:"true"`
	Signature string `header:"X-Signature"`
}

func TestBindRawBody(t *testing.T) {
	t.Run("Successfully binds raw body", func(t *testing.T) {
		body := []byte(`{"webhook":"payload","event":"test"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))

		dst := &rawBodyTestRequest{}
		err := BindRawBody(req, dst)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !bytes.Equal(dst.RawBody, body) {
			t.Errorf("expected body %s, got %s", body, dst.RawBody)
		}
	})

	t.Run("Binds empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte{}))

		dst := &rawBodyTestRequest{}
		err := BindRawBody(req, dst)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(dst.RawBody) != 0 {
			t.Errorf("expected empty body, got %s", dst.RawBody)
		}
	})

	t.Run("Ignores fields without rawbody tag", func(t *testing.T) {
		type noTagRequest struct {
			Body []byte
			Name string
		}

		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("test")))
		dst := &noTagRequest{}
		err := BindRawBody(req, dst)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if dst.Body != nil {
			t.Errorf("expected nil body, got %v", dst.Body)
		}
	})

	t.Run("Errors on non-byte-slice field with rawbody tag", func(t *testing.T) {
		type invalidTagRequest struct {
			Body string `rawbody:"true"`
		}

		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("test")))
		dst := &invalidTagRequest{}
		err := BindRawBody(req, dst)

		if err == nil {
			t.Error("expected error for non-[]byte field with rawbody tag")
		}
	})
}

func TestBindFromQuery(t *testing.T) {
	t.Run("Binds valid query parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?id=123", nil)
		dst := &testRequest{}
		err := BindFromQuery(req.URL, dst)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if dst.ID != 123 {
			t.Errorf("expected ID 123, got %d", dst.ID)
		}
	})

	t.Run("Returns error for invalid type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?id=not-an-int", nil)
		dst := &testRequest{}
		err := BindFromQuery(req.URL, dst)
		if err == nil {
			t.Error("expected error for invalid int, got nil")
		}
	})
}

func TestDecodeJSONInto(t *testing.T) {
	t.Run("Decodes valid JSON", func(t *testing.T) {
		body := `{"id": 1, "name": "Test", "email": "test@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		dst := &testRequest{}
		err := DecodeJSONInto(dst, req, true)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if dst.ID != 1 || dst.Name != "Test" || dst.Email != "test@example.com" {
			t.Errorf("unexpected values: %+v", dst)
		}
	})

	t.Run("Returns error for unknown fields when disallowed", func(t *testing.T) {
		body := `{"id": 1, "name": "Test", "email": "test@example.com", "unknown": "field"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		dst := &testRequest{}
		err := DecodeJSONInto(dst, req, true)
		if err == nil {
			t.Error("expected error for unknown field, got nil")
		}
		apiErr, ok := err.(*apierror.APIError)
		if !ok {
			t.Errorf("expected APIError, got %T", err)
			return
		}
		if apiErr.Param != "unknown" {
			t.Errorf("expected param 'unknown', got %s", apiErr.Param)
		}
	})

	t.Run("Allows unknown fields when not disallowed", func(t *testing.T) {
		body := `{"id": 1, "name": "Test", "email": "test@example.com", "unknown": "field"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		dst := &testRequest{}
		err := DecodeJSONInto(dst, req, false)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Returns error for type mismatch", func(t *testing.T) {
		body := `{"id": "not-an-int", "name": "Test", "email": "test@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		dst := &testRequest{}
		err := DecodeJSONInto(dst, req, true)
		if err == nil {
			t.Error("expected error for type mismatch, got nil")
		}
	})
}

func TestShouldDecodeBody(t *testing.T) {
	t.Run("Returns true for POST with content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("body"))
		if !ShouldDecodeBody(req) {
			t.Error("expected true for POST with content")
		}
	})

	t.Run("Returns true for PUT with content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewBufferString("body"))
		if !ShouldDecodeBody(req) {
			t.Error("expected true for PUT with content")
		}
	})

	t.Run("Returns true for PATCH with content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/test", bytes.NewBufferString("body"))
		if !ShouldDecodeBody(req) {
			t.Error("expected true for PATCH with content")
		}
	})

	t.Run("Returns false for GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		if ShouldDecodeBody(req) {
			t.Error("expected false for GET")
		}
	})

	t.Run("Returns false for empty POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.ContentLength = 0
		if ShouldDecodeBody(req) {
			t.Error("expected false for empty POST")
		}
	})
}

func TestAllocIfPtr(t *testing.T) {
	t.Run("Allocates nil pointer", func(t *testing.T) {
		var ptr *testRequest
		result := AllocIfPtr(ptr)
		if result == nil {
			t.Error("expected non-nil pointer")
		}
	})

	t.Run("Keeps non-nil pointer", func(t *testing.T) {
		ptr := &testRequest{ID: 42}
		result := AllocIfPtr(ptr)
		if result.ID != 42 {
			t.Errorf("expected ID 42, got %d", result.ID)
		}
	})
}

func TestBindFromHeaders(t *testing.T) {
	type headerRequest struct {
		Auth string `header:"Authorization" scheme:"Bearer"`
	}

	t.Run("Binds authorization header with Bearer scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer token123")
		dst := &headerRequest{}
		err := BindFromHeaders(req, dst)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if dst.Auth != "token123" {
			t.Errorf("expected 'token123', got '%s'", dst.Auth)
		}
	})
}

func TestApplyPagination(t *testing.T) {
	t.Run("Parses valid pagination params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?limit=20&cursor=abc123&q=search", nil)
		params, err := ApplyPagination(req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if params.Limit != 20 {
			t.Errorf("expected limit 20, got %d", params.Limit)
		}
		if params.Cursor == nil || *params.Cursor != "abc123" {
			t.Errorf("expected cursor 'abc123', got %v", params.Cursor)
		}
		if params.Query == nil || *params.Query != "search" {
			t.Errorf("expected query 'search', got %v", params.Query)
		}
	})

	t.Run("Uses defaults when params missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		params, err := ApplyPagination(req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if params.Limit != 10 {
			t.Errorf("expected default limit 10, got %d", params.Limit)
		}
		if params.Cursor != nil {
			t.Errorf("expected nil cursor, got %v", params.Cursor)
		}
	})

	t.Run("Returns error for invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?limit=invalid", nil)
		_, err := ApplyPagination(req)
		if err == nil {
			t.Error("expected error for invalid limit")
		}
	})

	t.Run("Returns error for negative limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?limit=-1", nil)
		_, err := ApplyPagination(req)
		if err == nil {
			t.Error("expected error for negative limit")
		}
	})
}

func TestRespondWithJSON(t *testing.T) {
	t.Run("Writes JSON response", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx := context.Background()
		payload := map[string]string{"message": "hello"}

		RespondWithJSON(ctx, w, http.StatusOK, payload)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
		}

		var result map[string]string
		json.Unmarshal(w.Body.Bytes(), &result)
		if result["message"] != "hello" {
			t.Errorf("expected message 'hello', got %s", result["message"])
		}
	})
}
