package accountstatusep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveAccountStatusRequest is the request to get an account status.
type RetrieveAccountStatusRequest struct {
	// Account status ID or code.
	AccountStatusID string `path:"id" validate:"required"`
}

type RetrieveAccountStatusEndpoint struct{}

func (e *RetrieveAccountStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountStatusRequest, *apiresource.AccountStatus] {
	return &apiendpoint.APIEndpoint[*RetrieveAccountStatusRequest, *apiresource.AccountStatus]{
		Title:             "Retrieve Account Status",
		Description:       "Returns an account status by ID or code.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-statuses/{id}",
		Request:           &RetrieveAccountStatusRequest{},
		Response:          &apiresource.AccountStatus{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountStatusRequest) (*apiresource.AccountStatus, *apierror.APIError) {
			return svc.(AccountStatusSvc).GetAccountStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountStatus,
			Fields:     []string{"owner"},
		}),
	}
}
