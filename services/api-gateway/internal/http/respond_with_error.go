package httptransport

import (
	"context"
	"net/http"
	"runtime"

	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
)

var frontendURL string

// SetFrontendURL configures the frontend base URL used to build request log links
// in error responses. When empty, the request_log_url field will be null.
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
		if apiErr.Code == apierror.ErrorCodeInternalError {
			rl.InternalErrorMessage = &apiErr.InternalMessage
			stackTrace := make([]byte, 32768) // 32KB
			length := runtime.Stack(stackTrace, false)
			st := string(stackTrace[:length])
			rl.StackTrace = &st
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
