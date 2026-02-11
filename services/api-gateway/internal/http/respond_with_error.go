package httptransport

import (
	"context"
	"net/http"
	"runtime"

	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
)

func RespondWithAPIError(ctx context.Context, w http.ResponseWriter, apiErr *apierror.APIError, opts ...RespondOption) {
	if apiErr == nil {
		panic("RespondWithAPIError: apiErr received is nil.")
	}

	if rl, ok := appctx.GetRequestLog(ctx); ok && rl != nil {
		errorCode := string(apiErr.Code)
		rl.ErrorCode = &errorCode
		rl.ErrorMessage = &apiErr.PublicMessage
		if apiErr.Code == apierror.ErrorCodeInternalError {
			rl.InternalErrorMessage = &apiErr.InternalMessage
			stackTrace := make([]byte, 32768)
			length := runtime.Stack(stackTrace, false)
			st := string(stackTrace[:length])
			rl.StackTrace = &st
		}
	}

	statusCode := apierror.GetHTTPStatusCode(apiErr.Code)
	RespondWithJSON(ctx, w, statusCode, apiErr.ToResponseMap(), opts...)
}
