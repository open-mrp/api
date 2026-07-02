package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list transaction allocation entries.
type ListAllocationEntriesRequest struct {
	apiresource.PaginationRequest
	// Filter by the underlying transaction's type code (`payment`, `credit_memo`, `adjustment`, or `rebate`).
	TransactionType *string `query:"transaction_type"`
	// Only include allocations created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"start_date"`
	// Only include allocations created before this date (`YYYY-MM-DD`).
	EndDate *string `query:"end_date"`
}

// Returns a paginated list of transaction allocation entries for the current account.
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
