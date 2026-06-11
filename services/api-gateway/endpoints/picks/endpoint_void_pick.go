package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// VoidPickRequest is the request to void a pick.
type VoidPickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
}

// Voids a pick, cancelling all lines.
//
// Resets the picked quantity on every unpacked line to zero and clears the pick's `finished_at` timestamp. Fails if a shipment has already been created for the pick's sales order.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidPickRequest) (*apiresource.Pick, *apierror.APIError) {
			return svc.(PickSvc).VoidPick
		},
	})
}
