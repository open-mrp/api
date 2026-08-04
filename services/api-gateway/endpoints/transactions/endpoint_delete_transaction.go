package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a transaction.
type DeleteTransactionRequest struct {
	// Transaction ID.
	TransactionID string `path:"id" validate:"required"`
}

// Deletes a transaction along with every allocation that applied it to an invoice, and returns the deleted transaction.
//
// Invoice payment status is not recomputed, so an invoice this transaction had paid off stays marked paid in full until the next settlement against it recalculates the flag. Deleting a transaction that was already deleted returns an already-deleted error rather than a not-found error.
type DeleteTransactionEndpoint struct{}

func (e *DeleteTransactionEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTransactionRequest, *apiresource.TransactionDetail] {
	return (&apiendpoint.APIEndpoint[*DeleteTransactionRequest, *apiresource.TransactionDetail]{
		Title:             "Delete Transaction",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTransactions, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).DeleteTransaction
		},
		ObjectType: constants.ObjectTypeTransaction,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTransaction,
			Fields:     []string{"allocations", "customer", "responsible_user", "responsible_user.user"},
		}),
	})
}
