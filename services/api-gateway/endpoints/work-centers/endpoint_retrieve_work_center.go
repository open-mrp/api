package workcenterep

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

// Request to retrieve a work center.
type RetrieveWorkCenterRequest struct {
	// ID of the work center to retrieve.
	WorkCenterID string `path:"id" validate:"required"`
}

// Retrieves a work center.
type RetrieveWorkCenterEndpoint struct{}

func (e *RetrieveWorkCenterEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveWorkCenterRequest, *apiresource.WorkCenter] {
	return (&apiendpoint.APIEndpoint[*RetrieveWorkCenterRequest, *apiresource.WorkCenter]{
		Title:             "Retrieve Work Center",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/work-centers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError) {
			return svc.(WorkCenterSvc).GetWorkCenter
		},
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionRead},
		}},
		ObjectType: constants.ObjectTypeWorkCenter,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeWorkCenter,
			Fields:     workCenterIncludes,
		}),
	})
}
