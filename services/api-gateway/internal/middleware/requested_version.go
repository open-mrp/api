package middleware

import (
	"net/http"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/version"
)

// RequestedVersion puts the version a request names onto its context before routing, so every response is shaped
// for that version, including a 404 from the router, a 429 from the rate limit and a 401 from authentication. It
// never rejects a request: a missing or unknown version is refused later, after authentication, by apikit's
// middleware.Version, so an unauthenticated caller still gets its 401.
func RequestedVersion(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v, err := version.Parse(r.Header.Get(apiversion.Header)); err == nil {
			r = r.WithContext(appctx.WithAPIVersion(r.Context(), v))
		}
		next.ServeHTTP(w, r)
	})
}
