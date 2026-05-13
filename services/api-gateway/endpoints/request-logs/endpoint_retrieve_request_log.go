package requestlogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveRequestLogRequest is a request to get a request log.
type RetrieveRequestLogRequest struct {
	// Request log ID.
	ID string `path:"id" validate:"required"`
}

type RetrieveRequestLogEndpoint struct{}

func (e *RetrieveRequestLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRequestLogRequest, *apiresource.RequestLog] {
	return &apiendpoint.APIEndpoint[*RetrieveRequestLogRequest, *apiresource.RequestLog]{
		Title:             "Retrieve Request Log",
		Description:       "Returns a request log by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/core/request-logs/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveRequestLogRequest{},
		Response:          &apiresource.RequestLog{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError) {
			return svc.(RequestLogSvc).GetRequestLog
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRequestLog,
			Fields:     []string{"account", "actor", "actor.role", "actor.role.permissions", "query_params", "request_body", "response_body"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
	}
}
