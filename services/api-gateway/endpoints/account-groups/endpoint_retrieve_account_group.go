package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an account group.
type RetrieveAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
}

type RetrieveAccountGroupEndpoint struct{}

func (e *RetrieveAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountGroupRequest, *apiresource.AccountGroup] {
	return &apiendpoint.APIEndpoint[*RetrieveAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Retrieve Account Group",
		Description:       "Returns an account group by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveAccountGroupRequest{},
		Response:          &apiresource.AccountGroup{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).GetAccountGroup
		},
	}
}
