package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list transactions for a customer account.
type ListAccountTransactionsRequest struct {
	apiresource.PaginationRequest
	// Customer account ID.
	CustomerAccountID string `path:"account_id" validate:"required"`
	// Filter by allocation status: `allocated` (marked fully applied to invoices) or `unallocated` (still counted as an open credit).
	Status *constants.TransactionAllocationStatus `query:"status"`
	// Filter by transaction type code.
	Type *constants.TransactionType `query:"type"`
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
