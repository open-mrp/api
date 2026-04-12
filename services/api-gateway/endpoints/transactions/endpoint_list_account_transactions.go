package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListAccountTransactionsRequest is the request to list transactions for a customer account.
type ListAccountTransactionsRequest struct {
	apiresource.PaginationRequest
	// The customer account ID.
	CustomerAccountID string `path:"account_id" validate:"required"`
	// Filter by allocation status.
	Status *string `query:"status"`
	// Filter by transaction type.
	Type *string `query:"type"`
	// Whether to include transactions from child accounts. Defaults to true.
	IncludeChildAccounts *bool `query:"include_child_accounts"`
}

type ListAccountTransactionsEndpoint struct{}

func (e *ListAccountTransactionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountTransactionsRequest, *apiresource.List[apiresource.TransactionDetail]] {
	return &apiendpoint.APIEndpoint[*ListAccountTransactionsRequest, *apiresource.List[apiresource.TransactionDetail]]{
		Title:             "List Account Transactions",
		Description:       "Returns a paginated list of transactions scoped to a specific customer account, optionally including transactions from child accounts.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/accounts/{account_id}/transactions",
		Request:           &ListAccountTransactionsRequest{},
		Response:          &apiresource.List[apiresource.TransactionDetail]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountTransactionsRequest) (*apiresource.List[apiresource.TransactionDetail], *apierror.APIError) {
			return svc.(TransactionSvc).ListAccountTransactions
		},
	}
}
