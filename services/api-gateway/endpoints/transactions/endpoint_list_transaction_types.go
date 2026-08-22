package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list transaction types.
type ListTransactionTypesRequest struct {
	apiresource.PaginationRequest
}

// Returns the transaction types that can be recorded against a customer: payments, credit memos, adjustments, and rebates.
//
// The set is fixed by the platform and identical for every account, so the results come back in one page; supplying a pagination cursor returns a validation error. Free-text search matches the display name.
type ListTransactionTypesEndpoint struct{}

func (e *ListTransactionTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTransactionTypesRequest, *apiresource.List[apiresource.TransactionType]] {
	return (&apiendpoint.APIEndpoint[*ListTransactionTypesRequest, *apiresource.List[apiresource.TransactionType]]{
		Title:             "List Transaction Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-types",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTransactionTypesRequest) (*apiresource.List[apiresource.TransactionType], *apierror.APIError) {
			return svc.(TransactionSvc).ListTransactionTypes
		},
	})
}
