package apiendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/header"
	apierror "github.com/augno/api/shared/errors"
)

type stubRequest struct {
	Name string `json:"name"`
}

type stubResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestExecute_ErrorResponse_NoLocationHeader(t *testing.T) {
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusCreated,
		LocationFunc: func(resp *stubResponse) string {
			return "/v1/things/" + resp.ID
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, req *stubRequest) (*stubResponse, *apierror.APIError) {
				return nil, apierror.NewValidationError("name is required")
			}
		},
	}

	ep.boundServiceHandler = ep.ServiceHandler(nil)

	body := strings.NewReader(`{"name":""}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/things", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Header().Get(header.LocationHeader) != "" {
		t.Fatalf("expected no Location header on error response, got %q", w.Header().Get(header.LocationHeader))
	}
	if w.Code == http.StatusCreated {
		t.Fatalf("expected error status code, got %d", w.Code)
	}
}

func TestExecute_SuccessResponse_SetsLocationHeader(t *testing.T) {
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusCreated,
		LocationFunc: func(resp *stubResponse) string {
			return "/v1/things/" + resp.ID
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, req *stubRequest) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_123", Name: "test"}, nil
			}
		},
	}

	ep.boundServiceHandler = ep.ServiceHandler(nil)

	body := strings.NewReader(`{"name":"test"}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/things", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	if w.Header().Get(header.LocationHeader) != "/v1/things/th_123" {
		t.Fatalf("expected Location %q, got %q", "/v1/things/th_123", w.Header().Get(header.LocationHeader))
	}
}
