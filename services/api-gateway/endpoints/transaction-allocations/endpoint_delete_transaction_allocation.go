package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteTransactionAllocationRequest is the request to delete a transaction allocation.
type DeleteTransactionAllocationRequest struct {
	// The ID of the transaction allocation to delete.
	AllocationID string `path:"id" validate:"required"`
}

type DeleteTransactionAllocationEndpoint struct{}

func (e *DeleteTransactionAllocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *apiresource.EmptyResource]{
		Title:             "Delete Transaction Allocation",
		Description:       "Deletes a transaction allocation.",
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
	}
}
