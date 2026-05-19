package apiendpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/version"
)

func bindHandler[TReq, TResp any](ep *APIEndpoint[TReq, TResp]) {
	ep.boundServiceHandler = ep.ServiceHandler(nil)
}

func decodeErrEnvelope(t *testing.T, w *httptest.ResponseRecorder) apierror.APIErrorResponse {
	t.Helper()
	var envelope apierror.APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v body=%s", err, w.Body.String())
	}
	return envelope
}

func TestExecute_MinVersion_rejectsOlderRequestVersion(t *testing.T) {
	t.Parallel()
	min := version.Latest
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		MinVersion:        &min,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run when version gate fails")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	older := version.APIVersion{
		Version:   "0.1.legacy-test",
		Minor:     0,
		Patch:     1,
		Codename:  "legacy",
		IsPreview: false,
	}
	bodyStr := `{"name":"x"}`
	ctx := appctx.WithAPIVersion(context.Background(), older)
	body := strings.NewReader(bodyStr)
	r := httptest.NewRequest(http.MethodPost, "/v1/things", body)
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(bodyStr))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Code != apierror.ErrorCodeAPIVersionTooOld {
		t.Fatalf("code=%q want api_version_too_old", env.Error.Code)
	}
}

func TestExecute_MinVersion_skipsWhenNoAPIVersionOnContext(t *testing.T) {
	t.Parallel()
	min := version.Latest
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		MinVersion:        &min,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_1", Name: "ok"}, nil
			}
		},
	}
	bindHandler(ep)

	body := strings.NewReader(`{"name":"x"}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/things", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestExecute_SkipRequestLogging_setsSkipSaveOnRequestLog(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		Extras:            APIEndpointExtras{SkipRequestLogging: true},
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	rl := &appctx.RequestLog{ID: "rq_skip"}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if !rl.SkipSave {
		t.Fatal("expected RequestLog.SkipSave")
	}
}

func TestExecute_olderAPIVersionRunsTransformAndDecodesJSON(t *testing.T) {
	t.Parallel()

	type verBodyReq struct {
		Name string `json:"name"`
	}

	older := version.APIVersion{
		Version:   "0.0.transform-smoke",
		Minor:     0,
		Patch:     0,
		Codename:  "smoke",
		IsPreview: true,
		Preview:   1,
	}

	ep := &APIEndpoint[*verBodyReq, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeUser,
		ServiceHandler: func(svc any) func(context.Context, *verBodyReq) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, req *verBodyReq) (*stubResponse, *apierror.APIError) {
				if req.Name != "from-body" {
					t.Fatalf("name=%q", req.Name)
				}
				return &stubResponse{ID: "th_ok", Name: req.Name}, nil
			}
		},
	}
	bindHandler(ep)

	payload, err := json.Marshal(map[string]string{"name": "from-body"})
	if err != nil {
		t.Fatal(err)
	}

	ctx := appctx.WithAPIVersion(context.Background(), older)
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(string(payload)))
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))

	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200 got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecute_pathBindingInvalid_reportsParameterInvalid(t *testing.T) {
	t.Parallel()

	type pathNumReq struct {
		Num int `path:"num"`
	}

	ep := &APIEndpoint[*pathNumReq, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things/{num}",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *pathNumReq) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *pathNumReq) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	ctx := appctx.WithPathParams(context.Background(), map[string]string{"num": "not-int"})
	r := httptest.NewRequest(http.MethodGet, "/v1/things/not-int", nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", w.Code)
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Param == nil || *env.Error.Param != "num" {
		t.Fatalf("param=%v", env.Error.Param)
	}
}

func TestExecute_queryBindingInvalid_reportsParameterInvalid(t *testing.T) {
	t.Parallel()

	type qLim struct {
		Limit int `query:"limit"`
	}

	ep := &APIEndpoint[*qLim, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *qLim) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *qLim) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things?limit=bogus", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestExecute_headerSchemeInvalid_reportsParameterInvalid(t *testing.T) {
	t.Parallel()

	type hdrTok struct {
		Token string `header:"Augno-Routing-Namespace" scheme:"Bearer"`
	}

	ep := &APIEndpoint[*hdrTok, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *hdrTok) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *hdrTok) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	r.Header.Set("Augno-Routing-Namespace", "Digest abc")
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestExecute_invalidIncludeValue_viaExecute_returns400(t *testing.T) {
	t.Parallel()

	type emptyQ struct{}

	ep := &APIEndpoint[*emptyQ, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		IncludeConfig:     roleConfig(),
		ServiceHandler: func(svc any) func(context.Context, *emptyQ) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *emptyQ) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things?include[]=nope_field", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecute_validInclude_passesIncludeSetToHandlerContext(t *testing.T) {
	t.Parallel()

	type emptyQ struct{}

	ep := &APIEndpoint[*emptyQ, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		IncludeConfig:     roleConfig(),
		ServiceHandler: func(svc any) func(context.Context, *emptyQ) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, _ *emptyQ) (*stubResponse, *apierror.APIError) {
				inc := appctx.GetRequestedIncludes(ctx)
				if inc == nil || !inc["role"] {
					t.Fatalf("includes=%#v", inc)
				}
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things?include[]=role", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
}

func TestExecute_IdempotencyKeyHeader_contextCarriesKey(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, _ *stubRequest) (*stubResponse, *apierror.APIError) {
				k, ok := appctx.GetIdempotencyKey(ctx)
				if !ok || k != "idem_client_1" {
					t.Fatalf("key=%q ok=%v", k, ok)
				}
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	r.Header.Set(header.IdempotencyKeyHeader, "idem_client_1")
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
}

func TestExecute_IdempotencyKey_fallsBackToRequestLogID(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, _ *stubRequest) (*stubResponse, *apierror.APIError) {
				k, ok := appctx.GetIdempotencyKey(ctx)
				if !ok || k != "rq_log_fallback" {
					t.Fatalf("key=%q ok=%v", k, ok)
				}
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	rl := &appctx.RequestLog{ID: "rq_log_fallback"}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
}

type rawBodyReq struct {
	Payload []byte `rawbody:"body"`
}

func TestExecute_SkipRequestBodyParsing_rawBodyBinding(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*rawBodyReq, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		Extras:            APIEndpointExtras{SkipRequestBodyParsing: true},
		ServiceHandler: func(svc any) func(context.Context, *rawBodyReq) (*stubResponse, *apierror.APIError) {
			return func(_ context.Context, req *rawBodyReq) (*stubResponse, *apierror.APIError) {
				if string(req.Payload) != "raw-bytes" {
					t.Fatalf("payload=%q", req.Payload)
				}
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader("raw-bytes"))
	r.ContentLength = int64(len("raw-bytes"))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}
}

type errReader struct{ err error }

func (e errReader) Read(p []byte) (int, error) {
	return 0, e.err
}

func TestExecute_SkipRequestBodyParsing_rawBodyReadError(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*rawBodyReq, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		Extras:            APIEndpointExtras{SkipRequestBodyParsing: true},
		ServiceHandler: func(svc any) func(context.Context, *rawBodyReq) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *rawBodyReq) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodPost, "/v1/things", nil)
	r.Body = io.NopCloser(errReader{err: fmt.Errorf("read failed")})
	r.ContentLength = 10
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecute_JSON_unknownField_returns400(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	payload := `{"name":"a","evil":true}`
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", w.Code)
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Code != apierror.ErrorCodeParameterUnknown {
		t.Fatalf("code=%v", env.Error.Code)
	}
}

func TestExecute_JSON_wrongScalarType_returnsInvalidFormat(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	payload := `{"name":123}`
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Code != apierror.ErrorCodeInvalidFormat {
		t.Fatalf("code=%v", env.Error.Code)
	}
}

func TestExecute_JSON_trailingGarbage_returns400(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	payload := `{"name":"ok"}{}`
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestExecute_requestLog_truncatesLargeBodyJSON(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	pad := strings.Repeat("n", 300*1024)
	payload, err := json.Marshal(map[string]string{"name": pad})
	if err != nil {
		t.Fatal(err)
	}

	rl := &appctx.RequestLog{ID: "rq_big"}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(string(payload)))
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
	if rl.BodyJSON == nil {
		t.Fatal("expected BodyJSON")
	}
	var meta struct {
		Truncated    bool `json:"_truncated"`
		OriginalSize int  `json:"_original_size"`
	}
	if err := json.Unmarshal([]byte(*rl.BodyJSON), &meta); err != nil {
		t.Fatalf("BodyJSON: %v", err)
	}
	if !meta.Truncated || meta.OriginalSize != len(payload) {
		t.Fatalf("truncation meta wrong: %+v", meta)
	}
}

func TestExecute_JSON_nullOnNullableFalse_returns400(t *testing.T) {
	t.Parallel()

	type nullDisallow struct {
		Title *string `json:"title,omitempty"`
	}

	ep := &APIEndpoint[*nullDisallow, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *nullDisallow) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *nullDisallow) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	payload := `{"title":null}`
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestExecute_PATCH_emptyObject_returns400(t *testing.T) {
	t.Parallel()

	type patchName struct {
		Name string `json:"name"`
	}

	ep := &APIEndpoint[*patchName, *stubResponse]{
		Method:            http.MethodPatch,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *patchName) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *patchName) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	payload := `{}`
	r := httptest.NewRequest(http.MethodPatch, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestExecute_bodyReadError_returns400(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodPost, "/v1/things", nil)
	r.Body = io.NopCloser(errReader{err: fmt.Errorf("simulated read failure")})
	r.ContentLength = 50
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

type enumBodyReq struct {
	Kind constants.ObjectType `json:"kind" validate:"required"`
}

func TestExecute_enumFieldInvalid_returns400(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*enumBodyReq, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *enumBodyReq) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *enumBodyReq) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	payload := `{"kind":"not_a_registered_object_kind_xyz"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

type requiredFieldReq struct {
	Name string `json:"name" validate:"required"`
}

func TestExecute_validateRequired_missingField_returns400(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*requiredFieldReq, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *requiredFieldReq) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *requiredFieldReq) (*stubResponse, *apierror.APIError) {
				t.Fatal("handler must not run")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	payload := `{}`
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Param == nil || *env.Error.Param != "name" {
		t.Fatalf("param=%v", env.Error.Param)
	}
}

func TestExecute_handlerNotFoundError_noLocationHeader(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusCreated,
		LocationFunc: func(*stubResponse) string {
			return "/v1/things/should-not-emit"
		},
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				return nil, apierror.NewResourceNotFoundError("missing")
			}
		},
	}
	bindHandler(ep)

	payload := `{"name":"x"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(payload))
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d", w.Code)
	}
	if w.Header().Get(header.LocationHeader) != "" {
		t.Fatalf("unexpected Location: %q", w.Header().Get(header.LocationHeader))
	}
}

func TestExecute_clientDisconnect_returns499(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(c context.Context, _ *stubRequest) (*stubResponse, *apierror.APIError) {
				cancel()
				<-c.Done()
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/things", http.NoBody)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != 499 {
		t.Fatalf("want 499 got %d body=%s", w.Code, w.Body.String())
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Code != apierror.ErrorCodeClientClosedRequest {
		t.Fatalf("code=%v", env.Error.Code)
	}
}

func TestExecute_success_setsCookiesAndReplayHeader(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(ctx context.Context, _ *stubRequest) (*stubResponse, *apierror.APIError) {
				appctx.AddCookies(ctx, []*http.Cookie{{Name: "sid", Value: "abc", Path: "/"}})
				appctx.SetHTTPReplayed(ctx, true)
				return &stubResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	if w.Header().Get(header.IdempotentReplayedHeader) != "true" {
		t.Fatalf("replay header=%q", w.Header().Get(header.IdempotentReplayedHeader))
	}
	res := w.Result()
	cookies := res.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "sid" || cookies[0].Value != "abc" {
		t.Fatalf("cookies=%v", cookies)
	}
}

func TestExecute_fileDownloadResponse(t *testing.T) {
	t.Parallel()

	type emptyQ struct{}

	ep := &APIEndpoint[*emptyQ, *httptransport.FileDownload]{
		Method:            http.MethodGet,
		Route:             "/v1/export.csv",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *emptyQ) (*httptransport.FileDownload, *apierror.APIError) {
			return func(context.Context, *emptyQ) (*httptransport.FileDownload, *apierror.APIError) {
				return &httptransport.FileDownload{
					ContentType: "text/csv",
					Filename:    "rows.csv",
					Body:        []byte("a,b,c\n1,2,3"),
				}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/export.csv", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "rows.csv") {
		t.Fatalf("Content-Disposition=%q", w.Header().Get("Content-Disposition"))
	}
	if w.Body.String() != "a,b,c\n1,2,3" {
		t.Fatalf("body=%q", w.Body.String())
	}
}

type expandableRoleNested struct {
	ID     string               `json:"id" validate:"required"`
	Object constants.ObjectType `json:"object" validate:"required"`
	Name   string               `json:"name" validate:"required"`
}

type includeExpandResp struct {
	ID     string                `json:"id"`
	Object constants.ObjectType  `json:"object"`
	Name   string                `json:"name"`
	Role   *expandableRoleNested `json:"role" expandable:"true"`
}

func TestExecute_includeTransform_collapsesUnrequestedRole(t *testing.T) {
	t.Parallel()

	roleStub := expandableRoleNested{
		ID:     "rl_1",
		Object: constants.ObjectTypeRole,
		Name:   "Admin",
	}
	ep := &APIEndpoint[*stubRequest, *includeExpandResp]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		IncludeConfig:     roleConfig(),
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*includeExpandResp, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*includeExpandResp, *apierror.APIError) {
				return &includeExpandResp{
					ID:     "apk_1",
					Object: constants.ObjectTypeAPIKey,
					Name:   "Key",
					Role:   &roleStub,
				}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["role"] != nil {
		t.Fatalf("expected unrequested role collapsed to null, got %#v", decoded["role"])
	}
}

func TestExecute_includeTransform_expandsRequestedRole(t *testing.T) {
	t.Parallel()

	roleStub := expandableRoleNested{
		ID:     "rl_1",
		Object: constants.ObjectTypeRole,
		Name:   "Admin",
	}
	ep := &APIEndpoint[*stubRequest, *includeExpandResp]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		IncludeConfig:     roleConfig(),
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*includeExpandResp, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*includeExpandResp, *apierror.APIError) {
				return &includeExpandResp{
					ID:     "apk_1",
					Object: constants.ObjectTypeAPIKey,
					Name:   "Key",
					Role:   &roleStub,
				}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things?include[]=role", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	role, ok := decoded["role"].(map[string]any)
	if !ok {
		t.Fatalf("role missing")
	}
	if role["name"] != "Admin" {
		t.Fatalf("want expanded role %+v", role)
	}
}

func TestExecute_includeTransform_invalidExpandableStub_returns500(t *testing.T) {
	t.Parallel()

	roleIncomplete := expandableRoleNested{
		ID:     "rl_bad",
		Object: constants.ObjectTypeRole,
		Name:   "", // violates validate:"required" when include forces validation
	}
	ep := &APIEndpoint[*stubRequest, *includeExpandResp]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		IncludeConfig:     roleConfig(),
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*includeExpandResp, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*includeExpandResp, *apierror.APIError) {
				return &includeExpandResp{
					ID:     "apk_1",
					Object: constants.ObjectTypeAPIKey,
					Name:   "Key",
					Role:   &roleIncomplete,
				}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things?include[]=role", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 got %d body=%s", w.Code, w.Body.String())
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Code != apierror.ErrorCodeInternalError {
		t.Fatalf("code=%v", env.Error.Code)
	}
}

func TestExecute_GET_jsonSuccess(t *testing.T) {
	t.Parallel()
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
				return &stubResponse{ID: "th_go", Name: "ok"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	w := httptest.NewRecorder()

	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	var decoded stubResponse
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != "th_go" || decoded.Name != "ok" {
		t.Fatalf("%+v", decoded)
	}
}
