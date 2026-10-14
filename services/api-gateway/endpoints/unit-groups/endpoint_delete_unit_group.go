package unitgroupep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		Title:             "Delete Unit Group",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/unit-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitGroupSvc).DeleteUnitGroup
		},
	})
}
