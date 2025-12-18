package middleware

import (
	"net/http"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	"github.com/augno/api/shared/constants"
)

func PlatformMiddleware(platform constants.PlatformMode) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := apicontext.WithPlatform(r.Context(), platform)
			next(w, r.WithContext(ctx))
		}
	}
}
