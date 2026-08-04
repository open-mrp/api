package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
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
// Every transaction the settlement drew from is marked not fully allocated, so it reappears in List Open Credits even when allocations from other settlements already cover its full amount.
//
// Every invoice the settlement touched has its paid-in-full and overpaid flags cleared rather than recomputed, so its `payment_status` returns to `unpaid` even when other settlements still pay it off; those flags are only recomputed the next time a settlement allocates to that invoice.
//
// Adjustment transactions referenced only by this settlement are deleted along with it.
type DeleteSettlementEndpoint struct{}

func (e *DeleteSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSettlementRequest, *apiresource.Settlement] {
	return (&apiendpoint.APIEndpoint[*DeleteSettlementRequest, *apiresource.Settlement]{
		Title:               "Delete Settlement",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/finance/settlements/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSettlements, Action: types.ActionDelete}},
		ObjectType:          constants.ObjectTypeSettlement,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).DeleteSettlement
		},
	})
}
