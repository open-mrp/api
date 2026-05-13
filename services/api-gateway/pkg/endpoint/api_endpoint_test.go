package apiendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/shared/constants"
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
	t.Parallel()
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
	t.Parallel()
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

func TestParseIncludeParams_SplitsCommaSeparatedIncludeArrayValues(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		IncludeConfig: IncludesFor(IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"actor", "changes"},
		}),
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/core/audit-events?include[]=actor,changes", nil)
	requested, apiErr := ep.parseIncludeParams(r)
	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	if !requested["actor"] || !requested["changes"] {
		t.Fatalf("expected actor and changes to be requested, got: %#v", requested)
	}
}

func TestExecute_RejectsUnknownQueryParameter(t *testing.T) {
	t.Parallel()
	type getReq struct {
		X string `query:"x"`
	}
	ep := &APIEndpoint[*getReq, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(ctx context.Context, req *getReq) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, req *getReq) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_1", Name: "ok"}, nil
			}
		},
	}
	ep.boundServiceHandler = ep.ServiceHandler(nil)

	r := httptest.NewRequest(http.MethodGet, "/v1/things?x=a&not_allowed=1", nil)
	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseIncludeParams_MergesIncludeAndIncludeArrayFormats(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		IncludeConfig: IncludesFor(IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"actor", "changes"},
		}),
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/core/audit-events?include=actor&include[]=changes", nil)
	requested, apiErr := ep.parseIncludeParams(r)
	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	if !requested["actor"] || !requested["changes"] {
		t.Fatalf("expected actor and changes to be requested, got: %#v", requested)
	}
}
