package middleware

import (
	"net/http"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/omrpctx"
)

func PlatformMiddleware(platform constants.PlatformMode) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := omrpctx.WithPlatform(r.Context(), platform)
			next(w, r.WithContext(ctx))
		}
	}
}
