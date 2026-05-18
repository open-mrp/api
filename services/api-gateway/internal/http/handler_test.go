package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/augno/api/shared/appctx"
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestRejectUnknownQueryParams(t *testing.T) {
	t.Parallel()

	type listReq struct {
		Cursor *string `query:"cursor"`
		Limit  int32   `query:"limit" default:"100"`
	}

	t.Run("allows declared keys only", func(t *testing.T) {
		u := mustParseURL(t, "/items?cursor=c1&limit=10")
		dst := &listReq{}
		if err := BindFromQuery(u, dst); err != nil {
			t.Fatalf("BindFromQuery: %v", err)
		}
		if apiErr := RejectUnknownQueryParams(u, dst, false); apiErr != nil {
			t.Fatalf("expected nil, got %v", apiErr)
		}
	})

	t.Run("rejects undeclared key", func(t *testing.T) {
		u := mustParseURL(t, "/items?cursor=c1&unexpected=1")
		dst := &listReq{}
		if err := BindFromQuery(u, dst); err != nil {
			t.Fatalf("BindFromQuery: %v", err)
		}
		apiErr := RejectUnknownQueryParams(u, dst, false)
		if apiErr == nil {
			t.Fatal("expected error")
		}
		if apiErr.Code != apierror.ErrorCodeParameterUnknown {
			t.Fatalf("expected parameter_unknown, got %s", apiErr.Code)
		}
		if apiErr.Param != "unexpected" {
			t.Fatalf("expected param unexpected, got %q", apiErr.Param)
		}
	})

	t.Run("allowInclude permits include and include bracket form", func(t *testing.T) {
		u := mustParseURL(t, "/items?include=role&include[]=department")
		dst := &listReq{}
		if err := BindFromQuery(u, dst); err != nil {
			t.Fatalf("BindFromQuery: %v", err)
		}
		if apiErr := RejectUnknownQueryParams(u, dst, true); apiErr != nil {
			t.Fatalf("expected nil, got %v", apiErr)
		}
	})

	t.Run("slice field allows bracket query key", func(t *testing.T) {
		type withSlice struct {
			Tags []string `query:"tags"`
		}
		u := mustParseURL(t, "/x?tags[]=a&tags[]=b")
		dst := &withSlice{}
		if err := BindFromQuery(u, dst); err != nil {
			t.Fatalf("BindFromQuery: %v", err)
		}
		if apiErr := RejectUnknownQueryParams(u, dst, false); apiErr != nil {
			t.Fatalf("expected nil, got %v", apiErr)
		}
	})
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return u
}

func TestDecodeJSONInto(t *testing.T) {
	t.Parallel()
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

type unwrapTestType struct{}

func (*unwrapTestType) UnmarshalJSON([]byte) error {
	return fmt.Errorf("wrap: %w", apierror.NewInvalidFormatError("bad", "field"))
}

func TestDecodeJSONInto_unwrapsWrappedAPIError(t *testing.T) {
	t.Parallel()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	var dst any = &unwrapTestType{}
	err := DecodeJSONInto(dst, req, true)
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *apierror.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *apierror.APIError via errors.As, got %T: %v", err, err)
	}
	if ae.Code != apierror.ErrorCodeInvalidFormat {
		t.Fatalf("expected invalid_format, got %s", ae.Code)
	}
}

