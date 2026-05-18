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
	// Child account ID.
	ChildAccountID string `path:"child_account_id" validate:"required"`
}

// Removes a child account from the target account.
type RemoveChildAccountEndpoint struct{}

func (e *RemoveChildAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveChildAccountRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveChildAccountRequest, *apiresource.EmptyResource]{
		Title:             "Remove Child Account",
		Method:            http.MethodDelete,
		Route:             "/v1/identity/child-accounts/{child_account_id}",
		ContentType:       "application/json",
		Request:           &RemoveChildAccountRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveChildAccountRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ChildAccountSvc).RemoveChildAccount
		},
	}).WithDocSource(e)
}
