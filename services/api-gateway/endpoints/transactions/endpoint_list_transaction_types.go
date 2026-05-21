package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list transaction types.
type ListTransactionTypesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of transaction types.
type ListTransactionTypesEndpoint struct{}

func (e *ListTransactionTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTransactionTypesRequest, *apiresource.List[apiresource.TransactionType]] {
	return (&apiendpoint.APIEndpoint[*ListTransactionTypesRequest, *apiresource.List[apiresource.TransactionType]]{
		Title:             "List Transaction Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-types",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTransactionTypesRequest) (*apiresource.List[apiresource.TransactionType], *apierror.APIError) {
			return svc.(TransactionSvc).ListTransactionTypes
		},
	})
}
