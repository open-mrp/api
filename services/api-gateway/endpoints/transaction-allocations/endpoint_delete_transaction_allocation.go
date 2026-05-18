package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a transaction allocation.
type DeleteTransactionAllocationRequest struct {
	// Transaction allocation ID.
	AllocationID string `path:"id" validate:"required"`
}

// Deletes a transaction allocation.
type DeleteTransactionAllocationEndpoint struct{}

func (e *DeleteTransactionAllocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *apiresource.EmptyResource]{
		Title:             "Delete Transaction Allocation",
		Method:            http.MethodDelete,
		Route:             "/v1/finance/transaction-allocations/{id}",
		ContentType:       "application/json",
		Request:           &DeleteTransactionAllocationRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTransactionAllocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(TransactionAllocationSvc).DeleteTransactionAllocation
		},
	}).WithDocSource(e)
}
