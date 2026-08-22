package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to void a pick.
type VoidPickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
}

// Voids a pick, undoing all picking work recorded on it.
//
// Resets the picked quantity on every unpacked line to zero and clears the pick's `finished_at` timestamp, so the pick starts over as open with nothing picked. The pick itself is not deleted, and the sales order is unaffected.
//
// Returns a validation error if any shipment exists for the pick's sales order. Voiding those shipments is not enough — they must be deleted, since a voided shipment still exists.
type VoidPickEndpoint struct{}

func (e *VoidPickEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidPickRequest, *apiresource.Pick] {
	return (&apiendpoint.APIEndpoint[*VoidPickRequest, *apiresource.Pick]{
		Title:             "Void Pick",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}/actions/void",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePick,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidPickRequest) (*apiresource.Pick, *apierror.APIError) {
			return svc.(PickSvc).VoidPick
		},
	})
}
