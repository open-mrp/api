package priorityep

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

// Request to list priorities.
type ListPrioritiesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Lists the priority levels that can be set on a sales order or purchase order.
//
// The levels are platform-provided and the same for every account, so the result is small and stable enough to cache. Results are ordered newest first rather than by urgency.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge has no such route.
type ListPrioritiesEndpoint struct{}

func (e *ListPrioritiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPrioritiesRequest, *apiresource.List[apiresource.Priority]] {
	return (&apiendpoint.APIEndpoint[*ListPrioritiesRequest, *apiresource.List[apiresource.Priority]]{
		Title:             "List Priorities",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/priorities",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPriorities, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypePriority,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPrioritiesRequest) (*apiresource.List[apiresource.Priority], *apierror.APIError) {
			return svc.(PrioritySvc).ListPriorities
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePriority,
			Fields:     []string{"owner"},
		}),
	})
}
