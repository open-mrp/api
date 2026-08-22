package middleware

import (
	"net/http"
	"strings"

	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// SandboxBillingMiddleware blocks sandbox accounts from accessing billing endpoints. Sandbox accounts inherit their owner's plan and should never interact with Stripe directly.
func SandboxBillingMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/v1/billing/") {
				next.ServeHTTP(w, r)
				return
			}

			identity, ok := appctx.GetIdentityFromContext(r.Context())
			if !ok || identity == nil {
				next.ServeHTTP(w, r)
				return
			}

			if identity.AccountMode == constants.AccountModeSandbox {
				httptransport.RespondWithAPIError(r.Context(), w,
					apierror.NewAuthorizationError("Billing operations are not available in sandbox mode."),
				)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
