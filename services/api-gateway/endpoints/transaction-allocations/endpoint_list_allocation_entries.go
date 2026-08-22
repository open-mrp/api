package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list transaction allocation entries.
type ListAllocationEntriesRequest struct {
	apiresource.PaginationRequest
	// Filter by the underlying transaction's type code (`payment`, `credit_memo`, `adjustment`, or `rebate`).
	TransactionType *string `query:"transaction_type"`
	// Only include allocations created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"starts_at"`
	// Only include allocations created on or before this date (`YYYY-MM-DD`), covering that whole day.
	EndDate *string `query:"ends_at"`
}

// Returns a paginated list of the individual applications of transaction money to invoices, newest first.
//
// Each entry pairs one transaction with one invoice and the amount applied. Entries are created by recording a settlement; there is no endpoint that creates one directly. Free-text search matches the invoice number and the transaction number.
type ListAllocationEntriesEndpoint struct{}

func (e *ListAllocationEntriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAllocationEntriesRequest, *apiresource.List[apiresource.AllocationEntry]] {
	return (&apiendpoint.APIEndpoint[*ListAllocationEntriesRequest, *apiresource.List[apiresource.AllocationEntry]]{
		Title:             "List Allocation Entries",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-allocations",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSettlements, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeAllocationEntry,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAllocationEntriesRequest) (*apiresource.List[apiresource.AllocationEntry], *apierror.APIError) {
			return svc.(TransactionAllocationSvc).ListAllocationEntries
		},
	})
}
