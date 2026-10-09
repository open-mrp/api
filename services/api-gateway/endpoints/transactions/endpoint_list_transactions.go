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

// Request to list transactions.
type ListTransactionsRequest struct {
	apiresource.PaginationRequest
	// Search matches transactions whose number contains every word of `q` as a prefix.
	//
	// Filter by allocation status: `allocated` (marked fully applied to invoices) or `unallocated` (not yet marked fully applied).
	Status *constants.TransactionAllocationStatus `query:"status"`
	// Filter by transaction type codes.
	TypeCodes []constants.TransactionType `query:"types"`
	// Filter by adjustment type codes (see List Adjustment Types for available values).
	AdjustmentTypeCodes []string `query:"adjustment_types"`
	// Filter by payment method codes.
	MethodCodes []constants.TransactionMethod `query:"methods"`
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by the account group each customer belongs to.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Only include transactions whose funds were received on or after this date (`YYYY-MM-DD`, UTC). A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	StartDate *string `query:"starts_at"`
	// Only include transactions whose funds were received on or before this date (`YYYY-MM-DD`, UTC), covering that whole day. A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	EndDate *string `query:"ends_at"`
}

// TODO: stop returning TransactionSummary; return the full Transaction apiresource and use proper includes values to control expansion.

// Returns a paginated list of transactions for the current account, newest first.
//
// Free-text search matches the transaction number and note.
type ListTransactionsEndpoint struct{}

func (e *ListTransactionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTransactionsRequest, *apiresource.List[apiresource.TransactionSummary]] {
	return (&apiendpoint.APIEndpoint[*ListTransactionsRequest, *apiresource.List[apiresource.TransactionSummary]]{
		Title:             "List Transactions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTransactions, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeTransactionSummary,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTransactionsRequest) (*apiresource.List[apiresource.TransactionSummary], *apierror.APIError) {
			return svc.(TransactionSvc).ListTransactions
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTransactionSummary,
			Fields:     []string{"customer", "customer.bill_to_address"},
		}),
	})
}
