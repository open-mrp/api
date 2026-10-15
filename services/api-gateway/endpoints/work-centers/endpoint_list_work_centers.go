package workcenterep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list work centers.
type ListWorkCentersRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a list of your work centers, newest first.
type ListWorkCentersEndpoint struct{}

// workCenterIncludes are the related objects a work center response can expand.
var workCenterIncludes = []string{"location", "scanning_stations", "machines"}

func (e *ListWorkCentersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListWorkCentersRequest, *apiresource.List[apiresource.WorkCenter]] {
	return (&apiendpoint.APIEndpoint[*ListWorkCentersRequest, *apiresource.List[apiresource.WorkCenter]]{
		Title:             "List Work Centers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/work-centers",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListWorkCentersRequest) (*apiresource.List[apiresource.WorkCenter], *apierror.APIError) {
			return svc.(WorkCenterSvc).ListWorkCenters
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
