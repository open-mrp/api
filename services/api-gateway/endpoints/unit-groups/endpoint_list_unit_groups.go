package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListUnitGroupsRequest is the request to list unit groups with optional filters.
type ListUnitGroupsRequest struct {
	apiresource.PaginationRequest
	// Filter by unit type code (e.g. "mass", "quantity").
	Type *constants.UnitType `query:"type"`
}

type ListUnitGroupsEndpoint struct{}

func (e *ListUnitGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitGroupsRequest, *apiresource.List[apiresource.UnitGroup]] {
	return &apiendpoint.APIEndpoint[*ListUnitGroupsRequest, *apiresource.List[apiresource.UnitGroup]]{
		Title:             "List Unit Groups",
		Description:       "Returns a paginated list of unit groups for the current account, including system unit groups.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/unit-groups",
		Request:           &ListUnitGroupsRequest{},
		Response:          &apiresource.List[apiresource.UnitGroup]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUnitGroupsRequest) (*apiresource.List[apiresource.UnitGroup], *apierror.APIError) {
			return svc.(UnitGroupSvc).ListUnitGroups
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroup,
			Fields:     []string{"owner", "base_unit", "associated_units"},
		}),
	}
}
