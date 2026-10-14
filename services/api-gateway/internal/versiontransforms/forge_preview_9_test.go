package versiontransforms

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/version"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(s), &m))
	return m
}

func downgrade(t *testing.T, ot constants.ObjectType, body string) map[string]any {
	t.Helper()
	return version.Transform(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, ot, decode(t, body))
}

func TestPreview9_PageInfoBooleans(t *testing.T) {
	t.Parallel()

	got := downgrade(t, "", `{"object":"list","page_info":{"next_page_url":"/v1/x?cursor=a","previous_page_url":null},"data":[
		{"id":"c_1","lines":{"object":"list","page_info":{"next_page_url":null,"previous_page_url":"/v1/y"},"data":[]}}]}`)
	info := got["page_info"].(map[string]any)
	assert.Equal(t, true, info["has_next_page"])
	assert.Equal(t, false, info["has_prev_page"])
	nested := got["data"].([]any)[0].(map[string]any)["lines"].(map[string]any)["page_info"].(map[string]any)
	assert.Equal(t, false, nested["has_next_page"])
	assert.Equal(t, true, nested["has_prev_page"])
}

func TestPreview9_LimitOverTheMaximumIsCapped(t *testing.T) {
	t.Parallel()

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeCustomer, "/v1/sales/customers", url.Values{"limit": {"1000"}})
	assert.Equal(t, "100", q.Get("limit"))
	q = version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeCustomer, "/v1/sales/customers", url.Values{"limit": {"40"}})
	assert.Equal(t, "40", q.Get("limit"))
}

func TestPreview9_DeletedStubIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]any{}, downgrade(t, "", `{"id":"pt_1","object":"payment_term","deleted":true}`))
	kept := downgrade(t, constants.ObjectTypePaymentTerm, `{"id":"pt_1","object":"payment_term","name":"Net 30"}`)
	assert.Equal(t, "Net 30", kept["name"])
}

func TestPreview9_AsyncJobIsAJob(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeProduct, `{"id":"jb_1","object":"async_job","status":"canceled","canceled_at":"2026-10-14T00:00:00Z",
		"results":{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[{"object":"async_job_result","index":0,"status":"created"}]},
		"export":{"object":"async_job_export","url":"https://example.com/f"}}`)
	assert.Equal(t, "job", got["object"])
	assert.Equal(t, "cancelled", got["status"])
	assert.Equal(t, "2026-10-14T00:00:00Z", got["cancelled_at"])
	assert.NotContains(t, got, "canceled_at")
	assert.Equal(t, "job_result", got["results"].(map[string]any)["data"].([]any)[0].(map[string]any)["object"])
	assert.Equal(t, "created", got["results"].(map[string]any)["data"].([]any)[0].(map[string]any)["status"], "a row's created is unchanged")
	assert.Equal(t, "job_export", got["export"].(map[string]any)["object"])

	for forge, preview := range map[string]string{"queued": "created", "running": "started", "completed": "completed", "failed": "failed"} {
		assert.Equal(t, preview, downgrade(t, "", `{"object":"async_job","status":"`+forge+`"}`)["status"])
	}
}

func TestPreview9_JobRoutes(t *testing.T) {
	t.Parallel()

	var seen string
	h := RoutePreview9Jobs(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.Method + " " + r.URL.Path }))
	for _, tc := range []struct{ method, path, version, want string }{
		{http.MethodGet, "/v1/core/jobs/jb_1", "1.0.forge-preview.9", "GET /v1/core/async-jobs/jb_1"},
		{http.MethodPost, "/v1/core/jobs/jb_1/cancel", "1.0.forge-preview.9", "POST /v1/core/async-jobs/jb_1/actions/cancel"},
		{http.MethodGet, "/v1/core/jobs/jb_1", "1.0.forge", "GET /v1/core/jobs/jb_1"},
		{http.MethodGet, "/v1/core/async-jobs/jb_1", "1.0.forge-preview.9", "GET /v1/core/async-jobs/jb_1"},
	} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		r.Header.Set(apiversion.Header, tc.version)
		h.ServeHTTP(httptest.NewRecorder(), r)
		assert.Equal(t, tc.want, seen)
	}
}

func errorBody(t *testing.T, apiErr *apierror.APIError) (int, map[string]any) {
	t.Helper()
	b, err := json.Marshal(apiErr.Response())
	require.NoError(t, err)
	return version.TransformError(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, apiErr.Status(), decode(t, string(b)))
}

func TestPreview9_ErrorShapeAndStatus(t *testing.T) {
	t.Parallel()

	status, body := errorBody(t, apierror.NewMissingFieldError("name", "name is required."))
	e := body["error"].(map[string]any)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "missing_field", e["code"], "a single failing field keeps its own code")
	assert.Equal(t, "name", e["param"])
	assert.NotContains(t, e, "errors")
	assert.Contains(t, e, "request_log_url")
	assert.Contains(t, e, "quota")

	status, body = errorBody(t, apierror.NewValidationError("2 fields are invalid.",
		apierror.Field("a", apierror.CodeMissingField, "a"), apierror.Field("b", apierror.CodeInvalidFormat, "b")))
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "validation_failed", body["error"].(map[string]any)["code"])

	for forge, preview := range map[*apierror.APIError]string{
		apierror.NewRateLimitedError("slow down"):                                   "rate_limit_exceeded",
		apierror.NewIdempotencyKeyReusedError("k"):                                  "validation_failed",
		apierror.NewInUseError("in use"):                                            "resource_conflict",
		apierror.NewParameterInvalidError("", "bad JSON"):                           "validation_failed",
		apierror.NewParameterInvalidError("limit", "bad limit"):                     "parameter_invalid",
		apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor."):   "invalid_format",
		apierror.NewInvalidFormatError("kind", "Field 'kind' must be one of: a, b"): "parameter_invalid",
		apierror.NewNotFoundError("missing"):                                        "resource_not_found",
	} {
		_, body := errorBody(t, forge)
		assert.Equal(t, preview, body["error"].(map[string]any)["code"], string(forge.Code))
	}

	status, _ = errorBody(t, apierror.NewIdempotencyKeyReusedError("k"))
	assert.Equal(t, http.StatusBadRequest, status)
	status, _ = errorBody(t, apierror.NewRateLimitedError("slow down"))
	assert.Equal(t, http.StatusTooManyRequests, status)
}
