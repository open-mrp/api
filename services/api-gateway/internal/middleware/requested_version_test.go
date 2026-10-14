package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/open-mrp/api/services/api-gateway/internal/versiontransforms"
	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/apikit/apierror"
	kitmiddleware "github.com/open-mrp/apikit/middleware"
	"github.com/open-mrp/apikit/ratelimit"
	kitrouter "github.com/open-mrp/apikit/router"
	"github.com/open-mrp/apikit/transport"
)

// versionedGateway is the gateway's order in miniature: the requested version is read before routing, then the
// rate limit, an authentication gate that refuses every caller, and the version requirement after it.
func versionedGateway() http.Handler {
	r := kitrouter.NewRouter()
	r.AddMiddleware(kitmiddleware.RateLimit(kitmiddleware.RateLimitConfig{Limiter: ratelimit.NewMemory(1, time.Minute)}))
	r.AddMiddleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			transport.RespondWithAPIError(req.Context(), w, apierror.NewAuthenticationError("Sign in."))
		}
	})
	r.AddMiddleware(kitmiddleware.Version())
	r.HandleEndpoint(http.MethodGet, "/v1/things", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }, true)
	return RequestedVersion(r)
}

func errorObject(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body), rr.Body.String())
	e, ok := body["error"].(map[string]any)
	require.True(t, ok, rr.Body.String())
	return e
}

func serve(h http.Handler, path, ver, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = ip + ":1234"
	if ver != "" {
		req.Header.Set(apiversion.Header, ver)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// Errors raised before the version requirement, by the router, the rate limit or authentication, still reach a
// preview.9 caller in the preview.9 shape.
func TestRequestedVersion_EarlyErrorsAreDowngraded(t *testing.T) {
	t.Parallel()
	h := versionedGateway()
	preview9 := apiversion.V1_0_Forge_Preview9.String()

	notFound := serve(h, "/v1/nowhere", preview9, "198.51.100.1")
	assert.Equal(t, http.StatusNotFound, notFound.Code)
	assert.NotContains(t, errorObject(t, notFound), "errors", "a router 404 is shaped for preview.9")
	assert.Contains(t, errorObject(t, notFound), "request_log_url")

	unauthenticated := serve(h, "/v1/things", preview9, "198.51.100.2")
	assert.Equal(t, http.StatusUnauthorized, unauthenticated.Code)
	assert.NotContains(t, errorObject(t, unauthenticated), "errors", "an authentication 401 is shaped for preview.9")

	limited := serve(h, "/v1/things", preview9, "198.51.100.2")
	assert.Equal(t, http.StatusTooManyRequests, limited.Code)
	assert.Equal(t, "rate_limit_exceeded", errorObject(t, limited)["code"], "a 429 is shaped for preview.9")

	forge := serve(h, "/v1/nowhere", apiversion.V1_0_Forge.String(), "198.51.100.3")
	assert.Contains(t, errorObject(t, forge), "errors", "a forge caller gets the forge error object")
}

// It never refuses a request: without a version, or with an unknown one, authentication still answers first.
func TestRequestedVersion_LeavesTheRequirementToItsMiddleware(t *testing.T) {
	t.Parallel()
	h := versionedGateway()
	for i, ver := range []string{"", "1.0.nonsense"} {
		rr := serve(h, "/v1/things", ver, fmt.Sprintf("198.51.100.%d", 10+i))
		assert.Equal(t, http.StatusUnauthorized, rr.Code, "version %q", ver)
		assert.Contains(t, errorObject(t, rr), "errors", "an unparsed version leaves the latest shape")
	}
}
