package accountstatusep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an account status.
type RetrieveAccountStatusRequest struct {
	// Account status ID or code.
	//
	// A code such as `hold_shipment` resolves to the same status as that status's ID.
	AccountStatusID string `path:"id" validate:"required"`
}

// Returns a single account status, looked up by either its ID or its code.
type RetrieveAccountStatusEndpoint struct{}

func (e *RetrieveAccountStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountStatusRequest, *apiresource.AccountStatus] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountStatusRequest, *apiresource.AccountStatus]{
		Title:             "Retrieve Account Status",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-statuses/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountStatus,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountStatusRequest) (*apiresource.AccountStatus, *apierror.APIError) {
			return svc.(AccountStatusSvc).GetAccountStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountStatus,
			Fields:     []string{"owner"},
		}),
	})
}
