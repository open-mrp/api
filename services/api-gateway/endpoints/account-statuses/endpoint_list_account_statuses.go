package accountstatusep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list account statuses.
type ListAccountStatusesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of account statuses.
//
// Account statuses are system-provided lookup values shared across all accounts, used to set a customer's status (for example, placing a customer on a credit hold). The list is fixed — statuses cannot be created, edited, or deleted — so use it to populate a status picker or to resolve a code to its display name.
type ListAccountStatusesEndpoint struct{}

func (e *ListAccountStatusesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountStatusesRequest, *apiresource.List[apiresource.AccountStatus]] {
	return (&apiendpoint.APIEndpoint[*ListAccountStatusesRequest, *apiresource.List[apiresource.AccountStatus]]{
		Title:             "List Account Statuses",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-statuses",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountStatus,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountStatusesRequest) (*apiresource.List[apiresource.AccountStatus], *apierror.APIError) {
			return svc.(AccountStatusSvc).ListAccountStatuses
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountStatus,
			Fields:     []string{"owner"},
		}),
	})
}
