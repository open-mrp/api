package unitgroupep

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

// Request to list unit groups.
type ListUnitGroupsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filter by unit dimension.
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
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionRead}}},
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
