package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a transaction.
type GetTransactionRequest struct {
	// Transaction ID.
	TransactionID string `path:"id" validate:"required"`
	// Sub-resources to include in the response.
	Includes []string `include:"true"`
}

type GetTransactionEndpoint struct{}

func (e *GetTransactionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetTransactionRequest, *apiresource.TransactionDetail] {
	return &apiendpoint.APIEndpoint[*GetTransactionRequest, *apiresource.TransactionDetail]{
		Title:             "Get Transaction",
		Description:       "Returns a transaction by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions/{id}",
		Request:           &GetTransactionRequest{},
		Response:          &apiresource.TransactionDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).GetTransaction
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTransaction,
			Fields:     []string{"allocations", "customer", "responsible_user"},
		}),
	}
}
