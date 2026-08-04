package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove an associated unit from a unit group.
type DeleteUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// ID of the unit's association with the group, not the ID of the unit itself.
	AssociatedUnitID string `path:"id" validate:"required"`
}

// Removes a unit from a unit group so that products using the group can no longer be ordered in it.
//
// Only the association is deleted; the unit itself remains available. Associations cannot be removed from system unit groups.
type DeleteUnitGroupUnitEndpoint struct{}

func (e *DeleteUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *apiresource.EmptyResource]{
		Title:               "Delete Unit Group Associated Unit",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/unit-groups/{unit_group_id}/units/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitGroupSvc).DeleteUnitGroupUnit
		},
	})
}
