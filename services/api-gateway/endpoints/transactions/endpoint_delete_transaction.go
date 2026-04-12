package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteTransactionRequest is the request to delete a transaction.
type DeleteTransactionRequest struct {
	// The ID of the transaction to delete.
	TransactionID string `path:"id" validate:"required"`
}

type DeleteTransactionEndpoint struct{}

func (e *DeleteTransactionEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTransactionRequest, *apiresource.TransactionDetail] {
	return &apiendpoint.APIEndpoint[*DeleteTransactionRequest, *apiresource.TransactionDetail]{
		Title:             "Delete Transaction",
		Description:       "Deletes a transaction and cascades deletion to its allocations.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions/{id}",
		Request:           &DeleteTransactionRequest{},
		Response:          &apiresource.TransactionDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).DeleteTransaction
		},
	}
}
