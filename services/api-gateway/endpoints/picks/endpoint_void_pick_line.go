package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// VoidPickLineRequest is the request to void a pick line.
type VoidPickLineRequest struct {
	// Pick ID.
	PickID string `path:"pick_id" validate:"required"`
	// Pick line ID.
	PickLineID string `path:"id" validate:"required"`
}

// Voids a pick line.
type VoidPickLineEndpoint struct{}

func (e *VoidPickLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidPickLineRequest, *apiresource.PickLineDetail] {
	return (&apiendpoint.APIEndpoint[*VoidPickLineRequest, *apiresource.PickLineDetail]{
		Title:             "Void Pick Line",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{pick_id}/lines/{id}/actions/void",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
			return svc.(PickSvc).VoidPickLine
		},
	})
}
