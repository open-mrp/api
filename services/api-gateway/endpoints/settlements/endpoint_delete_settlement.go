package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteSettlementRequest is the request to delete a settlement.
type DeleteSettlementRequest struct {
	// Settlement ID.
	SettlementID string `path:"id" validate:"required"`
}

// Deletes a settlement, removing allocations and reverting payment statuses on affected invoices and transactions.
type DeleteSettlementEndpoint struct{}

func (e *DeleteSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSettlementRequest, *apiresource.Settlement] {
	return (&apiendpoint.APIEndpoint[*DeleteSettlementRequest, *apiresource.Settlement]{
		Title:             "Delete Settlement",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/finance/settlements/{id}",
		Request:           &DeleteSettlementRequest{},
		Response:          &apiresource.Settlement{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).DeleteSettlement
		},
	}).WithDocSource(e)
}
