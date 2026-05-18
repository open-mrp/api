package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an account group.
type DeleteAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
}

// Deletes an account group. Fails if the account group is actively used in production.
type DeleteAccountGroupEndpoint struct{}

func (e *DeleteAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *apiresource.EmptyResource]{
		Title:             "Delete Account Group",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountGroupSvc).DeleteAccountGroup
		},
	})
}
