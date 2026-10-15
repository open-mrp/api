package accountgroupep

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

// Request to delete an account group.
type DeleteAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
}

// Deletes an account group.
//
// Only while no customer has it as their `group`, and it backs no product line access or registration flow. Its price list assignments are removed with it.
type DeleteAccountGroupEndpoint struct{}

func (e *DeleteAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *object.Deleted]{
		Title:             "Delete Account Group",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAccountGroup,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AccountGroupSvc).DeleteAccountGroup, constants.ObjectTypeAccountGroup, "id")
		},
	})
}
