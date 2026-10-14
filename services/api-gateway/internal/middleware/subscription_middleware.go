package middleware

import (
	"net/http"
	"strings"

	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/appctx"
)

func SubscriptionMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip health checks and billing endpoints so users can fix payment
			if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/v1/billing/") {
				next.ServeHTTP(w, r)
				return
			}

			identity, ok := appctx.Identity[*types.Identity](r.Context())
			if !ok || identity == nil || !identity.IsAuthenticated() || identity.SubscriptionStatus == nil {
				next.ServeHTTP(w, r)
				return
			}

			status := constants.SubscriptionStatus(*identity.SubscriptionStatus)
			switch status {
			case constants.SubscriptionStatusPastDue, constants.SubscriptionStatusCanceled, constants.SubscriptionStatusUnpaid:
				httptransport.RespondWithAPIError(r.Context(), w,
					apperr.NewPaymentRequiredError("Your subscription is "+string(status)+". Please update your billing information to continue."),
				)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
