package versiontransforms

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/apikit/version"
)

type seenRequest struct {
	method, path, query, body string
}

func recordRequest(seen *seenRequest) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen.method, seen.path, seen.query, seen.body = r.Method, r.URL.Path, r.URL.RawQuery, ""
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			seen.body = string(b)
		}
	})
}

func versionedRequest(method, target, body string, v version.APIVersion) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(apiversion.Header, v.String())
	return req
}

func TestRoutePreview9Analytics(t *testing.T) {
	var seen seenRequest
	h := RoutePreview9Analytics(recordRequest(&seen))

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPut, "/v1/core/analytics/oee", `{"starts_at":"x"}`, apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, seenRequest{http.MethodPost, "/v1/core/analytics/oee/actions/query", "", `{"starts_at":"x"}`}, seen)

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPut, "/v1/core/analytics/sales-lines", `{}`, apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, "/v1/core/analytics/sales-lines/actions/query", seen.path)
	assert.Equal(t, "limit=50", seen.query, "preview.9's default page size is kept")

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPut, "/v1/core/analytics/sales-lines?limit=10", `{}`, apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, "limit=10", seen.query)

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPut, "/v1/core/analytics/oee", `{}`, apiversion.V1_0_Forge))
	assert.Equal(t, seenRequest{http.MethodPut, "/v1/core/analytics/oee", "", `{}`}, seen, "forge callers are not rerouted")

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodGet, "/v1/core/analytics/weeks-of-sales", ``, apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, seenRequest{http.MethodGet, "/v1/core/analytics/weeks-of-sales", "", ""}, seen, "routes that did not move pass through")
}

func TestRoutePreview9PackList(t *testing.T) {
	var seen seenRequest
	h := RoutePreview9PackList(recordRequest(&seen))

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPost, preview9PackListRoute, `{"shipment_id":"shpm_1"}`, apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, seenRequest{http.MethodPost, "/v1/operations/shipments/shpm_1/actions/generate-pack-list", "", `{}`}, seen)

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPost, preview9PackListRoute, `{"other":1}`, apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, seenRequest{http.MethodPost, preview9PackListRoute, "", `{"other":1}`}, seen, "a body without shipment_id passes through")

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPost, preview9PackListRoute, `not json`, apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, seenRequest{http.MethodPost, preview9PackListRoute, "", `not json`}, seen)

	h.ServeHTTP(httptest.NewRecorder(), versionedRequest(http.MethodPost, preview9PackListRoute, `{"shipment_id":"shpm_1"}`, apiversion.V1_0_Forge))
	assert.Equal(t, seenRequest{http.MethodPost, preview9PackListRoute, "", `{"shipment_id":"shpm_1"}`}, seen, "forge callers are not rerouted")
}