func TestShouldDecodeBody(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// patchRequest mimics a PATCH endpoint with optional nested struct pointers.
type patchRequest struct {
	ID     string       `path:"id" validate:"required"`
	Name   *string      `json:"name,omitempty"`
	Config *nestedInput `json:"config,omitempty"`
	Items  *[]string    `json:"items,omitempty"`
}

type nestedInput struct {
	Model       *string   `json:"model,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	Sub         *subInput `json:"sub,omitempty"`
}

type subInput struct {
	Schedule *string `json:"schedule,omitempty"`
}

func newRequestWithPathParams(method, url string, body *bytes.Buffer, params map[string]string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, body)
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	ctx := appctx.WithPathParams(req.Context(), params)
	return req.WithContext(ctx)
}

func TestWalkStruct_doesNotInitNilPointerToStruct(t *testing.T) {
	t.Parallel()
	t.Run("BindFromPath preserves nil pointer-to-struct fields", func(t *testing.T) {
		req := newRequestWithPathParams(http.MethodPatch, "/agents/123", nil, map[string]string{"id": "123"})
		dst := &patchRequest{}
		err := BindFromPath(req, dst)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dst.ID != "123" {
			t.Errorf("expected ID '123', got '%s'", dst.ID)
		}
		if dst.Config != nil {
			t.Error("expected Config to remain nil after BindFromPath")
		}
		if dst.Items != nil {
			t.Error("expected Items to remain nil after BindFromPath")
		}
	})

	t.Run("BindFromHeaders preserves nil pointer-to-struct fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/agents/123", nil)
		dst := &patchRequest{}
		err := BindFromHeaders(req, dst)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dst.Config != nil {
			t.Error("expected Config to remain nil after BindFromHeaders")
		}
	})

	t.Run("BindFromQuery preserves nil pointer-to-struct fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/agents/123", nil)
		dst := &patchRequest{}
		err := BindFromQuery(req.URL, dst)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dst.Config != nil {
			t.Error("expected Config to remain nil after BindFromQuery")
		}
	})

	t.Run("Non-nil pointer-to-struct is still traversed", func(t *testing.T) {
		req := newRequestWithPathParams(http.MethodPatch, "/agents/123", nil, map[string]string{"id": "123"})
		model := "claude-sonnet-4"
		dst := &patchRequest{
			Config: &nestedInput{Model: &model},
		}
		err := BindFromPath(req, dst)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dst.Config == nil {
			t.Fatal("expected Config to remain non-nil")
		}
		if dst.Config.Model == nil || *dst.Config.Model != "claude-sonnet-4" {
			t.Error("expected Config.Model to be preserved")
		}
		if dst.Config.Sub != nil {
			t.Error("expected Config.Sub to remain nil")
		}
	})
}

func TestPatchRequest_JSONDecodePreservesNilStructPointers(t *testing.T) {
	t.Parallel()
	t.Run("Absent config stays nil after full bind+decode flow", func(t *testing.T) {
		body := `{"name": "Updated"}`
		httpReq := newRequestWithPathParams(http.MethodPatch, "/agents/123", bytes.NewBufferString(body), map[string]string{"id": "123"})

		dst := &patchRequest{}

		// Simulate the api_endpoint.go flow: single incoming bind + JSON decode
		if err := BindIncomingRequest(httpReq, dst, false); err != nil {
			t.Fatalf("BindIncomingRequest: %v", err)
		}
		if err := DecodeJSONInto(dst, httpReq, false); err != nil {
			t.Fatalf("DecodeJSONInto: %v", err)
		}

		if dst.ID != "123" {
			t.Errorf("expected ID '123', got '%s'", dst.ID)
		}
		if dst.Name == nil || *dst.Name != "Updated" {
			t.Errorf("expected Name 'Updated', got %v", dst.Name)
		}
		if dst.Config != nil {
			t.Errorf("expected Config nil, got %+v", dst.Config)
		}
		if dst.Items != nil {
			t.Errorf("expected Items nil, got %v", dst.Items)
		}
	})

	t.Run("Provided config is decoded correctly", func(t *testing.T) {
		body := `{"name": "Updated", "config": {"model": "claude-sonnet-4", "temperature": 0.5}}`
		httpReq := newRequestWithPathParams(http.MethodPatch, "/agents/123", bytes.NewBufferString(body), map[string]string{"id": "123"})

		dst := &patchRequest{}

		if err := BindIncomingRequest(httpReq, dst, false); err != nil {
			t.Fatalf("BindIncomingRequest: %v", err)
		}
		if err := DecodeJSONInto(dst, httpReq, false); err != nil {
			t.Fatalf("DecodeJSONInto: %v", err)
		}

		if dst.Config == nil {
			t.Fatal("expected Config to be non-nil")
		}
		if dst.Config.Model == nil || *dst.Config.Model != "claude-sonnet-4" {
			t.Errorf("expected Model 'claude-sonnet-4', got %v", dst.Config.Model)
		}
		if dst.Config.Temperature == nil || *dst.Config.Temperature != 0.5 {
			t.Errorf("expected Temperature 0.5, got %v", dst.Config.Temperature)
		}
		if dst.Config.Sub != nil {
			t.Error("expected Config.Sub to remain nil")
		}
	})

	t.Run("Empty config object is non-nil with nil inner fields", func(t *testing.T) {
		body := `{"config": {}}`
		httpReq := newRequestWithPathParams(http.MethodPatch, "/agents/123", bytes.NewBufferString(body), map[string]string{"id": "123"})

		dst := &patchRequest{}
		if err := BindIncomingRequest(httpReq, dst, false); err != nil {
			t.Fatalf("BindIncomingRequest: %v", err)
		}
		_ = DecodeJSONInto(dst, httpReq, false)

		if dst.Config == nil {
			t.Fatal("expected Config to be non-nil for empty object")
		}
		if dst.Config.Model != nil {
			t.Error("expected Config.Model to be nil")
		}
	})
}

func TestRespondWithJSON(t *testing.T) {
	t.Parallel()
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
