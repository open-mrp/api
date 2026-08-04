package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a transaction allocation.
type UpdateTransactionAllocationRequest struct {
	// Transaction allocation ID.
	AllocationID string `path:"id" validate:"required"`
	// New amount of the transaction to apply to this invoice, as a decimal string in US dollars.
	//
	// The new amount is not checked against the transaction's total or the invoice's balance.
	Amount field.Optional[string] `json:"amount,omitzero"`
}

var sampleUpdateTransactionAllocationAmount = "150.00"
var sampleUpdateTransactionAllocationRequest = &UpdateTransactionAllocationRequest{
	Amount: field.Some(sampleUpdateTransactionAllocationAmount),
}

func (*UpdateTransactionAllocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTransactionAllocationRequest)
}

// Changes how much of a transaction is applied to the invoice it was allocated to.
//
// Only the amount can be changed; the transaction and invoice the allocation links are fixed. Payment roll-ups are left alone, so the transaction's `is_fully_allocated` flag and the invoice's paid-in-full status keep their previous values.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSettlements, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeTransactionAllocation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTransactionAllocationRequest) (*apiresource.TransactionAllocation, *apierror.APIError) {
			return svc.(TransactionAllocationSvc).UpdateTransactionAllocation
		},
	})
}
