package middleware

import (
	"net/http"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
)

func PlatformMiddleware(platform constants.PlatformMode) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := appctx.WithPlatform(r.Context(), platform)
			next(w, r.WithContext(ctx))
		}
	}
}
