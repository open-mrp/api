package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListTransactionTypesRequest is the request to list transaction types.
type ListTransactionTypesRequest struct {
	apiresource.PaginationRequest
}

type ListTransactionTypesEndpoint struct{}

func (e *ListTransactionTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTransactionTypesRequest, *apiresource.List[apiresource.TransactionType]] {
	return &apiendpoint.APIEndpoint[*ListTransactionTypesRequest, *apiresource.List[apiresource.TransactionType]]{
		Title:             "List Transaction Types",
		Description:       "Returns a paginated list of transaction types.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-types",
		Request:           &ListTransactionTypesRequest{},
		Response:          &apiresource.List[apiresource.TransactionType]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTransactionTypesRequest) (*apiresource.List[apiresource.TransactionType], *apierror.APIError) {
			return svc.(TransactionSvc).ListTransactionTypes
		},
	}
}
