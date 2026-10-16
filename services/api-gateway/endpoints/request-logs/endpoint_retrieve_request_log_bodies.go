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

// Request to retrieve a request log's bodies.
type RetrieveRequestLogBodiesRequest struct {
	// Request log ID.
	ID string `path:"id" validate:"required"`
}

// Returns the request and response bodies of a request log.
//
// Needs `request_logs:read`. Each call reads the stored bodies afresh, so fetch them only for the logs you need.
type RetrieveRequestLogBodiesEndpoint struct{}

func (e *RetrieveRequestLogBodiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRequestLogBodiesRequest, *apiresource.RequestLogBodies] {
	return (&apiendpoint.APIEndpoint[*RetrieveRequestLogBodiesRequest, *apiresource.RequestLogBodies]{
		Title:             "Retrieve Request Log Bodies",
		Method:            http.MethodGet,
		Route:             "/v1/core/request-logs/{id}/bodies",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainRequestLogs, Action: types.ActionRead},
		}},
		ObjectType: constants.ObjectTypeRequestLogBodies,
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRequestLogBodiesRequest) (*apiresource.RequestLogBodies, *apierror.APIError) {
			return svc.(RequestLogSvc).GetRequestLogBodies
		},
	})
}
