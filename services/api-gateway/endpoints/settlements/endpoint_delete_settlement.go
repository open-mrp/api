package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a settlement.
type DeleteSettlementRequest struct {
	// Settlement ID.
	SettlementID string `path:"id" validate:"required"`
}

// Deletes a settlement and all of its allocations.
//
// Affected invoices revert to an `unpaid` payment status, affected transactions are no longer marked fully allocated, and adjustment transactions referenced only by this settlement are removed.
type DeleteSettlementEndpoint struct{}

func (e *DeleteSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSettlementRequest, *apiresource.Settlement] {
	return (&apiendpoint.APIEndpoint[*DeleteSettlementRequest, *apiresource.Settlement]{
		Title:             "Delete Settlement",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/finance/settlements/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSettlement,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).DeleteSettlement
		},
	})
}
