package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// VoidPickRequest is the request to void a pick.
type VoidPickRequest struct {
	// The ID of the pick to void.
	PickID string `path:"id" validate:"required"`
}

type VoidPickEndpoint struct{}

func (e *VoidPickEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidPickRequest, *apiresource.PickDetail] {
	return &apiendpoint.APIEndpoint[*VoidPickRequest, *apiresource.PickDetail]{
		Title:             "Void Pick",
		Description:       "Voids a pick, cancelling all of its lines.",
		Method:            http.MethodPut,
		Route:             "/v1/operations/picks/{id}/actions/void",
		Request:           &VoidPickRequest{},
		Response:          &apiresource.PickDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidPickRequest) (*apiresource.PickDetail, *apierror.APIError) {
			return svc.(PickSvc).VoidPick
		},
	}
}
