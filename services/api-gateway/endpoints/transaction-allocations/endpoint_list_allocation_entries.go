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
	// Filter by the underlying transaction's type code.
	TransactionType *constants.TransactionType `query:"transaction_type"`
	// Only include allocations created on or after this date (`YYYY-MM-DD`, UTC). A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	StartDate *string `query:"starts_at"`
	// Only include allocations created on or before this date (`YYYY-MM-DD`, UTC), covering that whole day. A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	EndDate *string `query:"ends_at"`
}

// Returns a paginated list of the individual applications of transaction money to invoices, newest first.
//
// Each entry pairs one transaction with one invoice and the amount applied. Entries are created by recording a settlement; there is no endpoint that creates one directly. Free-text search matches an exact invoice number, transaction number or customer number, or part of the customer name.
type ListAllocationEntriesEndpoint struct{}

func (e *ListAllocationEntriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAllocationEntriesRequest, *apiresource.List[apiresource.AllocationEntry]] {
	return (&apiendpoint.APIEndpoint[*ListAllocationEntriesRequest, *apiresource.List[apiresource.AllocationEntry]]{
		Title:             "List Allocation Entries",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-allocations",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
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
