package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// PickAllLinesRequest is the request to mark all lines on a pick as picked.
type PickAllLinesRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
}

// Marks all lines on a pick as picked.
type PickAllLinesEndpoint struct{}

func (e *PickAllLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*PickAllLinesRequest, *apiresource.PickDetail] {
	return (&apiendpoint.APIEndpoint[*PickAllLinesRequest, *apiresource.PickDetail]{
		Title:             "Pick All Lines",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}/actions/pick",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePick,
		ServiceHandler: func(svc any) func(ctx context.Context, req *PickAllLinesRequest) (*apiresource.PickDetail, *apierror.APIError) {
			return svc.(PickSvc).PickAllLines
		},
	})
}
