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
// Deletion fails with a validation error while the group is still in use: a `type_group` that is set as a customer's type cannot be deleted, and no group can be deleted while it grants product line access, backs a volume discount, or is attached to a customer registration flow.
//
// Deleting a `pricing_group` first unassigns it from every customer it was applied to, so those customers immediately stop receiving its pricing.
type DeleteAccountGroupEndpoint struct{}

func (e *DeleteAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *object.Deleted]{
		Title:             "Delete Account Group",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AccountGroupSvc).DeleteAccountGroup, constants.ObjectTypeAccountGroup, "id")
		},
	})
}
