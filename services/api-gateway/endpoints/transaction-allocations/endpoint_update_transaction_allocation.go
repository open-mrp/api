package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateTransactionAllocationRequest is the request to update a transaction allocation.
type UpdateTransactionAllocationRequest struct {
	// The ID of the transaction allocation to update.
	AllocationID string `path:"id" validate:"required"`
	// The new allocation amount as a decimal string.
	Amount *string `json:"amount" nullable:"false"`
}

var sampleUpdateTransactionAllocationAmount = "150.00"
var sampleUpdateTransactionAllocationRequest = &UpdateTransactionAllocationRequest{
	Amount: &sampleUpdateTransactionAllocationAmount,
}

func (*UpdateTransactionAllocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTransactionAllocationRequest)
}

type UpdateTransactionAllocationEndpoint struct{}

func (e *UpdateTransactionAllocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateTransactionAllocationRequest, *apiresource.TransactionAllocation] {
	return &apiendpoint.APIEndpoint[*UpdateTransactionAllocationRequest, *apiresource.TransactionAllocation]{
		Title:             "Update Transaction Allocation",
		Description:       "Partially updates a transaction allocation.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/finance/transaction-allocations/{id}",
		Request:           &UpdateTransactionAllocationRequest{},
		Response:          &apiresource.TransactionAllocation{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTransactionAllocationRequest) (*apiresource.TransactionAllocation, *apierror.APIError) {
			return svc.(TransactionAllocationSvc).UpdateTransactionAllocation
		},
	}
}
