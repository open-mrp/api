package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list unit groups.
type ListUnitGroupsRequest struct {
	apiresource.PaginationRequest
	// Filter by unit dimension (e.g. `mass`).
	Type *constants.UnitType `query:"type"`
}

// Returns a paginated list of unit groups, including system unit groups.
type ListUnitGroupsEndpoint struct{}

func (e *ListUnitGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitGroupsRequest, *apiresource.List[apiresource.UnitGroup]] {
	return (&apiendpoint.APIEndpoint[*ListUnitGroupsRequest, *apiresource.List[apiresource.UnitGroup]]{
		Title:             "List Unit Groups",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/unit-groups",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUnitGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUnitGroupsRequest) (*apiresource.List[apiresource.UnitGroup], *apierror.APIError) {
			return svc.(UnitGroupSvc).ListUnitGroups
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroup,
			Fields:     []string{"owner", "owner.account", "base_unit", "associated_units"},
		}),
	})
}
