package httptransport

import (
	"context"
	"net/http"
	"runtime"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	"github.com/augno/api/shared/contracts"
)

func RespondWithAPIError(ctx context.Context, w http.ResponseWriter, apiErr *contracts.APIError) {
	if apiErr == nil {
		panic("RespondWithAPIError: apiErr received is nil.")
	}

	if rl, ok := apicontext.GetRequestLogFromContext(ctx); ok && rl != nil {
		rl.ErrorCode = string(apiErr.Code)
		rl.ErrorMessage = apiErr.PublicMessage
		if apiErr.Code == contracts.ErrorCodeInternalError {
			rl.InternalErrorMessage = apiErr.InternalMessage
			stackTrace := make([]byte, 32768)
			length := runtime.Stack(stackTrace, false)
			rl.StackTrace = string(stackTrace[:length])
		}
	}

	statusCode := contracts.GetHTTPStatusCode(apiErr.Code)
	RespondWithJSON(ctx, w, statusCode, apiErr.ToResponseMap())
}
