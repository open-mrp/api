package httptransport

import (
	"context"
	"net/http"
	"runtime"

	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/apikit/apierror"
)

func RespondWithAPIError(ctx context.Context, w http.ResponseWriter, apiErr *apierror.APIError, opts ...RespondOption) {
	if apiErr == nil {
		panic("RespondWithAPIError: apiErr received is nil.")
	}

	rl, hasRL := appctx.GetRequestLog(ctx)
	if hasRL && rl != nil {
		errorCode := string(apiErr.Code)
		rl.ErrorCode = &errorCode
		rl.ErrorMessage = &apiErr.PublicMessage
		// Record the internal chain and stack for every 5xx, not just internal_error: timeouts, service-unavailable, etc. are equally in need of diagnosis in the error alert, and NewAPIError already captures a stack at the origin for all of them.
		if apiErr.Code.Status() >= 500 {
			// Record the full internal chain (InternalMessage + wrapped Internal error), not just the top-level message — otherwise the underlying cause (e.g. the real driver error behind "Database request failed for unknown reason.") is lost.
			internalMessage := apiErr.Error()
			rl.InternalErrorMessage = &internalMessage
			// Prefer the stack captured where the error originated (NewInternalError). Fall back to capturing here only if the error carries no origin stack.
			if apiErr.Stack != "" {
				st := apiErr.Stack
				rl.StackTrace = &st
			} else {
				stackTrace := make([]byte, 32768) // 32KB
				length := runtime.Stack(stackTrace, false)
				st := string(stackTrace[:length])
				rl.StackTrace = &st
			}
		}
	}

	statusCode := apiErr.Code.Status()
	RespondWithJSON(ctx, w, statusCode, apiErr.Response(), opts...)
}
