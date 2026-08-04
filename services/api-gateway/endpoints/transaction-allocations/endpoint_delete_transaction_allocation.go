package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a transaction allocation.
type DeleteTransactionAllocationRequest struct {
	// Transaction allocation ID.
	AllocationID string `path:"id" validate:"required"`
}

// Removes the application of a transaction's money to one invoice, leaving both the transaction and the invoice in place.
//
// Payment roll-ups are left alone: the transaction's `is_fully_allocated` flag and the invoice's paid-in-full status keep their previous values, so set `is_fully_allocated` to `false` with Update Transaction to return the freed amount to the open credits list.
type DeleteTransactionAllocationEndpoint struct{}

func (e *DeleteTransactionAllocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *apiresource.EmptyResource]{
		Title:             "Delete Transaction Allocation",
		Method:            http.MethodDelete,
		Route:             "/v1/finance/transaction-allocations/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSettlements, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTransactionAllocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(TransactionAllocationSvc).DeleteTransactionAllocation
		},
	})
}
