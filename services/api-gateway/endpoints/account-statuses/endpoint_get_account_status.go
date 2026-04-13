package accountstatusep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetAccountStatusRequest is the request to retrieve a single account status.
type GetAccountStatusRequest struct {
	// The ID or code of the account status to retrieve.
	AccountStatusID string `path:"id" validate:"required"`
}

type GetAccountStatusEndpoint struct{}

func (e *GetAccountStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountStatusRequest, *apiresource.AccountStatus] {
	return &apiendpoint.APIEndpoint[*GetAccountStatusRequest, *apiresource.AccountStatus]{
		Title:             "Get Account Status",
		Description:       "Returns a single account status by its ID or code.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-statuses/{id}",
		Request:           &GetAccountStatusRequest{},
		Response:          &apiresource.AccountStatus{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountStatusRequest) (*apiresource.AccountStatus, *apierror.APIError) {
			return svc.(AccountStatusSvc).GetAccountStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountStatus,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
