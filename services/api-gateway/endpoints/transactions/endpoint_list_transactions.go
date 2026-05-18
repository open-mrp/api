package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list transactions.
type ListTransactionsRequest struct {
	apiresource.PaginationRequest
	// Filter by allocation status (allocated, unallocated, partially_allocated).
	Status *string `query:"status"`
	// Filter by transaction type codes.
	TypeCodes []string `query:"types"`
	// Filter by adjustment type codes.
	AdjustmentTypeCodes []string `query:"adjustment_types"`
	// Filter by transaction method codes.
	MethodCodes []string `query:"methods"`
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by customer group IDs.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
}

// TODO: stop returning TransactionSummary; return the full Transaction apiresource and use proper includes values to control expansion.

// Returns a paginated list of transactions for the current account.
type ListTransactionsEndpoint struct{}

func (e *ListTransactionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTransactionsRequest, *apiresource.List[apiresource.TransactionSummary]] {
	return (&apiendpoint.APIEndpoint[*ListTransactionsRequest, *apiresource.List[apiresource.TransactionSummary]]{
		Title:             "List Transactions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTransactionsRequest) (*apiresource.List[apiresource.TransactionSummary], *apierror.APIError) {
			return svc.(TransactionSvc).ListTransactions
		},
	})
}
