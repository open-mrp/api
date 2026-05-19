package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a transaction allocation.
type UpdateTransactionAllocationRequest struct {
	// Transaction allocation ID.
	AllocationID string `path:"id" validate:"required"`
	// Allocation amount as a decimal string.
	Amount *string `json:"amount"`
}

var sampleUpdateTransactionAllocationAmount = "150.00"
var sampleUpdateTransactionAllocationRequest = &UpdateTransactionAllocationRequest{
	Amount: &sampleUpdateTransactionAllocationAmount,
}

func (*UpdateTransactionAllocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTransactionAllocationRequest)
}

// Partially updates a transaction allocation.
type UpdateTransactionAllocationEndpoint struct{}

func (e *UpdateTransactionAllocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateTransactionAllocationRequest, *apiresource.TransactionAllocation] {
	return (&apiendpoint.APIEndpoint[*UpdateTransactionAllocationRequest, *apiresource.TransactionAllocation]{
		Title:             "Update Transaction Allocation",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-allocations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTransactionAllocationRequest) (*apiresource.TransactionAllocation, *apierror.APIError) {
			return svc.(TransactionAllocationSvc).UpdateTransactionAllocation
		},
	})
}
