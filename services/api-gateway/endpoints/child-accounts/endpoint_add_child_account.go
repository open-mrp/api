package childaccountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to add a child account.
type AddChildAccountRequest struct {
	// ID of the account to link as a child.
	ChildAccountID string `path:"child_account_id" validate:"required"`
}

// Links an existing account as a child of the target account.
//
// This call is idempotent: linking an account that is already a child of the target account succeeds without changes. Circular relationships (making an account a child of its own child) are rejected with a conflict error.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddChildAccountRequest) (*apiresource.ChildAccount, *apierror.APIError) {
			return svc.(ChildAccountSvc).AddChildAccount
		},
	})
}
