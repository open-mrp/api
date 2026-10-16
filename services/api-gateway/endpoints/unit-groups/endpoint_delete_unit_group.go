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

// Request to delete a unit group.
type DeleteUnitGroupRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"id" validate:"required"`
}

// Deletes a unit group along with every unit association it contains.
//
// The units themselves are not deleted and remain available to other groups. System unit groups, which are shared across all accounts, cannot be deleted.
type DeleteUnitGroupEndpoint struct{}

func (e *DeleteUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitGroupRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitGroupRequest, *object.Deleted]{
		Title:             "Delete Unit Group",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/unit-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeUnitGroup,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(UnitGroupSvc).DeleteUnitGroup, constants.ObjectTypeUnitGroup, "id")
		},
	})
}
