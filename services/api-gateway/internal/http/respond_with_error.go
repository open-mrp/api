package httptransport

import (
	"context"
	"net/http"
	"runtime"

	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
)

var frontendURL string

// SetFrontendURL configures the frontend base URL used to build request log links in error responses. When empty, the request_log_url field will be null.
func SetFrontendURL(url string) {
	frontendURL = url
}

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
		if apierror.Is5XXErrorCode(apiErr.Code) {
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

	statusCode := apierror.GetHTTPStatusCode(apiErr.Code)
	resp := apiErr.ToResponseMap()

	if frontendURL != "" && hasRL && rl != nil {
		if errResp, ok := resp.(apierror.APIErrorResponse); ok {
			url := frontendURL + "/dashboard/request-logs/" + rl.ID
			errResp.Error.RequestLogURL = &url
			resp = errResp
		}
	}

	RespondWithJSON(ctx, w, statusCode, resp, opts...)
}
