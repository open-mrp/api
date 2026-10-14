package transactionallocationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteTransactionAllocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteTransactionAllocationRequest, *object.Deleted]{
		Title:             "Delete Transaction Allocation",
		Method:            http.MethodDelete,
		Route:             "/v1/finance/transaction-allocations/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeTransactionAllocation,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSettlements, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTransactionAllocationRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(TransactionAllocationSvc).DeleteTransactionAllocation, constants.ObjectTypeTransactionAllocation, "id")
		},
	})
}
