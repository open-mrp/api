package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// PickPickLineRequest is the request to mark a single pick line as picked.
type PickPickLineRequest struct {
	// The ID of the pick.
	PickID string `path:"pickId" validate:"required"`
	// The ID of the pick line to pick.
	PickLineID string `path:"id" validate:"required"`
}

type PickPickLineEndpoint struct{}

func (e *PickPickLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*PickPickLineRequest, *apiresource.PickLineDetail] {
	return &apiendpoint.APIEndpoint[*PickPickLineRequest, *apiresource.PickLineDetail]{
		Title:             "Pick Pick Line",
		Description:       "Marks a single pick line as picked.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{pickId}/lines/{id}/actions/pick",
		Request:           &PickPickLineRequest{},
		Response:          &apiresource.PickLineDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
			return svc.(PickSvc).PickPickLine
		},
	}
}
