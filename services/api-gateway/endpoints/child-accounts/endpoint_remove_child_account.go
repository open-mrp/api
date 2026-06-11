package childaccountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove a child account.
type RemoveChildAccountRequest struct {
	// ID of the child account to unlink.
	ChildAccountID string `path:"child_account_id" validate:"required"`
}

// Unlinks a child account from the target account.
//
// Only the parent-child relationship is removed; the child account itself is not deleted. This call is idempotent: removing an account that is not currently a child succeeds without changes.
type RemoveChildAccountEndpoint struct{}

func (e *RemoveChildAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveChildAccountRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveChildAccountRequest, *apiresource.EmptyResource]{
		Title:             "Remove Child Account",
		Method:            http.MethodDelete,
		Route:             "/v1/identity/child-accounts/{child_account_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveChildAccountRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ChildAccountSvc).RemoveChildAccount
		},
	})
}
