package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list the associated units within a unit group.
type ListUnitGroupUnitsRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
}

// Returns the units associated with a unit group, along with the discount and customer portal visibility applied to each.
//
// Every association in the group is returned in a single response; this list is not paginated.
type ListUnitGroupUnitsEndpoint struct{}

func (e *ListUnitGroupUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitGroupUnitsRequest, *apiresource.List[apiresource.UnitGroupUnit]] {
	return (&apiendpoint.APIEndpoint[*ListUnitGroupUnitsRequest, *apiresource.List[apiresource.UnitGroupUnit]]{
		Title:               "List Unit Group Units",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/unit-groups/{unit_group_id}/units",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnitGroupUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUnitGroupUnitsRequest) (*apiresource.List[apiresource.UnitGroupUnit], *apierror.APIError) {
			return svc.(UnitGroupSvc).ListUnitGroupUnits
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	})
}
