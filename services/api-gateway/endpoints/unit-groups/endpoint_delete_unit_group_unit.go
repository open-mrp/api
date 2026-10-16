package unitgroupep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *object.Deleted]{
		Title:             "Delete Unit Group Associated Unit",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeUnitGroupUnit,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupUnitRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(UnitGroupSvc).DeleteUnitGroupUnit, constants.ObjectTypeUnitGroupUnit, "id")
		},
	})
}
