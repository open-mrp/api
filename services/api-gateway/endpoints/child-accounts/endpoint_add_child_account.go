package childaccountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to add a child account.
type AddChildAccountRequest struct {
	// Child account ID.
	ChildAccountID string `path:"child_account_id" validate:"required"`
}

// Adds a child account relationship to the target account.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddChildAccountRequest) (*apiresource.ChildAccount, *apierror.APIError) {
			return svc.(ChildAccountSvc).AddChildAccount
		},
	})
}
