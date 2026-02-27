package requestlogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetRequestLogRequest is the request to retrieve a single request log.
type GetRequestLogRequest struct {
	// The ID of the request log to retrieve.
	ID string `path:"id"`
}

const getRequestLogEndpointDescription string = `This endpoint returns a single request log by its ID.`

type GetRequestLogEndpoint struct{}

func (e *GetRequestLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRequestLogRequest, *apiresource.RequestLog] {
	return &apiendpoint.APIEndpoint[*GetRequestLogRequest, *apiresource.RequestLog]{
		Title:             "Get Request Log",
		Description:       getRequestLogEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/core/request-logs/{id}",
		ContentType:       "application/json",
		Request:           &GetRequestLogRequest{},
		Response:          &apiresource.RequestLog{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError) {
			return svc.(RequestLogSvc).GetRequestLog
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
			SkipRequestLogging:     true,
		},
	}
}
