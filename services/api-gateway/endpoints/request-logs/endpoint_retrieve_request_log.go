package requestlogep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a request log.
type RetrieveRequestLogRequest struct {
	// Request log ID.
	ID string `path:"id" validate:"required"`
}

// Returns a single API request log by ID.
//
// The log is readable when your account is either the acting account or the account that was acted upon. This is also the only endpoint that can return the captured query parameters and request and response bodies, and the only way to read the high-traffic-endpoint logs that are withheld from the list endpoint.
type RetrieveRequestLogEndpoint struct{}

func (e *RetrieveRequestLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRequestLogRequest, *apiresource.RequestLog] {
	return (&apiendpoint.APIEndpoint[*RetrieveRequestLogRequest, *apiresource.RequestLog]{
		Title:             "Retrieve Request Log",
		Method:            http.MethodGet,
		Route:             "/v1/core/request-logs/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainRequestLogs, Action: types.ActionRead}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError) {
			return svc.(RequestLogSvc).GetRequestLog
		},
		ObjectType: constants.ObjectTypeRequestLog,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRequestLog,
			Fields:     []string{"account", "actor", "actor.role", "actor.role.permissions", "query_params", "request_body", "response_body"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
	})
}
