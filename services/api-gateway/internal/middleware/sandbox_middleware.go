package middleware

import (
	"net/http"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	httptransport "github.com/open-mrp/apikit/transport"
)

// SandboxBillingMiddleware blocks sandbox accounts from accessing billing endpoints. Sandbox accounts inherit their owner's plan and should never interact with Stripe directly.
func SandboxBillingMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/v1/billing/") {
				next.ServeHTTP(w, r)
				return
			}

			identity, ok := appctx.Identity[*types.Identity](r.Context())
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
