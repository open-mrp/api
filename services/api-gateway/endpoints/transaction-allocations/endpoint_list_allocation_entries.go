package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListAllocationEntriesRequest is the request to list transaction allocation entries.
type ListAllocationEntriesRequest struct {
	apiresource.PaginationRequest
	// Filter by transaction type code (e.g. "payment", "credit").
	TransactionType *string `query:"transaction_type"`
	// Filter by start date (inclusive, YYYY-MM-DD).
	StartDate *string `query:"start_date"`
	// Filter by end date (exclusive, YYYY-MM-DD).
	EndDate *string `query:"end_date"`
}

type ListAllocationEntriesEndpoint struct{}

func (e *ListAllocationEntriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAllocationEntriesRequest, *apiresource.List[apiresource.AllocationEntry]] {
	return &apiendpoint.APIEndpoint[*ListAllocationEntriesRequest, *apiresource.List[apiresource.AllocationEntry]]{
		Title:             "List Allocation Entries",
		Description:       "Returns a paginated list of transaction allocation entries for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-allocations",
		Request:           &ListAllocationEntriesRequest{},
		Response:          &apiresource.List[apiresource.AllocationEntry]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAllocationEntriesRequest) (*apiresource.List[apiresource.AllocationEntry], *apierror.APIError) {
			return svc.(TransactionAllocationSvc).ListAllocationEntries
		},
	}
}
