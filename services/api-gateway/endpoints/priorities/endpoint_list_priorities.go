package priorityep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list priorities.
type ListPrioritiesRequest struct {
	apiresource.PaginationRequest
}

// Lists the priority levels that can be set on a sales order or purchase order.
//
// The levels are platform-provided and the same for every account, so the result is small and stable enough to cache. Results are ordered newest first rather than by urgency.
type ListPrioritiesEndpoint struct{}

func (e *ListPrioritiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPrioritiesRequest, *apiresource.List[apiresource.Priority]] {
	return (&apiendpoint.APIEndpoint[*ListPrioritiesRequest, *apiresource.List[apiresource.Priority]]{
		Title:               "List Priorities",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/sales/priorities",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainPriorities, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypePriority,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPrioritiesRequest) (*apiresource.List[apiresource.Priority], *apierror.APIError) {
			return svc.(PrioritySvc).ListPriorities
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePriority,
			Fields:     []string{"owner"},
		}),
	})
}
