package middleware

import (
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/internal/header"
	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/version"
)

func VersionMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip version validation for health check endpoint
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			versionStr := r.Header.Get(header.VersionHeader)

			// Require version header
			if versionStr == "" {
				apiErr := apierror.NewAPIVersionRequiredError()
				httptransport.RespondWithAPIError(r.Context(), w, apiErr)
				return
			}

			// Parse and validate version
			apiVersion, err := version.Parse(versionStr)
			if err != nil {
				supported := version.SupportedVersionStrings()
				apiErr := apierror.NewAPIVersionInvalidError(versionStr, supported)
				httptransport.RespondWithAPIError(r.Context(), w, apiErr)
				return
			}

			// Set parsed version in context
			ctx := appctx.WithAPIVersion(r.Context(), apiVersion)
			r = r.WithContext(ctx)

			// Echo version back in response header
			w.Header().Set(header.VersionHeader, apiVersion.String())

			next.ServeHTTP(w, r)
		}
	}
}
