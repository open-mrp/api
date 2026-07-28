package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
)

// RecoverMiddleware recovers from a handler panic, records the panic as the request log's error message (when one is not already set), and responds to the client with a generic internal server error instead of letting the connection drop.
func RecoverMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// The client only ever sees "Something went wrong", and the request log stores a generic message, so without this the panic and its stack are lost entirely.
					slog.ErrorContext(r.Context(), "panic recovered in HTTP handler",
						"panic", fmt.Sprintf("%v", rec),
						"method", r.Method,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)
					if rl, ok := appctx.GetRequestLog(r.Context()); ok && rl != nil && (rl.ErrorMessage == nil || *rl.ErrorMessage == "") {
						msg := "An unexpected error occurred during the request"
						rl.ErrorMessage = &msg
					}
					httptransport.RespondWithAPIError(r.Context(), w, apierror.NewInternalError(fmt.Errorf("%v", rec), fmt.Sprintf("A panic occurred during the request: %v", rec)))
				}
			}()
			next.ServeHTTP(w, r)
		}
	}
}
