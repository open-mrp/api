package apiendpoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/api-gateway/internal/header"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/redact"
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

func TestCollectIncludeQueryValues_SplitsCommaSeparatedIncludeArrayValues(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/v1/core/audit-events?include[]=actor,changes", nil)
	values := collectIncludeQueryValues(r)

	expected := map[string]bool{"actor": true, "changes": true}
	got := make(map[string]bool, len(values))
	for _, v := range values {
		got[v] = true
	}
	if len(got) != len(expected) || !got["actor"] || !got["changes"] {
		t.Fatalf("expected actor and changes, got: %v", values)
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

func TestExecute_redactsSensitiveRequestBodyIntoRequestLog(t *testing.T) {
	t.Parallel()
	type reqSens struct {
		Name   string `json:"name"`
		Secret string `json:"secret" sensitive:"true"` // #nosec G101 — test fixture
	}
	ep := &APIEndpoint[*reqSens, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *reqSens) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, r *reqSens) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_1", Name: r.Name}, nil
			}
		},
	}
	ep.boundServiceHandler = ep.ServiceHandler(nil)
	ep.sensitiveReqPaths = redact.SensitiveFields(reflect.TypeFor[*reqSens]())

	rl := &appctx.RequestLog{ID: "rq_test123"}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	body := strings.NewReader(`{"name":"alice","secret":"topsecret"}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/things", body)
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if rl.BodyJSON == nil {
		t.Fatal("expected BodyJSON on request log")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(*rl.BodyJSON), &got); err != nil {
		t.Fatalf("BodyJSON: %v", err)
	}
	if got["name"] != "alice" || got["secret"] != "****" {
		t.Fatalf("BodyJSON %v", got)
	}
}

func TestExecute_malformedJSONWithSensitivePaths_omitsBodyJSON(t *testing.T) {
	t.Parallel()
	type reqSens struct {
		Secret string `json:"secret" sensitive:"true"` // #nosec G101 — test fixture
	}
	ep := &APIEndpoint[*reqSens, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *reqSens) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *reqSens) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	ep.boundServiceHandler = ep.ServiceHandler(nil)
	ep.sensitiveReqPaths = redact.SensitiveFields(reflect.TypeFor[*reqSens]())

	rl := &appctx.RequestLog{ID: "rq_ab"}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	body := strings.NewReader(`{"secret":"x"`) // truncated JSON
	r := httptest.NewRequest(http.MethodPost, "/v1/things", body)
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected validation error status, got %d", w.Code)
	}
	if rl.BodyJSON != nil {
		t.Fatalf("malformed payload must not populate BodyJSON when redaction fails; got %q", *rl.BodyJSON)
	}
}

type stubResponseWithSecret struct {
	Name   string `json:"name"`
	Secret string `json:"secret" sensitive:"true"` // #nosec G101 — test fixture
}

func TestExecute_populatesSensitiveResponseFieldsOnRequestLog(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponseWithSecret]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponseWithSecret, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponseWithSecret, *apierror.APIError) {
				return &stubResponseWithSecret{Name: "ok", Secret: "raw"}, nil
			}
		},
	}
	ep.boundServiceHandler = ep.ServiceHandler(nil)
	ep.sensitiveRespPaths = redact.SensitiveFields(reflect.TypeFor[*stubResponseWithSecret]())

	rl := &appctx.RequestLog{ID: "rq_z"}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if !rl.SensitiveResponseFields["secret"] || len(rl.SensitiveResponseFields) != 1 {
		t.Fatalf("SensitiveResponseFields = %#v", rl.SensitiveResponseFields)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
}

func TestCollectIncludeQueryValues_MergesIncludeAndIncludeArrayFormats(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/v1/core/audit-events?include=actor&include[]=changes", nil)
	values := collectIncludeQueryValues(r)

	expected := map[string]bool{"actor": true, "changes": true}
	got := make(map[string]bool, len(values))
	for _, v := range values {
		got[v] = true
	}
	if len(got) != len(expected) || !got["actor"] || !got["changes"] {
		t.Fatalf("expected actor and changes, got: %v", values)
	}
}
