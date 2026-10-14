package childaccountep

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

// Request to remove a child account.
type RemoveChildAccountRequest struct {
	// ID of the child account to unlink.
	ChildAccountID string `path:"child_account_id" validate:"required"`
}

// Unlinks a child account from the target account.
//
// Only the parent-child relationship is removed; the child account itself, and your customer record for it, are left untouched. This call is idempotent: removing an account that is not currently a child of the target account succeeds without changes.
type RemoveChildAccountEndpoint struct{}

func (e *RemoveChildAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveChildAccountRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*RemoveChildAccountRequest, *object.Deleted]{
		Title:             "Remove Child Account",
		Method:            http.MethodDelete,
		Route:             "/v1/identity/child-accounts/{child_account_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeChildAccount,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveChildAccountRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ChildAccountSvc).RemoveChildAccount, constants.ObjectTypeChildAccount, "child_account_id")
		},
	})
}
