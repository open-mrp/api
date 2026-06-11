package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a transaction.
type DeleteTransactionRequest struct {
	// Transaction ID.
	TransactionID string `path:"id" validate:"required"`
}

// Deletes a transaction along with all of its invoice allocations, and returns the deleted transaction.
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
