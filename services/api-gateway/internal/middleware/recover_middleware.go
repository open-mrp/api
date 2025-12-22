package middleware

import (
	"fmt"
	"net/http"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/shared/contracts"
)

// Recover gracefully from a panic and send the client an internal server error
func RecoverMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if rl, ok := apicontext.GetRequestLogFromContext(r.Context()); ok && rl != nil && (rl.ErrorMessage == nil || *rl.ErrorMessage == "") {
						msg := "An unexpected error occurred during the request"
						rl.ErrorMessage = &msg
					}
					httptransport.RespondWithAPIError(r.Context(), w, contracts.NewInternalError(fmt.Errorf("%v", rec), fmt.Sprintf("A panic occurred during the request: %v", rec)))
				}
			}()
			next.ServeHTTP(w, r)
		}
	}
}
