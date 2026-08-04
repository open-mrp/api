package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list transactions for a customer account.
type ListAccountTransactionsRequest struct {
	apiresource.PaginationRequest
	// Customer account ID.
	CustomerAccountID string `path:"account_id" validate:"required"`
	// Filter by allocation status: `allocated` (marked fully applied to invoices) or `unallocated` (still counted as an open credit).
	Status *string `query:"status"`
	// Filter by transaction type code (`payment`, `credit_memo`, `adjustment`, or `rebate`).
	Type *string `query:"type"`
	// Whether to also include transactions recorded against the customer's child accounts.
	//
	// Child account transactions are included unless this is set to `false`.
	IncludeChildAccounts *bool `query:"include_child_accounts"`
}

// Returns a paginated list of the transactions recorded against one customer account, newest first.
//
// Transactions recorded against that customer's child accounts are included by default. Free-text search matches the transaction number and note.
type ListAccountTransactionsEndpoint struct{}

func (e *ListAccountTransactionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountTransactionsRequest, *apiresource.List[apiresource.TransactionDetail]] {
	return (&apiendpoint.APIEndpoint[*ListAccountTransactionsRequest, *apiresource.List[apiresource.TransactionDetail]]{
		Title:             "List Account Transactions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/accounts/{account_id}/transactions",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTransactions, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountTransactionsRequest) (*apiresource.List[apiresource.TransactionDetail], *apierror.APIError) {
			return svc.(TransactionSvc).ListAccountTransactions
		},
		ObjectType: constants.ObjectTypeTransaction,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTransaction,
			Fields:     []string{"allocations", "customer", "responsible_user", "responsible_user.user"},
		}),
	})
}
