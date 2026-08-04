package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list transaction methods.
type ListTransactionMethodsRequest struct {
	apiresource.PaginationRequest
}

// Returns the payment methods that can be recorded on a transaction, such as cash, check, and ACH.
//
// The set is fixed by the platform and identical for every account, so the results come back in one page; supplying a pagination cursor returns a validation error. Free-text search matches the display name.
type ListTransactionMethodsEndpoint struct{}

func (e *ListTransactionMethodsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTransactionMethodsRequest, *apiresource.List[apiresource.TransactionMethod]] {
	return (&apiendpoint.APIEndpoint[*ListTransactionMethodsRequest, *apiresource.List[apiresource.TransactionMethod]]{
		Title:             "List Transaction Methods",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-methods",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTransactionMethodsRequest) (*apiresource.List[apiresource.TransactionMethod], *apierror.APIError) {
			return svc.(TransactionSvc).ListTransactionMethods
		},
	})
}
