package accountstatusep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListAccountStatusesRequest is the request to list account statuses with optional filters.
type ListAccountStatusesRequest struct {
	apiresource.PaginationRequest
}

type ListAccountStatusesEndpoint struct{}

func (e *ListAccountStatusesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountStatusesRequest, *apiresource.List[apiresource.AccountStatus]] {
	return &apiendpoint.APIEndpoint[*ListAccountStatusesRequest, *apiresource.List[apiresource.AccountStatus]]{
		Title:             "List Account Statuses",
		Description:       "Returns a paginated list of account statuses, which are global lookup values used when setting account relationship statuses.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-statuses",
		Request:           &ListAccountStatusesRequest{},
		Response:          &apiresource.List[apiresource.AccountStatus]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountStatusesRequest) (*apiresource.List[apiresource.AccountStatus], *apierror.APIError) {
			return svc.(AccountStatusSvc).ListAccountStatuses
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountStatus,
			Fields:     []string{"owner"},
		}),
	}
}
