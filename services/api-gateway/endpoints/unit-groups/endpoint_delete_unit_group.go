package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a unit group.
type DeleteUnitGroupRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"id" validate:"required"`
}

// Deletes a unit group along with every unit association it contains.
//
// The units themselves are not deleted and remain available to other groups. System unit groups, which are shared across all accounts, cannot be deleted.
type DeleteUnitGroupEndpoint struct{}

func (e *DeleteUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitGroupRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitGroupRequest, *apiresource.EmptyResource]{
		Title:               "Delete Unit Group",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/unit-groups/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitGroupSvc).DeleteUnitGroup
		},
	})
}
