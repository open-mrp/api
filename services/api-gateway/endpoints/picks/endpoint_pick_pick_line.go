package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// PickPickLineRequest is the request to mark a pick line as picked.
type PickPickLineRequest struct {
	// Pick ID.
	PickID string `path:"pick_id" validate:"required"`
	// Pick line ID.
	PickLineID string `path:"id" validate:"required"`
}

// Marks a pick line as picked.
//
// Sets the line's picked quantity to the quantity still outstanding on its sales order line. Has no effect on a line that has already been packed.
type PickPickLineEndpoint struct{}

func (e *PickPickLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*PickPickLineRequest, *apiresource.PickLine] {
	return (&apiendpoint.APIEndpoint[*PickPickLineRequest, *apiresource.PickLine]{
		Title:             "Pick Pick Line",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{pick_id}/lines/{id}/actions/pick",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePickLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
			return svc.(PickSvc).PickPickLine
		},
	})
}
