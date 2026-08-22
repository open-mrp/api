package childaccountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to add a child account.
type AddChildAccountRequest struct {
	// ID of the account to link as a child.
	ChildAccountID string `path:"child_account_id" validate:"required"`
}

// Links an existing account as a child of the target account, so the two sit in a customer hierarchy such as a store location under its head office.
//
// Both the parent and the child must already be accounts you have a customer relationship with, and the child must be one you manage on its behalf — an account that runs its own OpenMRP subscription, or that also trades with other sellers, is rejected with an authorization error.
//
// This call is idempotent: linking an account that is already a child of the target account succeeds without changes. Circular relationships (making an account a child of its own child) are rejected with a conflict error. An account has at most one parent, so linking a child that already sits under a different parent moves it.
type AddChildAccountEndpoint struct{}

func (e *AddChildAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddChildAccountRequest, *apiresource.ChildAccount] {
	return (&apiendpoint.APIEndpoint[*AddChildAccountRequest, *apiresource.ChildAccount]{
		Title:             "Add Child Account",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/child-accounts/{child_account_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeChildAccount,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddChildAccountRequest) (*apiresource.ChildAccount, *apierror.APIError) {
			return svc.(ChildAccountSvc).AddChildAccount
		},
	})
}
